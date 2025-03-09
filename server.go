package main

import (
	"fmt"
	"log"
	"net/http"

	spotify "github.com/zmb3/spotify/v2"
)

var authenticatedClient *spotify.Client

// Use a more secure state value
const stateKey = "spotify-auth-state"

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a proper state string for security
	state := stateKey
	url := auth.AuthURL(state)
	fmt.Println("Login URL:", url)
	http.Redirect(w, r, url, http.StatusFound)
}

func StartServer() {
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/callback", CallbackHandler)
	http.HandleFunc("/success", SuccessHandler)
	http.HandleFunc("/create-playlist-weather", CreatePlaylistHandlerByWeather)
	http.HandleFunc("/create-playlist-genre", CreatePlaylistHandlerByGenre)
	fmt.Println("Server started on http://localhost:8081 - Visit /login to begin")
	// Add error handling for server
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("Server error:", err)
	}
}

func CreatePlaylistHandlerByWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if authenticatedClient != nil {
		CreatePlaylistWeather(authenticatedClient)
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					font-family: 'Circular', Helvetica, Arial, sans-serif;
					background-color: #121212;
					color: white;
					text-align: center;
					padding: 40px;
					max-width: 600px;
					margin: 0 auto;
				}
				h1 {
					color: #1DB954;
					font-size: 32px;
					margin-bottom: 20px;
				}
				p {
					font-size: 18px;
					margin-bottom: 30px;
				}
				.success-icon {
					font-size: 64px;
					color: #1DB954;
					margin-bottom: 20px;
				}
				.back-link {
					color: #1DB954;
					text-decoration: none;
					font-weight: bold;
					display: inline-block;
					margin-top: 20px;
				}
				.back-link:hover {
					text-decoration: underline;
				}
			</style>
		</head>
		<body>
			<div class="success-icon">✓</div>
			<h1>Playlist Created!</h1>
			<p>Your weather-based playlist has been successfully added to your Spotify account.</p>
		</body>
		</html>
		`
		fmt.Fprint(w, html)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func CreatePlaylistHandlerByGenre(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if authenticatedClient != nil {
		CreatePlaylistGenre(authenticatedClient)
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					font-family: 'Circular', Helvetica, Arial, sans-serif;
					background-color: #121212;
					color: white;
					text-align: center;
					padding: 40px;
					max-width: 600px;
					margin: 0 auto;
				}
				h1 {
					color: #1DB954;
					font-size: 32px;
					margin-bottom: 20px;
				}
				p {
					font-size: 18px;
					margin-bottom: 30px;
				}
				.success-icon {
					font-size: 64px;
					color: #1DB954;
					margin-bottom: 20px;
				}
				.back-link {
					color: #1DB954;
					text-decoration: none;
					font-weight: bold;
					display: inline-block;
					margin-top: 20px;
				}
				.back-link:hover {
					text-decoration: underline;
				}
			</style>
		</head>
		<body>
			<div class="success-icon">✓</div>
			<h1>Playlist Created!</h1>
			<p>Your genre-based playlist has been successfully added to your Spotify account.</p>
		</body>
		</html>
		`
		fmt.Fprint(w, html)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")

	// Validate state parameter
	if state != stateKey {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Get the token from callback
	token, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token: "+err.Error(), http.StatusForbidden)
		return
	}

	// Create authenticated client
	authenticatedClient = spotify.New(auth.Client(r.Context(), token))

	// Verify client works by getting current user
	user, err := authenticatedClient.CurrentUser(r.Context())
	if err != nil {
		http.Error(w, "Failed to get user details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Logged in as %s (%s)\n", user.DisplayName, user.ID)

	// Redirect to success page
	http.Redirect(w, r, "/success", http.StatusSeeOther)
}

func SuccessHandler(w http.ResponseWriter, r *http.Request) {
	if authenticatedClient == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <style>
            body {
                font-family: 'Circular', Helvetica, Arial, sans-serif;
                background-color: #121212;
                color: white;
                text-align: center;
                padding: 40px;
                max-width: 600px;
                margin: 0 auto;
            }
            h1 {
                color: #1DB954;
                font-size: 32px;
                margin-bottom: 20px;
            }
            p {
                font-size: 18px;
                margin-bottom: 30px;
            }
            button {
                background-color: #1DB954;
                color: white;
                border: none;
                padding: 16px 32px;
                font-size: 16px;
                font-weight: bold;
                border-radius: 30px;
                cursor: pointer;
                transition: background-color 0.3s;
            }
            button:hover {
                background-color: #1ed760;
                transform: scale(1.05);
            }
			.buttons {
				display: flex;
				justify-content: center;
				gap: 2em;
			}
        </style>
    </head>
    <body>
        <h1>Successfully logged in!</h1>
        <p>Click the button below to create a weather-based playlist:</p>
		<div class="buttons">
        <form method="POST" action="/create-playlist-weather">
            <button type="submit">Create Playlist By Weather</button>
        </form>
		<form method="POST" action="/create-playlist-genre">
			<button type="submit">Create Playlist by Genre</button>
		</form>
		</div>
    </body>
    </html>
    `

	fmt.Fprint(w, html)
}
