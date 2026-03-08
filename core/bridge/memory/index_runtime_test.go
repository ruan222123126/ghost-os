package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestMemoryManagerRunIndexOnceProjectsGraphFromLedgerAndWritesCheckpoint(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm: WarmConfig{Capacity: 8, Path: filepath.Join(baseDir, "warm.json")},
		Cold: ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Ledger: LedgerConfig{
			DualWrite: true,
		},
		Index: IndexConfig{
			Enabled:      true,
			PollInterval: time.Hour,
			BatchSize:    16,
		},
		Graph: GraphConfig{
			Enabled:          true,
			Path:             filepath.Join(baseDir, "graph"),
			ExtractOnArchive: true,
			Namespace:        "workspace:test",
		},
	})
	defer manager.StopDreaming()

	messages := []llm.Message{
		{Role: llm.RoleUser, Text: "Alice owns migration-project."},
		{Role: llm.RoleAssistant, Text: "migration-project depends on config.toml."},
	}
	if err := manager.AppendLedgerTurn("session-index-graph", "trace-index-graph", 0, messages); err != nil {
		t.Fatalf("append ledger turn: %v", err)
	}
	if err := manager.RunIndexOnce(context.Background()); err != nil {
		t.Fatalf("run index once: %v", err)
	}

	stats := manager.GraphStats("workspace:test")
	if stats.NodeCount == 0 || stats.EdgeCount == 0 {
		t.Fatalf("expected graph stats after runtime projection, got %+v", stats)
	}
	checkpointPath := filepath.Join(baseDir, "cold", "ledger", "_index", "graph", "default", "_", time.Now().UTC().Format("2006-01"), "checkpoint.json")
	if _, err := os.Stat(checkpointPath); err != nil {
		t.Fatalf("expected graph checkpoint at %s: %v", checkpointPath, err)
	}
	viewPath := filepath.Join(baseDir, "cold", "ledger", "_index", "graph", "default", "_", time.Now().UTC().Format("2006-01"), "view_manifest.json")
	if _, err := os.Stat(viewPath); err != nil {
		t.Fatalf("expected graph view manifest at %s: %v", viewPath, err)
	}
}

func TestMemoryManagerRunIndexOnceWritesDecisionShadowStore(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm: WarmConfig{Capacity: 8, Path: filepath.Join(baseDir, "warm.json")},
		Cold: ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Ledger: LedgerConfig{
			DualWrite: true,
		},
		Index: IndexConfig{
			Enabled:       true,
			ShadowCompare: true,
			PollInterval:  time.Hour,
			BatchSize:     16,
		},
		Decision: DecisionConfig{
			Enabled:       true,
			CaptureOnTurn: true,
			Path:          filepath.Join(baseDir, "decision"),
		},
		Graph: GraphConfig{Namespace: "workspace:test"},
	})
	defer manager.StopDreaming()

	messages := []llm.Message{
		{Role: llm.RoleUser, Text: "Please update the deployment checklist."},
		{Role: llm.RoleAssistant, Text: "Checklist updated and verified."},
	}
	if err := manager.AppendLedgerTurn("session-index-decision", "trace-index-decision", 0, messages); err != nil {
		t.Fatalf("append ledger turn: %v", err)
	}
	if err := manager.RunIndexOnce(context.Background()); err != nil {
		t.Fatalf("run index once: %v", err)
	}

	shadowMemosPath := filepath.Join(baseDir, "decision", "_index_shadow", "decision", defaultDecisionMemosPathName)
	data, err := os.ReadFile(shadowMemosPath)
	if err != nil {
		t.Fatalf("read decision shadow memos: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected decision shadow memos to be written at %s", shadowMemosPath)
	}
	mainMemosPath := filepath.Join(baseDir, "decision", defaultDecisionMemosPathName)
	if _, err := os.Stat(mainMemosPath); err == nil {
		t.Fatalf("expected live decision store to remain untouched when shadow compare is enabled: %s", mainMemosPath)
	}
	checkpointPath := filepath.Join(baseDir, "cold", "ledger", "_index", "decision", "default", "_", time.Now().UTC().Format("2006-01"), "checkpoint.json")
	if _, err := os.Stat(checkpointPath); err != nil {
		t.Fatalf("expected decision checkpoint at %s: %v", checkpointPath, err)
	}
	viewPath := filepath.Join(baseDir, "cold", "ledger", "_index", "decision", "default", "_", time.Now().UTC().Format("2006-01"), "view_manifest.json")
	if _, err := os.Stat(viewPath); err != nil {
		t.Fatalf("expected decision view manifest at %s: %v", viewPath, err)
	}
}
