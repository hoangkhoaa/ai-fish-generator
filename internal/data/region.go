package data

// Region represents an ocean region with specific characteristics
type Region struct {
	ID          string   `json:"id" bson:"_id"`
	Name        string   `json:"name" bson:"name"`
	Description string   `json:"description" bson:"description"`
	Location    Location `json:"location" bson:"location"`
	Tags        []string `json:"tags" bson:"tags"`
	CityIDs     []string `json:"city_ids" bson:"city_ids"` // OpenWeatherMap city IDs
	Temperature Range    `json:"temperature" bson:"temperature"`
	Depth       Range    `json:"depth" bson:"depth"`
	Salinity    Range    `json:"salinity" bson:"salinity"`
}

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

// Range represents a numeric range with minimum and maximum values
type Range struct {
	Min float64 `json:"min" bson:"min"`
	Max float64 `json:"max" bson:"max"`
}

// PredefinedRegions returns a list of predefined ocean regions
func PredefinedRegions() []Region {
	return []Region{
		{
			ID:          "north_atlantic",
			Name:        "North Atlantic",
			Description: "Cold, deep waters with diverse marine life",
			Location:    Location{Latitude: 45.0, Longitude: -30.0},
			Tags:        []string{"cold", "deep", "temperate"},
			CityIDs: []string{
				"5128581", // New York
				"4930956", // Boston
				"2643743", // London
				"2950159", // Berlin
			},
			Temperature: Range{Min: 5.0, Max: 15.0},
			Depth:       Range{Min: 500.0, Max: 3000.0},
			Salinity:    Range{Min: 34.0, Max: 35.0},
		},
		{
			ID:          "tropical_pacific",
			Name:        "Tropical Pacific",
			Description: "Warm, clear waters with vibrant coral reefs",
			Location:    Location{Latitude: 0.0, Longitude: 160.0},
			Tags:        []string{"warm", "tropical", "coral"},
			CityIDs: []string{
				"1850147", // Tokyo
				"1880252", // Singapore
				"2147714", // Sydney
				"5856195", // Honolulu
			},
			Temperature: Range{Min: 24.0, Max: 30.0},
			Depth:       Range{Min: 100.0, Max: 2000.0},
			Salinity:    Range{Min: 34.5, Max: 35.5},
		},
		{
			ID:          "mediterranean",
			Name:        "Mediterranean Sea",
			Description: "Warm, saltier waters with rich history and biodiversity",
			Location:    Location{Latitude: 35.0, Longitude: 18.0},
			Tags:        []string{"warm", "salty", "historic"},
			CityIDs: []string{
				"2988507", // Paris
				"3169070", // Rome
				"2510769", // Barcelona
				"3110044", // Madrid
			},
			Temperature: Range{Min: 15.0, Max: 26.0},
			Depth:       Range{Min: 100.0, Max: 1500.0},
			Salinity:    Range{Min: 36.0, Max: 39.0},
		},
		{
			ID:          "arctic",
			Name:        "Arctic Ocean",
			Description: "Extremely cold waters with unique ice-adapted species",
			Location:    Location{Latitude: 80.0, Longitude: 0.0},
			Tags:        []string{"frigid", "icy", "extreme"},
			CityIDs: []string{
				"3413829", // Reykjavik
				"5983720", // Oslo
				"524901",  // Moscow
				"2950158", // Helsinki
			},
			Temperature: Range{Min: -2.0, Max: 5.0},
			Depth:       Range{Min: 200.0, Max: 4000.0},
			Salinity:    Range{Min: 30.0, Max: 33.0},
		},
		{
			ID:          "south_pacific",
			Name:        "South Pacific",
			Description: "Pristine waters with diverse island ecosystems",
			Location:    Location{Latitude: -20.0, Longitude: -170.0},
			Tags:        []string{"tropical", "island", "diverse"},
			CityIDs: []string{
				"2147714", // Sydney
				"2179537", // Wellington
				"4164138", // Miami
				"3451190", // Rio de Janeiro
			},
			Temperature: Range{Min: 20.0, Max: 28.0},
			Depth:       Range{Min: 300.0, Max: 5000.0},
			Salinity:    Range{Min: 34.0, Max: 36.0},
		},
	}
}

// GetRegionByID returns a region by its ID
func GetRegionByID(id string) (Region, bool) {
	for _, region := range PredefinedRegions() {
		if region.ID == id {
			return region, true
		}
	}
	return Region{}, false
}

// GetAllRegionIDs returns all region IDs
func GetAllRegionIDs() []string {
	regions := PredefinedRegions()
	ids := make([]string, len(regions))
	for i, region := range regions {
		ids[i] = region.ID
	}
	return ids
}
