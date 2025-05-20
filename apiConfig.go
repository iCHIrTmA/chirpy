package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) getNumRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load()) // Write HTML directly
}

func (cfg *apiConfig) resetNumRequests(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully reset number of hits"))
}

func (cfg *apiConfig) validateChirp(w http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type response struct {
		// the key will be the name of struct field unless you give it an explicit JSON tag
		Error       string `json:"error,omitempty"`
		CleanedBody string `json:"cleaned_body,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(req.Body)
	chirpVal := chirp{}

	err := decoder.Decode(&chirpVal)

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(500)
		w.Write(errResponse)
		return
	}

	if chirpVal.Body == "" {
		errResponse, _ := json.Marshal(response{
			Error: "Chirp json request body is missing",
		})
		w.WriteHeader(400)
		w.Write(errResponse)
		return
	}

	if len(chirpVal.Body) > 140 {
		errResponse, _ := json.Marshal(response{
			Error: "Chirp is too long",
		})
		w.WriteHeader(400)
		w.Write(errResponse)
		return
	}

	// clean profanity
	words := strings.Split(chirpVal.Body, " ")
	profanities := []string{"kerfuffle", "sharbert", "fornax"}
	for i, word := range words {
		if slices.Contains(profanities, strings.ToLower(word)) {
			words[i] = "****"
		}
	}

	successResponse, _ := json.Marshal(response{
		CleanedBody: strings.Join(words, " "),
	})
	w.WriteHeader(200)
	w.Write(successResponse)
}
