package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type Env struct {
	dbPath         string
	env            string
	port           string
	cpApiKey       string
	cpClientID     string
	cpClientSecret string
}

func main() {
	// Read Env
	env := readEnv()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "Hello! I am running in: %s dbpath %s", env.env, env.dbPath)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("Starting...%s\n", env.dbPath)

	cpClient := NewCPClient(
		"https://api-gateway.cp.pt/cp/services/travel-api/stations",
		env.cpApiKey, env.cpClientID, env.cpClientSecret,
	)

	err := InitDB(env.dbPath)
	if err != nil {
		fmt.Printf("error initing DB: %v\n", err)
	}

	date := "2026-03-24"
	err = getAndStoreTrips(ctx, *cpClient, date)
	if err != nil {
		fmt.Printf("error getting delays: %v\n", err)
	}

	fmt.Printf("Server starting on port %s...\n", env.port)
	if err := http.ListenAndServe(":"+env.port, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}

}

func readEnv() Env {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local-dev"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/trips.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cpApiKey := os.Getenv("CP_API_KEY")
	if cpApiKey == "" {
		panic("missing CP_API_KEY")
	}

	cpClientID := os.Getenv("CP_CLIENT_ID")
	if cpClientID == "" {
		panic("missing CP_CLIENT_ID")
	}

	cpClientSecret := os.Getenv("CP_CLIENT_SECRET")
	if cpClientSecret == "" {
		panic("missing CP_CLIENT_SECRET")
	}

	return Env{
		dbPath:         dbPath,
		env:            env,
		port:           port,
		cpApiKey:       cpApiKey,
		cpClientID:     cpClientID,
		cpClientSecret: cpClientSecret,
	}

}
