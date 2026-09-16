package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/thelol3882/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, _ *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hits)
	fmt.Fprint(w, html)
}

// func (cfg *apiConfig) habdlerReset(w http.ResponseWriter, _ *http.Request) {
// 	cfg.fileserverHits.Store(0)
// 	w.WriteHeader(http.StatusOK)
// }

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errResponse{Error: msg})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("cannot load env")
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	platformMode := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("cannot connect to database")
		os.Exit(1)
	}

	dbQueries := database.New(db)
	apiCfg := apiConfig{
		db:       dbQueries,
		platform: platformMode,
	}
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	// mux.HandleFunc("POST /admin/reset", apiCfg.habdlerReset)
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}

		badWords := [3]string{"kerfuffle", "sharbert", "fornax"}

		decoder := json.NewDecoder(r.Body)
		var params parameters
		if err := decoder.Decode(&params); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		if len(params.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Body message is long")
			return
		}

		body := strings.Split(params.Body, " ")

		for i, word := range body {
			for _, bardWord := range badWords {
				if strings.ToLower(word) == bardWord {
					body[i] = "****"
				}
			}
		}

		result := strings.Join(body, " ")

		type validResponse struct {
			CleanedBody string `json:"cleaned_body"`
		}
		respondWithJSON(w, http.StatusOK, validResponse{CleanedBody: result})
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email string `json:"email"`
		}

		decoder := json.NewDecoder(r.Body)
		var params parameters
		if err := decoder.Decode(&params); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		if params.Email == "" {
			respondWithError(w, http.StatusBadRequest, "Use correct email")
			return
		}

		user, err := apiCfg.db.CreateUser(r.Context(), params.Email)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong or email already used")
			return
		}

		type User struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}

		respondWithJSON(w, http.StatusCreated, User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		})
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if apiCfg.platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := apiCfg.db.ResetUsers(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	s := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(s.ListenAndServe())
}
