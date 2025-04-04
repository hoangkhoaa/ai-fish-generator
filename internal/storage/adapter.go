package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fish-generate/internal/data"
	"fish-generate/internal/fish"
)

// MongoDBAdapter adapts MongoDB to DatabaseClient interface
type MongoDBAdapter struct {
	db *MongoDB
}

// NewMongoDBAdapter creates a new adapter for MongoDB
func NewMongoDBAdapter(db *MongoDB) *MongoDBAdapter {
	return &MongoDBAdapter{
		db: db,
	}
}

// SaveWeatherData saves weather data to MongoDB
func (a *MongoDBAdapter) SaveWeatherData(ctx context.Context, weatherInfo *data.WeatherInfo, regionID, cityID string) error {
	return a.db.SaveWeatherData(ctx, weatherInfo, regionID, cityID)
}

// SavePriceData saves price data to MongoDB
func (a *MongoDBAdapter) SavePriceData(ctx context.Context, assetType string, price, volume, changePercent, volumeChange float64, source string) error {
	return a.db.SavePriceData(ctx, assetType, price, volume, changePercent, volumeChange, source)
}

// SaveNewsData saves news data to MongoDB
func (a *MongoDBAdapter) SaveNewsData(ctx context.Context, newsItem *data.NewsItem) error {
	return a.db.SaveNewsData(ctx, newsItem)
}

// SaveFishData saves fish data to MongoDB
func (a *MongoDBAdapter) SaveFishData(ctx context.Context, fishItem interface{}) error {
	// Try to convert from fish.Fish type
	if fishObj, ok := fishItem.(*fish.Fish); ok {
		// Convert fish to FishData
		fishData := &FishData{
			Name:             fishObj.Name,
			Description:      fishObj.Description,
			Rarity:           string(fishObj.Rarity),
			Length:           fishObj.Size,
			Weight:           fishObj.Size * 2.5, // Approximation
			Color:            extractColorFromDescription(fishObj.Description),
			Habitat:          extractHabitatFromDescription(fishObj.Description),
			Diet:             extractDietFromDescription(fishObj.Description),
			GeneratedAt:      time.Now(),
			IsAIGenerated:    fishObj.IsAIGenerated,
			DataSource:       fishObj.DataSource,
			StatEffects:      convertStatEffects(fishObj.StatEffects),
			GenerationReason: fishObj.GenerationReason,
		}

		return a.db.SaveFishData(ctx, fishData)
	}

	// If it's already a map, pass it directly
	if fishMap, ok := fishItem.(map[string]interface{}); ok {
		// Try to extract color, habitat, and diet from description if present
		if desc, hasDesc := fishMap["description"].(string); hasDesc {
			if _, hasColor := fishMap["color"]; !hasColor {
				fishMap["color"] = extractColorFromDescription(desc)
			}
			if _, hasHabitat := fishMap["habitat"]; !hasHabitat {
				fishMap["habitat"] = extractHabitatFromDescription(desc)
			}
			if _, hasDiet := fishMap["diet"]; !hasDiet {
				fishMap["diet"] = extractDietFromDescription(desc)
			}
		}

		// Set generation time if not already set
		if _, hasTime := fishMap["generated_at"]; !hasTime {
			fishMap["generated_at"] = time.Now()
		}

		return a.db.SaveFishData(ctx, fishMap)
	}

	// For any other type, pass it to the DB which will handle conversion
	return a.db.SaveFishData(ctx, fishItem)
}

// GetDailyFishCount retrieves the number of fish generated today
func (a *MongoDBAdapter) GetDailyFishCount(ctx context.Context) (int, error) {
	return a.db.GetDailyFishCount(ctx)
}

// GetSimilarFish retrieves a similar fish from the database
func (a *MongoDBAdapter) GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*fish.Fish, error) {
	fishData, err := a.db.GetSimilarFish(ctx, dataSource, rarityLevel)
	if err != nil {
		return nil, err
	}

	// Convert to fish.Fish
	return convertToFish(fishData), nil
}

// GetFishByRegion retrieves fish for a specific region
func (a *MongoDBAdapter) GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*fish.Fish, error) {
	fishDataList, err := a.db.GetFishByRegion(ctx, regionID, limit)
	if err != nil {
		return nil, err
	}

	// Convert to []*fish.Fish
	result := make([]*fish.Fish, len(fishDataList))
	for i, fishData := range fishDataList {
		result[i] = convertToFish(fishData)
	}

	return result, nil
}

// GetFishByDataSource retrieves fish from a specific data source
func (a *MongoDBAdapter) GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*fish.Fish, error) {
	fishDataList, err := a.db.GetFishByDataSource(ctx, dataSource, limit)
	if err != nil {
		return nil, err
	}

	// Convert to []*fish.Fish
	result := make([]*fish.Fish, len(fishDataList))
	for i, fishData := range fishDataList {
		result[i] = convertToFish(fishData)
	}

	return result, nil
}

// GetRecentWeatherData retrieves recent weather data for a specific region
func (a *MongoDBAdapter) GetRecentWeatherData(ctx context.Context, regionID string, limit int) ([]*data.WeatherInfo, error) {
	mongoData, err := a.db.GetRecentWeatherData(ctx, regionID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*data.WeatherInfo, len(mongoData))
	for i, data := range mongoData {
		result[i] = convertToWeatherInfo(data)
	}

	return result, nil
}

// GetRecentPriceData retrieves recent price data for a specific asset type
func (a *MongoDBAdapter) GetRecentPriceData(ctx context.Context, assetType string, limit int) ([]map[string]interface{}, error) {
	return a.db.GetRecentPriceData(ctx, assetType, limit)
}

// GetRecentNewsData retrieves recent news data
func (a *MongoDBAdapter) GetRecentNewsData(ctx context.Context, limit int) ([]*data.NewsItem, error) {
	mongoData, err := a.db.GetRecentNewsData(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*data.NewsItem, len(mongoData))
	for i, data := range mongoData {
		result[i] = convertToNewsItem(data)
	}

	return result, nil
}

// SaveUsedNewsIDs saves a map of used news IDs to the database
func (a *MongoDBAdapter) SaveUsedNewsIDs(ctx context.Context, usedIDs map[string]bool) error {
	return a.db.SaveUsedNewsIDs(ctx, usedIDs)
}

// GetUsedNewsIDs retrieves all used news IDs from the database
func (a *MongoDBAdapter) GetUsedNewsIDs(ctx context.Context) (map[string]bool, error) {
	return a.db.GetUsedNewsIDs(ctx)
}

// SaveGenerationQueue saves the current generation queue to the database
func (a *MongoDBAdapter) SaveGenerationQueue(ctx context.Context, queue []data.GenerationRequest) error {
	return a.db.SaveGenerationQueue(ctx, queue)
}

// GetGenerationQueue retrieves the pending generation requests from the database
func (a *MongoDBAdapter) GetGenerationQueue(ctx context.Context) ([]data.GenerationRequest, error) {
	return a.db.GetGenerationQueue(ctx)
}

// Helper functions to convert between MongoDB and data types

// convertToFish converts MongoDB fish data to internal type
func convertToFish(fishData *FishData) *fish.Fish {
	// Extract stat effects
	statEffects := make(fish.StatEffects, 0)
	for _, effect := range fishData.StatEffects {
		if statType, ok := effect["stat"].(string); ok {
			statEffect := fish.StatEffect{
				Stat: fish.StatType(statType),
			}

			// Extract value
			if value, ok := effect["value"].(float64); ok {
				statEffect.Value = value
			}

			// Extract is_percentage
			if isPercentage, ok := effect["is_percentage"].(bool); ok {
				statEffect.IsPercent = isPercentage
			}

			// Extract duration
			if duration, ok := effect["duration"].(float64); ok {
				statEffect.Duration = int(duration)
			}

			statEffects = append(statEffects, statEffect)
		}
	}

	reason := fishData.GenerationReason
	if reason == "" {
		reason = "Unknown reason" // Provide default for older records
	}

	return &fish.Fish{
		Name:             fishData.Name,
		Description:      fishData.Description,
		Rarity:           fish.Rarity(fishData.Rarity),
		Size:             fishData.Length,
		Value:            float64(int(fishData.Length * 10 * float64(rarityToMultiplier(fishData.Rarity)))),
		Effect:           generateEffectFromStatEffects(statEffects),
		DataSource:       fishData.DataSource,
		IsAIGenerated:    fishData.IsAIGenerated,
		StatEffects:      statEffects,
		GenerationReason: reason,
	}
}

// rarityToMultiplier converts a rarity string to a value multiplier
func rarityToMultiplier(rarity string) int {
	switch rarity {
	case "Common":
		return 1
	case "Uncommon":
		return 3
	case "Rare":
		return 6
	case "Epic":
		return 12
	case "Legendary":
		return 25
	default:
		return 1
	}
}

// generateEffectFromStatEffects creates a human-readable effect description
func generateEffectFromStatEffects(effects fish.StatEffects) string {
	if len(effects) == 0 {
		return "No special effects."
	}

	effectDescriptions := []string{}
	for _, effect := range effects {
		effectDesc := formatStatEffect(effect)
		effectDescriptions = append(effectDescriptions, effectDesc)
	}

	return strings.Join(effectDescriptions, " ")
}

// formatStatEffect formats a single stat effect
func formatStatEffect(effect fish.StatEffect) string {
	valueStr := ""
	if effect.IsPercent {
		valueStr = fmt.Sprintf("%.0f%%", effect.Value)
	} else {
		valueStr = fmt.Sprintf("%.0f", effect.Value)
	}

	switch effect.Stat {
	case fish.CatchChance:
		return fmt.Sprintf("Increases catch chance by %s for %d seconds", valueStr, effect.Duration)
	case fish.CriticalCatch:
		return fmt.Sprintf("Increases critical catch rate by %s for %d seconds", valueStr, effect.Duration)
	case fish.Luck:
		return fmt.Sprintf("Increases fishing luck by %s for %d seconds", valueStr, effect.Duration)
	case fish.StaminaRegen:
		return fmt.Sprintf("Increases stamina regeneration by %s for %d seconds", valueStr, effect.Duration)
	case fish.SellValue:
		return fmt.Sprintf("Increases sell value of fish by %s for %d seconds", valueStr, effect.Duration)
	case fish.MarketDemand:
		return fmt.Sprintf("Increases market demand by %s for %d seconds", valueStr, effect.Duration)
	case fish.BaitCost:
		return fmt.Sprintf("Reduces bait cost by %s for %d seconds", valueStr, effect.Duration)
	case fish.ExploreSpeed:
		return fmt.Sprintf("Increases exploration speed by %s for %d seconds", valueStr, effect.Duration)
	case fish.AreaAccess:
		return fmt.Sprintf("Grants access to restricted fishing areas for %d seconds", effect.Duration)
	case fish.WeatherResist:
		return fmt.Sprintf("Increases weather resistance by %s for %d seconds", valueStr, effect.Duration)
	case fish.StorageSpace:
		return fmt.Sprintf("Increases storage capacity by %s for %d seconds", valueStr, effect.Duration)
	case fish.PreserveDuration:
		return fmt.Sprintf("Increases fish preservation time by %s for %d seconds", valueStr, effect.Duration)
	case fish.CollectionBonus:
		return fmt.Sprintf("Increases collection bonuses by %s for %d seconds", valueStr, effect.Duration)
	default:
		return fmt.Sprintf("Unknown effect on %s by %s for %d seconds", effect.Stat, valueStr, effect.Duration)
	}
}

// Convert stat effects from fish.StatEffects to []map[string]interface{}
func convertStatEffects(effects fish.StatEffects) []map[string]interface{} {
	result := make([]map[string]interface{}, len(effects))

	for i, effect := range effects {
		result[i] = map[string]interface{}{
			"stat":          string(effect.Stat),
			"value":         effect.Value,
			"is_percentage": effect.IsPercent,
			"duration":      effect.Duration,
		}
	}

	return result
}

// convertToWeatherInfo converts MongoDB weather data to internal type
func convertToWeatherInfo(mongoData *WeatherData) *data.WeatherInfo {
	return &data.WeatherInfo{
		Condition: mongoData.Condition,
		TempC:     mongoData.TempC,
		Humidity:  int(mongoData.Humidity),
		WindKph:   mongoData.WindSpeed * 3.6, // Convert m/s to km/h
		Location:  mongoData.RegionID,
		IsExtreme: isExtremeWeather(mongoData.Condition, mongoData.TempC),
	}
}

// convertToNewsItem converts MongoDB news data to internal type
func convertToNewsItem(mongoData *NewsData) *data.NewsItem {
	return &data.NewsItem{
		Headline:    mongoData.Headline,
		Source:      mongoData.Source,
		URL:         mongoData.URL,
		Sentiment:   mongoData.Sentiment,
		Keywords:    mongoData.Keywords,
		Category:    getCategoryFromKeywords(mongoData.Keywords),
		PublishedAt: mongoData.PublishedAt,
	}
}

// Helper functions for extracting data from text descriptions

// extractColorFromDescription attempts to extract a color from fish description
func extractColorFromDescription(description string) string {
	colorKeywords := map[string]bool{
		"red": true, "blue": true, "green": true, "yellow": true, "orange": true,
		"purple": true, "black": true, "white": true, "silver": true, "gold": true,
		"pink": true, "brown": true, "gray": true, "grey": true, "teal": true,
		"crimson": true, "azure": true, "turquoise": true, "violet": true,
	}

	words := strings.Fields(strings.ToLower(description))
	for _, word := range words {
		clean := strings.Trim(word, ",.;:!?")
		if colorKeywords[clean] {
			return clean
		}
	}

	// Default color if none found
	return "silver"
}

// extractHabitatFromDescription attempts to extract a habitat from fish description
func extractHabitatFromDescription(description string) string {
	habitatKeywords := map[string]string{
		"deep": "deep sea", "reef": "coral reef", "shallow": "shallow waters",
		"coastal": "coastal waters", "river": "riverine", "lake": "lake",
		"tropic": "tropical", "arctic": "arctic", "polar": "polar",
		"warm": "warm waters", "cold": "cold waters", "fresh": "freshwater",
		"salt": "saltwater", "ocean": "oceanic", "sea": "sea",
		"cave": "underwater caves", "rocky": "rocky bottoms", "sandy": "sandy bottoms",
	}

	lowerDesc := strings.ToLower(description)
	for keyword, habitat := range habitatKeywords {
		if strings.Contains(lowerDesc, keyword) {
			return habitat
		}
	}

	// Default habitat if none found
	return "oceanic"
}

// extractDietFromDescription attempts to extract a diet from fish description
func extractDietFromDescription(description string) string {
	dietKeywords := map[string]string{
		"plankton": "plankton", "algae": "algae", "plant": "herbivore",
		"crab": "crustaceans", "shrimp": "crustaceans", "worm": "worms",
		"insect": "insects", "larva": "larvae", "fish": "piscivore",
		"mollusk": "mollusks", "omnivore": "omnivore", "carnivore": "carnivore",
		"herbivore": "herbivore", "coral": "coral polyps", "eat": "omnivore",
		"feed": "omnivore", "prey": "carnivore", "hunt": "carnivore",
	}

	lowerDesc := strings.ToLower(description)
	for keyword, diet := range dietKeywords {
		if strings.Contains(lowerDesc, keyword) {
			return diet
		}
	}

	// Default diet if none found
	return "omnivore"
}

// isExtremeWeather determines if weather conditions are extreme
func isExtremeWeather(condition string, tempC float64) bool {
	extremeConditions := map[string]bool{
		"thunderstorm": true,
		"tornado":      true,
		"hurricane":    true,
		"blizzard":     true,
		"hail":         true,
	}

	// Check for extreme temperatures
	if tempC > 40 || tempC < -20 {
		return true
	}

	// Check for extreme conditions
	for extreme := range extremeConditions {
		if strings.Contains(strings.ToLower(condition), extreme) {
			return true
		}
	}

	return false
}

// getCategoryFromKeywords determines a category from keywords
func getCategoryFromKeywords(keywords []string) string {
	categories := map[string][]string{
		"politics":      {"politics", "government", "election", "president", "congress"},
		"technology":    {"technology", "software", "hardware", "digital", "app", "mobile"},
		"science":       {"science", "research", "discovery", "scientific", "biology", "physics"},
		"health":        {"health", "medical", "medicine", "wellness", "disease", "doctor"},
		"environment":   {"environment", "climate", "pollution", "renewable", "sustainable"},
		"business":      {"business", "economy", "stock", "market", "company", "financial"},
		"entertainment": {"entertainment", "movie", "music", "celebrity", "game", "media"},
		"sports":        {"sport", "athlete", "team", "championship", "olympic", "match"},
	}

	categoryCount := make(map[string]int)

	for _, keyword := range keywords {
		keywordLower := strings.ToLower(keyword)
		for category, categoryKeywords := range categories {
			for _, categoryKeyword := range categoryKeywords {
				if strings.Contains(keywordLower, categoryKeyword) {
					categoryCount[category]++
					break
				}
			}
		}
	}

	// Find category with highest count
	bestCategory := "general"
	bestCount := 0

	for category, count := range categoryCount {
		if count > bestCount {
			bestCount = count
			bestCategory = category
		}
	}

	return bestCategory
}
