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

	err = http.ListenAndServe(":"+env.port, mux)
	if err != nil {
		fmt.Println(err)
	}

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
