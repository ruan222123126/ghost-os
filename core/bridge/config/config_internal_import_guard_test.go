package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestTopLevelProductionFilesStayFacadeOnly(t *testing.T) {
	configDir := currentConfigDir(t)
	allowed := map[string]bool{
		"config.go":            true,
		"internal_adapters.go": true,
		"load.go":              true,
		"preset_activation.go": true,
		"project_root.go":      true,
		"public_defaults.go":   true,
		"public_facades.go":    true,
		"public_runtime.go":    true,
		"public_store.go":      true,
		"store.go":             true,
		"store_domains.go":     true,
		"store_update.go":      true,
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		t.Fatalf("read config dir: %v", err)
	}
	var unexpected []string
	var production []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		production = append(production, name)
		if !allowed[name] {
			unexpected = append(unexpected, name)
		}
	}
	sort.Strings(production)
	sort.Strings(unexpected)
	if len(production) > 15 {
		t.Fatalf("top-level config production files exceed budget: got %d files: %s", len(production), strings.Join(production, ", "))
	}
	if len(unexpected) > 0 {
		t.Fatalf("top-level config production files must stay facade-only; unexpected files: %s", strings.Join(unexpected, ", "))
	}
}

func TestExternalPackagesDoNotImportConfigInternal(t *testing.T) {
	configDir := currentConfigDir(t)
	bridgeDir := filepath.Dir(configDir)
	forbidden := `"ghost-os/bridge/config/internal/`
	var offenders []string

	err := filepath.WalkDir(bridgeDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == configDir {
				return filepath.SkipDir
			}
			switch entry.Name() {
			case ".git", ".ghost":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), forbidden) {
			rel, relErr := filepath.Rel(bridgeDir, path)
			if relErr != nil {
				rel = path
			}
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan bridge imports: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("external packages import config/internal: %s", strings.Join(offenders, ", "))
	}
}

func currentConfigDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Dir(file)
}
