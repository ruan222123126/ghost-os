package runtime

import "testing"

func TestDefaultExecutionAllowedPathsUsesProjectRootFallback(t *testing.T) {
	read, write := defaultExecutionAllowedPaths(nil, nil, " /tmp/ghost-os ")

	if len(read) != 1 || read[0] != "/tmp/ghost-os" {
		t.Fatalf("unexpected read paths: %#v", read)
	}
	if len(write) != 1 || write[0] != "/tmp/ghost-os" {
		t.Fatalf("unexpected write paths: %#v", write)
	}
}

func TestDefaultExecutionAllowedPathsPreservesExplicitPaths(t *testing.T) {
	read, write := defaultExecutionAllowedPaths(
		[]string{"/read/a"},
		[]string{"/write/a"},
		"/tmp/ghost-os",
	)

	if len(read) != 1 || read[0] != "/read/a" {
		t.Fatalf("unexpected read paths: %#v", read)
	}
	if len(write) != 1 || write[0] != "/write/a" {
		t.Fatalf("unexpected write paths: %#v", write)
	}
}
