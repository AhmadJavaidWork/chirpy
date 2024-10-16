package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ahmadjavaidwork/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	apiCfg := apiConfig{}

	err := setUpCfg(&apiCfg)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	mux.Handle("/app/", fsHandler)

	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerWebhook)

	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	mux.HandleFunc("POST /api/users", apiCfg.handlerUsersCreate)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUsersUpdate)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirpsCreate)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsRetrieve)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerChirpsGet)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerChirpsDelete)

	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}

func setUpCfg(apiCfg *apiConfig) error {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return errors.New("DB_URL must be set")
	}
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("error opening database: %s", err)
	}
	dbQueries := database.New(dbConn)

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		return errors.New("PLATFORM must be set")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("JWT_SECRET environment variable is not set")
	}
	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		return errors.New("POLKA_KEY environment variable is not set")
	}
	*apiCfg = apiConfig{
		db:        dbQueries,
		platform:  platform,
		jwtSecret: jwtSecret,
		polkaKey:  polkaKey,
	}
	return nil
}
