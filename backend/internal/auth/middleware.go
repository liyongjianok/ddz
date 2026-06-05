package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type claimsContextKey struct{}

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

// Middleware 负责从 HTTP 请求中提取并校验访问令牌。
type Middleware struct {
	jwt *JWTManager
}

// NewMiddleware 创建认证中间件。
func NewMiddleware(jwt *JWTManager) *Middleware {
	return &Middleware{jwt: jwt}
}

// RequireAuth 校验 Bearer token，并将解析后的 claims 放入请求上下文。
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.jwt == nil {
			writeUnauthorized(w, r)
			return
		}

		token, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w, r)
			return
		}

		claims, err := m.jwt.Parse(token)
		if err != nil {
			writeUnauthorized(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext 读取认证中间件写入上下文的 claims。
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func extractBearerToken(header string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:      "unauthorized",
		Message:   "unauthorized",
		Data:      nil,
		RequestID: r.Header.Get("X-Request-ID"),
	})
}
