package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Env struct {
	env     string
	dbUrl   string
	dbToken string
}

func main() {
	backfill := flag.Bool("backfill", false, "roll up every past day in trips instead of only yesterday")
	flag.Parse()

	if err := run(*backfill); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

// run exists so the DB is closed before main's os.Exit.
func run(backfill bool) error {
	env := readEnv()
	log.Printf("Starting rollup on %s...\n", env.env)

	ctx := context.Background()
	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", env.dbUrl, env.dbToken))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := initSchema(ctx, db); err != nil {
		return fmt.Errorf("error initing DB: %w", err)
	}

	nowLisbon, err := lisbonNow(time.Now())
	if err != nil {
		return err
	}

	if backfill {
		log.Printf("WARNING: BACKFILL MODE on %s - rebuilding daily rollups for all past days\n", env.env)
		if err := backfillRollups(ctx, db, nowLisbon); err != nil {
			return fmt.Errorf("error backfilling rollups: %w", err)
		}
		log.Printf("backfill finished successfully - %s\n", env.env)
		return nil
	}

	if err := rollupYesterdayIfDue(ctx, db, nowLisbon); err != nil {
		return fmt.Errorf("error rolling up yesterday: %w", err)
	}
	log.Printf("rollup finished successfully - %s\n", env.env)
	return nil
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
