package transport

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var forbiddenTransportImports = map[string]string{
	"ghost-os/bridge/session":   "session domain access belongs behind orchestration/session usecases",
	"ghost-os/bridge/artifacts": "artifact store access belongs behind orchestration/session usecases",
	"ghost-os/bridge/config":    "config domain access belongs behind orchestration/config usecases",
	"ghost-os/bridge/skills":    "skills domain access belongs behind orchestration/skill usecases",
	"os":                        "business file IO belongs behind orchestration usecases",
	"path/filepath":             "business path validation belongs behind orchestration usecases",
	"mime":                      "business download MIME resolution belongs behind orchestration usecases",
}

var allowedTransportImportDebt = map[string]map[string]string{}

var forbiddenTransportContent = map[string]string{
	"TaskScopeKind":              "task scope selection belongs behind orchestration task facades",
	"taskScopeKind":              "task scope selection belongs behind orchestration task facades",
	"orchestrationTaskScopeKind": "task scope selection belongs behind orchestration task facades",
	"Scope:":                     "task scope field injection belongs behind orchestration task facades",
}

func TestTransportProductionImportGuard(t *testing.T) {
	dir := transportPackageDir(t)
	seen := map[string]map[string]bool{}
	violations := scanTransportProductionImports(t, dir, seen)
	violations = append(violations, staleTransportImportAllowlistEntries(seen)...)
	if len(violations) == 0 {
		return
	}

	sort.Strings(violations)
	t.Fatalf("transport production import guard failed:\n%s", strings.Join(violations, "\n"))
}

func TestTransportProductionScopeGuard(t *testing.T) {
	dir := transportPackageDir(t)
	violations := scanTransportProductionContent(t, dir, forbiddenTransportContent)
	if len(violations) == 0 {
		return
	}

	sort.Strings(violations)
	t.Fatalf("transport production scope guard failed:\n%s", strings.Join(violations, "\n"))
}

func transportPackageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve transport package dir: runtime caller unavailable")
	}
	return filepath.Dir(file)
}

func scanTransportProductionContent(
	t *testing.T,
	dir string,
	forbidden map[string]string,
) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read transport package dir: %v", err)
	}

	violations := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read transport production file %s: %v", name, err)
		}
		for marker, reason := range forbidden {
			if strings.Contains(string(content), marker) {
				violations = append(violations, name+" contains "+marker+": "+reason)
			}
		}
	}
	return violations
}

func scanTransportProductionImports(t *testing.T, dir string, seen map[string]map[string]bool) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read transport package dir: %v", err)
	}

	violations := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		violations = append(violations, scanTransportFileImports(t, dir, name, seen)...)
	}
	return violations
}

func scanTransportFileImports(
	t *testing.T,
	dir string,
	name string,
	seen map[string]map[string]bool,
) []string {
	t.Helper()
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, name), nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse transport imports for %s: %v", name, err)
	}

	violations := []string{}
	for _, imported := range parsed.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatalf("decode import path in %s: %v", name, err)
		}
		if _, ok := seen[name]; !ok {
			seen[name] = map[string]bool{}
		}
		seen[name][path] = true

		reason, forbidden := forbiddenTransportImports[path]
		if !forbidden || transportImportDebtAllowed(name, path) {
			continue
		}
		violations = append(violations, name+" imports "+path+": "+reason)
	}
	return violations
}

func transportImportDebtAllowed(file string, importPath string) bool {
	allowlist := allowedTransportImportDebt[file]
	if len(allowlist) == 0 {
		return false
	}
	return strings.TrimSpace(allowlist[importPath]) != ""
}

func staleTransportImportAllowlistEntries(seen map[string]map[string]bool) []string {
	violations := []string{}
	for file, imports := range allowedTransportImportDebt {
		for importPath, reason := range imports {
			if _, ok := forbiddenTransportImports[importPath]; !ok {
				violations = append(violations, file+" allowlists non-forbidden import "+importPath)
				continue
			}
			if strings.TrimSpace(reason) == "" {
				violations = append(violations, file+" allowlists "+importPath+" without a TODO reason")
			}
			if !seen[file][importPath] {
				violations = append(violations, file+" allowlists absent import "+importPath)
			}
		}
	}
	return violations
}
