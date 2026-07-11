package transport

import (
	"net/http"
	"testing"
)

func TestAuthMissingOrWrongTokenReturnsUnauthorized(t *testing.T) {
	t.Setenv("GHOST_API_TOKEN", "secret-token")
	handler := newTestHandler(t, nil)

	missing := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status without token: got %d want %d", missing.Code, http.StatusUnauthorized)
	}

	wrong := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"X-API-Token": "wrong"})
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status with wrong token: got %d want %d", wrong.Code, http.StatusUnauthorized)
	}
}

func TestAuthAcceptsXAPITokenAndBearer(t *testing.T) {
	t.Setenv("GHOST_API_TOKEN", "secret-token")
	handler := newTestHandler(t, nil)

	xToken := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"X-API-Token": "secret-token"})
	if xToken.Code != http.StatusOK {
		t.Fatalf("unexpected status with X-API-Token: got %d want %d", xToken.Code, http.StatusOK)
	}

	bearer := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"Authorization": "Bearer secret-token"})
	if bearer.Code != http.StatusOK {
		t.Fatalf("unexpected status with bearer token: got %d want %d", bearer.Code, http.StatusOK)
	}
}

func TestWithCORSAllowlist(t *testing.T) {
	t.Setenv("GHOST_CORS_ORIGINS", "https://console.ghost.local,http://localhost:5173")
	handler := newTestHandler(t, nil)

	allowed := serveRequest(handler, http.MethodOptions, "/api/config", "", map[string]string{"Origin": "http://localhost:5173"})
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("unexpected status for allowed origin: got %d want %d", allowed.Code, http.StatusNoContent)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin: got %q want %q", got, "http://localhost:5173")
	}

	forbidden := serveRequest(handler, http.MethodOptions, "/api/config", "", map[string]string{"Origin": "https://evil.example"})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("unexpected status for disallowed origin: got %d want %d", forbidden.Code, http.StatusForbidden)
	}
}
