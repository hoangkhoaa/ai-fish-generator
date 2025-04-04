package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"fish-generate/internal/data"
	"fish-generate/internal/fish"
	"fish-generate/internal/storage"
)

// FishingService handles the fishing mechanics and retrieval of fish
type FishingService struct {
	storage     storage.StorageAdapter
	dataManager *data.DataManager
}

// FishingParams contains parameters for a fishing request
type FishingParams struct {
	RegionID         string    // Optional region ID
	Location         string    // Location name (city, ocean, etc.)
	Coordinates      []float64 // [lat, lng]
	WeatherCondition string    // Current weather condition
	Temperature      float64   // Current temperature
	FishingSkill     int       // User's fishing skill level (1-100)
	BaitType         string    // Type of bait used
	TimeOfDay        string    // "morning", "afternoon", "evening", "night"
}

// CatchResult represents the result of a fishing attempt
type CatchResult struct {
	Success      bool        `json:"success"`
	Fish         *fish.Fish  `json:"fish,omitempty"`
	Message      string      `json:"message"`
	RarityFactor float64     `json:"rarity_factor"`
	Conditions   *Conditions `json:"conditions"`
	CatchTime    time.Time   `json:"catch_time"`
}

// Conditions represents the current fishing conditions
type Conditions struct {
	Weather    string   `json:"weather"`
	Region     string   `json:"region"`
	RegionTags []string `json:"region_tags"`
	TimeOfDay  string   `json:"time_of_day"`
	Quality    int      `json:"quality"` // 1-10 rating of fishing conditions
}

// NewFishingService creates a new fishing service
func NewFishingService(storage storage.StorageAdapter, dataManager *data.DataManager) *FishingService {
	return &FishingService{
		storage:     storage,
		dataManager: dataManager,
	}
}

// CatchFish simulates a fishing attempt and returns a fish if successful
func (s *FishingService) CatchFish(ctx context.Context, params FishingParams) (*CatchResult, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Determine region if not provided
	regionID := params.RegionID
	if regionID == "" {
		var err error
		regionID, err = s.findRegionByLocation(params.Location, params.Coordinates)
		if err != nil {
			return nil, fmt.Errorf("could not determine region: %v", err)
		}
	}

	// Calculate fishing conditions and chances
	conditions, rarityFactor := s.calculateFishingConditions(ctx, params, regionID)

	// Determine if the fishing attempt is successful
	successChance := s.calculateSuccessChance(params, conditions)
	success := rand.Float64() < successChance

	if !success {
		return &CatchResult{
			Success:      false,
			Message:      getFishingFailMessage(),
			RarityFactor: rarityFactor,
			Conditions:   conditions,
			CatchTime:    time.Now(),
		}, nil
	}

	// Determine which type of data source to use for fish selection
	// based on conditions (weather, time, etc.)
	dataSource := s.selectDataSource(params, conditions)

	// Get fish from the chosen data source matching the region
	fish, err := s.getFishByConditions(ctx, regionID, dataSource, rarityFactor)
	if err != nil {
		log.Printf("Error getting fish: %v, trying any fish", err)
		// Fallback to any fish if we couldn't find one matching specific conditions
		fish, err = s.getAnyFish(ctx, regionID)
		if err != nil {
			return nil, fmt.Errorf("failed to catch fish: %v", err)
		}
	}

	return &CatchResult{
		Success:      true,
		Fish:         fish,
		Message:      getSuccessCatchMessage(fish),
		RarityFactor: rarityFactor,
		Conditions:   conditions,
		CatchTime:    time.Now(),
	}, nil
}

// findRegionByLocation determines the best region match for a given location
func (s *FishingService) findRegionByLocation(location string, coordinates []float64) (string, error) {
	// Get all regions
	regions := data.PredefinedRegions()

	// Default region in case we can't find a match
	defaultRegion := regions[0].ID

	// If no location info, return default
	if location == "" && (coordinates == nil || len(coordinates) != 2) {
		return defaultRegion, nil
	}

	// If we have coordinates, find region by proximity
	if coordinates != nil && len(coordinates) == 2 {
		lat, lng := coordinates[0], coordinates[1]

		// Find closest region by checking if coordinates fall within region boundaries
		// This is a simple implementation - a real one would use proper geo calculations
		for _, region := range regions {
			// Check if region has location data
			if region.Location.Latitude != 0 || region.Location.Longitude != 0 {
				// Calculate rough distance (this is simplified)
				rLat := region.Location.Latitude
				rLng := region.Location.Longitude

				// Very simple distance check (not accurate, just for demonstration)
				if (lat-rLat)*(lat-rLat)+(lng-rLng)*(lng-rLng) < 100 {
					return region.ID, nil
				}
			}
		}
	}

	// If we have location name, try to match it to region names or tags
	if location != "" {
		location = strings.ToLower(location)

		// Check for region name match
		for _, region := range regions {
			if strings.Contains(strings.ToLower(region.Name), location) {
				return region.ID, nil
			}

			// Check for tags match
			for _, tag := range region.Tags {
				if strings.Contains(strings.ToLower(tag), location) {
					return region.ID, nil
				}
			}
		}
	}

	// If no match found, return default region
	return defaultRegion, nil
}

// calculateFishingConditions evaluates the current fishing conditions
func (s *FishingService) calculateFishingConditions(ctx context.Context, params FishingParams, regionID string) (*Conditions, float64) {
	// Get region info
	regions := data.PredefinedRegions()
	var regionName string
	var regionTags []string

	// Find region details
	for _, r := range regions {
		if r.ID == regionID {
			regionName = r.Name
			regionTags = r.Tags
			break
		}
	}

	// Calculate quality of fishing conditions (1-10)
	quality := 5 // Default average

	// Adjust based on weather
	weatherFactor := 0
	if isGoodWeatherForFishing(params.WeatherCondition) {
		weatherFactor = 2
	} else if isBadWeatherForFishing(params.WeatherCondition) {
		weatherFactor = -2
	}

	// Adjust based on time of day
	timeFactor := 0
	if params.TimeOfDay == "morning" || params.TimeOfDay == "evening" {
		timeFactor = 1 // Dawn and dusk are good for fishing
	} else if params.TimeOfDay == "night" {
		timeFactor = -1 // Night is typically harder
	}

	// Calculate final quality
	quality += weatherFactor + timeFactor
	if quality < 1 {
		quality = 1
	} else if quality > 10 {
		quality = 10
	}

	// Calculate rarity factor (0.0-1.0) - higher quality means more chance of rare fish
	rarityFactor := float64(quality) / 10.0

	return &Conditions{
		Weather:    params.WeatherCondition,
		Region:     regionName,
		RegionTags: regionTags,
		TimeOfDay:  params.TimeOfDay,
		Quality:    quality,
	}, rarityFactor
}

// calculateSuccessChance determines the chance of a successful catch
func (s *FishingService) calculateSuccessChance(params FishingParams, conditions *Conditions) float64 {
	// Base chance of success
	baseChance := 0.5 // 50% base chance

	// Adjust based on fishing skill (1-100)
	skillBonus := float64(params.FishingSkill) / 200.0 // Up to +0.5 (50%)

	// Adjust based on conditions quality
	conditionsBonus := float64(conditions.Quality-5) / 20.0 // -0.2 to +0.25

	// Adjust based on bait type
	baitBonus := 0.0
	if params.BaitType != "" {
		baitBonus = 0.1 // +10% for using any bait
	}

	// Calculate final chance
	finalChance := baseChance + skillBonus + conditionsBonus + baitBonus

	// Ensure chance is between 0.1 and 0.95
	if finalChance < 0.1 {
		finalChance = 0.1 // Always at least 10% chance to catch something
	}
	if finalChance > 0.95 {
		finalChance = 0.95 // Never 100% guaranteed
	}

	return finalChance
}

// selectDataSource determines which data source to use based on conditions
func (s *FishingService) selectDataSource(params FishingParams, conditions *Conditions) string {
	// Default
	dataSource := ""

	// Select data source by weather conditions
	if isExtremeWeather(params.WeatherCondition, params.Temperature) {
		dataSource = "weather" // Extreme weather affects fish type
	} else if conditions.Quality >= 8 {
		// For really good conditions, use more exciting sources
		sources := []string{"bitcoin", "news", "news-ai"}
		dataSource = sources[rand.Intn(len(sources))]
	} else {
		// For normal conditions, use a mix of sources weighted toward normal types
		sources := []string{"weather", "weather", "bitcoin", "oil", "news"}
		dataSource = sources[rand.Intn(len(sources))]
	}

	return dataSource
}

// getFishByConditions finds a suitable fish based on region, data source, and rarity
func (s *FishingService) getFishByConditions(ctx context.Context, regionID string, dataSource string, rarityFactor float64) (*fish.Fish, error) {
	// Determine rarity tier based on rarity factor
	rarityLevel := determineRarity(rarityFactor)

	// Try to get a fish by data source and rarity
	if dataSource != "" {
		fishList, err := s.storage.GetFishByDataSource(ctx, dataSource, 50)
		if err == nil && len(fishList) > 0 {
			// Filter by rarity
			matchingFish := filterFishByRarity(fishList, rarityLevel)
			if len(matchingFish) > 0 {
				// Return a random fish from the matching ones
				return matchingFish[rand.Intn(len(matchingFish))], nil
			}
		}
	}

	// Try to find similar fish by rarity
	fish, err := s.storage.GetSimilarFish(ctx, dataSource, rarityLevel)
	if err == nil && fish != nil {
		return fish, nil
	}

	// Finally, try getting fish by region (if it has been set in the database)
	fishList, err := s.storage.GetFishByRegion(ctx, regionID, 20)
	if err == nil && len(fishList) > 0 {
		// Return a random fish from the region
		return fishList[rand.Intn(len(fishList))], nil
	}

	return nil, fmt.Errorf("no suitable fish found")
}

// getAnyFish gets any available fish from the database
func (s *FishingService) getAnyFish(ctx context.Context, regionID string) (*fish.Fish, error) {
	// Try sources one by one
	sources := []string{"weather", "bitcoin", "oil", "news", "news-ai"}

	for _, source := range sources {
		fishList, err := s.storage.GetFishByDataSource(ctx, source, 10)
		if err == nil && len(fishList) > 0 {
			return fishList[rand.Intn(len(fishList))], nil
		}
	}

	return nil, fmt.Errorf("no fish available in the database")
}

// Helper functions

// isGoodWeatherForFishing determines if current weather is good for fishing
func isGoodWeatherForFishing(condition string) bool {
	goodConditions := []string{
		"partly cloudy", "cloudy", "overcast", "light rain", "drizzle", "mist",
	}

	condition = strings.ToLower(condition)
	for _, good := range goodConditions {
		if strings.Contains(condition, good) {
			return true
		}
	}

	return false
}

// isBadWeatherForFishing determines if current weather is bad for fishing
func isBadWeatherForFishing(condition string) bool {
	badConditions := []string{
		"thunderstorm", "heavy rain", "storm", "snow", "blizzard", "hurricane", "tornado",
	}

	condition = strings.ToLower(condition)
	for _, bad := range badConditions {
		if strings.Contains(condition, bad) {
			return true
		}
	}

	return false
}

// isExtremeWeather checks if weather is extreme
func isExtremeWeather(condition string, tempC float64) bool {
	// Check for extreme conditions
	extremeConditions := []string{
		"thunderstorm", "hurricane", "tornado", "blizzard", "storm",
	}

	condition = strings.ToLower(condition)
	for _, extreme := range extremeConditions {
		if strings.Contains(condition, extreme) {
			return true
		}
	}

	// Check for extreme temperatures
	if tempC > 35 || tempC < 0 {
		return true
	}

	return false
}

// determineRarity returns a rarity string based on rarity factor
func determineRarity(rarityFactor float64) string {
	// Adjust chance based on rarity factor (higher factor = better chance for rare)
	roll := rand.Float64()

	// Apply the rarity factor to skew the distribution
	roll = roll / (rarityFactor + 0.5)

	// Determine rarity tier based on roll
	if roll < 0.5 {
		return string(fish.Common)
	} else if roll < 0.75 {
		return string(fish.Uncommon)
	} else if roll < 0.9 {
		return string(fish.Rare)
	} else if roll < 0.98 {
		return string(fish.Epic)
	} else {
		return string(fish.Legendary)
	}
}

// filterFishByRarity returns only fish matching the given rarity
func filterFishByRarity(fishList []*fish.Fish, rarityLevel string) []*fish.Fish {
	var result []*fish.Fish

	for _, f := range fishList {
		if string(f.Rarity) == rarityLevel {
			result = append(result, f)
		}
	}

	return result
}

// getSuccessCatchMessage returns a message for a successful catch
func getSuccessCatchMessage(fish *fish.Fish) string {
	messages := []string{
		"You caught a %s!",
		"Success! You reeled in a %s!",
		"A %s took the bait!",
		"You've caught a %s. Nice catch!",
		"The %s couldn't resist your bait!",
	}

	// Add rarity-specific messages for rare and legendary fish
	if fish.Rarity == "Rare" || fish.Rarity == "Epic" || fish.Rarity == "Legendary" {
		messages = append(messages,
			"Incredible! You've caught a %s!",
			"An extraordinary %s has been caught!",
			"What a catch! You've landed a %s!",
		)
	}

	msg := messages[rand.Intn(len(messages))]
	return fmt.Sprintf(msg, fish.Name)
}

// getFishingFailMessage returns a message for an unsuccessful catch
func getFishingFailMessage() string {
	messages := []string{
		"The fish got away!",
		"Something nibbled at your bait, but you missed it.",
		"You feel a tug, but the fish escapes.",
		"Not even a bite. Try again?",
		"The waters seem quiet right now.",
		"No luck this time. Maybe try different bait?",
		"The fish aren't biting right now.",
	}

	return messages[rand.Intn(len(messages))]
}
