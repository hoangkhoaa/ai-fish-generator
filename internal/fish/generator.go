package fish

import (
	"context"
	"fish-generate/internal/data"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

// Generator is responsible for generating fish based on various data sources
type Generator struct {
	// Optional: add any configuration parameters here
	rarityThresholds map[Rarity]float64
	rand             *rand.Rand
	geminiClient     *data.GeminiClient
	useAI            bool
	geminiAPIKey     string
	options          GeneratorOptions
}

// GeneratorOptions provides configuration options for the fish generator
type GeneratorOptions struct {
	GeminiAPIKey string // API key for Google Gemini
	UseAI        bool   // Whether to use AI for fish generation
	TestMode     bool   // Whether to run in test mode
}

// NewGenerator creates a new fish generator
func NewGenerator(opts ...GeneratorOptions) *Generator {
	// Create a seeded random number generator for deterministic testing
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	g := &Generator{
		rarityThresholds: map[Rarity]float64{
			Common:    0.6,  // 60% chance
			Uncommon:  0.85, // 25% chance
			Rare:      0.95, // 10% chance
			Epic:      0.99, // 4% chance
			Legendary: 1.0,  // 1% chance
		},
		rand:  r,
		useAI: false,
		options: GeneratorOptions{
			TestMode: false,
		},
	}

	// Apply options if provided
	if len(opts) > 0 {
		opt := opts[0]
		g.useAI = opt.UseAI
		g.options = opt

		if opt.UseAI && opt.GeminiAPIKey != "" {
			g.geminiClient = data.NewGeminiClient(opt.GeminiAPIKey)
			g.geminiAPIKey = opt.GeminiAPIKey
		}
	}

	return g
}

// GenerateFromWeather creates a fish based on weather data
func (g *Generator) GenerateFromWeather(weatherInfo *data.WeatherInfo, reason string) *Fish {
	// Weather-based fish characteristics
	var name, description, effect string
	var size, value float64

	// Extreme weather creates rarer fish
	rarityBoost := 0.0
	if weatherInfo.IsExtreme {
		rarityBoost = 0.4 // 40% boost to rarity for extreme weather
	}
	rarity := g.determineRarity(rarityBoost)

	// Determine fish type based on weather condition
	condition := strings.ToLower(weatherInfo.Condition)

	// Default reason if none provided
	if reason == "" {
		reason = "Weather condition: " + weatherInfo.Condition
	}

	// Create fish characteristics based on weather
	switch {
	case strings.Contains(condition, "clear") || strings.Contains(condition, "sunny"):
		name = "Sunray Shimmer"
		description = "A bright, golden fish that thrives in clear, sunny waters. Its scales reflect sunlight, creating a dazzling display."
		effect = "Increases visibility in water by 20% during daylight hours."

	case strings.Contains(condition, "rain") || strings.Contains(condition, "drizzle"):
		name = "Rainbowfin Splasher"
		description = "A colorful fish that thrives in rainy conditions. Its scales shimmer with rainbow colors when wet."
		effect = "Increases chance of finding treasure during rainy weather."

	case strings.Contains(condition, "storm") || strings.Contains(condition, "thunder"):
		name = "Stormray Charger"
		description = "An electric blue fish that appears during stormy weather. It seems to be charged with the energy of the storm."
		effect = "Provides resistance to electric damage and stuns."

	case strings.Contains(condition, "cloud") || strings.Contains(condition, "overcast"):
		name = "Misty Veilfish"
		description = "A fish with translucent fins that thrives in cloudy, overcast conditions. It can almost disappear into the mist."
		effect = "Increases stealth and reduces visibility to predators."

	case strings.Contains(condition, "snow"):
		name = "Frostfin Glider"
		description = "A white fish with ice-like scales that thrives in snowy conditions. It can survive in near-freezing waters."
		effect = "Grants immunity to cold and freezing effects."

	case strings.Contains(condition, "wind"):
		name = "Galeforce Swimmer"
		description = "A streamlined fish that excels in windy conditions. Its fins are specially adapted to navigate choppy waters."
		effect = "Increases swimming speed by 30% during windy weather."

	default:
		name = "Weather Wanderer"
		description = "An adaptable fish that changes its characteristics based on the weather."
		effect = "Provides a small boost to all stats regardless of weather conditions."
	}

	// Size based on temperature (warmer = bigger)
	// Range: 0.1-1.5 meters
	normalizedTemp := (weatherInfo.TempC + 10.0) / 40.0 // Normalize from -10C to 30C to 0-1 range
	normalizedTemp = math.Max(0.0, math.Min(1.0, normalizedTemp))
	size = 0.1 + (normalizedTemp * 1.4)

	// Value based on rarity and extremeness
	baseValue := 50.0
	if weatherInfo.IsExtreme {
		baseValue *= 2.0
	}
	rarityMultiplier := 1.0
	switch rarity {
	case Common:
		rarityMultiplier = 1.0
	case Uncommon:
		rarityMultiplier = 2.5
	case Rare:
		rarityMultiplier = 6.0
	case Epic:
		rarityMultiplier = 12.0
	case Legendary:
		rarityMultiplier = 25.0
	}
	value = baseValue * rarityMultiplier

	return NewFish(name, rarity, size, value, description, effect, "weather", reason)
}

// GenerateFromBitcoin creates a fish based on Bitcoin price data
func (g *Generator) GenerateFromBitcoin(cryptoPrice *data.CryptoPrice, reason string) *Fish {
	// Bitcoin-based fish characteristics
	var name, description, effect string
	var size, value float64

	// Significant price changes create rarer fish
	priceChangeAbs := math.Abs(cryptoPrice.Change24h)
	rarityBoost := math.Min(0.5, priceChangeAbs/20.0) // 10% change = 0.5 boost (max)
	rarity := g.determineRarity(rarityBoost)

	// Default reason if none provided
	if reason == "" {
		reason = fmt.Sprintf("Bitcoin price update: $%.2f (%.2f%%)", cryptoPrice.PriceUSD, cryptoPrice.Change24h)
	}

	// Determine fish type based on price change direction
	if cryptoPrice.Change24h > 5.0 {
		// Significant price increase
		name = "Crypto Bullfish"
		description = "A golden fish with upward-pointing fins that appears during cryptocurrency bull markets. Its scales seem to shimmer with digital patterns."
		effect = "Increases luck when fishing in technology-rich areas by " + formatFloat(math.Min(50, cryptoPrice.Change24h)) + "%."
	} else if cryptoPrice.Change24h < -5.0 {
		// Significant price decrease
		name = "Crypto Bearfish"
		description = "A silver fish with downward-pointing fins that appears during cryptocurrency bear markets. It has adapted to survive in volatile conditions."
		effect = "Reduces resource costs by " + formatFloat(math.Min(30, -cryptoPrice.Change24h)) + "% during economic downturns."
	} else {
		// Stable price
		name = "Blockchain Bass"
		description = "A distinctive fish with scale patterns resembling linked chains. It maintains a steady presence regardless of market conditions."
		effect = "Provides consistent bonuses regardless of in-game economic conditions."
	}

	// Size based on price (higher price = bigger fish)
	// Range: 0.2-2.0 meters
	normalizedPrice := math.Min(cryptoPrice.PriceUSD/100000.0, 1.0) // Normalize based on a max expected price of 100K
	size = 0.2 + (normalizedPrice * 1.8)

	// Value based on rarity and price
	baseValue := cryptoPrice.PriceUSD / 100.0 // Base value is 1/100th of BTC price
	rarityMultiplier := 1.0
	switch rarity {
	case Common:
		rarityMultiplier = 1.0
	case Uncommon:
		rarityMultiplier = 3.0
	case Rare:
		rarityMultiplier = 7.0
	case Epic:
		rarityMultiplier = 15.0
	case Legendary:
		rarityMultiplier = 30.0
	}
	value = baseValue * rarityMultiplier

	return NewFish(name, rarity, size, value, description, effect, "bitcoin", reason)
}

// GenerateFromOilPrice creates a fish based on oil price data
func (g *Generator) GenerateFromOilPrice(oilPrice *data.OilPrice, reason string) *Fish {
	// Oil price-based fish characteristics
	var name, description, effect string
	var size, value float64

	// Significant price changes create rarer fish
	priceChangeAbs := math.Abs(oilPrice.Change24h)
	rarityBoost := math.Min(0.5, priceChangeAbs/10.0) // 5% change = 0.5 boost (max)
	rarity := g.determineRarity(rarityBoost)

	// Default reason if none provided
	if reason == "" {
		reason = fmt.Sprintf("Oil price update: $%.2f (%.2f%%)", oilPrice.PriceUSD, oilPrice.Change24h)
	}

	// Determine fish type based on price
	if oilPrice.PriceUSD > 80.0 {
		// High oil price
		name = "Petroleum Puffer"
		description = "A dark, oil-slick colored fish that becomes more common when oil prices are high. It has adapted to survive in environments affected by oil production."
		effect = "Provides resistance to pollution and toxic environments."
	} else if oilPrice.PriceUSD < 60.0 {
		// Low oil price
		name = "Crude Carp"
		description = "A fish with an iridescent sheen that thrives when oil prices are low. It seems to represent the balance of energy in the ecosystem."
		effect = "Reduces energy consumption of fishing equipment by " + formatFloat(math.Min(30, 90-oilPrice.PriceUSD)) + "%."
	} else {
		// Medium oil price
		name = "Barrel Barracuda"
		description = "A streamlined fish that maintains a steady presence regardless of oil market fluctuations."
		effect = "Provides consistent performance in varying economic conditions."
	}

	// Size based on price (higher price = bigger fish)
	// Range: 0.3-1.5 meters
	normalizedPrice := math.Min(oilPrice.PriceUSD/150.0, 1.0) // Normalize based on a max expected price of $150
	size = 0.3 + (normalizedPrice * 1.2)

	// Value based on rarity and price relationship
	baseValue := oilPrice.PriceUSD * 2.0 // Base value is twice the oil price
	rarityMultiplier := 1.0
	switch rarity {
	case Common:
		rarityMultiplier = 1.0
	case Uncommon:
		rarityMultiplier = 2.5
	case Rare:
		rarityMultiplier = 5.0
	case Epic:
		rarityMultiplier = 10.0
	case Legendary:
		rarityMultiplier = 20.0
	}
	value = baseValue * rarityMultiplier

	return NewFish(name, rarity, size, value, description, effect, "oil", reason)
}

// GenerateFromNews creates a fish based on news data
func (g *Generator) GenerateFromNews(ctx context.Context, newsItem *data.NewsItem, reason string) *Fish {
	// Set a default reason if none provided
	if reason == "" {
		reason = "News update: " + truncateString(newsItem.Headline, 30)
	}

	// If AI is enabled and client is configured, use Gemini-powered generation
	if g.useAI && g.geminiClient != nil {
		fish, err := g.generateAIFishFromNews(ctx, newsItem, reason)
		if err == nil && fish != nil {
			return fish
		}

		// Fall back to regular generation if AI fails
		log.Printf("AI fish generation failed, falling back to rule-based: %v", err)
	}

	// Use the original rule-based method as fallback
	return g.generateRuleBasedFishFromNews(newsItem, reason)
}

// generateAIFishFromNews creates a fish using the Gemini API
func (g *Generator) generateAIFishFromNews(ctx context.Context, newsItem *data.NewsItem, reason string) (*Fish, error) {
	// Create a new context with the API key
	apiKeyCtx := context.WithValue(ctx, "gemini_api_key", g.geminiAPIKey)

	// Generate fish using Gemini
	aiResponse, err := g.geminiClient.GenerateFishFromNews(apiKeyCtx, newsItem)
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	// Validate the response
	if aiResponse.Name == "" || aiResponse.Description == "" || aiResponse.Effect == "" {
		return nil, fmt.Errorf("incomplete AI response")
	}

	// Convert rarity string to enum
	var rarity Rarity
	switch strings.ToLower(aiResponse.Rarity) {
	case "common":
		rarity = Common
	case "uncommon":
		rarity = Uncommon
	case "rare":
		rarity = Rare
	case "epic":
		rarity = Epic
	case "legendary":
		rarity = Legendary
	default:
		// Default to Uncommon if invalid rarity
		rarity = Uncommon
	}

	// Enhance description with appearance if provided
	fullDescription := aiResponse.Description
	if aiResponse.Appearance != "" {
		fullDescription = fullDescription + " " + aiResponse.Appearance
	}

	// Create fish with AI-generated attributes
	return NewFish(
		aiResponse.Name,
		rarity,
		aiResponse.Size,
		aiResponse.Value,
		fullDescription,
		aiResponse.Effect,
		"news-ai",
		reason,
	), nil
}

// generateRuleBasedFishFromNews is the original news-based fish generation logic
func (g *Generator) generateRuleBasedFishFromNews(newsItem *data.NewsItem, reason string) *Fish {
	// The existing fish generation logic
	var name, description, effect string
	var size, value float64

	// News-based fish are inherently more interesting
	// Sentiment affects rarity - extremely positive or negative news creates rarer fish
	sentimentIntensity := math.Abs(newsItem.Sentiment)
	rarityBoost := math.Min(0.6, sentimentIntensity*0.8) // Max 60% boost at extreme sentiment
	rarity := g.determineRarity(rarityBoost)

	// Use news category and sentiment to determine fish type
	category := strings.ToLower(newsItem.Category)

	// Create name based on category and sentiment
	switch {
	case strings.Contains(category, "politics") || strings.Contains(category, "government"):
		if newsItem.Sentiment > 0.3 {
			name = "Diplomacy Darter"
			description = "A peaceful fish that emerges during times of political cooperation. It has the unusual ability to mediate between different fish species."
			effect = "Reduces conflict between different fish types in your collection."
		} else if newsItem.Sentiment < -0.3 {
			name = "Rhetoric Razormouth"
			description = "An aggressive fish that appears during political conflicts. Its sharp teeth and bold patterns reflect political tensions."
			effect = "Increases critical hit chance in fishing battles by " + formatFloat(math.Min(30, sentimentIntensity*50)) + "%."
		} else {
			name = "Policy Patroller"
			description = "A methodical fish that watches over territories, establishing rules like a governing body."
			effect = "Improves organization of your fish collection, increasing overall efficiency."
		}

	case strings.Contains(category, "tech") || strings.Contains(category, "technology"):
		name = "Silicon Surfer"
		description = "A highly intelligent fish with circuit-like patterns on its scales. It processes information about its environment with remarkable efficiency."
		effect = "Increases technology research speed by " + formatFloat(math.Min(50, sentimentIntensity*60)) + "%."

	case strings.Contains(category, "business") || strings.Contains(category, "economy"):
		if newsItem.Sentiment > 0.3 {
			name = "Market Bullfish"
			description = "A prosperous fish that thrives in strong economic conditions. Its golden scales become more vibrant when the economy is booming."
			effect = "Increases the selling price of all fish by " + formatFloat(math.Min(30, newsItem.Sentiment*50)) + "%."
		} else if newsItem.Sentiment < -0.3 {
			name = "Recession Remora"
			description = "A resilient fish that has adapted to economic downturns. It attaches to larger entities and helps them survive tough times."
			effect = "Reduces maintenance costs by " + formatFloat(math.Min(40, -newsItem.Sentiment*60)) + "%."
		} else {
			name = "Commerce Cod"
			description = "A balanced fish that represents steady economic activity."
			effect = "Provides a small boost to all economic activities."
		}

	case strings.Contains(category, "entertain") || strings.Contains(category, "celeb"):
		name = "Celebrity Starfish"
		description = "A flamboyant fish that draws attention with its colorful appearance and dramatic behaviors. It's often seen performing for other fish."
		effect = "Attracts more fish to your fishing area, increasing catch rates."

	case strings.Contains(category, "health") || strings.Contains(category, "science"):
		name = "Research Reedfish"
		description = "A fish with highly adaptable biology that seems to evolve in response to scientific breakthroughs."
		effect = "Improves healing rate and provides immunity to negative status effects."

	case strings.Contains(category, "sports"):
		name = "Athletic Anglerfish"
		description = "An energetic fish known for its speed and agility. It challenges other fish to races through the coral reefs."
		effect = "Increases player stamina and movement speed."

	case strings.Contains(category, "disaster") || strings.Contains(category, "crisis"):
		name = "Crisis Crestfish"
		description = "A rare fish that appears during times of crisis. It has evolved remarkable survival mechanisms and can endure extreme conditions."
		effect = "Provides resilience to environmental damage and disasters."

	default:
		// General news
		if newsItem.Sentiment > 0.3 {
			name = "Headline Hopeful"
			description = "A bright, optimistic fish that appears when positive news is spreading. It brings good fortune to those who catch it."
			effect = "Increases luck by " + formatFloat(math.Min(40, newsItem.Sentiment*60)) + "%."
		} else if newsItem.Sentiment < -0.3 {
			name = "Headline Harbinger"
			description = "A dark, mysterious fish that appears during troubling news. It has adapted to thrive in chaos."
			effect = "Provides resistance to negative effects and increased resources in difficult conditions."
		} else {
			name = "Current Eventfish"
			description = "A fish that changes its appearance based on ongoing events. It always seems to reflect the current zeitgeist."
			effect = "Adapts its bonuses based on current game conditions."
		}
	}

	// Size based on news impact (estimated by sentiment intensity)
	size = 0.2 + (sentimentIntensity * 2.0) // 0.2-2.2 meters

	// Value based on rarity and sentiment intensity
	baseValue := 100.0 * (1.0 + sentimentIntensity*2.0) // 100-300 base value
	rarityMultiplier := 1.0
	switch rarity {
	case Common:
		rarityMultiplier = 1.0
	case Uncommon:
		rarityMultiplier = 3.0
	case Rare:
		rarityMultiplier = 7.0
	case Epic:
		rarityMultiplier = 15.0
	case Legendary:
		rarityMultiplier = 30.0
	}

	value = baseValue * rarityMultiplier

	// Create the fish with news-specific information
	return NewFish(name, rarity, size, value, description, effect, "news", reason)
}

// truncateString truncates a string to the specified length if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// determineRarity selects a rarity level based on random chance with the given boost
// boost parameter increases chances of rare fish (0.0-1.0)
func (g *Generator) determineRarity(boost float64) Rarity {
	// Generate a random number
	roll := g.rand.Float64()

	// Adjust roll with boost (making rarer fish more likely)
	// We apply the boost progressively, affecting rarer tiers more
	adjustedRoll := roll
	if boost > 0 {
		// Making the probability curve more favorable for rare fish
		adjustedRoll = math.Pow(roll, 1.0+boost)
	}

	// Determine rarity based on thresholds
	if adjustedRoll < g.rarityThresholds[Common] {
		return Common
	} else if adjustedRoll < g.rarityThresholds[Uncommon] {
		return Uncommon
	} else if adjustedRoll < g.rarityThresholds[Rare] {
		return Rare
	} else if adjustedRoll < g.rarityThresholds[Epic] {
		return Epic
	} else {
		return Legendary
	}
}

// Helper function to format float to 2 decimal places as string
func formatFloat(val float64) string {
	return strings.TrimRight(strings.TrimRight(
		strings.ReplaceAll(
			fmt.Sprintf("%.2f", val),
			".00", ""),
		"0"), ".")
}

// GeminiClient returns the generator's Gemini client
func (g *Generator) GeminiClient() *data.GeminiClient {
	return g.geminiClient
}
