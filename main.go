package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	cfg := &apiConfig{}

	mux.Handle(
		"/app/",
		http.StripPrefix("/app/",
			cfg.middlewareMetricsInc(
				http.FileServer(http.Dir(".")),
			),
		),
	)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /admin/metrics", cfg.getNumRequests)
	mux.HandleFunc("POST /admin/reset", cfg.resetNumRequests)

	mux.HandleFunc("POST /api/validate_chirp", cfg.validateChirp)

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()
}
