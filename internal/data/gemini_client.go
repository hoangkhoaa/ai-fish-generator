package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient provides access to Google's Gemini API
type GeminiClient struct {
	client *genai.Client
	model  string
}

// FishGenerationResponse contains structured data for fish generation
type FishGenerationResponse struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Appearance      string  `json:"appearance"`
	Color           string  `json:"color"`
	Diet            string  `json:"diet"`
	Habitat         string  `json:"habitat"`
	Effect          string  `json:"effect"`
	Rarity          string  `json:"rarity"`
	Size            float64 `json:"size"`
	SizeUnits       string  `json:"size_units"`
	Value           float64 `json:"value"`
	FavoriteWeather string  `json:"favorite_weather"`
	CatchChance     float64 `json:"catch_chance"`
	ExistenceReason string  `json:"existence_reason"`
	OriginContext   string  `json:"origin_context"`
}

// NewGeminiClient creates a new client for the Gemini API
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		// We'll initialize the actual client when needed
		model: "gemma-3-27b-it", // Updated to use Gemma model
	}
}

// initClient initializes the Gemini client if it hasn't been already
func (c *GeminiClient) initClient(ctx context.Context, apiKey string) error {
	if c.client != nil {
		return nil
	}

	var err error
	c.client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return nil
}

// GenerateFishFromNews uses Gemini to generate a creative fish based on news
func (c *GeminiClient) GenerateFishFromNews(ctx context.Context, newsItem *NewsItem) (*FishGenerationResponse, error) {
	// Extract API key from context
	apiKey, ok := ctx.Value("gemini_api_key").(string)
	if !ok || apiKey == "" {
		log.Printf("ERROR: No API key provided for Gemini. Fish generation will fail.")
		return nil, fmt.Errorf("no API key provided in context")
	}

	// Create a fresh client for each request to avoid stale connections
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("ERROR: Failed to create Gemini client: %v", err)
		return nil, err
	}
	defer client.Close()

	// Create a model instance directly
	model := client.GenerativeModel("gemma-3-27b-it")

	// Configure model settings
	model.SetTemperature(0.85) // Slightly reduced from 1.0 for more consistent output
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"

	// Start a fresh chat session
	session := model.StartChat()

	// Build prompt text
	prompt := c.buildFishGenerationPrompt(newsItem)

	log.Printf("Sending request to Gemini API using model: gemma-3-27b-it")
	log.Printf("News headline: \"%s\", Category: %s", newsItem.Headline, newsItem.Category)

	// Send the message
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Gemini API request failed: %v", err)
		log.Printf("Using text context from news headline instead")
		return nil, fmt.Errorf("error sending message: %w", err)
	}

	// Process response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("ERROR: Empty response from Gemini model")
		log.Printf("Using text context from news headline instead")
		return nil, fmt.Errorf("empty response from model")
	}

	// Extract response text
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if str, ok := part.(genai.Text); ok {
			responseText += string(str)
		}
	}

	// Parse the JSON response
	fish, err := c.parseFishResponse(responseText)
	if err != nil {
		log.Printf("ERROR: Failed to parse Gemini response: %v", err)
		log.Printf("Response text: %s", responseText)
		log.Printf("Using text context from news headline instead")
		return nil, fmt.Errorf("failed to parse model response: %w", err)
	}

	log.Printf("Successfully generated fish using Gemini: %s (Rarity: %s)", fish.Name, fish.Rarity)
	return fish, nil
}

// buildFishGenerationPrompt creates a detailed prompt for the Gemini API
func (c *GeminiClient) buildFishGenerationPrompt(newsItem *NewsItem) string {
	var sentiment string
	if newsItem.Sentiment > 0.3 {
		sentiment = "positive"
	} else if newsItem.Sentiment < -0.3 {
		sentiment = "negative"
	} else {
		sentiment = "neutral"
	}

	return fmt.Sprintf(`You are a creative fish species designer for a fishing game. 
Create a unique and imaginative fish species inspired by the following news headline:

NEWS HEADLINE: "%s"
NEWS CATEGORY: %s
SENTIMENT: %s

The fish should have characteristics that reflect the theme, content, and sentiment of the news.
Be extremely creative - your goal is to create a fascinating, magical fish with unique traits.

IMPORTANT GUIDELINES:
- Make the fish's name, appearance, and effects closely related to the news content
- More significant news should produce rarer and more valuable fish
- Positive news should create beneficial fish, negative news should create darker/mysterious fish
- For technology news, create a high-tech fish with glowing or electronic features
- For financial news, the fish's value and appearance should reflect market conditions
- For political news, the fish should have diplomatic or leadership traits
- For environmental news, create a fish with corresponding elemental attributes

Respond with a JSON object containing ONLY the following fields:
{
  "name": "A unique and creative name for the fish species",
  "description": "A short description of the fish, including any relevant traits or abilities",
  "appearance": "A vivid description of the fish's physical appearance",
  "effect": "A gameplay effect that the fish provides to the player",
  "rarity": "One of: Common, Uncommon, Rare, Epic, Legendary",
  "size": "A number representing the size of the fish in meters (between 0.1 and 3.0)",
  "size_units": "meters",
  "value": "A number representing the market value of the fish in USD (between 5 and 10000)"
}

Return ONLY the valid JSON object with no additional text.
`, newsItem.Headline, newsItem.Category, sentiment)
}

// parseFishResponse extracts the JSON fish data from the Gemini response
func (c *GeminiClient) parseFishResponse(response string) (*FishGenerationResponse, error) {
	// Remove Markdown code block markers if present
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	response = strings.TrimSpace(response)

	// Extract JSON object from response (in case there's any text before or after)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		// If we can't find valid JSON structure, try to reconstruct a basic one
		// from the fields we can extract from the partial response
		return c.reconstructPartialJSON(response)
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var fish FishGenerationResponse
	if err := json.Unmarshal([]byte(jsonStr), &fish); err != nil {
		// If unmarshaling fails, try to repair the JSON
		repairedJSON, repairErr := c.repairJSON(jsonStr)
		if repairErr != nil {
			return c.reconstructPartialJSON(response)
		}

		// Try with the repaired JSON
		if err := json.Unmarshal([]byte(repairedJSON), &fish); err != nil {
			return c.reconstructPartialJSON(response)
		}
	}

	// Validate essential fields and provide defaults if missing
	if fish.Name == "" {
		fish.Name = "Mysterious Fish"
	}
	if fish.Description == "" {
		fish.Description = "A mysterious fish that appeared suddenly in the depths."
	}
	if fish.Rarity == "" {
		fish.Rarity = "Uncommon"
	}
	if fish.Size == 0 {
		fish.Size = 1.0
	}
	if fish.Value == 0 {
		fish.Value = 100.0
	}

	return &fish, nil
}

// reconstructPartialJSON attempts to build a valid fish response from partial text
func (c *GeminiClient) reconstructPartialJSON(text string) (*FishGenerationResponse, error) {
	// Create a default fish
	fish := &FishGenerationResponse{
		Name:            "Mysterious Fish",
		Description:     "A mysterious fish that appeared suddenly in the depths.",
		Appearance:      "Shimmering scales with an otherworldly glow.",
		Color:           "Iridescent blue",
		Diet:            "Small aquatic organisms and plankton",
		Habitat:         "Unknown depths",
		Effect:          "Provides a sense of wonder when caught.",
		Rarity:          "Rare",
		Size:            1.0,
		SizeUnits:       "meters",
		Value:           200.0,
		FavoriteWeather: "Cloudy",
		CatchChance:     25.0,
		ExistenceReason: "Appeared due to mysterious oceanic currents",
		OriginContext:   "Generated from incomplete data",
	}

	// Try to extract name
	nameMatch := regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(nameMatch) > 1 {
		fish.Name = nameMatch[1]
	}

	// Try to extract description
	descMatch := regexp.MustCompile(`"description"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(descMatch) > 1 {
		fish.Description = descMatch[1]
	}

	// Try to extract appearance
	appearanceMatch := regexp.MustCompile(`"appearance"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(appearanceMatch) > 1 {
		fish.Appearance = appearanceMatch[1]
	}

	// Try to extract color
	colorMatch := regexp.MustCompile(`"color"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(colorMatch) > 1 {
		fish.Color = colorMatch[1]
	}

	// Try to extract diet
	dietMatch := regexp.MustCompile(`"diet"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(dietMatch) > 1 {
		fish.Diet = dietMatch[1]
	}

	// Try to extract habitat
	habitatMatch := regexp.MustCompile(`"habitat"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(habitatMatch) > 1 {
		fish.Habitat = habitatMatch[1]
	}

	// Try to extract effect
	effectMatch := regexp.MustCompile(`"effect"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(effectMatch) > 1 {
		fish.Effect = effectMatch[1]
	}

	// Try to extract rarity
	rarityMatch := regexp.MustCompile(`"rarity"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(rarityMatch) > 1 {
		fish.Rarity = rarityMatch[1]
	}

	// Try to extract size
	sizeMatch := regexp.MustCompile(`"size"\s*:\s*([0-9.]+)`).FindStringSubmatch(text)
	if len(sizeMatch) > 1 {
		if size, err := strconv.ParseFloat(sizeMatch[1], 64); err == nil {
			fish.Size = size
		}
	}

	// Try to extract value
	valueMatch := regexp.MustCompile(`"value"\s*:\s*([0-9.]+)`).FindStringSubmatch(text)
	if len(valueMatch) > 1 {
		if value, err := strconv.ParseFloat(valueMatch[1], 64); err == nil {
			fish.Value = value
		}
	}

	// Try to extract favorite_weather
	weatherMatch := regexp.MustCompile(`"favorite_weather"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(weatherMatch) > 1 {
		fish.FavoriteWeather = weatherMatch[1]
	}

	// Try to extract catch_chance
	chanceMatch := regexp.MustCompile(`"catch_chance"\s*:\s*([0-9.]+)`).FindStringSubmatch(text)
	if len(chanceMatch) > 1 {
		if chance, err := strconv.ParseFloat(chanceMatch[1], 64); err == nil {
			fish.CatchChance = chance
		}
	}

	// Try to extract existence_reason
	reasonMatch := regexp.MustCompile(`"existence_reason"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(reasonMatch) > 1 {
		fish.ExistenceReason = reasonMatch[1]
	}

	// Try to extract origin_context
	contextMatch := regexp.MustCompile(`"origin_context"\s*:\s*"([^"]+)"`).FindStringSubmatch(text)
	if len(contextMatch) > 1 {
		fish.OriginContext = contextMatch[1]
	}

	log.Printf("Reconstructed partial fish data: %s", fish.Name)
	return fish, nil
}

// repairJSON attempts to fix incomplete JSON
func (c *GeminiClient) repairJSON(jsonStr string) (string, error) {
	// Count opening and closing braces
	openBraces := strings.Count(jsonStr, "{")
	closeBraces := strings.Count(jsonStr, "}")

	// If we have more opening braces than closing, add the missing closing braces
	if openBraces > closeBraces {
		jsonStr = jsonStr + strings.Repeat("}", openBraces-closeBraces)
	}

	// Check if we're missing a comma between properties
	fixedJson := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"\s*"([^"]+)"`).ReplaceAllString(jsonStr, "\"$1\":\"$2\",\"$3\"")

	// Fix any trailing commas before closing braces
	fixedJson = regexp.MustCompile(`,\s*}`).ReplaceAllString(fixedJson, "}")

	return fixedJson, nil
}

// Close closes the Gemini client
func (c *GeminiClient) Close() error {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
	return nil
}

// GenerateUniqueFishFromContext uses Gemini to generate a unique fish based on multiple data sources
func (c *GeminiClient) GenerateUniqueFishFromContext(ctx context.Context, contextData map[string]interface{}, reason string) (*FishGenerationResponse, error) {
	// Extract API key from context
	apiKey, ok := ctx.Value("gemini_api_key").(string)
	if !ok || apiKey == "" {
		log.Printf("ERROR: No API key provided for Gemini. Fish generation will fail.")
		return nil, fmt.Errorf("no API key provided in context")
	}

	// Create a fresh client for each request to avoid stale connections
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("ERROR: Failed to create Gemini client: %v", err)
		return nil, err
	}
	defer client.Close()

	// Create a model instance directly
	model := client.GenerativeModel("gemma-3-27b-it")

	// Configure model settings
	model.SetTemperature(0.85) // Slightly reduced for more consistent output
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"

	// Start a fresh chat session
	session := model.StartChat()

	// Build prompt text with comprehensive context
	prompt := c.buildComprehensivePrompt(contextData, reason)

	log.Printf("Sending request to Gemini API using model: gemma-3-27b-it")
	log.Printf("Context includes %d data sources for unique fish generation", len(contextData))

	// Send the message
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Gemini API request failed: %v", err)
		return nil, fmt.Errorf("error sending message: %w", err)
	}

	// Process response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("ERROR: Empty response from Gemini model")
		return nil, fmt.Errorf("empty response from model")
	}

	// Extract response text
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if str, ok := part.(genai.Text); ok {
			responseText += string(str)
		}
	}

	// Parse the JSON response
	fish, err := c.parseFishResponse(responseText)
	if err != nil {
		log.Printf("ERROR: Failed to parse Gemini response: %v", err)
		log.Printf("Response text: %s", responseText)
		return nil, fmt.Errorf("failed to parse model response: %w", err)
	}

	log.Printf("Successfully generated unique fish using Gemini: %s (Rarity: %s)", fish.Name, fish.Rarity)
	return fish, nil
}

// buildComprehensivePrompt creates a detailed prompt for the Gemini API with multiple data sources
func (c *GeminiClient) buildComprehensivePrompt(contextData map[string]interface{}, reason string) string {
	var newsHeadline, newsCategory, sentiment string
	var mergedNewsHeadline, mergedNewsCategory, mergedNewsSentiment string
	var weatherCondition, weatherDescription string
	var temperature float64
	var bitcoinPrice, bitcoinChange, goldPrice, goldChange float64
	var hasMergedNews bool

	// Extract news data
	if news, ok := contextData["news"]; ok {
		if newsItem, ok := news.(*NewsItem); ok {
			newsHeadline = newsItem.Headline
			newsCategory = newsItem.Category

			if newsItem.Sentiment > 0.3 {
				sentiment = "positive"
			} else if newsItem.Sentiment < -0.3 {
				sentiment = "negative"
			} else {
				sentiment = "neutral"
			}
		}
	}

	// Extract merged news data if available
	if mergedNews, ok := contextData["merged_news"]; ok {
		hasMergedNews = true
		if newsItem, ok := mergedNews.(*NewsItem); ok {
			mergedNewsHeadline = newsItem.Headline
			mergedNewsCategory = newsItem.Category

			if newsItem.Sentiment > 0.3 {
				mergedNewsSentiment = "positive"
			} else if newsItem.Sentiment < -0.3 {
				mergedNewsSentiment = "negative"
			} else {
				mergedNewsSentiment = "neutral"
			}
		}
	}

	// Extract weather data
	if weather, ok := contextData["weather"]; ok {
		type WeatherGetter interface {
			GetCondition() string
			GetTempC() float64
			GetDescription() string
		}

		if w, ok := weather.(WeatherGetter); ok {
			weatherCondition = w.GetCondition()
			temperature = w.GetTempC()
			weatherDescription = w.GetDescription()
		} else if w, ok := weather.(*struct {
			Condition   string  `json:"condition"`
			TempC       float64 `json:"temp_c"`
			Description string  `json:"description"`
		}); ok {
			weatherCondition = w.Condition
			temperature = w.TempC
			weatherDescription = w.Description
		}
	}

	// Extract bitcoin data
	if bitcoin, ok := contextData["bitcoin"]; ok {
		type CryptoGetter interface {
			GetPriceUSD() float64
			GetChange24h() float64
		}

		if btc, ok := bitcoin.(CryptoGetter); ok {
			bitcoinPrice = btc.GetPriceUSD()
			bitcoinChange = btc.GetChange24h()
		} else if btc, ok := bitcoin.(*struct {
			PriceUSD  float64 `json:"price_usd"`
			Change24h float64 `json:"change_24h"`
		}); ok {
			bitcoinPrice = btc.PriceUSD
			bitcoinChange = btc.Change24h
		}
	}

	// Extract gold data
	if gold, ok := contextData["gold"]; ok {
		type GoldGetter interface {
			GetPriceUSD() float64
			GetChange24h() float64
		}

		if g, ok := gold.(GoldGetter); ok {
			goldPrice = g.GetPriceUSD()
			goldChange = g.GetChange24h()
		} else if g, ok := gold.(*struct {
			PriceUSD  float64 `json:"price_usd"`
			Change24h float64 `json:"change_24h"`
		}); ok {
			goldPrice = g.PriceUSD
			goldChange = g.Change24h
		}
	}

	// Determine if context is economic-focused
	isEconomicContext := false
	if newsCategory == "business" || newsCategory == "economy" || newsCategory == "finance" ||
		strings.Contains(strings.ToLower(newsHeadline), "econom") ||
		strings.Contains(strings.ToLower(newsHeadline), "market") ||
		strings.Contains(strings.ToLower(newsHeadline), "financ") ||
		strings.Contains(strings.ToLower(newsHeadline), "stock") {
		isEconomicContext = true
	}

	// Check merged news for economic context too
	if hasMergedNews && (mergedNewsCategory == "business" || mergedNewsCategory == "economy" || mergedNewsCategory == "finance" ||
		strings.Contains(strings.ToLower(mergedNewsHeadline), "econom") ||
		strings.Contains(strings.ToLower(mergedNewsHeadline), "market") ||
		strings.Contains(strings.ToLower(mergedNewsHeadline), "financ") ||
		strings.Contains(strings.ToLower(mergedNewsHeadline), "stock")) {
		isEconomicContext = true
	}

	// Build the context description, adding merged news if available
	contextDescription := buildContextDescriptionWithMergedNews(
		newsHeadline, newsCategory, sentiment,
		mergedNewsHeadline, mergedNewsCategory, mergedNewsSentiment, hasMergedNews,
		weatherCondition, weatherDescription, temperature,
		bitcoinPrice, bitcoinChange, goldPrice, goldChange)

	// Create the prompt template - UPDATED for efficiency
	promptTemplate := `You are a creative fish species designer for a fishing game. 
Create a completely unique and imaginative fish species that does not exist in the real world,
inspired by the following data sources. Focus on creative aspects like appearance, behavior, and abilities.

AVAILABLE CONTEXT DATA:
%s

GENERATION REASON: %s
ECONOMIC CONTEXT: %t

DESIGN FOCUS - CREATIVE ASPECTS ONLY:
- Create a fish with a highly unique NAME and APPEARANCE that has never been seen before
- Design a distinctive COLOR scheme that reflects the context data
- Invent a specific DIET that would make sense for the fish's habitat and appearance
- Create a HABITAT description that connects to the available weather/economic data
- Define a FAVORITE_WEATHER condition when this fish is most likely to be caught
- Provide a creative EFFECT that happens when the player catches this fish
- Write a compelling EXISTENCE_REASON that explains WHY this fish evolved or appeared

IMPORTANT GUIDELINES:
- Make the fish's name ABSOLUTELY UNIQUE - it should not match any real fish species
- No need to focus on size, rarity, value or catch chance - those will be calculated separately
- Create abilities and effects that reflect the combined context of all data sources
- For economic news, the fish's appearance should incorporate gold/digital elements
- Bitcoin changes should influence technological or energetic aspects
- Gold changes should influence the luster and material qualities
- Weather conditions should influence the fish's preferred habitat and appearance

CRITICAL RESPONSE FORMAT INSTRUCTIONS:
1. Respond with ONLY a valid JSON object - no Markdown formatting, no code blocks
2. Make sure to include ONLY the fields shown in the template below
3. Use valid JSON format with double quotes around property names and string values
4. Do not include numeric fields like size, value, or catch_chance
5. Make sure all JSON properties are properly separated by commas
6. The response must be a complete, properly formatted JSON object

JSON OBJECT TEMPLATE:
{
  "name": "A highly unique and creative name for the fish species that doesn't exist in reality",
  "description": "A short description including habitat and notable behaviors",
  "appearance": "A vivid description of colors, patterns, and unique features",
  "color": "The main color(s) of the fish (e.g., 'iridescent blue', 'golden')",
  "diet": "What the fish eats (affects bait selection in game)",
  "habitat": "The specific environment where this fish lives",
  "effect": "A gameplay effect when caught (be creative!)",
  "favorite_weather": "The specific weather condition when this fish is most active",
  "existence_reason": "Why this fish evolved or appeared, connecting to the context data",
  "origin_context": "A brief explanation of how the data context influenced this fish's creation"
}

REMEMBER: Return ONLY the valid JSON object with NO markdown formatting or code blocks.`

	// Format the prompt with the context data
	prompt := fmt.Sprintf(promptTemplate, contextDescription, reason, isEconomicContext)

	return prompt
}

// Helper function to build context description from available data, including merged news
func buildContextDescriptionWithMergedNews(newsHeadline, newsCategory, sentiment,
	mergedNewsHeadline, mergedNewsCategory, mergedNewsSentiment string, hasMergedNews bool,
	weatherCondition, weatherDescription string, temperature float64,
	bitcoinPrice, bitcoinChange, goldPrice, goldChange float64) string {

	var contextParts []string

	if newsHeadline != "" {
		contextParts = append(contextParts, fmt.Sprintf("NEWS HEADLINE: \"%s\"", newsHeadline))
	}
	if newsCategory != "" {
		contextParts = append(contextParts, fmt.Sprintf("NEWS CATEGORY: %s", newsCategory))
	}
	if sentiment != "" {
		contextParts = append(contextParts, fmt.Sprintf("NEWS SENTIMENT: %s", sentiment))
	}

	// Add merged news if available
	if hasMergedNews && mergedNewsHeadline != "" {
		contextParts = append(contextParts, fmt.Sprintf("SECOND NEWS HEADLINE: \"%s\"", mergedNewsHeadline))
		if mergedNewsCategory != "" {
			contextParts = append(contextParts, fmt.Sprintf("SECOND NEWS CATEGORY: %s", mergedNewsCategory))
		}
		if mergedNewsSentiment != "" {
			contextParts = append(contextParts, fmt.Sprintf("SECOND NEWS SENTIMENT: %s", mergedNewsSentiment))
		}
	}

	if weatherCondition != "" {
		contextParts = append(contextParts, fmt.Sprintf("WEATHER CONDITION: %s", weatherCondition))
	}
	if weatherDescription != "" {
		contextParts = append(contextParts, fmt.Sprintf("WEATHER DETAILS: %s", weatherDescription))
	}
	if temperature != 0 {
		contextParts = append(contextParts, fmt.Sprintf("TEMPERATURE: %.1f°C", temperature))
	}
	if bitcoinPrice != 0 {
		contextParts = append(contextParts, fmt.Sprintf("BITCOIN PRICE: $%.2f (%.2f%% change)", bitcoinPrice, bitcoinChange))
	}
	if goldPrice != 0 {
		contextParts = append(contextParts, fmt.Sprintf("GOLD PRICE: $%.2f (%.2f%% change)", goldPrice, goldChange))
	}

	return strings.Join(contextParts, "\n")
}
