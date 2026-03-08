package memory

import "testing"

func TestTruthPredicatePolicyDefaults(t *testing.T) {
	if got := truthPredicatePolicyFor("owner_of").Cardinality; got != truthPredicateCardinalitySingleValue {
		t.Fatalf("expected owner_of to be single_value, got %q", got)
	}
	if got := truthPredicatePolicyFor("uses_tool").Cardinality; got != truthPredicateCardinalityMultiValue {
		t.Fatalf("expected uses_tool to be multi_value, got %q", got)
	}
	if got := truthPredicatePolicyFor("unknown.predicate").Cardinality; got != truthPredicateCardinalityMultiValue {
		t.Fatalf("expected unknown predicates to default to multi_value, got %q", got)
	}
}
