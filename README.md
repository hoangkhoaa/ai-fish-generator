# Fish Generator

A Go application that generates unique fish species influenced by real-time data from various sources.

## Overview

Fish Generator creates fish with different characteristics based on real-world data from various sources:

- Weather conditions (via OpenWeatherMap API)
- Cryptocurrency prices (via CoinGecko API)
- Oil prices (via EIA API)
- News headlines (via NewsAPI)
- Gold prices (via Metal Price API)

Each data source influences different aspects of the generated fish, such as rarity, size, value, and special effects.

## Features

- Real-time data collection from various APIs
- AI-powered fish generation using Google Gemini
- MongoDB integration for data persistence 
- Ocean region-based data collection
- Game statistics effects that impact gameplay
- Configurable test and production modes
- Comprehensive reporting and statistics

## Getting Started

### Prerequisites

- Go 1.18 or higher
- MongoDB (optional, for data persistence)
- API keys for:
  - Google Gemini (for AI-generated fish)
  - OpenWeatherMap (for weather data)
  - EIA - Energy Information Administration (for oil price data)
  - NewsAPI (for news headlines)
  - Metal Price API (for gold price data)

### Installation

1. Clone the repository
2. Create a `.env` file with your API keys and configuration (see configuration section)
3. Run `go build -o fish-generator ./cmd/fish-generate`

### Configuration

Create a `.env` file in the project root based on the provided `.env.example`:

```
# API Keys
GEMINI_API_KEY=your_gemini_api_key
OPENWEATHER_API_KEY=your_openweather_api_key
EIA_API_KEY=your_eia_api_key
NEWSAPI_KEY=your_newsapi_key
METALPRICE_API_KEY=your_metalprice_api_key

# Feature Toggles
USE_AI=true
TEST_MODE=false

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DB=fish_generator
MONGO_USER=
MONGO_PASSWORD=

# Collection Intervals (in hours)
WEATHER_INTERVAL=6
PRICE_INTERVAL=12
NEWS_INTERVAL=12
```

### Running the Application

The application now has a simpler command structure:

```bash
# Show help information
./fish-generator -help

# Start the generator in normal mode (30-minute intervals)
./fish-generator generate

# Start the generator in test mode (5-second intervals)
./fish-generator test

# Show current configuration
./fish-generator config
```

## MongoDB Integration

The application now supports MongoDB for data persistence. When MongoDB is configured, the application will:

1. Store all generated fish with their properties
2. Collect and store weather data every 6 hours (by default)
3. Collect and store price data (BTC, gold, oil) every 12 hours
4. Collect and store news headlines every 12 hours
5. Track collection statistics

The stored data influences future fish generation, particularly for region-based fish. For example, fish generated in a region with stormy weather will have different characteristics than those in sunny regions.

### MongoDB Collections

- `weather`: Weather data for different regions and cities
- `prices`: Price data for cryptocurrencies, gold, and oil
- `news`: News headlines with sentiment analysis
- `fish`: Generated fish data
- `regions`: Ocean region definitions
- `stats`: Collection statistics

### Ocean Regions

The application defines several ocean regions, each with specific characteristics:

- **North Atlantic**: Cold, deep waters with diverse marine life
- **Tropical Pacific**: Warm, clear waters with vibrant coral reefs
- **Mediterranean Sea**: Warm, saltier waters with rich history and biodiversity
- **Arctic Ocean**: Extremely cold waters with unique ice-adapted species
- **South Pacific**: Pristine waters with diverse island ecosystems

Each region has representative cities whose weather is monitored to influence fish generation.

## Game Statistics

Fish can affect various game statistics:

- **Basic stats**: Catch chance, critical catch probability, luck, stamina regeneration
- **Economic stats**: Sell value, market demand, bait cost
- **Exploration stats**: Explore speed, area access, weather resistance
- **Collection stats**: Storage space, preserve duration, collection bonus

## License

[MIT License](LICENSE)

## Fish Characteristics

Each generated fish has:

- **Name**: A unique name for the species based on the data source
- **Rarity**: Common, Uncommon, Rare, Epic, or Legendary
- **Size**: Physical size in meters
- **Value**: Market value in USD
- **Description**: A short description of the fish's appearance and traits
- **Effect**: A gameplay effect that relates to the real-world data
- **Data Source**: Which data source influenced this fish's creation
- **StatEffects**: Specific gameplay statistics affected by this fish

## AI-Powered Fish Generation

When enabled, the application uses Google's Gemini API to create more creative and unique fish based on news headlines. The AI analyzes the news content, sentiment, and category to generate fish with characteristics that reflect the news story in interesting ways.

To use AI-powered fish generation:

1. Obtain a Google Gemini API key from the [Google AI Studio](https://makersuite.google.com/)
2. Set up your .env file with your API key and enable AI generation:
   ```
   GEMINI_API_KEY=your_api_key_here
   USE_AI=true
   ```

The AI-generated fish are marked with "news-ai" as their data source and with the 🤖 emoji in reports.

## Example Fish

```json
{
  "name": "Thunderfin Shockray",
  "rarity": "Epic",
  "size": 2.5,
  "value": 1000,
  "description": "A rare fish found only in stormy weather conditions. It has the ability to generate electricity during thunderstorms.",
  "effect": "Increases the player's chance of catching other rare fish during storms.",
  "data_source": "weather",
  "stat_effects": [
    {
      "stat": "CatchChance",
      "value": 15,
      "is_percentage": true,
      "duration": 600
    },
    {
      "stat": "WeatherResist",
      "value": 20,
      "is_percentage": true,
      "duration": 1200
    }
  ]
}
```

## Project Structure

- `cmd/fish-generate`: Main application code
- `internal/fish`: Fish model and generation logic
- `internal/data`: Data collection interfaces and implementations
- `internal/config`: Configuration and environment handling
- `internal/storage`: Database storage implementations

## Use in Games

The generated fish can be used in fishing games where:

- Players catch fish based on current real-world conditions
- Each fish provides unique effects or bonuses
- The rarity and value of available fish changes based on external factors

This makes the game more dynamic and connected to real-world events!

## API Endpoints

The application exposes several RESTful API endpoints for fishing simulation:

### `/api/fish`

**Method**: GET

**Description**: Attempt to catch a fish based on provided parameters

**Parameters**:
- `region_id` (optional): Specific region ID to fish in
- `location` (optional): Location name (city, ocean, etc.)
- `lat`, `lng` (optional): Coordinates for location-based fishing
- `weather` (optional): Current weather condition
- `temp` (optional): Current temperature
- `skill` (optional): User's fishing skill level (1-100)
- `bait` (optional): Type of bait used
- `time` (optional): Time of day ("morning", "afternoon", "evening", "night")

**Response Example**:
```json
{
  "success": true,
  "fish": {
    "name": "Solarbeam Goldscale",
    "description": "A bright golden fish that absorbs sunlight through its scales.",
    "rarity": "Rare",
    "size": 1.2,
    "value": 450,
    "effect": "Increases fishing luck by 10% for 30 minutes",
    "data_source": "weather"
  },
  "message": "You caught a magnificent Rare fish!",
  "rarity_factor": 0.75,
  "conditions": {
    "weather": "sunny",
    "region": "Pacific Coast",
    "region_tags": ["saltwater", "coastal"],
    "time_of_day": "afternoon",
    "quality": 7
  },
  "catch_time": "2023-06-15T14:23:45Z"
}
```

### `/api/regions`

**Method**: GET

**Description**: Get a list of available fishing regions

**Response Example**:
```json
[
  {
    "id": "pacific_coast",
    "name": "Pacific Coast",
    "description": "The western coastline with diverse marine life",
    "tags": ["saltwater", "coastal", "cold"],
    "fish_density": 8
  },
  {
    "id": "mountain_lake",
    "name": "Mountain Lake",
    "description": "A pristine lake surrounded by mountains",
    "tags": ["freshwater", "alpine", "clear"],
    "fish_density": 6
  }
]
```

### `/api/conditions`

**Method**: GET

**Description**: Get current fishing conditions for a location

**Parameters**:
- `region_id` (optional): Region ID to get conditions for
- `location` (optional): Location name

**Response Example**:
```json
{
  "weather": "partly cloudy",
  "temperature": 22.5,
  "wind_speed": 8.3,
  "time_of_day": "afternoon",
  "moon_phase": "full moon",
  "quality": 7,
  "region": "Pacific Coast",
  "best_baits": ["silver spinner", "crab lure"]
}
```

### Health Check

**Endpoint**: `/health`

**Method**: GET

**Description**: Check if the API is running

**Response Example**:
```json
{
  "status": "ok"
}
```

# Fish Generate Service

This service generates unique fish based on real-world data from multiple sources including news, weather, Bitcoin, and gold prices.

## Development Setup

### Prerequisites
- Go 1.20+
- MongoDB
- Required API keys (see Configuration)

### Local Setup
1. Clone the repository
2. Create a `.env` file based on `.env.example` and fill in your API keys
3. Build the application:
   ```
   go build -o fish-generate cmd/fish-generate/main.go
   ```
4. Run the application:
   ```
   ./fish-generate
   ```

## Staging Deployment

### Prerequisites
- Docker and Docker Compose
- Required API keys (see Configuration)

### Deployment Steps
1. Clone the repository on your staging server
2. Create a `.env` file based on `.env.example` and fill in your API keys
3. Deploy using Docker Compose:
   ```
   docker-compose up -d
   ```
4. Check the logs to verify the application is running:
   ```
   docker-compose logs -f app
   ```

### Stopping the Service
```
docker-compose down
```

### Restarting the Service
```
docker-compose restart
```

## Configuration

The service requires several API keys to function properly:

- `WEATHER_API_KEY`: For weather data collection
- `NEWS_API_KEY`: For news data collection
- `METAL_PRICE_API_KEY`: For gold price data collection
- `GEMINI_API_KEY`: For AI-based fish generation

Additional configuration can be done through environment variables in the docker-compose.yml file:

- `COLLECTION_INTERVAL_WEATHER`: Interval for weather data collection (default: 1 hour)
- `COLLECTION_INTERVAL_PRICE`: Interval for price data collection (default: 6 hours)
- `COLLECTION_INTERVAL_NEWS`: Interval for news data collection (default: 12 hours)
- `GENERATION_COOLDOWN`: Cooldown period between fish generations (default: 15 minutes)

## MongoDB Configuration

By default, the MongoDB service runs without authentication. To enable authentication:

1. Uncomment the MongoDB username and password environment variables in both `.env` and `docker-compose.yml`
2. Update the MongoDB connection string in `docker-compose.yml` to include authentication:
   ```
   MONGODB_URI=mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@mongodb:27017/fish_generator
   ```

## Data Persistence

MongoDB data is persisted using Docker volumes. The volume is named `mongodb_data` and will survive container restarts and removals 