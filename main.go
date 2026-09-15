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

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/thelol3882/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
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

func (cfg *apiConfig) habdlerReset(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("cannot load env")
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("cannot connect to database")
		os.Exit(1)
	}

	dbQueries := database.New(db)
	apiCfg := apiConfig{
		db: dbQueries,
	}
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.habdlerReset)
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}

		badWords := [3]string{"kerfuffle", "sharbert", "fornax"}

		decoder := json.NewDecoder(r.Body)
		var params parameters
		if err := decoder.Decode(&params); err != nil {
			type errResponse struct {
				Error string `json:"error"`
			}
			errR := errResponse{
				Error: "Something went wrong",
			}
			data, _ := json.Marshal(errR)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(data)
			return
		}

		if len(params.Body) > 140 {
			type errResponse struct {
				Error string `json:"error"`
			}
			errR := errResponse{
				Error: "Body message is long",
			}
			data, _ := json.Marshal(errR)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(data)
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
		validR := validResponse{
			CleanedBody: result,
		}
		data, _ := json.Marshal(validR)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	s := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(s.ListenAndServe())
}
