package main

import (
	"log"
	"os"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var auth *spotifyauth.Authenticator

func main() {
	// Load environment variables from build-time values
	envVars := LoadEnvVars()

	// Set environment variables for the application
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	auth = Auth()
	StartServer()
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
