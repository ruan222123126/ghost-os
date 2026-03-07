package memory

import (
	"strings"
	"testing"
)

func TestIntentPlannerParsesFiveFacets(t *testing.T) {
	metrics := &memoryCounters{}
	planner := NewIntentPlanner(true, metrics)
	plan, err := planner.Plan(MemoryQuery{
		SemanticQuery: "在 /workspace/ghost-os 修复 config migration，不要改 API，避免破坏现有行为，并检查 memory manager",
		Keywords:      []string{"config", "migration", "memory"},
	}, SessionScope{Environment: &DecisionEnvFingerprint{
		Domain:        "coding",
		Platform:      "linux/amd64",
		WorkspaceRoot: "/workspace/ghost-os",
	}})
	if err != nil {
		t.Fatalf("plan query intent: %v", err)
	}
	if !strings.HasPrefix(plan.IntentKey, "intent.fix") {
		t.Fatalf("expected fix intent key, got %q", plan.IntentKey)
	}
	if !containsString(plan.Constraints, "不要改 API") {
		t.Fatalf("expected constraint facet, got %#v", plan.Constraints)
	}
	if !containsString(plan.Entities, "/workspace/ghost-os") || !containsString(plan.Entities, "config") {
		t.Fatalf("expected entity facets, got %#v", plan.Entities)
	}
	if !containsString(plan.Environment, "domain:coding") || !containsString(plan.Environment, "platform:linux/amd64") {
		t.Fatalf("expected environment facets, got %#v", plan.Environment)
	}
	if !containsString(plan.Risks, "避免破坏现有行为") {
		t.Fatalf("expected risk facet, got %#v", plan.Risks)
	}
	if len(plan.Terms) == 0 {
		t.Fatal("expected planner terms to be populated")
	}
	if metrics.snapshot().PlannerRuns != 1 {
		t.Fatalf("expected planner run metric to increment, got %+v", metrics.snapshot())
	}
}

func TestIntentPlannerPrefersScopeEnvironment(t *testing.T) {
	planner := NewIntentPlanner(true, &memoryCounters{})
	plan, err := planner.Plan(MemoryQuery{
		SemanticQuery: "inspect config migration in staging",
		Environment: &DecisionEnvFingerprint{
			Domain:        "web",
			Platform:      "darwin/arm64",
			WorkspaceRoot: "/query-env",
		},
	}, SessionScope{Environment: &DecisionEnvFingerprint{
		Domain:        "coding",
		Platform:      "linux/amd64",
		WorkspaceRoot: "/scope-env",
	}})
	if err != nil {
		t.Fatalf("plan query intent with scope env: %v", err)
	}
	if len(plan.Environment) == 0 || plan.Environment[0] != "domain:coding" {
		t.Fatalf("expected scope environment to win, got %#v", plan.Environment)
	}
	if containsString(plan.Environment, "platform:darwin/arm64") || containsString(plan.Environment, "workspace:/query-env") {
		t.Fatalf("expected query environment to be ignored when scope is present, got %#v", plan.Environment)
	}
}
