package fish

import (
	"context"
	"log"
	"time"
)

// WeatherInfo contains basic weather data for fish generation
type WeatherInfo struct {
	Condition string  `json:"condition"`
	TempC     float64 `json:"temp_c"`
}

// FishGenerationService provides a simplified interface for generating fish
type FishGenerationService struct {
	service        *Service
	apiKey         string
	storageAdapter StorageAdapter
	dataManager    interface {
		GenerateFishFromContext(ctx context.Context, reason string) error
	}
}

// NewFishGenerationService creates a new fish generation service
func NewFishGenerationService(apiKey string, storageAdapter StorageAdapter, dataManager interface {
	GenerateFishFromContext(ctx context.Context, reason string) error
}) *FishGenerationService {
	// Create service options
	options := ServiceOptions{
		GeminiAPIKey:   apiKey,
		UseAI:          apiKey != "",
		StorageAdapter: storageAdapter,
	}

	// Create the underlying service
	service := NewServiceSimple(options)

	return &FishGenerationService{
		service:        service,
		apiKey:         apiKey,
		storageAdapter: storageAdapter,
		dataManager:    dataManager,
	}
}

// Run starts the fish generation service in production mode
// This will generate fish at regular intervals
func (s *FishGenerationService) Run(ctx context.Context) {
	// Start the underlying service
	if err := s.service.Start(ctx); err != nil {
		log.Printf("Error starting fish generation service: %v", err)
		log.Fatalf("Fish generation service failed to start. Stopping application.")
		return
	}

	// Set up ticker for fish generation (30 minutes in production)
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	log.Println("Fish generation service started in production mode")

	// Listen for ticks to generate fish
	for {
		select {
		case <-ticker.C:
			s.generateFish(ctx)
		case <-ctx.Done():
			log.Println("Fish generation service stopped")
			return
		}
	}
}

// RunTest starts the fish generation service in test mode
// This will generate fish at shorter intervals for testing
func (s *FishGenerationService) RunTest(ctx context.Context) {
	// Start the underlying service
	if err := s.service.Start(ctx); err != nil {
		log.Printf("Error starting fish generation service: %v", err)
		log.Fatalf("Fish generation service failed to start. Stopping application.")
		return
	}

	// Set up ticker for fish generation (5 seconds in test mode)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Fish generation service started in test mode")

	// Listen for ticks to generate fish
	for {
		select {
		case <-ticker.C:
			s.generateFish(ctx)
		case <-ctx.Done():
			log.Println("Fish generation service stopped")
			return
		}
	}
}

// generateFish calls the data manager to generate a fish using AI with current context
func (s *FishGenerationService) generateFish(ctx context.Context) {
	// Get current time for generation
	timestamp := time.Now()
	timeString := timestamp.Format("2006-01-02 15:04:05")

	// Generate a reason for this fish generation
	reason := "scheduled generation at " + timeString

	log.Println("Initiating AI fish generation with current data context")

	// Call the data manager to generate a fish using AI with the current context
	err := s.dataManager.GenerateFishFromContext(ctx, reason)

	// If the generation fails, the data manager should already log and exit
	// This is just an additional safeguard
	if err != nil {
		log.Printf("Error in fish generation service: %v", err)
		log.Fatalf("Fish generation service failed. Stopping application.")
	}
}

// SubscribeToFishGeneration allows subscribing to fish generation events
func (s *FishGenerationService) SubscribeToFishGeneration(ctx context.Context, fishChan chan<- *Fish) {
	s.service.SubscribeToFishCreation(ctx, fishChan)
}
