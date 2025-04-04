package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	apiService "fish-generate/internal/api/service"
)

// FishingHandler handles API requests related to fishing
type FishingHandler struct {
	fishingService *apiService.FishingService
}

// NewFishingHandler creates a new fishing handler
func NewFishingHandler(fishingService *apiService.FishingService) *FishingHandler {
	return &FishingHandler{
		fishingService: fishingService,
	}
}

// CatchFish handles fishing attempts
func (h *FishingHandler) CatchFish(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	params := parseParams(r)

	// Call the service to attempt a catch
	result, err := h.fishingService.CatchFish(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to process fishing request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the result as JSON
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetRegions returns a list of available fishing regions
func (h *FishingHandler) GetRegions(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get regions from data package
	regions := getRegionsWithDetails()

	// Return the regions as JSON
	if err := json.NewEncoder(w).Encode(regions); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetCurrentConditions returns current weather and conditions for a location
func (h *FishingHandler) GetCurrentConditions(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse location parameters
	location := r.URL.Query().Get("location")
	regionID := r.URL.Query().Get("region_id")

	if location == "" && regionID == "" {
		http.Error(w, "Location or region_id parameter is required", http.StatusBadRequest)
		return
	}

	// Get current conditions (simplified - would connect to weather API in production)
	conditions := getCurrentConditions(location, regionID)

	// Return the conditions as JSON
	if err := json.NewEncoder(w).Encode(conditions); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Helper function to parse fishing parameters from request
func parseParams(r *http.Request) apiService.FishingParams {
	query := r.URL.Query()

	// Extract location info
	regionID := query.Get("region_id")
	location := query.Get("location")

	// Extract coordinates if provided
	var coordinates []float64
	if lat, err := strconv.ParseFloat(query.Get("lat"), 64); err == nil {
		if lng, err := strconv.ParseFloat(query.Get("lng"), 64); err == nil {
			coordinates = []float64{lat, lng}
		}
	}

	// Extract weather condition and temperature
	weatherCondition := query.Get("weather")
	if weatherCondition == "" {
		weatherCondition = "clear" // Default
	}

	temperature := 20.0 // Default to 20°C
	if tempStr := query.Get("temp"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			temperature = temp
		}
	}

	// Extract user's fishing skill
	fishingSkill := 50 // Default to 50 (average)
	if skillStr := query.Get("skill"); skillStr != "" {
		if skill, err := strconv.Atoi(skillStr); err == nil {
			if skill >= 1 && skill <= 100 {
				fishingSkill = skill
			}
		}
	}

	// Extract bait type
	baitType := query.Get("bait")

	// Determine time of day
	timeOfDay := getTimeOfDay()
	if todStr := query.Get("time_of_day"); todStr != "" {
		timeOfDay = todStr
	}

	return apiService.FishingParams{
		RegionID:         regionID,
		Location:         location,
		Coordinates:      coordinates,
		WeatherCondition: weatherCondition,
		Temperature:      temperature,
		FishingSkill:     fishingSkill,
		BaitType:         baitType,
		TimeOfDay:        timeOfDay,
	}
}

// getTimeOfDay returns the current time of day
func getTimeOfDay() string {
	hour := time.Now().Hour()

	if hour >= 5 && hour < 12 {
		return "morning"
	} else if hour >= 12 && hour < 17 {
		return "afternoon"
	} else if hour >= 17 && hour < 21 {
		return "evening"
	} else {
		return "night"
	}
}

// getRegionsWithDetails returns fishing regions with detailed information
func getRegionsWithDetails() interface{} {
	// In a full implementation, this would come from a database or a more detailed source
	// For now, we'll return a simplified version
	return map[string]interface{}{
		"regions": []map[string]interface{}{
			{
				"id":          "north_atlantic",
				"name":        "North Atlantic",
				"description": "Cold, deep waters with diverse marine life.",
				"difficulty":  "Medium",
				"fish_types":  []string{"Cod", "Halibut", "Mackerel", "Haddock"},
				"climate":     "Temperate to Cold",
			},
			{
				"id":          "tropical_pacific",
				"name":        "Tropical Pacific",
				"description": "Warm, clear waters with vibrant coral reefs.",
				"difficulty":  "Easy",
				"fish_types":  []string{"Tuna", "Mahi-mahi", "Reef Fish", "Flying Fish"},
				"climate":     "Tropical",
			},
			{
				"id":          "mediterranean",
				"name":        "Mediterranean Sea",
				"description": "Warm, saltier waters with rich biodiversity.",
				"difficulty":  "Easy",
				"fish_types":  []string{"Sea Bass", "Bream", "Mullet", "Sardines"},
				"climate":     "Mediterranean",
			},
			{
				"id":          "arctic_ocean",
				"name":        "Arctic Ocean",
				"description": "Extremely cold waters with unique ice-adapted species.",
				"difficulty":  "Hard",
				"fish_types":  []string{"Arctic Char", "Greenland Halibut", "Polar Cod"},
				"climate":     "Polar",
			},
			{
				"id":          "south_pacific",
				"name":        "South Pacific",
				"description": "Pristine waters with diverse island ecosystems.",
				"difficulty":  "Medium",
				"fish_types":  []string{"Marlin", "Sailfish", "Snapper", "Grouper"},
				"climate":     "Tropical to Temperate",
			},
		},
	}
}

// getCurrentConditions returns current fishing conditions for a location
func getCurrentConditions(location, regionID string) interface{} {
	// In a real implementation, this would fetch actual weather data
	// For now, we'll return mock data
	return map[string]interface{}{
		"location":          location,
		"region_id":         regionID,
		"weather":           "partly cloudy",
		"temperature":       22.5,
		"wind_speed":        10.2,
		"humidity":          65,
		"fishing_quality":   "Good",
		"suggested_baits":   []string{"Worms", "Small Fish", "Shrimp"},
		"active_fish_types": []string{"Bass", "Trout", "Catfish"},
		"time_of_day":       getTimeOfDay(),
		"timestamp":         time.Now(),
	}
}
