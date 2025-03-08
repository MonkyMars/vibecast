package main

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
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

	// Get user's liked songs - this is critical for strict filtering
	likedTracks, likedTracksErr := GetUserLikedTracks(client)
	if likedTracksErr != nil {
		return nil, fmt.Errorf("failed to get liked songs: %v", likedTracksErr)
	}

	if len(likedTracks) == 0 {
		return nil, fmt.Errorf("no liked songs found - please like some songs on Spotify first")
	}

	fmt.Println("STRICT FILTERING: Only songs you've explicitly liked will be included in the playlist")
	fmt.Printf("MOOD ACCURACY: Using audio analysis to ensure songs match the '%s' mood\n", mood)

	// Get user's liked artists for additional filtering
	likedArtists, _ := GetUserLikedArtists(client)

	// Get user's top artists and tracks for recommendation seeds
	topArtists, _ := GetUserTopArtists(client)
	topTracks, _ := GetUserTopTracks(client)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// We'll collect tracks from multiple sources to ensure we have enough
	var allTracks []spotify.FullTrack

	// Track IDs we've already seen to avoid duplicates
	seenTrackIDs := make(map[string]bool)

	// Function to add unique tracks to our collection
	addUniqueTracks := func(tracks []spotify.FullTrack) {
		for _, track := range tracks {
			trackID := track.ID.String()
			if !seenTrackIDs[trackID] && likedTracks[trackID] {
				// Only add if the track is in the user's liked songs
				allTracks = append(allTracks, track)
				seenTrackIDs[trackID] = true
			}
		}
	}

	fmt.Println("Creating personalized recommendations based on your music taste and current mood...")
	fmt.Println("Only including songs you've explicitly liked that match the current mood!")

	// 1. First, directly add all liked songs that match the mood
	fmt.Println("Analyzing your liked songs to find ones that match the current mood...")

	// Get all the user's liked songs
	var userLikedSongs []spotify.FullTrack
	var likedTrackIDs []spotify.ID

	// Get tracks in batches of 20 (API limit)
	var trackIDs []spotify.ID
	for trackID := range likedTracks {
		trackIDs = append(trackIDs, spotify.ID(trackID))
		likedTrackIDs = append(likedTrackIDs, spotify.ID(trackID))

		// Process in batches of 20
		if len(trackIDs) >= 20 {
			tracks, err := client.GetTracks(ctx, trackIDs)
			if err == nil && len(tracks) > 0 {
				for _, track := range tracks {
					if track != nil {
						userLikedSongs = append(userLikedSongs, *track)
					}
				}
			}
			trackIDs = nil // Reset for next batch
		}
	}

	// Process any remaining tracks
	if len(trackIDs) > 0 {
		tracks, err := client.GetTracks(ctx, trackIDs)
		if err == nil && len(tracks) > 0 {
			for _, track := range tracks {
				if track != nil {
					userLikedSongs = append(userLikedSongs, *track)
				}
			}
		}
	}

	fmt.Printf("Found %d liked songs in your library\n", len(userLikedSongs))

	// 2. Analyze audio features to find tracks that match the mood
	fmt.Println("Analyzing audio features of your liked songs to match the mood...")

	// Get matching track IDs based on audio features
	matchingTrackIDs, err := AnalyzeAudioFeaturesForMood(client, likedTrackIDs, mood)
	if err != nil {
		fmt.Printf("Warning: Error analyzing audio features: %v\n", err)
		fmt.Println("Falling back to genre-based and playlist-based mood matching...")

		// Since we can't use audio features, we'll rely more heavily on genre matching
		// and mood-based playlists to ensure accurate mood matching
	} else {
		fmt.Printf("Found %d tracks that match the '%s' mood based on audio features\n", len(matchingTrackIDs), mood)

		// Create a map for quick lookup
		matchingTrackIDMap := make(map[string]bool)
		for _, id := range matchingTrackIDs {
			matchingTrackIDMap[id.String()] = true
		}

		// Add matching tracks to our collection
		for _, track := range userLikedSongs {
			if matchingTrackIDMap[track.ID.String()] {
				if !seenTrackIDs[track.ID.String()] {
					allTracks = append(allTracks, track)
					seenTrackIDs[track.ID.String()] = true
				}
			}
		}

		fmt.Printf("Added %d tracks that match the mood based on audio features\n", len(allTracks))
	}

	// 3. Try with genre-based filtering if we don't have enough tracks
	if len(allTracks) < 50 {
		fmt.Println("Using enhanced genre-based filtering to find mood-matching tracks...")

		// Get genres that match the mood
		moodGenres := GetMoodMatchingGenres(mood)
		fmt.Printf("Using %d genres associated with the '%s' mood\n", len(moodGenres), mood)

		// Create a map for quick genre lookup
		moodGenreMap := make(map[string]bool)
		for _, genre := range moodGenres {
			moodGenreMap[strings.ToLower(genre)] = true
		}

		// Track artist genres to avoid repeated API calls
		artistGenreCache := make(map[string][]string)

		// Filter tracks by genre
		for _, track := range userLikedSongs {
			// Skip tracks we've already added
			if seenTrackIDs[track.ID.String()] {
				continue
			}

			// Try to get the track's genres through its artists
			trackMatchesMood := false

			for _, artist := range track.Artists {
				artistID := artist.ID.String()

				// Check if we've already cached this artist's genres
				var artistGenres []string
				var ok bool

				if artistGenres, ok = artistGenreCache[artistID]; !ok {
					// Not in cache, fetch from API
					artistInfo, err := client.GetArtist(ctx, artist.ID)
					if err != nil {
						continue
					}

					artistGenres = artistInfo.Genres
					artistGenreCache[artistID] = artistGenres
				}

				// Check if any of the artist's genres match our mood genres
				for _, artistGenre := range artistGenres {
					artistGenreLower := strings.ToLower(artistGenre)

					// Direct match
					if moodGenreMap[artistGenreLower] {
						trackMatchesMood = true
						break
					}

					// Partial match (genre contains a mood genre keyword)
					for moodGenre := range moodGenreMap {
						if strings.Contains(artistGenreLower, moodGenre) {
							trackMatchesMood = true
							break
						}
					}

					if trackMatchesMood {
						break
					}
				}

				if trackMatchesMood {
					break
				}
			}

			if trackMatchesMood {
				allTracks = append(allTracks, track)
				seenTrackIDs[track.ID.String()] = true

				if len(allTracks) >= 100 {
					break
				}
			}
		}

		fmt.Printf("Added %d tracks based on enhanced genre matching\n", len(allTracks))
	}

	// 4. Try with mood-based playlists if we still don't have enough tracks
	if len(allTracks) < 50 {
		fmt.Println("Looking for tracks in popular mood-based playlists...")

		// Try to get tracks from multiple mood-based playlists
		var moodPlaylistTracks []spotify.FullTrack

		// Try different search queries for the mood
		searchQueries := getMoodPlaylistSearchQueries(mood)

		for _, query := range searchQueries {
			if len(moodPlaylistTracks) >= 200 {
				break
			}

			fmt.Printf("Searching for '%s' playlists...\n", query)

			results, err := client.Search(ctx, query, spotify.SearchTypePlaylist, spotify.Limit(5))
			if err != nil || results == nil || results.Playlists == nil || len(results.Playlists.Playlists) == 0 {
				continue
			}

			// Get tracks from each playlist
			for _, playlist := range results.Playlists.Playlists {
				if len(moodPlaylistTracks) >= 200 {
					break
				}

				fmt.Printf("Checking playlist: %s\n", playlist.Name)

				playlistTracks, err := client.GetPlaylistItems(ctx, playlist.ID)
				if err != nil {
					continue
				}

				// Extract full tracks
				for _, item := range playlistTracks.Items {
					if item.Track.Track != nil {
						// Convert PlaylistTrack to FullTrack
						track := item.Track.Track
						moodPlaylistTracks = append(moodPlaylistTracks, *track)
					}
				}
			}
		}

		fmt.Printf("Found %d tracks from mood-based playlists\n", len(moodPlaylistTracks))

		// Filter to only include tracks in the user's library
		for _, track := range moodPlaylistTracks {
			if likedTracks[track.ID.String()] && !seenTrackIDs[track.ID.String()] {
				allTracks = append(allTracks, track)
				seenTrackIDs[track.ID.String()] = true

				if len(allTracks) >= 100 {
					break
				}
			}
		}

		fmt.Printf("Added tracks from mood-based playlists, now have %d tracks\n", len(allTracks))
	}

	// 5. If we still don't have enough tracks, try with recommendations
	if len(allTracks) < 50 {
		fmt.Println("Using Spotify recommendations to find more tracks...")

		// Create seed artists and tracks
		var seedArtists []spotify.ID
		var seedTracks []spotify.ID

		// Prioritize artists that are in the user's liked artists
		if len(topArtists) > 0 && len(likedArtists) > 0 {
			for i := 0; i < min(2, len(topArtists)); i++ {
				if likedArtists[topArtists[i].ID.String()] {
					seedArtists = append(seedArtists, topArtists[i].ID)
					fmt.Printf("Using top artist as seed: %s (in your liked artists)\n", topArtists[i].Name)
				}
			}
		}

		// Add some top tracks if we have room
		if len(topTracks) > 0 && len(seedArtists) < 5 {
			for i := 0; i < min(5-len(seedArtists), len(topTracks)); i++ {
				// Only use tracks that are in the user's liked songs
				if likedTracks[topTracks[i].ID.String()] {
					seedTracks = append(seedTracks, topTracks[i].ID)
					fmt.Printf("Using top track as seed: %s by %s (in your liked songs)\n",
						topTracks[i].Name, topTracks[i].Artists[0].Name)
				}
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

		// Add genre seeds if we have room (max 5 seeds total)
		if len(seedArtists)+len(seedTracks) < 5 {
			// Get more genres per mood
			switch mood {
			case "energetic":
				seeds.Genres = []string{"pop", "dance", "edm", "party", "house"}[:min(5-len(seedArtists)-len(seedTracks), 5)]
			case "relaxed":
				seeds.Genres = []string{"chill", "acoustic", "ambient", "jazz", "lofi"}[:min(5-len(seedArtists)-len(seedTracks), 5)]
			case "intense":
				seeds.Genres = []string{"rock", "metal", "punk", "hard-rock", "alt-rock"}[:min(5-len(seedArtists)-len(seedTracks), 5)]
			case "thoughtful":
				seeds.Genres = []string{"indie", "folk", "classical", "singer-songwriter", "ambient"}[:min(5-len(seedArtists)-len(seedTracks), 5)]
			default:
				seeds.Genres = []string{"pop", "indie", "alternative", "rock", "electronic"}[:min(5-len(seedArtists)-len(seedTracks), 5)]
			}
		}

		// Get recommendations
		fmt.Printf("Getting recommendations with %d artist seeds, %d track seeds, and %d genre seeds\n",
			len(seeds.Artists), len(seeds.Tracks), len(seeds.Genres))

		recommendations, err := client.GetRecommendations(
			ctx,
			seeds,
			attrs,
			spotify.Limit(100), // Request more tracks to have enough after filtering
		)

		if err == nil && recommendations != nil && len(recommendations.Tracks) > 0 {
			fmt.Printf("Found %d initial recommendations\n", len(recommendations.Tracks))

			// Get the IDs of the recommended tracks
			recTrackIDs := make([]spotify.ID, 0, len(recommendations.Tracks))
			for _, track := range recommendations.Tracks {
				recTrackIDs = append(recTrackIDs, track.ID)
			}

			// Get the full tracks in batches of 20 (API limit)
			var fullTracks []spotify.FullTrack

			for i := 0; i < len(recTrackIDs); i += 20 {
				end := i + 20
				if end > len(recTrackIDs) {
					end = len(recTrackIDs)
				}

				batchIDs := recTrackIDs[i:end]
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

			// Add these tracks to our collection
			addUniqueTracks(fullTracks)
			fmt.Printf("Added %d tracks from personalized recommendations (only those in your liked songs)\n", len(allTracks))
		}
	}

	fmt.Printf("After all searches, found %d tracks from your liked songs that match the mood\n", len(allTracks))

	// Final filtering to ensure we only have tracks by liked artists
	filteredTracks := FilterTracksByLikedSongs(allTracks, likedTracks)

	if len(filteredTracks) == 0 {
		return nil, fmt.Errorf("no tracks found in your liked songs that match the criteria - please like more songs on Spotify")
	}

	// Limit the number of songs per artist to ensure variety
	const maxSongsPerArtist = 5
	filteredTracks = LimitSongsPerArtist(filteredTracks, maxSongsPerArtist)

	// Shuffle the tracks for variety
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(filteredTracks), func(i, j int) {
		filteredTracks[i], filteredTracks[j] = filteredTracks[j], filteredTracks[i]
	})

	// Limit to 50 tracks for the playlist
	if len(filteredTracks) > 50 {
		filteredTracks = filteredTracks[:50]
	}

	fmt.Printf("Final playlist will contain %d tracks, all from your liked songs that match the '%s' mood, with no artist having more than %d songs\n",
		len(filteredTracks), mood, maxSongsPerArtist)
	return filteredTracks, nil
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

// GetUserLikedArtists retrieves the user's liked songs and extracts unique artists
func GetUserLikedArtists(client *spotify.Client) (map[string]bool, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get user's saved tracks (liked songs)
	limit := 50 // Maximum allowed by Spotify API
	offset := 0
	likedArtists := make(map[string]bool)

	fmt.Println("Fetching your liked songs to identify your preferred artists...")

	// Keep track of how many tracks we've processed
	totalProcessed := 0

	for {
		savedTracks, err := client.CurrentUsersTracks(ctx, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			return nil, fmt.Errorf("failed to get user's liked songs: %v", err)
		}

		if savedTracks == nil || len(savedTracks.Tracks) == 0 {
			break // No more tracks
		}

		// Extract artists from each track
		for _, item := range savedTracks.Tracks {
			for _, artist := range item.FullTrack.Artists {
				likedArtists[artist.ID.String()] = true
			}
		}

		totalProcessed += len(savedTracks.Tracks)
		fmt.Printf("Processed %d liked songs, found %d unique artists so far...\n",
			totalProcessed, len(likedArtists))

		// If we got fewer tracks than requested, we've reached the end
		if len(savedTracks.Tracks) < limit {
			break
		}

		// Move to the next page
		offset += limit

		// Limit to 1000 tracks (20 pages) to avoid rate limiting
		if offset >= 1000 {
			fmt.Println("Reached the limit of 1000 tracks. If you have more liked songs, not all artists may be included.")
			break
		}
	}

	if len(likedArtists) == 0 {
		return nil, fmt.Errorf("no artists found in your liked songs")
	}

	fmt.Printf("Found %d unique artists in your liked songs\n", len(likedArtists))
	return likedArtists, nil
}

// FilterTracksByLikedArtists filters tracks to only include those by artists in the user's liked songs
func FilterTracksByLikedArtists(tracks []spotify.FullTrack, likedArtists map[string]bool) []spotify.FullTrack {
	if len(likedArtists) == 0 {
		fmt.Println("WARNING: No liked artists found. Cannot filter tracks.")
		return tracks // No filtering if we don't have liked artists
	}

	filteredTracks := make([]spotify.FullTrack, 0)
	skippedTracks := 0

	for _, track := range tracks {
		// Check if any of the track's artists are in the user's liked artists
		artistFound := false
		for _, artist := range track.Artists {
			if likedArtists[artist.ID.String()] {
				artistFound = true
				break // Found a match, no need to check other artists
			}
		}

		if artistFound {
			filteredTracks = append(filteredTracks, track)
		} else {
			skippedTracks++
		}
	}

	fmt.Printf("Filtered out %d tracks that weren't by your liked artists\n", skippedTracks)
	return filteredTracks
}

// LimitSongsPerArtist ensures no artist has more than the specified maximum number of songs
func LimitSongsPerArtist(tracks []spotify.FullTrack, maxSongsPerArtist int) []spotify.FullTrack {
	if len(tracks) == 0 || maxSongsPerArtist <= 0 {
		return tracks
	}

	// Count songs per artist
	artistSongCount := make(map[string]int)

	// Result tracks with limited songs per artist
	var limitedTracks []spotify.FullTrack

	// Track which songs we've already processed
	processedTrackIDs := make(map[string]bool)

	// First pass: count songs per artist
	for _, track := range tracks {
		for _, artist := range track.Artists {
			artistID := artist.ID.String()
			artistSongCount[artistID]++
		}
	}

	fmt.Println("Artist distribution before limiting:")
	printTopArtistCounts(artistSongCount, 5)

	// Second pass: add tracks while respecting the limit
	for _, track := range tracks {
		trackID := track.ID.String()

		// Skip if we've already processed this track
		if processedTrackIDs[trackID] {
			continue
		}

		// Check if any artist of this track has reached the limit
		exceedsLimit := false
		for _, artist := range track.Artists {
			artistID := artist.ID.String()
			if artistSongCount[artistID] > maxSongsPerArtist {
				exceedsLimit = true
				// Reduce the count for this artist since we're skipping this track
				artistSongCount[artistID]--
				break
			}
		}

		// If no artist has reached the limit, add the track
		if !exceedsLimit {
			limitedTracks = append(limitedTracks, track)
			processedTrackIDs[trackID] = true

			// Reduce the count for all artists in this track
			for _, artist := range track.Artists {
				artistID := artist.ID.String()
				artistSongCount[artistID]--
			}
		}
	}

	fmt.Printf("Limited playlist from %d to %d tracks to ensure no artist has more than %d songs\n",
		len(tracks), len(limitedTracks), maxSongsPerArtist)

	return limitedTracks
}

// Helper function to print the top N artists by song count
func printTopArtistCounts(artistCounts map[string]int, topN int) {
	// Convert map to slice for sorting
	type artistCount struct {
		ID    string
		Count int
	}

	counts := make([]artistCount, 0, len(artistCounts))
	for id, count := range artistCounts {
		counts = append(counts, artistCount{ID: id, Count: count})
	}

	// Sort by count (descending)
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	// Print top N
	fmt.Println("Top artists by song count:")
	for i := 0; i < min(topN, len(counts)); i++ {
		fmt.Printf("  Artist ID %s: %d songs\n", counts[i].ID, counts[i].Count)
	}
}

// GetUserLikedTracks retrieves the user's liked songs and creates a map for quick lookup
func GetUserLikedTracks(client *spotify.Client) (map[string]bool, error) {
	if client == nil {
		return nil, fmt.Errorf("spotify client is nil")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get user's saved tracks (liked songs)
	limit := 50 // Maximum allowed by Spotify API
	offset := 0
	likedTracks := make(map[string]bool)

	fmt.Println("Fetching your liked songs...")

	// Keep track of how many tracks we've processed
	totalProcessed := 0

	for {
		savedTracks, err := client.CurrentUsersTracks(ctx, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			return nil, fmt.Errorf("failed to get user's liked songs: %v", err)
		}

		if savedTracks == nil || len(savedTracks.Tracks) == 0 {
			break // No more tracks
		}

		// Add each track ID to the map
		for _, item := range savedTracks.Tracks {
			likedTracks[item.FullTrack.ID.String()] = true
		}

		totalProcessed += len(savedTracks.Tracks)
		fmt.Printf("Processed %d liked songs...\n", totalProcessed)

		// If we got fewer tracks than requested, we've reached the end
		if len(savedTracks.Tracks) < limit {
			break
		}

		// Move to the next page
		offset += limit

		// Limit to 1000 tracks (20 pages) to avoid rate limiting
		if offset >= 1000 {
			fmt.Println("Reached the limit of 1000 tracks. If you have more liked songs, not all may be included.")
			break
		}
	}

	if len(likedTracks) == 0 {
		return nil, fmt.Errorf("no liked songs found in your library")
	}

	fmt.Printf("Found %d liked songs in your library\n", len(likedTracks))
	return likedTracks, nil
}

// FilterTracksByLikedSongs filters tracks to only include those that are in the user's liked songs
func FilterTracksByLikedSongs(tracks []spotify.FullTrack, likedTracks map[string]bool) []spotify.FullTrack {
	if len(likedTracks) == 0 {
		fmt.Println("WARNING: No liked songs found. Cannot filter tracks.")
		return tracks // No filtering if we don't have liked songs
	}

	filteredTracks := make([]spotify.FullTrack, 0)
	skippedTracks := 0

	for _, track := range tracks {
		if likedTracks[track.ID.String()] {
			filteredTracks = append(filteredTracks, track)
		} else {
			skippedTracks++
		}
	}

	fmt.Printf("Filtered out %d tracks that weren't in your liked songs\n", skippedTracks)
	return filteredTracks
}

// AudioFeatureThresholds defines the thresholds for different moods
type AudioFeatureThresholds struct {
	MinEnergy           float32
	MaxEnergy           float32
	MinDanceability     float32
	MaxDanceability     float32
	MinValence          float32
	MaxValence          float32
	MinTempo            float32
	MaxTempo            float32
	MinAcousticness     float32
	MaxAcousticness     float32
	MinInstrumentalness float32
	MaxInstrumentalness float32
}

// GetMoodThresholds returns the audio feature thresholds for a specific mood
func GetMoodThresholds(mood string) AudioFeatureThresholds {
	switch mood {
	case "energetic":
		return AudioFeatureThresholds{
			MinEnergy:           0.7,
			MaxEnergy:           1.0,
			MinDanceability:     0.6,
			MaxDanceability:     1.0,
			MinValence:          0.5, // Moderately positive to very positive
			MaxValence:          1.0,
			MinTempo:            120, // Faster tempo
			MaxTempo:            300,
			MaxAcousticness:     0.4, // Less acoustic
			MaxInstrumentalness: 0.3, // Mostly with vocals
		}
	case "relaxed":
		return AudioFeatureThresholds{
			MinEnergy:           0.0,
			MaxEnergy:           0.5,
			MinDanceability:     0.0,
			MaxDanceability:     0.6,
			MinValence:          0.0,
			MaxValence:          0.7,
			MinTempo:            0,
			MaxTempo:            110,
			MinAcousticness:     0.4, // More acoustic
			MaxInstrumentalness: 1.0, // Can be instrumental
		}
	case "intense":
		return AudioFeatureThresholds{
			MinEnergy:           0.8,
			MaxEnergy:           1.0,
			MinDanceability:     0.0,
			MaxDanceability:     1.0,
			MinValence:          0.0,
			MaxValence:          0.5, // Less positive, more serious
			MinTempo:            100,
			MaxTempo:            300,
			MaxAcousticness:     0.3, // Less acoustic
			MaxInstrumentalness: 0.5,
		}
	case "thoughtful":
		return AudioFeatureThresholds{
			MinEnergy:           0.0,
			MaxEnergy:           0.6,
			MinDanceability:     0.0,
			MaxDanceability:     0.5,
			MinValence:          0.0,
			MaxValence:          0.6,
			MinTempo:            0,
			MaxTempo:            120,
			MinAcousticness:     0.3,
			MinInstrumentalness: 0.2,
		}
	default: // neutral
		return AudioFeatureThresholds{
			MinEnergy:       0.0,
			MaxEnergy:       1.0,
			MinDanceability: 0.0,
			MaxDanceability: 1.0,
			MinValence:      0.0,
			MaxValence:      1.0,
			MinTempo:        0,
			MaxTempo:        300,
		}
	}
}

// GetMoodMatchingGenres returns genres that match a specific mood
func GetMoodMatchingGenres(mood string) []string {
	switch mood {
	case "energetic":
		return []string{
			"dance", "edm", "electro", "house", "techno", "trance", "dubstep",
			"pop", "power-pop", "dance-pop", "party", "club",
			"disco", "funk", "happy", "upbeat", "workout", "gym",
		}
	case "relaxed":
		return []string{
			"chill", "acoustic", "ambient", "lofi", "sleep", "study",
			"jazz", "soul", "r-n-b", "folk", "indie-folk",
			"meditation", "calm", "piano", "classical", "soft-rock",
		}
	case "intense":
		return []string{
			"rock", "metal", "hard-rock", "heavy-metal", "punk", "hardcore",
			"alt-rock", "alternative", "grunge", "industrial",
			"emo", "post-hardcore", "thrash", "death-metal",
		}
	case "thoughtful":
		return []string{
			"indie", "indie-pop", "indie-rock", "alternative", "folk",
			"singer-songwriter", "ambient", "post-rock", "experimental",
			"classical", "instrumental", "soundtrack", "piano", "sad",
		}
	default:
		return []string{"pop", "rock", "indie", "alternative"}
	}
}

// AnalyzeAudioFeaturesForMood analyzes audio features for a batch of tracks and returns those that match the mood
func AnalyzeAudioFeaturesForMood(client *spotify.Client, trackIDs []spotify.ID, mood string) ([]spotify.ID, error) {
	if len(trackIDs) == 0 {
		return nil, fmt.Errorf("no tracks to analyze")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get audio features for tracks in batches of 100 (API limit)
	var matchingTrackIDs []spotify.ID
	thresholds := GetMoodThresholds(mood)

	// Try with a small batch first to check if we have access
	if len(trackIDs) > 0 {
		testBatch := trackIDs[:min(5, len(trackIDs))]
		_, testErr := client.GetAudioFeatures(ctx, testBatch...)

		if testErr != nil {
			// If we get a 403 error, we don't have permission to access audio features
			return nil, fmt.Errorf("cannot access audio features API: %v", testErr)
		}
	}

	for i := 0; i < len(trackIDs); i += 100 {
		end := i + 100
		if end > len(trackIDs) {
			end = len(trackIDs)
		}

		batchIDs := trackIDs[i:end]
		audioFeatures, err := client.GetAudioFeatures(ctx, batchIDs...)
		if err != nil {
			fmt.Printf("Error getting audio features for batch %d-%d: %v\n", i, end, err)
			continue
		}

		for j, features := range audioFeatures {
			if features == nil {
				continue
			}

			// Check if the track matches the mood based on audio features
			if matchesMood(features, thresholds) {
				matchingTrackIDs = append(matchingTrackIDs, batchIDs[j])
			}
		}
	}

	return matchingTrackIDs, nil
}

// matchesMood checks if a track's audio features match the mood thresholds
func matchesMood(features *spotify.AudioFeatures, thresholds AudioFeatureThresholds) bool {
	// Energy check
	if thresholds.MinEnergy > 0 && features.Energy < thresholds.MinEnergy {
		return false
	}
	if thresholds.MaxEnergy < 1.0 && features.Energy > thresholds.MaxEnergy {
		return false
	}

	// Danceability check
	if thresholds.MinDanceability > 0 && features.Danceability < thresholds.MinDanceability {
		return false
	}
	if thresholds.MaxDanceability < 1.0 && features.Danceability > thresholds.MaxDanceability {
		return false
	}

	// Valence check (positivity/happiness)
	if thresholds.MinValence > 0 && features.Valence < thresholds.MinValence {
		return false
	}
	if thresholds.MaxValence < 1.0 && features.Valence > thresholds.MaxValence {
		return false
	}

	// Tempo check
	if thresholds.MinTempo > 0 && features.Tempo < thresholds.MinTempo {
		return false
	}
	if thresholds.MaxTempo < 300 && features.Tempo > thresholds.MaxTempo {
		return false
	}

	// Acousticness check for the sake of variety
	if thresholds.MinAcousticness > 0 && features.Acousticness < thresholds.MinAcousticness {
		return false
	}
	if thresholds.MaxAcousticness < 1.0 && features.Acousticness > thresholds.MaxAcousticness {
		return false
	}

	// Instrumentalness check
	if thresholds.MinInstrumentalness > 0 && features.Instrumentalness < thresholds.MinInstrumentalness {
		return false
	}
	if thresholds.MaxInstrumentalness < 1.0 && features.Instrumentalness > thresholds.MaxInstrumentalness {
		return false
	}

	return true
}

// GetMoodBasedPlaylistTracks gets tracks from popular mood-based playlists
func GetMoodBasedPlaylistTracks(client *spotify.Client, mood string) ([]spotify.FullTrack, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Search for mood-based playlists
	var searchQuery string
	switch mood {
	case "energetic":
		searchQuery = "workout energy party upbeat"
	case "relaxed":
		searchQuery = "chill relax calm acoustic"
	case "intense":
		searchQuery = "intense rock metal hardcore"
	case "thoughtful":
		searchQuery = "thoughtful indie ambient calm"
	default:
		searchQuery = "mood"
	}

	results, err := client.Search(ctx, searchQuery, spotify.SearchTypePlaylist, spotify.Limit(5))
	if err != nil {
		return nil, fmt.Errorf("failed to search for mood playlists: %v", err)
	}

	if results == nil || results.Playlists == nil || len(results.Playlists.Playlists) == 0 {
		return nil, fmt.Errorf("no mood playlists found")
	}

	// Get tracks from the first playlist
	var allTracks []spotify.FullTrack

	for _, playlist := range results.Playlists.Playlists {
		playlistTracks, err := client.GetPlaylistItems(ctx, playlist.ID)
		if err != nil {
			continue
		}

		// Extract full tracks
		for _, item := range playlistTracks.Items {
			if item.Track.Track != nil {
				// Convert PlaylistTrack to FullTrack
				track := item.Track.Track
				allTracks = append(allTracks, *track)
			}
		}

		if len(allTracks) >= 50 {
			break
		}
	}

	return allTracks, nil
}

// getMoodPlaylistSearchQueries returns search queries for finding mood-based playlists
func getMoodPlaylistSearchQueries(mood string) []string {
	switch mood {
	case "energetic":
		return []string{
			"workout energy",
			"party upbeat",
			"dance energy",
			"gym motivation",
			"high energy",
		}
	case "relaxed":
		return []string{
			"chill relax",
			"calm acoustic",
			"sleep peaceful",
			"meditation calm",
			"lofi chill",
		}
	case "intense":
		return []string{
			"intense rock",
			"metal hardcore",
			"workout intense",
			"running intense",
			"epic intense",
		}
	case "thoughtful":
		return []string{
			"thoughtful indie",
			"ambient calm",
			"focus concentration",
			"study peaceful",
			"introspective mood",
		}
	default:
		return []string{"mood " + mood}
	}
}
