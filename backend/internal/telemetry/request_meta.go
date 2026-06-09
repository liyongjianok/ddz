package telemetry

import "context"

type requestMetaContextKey struct{}

// RequestMeta 保存一次请求或连接的基础观测上下文。
type RequestMeta struct {
	TraceID string
	UserID  string
	RoomID  string
	GameID  string
}

// WithRequestMeta 将请求观测信息写入上下文。
func WithRequestMeta(ctx context.Context, meta *RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaContextKey{}, meta)
}

// RequestMetaFromContext 读取请求观测信息。
func RequestMetaFromContext(ctx context.Context) (*RequestMeta, bool) {
	meta, ok := ctx.Value(requestMetaContextKey{}).(*RequestMeta)
	return meta, ok
}
