package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Env struct {
	env     string
	port    string
	dbUrl   string
	dbToken string
}

func main() {

	env := readEnv()
	log.Printf("Starting service...\n")

	ctx := context.Background()

	dbClient := NewDBClient(env.dbUrl, env.dbToken)
	err := dbClient.InitDB()
	if err != nil {
		log.Printf("error connecting to DB %v\n", err)
	}

	mux, err := setupHandlers(ctx, env, dbClient)
	if err != nil {
		log.Panic(err)
	}

	err = http.ListenAndServe(":"+env.port, withCORS(mux))
	if err != nil {
		fmt.Println(err)
	}

}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setupHandlers(ctx context.Context, env Env, dbClient *DBClient) (*http.ServeMux, error) {
	h := &Handler{}
	mux := &http.ServeMux{}
	mux.HandleFunc("GET /api/stats/summary", h.handleSummary(ctx, "", dbClient))
	mux.HandleFunc("GET /api/stats/worst", h.handleWorstDelays(ctx, "", dbClient))
	mux.HandleFunc("GET /api/stats/cancellations", h.handleCancellations(ctx, "", dbClient))
	mux.HandleFunc("GET /api/stats/worst-average", h.handleWorstAverageDelays(ctx, "", dbClient))
	return mux, nil
}

func readEnv() Env {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local-dev"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbUrl := os.Getenv("TURSO_DB_URL")
	if dbUrl == "" {
		panic("missing TURSO_DB_URL")
	}

	dbToken := os.Getenv("TURSO_DB_TOKEN")
	if dbToken == "" {
		panic("missing TURSO_DB_TOKEN")
	}

	return Env{
		env:     env,
		port:    port,
		dbUrl:   dbUrl,
		dbToken: dbToken,
	}

}
