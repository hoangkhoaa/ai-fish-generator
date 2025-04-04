package data

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// MockWeatherCollector collects mock weather data
type MockWeatherCollector struct {
	locations  []string
	conditions []string
	rand       *rand.Rand
}

// NewMockWeatherCollector creates a new mock weather data collector
func NewMockWeatherCollector() *MockWeatherCollector {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return &MockWeatherCollector{
		locations:  []string{"New York", "London", "Tokyo", "Sydney", "Paris", "Berlin", "Rio de Janeiro"},
		conditions: []string{"Sunny", "Rainy", "Cloudy", "Stormy", "Windy", "Snowy", "Clear"},
		rand:       r,
	}
}

// Collect generates mock weather data
func (c *MockWeatherCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Create random weather data
	location := c.locations[c.rand.Intn(len(c.locations))]
	condition := c.conditions[c.rand.Intn(len(c.conditions))]
	tempC := 10.0 + c.rand.Float64()*25.0  // 10-35 degrees C
	humidity := 30 + c.rand.Intn(70)       // 30-100% humidity
	windKph := 5.0 + c.rand.Float64()*45.0 // 5-50 km/h

	// 10% chance of extreme weather
	isExtreme := c.rand.Float64() < 0.1

	weatherInfo := &WeatherInfo{
		Condition: condition,
		Location:  location,
		TempC:     tempC,
		Humidity:  humidity,
		WindKph:   windKph,
		IsExtreme: isExtreme,
	}

	return &DataEvent{
		Type:      WeatherData,
		Value:     weatherInfo,
		Timestamp: time.Now(),
		Source:    "mock-weather-api",
		Raw:       weatherInfo,
	}, nil
}

// GetType returns the type of data collected
func (c *MockWeatherCollector) GetType() DataType {
	return WeatherData
}

// Start begins periodic collection of weather data
func (c *MockWeatherCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			}
		case <-ctx.Done():
			return
		}
	}
}

// MockBitcoinCollector collects mock Bitcoin price data
type MockBitcoinCollector struct {
	currentPrice float64
	rand         *rand.Rand
}

// NewMockBitcoinCollector creates a new mock Bitcoin data collector
func NewMockBitcoinCollector() *MockBitcoinCollector {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return &MockBitcoinCollector{
		currentPrice: 35000.0 + r.Float64()*10000.0, // Start between 35000-45000 USD
		rand:         r,
	}
}

// Collect generates mock Bitcoin price data
func (c *MockBitcoinCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Generate random price change (-5% to +5%)
	change24h := -5.0 + c.rand.Float64()*10.0

	// 10% chance of significant price movement (-15% to +15%)
	if c.rand.Float64() < 0.1 {
		change24h = -15.0 + c.rand.Float64()*30.0
	}

	// Update the current price based on the change
	c.currentPrice = c.currentPrice * (1.0 + (change24h / 100.0))

	// Ensure price doesn't go below 20000 or above 100000
	c.currentPrice = math.Max(20000.0, math.Min(100000.0, c.currentPrice))

	// Calculate 24h volume (in billions)
	volume24h := 20.0 + c.rand.Float64()*40.0

	btcData := &CryptoPrice{
		Symbol:    "BTC",
		PriceUSD:  c.currentPrice,
		Change24h: change24h,
		Volume24h: volume24h,
	}

	return &DataEvent{
		Type:      BitcoinData,
		Value:     btcData,
		Timestamp: time.Now(),
		Source:    "mock-crypto-api",
		Raw:       btcData,
	}, nil
}

// GetType returns the type of data collected
func (c *MockBitcoinCollector) GetType() DataType {
	return BitcoinData
}

// Start begins periodic collection of Bitcoin data
func (c *MockBitcoinCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			}
		case <-ctx.Done():
			return
		}
	}
}

// MockOilPriceCollector collects mock oil price data
type MockOilPriceCollector struct {
	currentPrice float64
	rand         *rand.Rand
}

// NewMockOilPriceCollector creates a new mock oil price data collector
func NewMockOilPriceCollector() *MockOilPriceCollector {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return &MockOilPriceCollector{
		currentPrice: 60.0 + r.Float64()*40.0, // Start between 60-100 USD
		rand:         r,
	}
}

// Collect generates mock oil price data
func (c *MockOilPriceCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Generate random price change (-3% to +3%)
	change24h := -3.0 + c.rand.Float64()*6.0

	// 10% chance of significant price movement (-10% to +10%)
	if c.rand.Float64() < 0.1 {
		change24h = -10.0 + c.rand.Float64()*20.0
	}

	// Update the current price based on the change
	c.currentPrice = c.currentPrice * (1.0 + (change24h / 100.0))

	// Ensure price doesn't go below 30 or above 150
	c.currentPrice = math.Max(30.0, math.Min(150.0, c.currentPrice))

	oilData := &OilPrice{
		PriceUSD:  c.currentPrice,
		Change24h: change24h,
	}

	return &DataEvent{
		Type:      OilPriceData,
		Value:     oilData,
		Timestamp: time.Now(),
		Source:    "mock-oil-api",
		Raw:       oilData,
	}, nil
}

// GetType returns the type of data collected
func (c *MockOilPriceCollector) GetType() DataType {
	return OilPriceData
}

// Start begins periodic collection of oil price data
func (c *MockOilPriceCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			}
		case <-ctx.Done():
			return
		}
	}
}

// MockNewsCollector collects mock news data
type MockNewsCollector struct {
	categories []string
	sources    []string
	rand       *rand.Rand
}

// NewMockNewsCollector creates a new mock news data collector
func NewMockNewsCollector() *MockNewsCollector {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return &MockNewsCollector{
		categories: []string{"Politics", "Business", "Technology", "Entertainment", "Sports", "Health", "Science", "Economy", "Disaster"},
		sources:    []string{"CNN", "BBC", "Reuters", "Bloomberg", "TechCrunch", "Associated Press", "Wall Street Journal"},
		rand:       r,
	}
}

// generateHeadline creates a mock headline based on category
func (c *MockNewsCollector) generateHeadline(category string) string {
	headlines := map[string][]string{
		"Politics": {
			"New trade agreement signed between countries",
			"Government announces major policy change",
			"Election results shock political analysts",
			"Peace talks begin in troubled region",
			"Controversial law passed amid protests",
		},
		"Business": {
			"Major company announces quarterly results",
			"Stock market reaches record high",
			"New merger creates industry giant",
			"Startup receives billion-dollar valuation",
			"Economic indicators show strong growth",
		},
		"Technology": {
			"Tech giant reveals revolutionary new product",
			"Breakthrough in AI research announced",
			"Major security vulnerability discovered",
			"New social media platform gaining popularity",
			"Space exploration company achieves milestone",
		},
		"Entertainment": {
			"Blockbuster movie breaks box office records",
			"Celebrity announces surprise retirement",
			"Award show sparks controversy",
			"Streaming service gains millions of subscribers",
			"Long-awaited sequel finally announced",
		},
		"Sports": {
			"Underdog team wins championship",
			"Star athlete signs record-breaking contract",
			"Olympic committee announces new event",
			"Coach fired after disappointing season",
			"Player breaks long-standing record",
		},
		"Health": {
			"New treatment shows promising results",
			"Health officials issue important advisory",
			"Study reveals surprising health benefits",
			"Pandemic response enters new phase",
			"Medical breakthrough could save millions",
		},
		"Science": {
			"Scientists make groundbreaking discovery",
			"New species discovered in remote location",
			"Climate report shows concerning trends",
			"Space telescope captures stunning images",
			"Research team achieves quantum computing milestone",
		},
		"Economy": {
			"Central bank adjusts interest rates",
			"Unemployment reaches historic low",
			"Inflation concerns grow among economists",
			"Trade deficit widens unexpectedly",
			"Housing market shows signs of cooling",
		},
		"Disaster": {
			"Powerful earthquake strikes coastal region",
			"Hurricane approaches populated areas",
			"Wildfire forces thousands to evacuate",
			"Flooding causes widespread damage",
			"Industrial accident prompts environmental concerns",
		},
	}

	// Get headlines for the category, default to politics if not found
	options, ok := headlines[category]
	if !ok {
		options = headlines["Politics"]
	}

	return options[c.rand.Intn(len(options))]
}

// generateKeywords extracts keywords from headline based on category
func (c *MockNewsCollector) generateKeywords(headline string, category string) []string {
	words := []string{category}

	// Add some generic keywords from the headline
	if len(headline) > 0 {
		words = append(words, "news", "current events")
	}

	return words
}

// generateSentiment creates a sentiment score for the headline
func (c *MockNewsCollector) generateSentiment(category string, headline string) float64 {
	// Base sentiment slightly positive
	sentiment := 0.1

	// Disasters are usually negative
	if category == "Disaster" {
		sentiment = -0.5 - c.rand.Float64()*0.5 // -0.5 to -1.0
	}

	// Technology and Science news often positive
	if category == "Technology" || category == "Science" {
		sentiment = 0.3 + c.rand.Float64()*0.5 // 0.3 to 0.8
	}

	// Add some randomness
	sentiment += -0.3 + c.rand.Float64()*0.6 // -0.3 to +0.3

	// Clamp to range [-1, 1]
	return math.Max(-1.0, math.Min(1.0, sentiment))
}

// Collect generates mock news data
func (c *MockNewsCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Pick a random category and source
	category := c.categories[c.rand.Intn(len(c.categories))]
	source := c.sources[c.rand.Intn(len(c.sources))]

	// Generate headline and other data
	headline := c.generateHeadline(category)
	keywords := c.generateKeywords(headline, category)
	sentiment := c.generateSentiment(category, headline)

	// Create mock URL
	url := "https://www." + strings.ToLower(source) + ".com/news/" +
		strings.ToLower(strings.ReplaceAll(category, " ", "-")) + "/" +
		strconv.FormatInt(time.Now().Unix(), 10)

	newsItem := &NewsItem{
		Headline:    headline,
		Source:      source,
		URL:         url,
		Category:    category,
		Keywords:    keywords,
		PublishedAt: time.Now(),
		Sentiment:   sentiment,
	}

	return &DataEvent{
		Type:      NewsData,
		Value:     newsItem,
		Timestamp: time.Now(),
		Source:    "mock-news-api",
		Raw:       newsItem,
	}, nil
}

// GetType returns the type of data collected
func (c *MockNewsCollector) GetType() DataType {
	return NewsData
}

// Start begins periodic collection of news data
func (c *MockNewsCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			}
		case <-ctx.Done():
			return
		}
	}
}
