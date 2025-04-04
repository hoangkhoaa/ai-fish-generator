package data

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Add colored logging constants at the top of the file
const (
	logColorReset  = "\033[0m"
	logColorRed    = "\033[31m"
	logColorGreen  = "\033[32m"
	logColorYellow = "\033[33m"
	logColorBlue   = "\033[34m"
	logColorPurple = "\033[35m"
	logColorCyan   = "\033[36m"
	logColorWhite  = "\033[37m"
	// Add a new log color for translations
	logColorTeal = "\033[36;1m" // Bright cyan for translation logs
)

// GenerationRequest represents a queued fish generation request
type GenerationRequest struct {
	ID      string          `bson:"_id,omitempty" json:"id,omitempty"`
	Ctx     context.Context `bson:"-" json:"-"` // Can't be persisted
	Reason  string          `bson:"reason" json:"reason"`
	AddedAt time.Time       `bson:"added_at" json:"added_at"`
}

// logWeather logs weather-related messages with blue color
func logWeather(format string, v ...interface{}) {
	log.Printf(logColorBlue+"[WEATHER] "+format+logColorReset, v...)
}

// logBitcoin logs bitcoin-related messages with yellow color
func logBitcoin(format string, v ...interface{}) {
	log.Printf(logColorYellow+"[BITCOIN] "+format+logColorReset, v...)
}

// logGold logs gold-related messages with yellow color
func logGold(format string, v ...interface{}) {
	log.Printf(logColorYellow+"[GOLD] "+format+logColorReset, v...)
}

// logNews logs news-related messages with purple color
func logNews(format string, v ...interface{}) {
	log.Printf(logColorPurple+"[NEWS] "+format+logColorReset, v...)
}

// logFish logs fish generation messages with green color
func logFish(format string, v ...interface{}) {
	log.Printf(logColorGreen+"[FISH] "+format+logColorReset, v...)
}

// logError logs error messages with red color
func logError(format string, v ...interface{}) {
	log.Printf(logColorRed+"[ERROR] "+format+logColorReset, v...)
}

// logTranslation logs translation-related messages with teal color
func logTranslation(format string, v ...interface{}) {
	log.Printf(logColorTeal+"[TRANSLATE] "+format+logColorReset, v...)
}

// DatabaseClient defines the interface for database operations
type DatabaseClient interface {
	SaveWeatherData(ctx context.Context, weatherInfo *WeatherInfo, regionID, cityID string) error
	SavePriceData(ctx context.Context, assetType string, price, volume, changePercent, volumeChange float64, source string) error
	SaveNewsData(ctx context.Context, newsItem *NewsItem) error
	GetRecentWeatherData(ctx context.Context, regionID string, limit int) ([]*WeatherInfo, error)
	GetRecentPriceData(ctx context.Context, assetType string, limit int) ([]map[string]interface{}, error)
	GetRecentNewsData(ctx context.Context, limit int) ([]*NewsItem, error)
	SaveFishData(ctx context.Context, fishData interface{}) error
	// New methods for persistence
	SaveUsedNewsIDs(ctx context.Context, usedIDs map[string]bool) error
	GetUsedNewsIDs(ctx context.Context) (map[string]bool, error)
	SaveGenerationQueue(ctx context.Context, queue []GenerationRequest) error
	GetGenerationQueue(ctx context.Context) ([]GenerationRequest, error)
	// Methods for translation
	GetUntranslatedFish(ctx context.Context, limit int) ([]map[string]interface{}, error)
	UpdateFishWithTranslation(ctx context.Context, fishID interface{}, translatedFish map[string]interface{}) error
}

// CollectionSettings holds configuration for data collection
type CollectionSettings struct {
	WeatherInterval     time.Duration // 1 hour in production
	PriceInterval       time.Duration // 6 hours in production
	NewsInterval        time.Duration // 20 minutes in production
	TestMode            bool
	GeminiApiKey        string        // API key for Gemini
	GenerationCooldown  time.Duration // Optional generation cooldown
	EnableTranslation   bool          // Whether to enable Vietnamese translation
	TranslationCooldown time.Duration // Cooldown between translations
}

// DataManager handles data collection across different regions and sources
type DataManager struct {
	settings         CollectionSettings
	db               DatabaseClient
	weatherCollector *WeatherCollector
	bitcoinCollector *CryptoCollector
	goldCollector    *GoldCollector
	newsCollector    *NewsCollector
	geminiClient     *GeminiClient
	translatorClient *TranslatorClient // Add translator client
	regions          []Region
	cancelFuncs      []context.CancelFunc
	mu               sync.Mutex
	// Store most recent data for each type
	lastWeatherData *WeatherInfo
	lastBitcoinData *CryptoPrice
	lastGoldData    *GoldPrice
	lastNewsData    *NewsItem
	// For merged news generation
	mergedNewsItem  *NewsItem   // For backward compatibility
	mergedNewsItems []*NewsItem // Store up to 2 additional news items
	// Test mode tracking
	initialDataCollected bool
	dataReady            bool
	// Cooldown tracking
	lastFishGeneration time.Time
	generationCooldown time.Duration
	// Track which news IDs have been used for generation
	usedNewsIDs map[string]bool
	// Generation queue for better cooldown management
	generationQueue     []GenerationRequest
	queueProcessRunning bool
	// Add WaitGroup to track running goroutines
	wg sync.WaitGroup
	// Translation tracking
	lastTranslation     time.Time
	translationCooldown time.Duration
}

// NewDataManager creates a new data manager
func NewDataManager(settings CollectionSettings, db DatabaseClient,
	weatherApiKey, newsApiKey, metalPriceApiKey, geminiApiKey string) *DataManager {

	regions := PredefinedRegions()

	// Set default intervals if not specified
	if settings.WeatherInterval == 0 {
		settings.WeatherInterval = 1 * time.Hour // 1 hour default for weather
	}
	if settings.PriceInterval == 0 {
		settings.PriceInterval = 6 * time.Hour // 6 hours default for prices
	}
	if settings.NewsInterval == 0 {
		settings.NewsInterval = 20 * time.Minute // 20 minutes default for news
	}

	// Set generation cooldown based on test mode or settings
	var generationCooldown time.Duration
	if settings.GenerationCooldown > 0 {
		// Use the value from settings if provided
		generationCooldown = settings.GenerationCooldown
	} else if settings.TestMode {
		// Use a fast cooldown in test mode
		generationCooldown = 10 * time.Second // 10 seconds in test mode
	} else {
		// Default production cooldown
		generationCooldown = 25 * time.Minute // 25 minutes in production
	}

	// Set translation cooldown or use default
	var translationCooldown time.Duration
	if settings.TranslationCooldown > 0 {
		translationCooldown = settings.TranslationCooldown
	} else {
		translationCooldown = 2 * time.Minute // 2 minutes default for translation
	}

	// Create translator client if translation is enabled
	var translatorClient *TranslatorClient
	if settings.EnableTranslation {
		translatorClient = NewTranslatorClient(geminiApiKey)
		log.Println("Vietnamese fish translation enabled with cooldown:", translationCooldown)
	}

	return &DataManager{
		settings:             settings,
		db:                   db,
		weatherCollector:     NewWeatherCollector(weatherApiKey),
		bitcoinCollector:     NewCryptoCollector(),
		goldCollector:        NewGoldCollector(metalPriceApiKey),
		newsCollector:        NewNewsCollector(newsApiKey),
		geminiClient:         NewGeminiClient(geminiApiKey),
		translatorClient:     translatorClient,
		regions:              regions,
		cancelFuncs:          make([]context.CancelFunc, 0),
		initialDataCollected: false,
		dataReady:            false,
		lastFishGeneration:   time.Time{}, // Zero time means no generation has happened yet
		generationCooldown:   generationCooldown,
		usedNewsIDs:          make(map[string]bool),
		generationQueue:      make([]GenerationRequest, 0),
		queueProcessRunning:  false,
		mergedNewsItem:       nil, // Initialize to nil
		mergedNewsItems:      make([]*NewsItem, 0),
		wg:                   sync.WaitGroup{},
		lastTranslation:      time.Time{}, // Zero time means no translation has happened yet
		translationCooldown:  translationCooldown,
	}
}

// Start begins all data collection processes
func (m *DataManager) Start(ctx context.Context) {
	baseCtx, cancel := context.WithCancel(ctx)
	m.cancelFuncs = append(m.cancelFuncs, cancel)

	// Immediately collect initial data from all sources
	m.collectInitialData(baseCtx)

	// Use a single goroutine for all periodic tasks
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		// Create tickers for each collection type
		weatherTicker := time.NewTicker(m.settings.WeatherInterval)
		priceTicker := time.NewTicker(m.settings.PriceInterval)
		newsTicker := time.NewTicker(m.settings.NewsInterval)

		defer weatherTicker.Stop()
		defer priceTicker.Stop()
		defer newsTicker.Stop()

		// Add a ticker for checking for fish to translate
		var translationTicker *time.Ticker
		if m.settings.EnableTranslation && m.translatorClient != nil {
			translationTicker = time.NewTicker(30 * time.Second)
			defer translationTicker.Stop()
		}

		for {
			// Use different select statements based on whether translation is enabled
			if translationTicker != nil {
				select {
				case <-weatherTicker.C:
					// Collect weather for all regions
					for _, region := range m.regions {
						m.collectWeatherDataForRegion(baseCtx, region)
					}

				case <-priceTicker.C:
					// Collect price data
					m.collectPriceData(baseCtx)

				case <-newsTicker.C:
					// Collect news data
					m.collectNewsData(baseCtx)

				case <-translationTicker.C:
					// Check for fish to translate
					m.checkForFishToTranslate(baseCtx)

				case <-baseCtx.Done():
					log.Println("Data collection goroutine stopped")
					return
				}
			} else {
				// Version without translation ticker
				select {
				case <-weatherTicker.C:
					// Collect weather for all regions
					for _, region := range m.regions {
						m.collectWeatherDataForRegion(baseCtx, region)
					}

				case <-priceTicker.C:
					// Collect price data
					m.collectPriceData(baseCtx)

				case <-newsTicker.C:
					// Collect news data
					m.collectNewsData(baseCtx)

				case <-baseCtx.Done():
					log.Println("Data collection goroutine stopped")
					return
				}
			}
		}
	}()

	// Start queue processor if any initialization data needs to be processed
	if len(m.generationQueue) > 0 {
		m.startQueueProcessor(baseCtx)
	}

	log.Println("Data Manager started successfully")
}

// collectInitialData collects data from all sources at startup
func (m *DataManager) collectInitialData(ctx context.Context) {
	log.Println("Collecting initial data from all sources...")

	// Load persistent state from database
	m.loadPersistentState(ctx)

	// We no longer mark all existing news as used at startup
	// This allows using older news for generations

	// Collect weather data for each region
	for _, region := range m.regions {
		m.collectWeatherDataForRegion(ctx, region)
	}

	// Collect price data
	m.collectPriceData(ctx)

	// Collect news data or retrieve from database
	err := m.collectNewsData(ctx)
	if err != nil && m.settings.TestMode && m.db != nil {
		log.Println("No news received, attempting to use latest news from database...")
		recentNews, err := m.db.GetRecentNewsData(ctx, 1)
		if err == nil && len(recentNews) > 0 {
			m.lastNewsData = recentNews[0]
			log.Printf("Using latest news from database: '%s'",
				truncateString(m.lastNewsData.Headline, 50))
		} else {
			log.Printf("Could not retrieve news from database: %v", err)
		}
	}

	// In test mode, generate fish at startup if we have news data
	if m.settings.TestMode && m.lastNewsData != nil {
		log.Println("Test mode: Generating initial fish using available data...")
		// Use the special startup reason that's allowed to reuse news
		m.generateFishWithLock(ctx, "initial test mode startup")
	}

	// If we have loaded any queued generation requests, start processing them
	if len(m.generationQueue) > 0 {
		logFish("Loaded %d pending generation requests from database", len(m.generationQueue))
		go m.processGenerationQueue()
	}

	m.initialDataCollected = true
	m.dataReady = true
	log.Println("Initial data collection completed")
}

// collectWeatherDataForRegion collects weather data for all cities in a region
func (m *DataManager) collectWeatherDataForRegion(ctx context.Context, region Region) {
	logWeather("Collecting weather data for region: %s", region.Name)

	// For each city in the region, collect weather data
	for _, cityID := range region.CityIDs {
		dataEvent, err := m.weatherCollector.Collect(ctx)
		if err != nil {
			logError("Error collecting weather data for city %s in region %s: %v",
				cityID, region.ID, err)
			continue
		}

		// Cast the value to WeatherInfo
		weatherInfo, ok := dataEvent.Value.(WeatherInfo)
		if !ok {
			// Try pointer type
			weatherInfoPtr, okPtr := dataEvent.Value.(*WeatherInfo)
			if !okPtr {
				logError("Invalid weather data type for city %s", cityID)
				continue
			}
			weatherInfo = *weatherInfoPtr
		}

		// Save to database
		if m.db != nil {
			err = m.db.SaveWeatherData(ctx, &weatherInfo, region.ID, cityID)
			if err != nil {
				logError("Error saving weather data for city %s: %v", cityID, err)
				continue
			}
			logWeather("Weather data saved for city %s in region %s: %s, %.1f°C",
				cityID, region.ID, weatherInfo.Condition, weatherInfo.TempC)
		}

		// Store the most recent weather data
		m.lastWeatherData = &weatherInfo
	}

	// Mark data as ready
	m.dataReady = true
}

// collectPriceData collects price data from all price sources
func (m *DataManager) collectPriceData(ctx context.Context) {
	// Collect Bitcoin data
	btcEvent, err := m.bitcoinCollector.Collect(ctx)
	if err != nil {
		logError("Error collecting Bitcoin data: %v", err)
	} else if m.db != nil {
		// Try both value and pointer types
		btcData, ok := btcEvent.Value.(CryptoPrice)
		if !ok {
			btcDataPtr, okPtr := btcEvent.Value.(*CryptoPrice)
			if !okPtr {
				logError("Invalid Bitcoin data type")
				return
			}
			btcData = *btcDataPtr
		}

		err = m.db.SavePriceData(ctx, "btc", btcData.PriceUSD, 0, btcData.Change24h, btcData.Volume24h, btcEvent.Source)
		if err != nil {
			logError("Error saving Bitcoin data: %v", err)
		} else {
			logBitcoin("Price data saved: $%.2f (%.2f%%)", btcData.PriceUSD, btcData.Change24h)
			// Store the most recent bitcoin data
			m.lastBitcoinData = &btcData
		}
	}

	// Collect Gold data - Try API first, then fall back to mock if needed
	goldEvent, err := m.goldCollector.Collect(ctx)
	if err != nil {
		logError("Error collecting Gold data: %v", err)
	} else if m.db != nil {
		// Try both value and pointer types
		goldData, ok := goldEvent.Value.(GoldPrice)
		if !ok {
			goldDataPtr, okPtr := goldEvent.Value.(*GoldPrice)
			if !okPtr {
				logError("Invalid Gold data type")
				return
			}
			goldData = *goldDataPtr
		}

		// Fix gold price if it's unrealistically low or not a valid number
		if goldData.PriceUSD < 100 || math.IsInf(goldData.PriceUSD, 0) || math.IsNaN(goldData.PriceUSD) {
			logError("Gold price invalid ($%.2f), adjusting to standard range", goldData.PriceUSD)
			goldData.PriceUSD = 1800.0 // Set a reasonable default gold price
		}

		err = m.db.SavePriceData(ctx, "gold", goldData.PriceUSD, 0, goldData.Change24h, 0, goldEvent.Source)
		if err != nil {
			logError("Error saving Gold data: %v", err)
		} else {
			logGold("Price data saved: $%.2f (%.2f%%)", goldData.PriceUSD, goldData.Change24h)
			// Store the most recent gold data
			m.lastGoldData = &goldData
		}
	}

	// Mark data as ready
	m.dataReady = true
}

// collectNewsData collects news data
func (m *DataManager) collectNewsData(ctx context.Context) error {
	newsEvent, err := m.newsCollector.Collect(ctx)
	if err != nil {
		logError("Error collecting news data: %v", err)
		return err
	}

	if m.db == nil {
		return fmt.Errorf("database client is nil")
	}

	logNews("Collected new batch of news articles from API")

	// Handle batch of news items or single news item
	switch value := newsEvent.Value.(type) {
	case []*NewsItem:
		// Process a batch of news items
		if len(value) == 0 {
			return fmt.Errorf("empty news batch returned")
		}

		savedCount := 0
		skippedCount := 0

		// Save all news items to the database
		for _, newsItem := range value {
			if newsItem == nil {
				continue
			}

			// Check if we already have this exact news item before saving
			newsID := newsItem.Source + ":" + newsItem.Headline
			if m.usedNewsIDs[newsID] {
				skippedCount++
				continue
			}

			err = m.db.SaveNewsData(ctx, newsItem)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					// This is a duplicate, just count it as skipped
					skippedCount++
					continue
				}

				logError("Error saving news item '%s': %v",
					truncateString(newsItem.Headline, 30), err)
				// Continue with other items even if one fails
				continue
			}

			savedCount++
			logNews("Saved: '%s' (%.2f sentiment, %s)",
				truncateString(newsItem.Headline, 50), newsItem.Sentiment, newsItem.Category)
		}

		logNews("Processed %d news items (%d saved, %d skipped)",
			len(value), savedCount, skippedCount)

		// Use the first item as the "latest" news for fish generation, if we saved any
		if savedCount > 0 {
			m.lastNewsData = value[0]
		}

	case NewsItem:
		// Handle single news item (backward compatibility)
		newsData := value
		err = m.db.SaveNewsData(ctx, &newsData)
		if err != nil {
			logError("Error saving news data: %v", err)
			return err
		}

		logNews("Data saved: '%s' (%.2f sentiment, category: %s)",
			truncateString(newsData.Headline, 50), newsData.Sentiment, newsData.Category)

		// Store the most recent news data
		m.lastNewsData = &newsData

	case *NewsItem:
		// Handle pointer to single news item (backward compatibility)
		if value == nil {
			return fmt.Errorf("nil news item returned")
		}

		newsData := *value
		err = m.db.SaveNewsData(ctx, value)
		if err != nil {
			logError("Error saving news data: %v", err)
			return err
		}

		logNews("Data saved: '%s' (%.2f sentiment, category: %s)",
			truncateString(newsData.Headline, 50), newsData.Sentiment, newsData.Category)

		// Store the most recent news data
		m.lastNewsData = value

	default:
		logError("Invalid news data type: %T", newsEvent.Value)
		return fmt.Errorf("invalid news data type: %T", newsEvent.Value)
	}

	// Simply save the news and initiate queue processing if needed
	// The queue processor will handle generating fish after cooldown
	m.mu.Lock()
	if !m.queueProcessRunning {
		m.mu.Unlock()
		// Try to find and process unused news items right away
		go func() {
			// Wait a short time to ensure all news items are saved
			time.Sleep(2 * time.Second)
			m.checkPendingNewsForGeneration(context.Background())

			// Start the queue processor
			m.startQueueProcessor(context.Background())
		}()
	} else {
		m.mu.Unlock()
	}

	return nil
}

// findNewsInSameCategory looks for another unused news item in the specified category
func (m *DataManager) findNewsInSameCategory(ctx context.Context, category string, excludeNewsID string) (*NewsItem, error) {
	// Get several recent news items
	recentNews, err := m.db.GetRecentNewsData(ctx, 20) // Check more news items to find a match
	if err != nil {
		return nil, fmt.Errorf("failed to get recent news: %v", err)
	}

	// Look for an unused news item in the same category
	for _, newsItem := range recentNews {
		newsID := newsItem.Source + ":" + newsItem.Headline

		// Skip the current news item and already used ones
		if newsID == excludeNewsID || m.usedNewsIDs[newsID] {
			continue
		}

		// If categories match, return this item
		if newsItem.Category == category {
			return newsItem, nil
		}
	}

	return nil, fmt.Errorf("no unused news found in category: %s", category)
}

// queueFishGenerationWithMergedNews queues fish generation with merged news context
func (m *DataManager) queueFishGenerationWithMergedNews(ctx context.Context, reason string, news1, news2 *NewsItem) {
	// Create a request and add it to the queue
	req := GenerationRequest{
		Ctx:     ctx,
		Reason:  reason,
		AddedAt: time.Now(),
	}

	// Store both news items for generation
	m.mu.Lock()
	// Save the primary news item
	m.lastNewsData = news1
	// Also store the secondary news item (will be handled at generation time)
	m.mergedNewsItems = append(m.mergedNewsItems, news2)
	m.generationQueue = append(m.generationQueue, req)
	m.mu.Unlock()

	// Save the queue to database in background
	go m.savePersistentState(ctx)

	// Start the queue processor if it's not already running
	if !m.queueProcessRunning {
		go m.processGenerationQueue()
	}

	logFish("Added to generation queue: %s (queue size: %d)", reason, len(m.generationQueue))
}

// generateFishFromData generates a fish using the current data context
// This function should be called with the mutex already locked
func (m *DataManager) generateFishFromData(ctx context.Context, reason string) {
	// Check for cooldown
	currentTime := time.Now()
	timeSinceLastGeneration := currentTime.Sub(m.lastFishGeneration)

	if !m.lastFishGeneration.IsZero() && timeSinceLastGeneration < m.generationCooldown {
		remaining := m.generationCooldown - timeSinceLastGeneration
		logFish("Fish generation is on cooldown (%.1f seconds remaining)",
			remaining.Seconds())
		return // Still on cooldown
	}

	// Check if we have necessary context data for a good fish
	if m.lastNewsData == nil {
		logError("No news data available for fish generation")
		return
	}

	var contextSummary []string
	sourcesAvailable := 0

	// Log how many merged news items we have for debugging
	if len(m.mergedNewsItems) > 0 {
		logFish("Using %d merged news items for this fish generation", len(m.mergedNewsItems))
	}

	// Always add primary news data first
	contextSummary = append(contextSummary, fmt.Sprintf("PRIMARY NEWS: \"%s\" (%.2f sentiment, %s)",
		m.lastNewsData.Headline, m.lastNewsData.Sentiment, m.lastNewsData.Category))

	// If we have merged news items, add them to the context summary
	if len(m.mergedNewsItems) > 0 {
		for i, mergedNews := range m.mergedNewsItems {
			contextSummary = append(contextSummary, fmt.Sprintf("MERGED NEWS %d: \"%s\" (%.2f sentiment, %s)",
				i+1, mergedNews.Headline, mergedNews.Sentiment, mergedNews.Category))
		}
	}

	if m.lastWeatherData != nil {
		sourcesAvailable++
		contextSummary = append(contextSummary, fmt.Sprintf("WEATHER: %s, %.1f°C",
			m.lastWeatherData.Condition, m.lastWeatherData.TempC))
	}

	// Add Bitcoin data
	if m.lastBitcoinData != nil {
		sourcesAvailable++
		changeDirection := ""
		if m.lastBitcoinData.Change24h > 2.0 {
			changeDirection = "up significantly"
		} else if m.lastBitcoinData.Change24h < -2.0 {
			changeDirection = "down significantly"
		} else if m.lastBitcoinData.Change24h > 0 {
			changeDirection = "up"
		} else {
			changeDirection = "down"
		}
		contextSummary = append(contextSummary, fmt.Sprintf("BITCOIN: $%.2f (%s %.2f%%%% in past 6hrs)",
			m.lastBitcoinData.PriceUSD, changeDirection, m.lastBitcoinData.Change24h))
	}

	// Add Gold data
	if m.lastGoldData != nil {
		sourcesAvailable++
		changeDirection := ""
		if m.lastGoldData.Change24h > 1.0 {
			changeDirection = "up"
		} else if m.lastGoldData.Change24h < -1.0 {
			changeDirection = "down"
		} else {
			changeDirection = "stable"
		}
		contextSummary = append(contextSummary, fmt.Sprintf("GOLD: $%.2f (%s %.2f%%%% in past 6hrs)",
			m.lastGoldData.PriceUSD, changeDirection, m.lastGoldData.Change24h))
	}

	// Print the context summary with divider lines for visibility
	logFish(strings.Repeat("-", 80))
	for _, line := range contextSummary {
		logFish(line)
	}
	logFish(strings.Repeat("-", 80))

	// We need at least 2 data sources (news is already checked above)
	if sourcesAvailable < 2 {
		logError("Insufficient context data for fish generation: only %d/4 sources available (minimum: 2)", sourcesAvailable)
		return
	}

	// Check if we have a valid API key
	if m.settings.GeminiApiKey == "" {
		logError("Gemini API key required for fish generation")
		return
	}

	logFish("Using Gemini API key (length: %d)", len(m.settings.GeminiApiKey))

	// Create a context with the API key
	fishCtx := context.WithValue(ctx, "gemini_api_key", m.settings.GeminiApiKey)

	// Prepare rich context for fish generation
	contextData := map[string]interface{}{
		"news": m.lastNewsData,
	}

	// Add merged news items if available
	if len(m.mergedNewsItems) > 0 {
		// Add array of merged news
		contextData["merged_news"] = m.mergedNewsItems
	}

	if m.lastWeatherData != nil {
		contextData["weather"] = m.lastWeatherData
	}

	if m.lastBitcoinData != nil {
		contextData["bitcoin"] = m.lastBitcoinData
	}

	if m.lastGoldData != nil {
		contextData["gold"] = m.lastGoldData
	}

	// Set cooldown time BEFORE generation to prevent simultaneous generations
	// This prevents multiple generations from being triggered while one is still in process
	m.lastFishGeneration = currentTime

	// Generate a unique fish using Gemini with all available context
	logFish("Generating unique fish using %d data sources and %d news articles (Reason: %s)...",
		sourcesAvailable, 1+len(m.mergedNewsItems), reason)
	fishData, err := m.geminiClient.GenerateUniqueFishFromContext(fishCtx, contextData, reason)

	if err != nil {
		logError("Error generating fish: %v", err)
		return
	}

	// Display fish information in an easy-to-read format
	logFish(strings.Repeat("=", 80))
	logFish("FISH GENERATED: \"%s\" (%s)", fishData.Name, fishData.Rarity)
	logFish("Description: %s", fishData.Description)
	logFish("Appearance: %s", fishData.Appearance)
	logFish("Color: %s", fishData.Color)
	logFish("Diet: %s", fishData.Diet)
	if fishData.Habitat != "" {
		logFish("Habitat: %s", fishData.Habitat)
	}

	// Override AI-generated numerical values with more varied random values

	// Create a new random source with current time as seed for better randomness
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	// Generate random rarity with weighted distribution
	rarityRoll := rng.Float64() * 100
	var rarity string
	var catchChanceMin, catchChanceMax float64
	var rarityMultiplier float64

	switch {
	case rarityRoll < 50: // 50% chance
		rarity = "Common"
		catchChanceMin, catchChanceMax = 60, 90
		rarityMultiplier = 1
	case rarityRoll < 75: // 25% chance
		rarity = "Uncommon"
		catchChanceMin, catchChanceMax = 40, 60
		rarityMultiplier = 3
	case rarityRoll < 90: // 15% chance
		rarity = "Rare"
		catchChanceMin, catchChanceMax = 20, 40
		rarityMultiplier = 6
	case rarityRoll < 98: // 8% chance
		rarity = "Epic"
		catchChanceMin, catchChanceMax = 10, 20
		rarityMultiplier = 12
	default: // 2% chance
		rarity = "Legendary"
		catchChanceMin, catchChanceMax = 1, 10
		rarityMultiplier = 25
	}

	// Generate random size - with occasional extreme values
	var lengthMeters float64
	sizeRoll := rng.Float64() * 100

	switch {
	case sizeRoll < 1: // 1% chance for tiny fish (1-5cm)
		lengthMeters = 0.01 + rng.Float64()*0.04
	case sizeRoll < 5: // 4% chance for small fish (5-15cm)
		lengthMeters = 0.05 + rng.Float64()*0.10
	case sizeRoll < 80: // 75% chance for medium fish (15cm-1m)
		lengthMeters = 0.15 + rng.Float64()*0.85
	case sizeRoll < 95: // 15% chance for large fish (1m-3m)
		lengthMeters = 1.0 + rng.Float64()*2.0
	case sizeRoll < 99: // 4% chance for huge fish (3m-10m)
		lengthMeters = 3.0 + rng.Float64()*7.0
	default: // 1% chance for massive fish (10m-25m)
		lengthMeters = 10.0 + rng.Float64()*15.0
	}

	// Round to 3 decimal places
	lengthMeters = math.Round(lengthMeters*1000) / 1000

	// Weight is roughly proportional to cube of length (volume)
	// For fish, use a more realistic density factor (average fish density is ~1.025g/cm³)
	// Using a formula: Weight (kg) = Length³ (m³) * Density (kg/m³) * Body Shape Factor
	// Where density for fish is ~1025 kg/m³ and body shape factor varies by type

	// Generate a random body shape factor based on vague fish description
	bodyShapeFactor := 0.0

	// Determine shape factor based on length (large fish tend to be more streamlined)
	if lengthMeters < 0.1 { // Tiny fish (minnows, tetras)
		bodyShapeFactor = 0.4 + rng.Float64()*0.2 // 0.4-0.6
	} else if lengthMeters < 0.5 { // Small fish (average aquarium fish)
		bodyShapeFactor = 0.3 + rng.Float64()*0.2 // 0.3-0.5
	} else if lengthMeters < 2.0 { // Medium fish (bass, salmon)
		bodyShapeFactor = 0.2 + rng.Float64()*0.15 // 0.2-0.35
	} else if lengthMeters < 5.0 { // Large fish (tuna, sharks)
		bodyShapeFactor = 0.15 + rng.Float64()*0.1 // 0.15-0.25
	} else { // Massive fish (whales)
		bodyShapeFactor = 0.1 + rng.Float64()*0.05 // 0.1-0.15 (more streamlined)
	}

	// Calculate weight: Length³ * Density * Shape Factor
	// 1025 kg/m³ is approximate density of fish tissue
	densityFactor := 1025.0
	weightKg := lengthMeters * lengthMeters * lengthMeters * densityFactor * bodyShapeFactor

	// Add minor random variation (±5%)
	variation := 0.95 + rng.Float64()*0.1 // 0.95-1.05
	weightKg *= variation

	// Round to 3 decimal places
	weightKg = math.Round(weightKg*1000) / 1000

	// Generate random catch chance within the range for this rarity
	catchChance := catchChanceMin + rng.Float64()*(catchChanceMax-catchChanceMin)
	catchChance = math.Round(catchChance*10) / 10 // Round to 1 decimal place

	// Calculate fish value based on size and rarity
	// Base value = size in meters * 10 * rarity multiplier
	fishValue := math.Round(lengthMeters*10*rarityMultiplier*100) / 100

	// Override AI values with our random values
	fishData.Rarity = rarity
	fishData.Size = lengthMeters
	fishData.CatchChance = catchChance
	fishData.Value = fishValue

	logFish("Size: %.3f meters | Weight: %.3f kg", lengthMeters, weightKg)
	logFish("Value: $%.2f", fishValue)

	// Game mechanics information
	logFish("Favorite Weather: %s | Catch Chance: %.1f%%",
		fishData.FavoriteWeather, catchChance)

	// Origin information
	logFish("Existence Reason: %s", fishData.ExistenceReason)
	if fishData.OriginContext != "" {
		logFish("Origin Context: %s", fishData.OriginContext)
	}

	// Game effect
	logFish("Effect: %s", fishData.Effect)
	logFish(strings.Repeat("=", 80))

	// Save the fish to the database (if DB is available)
	if m.db != nil {
		// Choose a random region for the fish
		regionIndex := int(time.Now().UnixNano()) % len(m.regions)
		regionID := m.regions[regionIndex].ID

		// Create current time once to ensure consistent timestamps
		timestamp := time.Now()

		// Initialize the usedArticles array
		usedArticles := make([]map[string]interface{}, 0)

		// Add the primary news article
		if m.lastNewsData != nil {
			usedArticles = append(usedArticles, map[string]interface{}{
				"headline":  m.lastNewsData.Headline,
				"source":    m.lastNewsData.Source,
				"url":       m.lastNewsData.URL,
				"category":  m.lastNewsData.Category,
				"sentiment": m.lastNewsData.Sentiment,
				"keywords":  m.lastNewsData.Keywords,
				"published": m.lastNewsData.PublishedAt,
				"used_at":   timestamp,
				"is_merged": false,
			})
		}

		// Add all merged news items to the used articles
		if m.mergedNewsItems != nil && len(m.mergedNewsItems) > 0 {
			for _, news := range m.mergedNewsItems {
				if news != nil {
					usedArticles = append(usedArticles, map[string]interface{}{
						"headline":  news.Headline,
						"source":    news.Source,
						"url":       news.URL,
						"category":  news.Category,
						"sentiment": news.Sentiment,
						"keywords":  news.Keywords,
						"published": news.PublishedAt,
						"used_at":   timestamp,
						"is_merged": true,
					})
				}
			}
		}

		// Log the number of articles being saved
		logFish("Saving fish with %d used articles", len(usedArticles))

		// Create a structured fish data object that matches the MongoDB schema exactly
		fish := map[string]interface{}{
			"name":              fishData.Name,
			"description":       fishData.Description,
			"rarity":            rarity,
			"length":            lengthMeters,
			"weight":            weightKg,
			"color":             fishData.Color,
			"habitat":           fishData.Habitat,
			"diet":              fishData.Diet,
			"generated_at":      timestamp,
			"is_ai_generated":   true,
			"data_source":       "gemini-ai",
			"region_id":         regionID,
			"generation_reason": reason,
			"favorite_weather":  fishData.FavoriteWeather,
			"catch_chance":      catchChance,
			"existence_reason":  fishData.ExistenceReason,
			"used_articles":     usedArticles,
			"stat_effects": []map[string]interface{}{
				{
					"effect_type":  "environment",
					"modifier":     fishData.CatchChance / 100.0,
					"description":  fishData.Effect,
					"weather_type": fishData.FavoriteWeather,
				},
				{
					"effect_type": "player",
					"modifier":    fishData.Value / 1000.0,
					"description": "Affects player abilities based on fish value",
				},
			},
		}

		// Call the database method to save the fish
		if saveFishMethod, ok := m.db.(interface {
			SaveFishData(context.Context, interface{}) error
		}); ok {
			err := saveFishMethod.SaveFishData(ctx, fish)
			if err != nil {
				logError("Error saving generated fish: %v", err)
				// Log more details for debugging
				logError("Fish data: %+v", fish)
			} else {
				logFish("Fish saved to database: %s (ID: %s)", fishData.Name, regionID)
			}
		} else {
			logError("Database client doesn't support SaveFishData method")
		}
	}

	// Clear the merged news data to avoid reusing it - moved here to ensure all news items are saved first
	m.mergedNewsItems = nil
	m.mergedNewsItem = nil // For backward compatibility

	// After successful generation, mark all used news as used
	newsID := m.lastNewsData.Source + ":" + m.lastNewsData.Headline
	m.usedNewsIDs[newsID] = true
	logNews("Marked news item as used for generation: %s", truncateString(m.lastNewsData.Headline, 50))

	// Also mark all merged news items as used
	for _, mergedNews := range m.mergedNewsItems {
		if mergedNews != nil {
			mergedNewsID := mergedNews.Source + ":" + mergedNews.Headline
			m.usedNewsIDs[mergedNewsID] = true
			logNews("Marked merged news item as used for generation: %s", truncateString(mergedNews.Headline, 50))
		}
	}

	// Save the updated used news IDs to the database
	go m.savePersistentState(ctx)
}

// findUnusedNewsItem finds a news item that hasn't been used for fish generation yet
func (m *DataManager) findUnusedNewsItem(ctx context.Context) (*NewsItem, error) {
	// Get several recent news items
	recentNews, err := m.db.GetRecentNewsData(ctx, 10)
	if err != nil || len(recentNews) == 0 {
		return nil, fmt.Errorf("no news data available: %v", err)
	}

	// Look for one we haven't used yet
	for _, newsItem := range recentNews {
		newsID := newsItem.Source + ":" + newsItem.Headline
		if !m.usedNewsIDs[newsID] {
			return newsItem, nil
		}
	}

	return nil, fmt.Errorf("all available news items have been used")
}

// generateFishWithLock is a wrapper that handles locking for automated fish generation
// (used by collectNewsData)
func (m *DataManager) generateFishWithLock(ctx context.Context, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateFishFromData(ctx, reason)
}

// GenerateFishFromContext is the public method that the service calls to generate a fish
func (m *DataManager) GenerateFishFromContext(ctx context.Context, reason string) error {
	// Check for required data availability first
	m.mu.Lock()

	// Check if we have news data
	if m.lastNewsData == nil {
		// Try to fetch the most recent news if available
		if m.db != nil {
			newsData, err := m.db.GetRecentNewsData(ctx, 1)
			if err != nil || len(newsData) == 0 {
				m.mu.Unlock()
				logError("No news data available for fish generation: %v", err)
				return fmt.Errorf("no news data available for fish generation")
			}
			m.lastNewsData = newsData[0]
		} else {
			m.mu.Unlock()
			logError("No news data and no database available")
			return fmt.Errorf("no news data available and no database connection")
		}
	}

	// Check if we have the minimum required context data
	if !m.dataReady {
		m.mu.Unlock()
		logError("Waiting for initial data collection to complete...")
		return fmt.Errorf("data not ready yet, initial collection in progress")
	}
	m.mu.Unlock()

	// Instead of generating directly, add to queue
	logFish("Manually queueing generation with reason: %s", reason)
	m.queueFishGeneration(ctx, reason)
	return nil
}

// Stop stops all data collection
func (m *DataManager) Stop() {
	m.mu.Lock()
	for _, cancel := range m.cancelFuncs {
		cancel()
	}
	// Clear cancel functions
	m.cancelFuncs = make([]context.CancelFunc, 0)
	m.mu.Unlock()

	// Wait for all goroutines to complete
	log.Println("Waiting for all data collection goroutines to complete...")
	m.wg.Wait()

	log.Println("Data Manager stopped")
}

// Helper function to truncate a string to a certain length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetCollectors returns all data collectors managed by this data manager
func (m *DataManager) GetCollectors() []DataCollector {
	collectors := []DataCollector{
		m.weatherCollector,
		m.bitcoinCollector,
		m.goldCollector,
		m.newsCollector,
	}
	return collectors
}

// checkPendingNewsForGeneration checks for unused news in the same category and queues them for generation
func (m *DataManager) checkPendingNewsForGeneration(ctx context.Context) {
	// Use a separate context to prevent cancellation issues
	safeCtx := context.Background()

	// Cooldown must be over to process more news
	currentTime := time.Now()
	m.mu.Lock()
	timeSinceLastGeneration := currentTime.Sub(m.lastFishGeneration)
	if !m.lastFishGeneration.IsZero() && timeSinceLastGeneration < m.generationCooldown {
		m.mu.Unlock()
		// Don't log when skipping due to cooldown - this prevents log spam
		return // Still on cooldown
	}

	// Get recent news from database grouped by category to find related articles
	m.mu.Unlock()
	recentNews, err := m.db.GetRecentNewsData(safeCtx, 100) // Check more news items to find matching categories
	if err != nil || len(recentNews) == 0 {
		logNews("No recent news found, skipping fish generation")
		return
	}

	// Find groups of unused news with the same category
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have any unused news at all
	hasUnusedNews := false
	for _, news := range recentNews {
		newsID := news.Source + ":" + news.Headline
		if !m.usedNewsIDs[newsID] {
			hasUnusedNews = true
			break
		}
	}

	if !hasUnusedNews {
		logNews("No new unused news available, skipping fish generation")
		return
	}

	// Group news by category
	newsByCategory := make(map[string][]*NewsItem)
	for _, news := range recentNews {
		newsID := news.Source + ":" + news.Headline
		if !m.usedNewsIDs[newsID] {
			category := news.Category
			newsByCategory[category] = append(newsByCategory[category], news)
		}
	}

	// Log the news category distribution
	logNews("Found unused news articles in %d different categories", len(newsByCategory))
	for category, items := range newsByCategory {
		logNews("- Category '%s': %d unused articles", category, len(items))
	}

	// Find the category with the most unused articles and prioritize it
	var bestCategory string
	var maxArticles int
	for category, items := range newsByCategory {
		if len(items) >= 2 && len(items) > maxArticles {
			maxArticles = len(items)
			bestCategory = category
		}
	}

	// If we found a category with multiple articles, use it
	if bestCategory != "" && maxArticles >= 2 {
		// Get up to 3 news items in this best category
		newsItems := newsByCategory[bestCategory]
		maxToUse := 3
		if len(newsItems) < maxToUse {
			maxToUse = len(newsItems)
		}

		selectedNews := newsItems[:maxToUse]

		// Store all the news IDs we'll be using
		var usedNewsIDs []string
		var headlines []string
		for _, news := range selectedNews {
			newsID := news.Source + ":" + news.Headline
			usedNewsIDs = append(usedNewsIDs, newsID)
			headlines = append(headlines, truncateString(news.Headline, 20))
		}

		// Create merged reason with up to 3 headlines
		reason := fmt.Sprintf("merged news [%s]: %s",
			bestCategory,
			strings.Join(headlines, " + "))

		// Queue generation with these news items
		logNews("Selected category '%s' with %d articles for themed fish generation",
			bestCategory, len(selectedNews))

		// Create a request and add it to the queue
		req := GenerationRequest{
			Ctx:     safeCtx,
			Reason:  reason,
			AddedAt: time.Now(),
		}

		// Store primary news item
		m.lastNewsData = selectedNews[0]

		// Store secondary news items as an array in the mergedNewsItems field
		m.mergedNewsItems = selectedNews[1:]

		m.generationQueue = append(m.generationQueue, req)

		// Mark all selected news items as used
		for _, newsID := range usedNewsIDs {
			m.usedNewsIDs[newsID] = true
			logNews("Marked as used: %s", newsID)
		}

		// Save to database in background
		go m.savePersistentState(safeCtx)
		return
	}

	// If we didn't find a best category with multiple articles, fall back to single news item
	logNews("No category with multiple articles found, falling back to single news item")
	for _, newsItems := range newsByCategory {
		if len(newsItems) > 0 {
			news := newsItems[0]
			newsID := news.Source + ":" + news.Headline

			// Queue generation with this individual news item
			reason := fmt.Sprintf("news [%s]: %s",
				news.Category,
				truncateString(news.Headline, 40))

			logNews("Using single news item: %s", truncateString(news.Headline, 40))

			// Create a request and add it to the queue
			req := GenerationRequest{
				Ctx:     safeCtx,
				Reason:  reason,
				AddedAt: time.Now(),
			}

			// Store news for generation
			m.lastNewsData = news
			m.mergedNewsItems = nil
			m.generationQueue = append(m.generationQueue, req)

			// Mark as used
			m.usedNewsIDs[newsID] = true

			// Save to database in background
			go m.savePersistentState(safeCtx)
			break
		}
	}
}

// loadPersistentState loads used news IDs and queued generation requests from the database
func (m *DataManager) loadPersistentState(ctx context.Context) {
	if m.db == nil {
		logError("Cannot load persistent state: database not available")
		return
	}

	// Load used news IDs
	usedIDs, err := m.db.GetUsedNewsIDs(ctx)
	if err != nil {
		logError("Failed to load used news IDs: %v", err)
	} else if len(usedIDs) > 0 {
		m.mu.Lock()
		m.usedNewsIDs = usedIDs
		m.mu.Unlock()
		logNews("Loaded %d used news IDs from database", len(usedIDs))
	}

	// Load generation queue
	queue, err := m.db.GetGenerationQueue(ctx)
	if err != nil {
		logError("Failed to load generation queue: %v", err)
	} else if len(queue) > 0 {
		m.mu.Lock()
		m.generationQueue = queue
		m.mu.Unlock()
		logFish("Loaded %d generation requests from database", len(queue))
	}
}

// savePersistentState saves current state to the database for crash recovery
func (m *DataManager) savePersistentState(ctx context.Context) {
	if m.db == nil {
		return
	}

	// Create a local copy of data to prevent race conditions
	m.mu.Lock()
	usedIDsCopy := make(map[string]bool)
	for k, v := range m.usedNewsIDs {
		usedIDsCopy[k] = v
	}

	queueCopy := make([]GenerationRequest, len(m.generationQueue))
	copy(queueCopy, m.generationQueue)
	m.mu.Unlock()

	// Create a timeout context for database operations
	saveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Save used news IDs
	if err := m.db.SaveUsedNewsIDs(saveCtx, usedIDsCopy); err != nil {
		logError("Failed to save used news IDs: %v", err)
	}

	// Save generation queue
	if err := m.db.SaveGenerationQueue(saveCtx, queueCopy); err != nil {
		logError("Failed to save generation queue: %v", err)
	}
}

// startQueueProcessor starts the generation queue processor in a controlled manner
func (m *DataManager) startQueueProcessor(ctx context.Context) {
	m.mu.Lock()
	if m.queueProcessRunning {
		m.mu.Unlock()
		return // Already running
	}
	m.queueProcessRunning = true
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			m.queueProcessRunning = false
			m.mu.Unlock()
		}()

		m.processGenerationQueue()
	}()
}

// queueFishGeneration adds a fish generation request to the queue
func (m *DataManager) queueFishGeneration(ctx context.Context, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add request to queue
	request := GenerationRequest{
		Reason:  reason,
		AddedAt: time.Now(),
	}

	m.generationQueue = append(m.generationQueue, request)
	logFish("Added fish generation request to queue: %s (queue size: %d)",
		reason, len(m.generationQueue))

	// Persist the updated queue
	go m.savePersistentState(ctx)

	// Start the queue processor if it's not already running
	if !m.queueProcessRunning {
		m.startQueueProcessor(ctx)
	}
}

// processGenerationQueue processes the fish generation queue with proper timing
func (m *DataManager) processGenerationQueue() {
	defer func() {
		m.mu.Lock()
		m.queueProcessRunning = false
		m.mu.Unlock()

		// Recover from panics to prevent crashing the app
		if r := recover(); r != nil {
			logError("Recovered from panic in queue processor: %v", r)
		}
	}()

	for {
		// Check cooldown status first
		m.mu.Lock()
		currentTime := time.Now()
		timeUntilReady := time.Duration(0)
		if !m.lastFishGeneration.IsZero() {
			elapsed := currentTime.Sub(m.lastFishGeneration)
			if elapsed < m.generationCooldown {
				timeUntilReady = m.generationCooldown - elapsed
			}
		}
		m.mu.Unlock()

		// If we're on cooldown, just wait once without spamming logs
		if timeUntilReady > 0 {
			// Need to wait for cooldown to expire
			waitTime := timeUntilReady

			// Ensure wait time is never negative
			if waitTime < 0 {
				waitTime = 0
			}

			logFish("Waiting %v for cooldown before processing next request",
				waitTime.Round(time.Second))
			if waitTime > 0 {
				time.Sleep(waitTime)
			}
			// After waiting, continue to processing
		}

		// Only check for news if we're not on cooldown anymore
		m.mu.Lock()
		hasQueueItems := len(m.generationQueue) > 0
		m.mu.Unlock()

		// If queue is empty, try to find news to process
		if !hasQueueItems {
			// Check if we need to process any pending news items
			m.checkPendingNewsForGeneration(context.Background())

			// Check again if we added anything to the queue
			m.mu.Lock()
			hasQueueItems = len(m.generationQueue) > 0
			m.mu.Unlock()

			// If still no items, don't exit but wait and check again for new news items
			if !hasQueueItems {
				logFish("No generation requests found, waiting for 5 minutes before checking again")
				time.Sleep(5 * time.Minute)
				continue
			}
		}

		// Process a queue item
		m.mu.Lock()
		request := m.generationQueue[0]
		m.generationQueue = m.generationQueue[1:]
		queueLen := len(m.generationQueue)
		m.mu.Unlock()

		// Update the persistent queue state after removing an item
		ctx := context.Background()
		if request.Ctx != nil {
			ctx = request.Ctx
		}
		go m.savePersistentState(ctx)

		// Process the request (with proper locking inside the method)
		logFish("Processing queued fish generation: %s (remaining in queue: %d)",
			request.Reason, queueLen)
		m.generateFishWithLock(ctx, request.Reason)
	}
}

// checkForFishToTranslate checks for recently generated fish that need translation
func (m *DataManager) checkForFishToTranslate(ctx context.Context) {
	// Skip if translation is disabled or translator is not initialized
	if !m.settings.EnableTranslation || m.translatorClient == nil {
		return
	}

	// Check cooldown
	m.mu.Lock()
	currentTime := time.Now()
	timeSinceLastTranslation := currentTime.Sub(m.lastTranslation)

	if !m.lastTranslation.IsZero() && timeSinceLastTranslation < m.translationCooldown {
		m.mu.Unlock()
		return // Still on cooldown
	}
	m.mu.Unlock()

	// Find untranslated fish
	untranslatedFish, err := m.db.GetUntranslatedFish(ctx, 1)
	if err != nil {
		logError("Error finding untranslated fish: %v", err)
		return
	}

	if len(untranslatedFish) == 0 {
		// No untranslated fish found, nothing to do
		return
	}

	// Get the first untranslated fish
	fishToTranslate := untranslatedFish[0]

	// Pre-validate all string fields to ensure valid UTF-8
	// This helps identify problematic fields before we try to use them
	for key, value := range fishToTranslate {
		if strValue, ok := value.(string); ok {
			if !utf8.ValidString(strValue) {
				logError("Found invalid UTF-8 in field '%s', attempting to fix", key)
				fishToTranslate[key] = SanitizeUTF8(strValue)
			}
		}
	}

	// Start translation
	logTranslation("Translating fish: %s", fishToTranslate["name"])

	// Extract fields to translate - add defensive extraction with defaults
	fieldsToTranslate := TranslationFields{
		Name:            extractStringFieldSafely(fishToTranslate, "name", "Unnamed Fish"),
		Description:     extractStringFieldSafely(fishToTranslate, "description", "No description available"),
		Color:           extractStringFieldSafely(fishToTranslate, "color", "Unknown color"),
		Diet:            extractStringFieldSafely(fishToTranslate, "diet", "Unknown diet"),
		Habitat:         extractStringFieldSafely(fishToTranslate, "habitat", "Unknown habitat"),
		FavoriteWeather: extractStringFieldSafely(fishToTranslate, "favorite_weather", "Unknown weather"),
		ExistenceReason: extractStringFieldSafely(fishToTranslate, "existence_reason", "Unknown reason"),
		Effect:          "",
		PlayerEffect:    "Affects player abilities",
	}

	// Extract stat effects for translation with more careful handling
	if statEffectsInterface, ok := fishToTranslate["stat_effects"]; ok && statEffectsInterface != nil {
		if statEffects, ok := statEffectsInterface.([]interface{}); ok && len(statEffects) > 0 {
			for _, effectInterface := range statEffects {
				if effect, ok := effectInterface.(map[string]interface{}); ok {
					effectType, _ := effect["effect_type"].(string)
					if effectType == "environment" {
						description, ok := effect["description"].(string)
						if ok && utf8.ValidString(description) {
							fieldsToTranslate.Effect = description
						} else if ok {
							fieldsToTranslate.Effect = SanitizeUTF8(description)
						}

						weatherType, ok := effect["weather_type"].(string)
						if ok && utf8.ValidString(weatherType) {
							fieldsToTranslate.FavoriteWeather = weatherType
						} else if ok {
							fieldsToTranslate.FavoriteWeather = SanitizeUTF8(weatherType)
						}
					} else if effectType == "player" {
						description, ok := effect["description"].(string)
						if ok && utf8.ValidString(description) {
							fieldsToTranslate.PlayerEffect = description
						} else if ok {
							fieldsToTranslate.PlayerEffect = SanitizeUTF8(description)
						}
					}
				}
			}
		}
	}

	// Set the cooldown before translation to prevent multiple translations at once
	m.mu.Lock()
	m.lastTranslation = currentTime
	m.mu.Unlock()

	// Double safety check before translation
	if m.translatorClient == nil {
		logError("Translation client became nil after check, aborting translation")
		return
	}

	// Translate the fish
	translatedFields, err := m.translatorClient.TranslateFish(ctx, fieldsToTranslate)
	if err != nil {
		logError("Error translating fish: %v", err)
		return
	}

	// Create a new map for the translation instead of modifying the original
	translatedFish := make(map[string]interface{})

	// Only copy essential fields that we know are safe
	safeFields := []string{"_id", "name", "description", "rarity", "length", "weight",
		"region_id", "data_source", "generated_at", "is_ai_generated", "generation_reason"}

	for _, field := range safeFields {
		if val, ok := fishToTranslate[field]; ok {
			translatedFish[field] = val
		}
	}

	// Explicitly set translated fields as new fields with _vi suffix
	translatedFish["name_vi"] = translatedFields.Name
	translatedFish["description_vi"] = translatedFields.Description
	translatedFish["color_vi"] = translatedFields.Color
	translatedFish["diet_vi"] = translatedFields.Diet
	translatedFish["existence_reason_vi"] = translatedFields.ExistenceReason
	translatedFish["is_translated"] = true
	translatedFish["translated_at"] = time.Now()

	// Set habitat if it exists
	if translatedFields.Habitat != "" {
		translatedFish["habitat_vi"] = translatedFields.Habitat
	}

	// Handle stat effects with extra care
	if statEffectsInterface, ok := fishToTranslate["stat_effects"]; ok && statEffectsInterface != nil {
		if statEffects, ok := statEffectsInterface.([]interface{}); ok {
			translatedStatEffects := make([]map[string]interface{}, 0, len(statEffects))

			for _, effectInterface := range statEffects {
				if effect, ok := effectInterface.(map[string]interface{}); ok {
					// Create a new clean effect map
					translatedEffect := make(map[string]interface{})

					// Copy only essential fields we know are safe
					safeEffectFields := []string{"effect_type", "modifier", "stat", "value"}
					for _, field := range safeEffectFields {
						if val, ok := effect[field]; ok {
							translatedEffect[field] = val
						}
					}

					// Add translated fields
					effectType, _ := effect["effect_type"].(string)
					if effectType == "environment" {
						translatedEffect["description_vi"] = translatedFields.Effect
						translatedEffect["weather_type_vi"] = translatedFields.FavoriteWeather
					} else if effectType == "player" {
						translatedEffect["description_vi"] = translatedFields.PlayerEffect
					}

					translatedStatEffects = append(translatedStatEffects, translatedEffect)
				}
			}

			translatedFish["stat_effects"] = translatedStatEffects
		}
	}

	translatedFish["favorite_weather_vi"] = translatedFields.FavoriteWeather

	// Save the translated fish back to the database
	fishID := fishToTranslate["_id"]
	err = m.db.UpdateFishWithTranslation(ctx, fishID, translatedFish)
	if err != nil {
		logError("Error saving translated fish: %v", err)
		return
	}

	logTranslation("Successfully translated fish '%s' to Vietnamese", fishToTranslate["name"])
}

// Helper function to safely extract string fields with a default value
func extractStringFieldSafely(data map[string]interface{}, field string, defaultValue string) string {
	if value, ok := data[field]; ok && value != nil {
		if strValue, ok := value.(string); ok {
			if utf8.ValidString(strValue) {
				return strValue
			}
			// If not valid UTF-8, sanitize it
			return SanitizeUTF8(strValue)
		}
	}
	return defaultValue
}
