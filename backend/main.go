package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

func main() {
	// Read Env
	dbPath := os.Getenv("DB_PATH")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local-dev"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "Hello! I am running in: %s dbpath %s", env, dbPath)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("Starting...%s\n", dbPath)

	// TODO: Read from ENV
	cpClient := NewCPClient(
		"https://api-gateway.cp.pt/cp/services/travel-api/stations",
		"ca3923e4-1d3c-424f-a3d0-9554cf3ef859",
		"1483ea620b920be6328dcf89e808937a",
		"74bd06d5a2715c64c2f848c5cdb56e6b",
	)

	err := InitDB()
	if err != nil {
		fmt.Printf("error initing DB: %v\n", err)
	}
	err = foobarGetName(ctx, *cpClient)
	if err != nil {
		fmt.Printf("error getting delays: %v\n", err)
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}

}
