package main

import (
	"flag"
	"log"
	"os"
	"time"
)

type Env struct {
	env     string
	dbUrl   string
	dbToken string
}

func main() {
	backfill := flag.Bool("backfill", false, "roll up every past day in trips instead of only yesterday")
	flag.Parse()

	env := readEnv()
	log.Printf("Starting rollup on %s...\n", env.env)

	dbClient := NewDBClient(env.dbUrl, env.dbToken)
	err := dbClient.InitDB()
	if err != nil {
		log.Printf("error initing DB: %v\n", err)
		os.Exit(1)
	}

	if *backfill {
		log.Printf("WARNING: BACKFILL MODE on %s - rebuilding daily rollups for all past days\n", env.env)
		err = dbClient.BackfillRollups(time.Now())
		if err != nil {
			log.Printf("error backfilling rollups: %v", err)
			os.Exit(1)
		}
		log.Printf("backfill finished successfully - %s\n", env.env)
		os.Exit(0)
	}

	err = dbClient.RollupYesterdayIfDue(time.Now())
	if err != nil {
		log.Printf("error rolling up yesterday: %v", err)
		os.Exit(1)
	}

	log.Printf("rollup finished successfully - %s\n", env.env)
	os.Exit(0)
}

func readEnv() Env {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local-dev"
	}

	dbUrl := os.Getenv("TURSO_DB_URL")
	if dbUrl == "" {
		panic("missing TURSO_DB_URL")
	}

	dbToken := os.Getenv("TURSO_DB_TOKEN")
	if dbToken == "" {
		panic("missing TURSO_DB_TOKEN")
	}

	return Env{env: env, dbUrl: dbUrl, dbToken: dbToken}
}
