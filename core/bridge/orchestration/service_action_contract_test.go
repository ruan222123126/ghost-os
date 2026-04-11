package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestSchemaActionEnumMatchesRegisteredBusActions(t *testing.T) {
	schemaActions := loadSchemaActionEnum(t)
	runtimeActions := registeredBusActions()

	missingInRuntime := actionDiff(schemaActions, runtimeActions)
	if len(missingInRuntime) != 0 {
		t.Fatalf("schema declares unsupported bus actions: %v", missingInRuntime)
	}

	missingInSchema := actionDiff(runtimeActions, schemaActions)
	if len(missingInSchema) != 0 {
		t.Fatalf("runtime bus actions missing from schema enum: %v", missingInSchema)
	}
}

func loadSchemaActionEnum(t *testing.T) map[string]struct{} {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "base.json"))

	type actionSchema struct {
		Defs struct {
			Action struct {
				Enum []string `json:"enum"`
			} `json:"action"`
		} `json:"$defs"`
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read base schema: %v", err)
	}

	var parsed actionSchema
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode base schema: %v", err)
	}

	actions := make(map[string]struct{}, len(parsed.Defs.Action.Enum))
	for _, action := range parsed.Defs.Action.Enum {
		normalized := strings.TrimSpace(action)
		if normalized != "" {
			actions[normalized] = struct{}{}
		}
	}
	return actions
}

func registeredBusActions() map[string]struct{} {
	service := newBridgeServiceState(nil, nil)
	registerDefaultActions(service)

	names := service.registeredActionNames()
	actions := make(map[string]struct{}, len(names))
	for _, action := range names {
		actions[action] = struct{}{}
	}
	return actions
}

func actionDiff(left map[string]struct{}, right map[string]struct{}) []string {
	missing := make([]string, 0, len(left))
	for action := range left {
		if _, ok := right[action]; !ok {
			missing = append(missing, action)
		}
	}
	sort.Strings(missing)
	return missing
}
