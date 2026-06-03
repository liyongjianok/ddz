package app

import (
	"net/http"
)

func NewHTTPServer(cfg Config) *http.Server {
	return &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewHTTPHandler(),
	}
}

func NewHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
