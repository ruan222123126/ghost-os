package memory

import (
	"path/filepath"
	"testing"
)

func newDecisionEnabledManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		WarmCapacity:      32,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   3,
		DecisionEnabled:   true,
		DecisionPath:      filepath.Join(baseDir, "decision"),
	})
}
