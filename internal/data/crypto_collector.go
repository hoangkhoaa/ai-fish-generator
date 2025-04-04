package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CoinGeckoResponse represents the API response from CoinGecko
type CoinGeckoResponse struct {
	ID                                 string  `json:"id"`
	Symbol                             string  `json:"symbol"`
	Name                               string  `json:"name"`
	CurrentPrice                       float64 `json:"current_price"`
	MarketCap                          float64 `json:"market_cap"`
	PriceChangePercentage24h           float64 `json:"price_change_percentage_24h"`
	PriceChangePercentage7d            float64 `json:"price_change_percentage_7d"`
	TotalVolume                        float64 `json:"total_volume"`
	CirculatingSupply                  float64 `json:"circulating_supply"`
	PriceChangePercentage24hInCurrency float64 `json:"price_change_percentage_24h_in_currency"`
}

// CryptoCollector collects real cryptocurrency data from CoinGecko
type CryptoCollector struct {
	client  *http.Client
	coinIDs []string // List of coin IDs to collect data for
}

// NewCryptoCollector creates a new cryptocurrency data collector using the CoinGecko API
func NewCryptoCollector() *CryptoCollector {
	return &CryptoCollector{
		client:  &http.Client{Timeout: 10 * time.Second},
		coinIDs: []string{"bitcoin", "ethereum", "ripple", "litecoin", "cardano"},
	}
}

// Collect retrieves real cryptocurrency data from CoinGecko API
func (c *CryptoCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// We'll focus on Bitcoin for this implementation
	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=bitcoin&order=market_cap_desc&per_page=1&page=1&sparkline=false&price_change_percentage=24h%2C7d"

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add API headers
	req.Header.Add("Accept", "application/json")

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
	var coinGeckoResponse []CoinGeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&coinGeckoResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Check if we got any data
	if len(coinGeckoResponse) == 0 {
		return nil, fmt.Errorf("no cryptocurrency data returned")
	}

	btcData := &CryptoPrice{
		Symbol:    coinGeckoResponse[0].Symbol,
		PriceUSD:  coinGeckoResponse[0].CurrentPrice,
		Change24h: coinGeckoResponse[0].PriceChangePercentage24h,
		Volume24h: coinGeckoResponse[0].TotalVolume,
	}

	return &DataEvent{
		Type:      BitcoinData,
		Value:     btcData,
		Timestamp: time.Now(),
		Source:    "coingecko-api",
		Raw:       coinGeckoResponse[0],
	}, nil
}

// GetType returns the type of data collected
func (c *CryptoCollector) GetType() DataType {
	return BitcoinData
}

// Start begins periodic collection of cryptocurrency data
func (c *CryptoCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	log.Printf("Starting CoinGecko collector with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	event, err := c.Collect(ctx)
	if err == nil {
		eventCh <- event
	} else {
		log.Printf("Error collecting cryptocurrency data: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			} else {
				log.Printf("Error collecting cryptocurrency data: %v", err)
			}
		case <-ctx.Done():
			log.Println("Cryptocurrency collector stopped")
			return
		}
	}
}
