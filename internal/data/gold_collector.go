package data

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// MetalPriceAPIResponse represents the updated API response format from Metal Price API
type MetalPriceAPIResponse struct {
	Success bool `json:"success"`
	Query   struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Quote     float64 `json:"quote"`
		Timestamp int64   `json:"timestamp"`
	} `json:"info"`
	Result float64 `json:"result"`
}

// GoldPrice represents gold price data
type GoldPrice struct {
	PriceUSD  float64 `json:"price_usd"`
	Change24h float64 `json:"change_24h"` // 24-hour price change percentage
}

// GoldCollector collects real gold price data
type GoldCollector struct {
	apiKey             string
	lastPrice          float64
	client             *http.Client
	lastCollectionTime time.Time // Track when we last called the API
	yesterdayPrice     float64   // Store yesterday's price for change calculation
}

// NewGoldCollector creates a new gold price data collector
func NewGoldCollector(apiKey string) *GoldCollector {
	return &GoldCollector{
		apiKey:             apiKey,
		lastPrice:          0, // Will be set on first collection
		yesterdayPrice:     0, // Will be set after first collection
		client:             &http.Client{Timeout: 10 * time.Second},
		lastCollectionTime: time.Time{}, // Zero time initially
	}
}

// Collect retrieves real gold price data
func (c *GoldCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Check if we have an API key
	if c.apiKey != "" {
		// Check if we should call the API (once per day)
		currentTime := time.Now()
		shouldCallAPI := c.lastCollectionTime.IsZero() || // First time collecting
			currentTime.Sub(c.lastCollectionTime) >= 24*time.Hour // At least 24 hours since last collection

		if shouldCallAPI {
			log.Printf("Calling gold price API (last call: %v)", c.lastCollectionTime)

			// Save current price as yesterday's price before getting new price
			if c.lastPrice > 0 {
				c.yesterdayPrice = c.lastPrice
			}

			// New API endpoint
			url := "https://api.metalpriceapi.com/v1/convert?from=XAU&to=USD&amount=1"

			// Create HTTP request
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				log.Printf("Error creating gold API request: %v", err)
			} else {
				// Add required headers
				req.Header.Add("X-API-KEY", c.apiKey)
				req.Header.Add("Content-Type", "application/json")

				// Make the request
				resp, err := c.client.Do(req)
				if err != nil {
					log.Printf("Error making gold API request: %v", err)
				} else {
					defer resp.Body.Close()

					// Check if the response was successful
					if resp.StatusCode != http.StatusOK {
						log.Printf("Gold API returned status code %d", resp.StatusCode)
					} else {
						// Parse the new response format
						var apiResponse MetalPriceAPIResponse
						if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
							log.Printf("Error decoding gold API response: %v", err)
						} else if !apiResponse.Success {
							log.Printf("Gold API returned success=false")
						} else {
							// API call succeeded, use real data
							goldPricePerOunce := apiResponse.Result

							// Calculate change percentage based on yesterday's price
							changePercentage := 0.0
							if c.yesterdayPrice > 0 {
								changePercentage = ((goldPricePerOunce - c.yesterdayPrice) / c.yesterdayPrice) * 100.0
							}

							// Update last collection time
							c.lastCollectionTime = currentTime

							// Save the current price for future reference
							c.lastPrice = goldPricePerOunce

							goldData := &GoldPrice{
								PriceUSD:  goldPricePerOunce,
								Change24h: changePercentage,
							}

							return &DataEvent{
								Type:      GoldData,
								Value:     goldData,
								Timestamp: time.Now(),
								Source:    "metalpriceapi",
								Raw: map[string]interface{}{
									"price":           goldPricePerOunce,
									"previous_price":  c.yesterdayPrice,
									"change":          changePercentage,
									"last_collection": c.lastCollectionTime,
								},
							}, nil
						}
					}
				}
			}
		} else {
			log.Printf("Using cached gold price data (next API call in %v)",
				24*time.Hour-currentTime.Sub(c.lastCollectionTime))

			// Use last price we collected from the API
			goldData := &GoldPrice{
				PriceUSD:  c.lastPrice,
				Change24h: ((c.lastPrice - c.yesterdayPrice) / c.yesterdayPrice) * 100.0,
			}

			return &DataEvent{
				Type:      GoldData,
				Value:     goldData,
				Timestamp: time.Now(),
				Source:    "metalpriceapi-cached",
				Raw: map[string]interface{}{
					"price":           c.lastPrice,
					"previous_price":  c.yesterdayPrice,
					"change":          goldData.Change24h,
					"last_collection": c.lastCollectionTime,
					"cached":          true,
				},
			}, nil
		}
	} else {
		log.Println("No Gold API key provided, using mock data")
	}

	// Fall back to mock data
	return c.CollectMockData()
}

// CollectMockData generates mock gold price data when API is not available
func (c *GoldCollector) CollectMockData() (*DataEvent, error) {
	// If no previous price, start with a reasonable value
	if c.lastPrice == 0 {
		c.lastPrice = 1800.0 // Typical gold price per ounce in USD
	}

	// Generate a small change (-0.5% to +0.5%)
	changePercent := -0.5 + (rand.Float64() * 1.0)

	// Apply the change to the price
	newPrice := c.lastPrice * (1.0 + (changePercent / 100.0))

	// Keep the price within realistic bounds
	newPrice = math.Max(1600.0, math.Min(2200.0, newPrice))

	// Save the previous price
	previousPrice := c.lastPrice
	c.lastPrice = newPrice

	goldData := &GoldPrice{
		PriceUSD:  newPrice,
		Change24h: changePercent,
	}

	return &DataEvent{
		Type:      GoldData,
		Value:     goldData,
		Timestamp: time.Now(),
		Source:    "mock-gold-api",
		Raw: map[string]interface{}{
			"price":          newPrice,
			"previous_price": previousPrice,
			"change":         changePercent,
		},
	}, nil
}

// GetType returns the type of data collected
func (c *GoldCollector) GetType() DataType {
	return GoldData
}

// Start begins periodic collection of gold price data
func (c *GoldCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	log.Printf("Starting Gold Price collector with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	event, err := c.Collect(ctx)
	if err == nil {
		eventCh <- event
	} else {
		log.Printf("Error collecting gold price data: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			} else {
				log.Printf("Error collecting gold price data: %v", err)
			}
		case <-ctx.Done():
			log.Println("Gold price collector stopped")
			return
		}
	}
}
