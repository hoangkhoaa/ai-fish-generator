package fish

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"fish-generate/internal/data"
)

const (
	DailyFishLimit = 50 // Maximum number of fish that can be generated per day
)

// StorageAdapter is an interface for database adapters that can store and retrieve fish
type StorageAdapter interface {
	SaveFishData(ctx context.Context, fish *Fish) error
	GetDailyFishCount(ctx context.Context) (int, error)
	GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*Fish, error)
	GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*Fish, error)
	GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*Fish, error)
}

// Service coordinates the data collection and fish generation
type Service struct {
	generator      *Generator
	collectors     []data.DataCollector
	dataEvents     chan *data.DataEvent
	fishCreated    chan *Fish
	fishStore      map[string][]*Fish // Store fish by data source type
	useAI          bool               // Whether to use AI for fish generation
	apiKey         string             // API key for AI generation
	storageAdapter StorageAdapter     // For database operations (optional)
}

// ServiceOptions contains configuration options for the service
type ServiceOptions struct {
	GeminiAPIKey   string         // API key for Google Gemini
	UseAI          bool           // Whether to use AI for fish generation
	OpenWeatherKey string         // API key for OpenWeatherMap
	EIAKey         string         // API key for EIA (Energy Information Administration)
	NewsAPIKey     string         // API key for NewsAPI
	TestMode       bool           // Whether to run in test mode (shorter intervals)
	StorageAdapter StorageAdapter // Optional adapter for database operations
}

// NewService creates a new fish generation service
func NewService(collectors []data.DataCollector, opts ...ServiceOptions) *Service {
	// Set default options
	var options ServiceOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Create generator with options
	generatorOpts := GeneratorOptions{
		GeminiAPIKey: options.GeminiAPIKey,
		UseAI:        options.UseAI,
		TestMode:     options.TestMode,
	}

	return &Service{
		generator:      NewGenerator(generatorOpts),
		collectors:     collectors,
		dataEvents:     make(chan *data.DataEvent, 100),
		fishCreated:    make(chan *Fish, 100),
		fishStore:      make(map[string][]*Fish),
		useAI:          options.UseAI,
		apiKey:         options.GeminiAPIKey,
		storageAdapter: options.StorageAdapter,
	}
}

// NewServiceSimple creates a simplified service with just the generator
func NewServiceSimple(opts ...ServiceOptions) *Service {
	var options ServiceOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	generatorOpts := GeneratorOptions{
		GeminiAPIKey: options.GeminiAPIKey,
		UseAI:        options.UseAI,
		TestMode:     options.TestMode,
	}

	return &Service{
		generator:      NewGenerator(generatorOpts),
		dataEvents:     make(chan *data.DataEvent, 100),
		fishCreated:    make(chan *Fish, 100),
		fishStore:      make(map[string][]*Fish),
		useAI:          options.UseAI,
		apiKey:         options.GeminiAPIKey,
		storageAdapter: options.StorageAdapter,
	}
}

// GenerateFish creates a fish based on the given data event type and value
func (s *Service) GenerateFish(ctx context.Context, eventType data.DataType, value interface{}) (*Fish, error) {
	// Check daily fish limit if we have a storage adapter
	if s.storageAdapter != nil {
		count, err := s.storageAdapter.GetDailyFishCount(ctx)
		if err != nil {
			log.Printf("Warning: failed to check daily fish count: %v", err)
		} else if count >= DailyFishLimit {
			log.Printf("Daily fish generation limit (%d) reached. Attempting to find similar fish.", DailyFishLimit)

			// Try to find a similar fish from the database
			dataSource := string(eventType)
			var rarityLevel string // Will be determined based on data

			// Determine a suitable rarity based on the data
			switch eventType {
			case data.WeatherData:
				if wi, ok := value.(*data.WeatherInfo); ok && wi.IsExtreme {
					rarityLevel = string(Rare) // Extreme weather tends to produce rarer fish
				}
			case data.BitcoinData:
				if cp, ok := value.(*data.CryptoPrice); ok && cp.Change24h > 5.0 {
					rarityLevel = string(Rare) // Significant price changes produce rarer fish
				}
			case data.NewsData:
				if ni, ok := value.(*data.NewsItem); ok && (ni.Sentiment > 0.7 || ni.Sentiment < -0.7) {
					rarityLevel = string(Epic) // Strong sentiment news produces epics
				}
			}

			// If rarity wasn't determined, use a random one weighted toward common
			if rarityLevel == "" {
				rarityRoll := rand.Float64()
				if rarityRoll < 0.6 {
					rarityLevel = string(Common)
				} else if rarityRoll < 0.85 {
					rarityLevel = string(Uncommon)
				} else if rarityRoll < 0.95 {
					rarityLevel = string(Rare)
				} else if rarityRoll < 0.99 {
					rarityLevel = string(Epic)
				} else {
					rarityLevel = string(Legendary)
				}
			}

			// Try to retrieve a fish from the database
			fish, err := s.storageAdapter.GetSimilarFish(ctx, dataSource, rarityLevel)
			if err == nil && fish != nil {
				// Notify subscribers about the reused fish
				select {
				case s.fishCreated <- fish:
					// Fish sent to channel
					log.Printf("Reusing existing %s fish: %s", rarityLevel, fish.Name)
				default:
					// Channel full, just log
					log.Println("Fish created channel full, skipping notification")
				}
				return fish, nil
			}

			log.Printf("Could not find similar fish: %v. Proceeding with generation anyway.", err)
		}
	}

	// Generate a fish based on the event type and value
	var fish *Fish
	var reason string

	// Create a reason for the fish generation based on the event type
	switch eventType {
	case data.WeatherData:
		if weatherInfo, ok := value.(*data.WeatherInfo); ok {
			reason = fmt.Sprintf("Weather condition: %s in %s", weatherInfo.Condition, weatherInfo.Location)
			fish = s.generator.GenerateFromWeather(weatherInfo, reason)
		} else {
			return nil, fmt.Errorf("invalid weather info type")
		}
	case data.BitcoinData:
		if cryptoPrice, ok := value.(*data.CryptoPrice); ok {
			reason = fmt.Sprintf("Bitcoin price update: $%.2f (%.2f%%)", cryptoPrice.PriceUSD, cryptoPrice.Change24h)
			fish = s.generator.GenerateFromBitcoin(cryptoPrice, reason)
		} else {
			return nil, fmt.Errorf("invalid crypto price type")
		}
	case data.NewsData:
		if newsItem, ok := value.(*data.NewsItem); ok {
			// Use the truncateString from generator.go
			headline := newsItem.Headline
			if len(headline) > 50 {
				headline = headline[:47] + "..."
			}
			reason = fmt.Sprintf("News update: %s", headline)
			fish = s.generator.GenerateFromNews(ctx, newsItem, reason)
		} else {
			return nil, fmt.Errorf("invalid news item type")
		}
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	if fish == nil {
		return nil, fmt.Errorf("failed to generate fish for event type: %s", eventType)
	}

	// Update fish store
	s.fishStore[string(eventType)] = append(s.fishStore[string(eventType)], fish)

	// Save to database if we have a storage adapter
	if s.storageAdapter != nil {
		if err := s.storageAdapter.SaveFishData(ctx, fish); err != nil {
			log.Printf("Warning: failed to save fish to database: %v", err)
		}
	}

	// Notify subscribers
	select {
	case s.fishCreated <- fish:
		// Fish sent to channel
	default:
		// Channel full, just log
		log.Println("Fish created channel full, skipping notification")
	}

	return fish, nil
}

// Start begins collecting data and generating fish
func (s *Service) Start(ctx context.Context) error {
	// Initialize fish store for each data type
	s.fishStore = map[string][]*Fish{
		string(data.WeatherData): {},
		string(data.BitcoinData): {},
		string(data.NewsData):    {},
		string(data.GoldData):    {},
	}

	// Start all collectors
	for _, collector := range s.collectors {
		dataType := collector.GetType()
		log.Printf("Starting data collector for %s", dataType)

		// Use different intervals for different collectors
		var interval time.Duration

		if s.generator.options.TestMode {
			// Use very short intervals in test mode
			interval = 5 * time.Second
		} else {
			// Production mode - fish every 30 minutes
			switch dataType {
			case data.WeatherData:
				interval = 30 * time.Minute
			case data.BitcoinData:
				interval = 30 * time.Minute
			case data.NewsData:
				interval = 30 * time.Minute
			case data.GoldData:
				interval = 30 * time.Minute
			default:
				interval = 30 * time.Minute
			}
		}

		go collector.Start(ctx, interval, s.dataEvents)
	}

	// Start the fish generation processor
	go s.processDataEvents(ctx)

	return nil
}

// processDataEvents handles incoming data events and generates fish
func (s *Service) processDataEvents(ctx context.Context) {
	for {
		select {
		case event := <-s.dataEvents:
			// Create a context for API calls
			genCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			fish, err := s.GenerateFish(genCtx, event.Type, event.Value)
			cancel()

			if err != nil {
				log.Printf("Failed to generate fish from event type %s: %v", event.Type, err)
				continue
			}

			if fish != nil {
				log.Printf("Generated %s fish: %s (Rarity: %s, Size: %.2f, Value: $%.2f)",
					event.Type, fish.Name, fish.Rarity, fish.Size, fish.Value)
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetAllFish returns all fish generated so far
func (s *Service) GetAllFish() []*Fish {
	var allFish []*Fish

	for _, fishList := range s.fishStore {
		allFish = append(allFish, fishList...)
	}

	return allFish
}

// GetFishByType returns fish of a specific data source type
func (s *Service) GetFishByType(dataType data.DataType) []*Fish {
	return s.fishStore[string(dataType)]
}

// SubscribeToFishCreation allows subscribers to receive new fish as they're created
func (s *Service) SubscribeToFishCreation(ctx context.Context, subscriber chan<- *Fish) {
	for {
		select {
		case fish := <-s.fishCreated:
			subscriber <- fish
		case <-ctx.Done():
			return
		}
	}
}

// GetAIGenerationStats returns statistics about AI vs rule-based generation
func (s *Service) GetAIGenerationStats() map[string]int {
	stats := map[string]int{
		"ai_generated": 0,
		"rule_based":   0,
		"total":        0,
	}

	allFish := s.GetAllFish()
	for _, fish := range allFish {
		stats["total"]++
		if fish.IsAIGenerated {
			stats["ai_generated"]++
		} else {
			stats["rule_based"]++
		}
	}

	return stats
}

// GetFishReport generates a summary report of all fish created
func (s *Service) GetFishReport() string {
	allFish := s.GetAllFish()
	if len(allFish) == 0 {
		return "No fish have been generated yet."
	}

	// Count fish by rarity and source
	rarityCount := map[Rarity]int{
		Common:    0,
		Uncommon:  0,
		Rare:      0,
		Epic:      0,
		Legendary: 0,
	}
	sourceCount := map[string]int{}
	rarityStats := map[Rarity]map[StatType]int{}

	// Initialize rarity stats
	for _, rarity := range []Rarity{Common, Uncommon, Rare, Epic, Legendary} {
		rarityStats[rarity] = map[StatType]int{}
	}

	// Record stats
	for _, fish := range allFish {
		rarityCount[fish.Rarity]++
		sourceCount[fish.DataSource]++

		// Record stat effects
		for _, effect := range fish.StatEffects {
			rarityStats[fish.Rarity][effect.Stat]++
		}
	}

	// Generate report
	totalFish := len(allFish)
	var report string

	report = fmt.Sprintf("Fish Generation Report\n")
	report += fmt.Sprintf("=====================\n\n")
	report += fmt.Sprintf("Total Fish Generated: %d\n\n", totalFish)

	// Add rarity breakdown
	report += "Rarity Breakdown:\n"
	for rarity, count := range rarityCount {
		percentage := 0.0
		if totalFish > 0 {
			percentage = float64(count) / float64(totalFish) * 100.0
		}
		report += fmt.Sprintf("  %s: %d (%.1f%%)\n", rarity, count, percentage)
	}
	report += "\n"

	// Add source breakdown
	report += "Source Breakdown:\n"
	for source, count := range sourceCount {
		if count > 0 { // Only show sources with fish
			percentage := 0.0
			if totalFish > 0 {
				percentage = float64(count) / float64(totalFish) * 100.0
			}

			// Make the AI source more visible
			if source == "news-ai" {
				report += fmt.Sprintf("  🤖 AI (Gemma Model): %d (%.1f%%)\n", count, percentage)
			} else {
				report += fmt.Sprintf("  %s: %d (%.1f%%)\n", source, count, percentage)
			}
		}
	}

	// Add stat effects information by rarity
	report += "\nGame Stat Effects by Rarity:\n"
	report += "==========================\n"
	for _, rarity := range []Rarity{Common, Uncommon, Rare, Epic, Legendary} {
		if rarityCount[rarity] > 0 {
			report += fmt.Sprintf("\n%s Fish Effects:\n", rarity)

			// Get the most common stat effects for this rarity
			stats := rarityStats[rarity]
			if len(stats) == 0 {
				report += "  No effects recorded\n"
				continue
			}

			// Sort stats by frequency (we'll use a simple approach here)
			type statFreq struct {
				stat  StatType
				count int
			}

			statsList := []statFreq{}
			for stat, count := range stats {
				statsList = append(statsList, statFreq{stat, count})
			}

			// Sort by count (we'd use sort.Slice in a more complex implementation)
			// Here we'll just report all stats
			for _, sf := range statsList {
				percentage := float64(sf.count) / float64(rarityCount[rarity]) * 100.0
				report += fmt.Sprintf("  • %s: %.1f%% of %s fish\n",
					formatStatName(sf.stat), percentage, rarity)
			}
		}
	}

	return report
}

// Stop stops the service and cleans up resources
func (s *Service) Stop(ctx context.Context) error {
	// Nothing to clean up at the moment
	return nil
}
