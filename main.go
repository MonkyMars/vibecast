package main

import (
	"log"

	"github.com/joho/godotenv"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var auth *spotifyauth.Authenticator

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	auth = Auth()
	StartServer()
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
