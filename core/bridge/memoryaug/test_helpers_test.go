package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

type scriptedExtractor struct {
	outputs []ExtractOutput
	calls   int
}

func (e *scriptedExtractor) Extract(context.Context, ExtractInput) (ExtractOutput, error) {
	e.calls++
	if len(e.outputs) == 0 {
		return ExtractOutput{}, nil
	}
	output := e.outputs[0]
	e.outputs = e.outputs[1:]
	return output, nil
}

type scriptedEventExtractor struct {
	outputs []EventExtractOutput
	calls   int
}

func (e *scriptedEventExtractor) Extract(context.Context, EventExtractInput) (EventExtractOutput, error) {
	e.calls++
	if len(e.outputs) == 0 {
		return EventExtractOutput{}, nil
	}
	output := e.outputs[0]
	e.outputs = e.outputs[1:]
	return output, nil
}

func newTestStore(t *testing.T) *memorystore.Store {
	t.Helper()
	store, err := memorystore.NewStore(t.TempDir() + "/memory.db")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func newTestSettings() Settings {
	return Settings{
		Enabled:             true,
		LearningEnabled:     true,
		RecallEnabled:       true,
		MaxRecallItems:      3,
		MinConfidence:       0.7,
		SessionScopeEnabled: true,
		UserScopeEnabled:    true,
		UserScopeID:         memorystore.DefaultUserScopeID,
	}
}
