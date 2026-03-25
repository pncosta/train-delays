package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
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
	dbPath         string
}

func main() {
	// Read Env
	env := readEnv()
	log.Printf("Starting service...%s\n", env.dbPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbClient := newDBClient(env.dbUrl, env.dbToken)
	_, err := dbClient.Connect()
	if err != nil {
		log.Printf("error connecting to DB: %v\n", err)
	}
	err = dbClient.InitDB()
	if err != nil {
		log.Printf("error initing DB: %v\n", err)
	}

	cpClient := NewCPClient(env.cpBaseUrl, env.cpApiKey, env.cpClientID, env.cpClientSecret)

	triggerScrapper(ctx, cpClient, dbClient)

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "Hello! I am running in: %s dbpath %s", env.env, env.dbPath)
	})

	fmt.Printf("Server starting on port %s...\n", env.port)
	if err := http.ListenAndServe(":"+env.port, nil); err != nil {
		log.Printf("Error starting server: %s\n", err)
	}
}

// triggerScrapper initializes and runs periodic tasks
func triggerScrapper(ctx context.Context, cpClient *CPClient, dbClient *DBClient) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		log.Println("Background worker started...")
		err := getAndStoreTrips(ctx, cpClient, dbClient)
		if err != nil {
			fmt.Printf("error getting delays: %v\n", err)
		}
		for {
			select {
			case <-ticker.C:
				err := getAndStoreTrips(ctx, cpClient, dbClient)
				if err != nil {
					log.Printf("error getting delays: %v\n", err)
				}
			}
		}
	}()
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
		dbPath:         dbPath, // old db delete
		dbUrl:          dbUrl,
		dbToken:        dbToken,
		env:            env,
		port:           port,
		cpBaseUrl:      cpBaseUrl,
		cpApiKey:       cpApiKey,
		cpClientID:     cpClientID,
		cpClientSecret: cpClientSecret,
	}

}
