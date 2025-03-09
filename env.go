package main

// These values will be overridden at build time
var (
	spotifyClientID     = "cfaf06e7acc241cf893bdd897666bb4e"
	spotifyClientSecret = "62bd8e12d0fe45abbd19b39e3d9f6e4c"
	weatherAPIKey       = "149948df2606cea6e7c783fa3a9b6f7e"
)

// LoadEnvVars loads environment variables from build flags or returns defaults
func LoadEnvVars() map[string]string {
	// Create a map of environment variables
	envVars := map[string]string{
		"SPOTIFY_CLIENT_ID":     spotifyClientID,
		"SPOTIFY_CLIENT_SECRET": spotifyClientSecret,
		"WEATHER_API_KEY":       weatherAPIKey,
	}

	return envVars
}
