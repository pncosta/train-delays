package main

import (
	"context"
	"log"
	"os"
)

type Env struct {
	env            string
	cpBaseUrl      string
	cpApiKey       string
	cpClientID     string
	cpClientSecret string
	dbUrl          string
	dbToken        string
}

func main() {
	log.Println("Starting scraper...")
	env := readEnv()
	ctx := context.Background()

	dbClient := NewDBClient(env.dbUrl, env.dbToken)
	err := dbClient.InitDB()
	if err != nil {
		log.Printf("error initing DB: %v\n", err)
	}

	cpClient := NewCPClient(env.cpBaseUrl, env.cpApiKey, env.cpClientID, env.cpClientSecret)

	err = getAndStoreTrips(ctx, cpClient, dbClient)
	if err != nil {
		log.Printf("scraper failed: %v", err)
		os.Exit(1)
	}

	log.Println("scrape finished successfully")
	os.Exit(0)
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

	cpBaseUrl := "https://api-gateway.cp.pt"

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

	dbUrl := os.Getenv("TURSO_DB_URL")
	if dbUrl == "" {
		panic("missing TURSO_DB_URL")
	}

	dbToken := os.Getenv("TURSO_DB_TOKEN")
	if dbToken == "" {
		panic("missing TURSO_DB_TOKEN")
	}

	return Env{
		env:            env,
		dbUrl:          dbUrl,
		dbToken:        dbToken,
		cpBaseUrl:      cpBaseUrl,
		cpApiKey:       cpApiKey,
		cpClientID:     cpClientID,
		cpClientSecret: cpClientSecret,
	}
}
