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
		return nil, fmt.Errorf("no API key provided in context")
	}

	// Initialize the client
	err := t.initClient(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize translation client: %v", err)
	}

	// Create a model instance
	model := t.client.GenerativeModel(t.model)

	// Configure model settings
	model.SetTemperature(0.3) // Lower temperature for more accurate translations
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)

	// Build translation prompt
	prompt := t.buildTranslationPrompt(fields)

	// Send translation request
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("translation API call failed: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from translation API")
	}

	// Extract text response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Parse the response
	translatedFields, err := t.parseTranslationResponse(responseText)
	if err != nil {
		return nil, err
	}

	// Validate the translated fields
	if translatedFields.Name == "" || translatedFields.Description == "" {
		return nil, fmt.Errorf("critical translated fields are missing or empty")
	}

	// Create TranslatedFish record
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

	return translatedFish, nil
}

// buildTranslationPrompt creates a prompt for translating fish content to Vietnamese
func (t *TranslatorClient) buildTranslationPrompt(fields TranslationFields) string {
	return `Translate the following fish information from English to Vietnamese. 
Keep the translation natural and fluent in Vietnamese.
Just return the translated content exactly in the same JSON format, without any additional text.
Do not include any instructions, notes, or metadata in your response.

Input JSON to translate:
{
  "name": "` + fields.Name + `",
  "description": "` + fields.Description + `",
  "appearance": "` + fields.Appearance + `",
  "color": "` + fields.Color + `",
  "diet": "` + fields.Diet + `",
  "habitat": "` + fields.Habitat + `",
  "effect": "` + fields.Effect + `",
  "favorite_weather": "` + fields.FavoriteWeather + `",
  "existence_reason": "` + fields.ExistenceReason + `"
}

Your response must be a valid JSON object containing exactly these fields with their translations in Vietnamese. Do not include any explanatory text before or after the JSON.`
}

// parseTranslationResponse extracts the translated fields from the API response
func (t *TranslatorClient) parseTranslationResponse(response string) (TranslationFields, error) {
	var fields TranslationFields

	// Look for the JSON object
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return fields, fmt.Errorf("invalid JSON response format: %s", response)
	}

	// Extract just the JSON part
	jsonStr := response[jsonStart : jsonEnd+1]

	// Clean up any markdown code block markers
	jsonStr = strings.ReplaceAll(jsonStr, "```json", "")
	jsonStr = strings.ReplaceAll(jsonStr, "```", "")
	jsonStr = strings.TrimSpace(jsonStr)

	// Try to unmarshal the JSON
	err := json.Unmarshal([]byte(jsonStr), &fields)
	if err != nil {
		return fields, fmt.Errorf("failed to parse translation response: %v\nResponse: %s", err, jsonStr)
	}

	// Double check for empty fields
	if fields.Name == "" || fields.Description == "" {
		log.Printf("Warning: Some translated fields are empty: %+v", fields)
	}

	return fields, nil
}

// Close releases resources used by the translator client
func (t *TranslatorClient) Close() error {
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}
	return nil
}
