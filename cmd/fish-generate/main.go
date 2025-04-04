package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fish-generate/internal/api"
	"fish-generate/internal/config"
	"fish-generate/internal/data"
	"fish-generate/internal/fish"
	"fish-generate/internal/storage"
)

func main() {
	// Parse command line flags
	testMode := flag.Bool("test", false, "Run in test mode with shorter collection intervals")
	flag.Parse()

	// Load environment variables from .env file if it exists
	config.LoadEnv(".env")

	// Create configuration
	conf := config.NewConfig()

	// Initialize MongoDB if configured
	var mongoStorage *storage.MongoDB
	var storageAdapter storage.StorageAdapter

	if conf.MongoURI != "" {
		mongoURI := conf.GetMongoURI()
		mongoDB := conf.GetMongoDB()

		log.Printf("Connecting to MongoDB at %s...", mongoURI)
		var err error
		mongoStorage, err = storage.NewMongoDB(mongoURI, mongoDB)
		if err != nil {
			log.Printf("Warning: MongoDB connection failed: %v", err)
		} else {
			storageAdapter = storage.NewMongoDBAdapter(mongoStorage)
			log.Println("MongoDB connection established successfully")
			log.Printf("Daily fish generation limit: %d fish", fish.DailyFishLimit)
		}
	} else {
		log.Println("MongoDB not configured. Data will not be persisted.")
	}

	// Create context that is canceled when the program receives an interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		log.Println("Received signal, shutting down...")
		cancel()
	}()

	// Configure data collection intervals
	collectionSettings := data.CollectionSettings{
		WeatherInterval:    conf.GetWeatherInterval(),
		PriceInterval:      conf.GetPriceInterval(),
		NewsInterval:       conf.GetNewsInterval(),
		GenerationCooldown: conf.GetGenerationCooldown(),
		TestMode:           *testMode || conf.TestMode,
		GeminiApiKey:       conf.GeminiAPIKey,
	}

	// Create data manager
	dataManager := data.NewDataManager(
		collectionSettings,
		storageAdapter,
		conf.OpenWeatherKey,
		conf.NewsAPIKey,
		conf.MetalPriceKey,
		conf.GeminiAPIKey,
	)

	// Start data collection
	dataManager.Start(ctx)

	// Create fish generation service options
	serviceOpts := fish.ServiceOptions{
		GeminiAPIKey: conf.GeminiAPIKey,
		UseAI:        conf.UseAI,
		TestMode:     *testMode || conf.TestMode,
	}

	// Create the fish service
	fishService := fish.NewService(dataManager.GetCollectors(), serviceOpts)

	// Create a simplified fish generation service wrapper
	var fishGenService *fish.FishGenerationService
	if storageAdapter != nil {
		wrapper, err := fish.NewStorageWrapper(storageAdapter)
		if err != nil {
			log.Printf("Warning: could not create storage wrapper for fish generation service: %v", err)
			fishGenService = fish.NewFishGenerationService(conf.GeminiAPIKey, nil, dataManager)
		} else {
			fishGenService = fish.NewFishGenerationService(conf.GeminiAPIKey, wrapper, dataManager)
		}
	} else {
		fishGenService = fish.NewFishGenerationService(conf.GeminiAPIKey, nil, dataManager)
	}

	// Start the fish generation service
	if *testMode || conf.TestMode {
		log.Println("Running fish generation service in test mode")
		go fishGenService.RunTest(ctx)
	} else {
		log.Println("Running fish generation service in production mode")
		go fishGenService.Run(ctx)
	}

	// Initialize translation service if storage is available
	if storageAdapter != nil {
		// Load translation settings from config
		translationSettings := data.TranslationSettings{
			Enabled:  conf.EnableTranslation,
			Interval: time.Duration(conf.TranslationInterval) * time.Minute,
			ApiKey:   conf.GeminiAPIKey,
		}

		if translationSettings.Enabled {
			log.Println("Initializing translation service")
			translationManager := data.NewTranslationManager(translationSettings, storageAdapter)

			// Start translation service
			if err := translationManager.Start(ctx); err != nil {
				log.Printf("Error starting translation service: %v", err)
			} else {
				log.Printf("Translation service started with interval: %v", translationSettings.Interval)

				// Make sure to stop the translation service on shutdown
				defer translationManager.Stop()
			}
		} else {
			log.Println("Translation service is disabled. Set ENABLE_TRANSLATION=1 to enable")
		}
	}

	// Initialize and start the API server
	apiServer := api.NewServer(api.Config{
		Port:         "8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Storage:      storageAdapter,
		DataManager:  dataManager,
	})

	// Start the API server in a goroutine
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for context to be canceled (from signal handler)
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop the API server
	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Printf("Error during API server shutdown: %v", err)
	}

	// Stop the data manager
	dataManager.Stop()

	// Stop the fish service
	fishService.Stop(ctx)

	log.Println("Shutdown complete")
}

func printUsage() {
	fmt.Println("Fish Generator - A tool for generating random fish")
	fmt.Println("\nUsage:")
	fmt.Println("  fish-generate [command] [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  generate     Start the fish generation service")
	fmt.Println("  test         Run in test mode (faster fish generation)")
	fmt.Println("  config       Show current configuration")
	fmt.Println("\nOptions:")
	fmt.Println("  -help        Show this help message")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  GEMINI_API_KEY        API key for Gemini (required for AI generation)")
	fmt.Println("  USE_AI                Set to 'true' to enable AI-based generation")
	fmt.Println("  TEST_MODE             Set to 'true' to enable test mode")
	fmt.Println("  OPENWEATHER_API_KEY   API key for OpenWeather data")
	fmt.Println("  NEWSAPI_KEY           API key for News API")
	fmt.Println("  METALPRICE_API_KEY    API key for Metal Price API")
	fmt.Println("  MONGO_URI             MongoDB connection URI")
	fmt.Println("  MONGO_DB              MongoDB database name")
	fmt.Println("  MONGO_USER            MongoDB username")
	fmt.Println("  MONGO_PASSWORD        MongoDB password")
	fmt.Println("  WEATHER_INTERVAL      Weather collection interval in hours (default: 3)")
	fmt.Println("  PRICE_INTERVAL        Price collection interval in hours (default: 12)")
	fmt.Println("  NEWS_INTERVAL         News collection interval in hours (default: 0.5)")
	fmt.Println("  GENERATION_COOLDOWN   Minutes between fish generations (default: 15)")
}

// Helper function to mask API keys for display
func maskAPIKey(key string) string {
	if key == "" {
		return "Not Set"
	}

	if len(key) <= 8 {
		return "****" + key[len(key)-4:]
	}

	return key[:4] + "****" + key[len(key)-4:]
}

// Helper function to mask MongoDB URI for display
func maskURI(uri string) string {
	if uri == "" {
		return "Not Set"
	}

	return "mongodb://****"
}
