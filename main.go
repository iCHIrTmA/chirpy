package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	mux := http.NewServeMux()
	cfg := &apiConfig{}
	cfg.authSecret = os.Getenv("SECRET_AUTH_KEY")
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	cfg.db = database.New(db)

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
	// mux.HandleFunc("POST /admin/reset", cfg.resetNumRequests)
	mux.HandleFunc("POST /admin/reset", cfg.resetUsers)

	// mux.HandleFunc("POST /api/validate_chirp", cfg.validateChirp)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	mux.HandleFunc("POST /api/login", cfg.loginUser)
	mux.HandleFunc("POST /api/refresh", cfg.refreshAccessToken)
	mux.HandleFunc("POST /api/revoke", cfg.revokeAccessToken)

	mux.HandleFunc("POST /api/chirps", cfg.createChirp)
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirp)

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()
}
