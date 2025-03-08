package main

import (
	"context"
	"fmt"
	"time"

	spotify "github.com/zmb3/spotify/v2"
)

// min returns the smaller of x or y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// GetUserTopArtists retrieves the user's top artists from Spotify
func GetUserTopArtists(client *spotify.Client) ([]spotify.FullArtist, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user's top artists
	topArtists, err := client.CurrentUsersTopArtists(
		ctx,
		spotify.Limit(5),
		spotify.Timerange("medium_term"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user's top artists: %v", err)
	}

	if topArtists == nil || len(topArtists.Artists) == 0 {
		return nil, fmt.Errorf("no top artists found for user")
	}

	fmt.Printf("Found %d top artists\n", len(topArtists.Artists))
	return topArtists.Artists, nil
}

// GetUserTopTracks retrieves the user's top tracks from Spotify
func GetUserTopTracks(client *spotify.Client) ([]spotify.FullTrack, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user's top tracks
	topTracks, err := client.CurrentUsersTopTracks(
		ctx,
		spotify.Limit(5),
		spotify.Timerange("medium_term"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user's top tracks: %v", err)
	}

	if topTracks == nil || len(topTracks.Tracks) == 0 {
		return nil, fmt.Errorf("no top tracks found for user")
	}

	fmt.Printf("Found %d top tracks\n", len(topTracks.Tracks))
	return topTracks.Tracks, nil
}

// GetPersonalizedRecommendations gets recommendations based on user's top tracks and artists
func GetPersonalizedRecommendations(mood string, client *spotify.Client) ([]spotify.FullTrack, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Get user's top artists and tracks
	topArtists, artistErr := GetUserTopArtists(client)
	topTracks, trackErr := GetUserTopTracks(client)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// If we have top artists or tracks, use them for recommendations
	if (artistErr == nil && len(topArtists) > 0) || (trackErr == nil && len(topTracks) > 0) {
		fmt.Println("Creating personalized recommendations based on your music taste and current mood...")
		
		// Create seed artists and tracks
		var seedArtists []spotify.ID
		var seedTracks []spotify.ID
		
		if artistErr == nil && len(topArtists) > 0 {
			// Use up to 2 top artists as seeds
			for i := 0; i < min(2, len(topArtists)); i++ {
				seedArtists = append(seedArtists, topArtists[i].ID)
				fmt.Printf("Using top artist as seed: %s\n", topArtists[i].Name)
			}
		}
		
		if trackErr == nil && len(topTracks) > 0 {
			// Use up to 2 top tracks as seeds
			for i := 0; i < min(2, len(topTracks)); i++ {
				seedTracks = append(seedTracks, topTracks[i].ID)
				fmt.Printf("Using top track as seed: %s by %s\n", topTracks[i].Name, topTracks[i].Artists[0].Name)
			}
		}
		
		// Define mood-based attributes
		attrs := spotify.NewTrackAttributes()
		
		switch mood {
		case "energetic":
			attrs = attrs.MinEnergy(0.7).MinDanceability(0.6).TargetValence(0.8)
		case "relaxed":
			attrs = attrs.MaxEnergy(0.5).MinValence(0.3).TargetAcousticness(0.8)
		case "intense":
			attrs = attrs.MinEnergy(0.8).MaxValence(0.4).TargetLoudness(0.8)
		case "thoughtful":
			attrs = attrs.MaxEnergy(0.6).TargetInstrumentalness(0.5).TargetValence(0.5)
		default:
			attrs = attrs.TargetEnergy(0.6).TargetDanceability(0.6)
		}
		
		// Create seeds
		seeds := spotify.Seeds{
			Artists: seedArtists,
			Tracks:  seedTracks,
		}
		
		// Add a genre seed if we have room (max 5 seeds total)
		if len(seedArtists) + len(seedTracks) < 5 {
			switch mood {
			case "energetic":
				seeds.Genres = []string{"pop"}
			case "relaxed":
				seeds.Genres = []string{"chill"}
			case "intense":
				seeds.Genres = []string{"rock"}
			case "thoughtful":
				seeds.Genres = []string{"indie"}
			}
		}
		
		// Get recommendations
		fmt.Printf("Getting recommendations with %d artist seeds, %d track seeds, and %d genre seeds\n", 
			len(seeds.Artists), len(seeds.Tracks), len(seeds.Genres))
		
		recommendations, err := client.GetRecommendations(
			ctx,
			seeds,
			attrs,
			spotify.Limit(20),
		)
		
		if err == nil && recommendations != nil && len(recommendations.Tracks) > 0 {
			fmt.Printf("Found %d personalized recommendations\n", len(recommendations.Tracks))
			
			// Get the IDs of the recommended tracks
			trackIDs := make([]spotify.ID, 0, len(recommendations.Tracks))
			for _, track := range recommendations.Tracks {
				trackIDs = append(trackIDs, track.ID)
			}
			
			// Get the full tracks in batches of 20 (API limit)
			var fullTracks []spotify.FullTrack
			
			for i := 0; i < len(trackIDs); i += 20 {
				end := i + 20
				if end > len(trackIDs) {
					end = len(trackIDs)
				}
				
				batchIDs := trackIDs[i:end]
				tracks, err := client.GetTracks(ctx, batchIDs)
				if err == nil && len(tracks) > 0 {
					// Convert []*FullTrack to []FullTrack
					for _, track := range tracks {
						if track != nil {
							fullTracks = append(fullTracks, *track)
						}
					}
				}
			}
			
			if len(fullTracks) > 0 {
				return fullTracks, nil
			}
		}
		
		fmt.Printf("Error getting personalized recommendations: %v\n", err)
		fmt.Println("Falling back to search-based recommendations...")
	}
	
	// Fallback to search-based recommendations
	return GetSearchBasedRecommendations(mood, client)
}

// GetSearchBasedRecommendations gets recommendations based on search queries
func GetSearchBasedRecommendations(mood string, client *spotify.Client) ([]spotify.FullTrack, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
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
	
	// Search for tracks
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