package graphqlschema

import (
	"fmt"
	"strings"
)

type mutationPolicyBase struct {
	Name         string
	Source       string
	Domain       string
	RootMutation string
}

func parseMutationPolicyBase(cfg MutationPolicyConfig) (mutationPolicyBase, error) {
	base := mutationPolicyBase{
		Name:         strings.TrimSpace(cfg.Name),
		Source:       strings.TrimSpace(cfg.Source),
		Domain:       strings.TrimSpace(cfg.Domain),
		RootMutation: strings.TrimSpace(cfg.RootMutation),
	}
	if base.Name == "" {
		return mutationPolicyBase{}, fmt.Errorf("graphql mutation policy name is required")
	}
	if base.Source == "" {
		return mutationPolicyBase{}, fmt.Errorf("graphql mutation policy %q source is required", base.Name)
	}
	if base.Domain == "" {
		return mutationPolicyBase{}, fmt.Errorf("graphql mutation policy %q domain is required", base.Name)
	}
	if base.RootMutation == "" {
		return mutationPolicyBase{}, fmt.Errorf(
			"graphql mutation policy %q root_mutation is required",
			base.Name,
		)
	}
	return base, nil
}

func markMutationPolicyName(name string, seenNames map[string]bool) error {
	if seenNames[name] {
		return fmt.Errorf("graphql mutation policy %q is duplicated", name)
	}
	seenNames[name] = true
	return nil
}

func markMutationPolicyTarget(base mutationPolicyBase, seenTargets map[string]bool) error {
	targetKey := mutationPolicyKey(base.Source, base.Domain, base.RootMutation)
	if seenTargets[targetKey] {
		return fmt.Errorf(
			"graphql mutation policy for source %q domain %q root mutation %q is duplicated",
			base.Source,
			base.Domain,
			base.RootMutation,
		)
	}
	seenTargets[targetKey] = true
	return nil
}

func resolveMutationPolicySource(
	policyName string,
	sourceName string,
	sources map[string]*Source,
) (*Source, error) {
	source, ok := sources[sourceName]
	if !ok {
		return nil, fmt.Errorf(
			"graphql mutation policy %q source %q was not found",
			policyName,
			sourceName,
		)
	}
	return source, nil
}

func ensureMutationPolicyTargetExists(base mutationPolicyBase, source *Source) error {
	if _, err := source.ResolveDomain(base.Domain); err != nil {
		return fmt.Errorf("graphql mutation policy %q: %w", base.Name, err)
	}
	if _, ok := source.Schema.RootMutationByName(base.RootMutation); !ok {
		return fmt.Errorf(
			"graphql mutation policy %q root mutation %q was not found in source %q schema snapshot",
			base.Name,
			base.RootMutation,
			base.Source,
		)
	}
	return nil
}

func resolveMutationPolicyBudget(
	cfg MutationPolicyConfig,
	source *Source,
	domainName string,
) (Budget, error) {
	budget, err := source.DomainBudget(domainName)
	if err != nil {
		return Budget{}, err
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
	return budget, nil
}
