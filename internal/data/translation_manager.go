package data

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// logTranslate logs translation-related messages with cyan color
func logTranslate(format string, v ...interface{}) {
	log.Printf(logColorCyan+"[TRANSLATE] "+format+logColorReset, v...)
}

// TranslationSettings holds configuration for translation
type TranslationSettings struct {
	Enabled  bool          // Whether translation is enabled
	Interval time.Duration // How often to check for untranslated fish
	ApiKey   string        // API key for Gemini
}

// DatabaseTranslationClient is an interface for database operations related to translation
type DatabaseTranslationClient interface {
	GetFishByID(ctx context.Context, id string) (map[string]interface{}, error)
	SaveTranslatedFish(ctx context.Context, translatedFish *TranslatedFish) error
	GetUntranslatedFishIDs(ctx context.Context, limit int) ([]string, error)
}

// TranslationManager handles the translation of fish content to Vietnamese
type TranslationManager struct {
	settings         TranslationSettings
	db               DatabaseTranslationClient
	translatorClient *TranslatorClient
	cancelFuncs      []context.CancelFunc
	mu               sync.Mutex
	wg               sync.WaitGroup
	isRunning        bool
}

// NewTranslationManager creates a new translation manager
func NewTranslationManager(settings TranslationSettings, db DatabaseTranslationClient) *TranslationManager {
	return &TranslationManager{
		settings:         settings,
		db:               db,
		translatorClient: NewTranslatorClient(settings.ApiKey),
		cancelFuncs:      make([]context.CancelFunc, 0),
		isRunning:        false,
	}
}

// Start begins the translation process
func (t *TranslationManager) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.isRunning {
		t.mu.Unlock()
		return fmt.Errorf("translation manager is already running")
	}
	t.isRunning = true
	t.mu.Unlock()

	// Check if translation is enabled
	if !t.settings.Enabled {
		logTranslate("Translation feature is disabled. Set ENABLE_TRANSLATION=1 to enable")
		return nil
	}

	// Create a cancellable context
	baseCtx, cancel := context.WithCancel(ctx)
	t.cancelFuncs = append(t.cancelFuncs, cancel)

	// Start the translation ticker
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		ticker := time.NewTicker(t.settings.Interval)
		defer ticker.Stop()

		logTranslate("Translation service started with interval: %v", t.settings.Interval)

		// Run translation immediately on startup
		t.translateNextFish(baseCtx)

		for {
			select {
			case <-ticker.C:
				t.translateNextFish(baseCtx)
			case <-baseCtx.Done():
				logTranslate("Translation service stopped")
				return
			}
		}
	}()

	logTranslate("Translation Manager started successfully")
	return nil
}

// translateNextFish finds and translates the next untranslated fish
func (t *TranslationManager) translateNextFish(ctx context.Context) {
	// Get IDs of untranslated fish (limit to 1)
	untranslatedIDs, err := t.db.GetUntranslatedFishIDs(ctx, 1)
	if err != nil {
		logError("Failed to get untranslated fish IDs: %v", err)
		return
	}

	if len(untranslatedIDs) == 0 {
		logTranslate("No untranslated fish found")
		return
	}

	// Get the first untranslated fish
	fishID := untranslatedIDs[0]
	logTranslate("Translating fish with ID: %s", fishID)

	// Get fish data from database
	fishData, err := t.db.GetFishByID(ctx, fishID)
	if err != nil {
		logError("Failed to get fish data: %v", err)
		return
	}

	// Extract fields to translate
	fields := TranslationFields{
		Name:            extractStringField(fishData, "name"),
		Description:     extractStringField(fishData, "description"),
		Appearance:      extractStringField(fishData, "appearance"),
		Color:           extractStringField(fishData, "color"),
		Diet:            extractStringField(fishData, "diet"),
		Habitat:         extractStringField(fishData, "habitat"),
		Effect:          extractStringField(fishData, "effect"),
		FavoriteWeather: extractStringField(fishData, "favorite_weather"),
		ExistenceReason: extractStringField(fishData, "existence_reason"),
	}

	// Create context with API key
	translationCtx := context.WithValue(ctx, "gemini_api_key", t.settings.ApiKey)

	// Translate fish content
	translatedFish, err := t.translatorClient.TranslateFish(translationCtx, fishID, fields)
	if err != nil {
		logError("Translation failed: %v", err)
		return
	}

	// Save translated fish to database
	err = t.db.SaveTranslatedFish(ctx, translatedFish)
	if err != nil {
		logError("Failed to save translated fish: %v", err)
		return
	}

	logTranslate("Successfully translated fish: %s -> %s", fields.Name, translatedFish.Name)
}

// Stop stops the translation process
func (t *TranslationManager) Stop() {
	t.mu.Lock()
	for _, cancel := range t.cancelFuncs {
		cancel()
	}
	t.cancelFuncs = make([]context.CancelFunc, 0)
	t.isRunning = false
	t.mu.Unlock()

	// Wait for all goroutines to complete
	t.wg.Wait()

	if t.translatorClient != nil {
		t.translatorClient.Close()
	}

	logTranslate("Translation Manager stopped")
}

// Helper function to extract string fields from fish data
func extractStringField(fishData map[string]interface{}, field string) string {
	if value, ok := fishData[field]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

// LoadTranslationSettings loads translation settings from environment variables
func LoadTranslationSettings() TranslationSettings {
	enabled := os.Getenv("ENABLE_TRANSLATION") == "1"

	// Get interval in minutes (default: 2 minutes)
	intervalStr := os.Getenv("TRANSLATION_INTERVAL")
	intervalMin := 2 // Default 2 minutes
	if intervalStr != "" {
		if val, err := strconv.Atoi(intervalStr); err == nil && val > 0 {
			intervalMin = val
		}
	}

	return TranslationSettings{
		Enabled:  enabled,
		Interval: time.Duration(intervalMin) * time.Minute,
		ApiKey:   os.Getenv("GEMINI_API_KEY"),
	}
}
