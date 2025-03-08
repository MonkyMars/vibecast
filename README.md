# VibeCast

A Go application that generates personalized Spotify playlists based on your music taste and the current weather.

## Features

- Creates playlists with songs you've explicitly liked on Spotify
- Matches songs to the current mood based on weather
- Ensures variety by limiting to 5 songs per artist
- Securely handles API credentials

## Setup

### Prerequisites

- Go 1.16 or higher
- Spotify Developer Account
- OpenWeatherMap API Key

### Configuration

1. Copy `.env.example` to `.env`:
   ```
   cp .env.example .env
   ```

2. Edit `.env` with your API credentials:
   ```
   SPOTIFY_CLIENT_ID=your_spotify_client_id
   SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
   WEATHER_API_KEY=your_weather_api_key
   ```

## Building

### Development Build

For development, you can run the application directly:

```
go run .
```

### Production Build (Secure)

For production, use the build script to embed environment variables directly into the executable:

```
.\build.ps1
```

This creates a single `vibecast.exe` file with your API credentials securely embedded. You can distribute this executable without including the `.env` file.

## Usage

1. Run the application:
   ```
   .\vibecast.exe
   ```

2. Open a web browser and navigate to http://localhost:8080/login

3. Log in with your Spotify account

4. Click the button to create a playlist

5. Enter a city when prompted

6. Enjoy your personalized weather-based playlist!

## Security Notes

- The build script embeds your API credentials directly into the executable
- Never commit your `.env` file to version control
- If you need to distribute the application, use the build script to create a secure executable
- Rotate your API credentials if they are ever exposed 