package storage

import (
	"context"
	"fish-generate/internal/data"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Define collection names
const (
	weatherCollection    = "weather"
	priceCollection      = "prices"
	newsCollection       = "news"
	fishCollection       = "fish"
	regionCollection     = "regions"
	statsCollection      = "stats"
	usedNewsCollection   = "used_news"
	queueCollection      = "generation_queue"
	translatedCollection = "translated_fish" // New collection for translated fish
)

// WeatherData represents a weather data document in MongoDB
type WeatherData struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	RegionID    string             `bson:"region_id"`
	CityID      string             `bson:"city_id"`
	Condition   string             `bson:"condition"`
	TempC       float64            `bson:"temp_c"`
	Humidity    float64            `bson:"humidity"`
	WindSpeed   float64            `bson:"wind_speed"`
	RainMM      float64            `bson:"rain_mm"`
	Pressure    float64            `bson:"pressure"`
	Clouds      int                `bson:"clouds"`
	Description string             `bson:"description"`
	Timestamp   time.Time          `bson:"timestamp"`
	Source      string             `bson:"source"`
}

// PriceData represents an asset price document in MongoDB
type PriceData struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	AssetType     string             `bson:"asset_type"` // "btc", "gold", "oil"
	Price         float64            `bson:"price"`
	Volume        float64            `bson:"volume"`
	ChangePercent float64            `bson:"change_percent"`
	VolumeChange  float64            `bson:"volume_change"`
	Timestamp     time.Time          `bson:"timestamp"`
	Source        string             `bson:"source"`
}

// NewsData represents a news item document in MongoDB
type NewsData struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Headline    string             `bson:"headline"`
	Content     string             `bson:"content"`
	Source      string             `bson:"source"`
	URL         string             `bson:"url"`
	PublishedAt time.Time          `bson:"published_at"`
	Sentiment   float64            `bson:"sentiment"`
	Keywords    []string           `bson:"keywords"`
	Timestamp   time.Time          `bson:"timestamp"`
}

// FishData represents a generated fish document in MongoDB
type FishData struct {
	ID               primitive.ObjectID       `bson:"_id,omitempty"`
	Name             string                   `bson:"name"`
	Description      string                   `bson:"description"`
	Rarity           string                   `bson:"rarity"`
	Length           float64                  `bson:"length"`
	Weight           float64                  `bson:"weight"`
	Color            string                   `bson:"color"`
	Habitat          string                   `bson:"habitat"`
	Diet             string                   `bson:"diet"`
	GeneratedAt      time.Time                `bson:"generated_at"`
	IsAIGenerated    bool                     `bson:"is_ai_generated"`
	DataSource       string                   `bson:"data_source"`
	RegionID         string                   `bson:"region_id,omitempty"`
	FavoriteWeather  string                   `bson:"favorite_weather"`
	CatchChance      float64                  `bson:"catch_chance"`
	ExistenceReason  string                   `bson:"existence_reason"`
	WeatherID        primitive.ObjectID       `bson:"weather_id,omitempty"`
	NewsID           primitive.ObjectID       `bson:"news_id,omitempty"`
	PriceIDs         []primitive.ObjectID     `bson:"price_ids,omitempty"`
	StatEffects      []map[string]interface{} `bson:"stat_effects,omitempty"`
	GenerationReason string                   `bson:"generation_reason,omitempty"`
	UsedArticles     []map[string]interface{} `bson:"used_articles,omitempty"`
}

// CollectionStats tracks statistics about each collection
type CollectionStats struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	CollectionName string             `bson:"collection_name"`
	TotalCount     int64              `bson:"total_count"`
	LastUpdated    time.Time          `bson:"last_updated"`
	Date           string             `bson:"date"`
	RecordType     string             `bson:"record_type"`
}

// FishLimitRecord tracks the number of fish generated per day
type FishLimitRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Date        string             `bson:"date"`
	Count       int                `bson:"count"`
	LastUpdated time.Time          `bson:"last_updated"`
	RecordType  string             `bson:"record_type"`
}

// UsedNewsRecord represents an already used news item
type UsedNewsRecord struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	NewsID     string             `bson:"news_id"`
	UsedAt     time.Time          `bson:"used_at"`
	RecordType string             `bson:"record_type"`
}

// QueuedGenerationRecord represents a queued fish generation request
type QueuedGenerationRecord struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Reason     string             `bson:"reason"`
	AddedAt    time.Time          `bson:"added_at"`
	Status     string             `bson:"status"` // "pending", "processing", "completed", "failed"
	RecordType string             `bson:"record_type"`
}

// TranslatedFishData represents a translated fish document in MongoDB
type TranslatedFishData struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	OriginalID      primitive.ObjectID `bson:"original_id"` // Reference to original fish
	Name            string             `bson:"name"`
	Description     string             `bson:"description"`
	Appearance      string             `bson:"appearance"`
	Color           string             `bson:"color"`
	Diet            string             `bson:"diet"`
	Habitat         string             `bson:"habitat"`
	Effect          string             `bson:"effect"`
	FavoriteWeather string             `bson:"favorite_weather"`
	ExistenceReason string             `bson:"existence_reason"`
	TranslatedAt    time.Time          `bson:"translated_at"`
	Language        string             `bson:"language"` // e.g., "vi" for Vietnamese
}

// MongoDB implements database operations using MongoDB
type MongoDB struct {
	client    *mongo.Client
	database  string
	connected bool
}

// NewMongoDB creates a new MongoDB client
func NewMongoDB(uri, database string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create client options
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the MongoDB to verify connection
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	log.Println("Connected to MongoDB successfully")

	db := &MongoDB{
		client:    client,
		database:  database,
		connected: true,
	}

	// Initialize collections
	if err := db.initializeCollections(ctx); err != nil {
		log.Printf("Warning: Failed to initialize collections: %v", err)
		// Continue anyway, as we'll try to create collections on-demand
	}

	return db, nil
}

// initializeCollections ensures that all required collections exist
func (m *MongoDB) initializeCollections(ctx context.Context) error {
	db := m.client.Database(m.database)

	// Get the list of existing collections
	names, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to list collections: %v", err)
	}

	// Check for each required collection
	requiredCollections := []string{
		weatherCollection,
		priceCollection,
		newsCollection,
		fishCollection,
		regionCollection,
		statsCollection,
		usedNewsCollection,
		queueCollection,
		translatedCollection, // Add translated fish collection
	}

	existingCollections := make(map[string]bool)
	for _, name := range names {
		existingCollections[name] = true
	}

	// Create collections that don't exist
	for _, collName := range requiredCollections {
		if !existingCollections[collName] {
			log.Printf("Creating collection '%s'", collName)
			if err := db.CreateCollection(ctx, collName); err != nil {
				return fmt.Errorf("failed to create collection '%s': %v", collName, err)
			}

			// Create indexes for the new collections
			if err := m.createIndexesForCollection(ctx, collName); err != nil {
				log.Printf("Warning: Failed to create indexes for collection '%s': %v", collName, err)
			}
		}
	}

	log.Println("MongoDB collections initialized successfully")
	return nil
}

// createIndexesForCollection creates appropriate indexes for each collection
func (m *MongoDB) createIndexesForCollection(ctx context.Context, collectionName string) error {
	db := m.client.Database(m.database)
	collection := db.Collection(collectionName)

	switch collectionName {
	case weatherCollection:
		// Index on region_id, timestamp for faster weather queries
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "region_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		})
		return err

	case priceCollection:
		// Index on asset_type, timestamp for faster price queries
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "asset_type", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		})
		return err

	case newsCollection:
		// Index on headline to check for duplicates and search
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "headline", Value: 1},
			},
		})
		if err != nil {
			return err
		}

		// Index on timestamp for recent news queries
		_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "timestamp", Value: -1},
			},
		})
		return err

	case fishCollection:
		// Index on generation_time and region_id for faster fish queries
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "generated_at", Value: -1},
				{Key: "region_id", Value: 1},
			},
		})
		return err

	case statsCollection:
		// Compound index on collection_name and date
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "collection_name", Value: 1},
				{Key: "date", Value: 1},
			},
		})
		return err

	case usedNewsCollection:
		// Simple index on news_id
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "news_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		})
		return err

	case queueCollection:
		// Index on added_at and status for processing queue
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "added_at", Value: 1},
				{Key: "status", Value: 1},
			},
		})
		return err

	case translatedCollection:
		// Index on original_id for fast lookups
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "original_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return err
		}

		// Index on translated_at for querying recent translations
		_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "translated_at", Value: -1},
			},
		})
		return err
	}

	return nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	if !m.connected {
		return nil
	}

	err := m.client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %v", err)
	}

	m.connected = false
	log.Println("Disconnected from MongoDB")

	return nil
}

// SaveWeatherData saves weather data to MongoDB
func (m *MongoDB) SaveWeatherData(ctx context.Context, weatherInfo *data.WeatherInfo, regionID, cityID string) error {
	collection := m.client.Database(m.database).Collection(weatherCollection)

	// Convert weatherInfo to WeatherData
	wi := WeatherData{
		RegionID:    regionID,
		CityID:      cityID,
		Condition:   weatherInfo.Condition,
		TempC:       weatherInfo.TempC,
		Humidity:    0,  // Not available in basic WeatherInfo
		WindSpeed:   0,  // Not available in basic WeatherInfo
		RainMM:      0,  // Not available in basic WeatherInfo
		Pressure:    0,  // Not available in basic WeatherInfo
		Clouds:      0,  // Not available in basic WeatherInfo
		Description: "", // Not available in basic WeatherInfo
		Timestamp:   time.Now(),
		Source:      "internal", // Default source
	}

	// Create filter for upsert operation
	filter := bson.M{
		"region_id": regionID,
		"city_id":   cityID,
	}

	// Set up upsert options
	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)

	// Perform upsert operation
	var result WeatherData
	err := collection.FindOneAndReplace(ctx, filter, wi, opts).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to upsert weather data: %v", err)
	}

	return nil
}

// SavePriceData saves or updates price data in MongoDB
func (m *MongoDB) SavePriceData(ctx context.Context, assetType string, price, volume, changePercent, volumeChange float64, source string) error {
	collection := m.client.Database(m.database).Collection(priceCollection)

	priceData := PriceData{
		AssetType:     assetType,
		Price:         price,
		Volume:        volume,
		ChangePercent: changePercent,
		VolumeChange:  volumeChange,
		Timestamp:     time.Now(),
		Source:        source,
	}

	// Create filter for upsert operation
	filter := bson.M{"asset_type": assetType}

	// Set up upsert options
	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)

	// Perform upsert operation
	var result PriceData
	err := collection.FindOneAndReplace(ctx, filter, priceData, opts).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to upsert price data: %v", err)
	}

	return nil
}

// Helper function to truncate a string to a certain length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SaveNewsData saves news data to MongoDB
func (m *MongoDB) SaveNewsData(ctx context.Context, newsItem *data.NewsItem) error {
	collection := m.client.Database(m.database).Collection(newsCollection)

	// Convert newsItem to NewsData
	newsData := NewsData{
		Headline:    newsItem.Headline,
		Content:     newsItem.GetContent(),
		Source:      newsItem.Source,
		URL:         newsItem.URL,
		PublishedAt: newsItem.PublishedAt,
		Sentiment:   newsItem.Sentiment,
		Keywords:    newsItem.Keywords,
		Timestamp:   time.Now(),
	}

	// Create filter for upsert operation - use composite key of source+headline
	// This ensures we don't overwrite different news from the same source
	filter := bson.M{
		"source":   newsData.Source,
		"headline": newsData.Headline,
	}

	// Set up upsert options
	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)

	// Perform upsert operation
	var result NewsData
	err := collection.FindOneAndReplace(ctx, filter, newsData, opts).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to upsert news data: %v", err)
	}

	log.Printf("News data saved from source '%s': '%s'",
		newsData.Source, truncateString(newsData.Headline, 50))
	return nil
}

// SaveFishData saves fish data to MongoDB
func (m *MongoDB) SaveFishData(ctx context.Context, fish interface{}) error {
	// Make sure the fish collection exists
	collection, err := m.ensureCollection(ctx, fishCollection)
	if err != nil {
		return fmt.Errorf("failed to ensure fish collection exists: %v", err)
	}

	// Convert fish to FishData type
	var fishData FishData

	switch f := fish.(type) {
	case FishData:
		fishData = f
	case *FishData:
		fishData = *f
	case map[string]interface{}:
		// Extract fields from map
		if name, ok := f["name"].(string); ok {
			fishData.Name = name
		}
		if desc, ok := f["description"].(string); ok {
			fishData.Description = desc
		}
		if rarity, ok := f["rarity"].(string); ok {
			fishData.Rarity = rarity
		}
		if habitat, ok := f["habitat"].(string); ok {
			fishData.Habitat = habitat
		}
		if diet, ok := f["diet"].(string); ok {
			fishData.Diet = diet
		}
		if color, ok := f["color"].(string); ok {
			fishData.Color = color
		}
		if regionID, ok := f["region_id"].(string); ok {
			fishData.RegionID = regionID
		}
		if dataSource, ok := f["data_source"].(string); ok {
			fishData.DataSource = dataSource
		}
		if genReason, ok := f["generation_reason"].(string); ok {
			fishData.GenerationReason = genReason
		}
		if favWeather, ok := f["favorite_weather"].(string); ok {
			fishData.FavoriteWeather = favWeather
		}
		if existReason, ok := f["existence_reason"].(string); ok {
			fishData.ExistenceReason = existReason
		}

		// Numeric fields
		if length, ok := f["length"].(float64); ok {
			fishData.Length = length
		}
		if weight, ok := f["weight"].(float64); ok {
			fishData.Weight = weight
		}
		if catchChance, ok := f["catch_chance"].(float64); ok {
			fishData.CatchChance = catchChance
		}

		// Boolean fields
		if isAI, ok := f["is_ai_generated"].(bool); ok {
			fishData.IsAIGenerated = isAI
		}

		// Time fields
		if genTime, ok := f["generated_at"].(time.Time); ok {
			fishData.GeneratedAt = genTime
		}

		// Complex fields
		if effects, ok := f["stat_effects"].([]map[string]interface{}); ok {
			fishData.StatEffects = effects
		}

		// Add handling for used articles
		if articles, ok := f["used_articles"].([]map[string]interface{}); ok {
			fishData.UsedArticles = articles
		}
	default:
		// Try to convert from a fish type that implements required methods
		info, ok := fish.(interface {
			GetName() string
			GetDescription() string
			GetRarity() string
			GetLength() float64
			GetWeight() float64
			GetColor() string
			GetHabitat() string
			GetDiet() string
			IsAIGenerated() bool
			GetDataSource() string
			GetStatEffects() interface{}
		})

		if !ok {
			return fmt.Errorf("invalid fish type")
		}

		// This is a rough conversion - we'd need to adjust for the actual structure
		var statEffects []map[string]interface{}
		if se, ok := info.GetStatEffects().([]map[string]interface{}); ok {
			statEffects = se
		}

		fishData = FishData{
			Name:          info.GetName(),
			Description:   info.GetDescription(),
			Rarity:        info.GetRarity(),
			Length:        info.GetLength(),
			Weight:        info.GetWeight(),
			Color:         info.GetColor(),
			Habitat:       info.GetHabitat(),
			Diet:          info.GetDiet(),
			GeneratedAt:   time.Now(),
			IsAIGenerated: info.IsAIGenerated(),
			DataSource:    info.GetDataSource(),
			StatEffects:   statEffects,
		}
	}

	// Set generation time if not already set
	if fishData.GeneratedAt.IsZero() {
		fishData.GeneratedAt = time.Now()
	}

	// Insert document
	result, err := collection.InsertOne(ctx, fishData)
	if err != nil {
		return fmt.Errorf("failed to insert fish data: %v", err)
	}

	// Increment daily fish count
	err = m.incrementDailyFishCount(ctx)
	if err != nil {
		log.Printf("Warning: failed to increment daily fish count: %v", err)
	}

	log.Printf("Fish data saved: %s (ID: %s)", fishData.Name, result.InsertedID.(primitive.ObjectID).Hex())
	return nil
}

// GetRecentWeatherData retrieves recent weather data for a specific region
func (m *MongoDB) GetRecentWeatherData(ctx context.Context, regionID string, limit int) ([]*WeatherData, error) {
	collection := m.client.Database(m.database).Collection(weatherCollection)

	filter := bson.M{}
	if regionID != "" {
		filter["region_id"] = regionID
	}

	// Set options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{primitive.E{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	// Execute the query
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find weather data: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode the results
	var results []*WeatherData
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode weather data: %v", err)
	}

	return results, nil
}

// GetRecentPriceData retrieves recent price data for a specific asset type
func (m *MongoDB) GetRecentPriceData(ctx context.Context, assetType string, limit int) ([]map[string]interface{}, error) {
	collection := m.client.Database(m.database).Collection(priceCollection)

	filter := bson.M{}
	if assetType != "" {
		filter["asset_type"] = assetType
	}

	// Set options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{primitive.E{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	// Execute the query
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find price data: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode the results
	var results []map[string]interface{}
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode price data: %v", err)
	}

	return results, nil
}

// GetRecentNewsData retrieves recent news data
func (m *MongoDB) GetRecentNewsData(ctx context.Context, limit int) ([]*NewsData, error) {
	collection := m.client.Database(m.database).Collection(newsCollection)

	// Set options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{primitive.E{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	// Execute the query
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find news data: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode the results
	var results []*NewsData
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode news data: %v", err)
	}

	return results, nil
}

// incrementDailyFishCount updates the counter for fish generated today
func (m *MongoDB) incrementDailyFishCount(ctx context.Context) error {
	// Make sure the stats collection exists
	collection, err := m.ensureCollection(ctx, statsCollection)
	if err != nil {
		return fmt.Errorf("failed to ensure stats collection exists: %v", err)
	}

	// Get today's date in YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")
	now := time.Now()

	// Use upsert to update or create the record
	filter := bson.M{"date": today, "record_type": "daily_fish_count"}
	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{
			"last_updated": now,
			"record_type":  "daily_fish_count", // Add a record type to distinguish from collection stats
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update fish limit record: %v", err)
	}

	return nil
}

// ensureCollection makes sure a collection exists before using it
func (m *MongoDB) ensureCollection(ctx context.Context, collectionName string) (*mongo.Collection, error) {
	collection := m.client.Database(m.database).Collection(collectionName)

	// Try to find at least one document to check if the collection exists
	findResult := collection.FindOne(ctx, bson.M{})
	if findResult.Err() != nil && findResult.Err() != mongo.ErrNoDocuments {
		// Create the collection if it doesn't exist
		if err := m.client.Database(m.database).CreateCollection(ctx, collectionName); err != nil {
			// Ignore error if collection already exists (could happen in race condition)
			if !strings.Contains(err.Error(), "already exists") {
				return nil, fmt.Errorf("failed to create collection '%s': %v", collectionName, err)
			}
		} else {
			log.Printf("Created collection '%s'", collectionName)
		}

		// Create indexes for the collection
		if err := m.createIndexesForCollection(ctx, collectionName); err != nil {
			log.Printf("Warning: Failed to create indexes for collection '%s': %v", collectionName, err)
		}
	}

	return collection, nil
}

// GetSimilarFish retrieves a similar fish from the database
func (m *MongoDB) GetSimilarFish(ctx context.Context, dataSource string, rarityLevel string) (*FishData, error) {
	// Make sure the fish collection exists
	collection, err := m.ensureCollection(ctx, fishCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure fish collection exists: %v", err)
	}

	// Create filter based on data source and rarity
	filter := bson.M{}

	// Add data source to filter if provided
	if dataSource != "" {
		filter["data_source"] = dataSource
	}

	// Add rarity to filter if provided
	if rarityLevel != "" {
		filter["rarity"] = rarityLevel
	}

	// If no filters were added, return error
	if len(filter) == 0 {
		return nil, fmt.Errorf("at least one filter parameter (dataSource or rarityLevel) must be provided")
	}

	// Find fish that match the criteria
	// Sort by most recently generated to get the newest matching fish
	opts := options.FindOne().SetSort(bson.M{"generated_at": -1})

	var fish FishData
	err = collection.FindOne(ctx, filter, opts).Decode(&fish)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar fish: %v", err)
	}

	return &fish, nil
}

// GetFishByRegion retrieves fish for a specific region
func (m *MongoDB) GetFishByRegion(ctx context.Context, regionID string, limit int) ([]*FishData, error) {
	// Make sure the fish collection exists
	collection, err := m.ensureCollection(ctx, fishCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure fish collection exists: %v", err)
	}

	// Default limit if not specified
	if limit <= 0 {
		limit = 10
	}

	// Create filter for region
	filter := bson.M{}
	if regionID != "" {
		filter["region_id"] = regionID
	}

	// Find fish matching the criteria
	options := options.Find().SetLimit(int64(limit)).SetSort(bson.M{"generated_at": -1})

	cursor, err := collection.Find(ctx, filter, options)
	if err != nil {
		return nil, fmt.Errorf("failed to find fish by region: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var results []*FishData
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode fish data: %v", err)
	}

	return results, nil
}

// GetFishByDataSource retrieves fish from a specific data source
func (m *MongoDB) GetFishByDataSource(ctx context.Context, dataSource string, limit int) ([]*FishData, error) {
	collection := m.client.Database(m.database).Collection(fishCollection)

	// Set options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{primitive.E{Key: "generated_at", Value: -1}}).
		SetLimit(int64(limit))

	// Execute the query
	filter := bson.M{"data_source": dataSource}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find fish data: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode the results
	var results []*FishData
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode fish data: %v", err)
	}

	return results, nil
}

// GetDailyFishCount returns the number of fish generated today
func (m *MongoDB) GetDailyFishCount(ctx context.Context) (int, error) {
	collection, err := m.ensureCollection(ctx, statsCollection)
	if err != nil {
		return 0, fmt.Errorf("failed to ensure stats collection exists: %v", err)
	}

	// Get today's date in YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")

	// Find the fish limit record for today
	filter := bson.M{"date": today, "record_type": "daily_fish_count"}
	var result FishLimitRecord
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No fish generated today yet
			return 0, nil
		}
		return 0, fmt.Errorf("failed to find fish limit record: %v", err)
	}

	return result.Count, nil
}

// SaveUsedNewsIDs saves a map of used news IDs to the database
func (m *MongoDB) SaveUsedNewsIDs(ctx context.Context, usedIDs map[string]bool) error {
	collection, err := m.ensureCollection(ctx, usedNewsCollection)
	if err != nil {
		return fmt.Errorf("failed to ensure used news collection exists: %v", err)
	}

	// Don't try to clear all records first - instead use upsert operations
	if len(usedIDs) == 0 {
		return nil // Nothing to save
	}

	now := time.Now()

	// Use bulk write with individual upsert operations for each news ID
	var operations []mongo.WriteModel

	for newsID := range usedIDs {
		filter := bson.M{"news_id": newsID}
		update := bson.M{"$set": bson.M{
			"news_id":     newsID,
			"used_at":     now,
			"record_type": "used_news",
		}}

		// Create upsert operation (insert if not exists, update if exists)
		upsert := true
		operation := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(upsert)

		operations = append(operations, operation)
	}

	// Execute bulk write
	opts := options.BulkWrite().SetOrdered(false) // Continue on error
	_, err = collection.BulkWrite(ctx, operations, opts)
	if err != nil {
		return fmt.Errorf("failed to save used news IDs: %v", err)
	}

	return nil
}

// GetUsedNewsIDs retrieves all used news IDs from the database
func (m *MongoDB) GetUsedNewsIDs(ctx context.Context) (map[string]bool, error) {
	collection, err := m.ensureCollection(ctx, usedNewsCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure used news collection exists: %v", err)
	}

	// Query for all used news records
	cursor, err := collection.Find(ctx, bson.M{"record_type": "used_news"})
	if err != nil {
		return nil, fmt.Errorf("failed to query used news IDs: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode into a slice of records
	var records []UsedNewsRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode used news records: %v", err)
	}

	// Convert to a map
	result := make(map[string]bool)
	for _, record := range records {
		result[record.NewsID] = true
	}

	return result, nil
}

// SaveGenerationQueue saves the current generation queue to the database
func (m *MongoDB) SaveGenerationQueue(ctx context.Context, queue []data.GenerationRequest) error {
	collection, err := m.ensureCollection(ctx, queueCollection)
	if err != nil {
		return fmt.Errorf("failed to ensure queue collection exists: %v", err)
	}

	// First, clear out all pending requests (this is safer - we want a clean slate)
	_, err = collection.DeleteMany(ctx, bson.M{"status": "pending"})
	if err != nil {
		return fmt.Errorf("failed to clear pending generation requests: %v", err)
	}

	// If queue is empty, we're done (we've cleared existing items)
	if len(queue) == 0 {
		return nil
	}

	// Now insert new records after clearing the old ones
	var records []interface{}

	for _, request := range queue {
		// Generate a unique ID if not present
		id := request.ID
		if id == "" {
			id = primitive.NewObjectID().Hex()
		}

		records = append(records, QueuedGenerationRecord{
			ID:         primitive.NewObjectID(), // Always create a new record
			Reason:     request.Reason,
			AddedAt:    request.AddedAt,
			Status:     "pending",
			RecordType: "generation_request",
		})
	}

	// Use InsertMany since we've already cleared existing records
	_, err = collection.InsertMany(ctx, records)
	if err != nil {
		return fmt.Errorf("failed to save generation queue: %v", err)
	}

	return nil
}

// GetGenerationQueue retrieves the pending generation requests from the database
func (m *MongoDB) GetGenerationQueue(ctx context.Context) ([]data.GenerationRequest, error) {
	collection, err := m.ensureCollection(ctx, queueCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure queue collection exists: %v", err)
	}

	// Query for pending generation requests, ordered by added_at
	opts := options.Find().SetSort(bson.D{{Key: "added_at", Value: 1}})
	cursor, err := collection.Find(ctx, bson.M{"status": "pending"}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query generation queue: %v", err)
	}
	defer cursor.Close(ctx)

	// Decode into a slice of records
	var records []QueuedGenerationRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode queue records: %v", err)
	}

	// Convert to GenerationRequest objects
	var result []data.GenerationRequest
	for _, record := range records {
		result = append(result, data.GenerationRequest{
			ID:      record.ID.Hex(),
			Reason:  record.Reason,
			AddedAt: record.AddedAt,
			// Note: Ctx will be attached by the caller
		})
	}

	return result, nil
}

// SaveTranslatedFish saves the translated fish data to MongoDB
func (m *MongoDB) SaveTranslatedFish(ctx context.Context, translatedFish *data.TranslatedFish) error {
	collection := m.client.Database(m.database).Collection(translatedCollection)

	// Convert string ID to ObjectID
	originalID, err := primitive.ObjectIDFromHex(translatedFish.OriginalID)
	if err != nil {
		return fmt.Errorf("invalid original fish ID: %v", err)
	}

	// Create the document
	doc := TranslatedFishData{
		OriginalID:      originalID,
		Name:            translatedFish.Name,
		Description:     translatedFish.Description,
		Appearance:      translatedFish.Appearance,
		Color:           translatedFish.Color,
		Diet:            translatedFish.Diet,
		Habitat:         translatedFish.Habitat,
		Effect:          translatedFish.Effect,
		FavoriteWeather: translatedFish.FavoriteWeather,
		ExistenceReason: translatedFish.ExistenceReason,
		TranslatedAt:    translatedFish.TranslatedAt,
		Language:        "vi", // Vietnamese language code
	}

	// Filter for upsert based on original fish ID
	filter := bson.M{
		"original_id": originalID,
		"language":    "vi",
	}

	// Set up upsert options
	opts := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)

	// Perform upsert operation
	var result TranslatedFishData
	err = collection.FindOneAndReplace(ctx, filter, doc, opts).Decode(&result)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to save translated fish: %v", err)
	}

	log.Printf("Saved translated fish for original ID: %s", translatedFish.OriginalID)
	return nil
}

// GetTranslatedFish retrieves a translated fish by original ID
func (m *MongoDB) GetTranslatedFish(ctx context.Context, originalID string) (*data.TranslatedFish, error) {
	collection := m.client.Database(m.database).Collection(translatedCollection)

	// Convert string ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(originalID)
	if err != nil {
		return nil, fmt.Errorf("invalid original fish ID: %v", err)
	}

	// Create filter
	filter := bson.M{
		"original_id": objID,
		"language":    "vi",
	}

	// Perform query
	var result TranslatedFishData
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No translation found
		}
		return nil, fmt.Errorf("failed to retrieve translated fish: %v", err)
	}

	// Convert to TranslatedFish type
	translatedFish := &data.TranslatedFish{
		OriginalID:      originalID,
		Name:            result.Name,
		Description:     result.Description,
		Appearance:      result.Appearance,
		Color:           result.Color,
		Diet:            result.Diet,
		Habitat:         result.Habitat,
		Effect:          result.Effect,
		FavoriteWeather: result.FavoriteWeather,
		ExistenceReason: result.ExistenceReason,
		TranslatedAt:    result.TranslatedAt,
	}

	return translatedFish, nil
}

// GetUntranslatedFishIDs retrieves IDs of fish that haven't been translated yet
func (m *MongoDB) GetUntranslatedFishIDs(ctx context.Context, limit int) ([]string, error) {
	// Get all fish IDs
	fishColl := m.client.Database(m.database).Collection(fishCollection)
	translatedColl := m.client.Database(m.database).Collection(translatedCollection)

	// Find all fish, sort by newest first
	fishCursor, err := fishColl.Find(ctx, bson.M{},
		options.Find().SetSort(bson.M{"generated_at": -1}).SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to query fish collection: %v", err)
	}
	defer fishCursor.Close(ctx)

	// Get all translated fish ids
	translatedCursor, err := translatedColl.Find(ctx, bson.M{},
		options.Find().SetProjection(bson.M{"original_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to query translated fish collection: %v", err)
	}
	defer translatedCursor.Close(ctx)

	// Create a map of translated fish IDs
	translatedIDs := make(map[string]bool)
	var translatedDoc struct {
		OriginalID primitive.ObjectID `bson:"original_id"`
	}

	for translatedCursor.Next(ctx) {
		if err := translatedCursor.Decode(&translatedDoc); err != nil {
			return nil, fmt.Errorf("failed to decode translated fish document: %v", err)
		}
		translatedIDs[translatedDoc.OriginalID.Hex()] = true
	}

	// Find fish that are not translated
	var fishDoc struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	untranslatedIDs := make([]string, 0, limit)
	for fishCursor.Next(ctx) {
		if err := fishCursor.Decode(&fishDoc); err != nil {
			return nil, fmt.Errorf("failed to decode fish document: %v", err)
		}

		// Check if this fish has been translated
		fishIDStr := fishDoc.ID.Hex()
		if !translatedIDs[fishIDStr] {
			untranslatedIDs = append(untranslatedIDs, fishIDStr)
			if len(untranslatedIDs) >= limit {
				break
			}
		}
	}

	return untranslatedIDs, nil
}

// GetFishByID retrieves a fish by its ID
func (m *MongoDB) GetFishByID(ctx context.Context, id string) (map[string]interface{}, error) {
	collection := m.client.Database(m.database).Collection(fishCollection)

	// Convert string ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid fish ID: %v", err)
	}

	// Create filter
	filter := bson.M{"_id": objID}

	// Perform query
	var result bson.M
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("fish not found: %s", id)
		}
		return nil, fmt.Errorf("failed to retrieve fish: %v", err)
	}

	// Convert primitive.ObjectID fields to strings for easier handling
	if id, ok := result["_id"].(primitive.ObjectID); ok {
		result["_id"] = id.Hex()
	}

	return result, nil
}

// GetUntranslatedFish retrieves fish that haven't been translated to Vietnamese yet
func (m *MongoDB) GetUntranslatedFish(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// Create a filter for fish that haven't been translated yet
	filter := bson.M{
		"is_translated": bson.M{"$ne": true}, // Find fish without the is_translated flag
	}

	// Options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.M{"generated_at": -1}). // Most recent first
		SetLimit(int64(limit))

	// Execute the query
	coll := m.client.Database(m.database).Collection(fishCollection)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding untranslated fish: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode the results
	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("error decoding untranslated fish: %w", err)
	}

	if len(results) == 0 {
		return []map[string]interface{}{}, nil // Return empty slice, not nil
	}

	return results, nil
}

// UpdateFishWithTranslation updates a fish document with translated fields
func (m *MongoDB) UpdateFishWithTranslation(ctx context.Context, fishID interface{}, translatedFish map[string]interface{}) error {
	// Convert string ID to ObjectID if needed
	var objectID primitive.ObjectID
	switch id := fishID.(type) {
	case string:
		var err error
		objectID, err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid fish ID format: %w", err)
		}
	case primitive.ObjectID:
		objectID = id
	default:
		// Try to use the _id directly as provided
		objectID, _ = fishID.(primitive.ObjectID)
	}

	// Create the filter using the ID
	filter := bson.M{"_id": objectID}

	// Validate all string fields to ensure they are valid UTF-8
	// This is a defensive measure to prevent BSON encoding errors
	for key, value := range translatedFish {
		// Skip non-string fields
		if strValue, ok := value.(string); ok {
			// If we find an invalid UTF-8 string, replace it with a valid one
			if !utf8.ValidString(strValue) {
				translatedFish[key] = strings.ToValidUTF8(strValue, "\uFFFD")
				log.Printf("Fixed invalid UTF-8 in field: %s", key)
			}
		} else if mapValue, ok := value.(map[string]interface{}); ok {
			// For nested maps, check each string value
			for nestedKey, nestedValue := range mapValue {
				if nestedStrValue, ok := nestedValue.(string); ok {
					if !utf8.ValidString(nestedStrValue) {
						mapValue[nestedKey] = strings.ToValidUTF8(nestedStrValue, "\uFFFD")
						log.Printf("Fixed invalid UTF-8 in nested field: %s.%s", key, nestedKey)
					}
				}
			}
		} else if arrayValue, ok := value.([]interface{}); ok {
			// For array values, check if they contain maps with strings
			for i, item := range arrayValue {
				if itemMap, ok := item.(map[string]interface{}); ok {
					for itemKey, itemValue := range itemMap {
						if itemStrValue, ok := itemValue.(string); ok {
							if !utf8.ValidString(itemStrValue) {
								itemMap[itemKey] = strings.ToValidUTF8(itemStrValue, "\uFFFD")
								log.Printf("Fixed invalid UTF-8 in array field: %s[%d].%s", key, i, itemKey)
							}
						}
					}
				}
			}
		}
	}

	// Define the update operation (replace the entire document)
	update := bson.M{"$set": translatedFish}

	// Execute the update
	coll := m.client.Database(m.database).Collection(fishCollection)
	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating fish with translation: %w", err)
	}

	return nil
}
