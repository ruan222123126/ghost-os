package memory

import (
	"path/filepath"
	"testing"
)

func newDecisionEnabledManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		Warm:     WarmConfig{Capacity: 32, Path: filepath.Join(baseDir, "warm.json"), AutoRecallEnabled: true, AutoRecallLimit: 3},
		Cold:     ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Decision: DecisionConfig{Enabled: true, Path: filepath.Join(baseDir, "decision")},
	})
}
