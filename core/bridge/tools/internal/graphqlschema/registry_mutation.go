package graphqlschema

import (
	"fmt"
	"sort"
	"strings"
)

type MutationPolicy struct {
	Name                    string
	Description             string
	Source                  string
	Domain                  string
	RootMutation            string
	IdempotencyMode         string
	IdempotencyHeader       string
	IdempotencyVariablePath string
	Budget                  Budget
}

const (
	mutationIdempotencyModeHeader       = "header"
	mutationIdempotencyModeVariablePath = "variable_path"
)

func (r *Registry) HasMutationPolicies() bool {
	return r != nil && len(r.mutationPolicies) > 0
}

func (r *Registry) SourceHasMutationPolicies(sourceName string) bool {
	if r == nil {
		return false
	}
	source := strings.TrimSpace(sourceName)
	for _, policy := range r.mutationPolicies {
		if policy.Source == source {
			return true
		}
	}
	return false
}

func (r *Registry) ListMutationPolicies(sourceName string, domainName string) ([]MutationPolicy, error) {
	if r == nil {
		return nil, fmt.Errorf("graphql source registry is not configured")
	}
	source, err := r.ResolveSource(sourceName)
	if err != nil {
		return nil, err
	}
	domain := strings.TrimSpace(domainName)
	if domain == "" {
		return nil, fmt.Errorf("domain is required for graphql mutation policies")
	}
	if _, err := source.ResolveDomain(domain); err != nil {
		return nil, err
	}
	out := make([]MutationPolicy, 0, len(r.mutationPolicies))
	for _, policy := range r.mutationPolicies {
		if policy.Source == source.Name && policy.Domain == domain {
			out = append(out, cloneMutationPolicy(policy))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RootMutation == out[j].RootMutation {
			return out[i].Name < out[j].Name
		}
		return out[i].RootMutation < out[j].RootMutation
	})
	return out, nil
}

func (r *Registry) ResolveMutationPolicy(
	sourceName string,
	domainName string,
	rootMutation string,
) (MutationPolicy, error) {
	if r == nil {
		return MutationPolicy{}, fmt.Errorf("graphql source registry is not configured")
	}
	key := mutationPolicyKey(sourceName, domainName, rootMutation)
	policy, ok := r.mutationPolicyKeys[key]
	if !ok {
		return MutationPolicy{}, fmt.Errorf(
			"graphql mutation %q is not allowed for source %q domain %q",
			strings.TrimSpace(rootMutation),
			strings.TrimSpace(sourceName),
			strings.TrimSpace(domainName),
		)
	}
	return cloneMutationPolicy(*policy), nil
}

func buildMutationPolicies(
	raw []MutationPolicyConfig,
	sources map[string]*Source,
) ([]MutationPolicy, map[string]*MutationPolicy, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}

	policies := make([]MutationPolicy, 0, len(raw))
	seenNames := make(map[string]bool, len(raw))
	seenTargets := make(map[string]bool, len(raw))
	for _, cfg := range raw {
		policy, err := buildMutationPolicy(cfg, sources, seenNames, seenTargets)
		if err != nil {
			return nil, nil, err
		}
		policies = append(policies, policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].Source != policies[j].Source {
			return policies[i].Source < policies[j].Source
		}
		if policies[i].Domain != policies[j].Domain {
			return policies[i].Domain < policies[j].Domain
		}
		return policies[i].RootMutation < policies[j].RootMutation
	})
	index := make(map[string]*MutationPolicy, len(policies))
	for i := range policies {
		index[mutationPolicyKey(
			policies[i].Source,
			policies[i].Domain,
			policies[i].RootMutation,
		)] = &policies[i]
	}
	return policies, index, nil
}

func buildMutationPolicy(
	cfg MutationPolicyConfig,
	sources map[string]*Source,
	seenNames map[string]bool,
	seenTargets map[string]bool,
) (MutationPolicy, error) {
	name := strings.TrimSpace(cfg.Name)
	sourceName := strings.TrimSpace(cfg.Source)
	domainName := strings.TrimSpace(cfg.Domain)
	rootMutation := strings.TrimSpace(cfg.RootMutation)
	if name == "" {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy name is required")
	}
	if sourceName == "" {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy %q source is required", name)
	}
	if domainName == "" {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy %q domain is required", name)
	}
	if rootMutation == "" {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy %q root_mutation is required", name)
	}
	if err := validateMutationPolicyIdempotency(name, cfg); err != nil {
		return MutationPolicy{}, err
	}
	if seenNames[name] {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy %q is duplicated", name)
	}
	seenNames[name] = true

	targetKey := mutationPolicyKey(sourceName, domainName, rootMutation)
	if seenTargets[targetKey] {
		return MutationPolicy{}, fmt.Errorf(
			"graphql mutation policy for source %q domain %q root mutation %q is duplicated",
			sourceName,
			domainName,
			rootMutation,
		)
	}
	seenTargets[targetKey] = true

	source, ok := sources[sourceName]
	if !ok {
		return MutationPolicy{}, fmt.Errorf(
			"graphql mutation policy %q source %q was not found",
			name,
			sourceName,
		)
	}
	if _, err := source.ResolveDomain(domainName); err != nil {
		return MutationPolicy{}, fmt.Errorf("graphql mutation policy %q: %w", name, err)
	}
	if _, ok := source.Schema.RootMutationByName(rootMutation); !ok {
		return MutationPolicy{}, fmt.Errorf(
			"graphql mutation policy %q root mutation %q was not found in source %q schema snapshot",
			name,
			rootMutation,
			sourceName,
		)
	}
	budget, err := source.DomainBudget(domainName)
	if err != nil {
		return MutationPolicy{}, err
	}
	if cfg.MaxDepth > 0 {
		budget.MaxDepth = cfg.MaxDepth
	}
	if cfg.MaxFields > 0 {
		budget.MaxFields = cfg.MaxFields
	}
	if cfg.MaxRootFields > 0 {
		budget.MaxRootFields = cfg.MaxRootFields
	}
	if cfg.MaxFragments > 0 {
		budget.MaxFragments = cfg.MaxFragments
	}
	return MutationPolicy{
		Name:                    name,
		Description:             strings.TrimSpace(cfg.Description),
		Source:                  sourceName,
		Domain:                  domainName,
		RootMutation:            rootMutation,
		IdempotencyMode:         strings.TrimSpace(cfg.IdempotencyMode),
		IdempotencyHeader:       strings.TrimSpace(cfg.IdempotencyHeader),
		IdempotencyVariablePath: strings.TrimSpace(cfg.IdempotencyVariablePath),
		Budget:                  budget,
	}, nil
}

func validateMutationPolicyIdempotency(
	name string,
	cfg MutationPolicyConfig,
) error {
	mode := strings.TrimSpace(cfg.IdempotencyMode)
	switch mode {
	case mutationIdempotencyModeHeader:
		if strings.TrimSpace(cfg.IdempotencyHeader) == "" {
			return fmt.Errorf(
				"graphql mutation policy %q idempotency_header is required for idempotency_mode=%q",
				name,
				mode,
			)
		}
		if strings.TrimSpace(cfg.IdempotencyVariablePath) != "" {
			return fmt.Errorf(
				"graphql mutation policy %q idempotency_variable_path must be empty for idempotency_mode=%q",
				name,
				mode,
			)
		}
		return nil
	case mutationIdempotencyModeVariablePath:
		if strings.TrimSpace(cfg.IdempotencyVariablePath) == "" {
			return fmt.Errorf(
				"graphql mutation policy %q idempotency_variable_path is required for idempotency_mode=%q",
				name,
				mode,
			)
		}
		if strings.TrimSpace(cfg.IdempotencyHeader) != "" {
			return fmt.Errorf(
				"graphql mutation policy %q idempotency_header must be empty for idempotency_mode=%q",
				name,
				mode,
			)
		}
		return nil
	default:
		return fmt.Errorf(
			"graphql mutation policy %q idempotency_mode must be one of: %s|%s",
			name,
			mutationIdempotencyModeHeader,
			mutationIdempotencyModeVariablePath,
		)
	}
}

func mutationPolicyKey(sourceName string, domainName string, rootMutation string) string {
	return strings.Join([]string{
		strings.TrimSpace(sourceName),
		strings.TrimSpace(domainName),
		strings.TrimSpace(rootMutation),
	}, "\x00")
}

func cloneMutationPolicy(policy MutationPolicy) MutationPolicy {
	return MutationPolicy{
		Name:                    policy.Name,
		Description:             policy.Description,
		Source:                  policy.Source,
		Domain:                  policy.Domain,
		RootMutation:            policy.RootMutation,
		IdempotencyMode:         policy.IdempotencyMode,
		IdempotencyHeader:       policy.IdempotencyHeader,
		IdempotencyVariablePath: policy.IdempotencyVariablePath,
		Budget:                  policy.Budget,
	}
}
