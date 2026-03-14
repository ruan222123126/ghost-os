package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools"
)

func registerOptionalGraphQLTools(registry *tools.Registry, cfg Config) error {
	if registry == nil {
		return fmt.Errorf("tool registry is not configured")
	}

	graphql := cfg.GraphQL
	if !graphql.Enabled || strings.TrimSpace(graphql.Endpoint) == "" {
		return nil
	}

	registry.Register(tools.NewGraphQLQueryTool(tools.GraphQLQueryConfig{
		Endpoint:         graphql.Endpoint,
		APIKey:           graphql.APIKey,
		TimeoutMS:        graphql.TimeoutMS,
		MaxResponseBytes: graphql.MaxResponseBytes,
		Headers:          graphql.Headers,
	}))

	schemaPath := strings.TrimSpace(graphql.SchemaPath)
	if schemaPath == "" {
		return nil
	}
	tool, err := tools.NewGraphQLSchemaLookupToolFromPath(schemaPath)
	if err != nil {
		return fmt.Errorf("load graphql schema snapshot: %w", err)
	}
	registry.Register(tool)
	return nil
}
