package graphqlschema

import "fmt"

func validateDefaultSource(registry *Registry) error {
	if registry == nil || registry.defaultSource == "" {
		return nil
	}
	if _, ok := registry.sourceIndex[registry.defaultSource]; ok {
		return nil
	}
	return fmt.Errorf("graphql_default_source %q was not found in graphql_sources", registry.defaultSource)
}

func validateDomainRootQueries(sourceName string, schema *Schema, cfg DomainConfig) error {
	for _, name := range cfg.RootQueries {
		if _, ok := schema.RootQueryByName(name); ok {
			continue
		}
		return fmt.Errorf("graphql source %q domain %q root query %q was not found in the schema snapshot", sourceName, cfg.Name, name)
	}
	return nil
}

func validateDomainTypes(sourceName string, schema *Schema, cfg DomainConfig) error {
	for _, name := range cfg.Types {
		if _, ok := schema.TypeByName(name); ok {
			continue
		}
		return fmt.Errorf("graphql source %q domain %q type %q was not found in the schema snapshot", sourceName, cfg.Name, name)
	}
	return nil
}

func cloneSource(source Source) Source {
	out := source
	out.Headers = cloneHeaders(source.Headers)
	out.Budget = source.Budget
	out.Domains = cloneDomains(source.Domains)
	out.domainIndex = nil
	return out
}

func cloneDomains(raw []Domain) []Domain {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Domain, 0, len(raw))
	for _, domain := range raw {
		out = append(out, Domain{
			Name:        domain.Name,
			Description: domain.Description,
			RootQueries: append([]string(nil), domain.RootQueries...),
			Types:       append([]string(nil), domain.Types...),
			Budget:      domain.Budget,
		})
	}
	return out
}

func cloneHeaders(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}
