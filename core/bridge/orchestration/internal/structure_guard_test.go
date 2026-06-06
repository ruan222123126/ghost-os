package internal_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	maxM2TopLevelGoFiles         = 29
	maxM2ProductionFileLines     = 300
	maxM2OversizedGoDirectories  = 3
	maxGoFilesPerTargetDirectory = 15
)

func TestOrchestrationM2StructureBudget(t *testing.T) {
	root := orchestrationRoot(t)
	files := collectGoFiles(t, root)

	if count := topLevelGoFileCount(root, files); count > maxM2TopLevelGoFiles {
		t.Fatalf("top-level Go files = %d, want <= %d", count, maxM2TopLevelGoFiles)
	}
	if offenders := productionLineOffenders(t, files); len(offenders) > 0 {
		t.Fatalf("production Go files over %d lines: %s", maxM2ProductionFileLines, strings.Join(offenders, ", "))
	}
	if offenders := oversizedDirectories(root, files); len(offenders) > maxM2OversizedGoDirectories {
		t.Fatalf("directories over %d Go files = %d, want <= %d: %s",
			maxGoFilesPerTargetDirectory,
			len(offenders),
			maxM2OversizedGoDirectories,
			strings.Join(offenders, ", "),
		)
	}
}

func TestOrchestrationTopLevelFileAllowlist(t *testing.T) {
	root := orchestrationRoot(t)
	files := collectGoFiles(t, root)
	var offenders []string
	for _, file := range files {
		if filepath.Dir(file) != root {
			continue
		}
		name := filepath.Base(file)
		if !allowedTopLevelOrchestrationFiles[name] {
			offenders = append(offenders, name)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("new top-level orchestration Go files are blocked; only facade, compat, contract, or generated entrypoint files may be allowlisted; otherwise move new code under internal/app, internal/domain, ports, or adapters: %s",
			strings.Join(offenders, ", "))
	}
}

func TestOrchestrationTopLevelFileAllowlistBaselineIsExact(t *testing.T) {
	root := orchestrationRoot(t)
	files := collectGoFiles(t, root)
	current := map[string]bool{}
	for _, file := range files {
		if filepath.Dir(file) != root {
			continue
		}
		current[filepath.Base(file)] = true
	}

	var stale []string
	for name := range allowedTopLevelOrchestrationFiles {
		if current[name] {
			continue
		}
		stale = append(stale, name)
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("top-level orchestration allowlist has stale files; remove migrated entries: %s",
			strings.Join(stale, ", "))
	}
}

func TestDomainConcreteImportFreeze(t *testing.T) {
	imports := domainConcreteImports(t)
	var offenders []string
	for rel, importPaths := range imports {
		for importPath := range importPaths {
			if allowedDomainConcreteImports[rel][importPath] {
				continue
			}
			offenders = append(offenders, rel+" imports "+importPath)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("new concrete imports in internal/domain are blocked (%s and subpackages); use domain contracts or ports instead:\n%s",
			strings.Join(forbiddenConcreteDomainImports, ", "),
			strings.Join(offenders, "\n"))
	}
}

func TestDomainConcreteImportFreezeBaselineIsExact(t *testing.T) {
	imports := domainConcreteImports(t)
	var stale []string
	for rel, allowedImports := range allowedDomainConcreteImports {
		for importPath := range allowedImports {
			if imports[rel][importPath] {
				continue
			}
			stale = append(stale, rel+" allows stale "+importPath)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("domain concrete import freeze baseline has stale allowances; remove migrated entries:\n%s",
			strings.Join(stale, "\n"))
	}
}

func orchestrationRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, ".."))
}

func domainConcreteImports(t *testing.T) map[string]map[string]bool {
	t.Helper()
	root := orchestrationRoot(t)
	domainRoot := filepath.Join(root, "internal", "domain")
	files := collectGoFiles(t, domainRoot)
	imports := map[string]map[string]bool{}
	for _, file := range files {
		rel := relativeToRoot(t, root, file)
		for _, importPath := range importPaths(t, file) {
			if !isForbiddenConcreteDomainImport(importPath) {
				continue
			}
			if imports[rel] == nil {
				imports[rel] = map[string]bool{}
			}
			imports[rel][importPath] = true
		}
	}
	return imports
}

func collectGoFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk orchestration tree: %v", err)
	}
	return files
}

func topLevelGoFileCount(root string, files []string) int {
	count := 0
	for _, file := range files {
		if filepath.Dir(file) == root {
			count++
		}
	}
	return count
}

func relativeToRoot(t *testing.T, root string, file string) string {
	t.Helper()
	rel, err := filepath.Rel(root, file)
	if err != nil {
		t.Fatalf("rel %s to %s: %v", file, root, err)
	}
	return filepath.ToSlash(rel)
}

func importPaths(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports in %s: %v", file, err)
	}
	paths := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		paths = append(paths, strings.Trim(spec.Path.Value, `"`))
	}
	return paths
}

func isForbiddenConcreteDomainImport(importPath string) bool {
	for _, forbidden := range forbiddenConcreteDomainImports {
		if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
			return true
		}
	}
	return false
}

func productionLineOffenders(t *testing.T, files []string) []string {
	t.Helper()
	var offenders []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if lines := lineCount(t, file); lines > maxM2ProductionFileLines {
			offenders = append(offenders, filepath.Base(file))
		}
	}
	return offenders
}

// allowedTopLevelOrchestrationFiles is the Phase 0 freeze baseline, not the
// target shape. Existing top-level business files remain listed only to avoid
// breaking the current tree. New entries should be limited to public facade,
// compatibility shim, contract, generated contract, or top-level contract test
// files; ordinary implementation belongs under internal/*.
var allowedTopLevelOrchestrationFiles = map[string]bool{
	"agent_contract.go":                                                     true,
	"agent_usecase_helpers.go":                                              true,
	"envelope_generated.go":                                                 true,
	"export_service.go":                                                     true,
	"orchestration_contract_agent_stream_misc_test.go":                      true,
	"orchestration_contract_agent_stream_test.go":                           true,
	"orchestration_contract_fixtures_test.go":                               true,
	"orchestration_contract_legacy_migration_test.go":                       true,
	"orchestration_contract_orchestration_owner_dispatch_test.go":           true,
	"orchestration_contract_orchestration_owner_runtime_validation_test.go": true,
	"orchestration_contract_relay_test.go":                                  true,
	"orchestration_contract_schema_config_guards_test.go":                   true,
	"orchestration_contract_session_history_test.go":                        true,
	"orchestration_contract_session_payload_test.go":                        true,
	"orchestration_contract_session_runner_test.go":                         true,
	"orchestration_contract_task_runtime_overrides_test.go":                 true,
	"orchestration_contract_tasks_basic_test.go":                            true,
	"orchestration_contract_workflow_fixtures_test.go":                      true,
	"orchestration_contract_workflow_helpers_test.go":                       true,
	"orchestration_contract_workflow_runner_test.go":                        true,
	"orchestration_contract_workflow_tools_test.go":                         true,
	"orchestration_contract_workflow_validation_api_test.go":                true,
	"service_prompts.go":                                                    true,
	"service_router.go":                                                     true,
	"service_sessions.go":                                                   true,
	"service_usecase_human.go":                                              true,
	"session_runner.go":                                                     true,
	"task_bridge.go":                                                        true,
	"task_executor_adapter.go":                                              true,
}

var forbiddenConcreteDomainImports = []string{
	// Phase 0 freezes these concrete bridge packages out of domain code.
	// Existing debt is tracked in allowedDomainConcreteImports below.
	"ghost-os/bridge/agent",
	"ghost-os/bridge/config",
	"ghost-os/bridge/session",
	"ghost-os/bridge/tasks",
	"ghost-os/bridge/tools",
}

// allowedDomainConcreteImports is the exact Phase 0 debt baseline for domain
// packages. check-layers.sh runs this guard so new concrete imports fail in
// the lightweight layer check. Remove entries as imports move behind
// contracts/ports; adding an entry means consciously accepting new
// domain-to-concrete coupling.
var allowedDomainConcreteImports = map[string]map[string]bool{}

func lineCount(t *testing.T, file string) int {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return strings.Count(string(data), "\n")
}

func oversizedDirectories(root string, files []string) []string {
	counts := map[string]int{}
	for _, file := range files {
		counts[filepath.Dir(file)]++
	}
	var offenders []string
	for dir, count := range counts {
		if count <= maxGoFilesPerTargetDirectory {
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			rel = dir
		}
		offenders = append(offenders, rel)
	}
	return offenders
}
