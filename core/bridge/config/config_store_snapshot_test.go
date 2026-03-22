package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestConfigStoreRuntimeConfigReturnsDeepClone(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		ActiveProvider: stringPointer("custom"),
		Providers: map[string]providerFileConfig{
			"custom": {
				Type:                     llm.ProviderCustom,
				BaseURL:                  "https://initial.example/v1",
				APIKey:                   stringPointer("initial-key"),
				ModelContextWindowTokens: map[string]int{"gpt-5.4": 8192},
			},
		},
		GraphQLDefaultSource: stringPointer("crm"),
		GraphQLSources: []graphQLSourceFileConfig{{
			Name:       "crm",
			Endpoint:   "https://crm.example/graphql",
			SchemaPath: "/schemas/crm.json",
			Headers:    map[string]string{"X-Tenant": "tenant-1"},
			Domains: []graphQLDomainFileConfig{{
				Name:        "orders",
				RootQueries: []string{"order"},
				Types:       []string{"Order"},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	runtime := store.RuntimeConfig()
	runtime.ModelContextWindowTokens["gpt-5.4"] = 4096
	runtime.GraphQL.Sources[0].Headers["X-Tenant"] = "mutated"
	runtime.GraphQL.Sources[0].Domains[0].RootQueries[0] = "mutated"
	runtime.GraphQL.Sources = append(runtime.GraphQL.Sources, GraphQLSourceConfig{Name: "extra"})

	fresh := store.RuntimeConfig()
	if fresh.ModelContextWindowTokens["gpt-5.4"] != 8192 {
		t.Fatalf("unexpected model context window tokens: %+v", fresh.ModelContextWindowTokens)
	}
	if len(fresh.GraphQL.Sources) != 1 {
		t.Fatalf("unexpected graphql sources: %+v", fresh.GraphQL.Sources)
	}
	source := findGraphQLSource(t, fresh.GraphQL.Sources, "crm")
	if source.Headers["X-Tenant"] != "tenant-1" {
		t.Fatalf("unexpected graphql headers: %+v", source.Headers)
	}
	if source.Domains[0].RootQueries[0] != "order" {
		t.Fatalf("unexpected graphql root queries: %+v", source.Domains[0].RootQueries)
	}
}

func TestConfigStoreListProvidersReturnsConfigReadError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	if err := os.WriteFile(configPath, []byte("providers = ["), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	providers, err := store.ListProviders()
	if err == nil {
		t.Fatal("expected ListProviders to return config read error")
	}
	if providers != nil {
		t.Fatalf("expected nil providers on error, got %+v", providers)
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
