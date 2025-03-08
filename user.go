package main

import (
	"context"
	"fmt"
	"time"

	spotify "github.com/zmb3/spotify/v2"
)

func SearchSpotify(searchQuery string, client *spotify.Client) {
	results, err := client.Search(context.TODO(), searchQuery, spotify.SearchTypeTrack)
	handleError(err)

	if len(results.Tracks.Tracks) > 0 {
		track := results.Tracks.Tracks[0]
		fmt.Printf("Found: %s by %s (Album: %s)\n", track.Name, track.Artists[0].Name, track.Album.Name)
	} else {
		fmt.Println("No tracks found")
	}
}

func GetUserPlaylists(client *spotify.Client) {
	playlists, err := client.CurrentUsersPlaylists(context.TODO())
	handleError(err)

	for _, playlist := range playlists.Playlists {
		fmt.Printf("Playlist: %s\n", playlist.Name)
	}
}

func GetSpotifyRecommendations(mood string, client *spotify.Client) ([]spotify.FullTrack, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Define search queries based on mood
	var searchQuery string
	switch mood {
	case "energetic":
		searchQuery = "pop dance"
	case "relaxed":
		searchQuery = "chill acoustic"
	case "intense":
		searchQuery = "rock metal"
	case "thoughtful":
		searchQuery = "indie ambient"
	default:
		searchQuery = "pop"
	}

	fmt.Printf("Searching for tracks with query: %s\n", searchQuery)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Search for tracks instead of using recommendations
	results, err := client.Search(
		ctx,
		searchQuery,
		spotify.SearchTypeTrack,
		spotify.Limit(20),
	)

	if err != nil {
		fmt.Printf("Error searching for tracks: %v\n", err)
		return nil, err
	}

	if results == nil || results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found for mood: %s", mood)
	}

	fmt.Printf("Found %d tracks\n", len(results.Tracks.Tracks))
	return results.Tracks.Tracks, nil
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
	fmt.Printf("Creating playlist for user: %s (%s)\n", user.DisplayName, user.ID)

	// Create a playlist for the user
	playlistName := fmt.Sprintf("Weather Mood Playlist - %s", time.Now().Format("Jan 02 15:04"))
	playlist, err := client.CreatePlaylistForUser(
		ctx,
		user.ID,
		playlistName,
		"Playlist generated based on your current weather mood",
		false,
		false,
	)
	if err != nil {
		return fmt.Errorf("failed to create playlist: %v", err)
	}
	fmt.Printf("Created playlist: %s (ID: %s)\n", playlist.Name, playlist.ID)

	// Convert tracks to track IDs
	trackIDs := make([]spotify.ID, len(tracks))
	for i, track := range tracks {
		trackIDs[i] = track.ID
	}

	// Add tracks to the playlist
	fmt.Printf("Adding %d tracks to playlist\n", len(trackIDs))
	_, err = client.AddTracksToPlaylist(ctx, playlist.ID, trackIDs...)
	if err != nil {
		return fmt.Errorf("failed to add tracks to playlist: %v", err)
	}

	fmt.Println("Successfully added tracks to playlist!")
	return nil
}

func CreatePlaylist(client *spotify.Client) {
	weather, mood := GetWeatherAndMood()

	if weather == nil || len(weather.Weather) == 0 {
		fmt.Println("Error: Weather data is incomplete")
		return
	}

	fmt.Printf("Weather: %.2f°C and %s\n", weather.Main.Temp, weather.Weather[0].Description)
	fmt.Printf("Mood selected: %s\n", mood)

	tracks, err := GetSpotifyRecommendations(mood, client)
	if err != nil {
		fmt.Printf("Error getting recommendations: %v\n", err)
		return
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks were recommended. Try again with a different mood or city.")
		return
	}

	fmt.Println("Recommended tracks:")
	for i, track := range tracks {
		fmt.Printf("%d. %s by %s\n", i+1, track.Name, track.Artists[0].Name)
	}

	err = CreatePlaylistAndAddTracks(client, tracks)
	if err != nil {
		fmt.Printf("Error creating playlist: %v\n", err)
		return
	}

	fmt.Println("Playlist created successfully!")
}

func GetAvailableGenres(client *spotify.Client) ([]string, error) {
	availableGenres, err := client.GetAvailableGenreSeeds(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get available genre seeds: %v", err)
	}
	return availableGenres, nil
}
