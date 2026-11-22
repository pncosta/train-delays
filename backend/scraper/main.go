package main

import (
	"context"
	"flag"
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
	backfill := flag.Bool("backfill", false, "roll up every past day in trips instead of scraping")
	flag.Parse()

	env := readEnv()
	log.Printf("Starting scraper on %s...\n", env.env)
	ctx := context.Background()

	dbClient := NewDBClient(env.dbUrl, env.dbToken)
	err := dbClient.InitDB()
	if err != nil {
		log.Printf("error initing DB: %v\n", err)
		if *backfill {
			os.Exit(1)
		}
	}

	if *backfill {
		log.Printf("WARNING: BACKFILL MODE on %s - rebuilding daily rollups for all past days, no scrape\n", env.env)
		err = dbClient.BackfillRollups(now())
		if err != nil {
			log.Printf("error backfilling rollups: %v", err)
			os.Exit(1)
		}
		log.Printf("backfill finished successfully - %s\n", env.env)
		os.Exit(0)
	}

	cpClient := NewCPClient(env.cpBaseUrl, env.cpApiKey, env.cpClientID, env.cpClientSecret)

	err = getAndStoreTrips(ctx, cpClient, dbClient)
	if err != nil {
		log.Printf("scraper failed: %v", err)
		os.Exit(1)
	}

	// rollup failures must not fail the scrape
	err = dbClient.RollupYesterdayIfDue(now())
	if err != nil {
		log.Printf("error rolling up yesterday: %v", err)
	}

	log.Printf("scrape finished successfully - %s\n", env.env)
	os.Exit(0)
}

func readEnv() Env {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local-dev"
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
