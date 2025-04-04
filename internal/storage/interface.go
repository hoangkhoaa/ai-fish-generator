package storage

import (
	"context"

	"fish-generate/internal/data"
	"fish-generate/internal/fish"
)

// StorageAdapter defines the interface for storage operations
type StorageAdapter interface {
	// Weather data operations
	SaveWeatherData(ctx context.Context, weatherInfo *data.WeatherInfo, regionID, cityID string) error
	GetRecentWeatherData(ctx context.Context, regionID string, limit int) ([]*data.WeatherInfo, error)

	// Price data operations
	SavePriceData(ctx context.Context, assetType string, price, volume, changePercent, volumeChange float64, source string) error
	GetRecentPriceData(ctx context.Context, assetType string, limit int) ([]map[string]interface{}, error)

	// News data operations
	SaveNewsData(ctx context.Context, newsItem *data.NewsItem) error
	GetRecentNewsData(ctx context.Context, limit int) ([]*data.NewsItem, error)

	// Fish data operations
	SaveFishData(ctx context.Context, fishItem interface{}) error
	GetDailyFishCount(ctx context.Context) (int, error)
	GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*fish.Fish, error)
	GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*fish.Fish, error)
	GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*fish.Fish, error)

	// Persistence operations for news and generation queue
	SaveUsedNewsIDs(ctx context.Context, usedIDs map[string]bool) error
	GetUsedNewsIDs(ctx context.Context) (map[string]bool, error)
	SaveGenerationQueue(ctx context.Context, queue []data.GenerationRequest) error
	GetGenerationQueue(ctx context.Context) ([]data.GenerationRequest, error)
}
