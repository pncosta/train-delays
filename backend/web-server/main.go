package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Env struct {
	env            string
	port           string
	cpBaseUrl      string
	cpApiKey       string
	cpClientID     string
	cpClientSecret string
	dbUrl          string
	dbToken        string
}

func main() {

	env := readEnv()
	log.Printf("Starting service...\n")

	ctx := context.Background()

	mux, err := setupHandlers(ctx, env)
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

func setupHandlers(ctx context.Context, env Env) (*http.ServeMux, error) {
	h := &Handler{}
	mux := &http.ServeMux{}
	mux.HandleFunc("GET /api/stats/summary", h.handleSummary(ctx, ""))
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

	return Env{
		env:  env,
		port: port,
	}

}
