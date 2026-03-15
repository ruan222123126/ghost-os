package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
	"ghost-os/bridge/tools/internal/tooljson"
)

type GraphQLSchemaLookupTool struct {
	registry *GraphQLSourceRegistry
}

func NewGraphQLSchemaLookupTool(registry *GraphQLSourceRegistry) Tool {
	return &GraphQLSchemaLookupTool{registry: registry}
}

func (GraphQLSchemaLookupTool) Name() string {
	return "graphql_schema_lookup"
}

func (GraphQLSchemaLookupTool) Description() string {
	return "Inspect configured GraphQL sources, domains, local schema snapshots, and allowed mutation policies. This tool never accesses the network."
}

func (GraphQLSchemaLookupTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["list_sources","list_domains","list_root_queries","list_root_mutations","describe_mutation_policy","describe_type","find_field"]},
			"source":{"type":"string","description":"Optional GraphQL source name. Omit only when a single source exists or graphql_default_source is configured."},
			"domain":{"type":"string","description":"Optional domain name used to clip visible root queries and types. Required for mutation policy actions."},
			"name":{"type":"string","description":"Type or field name for the selected action."}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t *GraphQLSchemaLookupTool) Execute(
	_ context.Context,
	argsJSON json.RawMessage,
	traceID string,
) (string, error) {
	args, err := decodeGraphQLSchemaLookupArgs(argsJSON)
	if err != nil {
		logGraphQLSchemaLookup(traceID, args.Source, args.Domain, args.Action, args.Name, err)
		return "", err
	}
	output, sourceName, err := t.execute(args)
	if sourceName == "" {
		sourceName = args.Source
	}
	logGraphQLSchemaLookup(traceID, sourceName, args.Domain, args.Action, args.Name, err)
	return output, err
}

func (t *GraphQLSchemaLookupTool) execute(args graphqlSchemaLookupArgs) (string, string, error) {
	if t == nil || t.registry == nil {
		return "", "", fmt.Errorf("graphql source registry is not configured")
	}
	if args.Action == graphqlSchemaLookupActionListSources {
		output, err := t.listSources()
		return output, "", err
	}

	source, err := t.registry.resolveSource(args.Source)
	if err != nil {
		return "", "", err
	}
	switch args.Action {
	case graphqlSchemaLookupActionListDomains:
		output, actionErr := listGraphQLDomains(source)
		return output, source.Name, actionErr
	case graphqlSchemaLookupActionListRootQueries:
		output, actionErr := listGraphQLRootQueries(source, args.Domain)
		return output, source.Name, actionErr
	case graphqlSchemaLookupActionListRootMutations:
		output, actionErr := listGraphQLRootMutations(source, args.Domain, t.registry)
		return output, source.Name, actionErr
	case graphqlSchemaLookupActionDescribeMutationPolicy:
		output, actionErr := describeGraphQLMutationPolicy(source, args.Domain, args.Name, t.registry)
		return output, source.Name, actionErr
	case graphqlSchemaLookupActionDescribeType:
		output, actionErr := describeGraphQLType(source, args.Domain, args.Name)
		return output, source.Name, actionErr
	case graphqlSchemaLookupActionFindField:
		output, actionErr := findGraphQLField(source, args.Domain, args.Name)
		return output, source.Name, actionErr
	default:
		return "", source.Name, fmt.Errorf("action must be one of: list_sources, list_domains, list_root_queries, list_root_mutations, describe_mutation_policy, describe_type, find_field")
	}
}

func decodeGraphQLSchemaLookupArgs(argsJSON json.RawMessage) (graphqlSchemaLookupArgs, error) {
	var args graphqlSchemaLookupArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return graphqlSchemaLookupArgs{}, fmt.Errorf("decode args: %w", err)
	}
	args.Action = strings.ToLower(strings.TrimSpace(args.Action))
	args.Source = strings.TrimSpace(args.Source)
	args.Domain = strings.TrimSpace(args.Domain)
	args.Name = strings.TrimSpace(args.Name)
	return args, nil
}

func (t *GraphQLSchemaLookupTool) listSources() (string, error) {
	sources := t.registry.listSources()
	defaultSource := ""
	if t.registry != nil && t.registry.inner != nil {
		defaultSource = t.registry.inner.DefaultSourceName()
	}
	items := make([]graphqlSchemaSourcePayload, 0, len(sources))
	for _, source := range sources {
		items = append(items, graphqlSchemaSourcePayload{
			Name:        source.Name,
			Description: source.Description,
			Default:     source.Name == defaultSource,
		})
	}
	return tooljson.Encode(struct {
		Action        string                       `json:"action"`
		DefaultSource string                       `json:"default_source,omitempty"`
		Sources       []graphqlSchemaSourcePayload `json:"sources"`
	}{
		Action:        graphqlSchemaLookupActionListSources,
		DefaultSource: defaultSource,
		Sources:       items,
	})
}

func listGraphQLDomains(source *graphqlschema.Source) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	domains := source.DomainList()
	items := make([]graphqlSchemaDomainPayload, 0, len(domains))
	for _, domain := range domains {
		items = append(items, graphqlSchemaDomainPayload{
			Name:          domain.Name,
			Description:   domain.Description,
			RootQueries:   append([]string(nil), domain.RootQueries...),
			Types:         append([]string(nil), domain.Types...),
			MaxDepth:      domain.Budget.MaxDepth,
			MaxFields:     domain.Budget.MaxFields,
			MaxRootFields: domain.Budget.MaxRootFields,
		})
	}
	return tooljson.Encode(struct {
		Action  string                       `json:"action"`
		Source  string                       `json:"source"`
		Domains []graphqlSchemaDomainPayload `json:"domains"`
	}{
		Action:  graphqlSchemaLookupActionListDomains,
		Source:  source.Name,
		Domains: items,
	})
}

func listGraphQLRootQueries(source *graphqlschema.Source, domain string) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	rootQueries, err := source.RootQueriesForDomain(strings.TrimSpace(domain))
	if err != nil {
		return "", err
	}
	items := make([]graphqlSchemaFieldPayload, 0, len(rootQueries))
	for _, field := range rootQueries {
		items = append(items, newGraphQLSchemaFieldPayload(field))
	}
	return tooljson.Encode(struct {
		Action      string                      `json:"action"`
		Source      string                      `json:"source"`
		Domain      string                      `json:"domain,omitempty"`
		RootQueries []graphqlSchemaFieldPayload `json:"root_queries"`
	}{
		Action:      graphqlSchemaLookupActionListRootQueries,
		Source:      source.Name,
		Domain:      strings.TrimSpace(domain),
		RootQueries: items,
	})
}

func describeGraphQLType(source *graphqlschema.Source, domain string, name string) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	typeName := strings.TrimSpace(name)
	if typeName == "" {
		return "", fmt.Errorf("name is required for describe_type")
	}
	item, ok, err := source.TypeForDomain(strings.TrimSpace(domain), typeName)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", graphQLTypeMissError(source.Name, strings.TrimSpace(domain), typeName)
	}
	return tooljson.Encode(struct {
		Action string                   `json:"action"`
		Source string                   `json:"source"`
		Domain string                   `json:"domain,omitempty"`
		Type   graphqlSchemaTypePayload `json:"type"`
	}{
		Action: graphqlSchemaLookupActionDescribeType,
		Source: source.Name,
		Domain: strings.TrimSpace(domain),
		Type:   newGraphQLSchemaTypePayload(item),
	})
}

func findGraphQLField(source *graphqlschema.Source, domain string, name string) (string, error) {
	if source == nil {
		return "", fmt.Errorf("graphql source is not configured")
	}
	fieldName := strings.TrimSpace(name)
	if fieldName == "" {
		return "", fmt.Errorf("name is required for find_field")
	}
	matches, err := source.FieldMatchesForDomain(strings.TrimSpace(domain), fieldName)
	if err != nil {
		return "", err
	}
	items := make([]graphqlSchemaFieldMatchPayload, 0, len(matches))
	for _, match := range matches {
		items = append(items, graphqlSchemaFieldMatchPayload{
			TypeName: match.TypeName,
			Field:    newGraphQLSchemaFieldPayload(match.Field),
		})
	}
	return tooljson.Encode(struct {
		Action  string                           `json:"action"`
		Source  string                           `json:"source"`
		Domain  string                           `json:"domain,omitempty"`
		Name    string                           `json:"name"`
		Matches []graphqlSchemaFieldMatchPayload `json:"matches"`
	}{
		Action:  graphqlSchemaLookupActionFindField,
		Source:  source.Name,
		Domain:  strings.TrimSpace(domain),
		Name:    fieldName,
		Matches: items,
	})
}

func graphQLTypeMissError(sourceName string, domain string, typeName string) error {
	if domain == "" {
		return fmt.Errorf("type %q was not found in graphql source %q", typeName, sourceName)
	}
	return fmt.Errorf("type %q was not found in graphql source %q domain %q", typeName, sourceName, domain)
}
