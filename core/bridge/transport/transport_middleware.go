// Transport middleware handles cross-cutting concerns like tracing and panic-safe responses.

package transport

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type corsPolicy struct {
	allowedOrigins map[string]struct{}
}

func newCORSPolicyFromConfig(fileCfg bridgeFileConfig) corsPolicy {
	allowed := make(map[string]struct{})
	origins := corsOriginsOrEnv(fileCfg.CORSOrigins)
	if len(origins) == 0 {
		return corsPolicy{allowedOrigins: allowed}
	}

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return corsPolicy{allowedOrigins: allowed}
}

// newCORSPolicyFromEnv 优先从配置文件读取 CORS 白名单，缺省时回退 GHOST_CORS_ORIGINS。
// 注意：当配置文件存在但读取/解析失败时，返回错误以避免 fail-open。
func newCORSPolicyFromEnv() (corsPolicy, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return corsPolicy{}, err
	}
	return newCORSPolicyFromConfig(fileCfg), nil
}

// allows 判定 origin 是否允许；无 Origin（同源/非浏览器）默认放行。
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

func newAPITokenAuthFromConfig(fileCfg bridgeFileConfig) apiTokenAuth {
	return apiTokenAuth{token: strings.TrimSpace(valueOrEnv(fileCfg.APIToken, "GHOST_API_TOKEN", ""))}
}

// newAPITokenAuthFromEnv 优先读取配置文件中的 API Token，缺省时回退 GHOST_API_TOKEN。
// 注意：当配置文件存在但读取/解析失败时，返回错误以避免 fail-open。
func newAPITokenAuthFromEnv() (apiTokenAuth, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return apiTokenAuth{}, err
	}
	return newAPITokenAuthFromConfig(fileCfg), nil
}

// enabled 表示是否启用 token 认证。
func (a apiTokenAuth) enabled() bool {
	return a.token != ""
}

// authorized 支持 X-API-Token 与 Bearer 两种传参方式，并使用常量时间比较。
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

// parseBearerToken 从 Authorization: Bearer <token> 中提取 token。
func parseBearerToken(header string) string {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// withAuth 为业务路由挂载认证中间件；OPTIONS 请求直接放行。
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

// withCORS 统一设置跨域响应头并处理预检请求。
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
