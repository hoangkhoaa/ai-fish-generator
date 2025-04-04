package fish

import (
	"fmt"
	"math"
	"math/rand"
)

// StatType represents types of game statistics that can be affected by fish
type StatType string

const (
	// Basic stats
	CatchChance   StatType = "catch_chance"   // Chance to catch a fish
	CriticalCatch StatType = "critical_catch" // Chance to perform a critical catch (better rewards)
	Luck          StatType = "luck"           // Affects rare fish appearance and treasure finds
	StaminaRegen  StatType = "stamina_regen"  // How quickly stamina regenerates

	// Economic stats
	SellValue    StatType = "sell_value"    // Base value multiplier when selling fish
	MarketDemand StatType = "market_demand" // How frequently market demands your fish (sell opportunities)
	BaitCost     StatType = "bait_cost"     // Reduction in bait cost

	// Exploration stats
	ExploreSpeed  StatType = "explore_speed"  // Movement/travel speed in exploration
	AreaAccess    StatType = "area_access"    // Access to special fishing areas
	WeatherResist StatType = "weather_resist" // Resistance to harsh weather conditions

	// Collection stats
	StorageSpace     StatType = "storage_space"     // Extra storage for caught fish
	PreserveDuration StatType = "preserve_duration" // How long fish stay fresh
	CollectionBonus  StatType = "collection_bonus"  // Bonus for completing collection sets
)

// StatEffect represents an effect on a specific game stat
type StatEffect struct {
	Stat      StatType `json:"stat"`       // Which stat is affected
	Value     float64  `json:"value"`      // Effect value (percentage or absolute)
	IsPercent bool     `json:"is_percent"` // Whether the value is a percentage
	Duration  int      `json:"duration"`   // Duration in minutes (0 = permanent)
}

// StatEffects is a collection of effects on game stats
type StatEffects []StatEffect

// FormatEffects returns a human-readable description of the stat effects
func FormatEffects(effects StatEffects) string {
	if len(effects) == 0 {
		return "No effects"
	}

	result := ""
	for i, effect := range effects {
		if i > 0 {
			result += " and "
		}

		// Format the effect value
		valueStr := fmt.Sprintf("%.1f", math.Abs(effect.Value))
		if effect.IsPercent {
			valueStr += "%"
		}

		// Format the effect description
		var effectStr string
		if effect.Value > 0 {
			effectStr = fmt.Sprintf("Increases %s by %s", formatStatName(effect.Stat), valueStr)
		} else {
			effectStr = fmt.Sprintf("Decreases %s by %s", formatStatName(effect.Stat), valueStr)
		}

		// Add duration if not permanent
		if effect.Duration > 0 {
			effectStr += fmt.Sprintf(" for %d minutes", effect.Duration)
		}

		result += effectStr
	}

	return result
}

// formatStatName returns a human-readable name for a stat type
func formatStatName(stat StatType) string {
	switch stat {
	case CatchChance:
		return "fishing success rate"
	case CriticalCatch:
		return "critical catch chance"
	case Luck:
		return "fishing luck"
	case StaminaRegen:
		return "stamina regeneration"
	case SellValue:
		return "fish selling price"
	case MarketDemand:
		return "market demand"
	case BaitCost:
		return "bait cost"
	case ExploreSpeed:
		return "exploration speed"
	case AreaAccess:
		return "access to special areas"
	case WeatherResist:
		return "weather resistance"
	case StorageSpace:
		return "storage capacity"
	case PreserveDuration:
		return "fish preservation time"
	case CollectionBonus:
		return "collection completion bonus"
	default:
		return string(stat)
	}
}

// GenerateBalancedEffects creates balanced stat effects based on fish rarity and data source
func GenerateBalancedEffects(rarity Rarity, dataSource string, isAI bool) StatEffects {
	// Base effect strength based on rarity
	var baseEffect float64
	var numEffects int

	switch rarity {
	case Common:
		baseEffect = 5.0
		numEffects = 1
	case Uncommon:
		baseEffect = 10.0
		numEffects = 1
	case Rare:
		baseEffect = 15.0
		numEffects = 2
	case Epic:
		baseEffect = 25.0
		numEffects = 2
	case Legendary:
		baseEffect = 40.0
		numEffects = 3
	}

	// If AI-generated, slightly boost effect strength
	if isAI {
		baseEffect *= 1.1
	}

	// Determine appropriate stats based on data source
	var appropriateStats []StatType

	switch dataSource {
	case "weather":
		appropriateStats = []StatType{
			WeatherResist,
			ExploreSpeed,
			CatchChance,
			StaminaRegen,
		}
	case "bitcoin", "oil":
		appropriateStats = []StatType{
			SellValue,
			MarketDemand,
			BaitCost,
			CollectionBonus,
		}
	case "news", "news-ai":
		appropriateStats = []StatType{
			Luck,
			CriticalCatch,
			StorageSpace,
			PreserveDuration,
		}
	default:
		// Fallback to a mix of stats
		appropriateStats = []StatType{
			CatchChance,
			Luck,
			SellValue,
			ExploreSpeed,
		}
	}

	// Pick a subset of appropriate stats
	selectedStats := selectRandomStats(appropriateStats, numEffects)

	// Create balanced effects
	effects := make(StatEffects, 0, numEffects)

	for _, stat := range selectedStats {
		// Determine if this should be a positive or negative effect
		// Higher rarity fish are more likely to have positive effects
		isPositive := true

		// Vary the effect strength slightly (80-120% of base)
		effectStrength := baseEffect * (0.8 + rand.Float64()*0.4)

		// Apply the sign based on whether it's positive or negative
		if !isPositive {
			effectStrength = -effectStrength
		}

		// Determine if this should be a percentage or absolute value
		isPercent := true

		// Some stats should always be percentage-based
		if stat == AreaAccess || stat == StorageSpace {
			isPercent = false
			// Scale absolute values appropriately
			effectStrength = math.Max(1.0, effectStrength/5.0)
		}

		// Determine duration (higher rarity = longer duration)
		// 0 means permanent
		duration := 0
		if stat == CriticalCatch || stat == Luck || stat == SellValue {
			// These powerful effects should be temporary
			duration = int(30 + (float64(rarityToInt(rarity)) * 15))
		}

		// Create the effect
		effect := StatEffect{
			Stat:      stat,
			Value:     effectStrength,
			IsPercent: isPercent,
			Duration:  duration,
		}

		effects = append(effects, effect)
	}

	return effects
}

// selectRandomStats picks a random subset of stats
func selectRandomStats(stats []StatType, count int) []StatType {
	if count >= len(stats) {
		return stats
	}

	// Shuffle the stats
	for i := range stats {
		j := rand.Intn(i + 1)
		stats[i], stats[j] = stats[j], stats[i]
	}

	return stats[:count]
}

// rarityToInt converts rarity to an integer value for calculations
func rarityToInt(rarity Rarity) int {
	switch rarity {
	case Common:
		return 1
	case Uncommon:
		return 2
	case Rare:
		return 3
	case Epic:
		return 4
	case Legendary:
		return 5
	default:
		return 1
	}
}
