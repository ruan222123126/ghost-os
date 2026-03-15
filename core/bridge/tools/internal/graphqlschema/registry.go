package graphqlschema

import (
	"fmt"
	"sort"
	"strings"
)

type Budget struct {
	MaxDepth      int
	MaxFields     int
	MaxRootFields int
	MaxFragments  int
}

type DomainConfig struct {
	Name          string
	Description   string
	RootQueries   []string
	Types         []string
	MaxDepth      int
	MaxFields     int
	MaxRootFields int
}

type SourceConfig struct {
	Name             string
	Description      string
	Endpoint         string
	APIKey           string
	SchemaPath       string
	TimeoutMS        int
	MaxResponseBytes int
	Headers          map[string]string
	MaxDepth         int
	MaxFields        int
	MaxRootFields    int
	MaxFragments     int
	Domains          []DomainConfig
}

type RegistryConfig struct {
	DefaultSource string
	Sources       []SourceConfig
}

type Domain struct {
	Name        string
	Description string
	RootQueries []string
	Types       []string
	Budget      Budget

	rootQuerySet map[string]bool
	typeSet      map[string]bool
}

type Source struct {
	Name             string
	Description      string
	Endpoint         string
	APIKey           string
	SchemaPath       string
	TimeoutMS        int
	MaxResponseBytes int
	Headers          map[string]string
	Budget           Budget
	Domains          []Domain
	Schema           *Schema

	domainIndex map[string]*Domain
}

type Registry struct {
	defaultSource string
	sources       []Source
	sourceIndex   map[string]*Source
}

func NewRegistry(cfg RegistryConfig) (*Registry, error) {
	sources, index, err := buildRegistrySources(cfg.Sources)
	if err != nil {
		return nil, err
	}
	registry := &Registry{
		defaultSource: strings.TrimSpace(cfg.DefaultSource),
		sources:       sources,
		sourceIndex:   index,
	}
	if err := validateDefaultSource(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *Registry) DefaultSourceName() string {
	if r == nil {
		return ""
	}
	return r.defaultSource
}

func (r *Registry) ListSources() []Source {
	if r == nil || len(r.sources) == 0 {
		return nil
	}
	out := make([]Source, 0, len(r.sources))
	for _, source := range r.sources {
		out = append(out, cloneSource(source))
	}
	return out
}

func (r *Registry) ResolveSource(name string) (*Source, error) {
	if r == nil || len(r.sources) == 0 {
		return nil, fmt.Errorf("graphql sources are not configured")
	}
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		source, ok := r.sourceIndex[trimmed]
		if !ok {
			return nil, fmt.Errorf("graphql source %q was not found", trimmed)
		}
		return source, nil
	}
	if len(r.sources) == 1 {
		return &r.sources[0], nil
	}
	if r.defaultSource == "" {
		return nil, fmt.Errorf("source is required when multiple graphql sources are configured without graphql_default_source")
	}
	return r.sourceIndex[r.defaultSource], nil
}

func buildRegistrySources(raw []SourceConfig) ([]Source, map[string]*Source, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}

	sources := make([]Source, 0, len(raw))
	for _, cfg := range raw {
		source, err := buildRegistrySource(cfg)
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})
	index := make(map[string]*Source, len(sources))
	for i := range sources {
		if _, exists := index[sources[i].Name]; exists {
			return nil, nil, fmt.Errorf("graphql source %q is duplicated", sources[i].Name)
		}
		index[sources[i].Name] = &sources[i]
	}
	return sources, index, nil
}

func buildRegistrySource(cfg SourceConfig) (Source, error) {
	schema, err := Load(cfg.SchemaPath)
	if err != nil {
		return Source{}, fmt.Errorf("load graphql schema snapshot for source %q: %w", cfg.Name, err)
	}
	domains, index, err := buildRegistryDomains(cfg.Name, schema, cfg.Domains)
	if err != nil {
		return Source{}, err
	}
	return Source{
		Name:             cfg.Name,
		Description:      cfg.Description,
		Endpoint:         cfg.Endpoint,
		APIKey:           cfg.APIKey,
		SchemaPath:       cfg.SchemaPath,
		TimeoutMS:        cfg.TimeoutMS,
		MaxResponseBytes: cfg.MaxResponseBytes,
		Headers:          cloneHeaders(cfg.Headers),
		Budget: Budget{
			MaxDepth:      cfg.MaxDepth,
			MaxFields:     cfg.MaxFields,
			MaxRootFields: cfg.MaxRootFields,
			MaxFragments:  cfg.MaxFragments,
		},
		Domains:     domains,
		Schema:      schema,
		domainIndex: index,
	}, nil
}

func buildRegistryDomains(
	sourceName string,
	schema *Schema,
	raw []DomainConfig,
) ([]Domain, map[string]*Domain, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}

	domains := make([]Domain, 0, len(raw))
	for _, cfg := range raw {
		domain, err := buildRegistryDomain(sourceName, schema, cfg)
		if err != nil {
			return nil, nil, err
		}
		domains = append(domains, domain)
	}
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name < domains[j].Name
	})
	index := make(map[string]*Domain, len(domains))
	for i := range domains {
		if _, exists := index[domains[i].Name]; exists {
			return nil, nil, fmt.Errorf("graphql source %q domain %q is duplicated", sourceName, domains[i].Name)
		}
		index[domains[i].Name] = &domains[i]
	}
	return domains, index, nil
}

func buildRegistryDomain(sourceName string, schema *Schema, cfg DomainConfig) (Domain, error) {
	if err := validateDomainRootQueries(sourceName, schema, cfg); err != nil {
		return Domain{}, err
	}
	if err := validateDomainTypes(sourceName, schema, cfg); err != nil {
		return Domain{}, err
	}
	rootQuerySet := make(map[string]bool, len(cfg.RootQueries))
	for _, name := range cfg.RootQueries {
		rootQuerySet[name] = true
	}
	typeSet := make(map[string]bool, len(cfg.Types))
	for _, name := range cfg.Types {
		typeSet[name] = true
	}
	return Domain{
		Name:        cfg.Name,
		Description: cfg.Description,
		RootQueries: append([]string(nil), cfg.RootQueries...),
		Types:       append([]string(nil), cfg.Types...),
		Budget: Budget{
			MaxDepth:      cfg.MaxDepth,
			MaxFields:     cfg.MaxFields,
			MaxRootFields: cfg.MaxRootFields,
		},
		rootQuerySet: rootQuerySet,
		typeSet:      typeSet,
	}, nil
}
