package config

import "fmt"

func validateGraphQLConfig(cfg GraphQLConfig) error {
	if len(cfg.Sources) == 0 {
		if cfg.DefaultSource != "" {
			return fmt.Errorf("graphql_default_source %q requires at least one graphql source", cfg.DefaultSource)
		}
		return nil
	}

	seen := make(map[string]bool, len(cfg.Sources))
	for _, source := range cfg.Sources {
		if err := validateGraphQLSourceConfig(source, seen); err != nil {
			return err
		}
	}
	if cfg.DefaultSource == "" || seen[cfg.DefaultSource] {
		return nil
	}
	return fmt.Errorf("graphql_default_source %q was not found in graphql_sources", cfg.DefaultSource)
}

func validateGraphQLSourceConfig(source GraphQLSourceConfig, seen map[string]bool) error {
	if source.Name == "" {
		return fmt.Errorf("graphql source name is required")
	}
	if seen[source.Name] {
		return fmt.Errorf("graphql source %q is duplicated", source.Name)
	}
	seen[source.Name] = true
	if source.Endpoint == "" {
		return fmt.Errorf("graphql source %q endpoint is required", source.Name)
	}
	if source.SchemaPath == "" {
		return fmt.Errorf("graphql source %q schema_path is required", source.Name)
	}
	if err := validateGraphQLBudgetValue(source.TimeoutMS, source.Name, "timeout_ms"); err != nil {
		return err
	}
	if err := validateGraphQLBudgetValue(source.MaxResponseBytes, source.Name, "max_response_bytes"); err != nil {
		return err
	}
	if err := validateGraphQLBudgetValue(source.MaxDepth, source.Name, "max_depth"); err != nil {
		return err
	}
	if err := validateGraphQLBudgetValue(source.MaxFields, source.Name, "max_fields"); err != nil {
		return err
	}
	if err := validateGraphQLBudgetValue(source.MaxRootFields, source.Name, "max_root_fields"); err != nil {
		return err
	}
	if err := validateGraphQLBudgetValue(source.MaxFragments, source.Name, "max_fragments"); err != nil {
		return err
	}
	return validateGraphQLDomains(source)
}

func validateGraphQLDomains(source GraphQLSourceConfig) error {
	seen := make(map[string]bool, len(source.Domains))
	for _, domain := range source.Domains {
		if domain.Name == "" {
			return fmt.Errorf("graphql source %q domain name is required", source.Name)
		}
		if seen[domain.Name] {
			return fmt.Errorf("graphql source %q domain %q is duplicated", source.Name, domain.Name)
		}
		seen[domain.Name] = true
		if len(domain.RootQueries) == 0 {
			return fmt.Errorf("graphql source %q domain %q must define at least one root query", source.Name, domain.Name)
		}
		if err := validateGraphQLOverrideValue(domain.MaxDepth, source.Name, domain.Name, "max_depth"); err != nil {
			return err
		}
		if err := validateGraphQLOverrideValue(domain.MaxFields, source.Name, domain.Name, "max_fields"); err != nil {
			return err
		}
		if err := validateGraphQLOverrideValue(domain.MaxRootFields, source.Name, domain.Name, "max_root_fields"); err != nil {
			return err
		}
	}
	return nil
}
