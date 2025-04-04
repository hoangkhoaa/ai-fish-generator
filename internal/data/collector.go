package data

import (
	"context"
	"time"
)

// DataType represents the type of data collected
type DataType string

const (
	WeatherData  DataType = "weather"
	BitcoinData  DataType = "bitcoin"
	OilPriceData DataType = "oil"
	NewsData     DataType = "news"
	GoldData     DataType = "gold"
)

// DataEvent represents a data event collected from an external source
type DataEvent struct {
	Type          DataType    `json:"type"`
	Value         interface{} `json:"value"`
	Timestamp     time.Time   `json:"timestamp"`
	Source        string      `json:"source"`
	Raw           interface{} `json:"raw,omitempty"`  // Raw data for reference
	GeneratedFish bool        `json:"generated_fish"` // Indicates if this event generated a fish
}

// WeatherInfo represents weather data for a specific location
type WeatherInfo struct {
	Condition string  `json:"condition"` // e.g., "Rainy", "Sunny", "Stormy"
	Location  string  `json:"location"`
	TempC     float64 `json:"temp_c"`
	Humidity  int     `json:"humidity"`
	WindKph   float64 `json:"wind_kph"`
	IsExtreme bool    `json:"is_extreme"` // Indicates extreme weather conditions
}

// GetCondition returns the weather condition
func (w *WeatherInfo) GetCondition() string {
	return w.Condition
}

// GetTempC returns the temperature in Celsius
func (w *WeatherInfo) GetTempC() float64 {
	return w.TempC
}

// GetHumidity returns the humidity as a float64
func (w *WeatherInfo) GetHumidity() float64 {
	return float64(w.Humidity)
}

// GetWindSpeed returns the wind speed in km/h
func (w *WeatherInfo) GetWindSpeed() float64 {
	return w.WindKph
}

// GetRainMM returns the rainfall in mm (not available in this implementation)
func (w *WeatherInfo) GetRainMM() float64 {
	return 0.0 // Not provided by our implementation
}

// GetPressure returns the atmospheric pressure (not available in this implementation)
func (w *WeatherInfo) GetPressure() float64 {
	return 0.0 // Not provided by our implementation
}

// GetClouds returns the cloud coverage percentage (not available in this implementation)
func (w *WeatherInfo) GetClouds() int {
	return 0 // Not provided by our implementation
}

// GetDescription returns a description of the weather
func (w *WeatherInfo) GetDescription() string {
	return w.Condition // Use the condition as the description
}

// GetSource returns the source of the weather data
func (w *WeatherInfo) GetSource() string {
	return "openweathermap-api"
}

// CryptoPrice represents cryptocurrency price data
type CryptoPrice struct {
	Symbol    string  `json:"symbol"`
	PriceUSD  float64 `json:"price_usd"`
	Change24h float64 `json:"change_24h"` // 24-hour price change percentage
	Volume24h float64 `json:"volume_24h"`
}

// OilPrice represents oil price data
type OilPrice struct {
	PriceUSD  float64 `json:"price_usd"`
	Change24h float64 `json:"change_24h"` // 24-hour price change percentage
}

// NewsItem represents a news headline or article
type NewsItem struct {
	Headline    string    `json:"headline"`
	Content     string    `json:"content"` // News article content
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Category    string    `json:"category"`
	Keywords    []string  `json:"keywords"`
	PublishedAt time.Time `json:"published_at"`
	Sentiment   float64   `json:"sentiment"` // -1.0 to 1.0 (negative to positive)
}

// GetHeadline returns the news headline
func (n *NewsItem) GetHeadline() string {
	return n.Headline
}

// GetContent returns the news content
func (n *NewsItem) GetContent() string {
	return n.Content // Return actual content field value
}

// GetSource returns the news source
func (n *NewsItem) GetSource() string {
	return n.Source
}

// GetURL returns the news URL
func (n *NewsItem) GetURL() string {
	return n.URL
}

// GetPublishedAt returns the news publication date
func (n *NewsItem) GetPublishedAt() time.Time {
	return n.PublishedAt
}

// GetSentiment returns the news sentiment
func (n *NewsItem) GetSentiment() float64 {
	return n.Sentiment
}

// GetKeywords returns the news keywords
func (n *NewsItem) GetKeywords() []string {
	return n.Keywords
}

// DataCollector interface defines methods for collecting data from external sources
type DataCollector interface {
	// Collect retrieves data from the source
	Collect(ctx context.Context) (*DataEvent, error)

	// GetType returns the type of data this collector provides
	GetType() DataType

	// Start begins the periodic data collection, sending events to the provided channel
	Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent)
}
