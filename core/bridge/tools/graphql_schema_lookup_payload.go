package tools

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/graphqlschema"
)

const (
	graphqlSchemaLookupActionListSources            = "list_sources"
	graphqlSchemaLookupActionListDomains            = "list_domains"
	graphqlSchemaLookupActionListRootQueries        = "list_root_queries"
	graphqlSchemaLookupActionListRootMutations      = "list_root_mutations"
	graphqlSchemaLookupActionDescribeMutationPolicy = "describe_mutation_policy"
	graphqlSchemaLookupActionDescribeType           = "describe_type"
	graphqlSchemaLookupActionFindField              = "find_field"
)

type graphqlSchemaLookupArgs struct {
	Action string `json:"action"`
	Source string `json:"source,omitempty"`
	Domain string `json:"domain,omitempty"`
	Name   string `json:"name,omitempty"`
}

type graphqlSchemaSourcePayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

type graphqlSchemaDomainPayload struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	RootQueries   []string `json:"root_queries,omitempty"`
	Types         []string `json:"types,omitempty"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	MaxFields     int      `json:"max_fields,omitempty"`
	MaxRootFields int      `json:"max_root_fields,omitempty"`
}

type graphqlSchemaFieldPayload struct {
	Name        string                         `json:"name"`
	Signature   string                         `json:"signature"`
	Description string                         `json:"description,omitempty"`
	ReturnType  string                         `json:"return_type"`
	Args        []graphqlSchemaArgumentPayload `json:"args,omitempty"`
	Deprecated  bool                           `json:"deprecated,omitempty"`
}

type graphqlSchemaArgumentPayload struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type graphqlSchemaTypePayload struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Fields      []graphqlSchemaFieldPayload `json:"fields"`
}

type graphqlSchemaFieldMatchPayload struct {
	TypeName string                    `json:"type_name"`
	Field    graphqlSchemaFieldPayload `json:"field"`
}

type graphqlSchemaMutationPolicyPayload struct {
	Name                    string                    `json:"name"`
	Description             string                    `json:"description,omitempty"`
	Domain                  string                    `json:"domain"`
	RootMutation            graphqlSchemaFieldPayload `json:"root_mutation"`
	ApprovalRequired        bool                      `json:"approval_required,omitempty"`
	IdempotencyMode         string                    `json:"idempotency_mode"`
	IdempotencyHeader       string                    `json:"idempotency_header,omitempty"`
	IdempotencyVariablePath string                    `json:"idempotency_variable_path,omitempty"`
	MaxDepth                int                       `json:"max_depth"`
	MaxFields               int                       `json:"max_fields"`
	MaxRootFields           int                       `json:"max_root_fields"`
	MaxFragments            int                       `json:"max_fragments"`
	Summary                 string                    `json:"summary"`
}

func newGraphQLSchemaFieldPayload(field graphqlschema.Field) graphqlSchemaFieldPayload {
	args := make([]graphqlSchemaArgumentPayload, 0, len(field.Args))
	for _, arg := range field.Args {
		args = append(args, graphqlSchemaArgumentPayload{
			Name:        arg.Name,
			Type:        arg.Type,
			Description: arg.Description,
		})
	}
	return graphqlSchemaFieldPayload{
		Name:        field.Name,
		Signature:   graphQLFieldSignature(field),
		Description: field.Description,
		ReturnType:  field.ReturnType,
		Args:        args,
		Deprecated:  field.Deprecated,
	}
}

func newGraphQLSchemaTypePayload(item graphqlschema.Type) graphqlSchemaTypePayload {
	fields := make([]graphqlSchemaFieldPayload, 0, len(item.Fields))
	for _, field := range item.Fields {
		fields = append(fields, newGraphQLSchemaFieldPayload(field))
	}
	return graphqlSchemaTypePayload{
		Name:        item.Name,
		Description: item.Description,
		Fields:      fields,
	}
}

func graphQLFieldSignature(field graphqlschema.Field) string {
	args := make([]string, 0, len(field.Args))
	for _, arg := range field.Args {
		args = append(args, arg.Name+": "+arg.Type)
	}
	if len(args) == 0 {
		return field.Name + ": " + field.ReturnType
	}
	return fmt.Sprintf("%s(%s): %s", field.Name, strings.Join(args, ", "), field.ReturnType)
}
