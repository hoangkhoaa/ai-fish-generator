package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// OpenWeatherMapResponse represents the API response from OpenWeatherMap
type OpenWeatherMapResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	Name string `json:"name"`
}

// WeatherCollector collects real weather data from OpenWeatherMap
type WeatherCollector struct {
	apiKey      string
	cityIDs     []string // List of city IDs to cycle through
	currentCity int      // Index of the current city
	client      *http.Client
}

// NewWeatherCollector creates a new weather data collector using the OpenWeatherMap API
func NewWeatherCollector(apiKey string) *WeatherCollector {
	// List of major city IDs (you can expand or customize this list)
	cityIDs := []string{
		"5128581", // New York
		"2643743", // London
		"1850147", // Tokyo
		"2147714", // Sydney
		"2988507", // Paris
		"2950159", // Berlin
		"3451190", // Rio de Janeiro
	}

	return &WeatherCollector{
		apiKey:      apiKey,
		cityIDs:     cityIDs,
		currentCity: 0,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Collect retrieves real weather data from OpenWeatherMap API
func (c *WeatherCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Cycle through cities
	cityID := c.cityIDs[c.currentCity]
	c.currentCity = (c.currentCity + 1) % len(c.cityIDs)

	// Construct API URL
	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?id=%s&units=metric&appid=%s",
		cityID, c.apiKey,
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
	var owmResponse OpenWeatherMapResponse
	if err := json.NewDecoder(resp.Body).Decode(&owmResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Extract weather condition
	condition := "Clear"
	if len(owmResponse.Weather) > 0 {
		condition = owmResponse.Weather[0].Main
	}

	// Determine if the weather is extreme
	isExtreme := false
	if owmResponse.Main.Temp > 35 || owmResponse.Main.Temp < -10 ||
		owmResponse.Wind.Speed > 20 || condition == "Thunderstorm" ||
		condition == "Tornado" || condition == "Hurricane" {
		isExtreme = true
	}

	// Convert wind speed from m/s to km/h
	windKph := owmResponse.Wind.Speed * 3.6

	// Create WeatherInfo
	weatherInfo := &WeatherInfo{
		Condition: condition,
		Location:  owmResponse.Name,
		TempC:     owmResponse.Main.Temp,
		Humidity:  owmResponse.Main.Humidity,
		WindKph:   windKph,
		IsExtreme: isExtreme,
	}

	return &DataEvent{
		Type:      WeatherData,
		Value:     weatherInfo,
		Timestamp: time.Now(),
		Source:    "openweathermap-api",
		Raw:       owmResponse,
	}, nil
}

// GetType returns the type of data collected
func (c *WeatherCollector) GetType() DataType {
	return WeatherData
}

// Start begins periodic collection of weather data
func (c *WeatherCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	log.Printf("Starting OpenWeatherMap collector with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	event, err := c.Collect(ctx)
	if err == nil {
		eventCh <- event
	} else {
		log.Printf("Error collecting weather data: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			} else {
				log.Printf("Error collecting weather data: %v", err)
			}
		case <-ctx.Done():
			log.Println("Weather collector stopped")
			return
		}
	}
}
