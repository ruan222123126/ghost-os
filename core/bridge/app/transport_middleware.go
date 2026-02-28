package app

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type corsPolicy struct {
	allowedOrigins map[string]struct{}
}

func newCORSPolicyFromEnv() corsPolicy {
	raw := strings.TrimSpace(getenvDefault("GHOST_CORS_ORIGINS", ""))
	allowed := make(map[string]struct{})
	if raw == "" {
		return corsPolicy{allowedOrigins: allowed}
	}

	for _, origin := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return corsPolicy{allowedOrigins: allowed}
}

func (p corsPolicy) allows(origin string) bool {
	if origin == "" {
		return true
	}
	_, ok := p.allowedOrigins[origin]
	return ok
}

type apiTokenAuth struct {
	token string
}

func newAPITokenAuthFromEnv() apiTokenAuth {
	return apiTokenAuth{token: strings.TrimSpace(getenvDefault("GHOST_API_TOKEN", ""))}
}

func (a apiTokenAuth) enabled() bool {
	return a.token != ""
}

func (a apiTokenAuth) authorized(r *http.Request) bool {
	if !a.enabled() {
		return true
	}

	provided := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if provided == "" {
		provided = parseBearerToken(r.Header.Get("Authorization"))
	}
	if provided == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) == 1
}

func parseBearerToken(header string) string {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func withAuth(auth apiTokenAuth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if auth.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized", "")
	})
}

func withCORS(policy corsPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && !policy.allows(origin) {
			writeError(w, http.StatusForbidden, "origin is not allowed", "")
			return
		}

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Trace-ID, X-API-Token, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
