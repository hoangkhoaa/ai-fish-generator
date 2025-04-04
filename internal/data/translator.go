package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// TranslationFields defines the fields to be translated
type TranslationFields struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Appearance      string `json:"appearance"`
	Color           string `json:"color"`
	Diet            string `json:"diet"`
	Habitat         string `json:"habitat"`
	Effect          string `json:"effect"`
	FavoriteWeather string `json:"favorite_weather"`
	ExistenceReason string `json:"existence_reason"`
}

// TranslatedFish contains both original fish ID and translated content
type TranslatedFish struct {
	OriginalID      string    `json:"original_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Appearance      string    `json:"appearance"`
	Color           string    `json:"color"`
	Diet            string    `json:"diet"`
	Habitat         string    `json:"habitat"`
	Effect          string    `json:"effect"`
	FavoriteWeather string    `json:"favorite_weather"`
	ExistenceReason string    `json:"existence_reason"`
	TranslatedAt    time.Time `json:"translated_at"`
}

// TranslatorClient provides functionality to translate fish content to Vietnamese
type TranslatorClient struct {
	client *genai.Client
	model  string
}

// NewTranslatorClient creates a new translator client
func NewTranslatorClient(apiKey string) *TranslatorClient {
	return &TranslatorClient{
		model: "gemma-3-27b-it", // Using Gemma model for translation
	}
}

// initClient initializes the Gemini client if it hasn't been already
func (t *TranslatorClient) initClient(ctx context.Context, apiKey string) error {
	if t.client != nil {
		return nil
	}

	var err error
	t.client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client for translation: %w", err)
	}

	return nil
}

// TranslateFish translates the fish content to Vietnamese
func (t *TranslatorClient) TranslateFish(ctx context.Context, fishID string, fields TranslationFields) (*TranslatedFish, error) {
	// Extract API key from context
	apiKey, ok := ctx.Value("gemini_api_key").(string)
	if !ok || apiKey == "" {
		log.Printf("ERROR: No API key provided for Gemini. Translation will fail.")
		return nil, fmt.Errorf("no API key provided in context")
	}

	// Create a fresh client for each request to avoid stale connections
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("ERROR: Failed to create Gemini client for translation: %v", err)
		return nil, err
	}
	defer client.Close()

	// Create a model instance
	model := client.GenerativeModel("gemma-3-27b-it")

	// Configure model settings
	model.SetTemperature(0.3) // Lower temperature for more accurate translations
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"

	// Start a fresh chat session
	session := model.StartChat()

	// Build the translation prompt
	prompt := t.buildTranslationPrompt(fields)

	log.Printf("Translating fish content to Vietnamese (Fish ID: %s)", fishID)

	// Send the message to Gemini
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Translation request failed: %v", err)
		return nil, fmt.Errorf("error sending translation request: %w", err)
	}

	// Process response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("ERROR: Empty response from translation model")
		return nil, fmt.Errorf("empty response from translation model")
	}

	// Extract response text
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if str, ok := part.(genai.Text); ok {
			responseText += string(str)
		}
	}

	// Parse the translation response
	translatedFields, err := t.parseTranslationResponse(responseText)
	if err != nil {
		log.Printf("ERROR: Failed to parse translation response: %v", err)
		return nil, fmt.Errorf("failed to parse translation response: %w", err)
	}

	// Create the translated fish object
	translatedFish := &TranslatedFish{
		OriginalID:      fishID,
		Name:            translatedFields.Name,
		Description:     translatedFields.Description,
		Appearance:      translatedFields.Appearance,
		Color:           translatedFields.Color,
		Diet:            translatedFields.Diet,
		Habitat:         translatedFields.Habitat,
		Effect:          translatedFields.Effect,
		FavoriteWeather: translatedFields.FavoriteWeather,
		ExistenceReason: translatedFields.ExistenceReason,
		TranslatedAt:    time.Now(),
	}

	log.Printf("Successfully translated fish content to Vietnamese (Fish ID: %s)", fishID)
	return translatedFish, nil
}

// buildTranslationPrompt creates a prompt for translating fish content to Vietnamese
func (t *TranslatorClient) buildTranslationPrompt(fields TranslationFields) string {
	return fmt.Sprintf(`You are a professional translator specializing in English to Vietnamese translation.
Your task is to translate the following fish description from a fishing game to Vietnamese.
Maintain the original meaning while making the Vietnamese text flow naturally.

Please translate ONLY the following fields:

Name: %s
Description: %s
Appearance: %s
Color: %s
Diet: %s
Habitat: %s
Effect: %s
Favorite Weather: %s
Existence Reason: %s

Respond with ONLY a JSON object containing the translated fields:
{
  "name": "translated name in Vietnamese",
  "description": "translated description in Vietnamese",
  "appearance": "translated appearance in Vietnamese",
  "color": "translated color in Vietnamese",
  "diet": "translated diet in Vietnamese",
  "habitat": "translated habitat in Vietnamese",
  "effect": "translated effect in Vietnamese",
  "favorite_weather": "translated favorite weather in Vietnamese",
  "existence_reason": "translated existence reason in Vietnamese"
}

Return ONLY the valid JSON object with no additional text.
`, fields.Name, fields.Description, fields.Appearance, fields.Color, fields.Diet,
		fields.Habitat, fields.Effect, fields.FavoriteWeather, fields.ExistenceReason)
}

// parseTranslationResponse extracts the translated fields from the Gemini response
func (t *TranslatorClient) parseTranslationResponse(response string) (*TranslationFields, error) {
	// Remove Markdown code block markers if present
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	response = strings.TrimSpace(response)

	// Extract JSON object from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("could not find valid JSON in translation response")
	}

	// Extract just the JSON part
	jsonStr := response[jsonStart : jsonEnd+1]

	// Parse the JSON response
	var translatedFields TranslationFields
	err := json.Unmarshal([]byte(jsonStr), &translatedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to parse translation JSON: %w", err)
	}

	return &translatedFields, nil
}

// Close releases resources used by the translator client
func (t *TranslatorClient) Close() error {
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}
	return nil
}
