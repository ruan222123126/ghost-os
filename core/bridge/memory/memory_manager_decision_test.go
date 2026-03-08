package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryManagerStopDreamingStopsDecisionDistiller(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:             8,
		WarmPath:                 filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:              filepath.Join(baseDir, "cold"),
		DecisionEnabled:          true,
		DecisionPath:             filepath.Join(baseDir, "decision"),
		DecisionRecipeEnabled:    true,
		DecisionRecipeInterval:   10 * time.Millisecond,
		DecisionRecipeMinSupport: 2,
	})
	if manager.decision == nil || !manager.decision.HasDistiller() {
		t.Fatal("expected decision distiller to be configured")
	}

	manager.StopDreaming()
	select {
	case <-manager.decision.debugDistillerStopCh():
	default:
		t.Fatal("expected decision distiller stop channel to be closed")
	}
	manager.StopDreaming()
}

func TestMemoryManagerDoesNotCreateDecisionDistillerWhenRecipeDisabled(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:    8,
		WarmPath:        filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:     filepath.Join(baseDir, "cold"),
		DecisionEnabled: true,
		DecisionPath:    filepath.Join(baseDir, "decision"),
	})
	if manager.decision == nil {
		t.Fatal("expected decision service to be configured")
	}
	if manager.decision.HasDistiller() {
		t.Fatal("expected no decision distiller when recipe feature is disabled")
	}
}
