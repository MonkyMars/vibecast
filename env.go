package main

// These values will be overridden at build time
var (
	spotifyClientID     = "default"
	spotifyClientSecret = "default"
	weatherAPIKey       = "default"
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
