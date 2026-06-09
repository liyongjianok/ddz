package app

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ddz/backend/internal/auth"
	"ddz/backend/internal/telemetry"
)

type readySnapshot struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

type metricsSnapshot struct {
	UptimeSeconds     int64            `json:"uptime_seconds"`
	HTTPRequestsTotal uint64           `json:"http_requests_total"`
	WSConnections     int64            `json:"ws_connections"`
	HTTPLatencyByPath map[string]int64 `json:"http_latency_ms_by_path"`
}

type metricsCollector struct {
	startedAt      time.Time
	httpRequests   atomic.Uint64
	wsConnections  atomic.Int64
	latencyMu      sync.Mutex
	latencyByRoute map[string]int64
}

func newMetricsCollector(now func() time.Time) *metricsCollector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &metricsCollector{
		startedAt:      now(),
		latencyByRoute: make(map[string]int64),
	}
}

func (m *metricsCollector) observeHTTPRequest(path string, duration time.Duration) {
	m.httpRequests.Add(1)
	m.latencyMu.Lock()
	m.latencyByRoute[path] = duration.Milliseconds()
	m.latencyMu.Unlock()
}

func (m *metricsCollector) AddWSConnection(delta int64) {
	m.wsConnections.Add(delta)
}

func (m *metricsCollector) snapshot(now func() time.Time) metricsSnapshot {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	m.latencyMu.Lock()
	latencyByPath := make(map[string]int64, len(m.latencyByRoute))
	for key, value := range m.latencyByRoute {
		latencyByPath[key] = value
	}
	m.latencyMu.Unlock()

	return metricsSnapshot{
		UptimeSeconds:     int64(now().Sub(m.startedAt).Seconds()),
		HTTPRequestsTotal: m.httpRequests.Load(),
		WSConnections:     m.wsConnections.Load(),
		HTTPLatencyByPath: latencyByPath,
	}
}

func newLogger(cfg Config) *slog.Logger {
	level := parseLogLevel(cfg.LogLevel)
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) ReadFrom(src io.Reader) (int64, error) {
	if readerFrom, ok := r.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(src)
	}
	return io.Copy(r.ResponseWriter, src)
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func withObservability(next http.Handler, logger *slog.Logger, metrics *metricsCollector) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = newMetricsCollector(nil)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := requestIDFromRequest(r)
		if traceID == "" {
			traceID = telemetry.NewTraceID()
		}

		meta := &telemetry.RequestMeta{
			TraceID: traceID,
			RoomID:  roomIDFromPath(r.URL.Path),
			GameID:  gameIDFromPath(r.URL.Path),
		}
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			meta.UserID = claims.Subject
		}

		ctx := telemetry.WithRequestMeta(r.Context(), meta)
		r = r.WithContext(ctx)
		r.Header.Set("X-Request-ID", traceID)
		w.Header().Set("X-Request-ID", traceID)

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		startedAt := time.Now()
		next.ServeHTTP(recorder, r)
		duration := time.Since(startedAt)

		metrics.observeHTTPRequest(normalizeMetricsPath(r.URL.Path), duration)
		logger.Info("http_request",
			"trace_id", meta.TraceID,
			"user_id", meta.UserID,
			"room_id", meta.RoomID,
			"game_id", meta.GameID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

func writeMetrics(w http.ResponseWriter, metrics *metricsCollector) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(metrics.snapshot(nil))
}

func writeReadyz(w http.ResponseWriter, checks map[string]string) {
	status := "ok"
	for _, value := range checks {
		if value != "ok" {
			status = "degraded"
			break
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(readySnapshot{
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	})
}

func normalizeMetricsPath(path string) string {
	if gameID, ok := parseRecordGameID(path); ok && gameID != "" {
		return "/api/v1/records/:game_id"
	}
	if roomID, action, ok := parseRoomActionPath(path); ok && roomID != "" && action != "" {
		return "/api/v1/rooms/:room_id/" + action
	}
	if strings.HasPrefix(path, "/ws/v1/rooms/") {
		return "/ws/v1/rooms/:room_id"
	}
	return path
}

func roomIDFromPath(path string) string {
	if roomID, _, ok := parseRoomActionPath(path); ok {
		return roomID
	}
	if strings.HasPrefix(path, "/ws/v1/rooms/") {
		return strings.Trim(strings.TrimPrefix(path, "/ws/v1/rooms/"), "/")
	}
	return ""
}

func gameIDFromPath(path string) string {
	gameID, _ := parseRecordGameID(path)
	return gameID
}
