package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
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

// func (cfg *apiConfig) resetNumRequests(w http.ResponseWriter, req *http.Request) {
// 	cfg.fileserverHits.Store(0)

// 	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Successfully reset number of hits"))
// }

func (cfg *apiConfig) resetUsers(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Error string `json:"error,omitempty"`
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if os.Getenv("PLATFORM") != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.db.DeleteAllUsers(req.Context())

	if err != nil {
		fmt.Println(err)
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusNotImplemented)
		w.Write(errResponse)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully removed all records in users table"))
}

// func (cfg *apiConfig) validateChirp(w http.ResponseWriter, req *http.Request) {
// 	type chirp struct {
// 		Body string `json:"body"`
// 	}

// 	type response struct {
// 		// the key will be the name of struct field unless you give it an explicit JSON tag
// 		Error       string `json:"error,omitempty"`
// 		CleanedBody string `json:"cleaned_body,omitempty"`
// 	}

// 	w.Header().Set("Content-Type", "application/json")

// 	decoder := json.NewDecoder(req.Body)
// 	chirpVal := chirp{}

// 	err := decoder.Decode(&chirpVal)

// 	if err != nil {
// 		errResponse, _ := json.Marshal(response{
// 			Error: "Something went wrong",
// 		})
// 		w.WriteHeader(500)
// 		w.Write(errResponse)
// 		return
// 	}

// 	if chirpVal.Body == "" {
// 		errResponse, _ := json.Marshal(response{
// 			Error: "Chirp json request body is missing",
// 		})
// 		w.WriteHeader(400)
// 		w.Write(errResponse)
// 		return
// 	}

// 	if len(chirpVal.Body) > 140 {
// 		errResponse, _ := json.Marshal(response{
// 			Error: "Chirp is too long",
// 		})
// 		w.WriteHeader(400)
// 		w.Write(errResponse)
// 		return
// 	}

// 	// clean profanity
// 	words := strings.Split(chirpVal.Body, " ")
// 	profanities := []string{"kerfuffle", "sharbert", "fornax"}
// 	for i, word := range words {
// 		if slices.Contains(profanities, strings.ToLower(word)) {
// 			words[i] = "****"
// 		}
// 	}

// 	successResponse, _ := json.Marshal(response{
// 		CleanedBody: strings.Join(words, ","),
// 	})
// 	w.WriteHeader(200)
// 	w.Write(successResponse)
// }

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Error string `json:"error,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	dbChirps, err := cfg.db.ListChirps(req.Context())

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	type Chirp struct {
		ID        string `json:"id"`
		Body      string `json:"body"`
		UserID    string `json:"user_id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirp := Chirp{
			ID:        dbChirp.ID.String(),
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID.String(),
			CreatedAt: dbChirp.CreatedAt.String(),
			UpdatedAt: dbChirp.UpdatedAt.String(),
		}
		chirps = append(chirps, chirp)
	}

	successResponse, _ := json.Marshal(chirps)
	w.WriteHeader(http.StatusOK)
	w.Write(successResponse)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	type requestData struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}

	type response struct {
		Error     string `json:"error,omitempty"`
		ID        string `json:"id,omitempty"`
		Body      string `json:"body,omitempty"`
		UserID    string `json:"user_id,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(req.Body)
	chirpData := requestData{}

	err := decoder.Decode(&chirpData)

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(500)
		w.Write(errResponse)
		return
	}

	if chirpData.Body == "" {
		errResponse, _ := json.Marshal(response{
			Error: "Chirp json request body is missing",
		})
		w.WriteHeader(400)
		w.Write(errResponse)
		return
	}

	if len(chirpData.Body) > 140 {
		errResponse, _ := json.Marshal(response{
			Error: "Chirp is too long",
		})
		w.WriteHeader(400)
		w.Write(errResponse)
		return
	}

	// clean profanity
	words := strings.Split(chirpData.Body, " ")
	profanities := []string{"kerfuffle", "sharbert", "fornax"}
	for i, word := range words {
		if slices.Contains(profanities, strings.ToLower(word)) {
			words[i] = "****"
		}
	}

	chirpDataUserId, err := uuid.Parse(chirpData.UserID)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(500)
		w.Write(errResponse)
		return
	}

	chirp, err := cfg.db.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   strings.Join(words, " "),
			UserID: chirpDataUserId,
		})

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	successResponse, _ := json.Marshal(response{
		ID:        chirp.ID.String(),
		Body:      chirp.Body,
		UserID:    chirp.UserID.String(),
		CreatedAt: chirp.CreatedAt.String(),
		UpdatedAt: chirp.UpdatedAt.String(),
	})
	w.WriteHeader(http.StatusCreated)
	w.Write(successResponse)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	type requestData struct {
		Email string `json:"email"`
	}

	type response struct {
		Error     string `json:"error,omitempty"`
		ID        string `json:"id,omitempty"`
		Email     string `json:"email,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(req.Body)
	userData := requestData{}

	err := decoder.Decode(&userData)

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	user, err := cfg.db.CreateUser(req.Context(), userData.Email)

	if err != nil {
		// fmt.Errorf("error: %w", err)
		fmt.Println("hello", err)
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	// printF(user)
	// fmt.Printf("%+v\n", user)

	successResponse, _ := json.Marshal(response{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.String(),
		UpdatedAt: user.UpdatedAt.String(),
	})
	w.WriteHeader(http.StatusCreated)
	w.Write(successResponse)
}
