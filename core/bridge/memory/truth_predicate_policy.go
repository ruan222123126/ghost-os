package memory

import "strings"

const (
	truthPredicateCardinalitySingleValue = "single_value"
	truthPredicateCardinalityMultiValue  = "multi_value"
)

type TruthPredicatePolicy struct {
	Predicate   string `json:"predicate,omitempty"`
	Cardinality string `json:"cardinality,omitempty"`
}

var truthPredicatePolicies = map[string]TruthPredicatePolicy{
	"owner_of":            {Predicate: "owner_of", Cardinality: truthPredicateCardinalitySingleValue},
	"preference.language": {Predicate: "preference.language", Cardinality: truthPredicateCardinalitySingleValue},
	"intent.key":          {Predicate: "intent.key", Cardinality: truthPredicateCardinalitySingleValue},
	"outcome.summary":     {Predicate: "outcome.summary", Cardinality: truthPredicateCardinalitySingleValue},
	"uses_tool":           {Predicate: "uses_tool", Cardinality: truthPredicateCardinalityMultiValue},
	"avoids_pattern":      {Predicate: "avoids_pattern", Cardinality: truthPredicateCardinalityMultiValue},
	"constraint.has":      {Predicate: "constraint.has", Cardinality: truthPredicateCardinalityMultiValue},
	"validation.check":    {Predicate: "validation.check", Cardinality: truthPredicateCardinalityMultiValue},
	"entity.ref":          {Predicate: "entity.ref", Cardinality: truthPredicateCardinalityMultiValue},
	"anchor.preference":   {Predicate: "anchor.preference", Cardinality: truthPredicateCardinalityMultiValue},
	"anchor.constraint":   {Predicate: "anchor.constraint", Cardinality: truthPredicateCardinalityMultiValue},
	"anchor.avoidance":    {Predicate: "anchor.avoidance", Cardinality: truthPredicateCardinalityMultiValue},
	"failure_reason":      {Predicate: "failure_reason", Cardinality: truthPredicateCardinalityMultiValue},
}

func truthPredicatePolicyFor(predicate string) TruthPredicatePolicy {
	normalized := truthIndexKey(predicate)
	if policy, ok := truthPredicatePolicies[normalized]; ok {
		return policy
	}
	return TruthPredicatePolicy{Predicate: normalized, Cardinality: truthPredicateCardinalityMultiValue}
}

func normalizeTruthPredicateCardinality(cardinality string) string {
	if strings.TrimSpace(cardinality) == truthPredicateCardinalitySingleValue {
		return truthPredicateCardinalitySingleValue
	}
	return truthPredicateCardinalityMultiValue
}
