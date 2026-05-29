package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	maxM2TopLevelGoFiles         = 70
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

func orchestrationRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, ".."))
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
