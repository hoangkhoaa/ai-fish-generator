package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration loaded from environment or .env file
type Config struct {
	GeminiAPIKey   string
	UseAI          bool
	TestMode       bool
	OpenWeatherKey string
	EIAKey         string
	NewsAPIKey     string
	MetalPriceKey  string

	// MongoDB connection details
	MongoURI      string
	MongoDB       string
	MongoUser     string
	MongoPassword string

	// Collection intervals (in hours)
	WeatherInterval    int
	PriceInterval      int
	NewsInterval       int
	GenerationCooldown int // in minutes
}

// LoadEnv loads environment variables from a .env file
func LoadEnv(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		// It's okay if the file doesn't exist
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments or empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse each line as key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if they exist
		value = strings.Trim(value, `"'`)

		// Set the environment variable if it's not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// NewConfig creates a new Config from environment variables
func NewConfig() *Config {
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

	generationCooldown, _ := strconv.Atoi(os.Getenv("GENERATION_COOLDOWN"))
	if generationCooldown <= 0 {
		generationCooldown = 15 // Default: 15 minutes between fish generations
	}

	return &Config{
		GeminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		UseAI:          os.Getenv("USE_AI") == "true" || os.Getenv("USE_AI") == "1",
		TestMode:       os.Getenv("TEST_MODE") == "true" || os.Getenv("TEST_MODE") == "1",
		OpenWeatherKey: os.Getenv("OPENWEATHER_API_KEY"),
		EIAKey:         os.Getenv("EIA_API_KEY"),
		NewsAPIKey:     os.Getenv("NEWSAPI_KEY"),
		MetalPriceKey:  os.Getenv("METALPRICE_API_KEY"),

		// MongoDB connection details
		MongoURI:      os.Getenv("MONGO_URI"),
		MongoDB:       os.Getenv("MONGO_DB"),
		MongoUser:     os.Getenv("MONGO_USER"),
		MongoPassword: os.Getenv("MONGO_PASSWORD"),

		// Collection intervals
		WeatherInterval:    weatherInterval,
		PriceInterval:      priceInterval,
		NewsInterval:       newsInterval,
		GenerationCooldown: generationCooldown,
	}
}

// GetWeatherInterval returns the weather collection interval as a time.Duration
func (c *Config) GetWeatherInterval() time.Duration {
	return time.Duration(c.WeatherInterval) * time.Hour
}

// GetPriceInterval returns the price collection interval as a time.Duration
func (c *Config) GetPriceInterval() time.Duration {
	return time.Duration(c.PriceInterval) * time.Hour
}

// GetNewsInterval returns the news collection interval as a time.Duration
func (c *Config) GetNewsInterval() time.Duration {
	return time.Duration(c.NewsInterval) * time.Hour
}

// GetGenerationCooldown returns the fish generation cooldown as a time.Duration
func (c *Config) GetGenerationCooldown() time.Duration {
	return time.Duration(c.GenerationCooldown) * time.Minute
}

// GetMongoURI returns the complete MongoDB connection URI
func (c *Config) GetMongoURI() string {
	// If a complete URI is provided, use it
	if c.MongoURI != "" {
		return c.MongoURI
	}

	// Otherwise, construct a URI from the other parameters
	if c.MongoUser != "" && c.MongoPassword != "" {
		return fmt.Sprintf("mongodb://%s:%s@localhost:27017", c.MongoUser, c.MongoPassword)
	}

	// Default local connection
	return "mongodb://localhost:27017"
}

// GetMongoDB returns the MongoDB database name
func (c *Config) GetMongoDB() string {
	if c.MongoDB != "" {
		return c.MongoDB
	}
	return "fish_generator"
}
