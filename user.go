package main

import (
	"context"
	"fmt"
	"time"

	spotify "github.com/zmb3/spotify/v2"
)

func SearchSpotify(searchQuery string, client *spotify.Client) {
	ctx := context.Background()
	results, err := client.Search(ctx, searchQuery, spotify.SearchTypeTrack)
	handleError(err)

	fmt.Println("Tracks:")
	for _, item := range results.Tracks.Tracks {
		fmt.Println("Found:", item.Name, "by", item.Artists[0].Name, "(Album:", item.Album.Name, ")")
	}
}

func GetUserPlaylists(client *spotify.Client) {
	ctx := context.Background()
	playlists, err := client.CurrentUsersPlaylists(ctx)
	handleError(err)

	fmt.Println("Playlists:")
	for _, playlist := range playlists.Playlists {
		fmt.Println(playlist.Name)
	}
}

func GetSpotifyRecommendations(mood string, client *spotify.Client) ([]spotify.FullTrack, error) {
	// Use the personalized recommendations
	return GetPersonalizedRecommendations(mood, client)
}

func CreatePlaylistAndAddTracks(client *spotify.Client, tracks []spotify.FullTrack) error {
	if len(tracks) == 0 {
		return fmt.Errorf("no tracks provided to add to playlist")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get the current user
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}
	fmt.Printf("Creating personalized playlist for user: %s (%s)\n", user.DisplayName, user.ID)

	// Create a playlist for the user
	playlistName := fmt.Sprintf("Your Personalized Weather Mood Playlist - %s", time.Now().Format("Jan 02 15:04"))
	playlistDescription := "Playlist generated based on your music taste and current weather mood"

	playlist, err := client.CreatePlaylistForUser(
		ctx,
		user.ID,
		playlistName,
		playlistDescription,
		false,
		false,
	)
	if err != nil {
		return fmt.Errorf("failed to create playlist: %v", err)
	}
	fmt.Printf("Created personalized playlist: %s (ID: %s)\n", playlist.Name, playlist.ID)

	// Convert tracks to track IDs
	trackIDs := make([]spotify.ID, len(tracks))
	for i, track := range tracks {
		trackIDs[i] = track.ID
	}

	// Add tracks to the playlist
	fmt.Printf("Adding %d personalized tracks to playlist\n", len(trackIDs))
	_, err = client.AddTracksToPlaylist(ctx, playlist.ID, trackIDs...)
	if err != nil {
		return fmt.Errorf("failed to add tracks to playlist: %v", err)
	}

	fmt.Println("Successfully added personalized tracks to playlist!")
	return nil
}

func CreatePlaylist(client *spotify.Client) {
	fmt.Println("\n=== Creating Your Personalized Weather-Based Playlist ===")

	// Get user info for personalization
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := client.CurrentUser(ctx)
	if err == nil {
		fmt.Printf("Hello %s! Let's create a playlist tailored to your music taste.\n", user.DisplayName)
	}

	// Get weather and mood
	weather, mood := GetWeatherAndMood()

	if weather == nil || len(weather.Weather) == 0 {
		fmt.Println("Error: Weather data is incomplete")
		return
	}

	fmt.Printf("Weather: %.2f°C and %s\n", weather.Main.Temp, weather.Weather[0].Description)
	fmt.Printf("Mood selected based on weather: %s\n", mood)
	fmt.Println("Analyzing your music taste to create personalized recommendations...")

	// Get personalized recommendations
	tracks, err := GetSpotifyRecommendations(mood, client)
	if err != nil {
		fmt.Printf("Error getting recommendations: %v\n", err)
		return
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks were recommended. Try again with a different mood or city.")
		return
	}

	fmt.Println("\nRecommended tracks for your personalized playlist:")
	for i, track := range tracks {
		fmt.Printf("%d. %s by %s\n", i+1, track.Name, track.Artists[0].Name)
	}

	// Create the playlist
	err = CreatePlaylistAndAddTracks(client, tracks)
	if err != nil {
		fmt.Printf("Error creating playlist: %v\n", err)
		return
	}

	fmt.Println("\n✅ Your personalized weather-based playlist has been created successfully!")
	fmt.Println("Check your Spotify account to listen to your new playlist.")
}

func GetAvailableGenres(client *spotify.Client) ([]string, error) {
	availableGenres, err := client.GetAvailableGenreSeeds(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get available genre seeds: %v", err)
	}
	return availableGenres, nil
}
