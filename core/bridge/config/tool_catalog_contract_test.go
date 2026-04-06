package config

import (
	"sort"
	"testing"

	"ghost-os/bridge/tools"
)

func TestConfiguredToolCatalogMatchesToolMetadata(t *testing.T) {
	catalogNames := normalizedConfiguredToolNames(configuredToolCatalog)
	metadataNames := metadataToolNames(tools.GetToolMetadata())
	if len(catalogNames) != len(metadataNames) {
		t.Fatalf("configured tool catalog mismatch: got %v want %v", catalogNames, metadataNames)
	}
	for index := range catalogNames {
		if catalogNames[index] == metadataNames[index] {
			continue
		}
		t.Fatalf("configured tool catalog mismatch: got %v want %v", catalogNames, metadataNames)
	}
}

func normalizedConfiguredToolNames(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}

func metadataToolNames(items []tools.ToolMetadata) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Name)
	}
	sort.Strings(out)
	return out
}
