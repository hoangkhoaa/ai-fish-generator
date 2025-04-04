package config

import (
	"os"
	"strconv"
)

// LoadConfig loads application configuration from environment variables
func LoadConfig() (*AppConfig, error) {
	// Load environment variables from .env file if it exists
	LoadEnv(".env")

	return &AppConfig{
		API: APIConfig{
			Weather:    os.Getenv("OPENWEATHER_API_KEY"),
			News:       os.Getenv("NEWSAPI_KEY"),
			MetalPrice: os.Getenv("METALPRICE_API_KEY"),
		},
		AI: AIConfig{
			GoogleAPIKey: os.Getenv("GEMINI_API_KEY"),
			UseAI:        os.Getenv("USE_AI") == "true" || os.Getenv("USE_AI") == "1",
		},
		MongoDB: MongoDBConfig{
			Enabled:  os.Getenv("MONGO_URI") != "",
			URI:      getMongoURI(),
			Database: getMongoDB(),
		},
	}, nil
}

// AppConfig is the main application configuration
type AppConfig struct {
	API     APIConfig
	AI      AIConfig
	MongoDB MongoDBConfig
}

// APIConfig contains API keys for various services
type APIConfig struct {
	Weather    string
	News       string
	MetalPrice string
}

// AIConfig contains AI-related configuration
type AIConfig struct {
	GoogleAPIKey string
	UseAI        bool
}

// MongoDBConfig contains MongoDB connection configuration
type MongoDBConfig struct {
	Enabled  bool
	URI      string
	Database string
}

// Helper functions

// getMongoURI returns the MongoDB connection URI
func getMongoURI() string {
	// If a complete URI is provided, use it
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		return uri
	}

	// Otherwise, construct a URI from the other parameters
	user := os.Getenv("MONGO_USER")
	password := os.Getenv("MONGO_PASSWORD")

	if user != "" && password != "" {
		return "mongodb://" + user + ":" + password + "@localhost:27017"
	}

	// Default local connection
	return "mongodb://localhost:27017"
}

// getMongoDB returns the MongoDB database name
func getMongoDB() string {
	if db := os.Getenv("MONGO_DB"); db != "" {
		return db
	}
	return "fish_generator"
}

// GetCollectionIntervals returns the intervals for data collection
func GetCollectionIntervals() (weather, price, news int) {
	// Parse collection intervals with fallbacks
	weatherInterval, _ := strconv.Atoi(os.Getenv("WEATHER_INTERVAL"))
	if weatherInterval <= 0 {
		weatherInterval = 6 // Default: collect weather data every 6 hours
	}

	priceInterval, _ := strconv.Atoi(os.Getenv("PRICE_INTERVAL"))
	if priceInterval <= 0 {
		priceInterval = 12 // Default: collect price data every 12 hours
	}

	newsInterval, _ := strconv.Atoi(os.Getenv("NEWS_INTERVAL"))
	if newsInterval <= 0 {
		newsInterval = 12 // Default: collect news data every 12 hours
	}

	return weatherInterval, priceInterval, newsInterval
}
