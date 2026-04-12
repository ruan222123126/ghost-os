package orchestration

import (
	"net/http"
	"strings"
	"testing"
)

func TestEnsureSessionActiveRequiresSessionStoreForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("session-1")
	if err == nil {
		t.Fatal("expected error when session store is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := legacyStatusFromServiceError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "session store is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionActiveSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("   ")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}

func TestEnsureSessionNotInflightRequiresRunRegistryForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("session-1")
	if err == nil {
		t.Fatal("expected error when run registry is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := legacyStatusFromServiceError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "run registry is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionNotInflightSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}
