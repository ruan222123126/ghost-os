package sessiondraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	contracts "ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type turnDraftFixture struct {
	Events            []streaming.Event          `json:"events"`
	ExpectedTurnDraft contracts.SessionTurnDraft `json:"expected_turn_draft"`
}

func TestProjectTurnDraftFixtures(t *testing.T) {
	fixtures := loadTurnDraftFixtures(t)
	when := time.Unix(10, 0).UTC()

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			sess := &session.Session{}
			for _, event := range fixture.data.Events {
				ProjectTurnDraft(sess, event, when)
			}

			actual := sessionturn.BuildSessionTurnDraftPayload(sess, true)
			if actual == nil {
				t.Fatal("expected turn_draft payload")
			}
			if !reflect.DeepEqual(actual, &fixture.data.ExpectedTurnDraft) {
				t.Fatalf("unexpected projected turn_draft:\n got: %+v\nwant: %+v", actual, fixture.data.ExpectedTurnDraft)
			}
		})
	}
}

type namedTurnDraftFixture struct {
	name string
	data turnDraftFixture
}

func loadTurnDraftFixtures(t *testing.T) []namedTurnDraftFixture {
	t.Helper()

	dir := filepath.Join(testFileDir(t), "..", "..", "..", "..", "..", "shared", "fixtures", "turn_draft")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}

	fixtures := make([]namedTurnDraftFixture, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), readErr)
		}

		var fixture turnDraftFixture
		if unmarshalErr := json.Unmarshal(raw, &fixture); unmarshalErr != nil {
			t.Fatalf("unmarshal fixture %s: %v", entry.Name(), unmarshalErr)
		}
		fixtures = append(fixtures, namedTurnDraftFixture{
			name: entry.Name(),
			data: fixture,
		})
	}

	slices.SortFunc(fixtures, func(left, right namedTurnDraftFixture) int {
		return strings.Compare(left.name, right.name)
	})
	return fixtures
}

func testFileDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file dir")
	}
	return filepath.Dir(file)
}
