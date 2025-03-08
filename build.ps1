# Read environment variables from .env file
$envFile = ".env"
$envVars = @{}

if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            $envVars[$key] = $value
        }
    }
}

# Extract the values we need
$spotifyClientID = $envVars["SPOTIFY_CLIENT_ID"]
$spotifyClientSecret = $envVars["SPOTIFY_CLIENT_SECRET"]
$weatherAPIKey = $envVars["WEATHER_API_KEY"]

# Verify we have all required values
if (-not $spotifyClientID -or -not $spotifyClientSecret -or -not $weatherAPIKey) {
    Write-Error "Missing required environment variables in .env file"
    exit 1
}

# Build the application with ldflags to set the variables
$buildCmd = "go build -ldflags=`"-X main.spotifyClientID=$spotifyClientID -X main.spotifyClientSecret=$spotifyClientSecret -X main.weatherAPIKey=$weatherAPIKey`" -o vibecast.exe"

Write-Host "Building application with embedded environment variables..."
Invoke-Expression $buildCmd

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Your application has been compiled to vibecast.exe with embedded environment variables."
    Write-Host "You can now distribute this single executable without needing the .env file."
} else {
    Write-Error "Build failed with exit code $LASTEXITCODE"
} 