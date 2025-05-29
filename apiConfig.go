package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chirpy/internal/auth"
	"github.com/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	authSecret     string
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

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, req *http.Request) {
	errorResponse := struct {
		Error string `json:"error,omitempty"`
	}{}

	w.Header().Set("Content-Type", "application/json")

	// check access token
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userUUID, err := auth.ValidateJWT(bearerToken, cfg.authSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chirpUUID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		errorResponse.Error = "Something went wrong"
		errResponse, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), chirpUUID)
	if err != nil {
		errorResponse.Error = "Chirp does not exist"
		errResponse, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusNotFound)
		w.Write(errResponse)
		return
	}

	if chirp.UserID != userUUID {
		errorResponse.Error = "Chirp does not belong to user"
		errResponse, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusForbidden)
		w.Write(errResponse)
		return
	}

	err = cfg.db.DeleteChirp(req.Context(), chirp.ID)
	if err != nil {
		errorResponse.Error = "Something went wrong"
		errResponse, _ := json.Marshal(errorResponse)
		w.WriteHeader(http.StatusNotFound)
		w.Write(errResponse)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	bearerToken, err := auth.GetBearerToken(req.Header)
	// println("bearerToken", bearerToken)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "No authorization header",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	userUUID, err := auth.ValidateJWT(bearerToken, cfg.authSecret)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "No authorization header",
		})
		w.WriteHeader(http.StatusUnauthorized)
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

	chirp, err := cfg.db.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   strings.Join(words, " "),
			UserID: userUUID,
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

func (cfg *apiConfig) getChirp(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Error     string `json:"error,omitempty"`
		ID        string `json:"id,omitempty"`
		Body      string `json:"body,omitempty"`
		UserID    string `json:"user_id,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	uuid, err := uuid.Parse(req.PathValue("chirpID"))

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), uuid)

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusNotFound)
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
	w.WriteHeader(http.StatusOK)
	w.Write(successResponse)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	type requestData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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

	hashedPassword, err := auth.HashPassword(userData.Password)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	user, err := cfg.db.CreateUser(req.Context(), database.CreateUserParams{Email: userData.Email, HashedPassword: hashedPassword})
	if err != nil {
		// fmt.Errorf("error: %w", err)
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

func (cfg *apiConfig) updateUser(w http.ResponseWriter, req *http.Request) {
	type requestData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		Error     string `json:"error,omitempty"`
		ID        string `json:"id,omitempty"`
		Email     string `json:"email,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	// check access token
	bearerToken, err := auth.GetBearerToken(req.Header)
	// println("bearerToken", bearerToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userUUID, err := auth.ValidateJWT(bearerToken, cfg.authSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(req.Body)
	userData := requestData{}

	err = decoder.Decode(&userData)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	hashedPassword, err := auth.HashPassword(userData.Password)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	user, err := cfg.db.UpdateUser(req.Context(), database.UpdateUserParams{ID: userUUID, Email: userData.Email, HashedPassword: hashedPassword})
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	successResponse, _ := json.Marshal(response{
		ID:        user.ID.String(),
		Email:     user.Email,
		UpdatedAt: user.UpdatedAt.String(),
	})
	w.WriteHeader(http.StatusOK)
	w.Write(successResponse)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, req *http.Request) {
	type requestData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		Error        string `json:"error,omitempty"`
		ID           string `json:"id,omitempty"`
		Email        string `json:"email,omitempty"`
		Token        string `json:"token,omitempty"`
		RefreshToken string `json:"refresh_token,omitempty"`
		CreatedAt    string `json:"created_at,omitempty"`
		UpdatedAt    string `json:"updated_at,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(req.Body)
	reqData := requestData{}

	err := decoder.Decode(&reqData)

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	// fetch user by email
	user, err := cfg.db.GetUserByEmail(req.Context(), reqData.Email)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Incorrect email or password",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	// printF(user)
	// fmt.Printf("%+v\n", user)
	fmt.Printf("%+v\n", reqData.Password)
	fmt.Printf("%+v\n", user.HashedPassword)

	err = auth.CheckPasswordHash(user.HashedPassword, reqData.Password)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Incorrect email or password",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	expiry, _ := time.ParseDuration("3600s")
	token, err := auth.MakeJWT(user.ID, cfg.authSecret, expiry)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errResponse)
		return
	}

	refreshTokenString, _ := auth.MakeRefreshToken()

	refresh, err := cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{Token: refreshTokenString, UserID: user.ID})
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errResponse)
		return
	}

	successResponse, _ := json.Marshal(response{
		ID:           user.ID.String(),
		Email:        user.Email,
		Token:        token,
		RefreshToken: refresh.Token,
		CreatedAt:    user.CreatedAt.String(),
		UpdatedAt:    user.UpdatedAt.String(),
	})
	w.WriteHeader(http.StatusOK)
	w.Write(successResponse)
}

func (cfg *apiConfig) refreshAccessToken(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Error string `json:"error,omitempty"`
		Token string `json:"token,omitempty"`
	}

	bearerRefreshToken, err := auth.GetBearerToken(req.Header)
	println("refreshAccessToken() bearerRefreshToken", bearerRefreshToken)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "No authorization header",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	refreshTokenDB, err := cfg.db.GetUserFromRefreshToken(req.Context(), bearerRefreshToken)
	println("refreshAccessToken() refreshTokenDB", refreshTokenDB.UserID.String(), refreshTokenDB.ExpiresAt.String())

	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Invalid token",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}
	if time.Now().After(refreshTokenDB.ExpiresAt) {
		println("refreshAccessToken() refresh token expired", time.Now().String())

		errResponse, _ := json.Marshal(response{
			Error: "Invalid token",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}
	if refreshTokenDB.RevokedAt.Valid {
		println("refreshAccessToken() refresh token revoked", time.Now().String())

		errResponse, _ := json.Marshal(response{
			Error: "Invalid token",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	// println("refreshAccessToken() no errors", refreshTokenDB.ExpiresAt.String(), refreshTokenDB.UserID.String())
	// return

	expiry, _ := time.ParseDuration("3600s")
	accessToken, err := auth.MakeJWT(refreshTokenDB.UserID, cfg.authSecret, expiry)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Something went wrong",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errResponse)
		return
	}

	successResponse, _ := json.Marshal(response{
		Token: accessToken,
	})
	w.WriteHeader(http.StatusOK)
	w.Write(successResponse)
}

func (cfg *apiConfig) revokeAccessToken(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Error   string `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
	}

	bearerRefreshToken, err := auth.GetBearerToken(req.Header)
	// println("bearerRefreshToken", bearerRefreshToken)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "No authorization header",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	err = cfg.db.RevokeRefreshToken(req.Context(), bearerRefreshToken)
	if err != nil {
		errResponse, _ := json.Marshal(response{
			Error: "Invalid token",
		})
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errResponse)
		return
	}

	successResponse, _ := json.Marshal(response{
		Message: "Access token revoked!",
	})
	w.WriteHeader(http.StatusNoContent)
	w.Write(successResponse)
}
