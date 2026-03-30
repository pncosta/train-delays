package main

import (
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

	// Read Env
	env := readEnv()
	log.Printf("Starting service...\n")

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "Hello! I am running : %s", env.env)
	})

	fmt.Printf("Server starting on port %s...\n", env.port)
	if err := http.ListenAndServe(":"+env.port, nil); err != nil {
		log.Printf("Error starting server: %s\n", err)
	}
}

// triggerScrapper initializes and runs periodic tasks

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
