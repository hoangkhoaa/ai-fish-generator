package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
	model.SetTemperature(0.9) // Slightly reduced from 1.0 for more consistent output
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
	model.SetTemperature(0.9) // Slightly reduced for more consistent output
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

// buildComprehensivePrompt constructs the detailed prompt for Gemini based on the context data
func (c *GeminiClient) buildComprehensivePrompt(contextData map[string]interface{}, reason string) string {
	// Variables to hold merged news data if available
	var mergedNewsHeadlines []string
	var mergedNewsCategories []string
	var mergedNewsSentiments []float64
	hasMergedNews := false

	// Check for merged news items
	if mergedNews, ok := contextData["merged_news"].([]*NewsItem); ok && len(mergedNews) > 0 {
		hasMergedNews = true

		// Extract data from each merged news item
		for _, news := range mergedNews {
			mergedNewsHeadlines = append(mergedNewsHeadlines, news.Headline)
			mergedNewsCategories = append(mergedNewsCategories, news.Category)
			mergedNewsSentiments = append(mergedNewsSentiments, news.Sentiment)
		}
	} else if singleMergedNews, ok := contextData["merged_news"].(*NewsItem); ok && singleMergedNews != nil {
		// Handle backward compatibility with single merged news
		hasMergedNews = true
		mergedNewsHeadlines = append(mergedNewsHeadlines, singleMergedNews.Headline)
		mergedNewsCategories = append(mergedNewsCategories, singleMergedNews.Category)
		mergedNewsSentiments = append(mergedNewsSentiments, singleMergedNews.Sentiment)
	}

	// Extract primary news data
	var newsHeadline string
	var newsCategory string
	var newsSentiment float64

	if news, ok := contextData["news"].(*NewsItem); ok && news != nil {
		newsHeadline = news.Headline
		newsCategory = news.Category
		newsSentiment = news.Sentiment
	}

	// Build context description
	contextDesc := c.buildContextDescriptionWithMergedNews(
		contextData,
		newsHeadline,
		newsCategory,
		newsSentiment,
		mergedNewsHeadlines,
		mergedNewsCategories,
		mergedNewsSentiments,
		hasMergedNews)

	// Set up the prompt template
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf(`
You are a creative AI that designs unique and imaginative fish species based on real-world contextual data.

Current Context:
%s

Your task is to create a new fish species inspired by this context. Be creative and imaginative!
`, contextDesc))

	// Add detailed instructions for the model
	prompt.WriteString(`
IMPORTANT: Design a fish that reflects the context and real-world data provided above. Your fish should have:

1. A creative name that's humorous, punny, or references the news/data
2. A detailed appearance description
3. Habitat and diet that make sense for this fish
4. A colorful and distinctive look that relates to the news or weather
5. An interesting effect or quality that makes this fish special

Do not provide numerical details for the following attributes, as these will be generated programmatically:
- Rarity level
- Size/length
- Weight
- Value
- Catch chance/difficulty

Respond with a JSON object that includes the following fields:
{
  "name": "The fish's creative name",
  "description": "Detailed, imaginative description",
  "appearance": "Physical characteristics and notable features",
  "color": "Primary colors and patterns",
  "diet": "What the fish eats",
  "habitat": "Where the fish lives",
  "effect": "Special quality or effect",
  "favorite_weather": "Weather condition this fish prefers",
  "existence_reason": "Brief explanation of why this fish evolved or exists"
}
`)

	return prompt.String()
}

// Helper function to describe sentiment as text
func describeSentiment(sentiment float64) string {
	if sentiment > 0.3 {
		return "positive"
	} else if sentiment < -0.3 {
		return "negative"
	}
	return "neutral"
}

// Helper function to infer a theme from multiple headlines
func inferThemeFromHeadlines(headlines []string, categories []string) string {
	// Simple implementation that combines category information
	categoryStr := strings.Join(categories, ", ")

	// Extract key phrases from headlines
	joinedHeadlines := strings.Join(headlines, " ")
	words := strings.Fields(joinedHeadlines)

	// Get most frequent meaningful words
	wordCount := make(map[string]int)
	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,;:!?\"'()[]{}"))
		if len(word) > 4 { // Only consider meaningful words
			wordCount[word]++
		}
	}

	// Find top 3 words
	type wordFreq struct {
		word  string
		count int
	}

	var freqs []wordFreq
	for word, count := range wordCount {
		freqs = append(freqs, wordFreq{word, count})
	}

	// Sort by frequency
	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].count > freqs[j].count
	})

	// Get top words
	var topWords []string
	for i := 0; i < 3 && i < len(freqs); i++ {
		topWords = append(topWords, freqs[i].word)
	}

	return fmt.Sprintf("A fish that combines elements from %s news with themes of %s",
		categoryStr, strings.Join(topWords, ", "))
}

// buildContextDescriptionWithMergedNews creates a detailed context description for the prompt
func (c *GeminiClient) buildContextDescriptionWithMergedNews(
	contextData map[string]interface{},
	newsHeadline string,
	newsCategory string,
	newsSentiment float64,
	mergedNewsHeadlines []string,
	mergedNewsCategories []string,
	mergedNewsSentiments []float64,
	hasMergedNews bool) string {

	var description strings.Builder

	// Current date and time
	currentTime := time.Now()
	description.WriteString(fmt.Sprintf("CURRENT DATE: %s\n\n", currentTime.Format("January 2, 2006")))

	// PRIMARY NEWS
	if newsHeadline != "" {
		description.WriteString("PRIMARY NEWS HEADLINE: " + newsHeadline + "\n")
		description.WriteString("CATEGORY: " + newsCategory + "\n")
		sentimentDesc := describeSentiment(newsSentiment)
		description.WriteString("SENTIMENT: " + sentimentDesc + "\n\n")
	}

	// MERGED NEWS (if available)
	if hasMergedNews {
		description.WriteString("RELATED NEWS HEADLINES:\n")
		for i, headline := range mergedNewsHeadlines {
			description.WriteString(fmt.Sprintf("%d. %s\n", i+1, headline))
			description.WriteString("   CATEGORY: " + mergedNewsCategories[i] + "\n")
			sentimentDesc := describeSentiment(mergedNewsSentiments[i])
			description.WriteString("   SENTIMENT: " + sentimentDesc + "\n")
		}
		description.WriteString("\n")
	}

	// ECONOMIC CONTEXT (check all news categories)
	allCategories := []string{newsCategory}
	allCategories = append(allCategories, mergedNewsCategories...)
	allHeadlines := []string{newsHeadline}
	allHeadlines = append(allHeadlines, mergedNewsHeadlines...)

	hasEconomicContext := false
	for _, category := range allCategories {
		if category == "business" || category == "economy" || strings.Contains(category, "finance") {
			hasEconomicContext = true
			break
		}
	}

	// Add economic context if relevant
	if hasEconomicContext {
		// Add bitcoin price if available
		if bitcoin, ok := contextData["bitcoin"].(*CryptoPrice); ok && bitcoin != nil {
			description.WriteString(fmt.Sprintf("BITCOIN PRICE: $%.2f (%.2f%% change)\n",
				bitcoin.PriceUSD, bitcoin.Change24h))
		}

		// Add gold price if available
		if gold, ok := contextData["gold"].(*GoldPrice); ok && gold != nil {
			description.WriteString(fmt.Sprintf("GOLD PRICE: $%.2f per ounce (%.2f%% change)\n",
				gold.PriceUSD, gold.Change24h))
		}
		description.WriteString("\n")
	}

	// WEATHER CONTEXT
	if weather, ok := contextData["weather"].(*WeatherInfo); ok && weather != nil {
		description.WriteString(fmt.Sprintf("CURRENT WEATHER: %s, %.1f°C\n",
			weather.Condition, weather.TempC))

		if weather.IsExtreme {
			description.WriteString("EXTREME WEATHER ALERT: This is unusual weather\n")
		}
		description.WriteString("\n")
	}

	// CONTEXTUAL THEME based on combining news
	description.WriteString("CONTEXTUAL THEME: ")
	if hasMergedNews {
		// Look for common themes across all news
		allHeadlines := []string{newsHeadline}
		allHeadlines = append(allHeadlines, mergedNewsHeadlines...)
		description.WriteString(fmt.Sprintf("Create a fish inspired by the following theme: %s\n\n",
			inferThemeFromHeadlines(allHeadlines, allCategories)))
	} else {
		description.WriteString(fmt.Sprintf("Create a fish inspired by %s news: %s\n\n",
			newsCategory, newsHeadline))
	}

	return description.String()
}
