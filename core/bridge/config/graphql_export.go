package config

type GraphQLDomainFileConfig = graphQLDomainFileConfig
type GraphQLSourceFileConfig = graphQLSourceFileConfig
type GraphQLMutationPolicyFileConfig = graphQLMutationPolicyFileConfig

type GraphQLDomainSnapshot struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	RootQueries   []string `json:"root_queries,omitempty"`
	Types         []string `json:"types,omitempty"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	MaxFields     int      `json:"max_fields,omitempty"`
	MaxRootFields int      `json:"max_root_fields,omitempty"`
}

type GraphQLSourceSnapshot struct {
	Name             string                  `json:"name"`
	Description      string                  `json:"description,omitempty"`
	Endpoint         string                  `json:"endpoint"`
	SchemaPath       string                  `json:"schema_path"`
	TimeoutMS        int                     `json:"timeout_ms"`
	MaxResponseBytes int                     `json:"max_response_bytes"`
	MaxDepth         int                     `json:"max_depth"`
	MaxFields        int                     `json:"max_fields"`
	MaxRootFields    int                     `json:"max_root_fields"`
	MaxFragments     int                     `json:"max_fragments"`
	Headers          map[string]string       `json:"headers,omitempty"`
	APIKeySet        bool                    `json:"api_key_set"`
	Domains          []GraphQLDomainSnapshot `json:"domains,omitempty"`
}

type GraphQLMutationPolicySnapshot struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Source        string `json:"source"`
	Domain        string `json:"domain"`
	RootMutation  string `json:"root_mutation"`
	MaxDepth      int    `json:"max_depth,omitempty"`
	MaxFields     int    `json:"max_fields,omitempty"`
	MaxRootFields int    `json:"max_root_fields,omitempty"`
	MaxFragments  int    `json:"max_fragments,omitempty"`
}

type GraphQLDomainInput struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	RootQueries   []string `json:"root_queries,omitempty"`
	Types         []string `json:"types,omitempty"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	MaxFields     int      `json:"max_fields,omitempty"`
	MaxRootFields int      `json:"max_root_fields,omitempty"`
}

type GraphQLSourceInput struct {
	Name             string               `json:"name"`
	Description      string               `json:"description,omitempty"`
	Endpoint         string               `json:"endpoint,omitempty"`
	APIKey           *string              `json:"api_key,omitempty"`
	SchemaPath       string               `json:"schema_path,omitempty"`
	TimeoutMS        int                  `json:"timeout_ms,omitempty"`
	MaxResponseBytes int                  `json:"max_response_bytes,omitempty"`
	Headers          map[string]string    `json:"headers,omitempty"`
	MaxDepth         int                  `json:"max_depth,omitempty"`
	MaxFields        int                  `json:"max_fields,omitempty"`
	MaxRootFields    int                  `json:"max_root_fields,omitempty"`
	MaxFragments     int                  `json:"max_fragments,omitempty"`
	Domains          []GraphQLDomainInput `json:"domains,omitempty"`
}

type GraphQLMutationPolicyInput struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Source        string `json:"source"`
	Domain        string `json:"domain"`
	RootMutation  string `json:"root_mutation"`
	MaxDepth      int    `json:"max_depth,omitempty"`
	MaxFields     int    `json:"max_fields,omitempty"`
	MaxRootFields int    `json:"max_root_fields,omitempty"`
	MaxFragments  int    `json:"max_fragments,omitempty"`
}
