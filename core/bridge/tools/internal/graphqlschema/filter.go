package graphqlschema

import (
	"fmt"
	"strings"
)

func (s *Source) ResolveDomain(name string) (*Domain, error) {
	trimmed := strings.TrimSpace(name)
	if s == nil || trimmed == "" {
		return nil, nil
	}
	domain, ok := s.domainIndex[trimmed]
	if ok {
		return domain, nil
	}
	return nil, fmt.Errorf("graphql source %q domain %q was not found", s.Name, trimmed)
}

func (s *Source) DomainBudget(name string) (Budget, error) {
	if s == nil {
		return Budget{}, fmt.Errorf("graphql source is not configured")
	}
	budget := s.Budget
	domain, err := s.ResolveDomain(name)
	if err != nil || domain == nil {
		return budget, err
	}
	if domain.Budget.MaxDepth > 0 {
		budget.MaxDepth = domain.Budget.MaxDepth
	}
	if domain.Budget.MaxFields > 0 {
		budget.MaxFields = domain.Budget.MaxFields
	}
	if domain.Budget.MaxRootFields > 0 {
		budget.MaxRootFields = domain.Budget.MaxRootFields
	}
	return budget, nil
}

func (s *Source) DomainList() []Domain {
	if s == nil || len(s.Domains) == 0 {
		return nil
	}
	return cloneDomains(s.Domains)
}

func (s *Source) RootQueriesForDomain(name string) ([]Field, error) {
	if s == nil || s.Schema == nil {
		return nil, fmt.Errorf("graphql source schema is not configured")
	}
	domain, err := s.ResolveDomain(name)
	if err != nil || domain == nil {
		return s.Schema.RootQueryList(), err
	}
	return filterRootQueriesByDomain(s.Schema.RootQueryList(), domain), nil
}

func (s *Source) TypeForDomain(domainName string, typeName string) (Type, bool, error) {
	if s == nil || s.Schema == nil {
		return Type{}, false, fmt.Errorf("graphql source schema is not configured")
	}
	domain, err := s.ResolveDomain(domainName)
	if err != nil {
		return Type{}, false, err
	}
	if domain != nil && !domain.typeSet[typeName] {
		return Type{}, false, nil
	}
	item, ok := s.Schema.TypeByName(typeName)
	return item, ok, nil
}

func (s *Source) FieldMatchesForDomain(domainName string, fieldName string) ([]FieldLocation, error) {
	if s == nil || s.Schema == nil {
		return nil, fmt.Errorf("graphql source schema is not configured")
	}
	domain, err := s.ResolveDomain(domainName)
	if err != nil {
		return nil, err
	}
	matches := s.Schema.FieldMatches(fieldName)
	if domain == nil {
		return matches, nil
	}
	return filterFieldMatchesByDomain(matches, domain), nil
}

func (s *Source) DomainAllowsRootQuery(domainName string, rootQuery string) (bool, error) {
	domain, err := s.ResolveDomain(domainName)
	if err != nil || domain == nil {
		return true, err
	}
	return domain.rootQuerySet[rootQuery], nil
}

func filterRootQueriesByDomain(fields []Field, domain *Domain) []Field {
	if domain == nil || len(fields) == 0 {
		return fields
	}
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		if domain.rootQuerySet[field.Name] {
			out = append(out, field)
		}
	}
	return out
}

func filterFieldMatchesByDomain(matches []FieldLocation, domain *Domain) []FieldLocation {
	if domain == nil || len(matches) == 0 {
		return matches
	}
	out := make([]FieldLocation, 0, len(matches))
	for _, match := range matches {
		if domain.typeSet[match.TypeName] {
			out = append(out, match)
		}
	}
	return out
}
