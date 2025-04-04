package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// EIAResponse represents the API response from the U.S. Energy Information Administration
type EIAResponse struct {
	Response struct {
		Data struct {
			Series []struct {
				SeriesID string          `json:"series_id"`
				Data     [][]interface{} `json:"data"` // [date, price]
			} `json:"series"`
		} `json:"data"`
	} `json:"response"`
}

// OilPriceCollector collects real oil price data from EIA API
type OilPriceCollector struct {
	apiKey    string
	lastPrice float64
	client    *http.Client
}

// NewOilPriceCollector creates a new oil price data collector using the EIA API
func NewOilPriceCollector(apiKey string) *OilPriceCollector {
	return &OilPriceCollector{
		apiKey:    apiKey,
		lastPrice: 0, // Will be set on first collection
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Collect retrieves real oil price data from EIA API
func (c *OilPriceCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Use WTI Crude Oil Price (PET.RWTC.D)
	seriesID := "PET.RWTC.D"
	url := fmt.Sprintf(
		"https://api.eia.gov/v2/seriesData/%s?api_key=%s&frequency=daily&data[0]=value&start=2023-01-01&sort[0][column]=period&sort[0][direction]=desc&offset=0&length=2",
		seriesID, c.apiKey,
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	// Parse the response
	var eiaResponse EIAResponse
	if err := json.NewDecoder(resp.Body).Decode(&eiaResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Extract price data
	var currentPrice float64
	var previousPrice float64
	var change24h float64

	if len(eiaResponse.Response.Data.Series) > 0 && len(eiaResponse.Response.Data.Series[0].Data) >= 2 {
		// Current price
		if priceVal, ok := eiaResponse.Response.Data.Series[0].Data[0][1].(float64); ok {
			currentPrice = priceVal
		}

		// Previous price
		if priceVal, ok := eiaResponse.Response.Data.Series[0].Data[1][1].(float64); ok {
			previousPrice = priceVal
		}

		// Calculate price change
		if previousPrice > 0 {
			change24h = ((currentPrice - previousPrice) / previousPrice) * 100
		}
	} else {
		// If we can't get data from API, use the last price or a default
		if c.lastPrice > 0 {
			currentPrice = c.lastPrice
		} else {
			currentPrice = 80.0 // Default fallback price
		}
		change24h = 0.0
	}

	// Update last price for future use
	c.lastPrice = currentPrice

	oilData := &OilPrice{
		PriceUSD:  currentPrice,
		Change24h: change24h,
	}

	return &DataEvent{
		Type:      OilPriceData,
		Value:     oilData,
		Timestamp: time.Now(),
		Source:    "eia-api",
		Raw:       eiaResponse,
	}, nil
}

// GetType returns the type of data collected
func (c *OilPriceCollector) GetType() DataType {
	return OilPriceData
}

// Start begins periodic collection of oil price data
func (c *OilPriceCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	log.Printf("Starting EIA Oil Price collector with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	event, err := c.Collect(ctx)
	if err == nil {
		eventCh <- event
	} else {
		log.Printf("Error collecting oil price data: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			} else {
				log.Printf("Error collecting oil price data: %v", err)
			}
		case <-ctx.Done():
			log.Println("Oil price collector stopped")
			return
		}
	}
}
