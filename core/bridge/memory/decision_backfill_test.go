package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func TestMemoryManagerRebuildDecisionFromColdArchives(t *testing.T) {
	manager, decisionDir := newDecisionRebuildTestManager(t)
	namespace := "workspace:test"
	if err := manager.cold.Archive("session-a", decisionRebuildTestMessages("fix config migration", false)); err != nil {
		t.Fatalf("archive session a: %v", err)
	}
	if err := manager.cold.Archive("session-b", decisionRebuildTestMessages("fix config migration", false)); err != nil {
		t.Fatalf("archive session b: %v", err)
	}
	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:        "node-config",
		SessionID: "session-md",
		Summary:   "Config migration checklist for this repo.",
		SourceIDs: []string{"config.toml"},
		Tags:      []string{"config", "migration"},
		Anchors: []MemoryAnchor{{
			Type:   MemoryAnchorConstraint,
			Key:    "config.toml",
			Value:  "migration",
			Weight: 0.9,
		}},
		CreatedAt:  time.Now().UTC(),
		LastSeenAt: time.Now().UTC(),
		Content:    "## Summary\nPatch config.toml then run the migration check.",
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}

	stats, err := manager.RebuildDecision(DecisionRebuildOptions{Namespace: namespace, RebuildMemos: true})
	if err != nil {
		t.Fatalf("rebuild decision: %v", err)
	}
	if stats.SessionsScanned != 2 {
		t.Fatalf("expected two archived sessions, got %+v", stats)
	}
	if stats.MarkdownScanned != 1 {
		t.Fatalf("expected markdown scan, got %+v", stats)
	}
	if stats.MemosCaptured < 3 {
		t.Fatalf("expected rebuilt memos from archives and markdown, got %+v", stats)
	}
	if got := manager.DecisionStats(namespace).MemoCount; got < 3 {
		t.Fatalf("expected persisted memos, got %d", got)
	}
	if _, err := os.Stat(filepath.Join(decisionDir, defaultDecisionMemosPathName)); err != nil {
		t.Fatalf("expected memo snapshot to exist: %v", err)
	}
}

func TestMemoryManagerRebuildDecisionDryRunDoesNotPersist(t *testing.T) {
	manager, decisionDir := newDecisionRebuildTestManager(t)
	namespace := "workspace:test"
	if err := manager.cold.Archive("session-dry-run", decisionRebuildTestMessages("fix config migration", false)); err != nil {
		t.Fatalf("archive session: %v", err)
	}

	stats, err := manager.RebuildDecision(DecisionRebuildOptions{
		Namespace:      namespace,
		RebuildMemos:   true,
		IncludeRecipes: true,
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("rebuild decision dry run: %v", err)
	}
	if stats.MemosCaptured == 0 {
		t.Fatalf("expected dry run stats to include captured memos, got %+v", stats)
	}
	if _, err := os.Stat(filepath.Join(decisionDir, defaultDecisionMemosPathName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no memo snapshot on dry run, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(decisionDir, defaultDecisionRecipesPathName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no recipe snapshot on dry run, got err=%v", err)
	}
}

func TestMemoryManagerRebuildDecisionResetsNamespaceWhenRequested(t *testing.T) {
	manager, _ := newDecisionRebuildTestManager(t)
	namespace := "workspace:test"
	otherNamespace := "workspace:other"
	if _, err := manager.decision.store.UpsertMemo(DecisionMemo{
		ID:            "memo-old",
		Namespace:     namespace,
		IntentKey:     "intent.fix_config",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		CreatedAt:     time.Now().UTC().Add(-2 * time.Hour),
		LastUsedAt:    time.Now().UTC().Add(-time.Hour),
		Environment:   decisionRebuildTestEnv(namespace),
	}); err != nil {
		t.Fatalf("upsert old memo: %v", err)
	}
	if _, err := manager.decision.store.UpsertMemo(DecisionMemo{
		ID:            "memo-other",
		Namespace:     otherNamespace,
		IntentKey:     "intent.fix_config",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		CreatedAt:     time.Now().UTC().Add(-2 * time.Hour),
		LastUsedAt:    time.Now().UTC().Add(-time.Hour),
		Environment:   decisionRebuildTestEnv(otherNamespace),
	}); err != nil {
		t.Fatalf("upsert other memo: %v", err)
	}
	if err := manager.decision.store.Persist(); err != nil {
		t.Fatalf("persist seed memos: %v", err)
	}
	if err := manager.cold.Archive("session-reset", decisionRebuildTestMessages("fix config migration", false)); err != nil {
		t.Fatalf("archive session: %v", err)
	}

	stats, err := manager.RebuildDecision(DecisionRebuildOptions{
		Namespace:      namespace,
		RebuildMemos:   true,
		ResetNamespace: true,
	})
	if err != nil {
		t.Fatalf("rebuild with reset: %v", err)
	}
	if stats.MemosCaptured != 1 {
		t.Fatalf("expected one rebuilt memo after reset, got %+v", stats)
	}
	if got := manager.DecisionStats(namespace).MemoCount; got != 1 {
		t.Fatalf("expected namespace reset to replace old memos, got %d", got)
	}
	if got := manager.DecisionStats(otherNamespace).MemoCount; got != 1 {
		t.Fatalf("expected other namespace to remain untouched, got %d", got)
	}
}

func TestMemoryManagerRebuildDecisionCanAlsoDistillRecipes(t *testing.T) {
	manager, _ := newDecisionRebuildTestManager(t)
	namespace := "workspace:test"
	for _, sessionID := range []string{"session-r1", "session-r2", "session-r3"} {
		if err := manager.cold.Archive(sessionID, decisionRebuildTestMessages("fix config migration", false)); err != nil {
			t.Fatalf("archive session %s: %v", sessionID, err)
		}
	}

	stats, err := manager.RebuildDecision(DecisionRebuildOptions{
		Namespace:      namespace,
		RebuildMemos:   true,
		IncludeRecipes: true,
	})
	if err != nil {
		t.Fatalf("rebuild and distill recipes: %v", err)
	}
	if stats.RecipesCreated == 0 {
		t.Fatalf("expected recipe creation during rebuild, got %+v", stats)
	}
	decisionStats := manager.DecisionStats(namespace)
	if decisionStats.RecipeCount == 0 {
		t.Fatalf("expected persisted recipes after rebuild, got %+v", decisionStats)
	}
	entries, hits, err := manager.DecisionQuery(MemoryQuery{
		IncludeDecision: true,
		SemanticQuery:   "fix config migration",
		DecisionTypes:   []string{DecisionHitTypeRecipe},
		Metadata:        map[string]any{"namespace": namespace},
	})
	if err != nil {
		t.Fatalf("query rebuilt recipes: %v", err)
	}
	if len(entries) == 0 || len(hits) == 0 || hits[0].Type != DecisionHitTypeRecipe {
		t.Fatalf("expected recipe hit after rebuild, got entries=%d hits=%+v", len(entries), hits)
	}
}

func TestMemoryManagerRebuildDecisionWorksWithoutWorker(t *testing.T) {
	manager, _ := newDecisionRebuildTestManager(t)
	namespace := "workspace:test"
	if err := manager.cold.Archive("session-rule-only", decisionRebuildTestMessages("fix config migration", false)); err != nil {
		t.Fatalf("archive session: %v", err)
	}

	if _, err := manager.RebuildDecision(DecisionRebuildOptions{Namespace: namespace, RebuildMemos: true}); err != nil {
		t.Fatalf("rebuild without worker: %v", err)
	}
	memos := manager.decision.store.ListMemos(namespace)
	if len(memos) == 0 {
		t.Fatal("expected rebuilt memos")
	}
	if memos[0].IntentSummary != "fix config migration" {
		t.Fatalf("expected rule-only rebuild to preserve user intent summary, got %+v", memos[0])
	}
}

func newDecisionRebuildTestManager(t *testing.T) (*MemoryManager, string) {
	t.Helper()
	baseDir := t.TempDir()
	decisionDir := filepath.Join(baseDir, "decision")
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:             16,
		WarmPath:                 filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:              filepath.Join(baseDir, "cold"),
		DecisionEnabled:          true,
		DecisionPath:             decisionDir,
		DecisionRecipeEnabled:    true,
		DecisionRecipeMinSupport: 2,
		DecisionRecipeInterval:   time.Hour,
	})
	return manager, decisionDir
}

func decisionRebuildTestMessages(userMessage string, withError bool) []llm.Message {
	args := json.RawMessage(`{"path":"config.toml"}`)
	resultErr := error(nil)
	resultOutput := "loaded config.toml"
	if withError {
		resultErr = errors.New("config file missing")
		resultOutput = ""
	}
	return []llm.Message{
		{Role: llm.RoleUser, Text: userMessage},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-read", Name: "read_file", Arguments: args}}},
		{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-rebuild", resultOutput, resultErr)},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-edit", Name: "apply_diff", Arguments: json.RawMessage(`{"path":"config.toml","diff":"patch"}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-edit", Text: agent.FormatToolResult("apply_diff", "trace-rebuild", "patched config.toml", nil)},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-check", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"go test ./..."}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-check", Text: agent.FormatToolResult("bash_exec", "trace-rebuild", "migration check passed", nil)},
		{Role: llm.RoleAssistant, Text: `{"signal":"END_SESSION","message":"Config migration fixed successfully."}`},
	}
}

func decisionRebuildTestEnv(namespace string) DecisionEnvFingerprint {
	return DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   namespace,
		Domain:           "coding",
		ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
		ToolsetSignature: "apply_diff,bash_exec,read_file",
	}
}
