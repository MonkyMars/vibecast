package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Weather struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func GetWeather(city string) (*Weather, error) {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}

	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var weather Weather

	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return nil, err
	}

	return &weather, nil
}

func GetMoodFromWeather(city string) string {
	weather, err := GetWeather(city)
	if err != nil {
		fmt.Println("Error getting weather:", err)
		return "neutral" // Default mood on error
	}

	if weather == nil || len(weather.Weather) == 0 {
		fmt.Println("No weather data available")
		return "neutral"
	}

	description := weather.Weather[0].Description
	fmt.Println("Weather description:", description)

	switch {
	case description == "clear sky":
		return "energetic"
	case description == "overcast clouds":
		return "thoughtful"
	case description == "light rain":
		return "relaxed"
	case description == "thunderstorm":
		return "intense"
	default:
		return "neutral"
	}
}

func GetWeatherAndMood() (*Weather, string) {
	var city string
	fmt.Println("Enter city: ")
	fmt.Scanln(&city)

	weather, err := GetWeather(city)
	if err != nil {
		fmt.Println("Error getting weather data:", err)
		return &Weather{}, "neutral"
	}

	// Check if weather data is valid
	if weather == nil || len(weather.Weather) == 0 {
		fmt.Println("No weather data available for", city)
		return &Weather{}, "neutral"
	}

	mood := GetMoodFromWeather(city)
	return weather, mood
}
