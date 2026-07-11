package session

import (
	"testing"

	"ghost-os/bridge/llm"
)

func TestStoreSearchMetadataMatchesUserAndAssistantMessages(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	userMatch := NewSession("system noise")
	userMatch.ID = "session-user-match"
	userMatch.Title = "User match"
	userMatch.AddMessage(llm.Message{Role: llm.RoleUser, Text: "please remember rare-user-term"})
	if err := store.Save(userMatch); err != nil {
		t.Fatalf("save user match: %v", err)
	}

	contentMatch := NewSession("system noise")
	contentMatch.ID = "session-content-match"
	contentMatch.Title = "Content match"
	contentMatch.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		Content: []llm.ContentPart{{
			Type: llm.ContentTypeText,
			Text: "assistant content includes rare-content-term",
		}},
	})
	if err := store.Save(contentMatch); err != nil {
		t.Fatalf("save content match: %v", err)
	}

	noise := NewSession("rare-noise-term in system prompt")
	noise.ID = "session-noise"
	noise.Title = "Noise"
	noise.AddMessage(llm.Message{Role: llm.RoleTool, Text: "rare-noise-term in tool output"})
	if err := store.Save(noise); err != nil {
		t.Fatalf("save noise: %v", err)
	}

	assertSearchIDs(t, store, "rare-user-term", []string{userMatch.ID})
	assertSearchIDs(t, store, "rare-content-term", []string{contentMatch.ID})
	assertSearchIDs(t, store, "rare-noise-term", nil)
}

func assertSearchIDs(t *testing.T, store *Store, query string, expected []string) {
	t.Helper()

	results, err := store.SearchMetadata(query, 0)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	if len(results) != len(expected) {
		t.Fatalf("unexpected result count for %q: got=%d want=%d results=%+v", query, len(results), len(expected), results)
	}
	for index, id := range expected {
		if results[index].ID != id {
			t.Fatalf("unexpected result[%d] for %q: got=%q want=%q", index, query, results[index].ID, id)
		}
	}
}
