package fish

import (
	"context"
	"fmt"
)

// StorageWrapper wraps a storage adapter that accepts interface{} and makes it compatible
// with the fish.StorageAdapter interface that requires *Fish
type StorageWrapper struct {
	adapter interface {
		SaveFishData(ctx context.Context, fish interface{}) error
		GetDailyFishCount(ctx context.Context) (int, error)
		GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*Fish, error)
		GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*Fish, error)
		GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*Fish, error)
	}
}

// NewStorageWrapper creates a new storage wrapper
func NewStorageWrapper(adapter interface{}) (*StorageWrapper, error) {
	// Check if the adapter implements the necessary methods
	if a, ok := adapter.(interface {
		SaveFishData(ctx context.Context, fish interface{}) error
		GetDailyFishCount(ctx context.Context) (int, error)
		GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*Fish, error)
		GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*Fish, error)
		GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*Fish, error)
	}); ok {
		return &StorageWrapper{adapter: a}, nil
	}

	return nil, fmt.Errorf("adapter does not implement required methods")
}

// SaveFishData converts the *Fish to interface{} and calls the underlying adapter
func (w *StorageWrapper) SaveFishData(ctx context.Context, fish *Fish) error {
	return w.adapter.SaveFishData(ctx, fish)
}

// GetDailyFishCount passes through to the underlying adapter
func (w *StorageWrapper) GetDailyFishCount(ctx context.Context) (int, error) {
	return w.adapter.GetDailyFishCount(ctx)
}

// GetSimilarFish passes through to the underlying adapter
func (w *StorageWrapper) GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*Fish, error) {
	return w.adapter.GetSimilarFish(ctx, dataSource, rarityLevel)
}

// GetFishByRegion passes through to the underlying adapter
func (w *StorageWrapper) GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*Fish, error) {
	return w.adapter.GetFishByRegion(ctx, regionID, limit)
}

// GetFishByDataSource passes through to the underlying adapter
func (w *StorageWrapper) GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*Fish, error) {
	return w.adapter.GetFishByDataSource(ctx, dataSource, limit)
}
