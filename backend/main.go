package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// Check for an environment variable
		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "local-dev"
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "Hello Delays! I am running in: %s", env)
	})

	// Cloud Run provides the PORT variable automatically
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
