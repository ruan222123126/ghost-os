package config

type GraphQLDomainConfig struct {
	Name          string
	Description   string
	RootQueries   []string
	Types         []string
	MaxDepth      int
	MaxFields     int
	MaxRootFields int
}

type GraphQLSourceConfig struct {
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
	Domains          []GraphQLDomainConfig
}

type GraphQLMutationPolicyConfig struct {
	Name                    string
	Description             string
	Source                  string
	Domain                  string
	RootMutation            string
	ApprovalRequired        bool
	IdempotencyMode         string
	IdempotencyHeader       string
	IdempotencyVariablePath string
	MaxDepth                int
	MaxFields               int
	MaxRootFields           int
	MaxFragments            int
}

type GraphQLConfig struct {
	ToolRuntimeEnabled  bool
	TextSanitizeEnabled bool
	DefaultSource       string
	Sources             []GraphQLSourceConfig
	MutationPolicies    []GraphQLMutationPolicyConfig
}
