package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// TranslationFields represents the fields from a fish that need translation
type TranslationFields struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Color           string `json:"color"`
	Diet            string `json:"diet"`
	Habitat         string `json:"habitat,omitempty"`
	FavoriteWeather string `json:"favorite_weather"`
	ExistenceReason string `json:"existence_reason"`
	Effect          string `json:"effect,omitempty"`        // Keeping for backward compatibility
	PlayerEffect    string `json:"player_effect,omitempty"` // Keeping for backward compatibility

	// New fields for stat effects
	StatEffects []StatEffectTranslation `json:"stat_effects,omitempty"`
}

// StatEffectTranslation represents an individual stat effect that needs translation
type StatEffectTranslation struct {
	ID          string `json:"id"`          // Identifier for the effect (used for mapping)
	EffectType  string `json:"effect_type"` // Type of effect (environment, player, etc.)
	Description string `json:"description"` // Description to translate
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

// TranslatorClient handles translation of fish content to Vietnamese
type TranslatorClient struct {
	apiKey     string
	client     *genai.Client
	model      string
	clientOnce sync.Once
	mu         sync.Mutex
}

// NewTranslatorClient creates a new translator client
func NewTranslatorClient(apiKey string) *TranslatorClient {
	return &TranslatorClient{
		apiKey: apiKey,
		model:  "gemma-3-27b-it", // Using the Pro model for translation tasks
	}
}

// initClient initializes the Gemini client (only once)
func (t *TranslatorClient) initClient(ctx context.Context) error {
	var initErr error

	t.clientOnce.Do(func() {
		// Extract API key from context or use the one provided at init
		apiKey := t.apiKey
		if ctxKey, ok := ctx.Value("gemini_api_key").(string); ok && ctxKey != "" {
			apiKey = ctxKey
		}

		if apiKey == "" {
			initErr = fmt.Errorf("no API key provided for Gemini client")
			return
		}

		client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			initErr = fmt.Errorf("failed to create Gemini client: %w", err)
			return
		}

		t.client = client
		log.Printf("Gemini client initialized for translation using model: %s", t.model)
	})

	return initErr
}

// TranslateFish translates the provided fish fields to Vietnamese
func (t *TranslatorClient) TranslateFish(ctx context.Context, fields TranslationFields) (*TranslationFields, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Initialize the client if it's not initialized yet
	if err := t.initClient(ctx); err != nil {
		return nil, err
	}

	// Build the translation prompt
	prompt := t.buildTranslationPrompt(fields)

	// Create a model instance
	model := t.client.GenerativeModel(t.model)

	// Configure the model
	model.SetTemperature(0.2) // Low temperature for more consistent translations
	model.SetTopP(0.95)       // Balanced top-p for translation quality
	model.SetTopK(40)         // Standard top-k

	// Send the translation request
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("translation request failed: %w", err)
	}

	// Check if we got a valid response
	if resp == nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from translation service")
	}

	// Extract the response text
	responsePart := resp.Candidates[0].Content.Parts[0]
	responseText, ok := responsePart.(genai.Text)
	if !ok || string(responseText) == "" {
		return nil, fmt.Errorf("empty or invalid text in translation response")
	}

	// Parse the response to extract the translated fields
	translatedFields, err := t.parseTranslationResponse(string(responseText))
	if err != nil {
		return nil, fmt.Errorf("failed to parse translation response: %w", err)
	}

	return translatedFields, nil
}

// buildTranslationPrompt creates a prompt for the Gemini API to translate fish content
func (t *TranslatorClient) buildTranslationPrompt(fields TranslationFields) string {
	// Build stat effects section if we have any
	statEffectsPrompt := ""
	if len(fields.StatEffects) > 0 {
		statEffectsPrompt = "Stat Effects:\n"
		for i, effect := range fields.StatEffects {
			statEffectsPrompt += fmt.Sprintf("- Effect %d (Type: %s): %s\n",
				i+1, effect.EffectType, effect.Description)
		}
	} else if fields.Effect != "" || fields.PlayerEffect != "" {
		// Include legacy effect fields if no stat effects are provided
		statEffectsPrompt = "Effects:\n"
		if fields.Effect != "" {
			statEffectsPrompt += fmt.Sprintf("- Environment Effect: %s\n", fields.Effect)
		}
		if fields.PlayerEffect != "" {
			statEffectsPrompt += fmt.Sprintf("- Player Effect: %s\n", fields.PlayerEffect)
		}
	}

	return fmt.Sprintf(`
You are a professional translator specializing in Vietnamese translations for a fish-themed game.
Please translate the following fish description fields from English to Vietnamese.
Maintain the tone and style, but adapt cultural references as needed for Vietnamese speakers.
Format your response as a valid JSON object containing only the translated fields.

Original fields:
- Name: %s
- Description: %s
- Color: %s
- Diet: %s
- Habitat: %s
- Favorite Weather: %s
- Existence Reason: %s
%s

Return only a JSON object with the translated fields in this exact format:
{
  "name": "[Vietnamese translation]",
  "description": "[Vietnamese translation]",
  "color": "[Vietnamese translation]",
  "diet": "[Vietnamese translation]",
  "habitat": "[Vietnamese translation]",
  "favorite_weather": "[Vietnamese translation]",
  "existence_reason": "[Vietnamese translation]",
  %s
}
`, fields.Name, fields.Description, fields.Color, fields.Diet,
		fields.Habitat, fields.FavoriteWeather, fields.ExistenceReason,
		statEffectsPrompt,
		t.buildStatEffectsResponseTemplate(fields.StatEffects))
}

// buildStatEffectsResponseTemplate creates the template part of the prompt for stat effects
func (t *TranslatorClient) buildStatEffectsResponseTemplate(effects []StatEffectTranslation) string {
	if len(effects) > 0 {
		template := "\"stat_effects\": [\n"
		for i, effect := range effects {
			template += fmt.Sprintf("    {\"id\": \"%s\", \"description\": \"[Vietnamese translation]\"}", effect.ID)
			if i < len(effects)-1 {
				template += ","
			}
			template += "\n"
		}
		template += "  ]"
		return template
	} else {
		// Backward compatibility
		return "\"effect\": \"[Vietnamese translation]\",\n  \"player_effect\": \"[Vietnamese translation]\""
	}
}

// SanitizeUTF8 ensures that all strings are valid UTF-8 before storing in MongoDB
// Export this function so it can be used by other packages
func SanitizeUTF8(s string) string {
	// Early return for empty strings
	if s == "" {
		return s
	}

	// Log if the string contains "MISSING" or special formatters for debugging
	if strings.Contains(s, "MISSING") || strings.Contains(s, "%!") {
		log.Printf("Found potentially problematic string: %s", s)
	}

	// Layer 1: Basic validation with replacement
	sanitized := strings.ToValidUTF8(s, "\uFFFD")

	// Layer 2: Handle specific problematic characters
	// Replace common problematic characters that might cause BSON issues
	problematicReplacements := map[string]string{
		// Common control characters that might cause issues
		"\u0000": "", // NULL
		"\u0001": "", // START OF HEADING
		"\u0002": "", // START OF TEXT
		"\u0003": "", // END OF TEXT
		"\u0004": "", // END OF TRANSMISSION
		"\u0005": "", // ENQUIRY
		"\u0006": "", // ACKNOWLEDGE
		"\u0007": "", // BELL
		"\u0008": "", // BACKSPACE
		"\u000B": "", // VERTICAL TAB
		"\u000C": "", // FORM FEED
		"\u000E": "", // SHIFT OUT
		"\u000F": "", // SHIFT IN
		"\u0010": "", // DATA LINK ESCAPE
		"\u0011": "", // DEVICE CONTROL 1
		"\u0012": "", // DEVICE CONTROL 2
		"\u0013": "", // DEVICE CONTROL 3
		"\u0014": "", // DEVICE CONTROL 4
		"\u0015": "", // NEGATIVE ACKNOWLEDGE
		"\u0016": "", // SYNCHRONOUS IDLE
		"\u0017": "", // END OF TRANSMISSION BLOCK
		"\u0018": "", // CANCEL
		"\u0019": "", // END OF MEDIUM
		"\u001A": "", // SUBSTITUTE
		"\u001B": "", // ESCAPE
		"\u001C": "", // INFORMATION SEPARATOR FOUR
		"\u001D": "", // INFORMATION SEPARATOR THREE
		"\u001E": "", // INFORMATION SEPARATOR TWO
		"\u001F": "", // INFORMATION SEPARATOR ONE

		// Special symbols that might cause BSON issues
		"\uFFFD": "", // Replace replacement character with nothing

		// Special formatting/encoding issues
		"%!S(MISSING)": "%", // Fix for broken %S format specifier
		"%!s(MISSING)": "%", // Fix for broken %s format specifier
		"%!d(MISSING)": "%", // Fix for broken %d format specifier
		"%!v(MISSING)": "%", // Fix for broken %v format specifier
		"%!f(MISSING)": "%", // Fix for broken %f format specifier
		"%!(MISSING)":  "%", // Generic fix for broken % format specifier
		"%%":           "%", // Double percent sign
	}

	for char, replacement := range problematicReplacements {
		sanitized = strings.ReplaceAll(sanitized, char, replacement)
	}

	// Layer 2.5: Fix common formatting artifacts from headlines
	// This regex-like approach handles cases where we have formatting artifacts
	sanitized = strings.ReplaceAll(sanitized, "%!S", "%")
	sanitized = strings.ReplaceAll(sanitized, "%!s", "%")
	sanitized = strings.ReplaceAll(sanitized, "%!d", "%")
	sanitized = strings.ReplaceAll(sanitized, "%!v", "%")
	sanitized = strings.ReplaceAll(sanitized, "%!f", "%")
	sanitized = strings.ReplaceAll(sanitized, "(MISSING)", "")

	// Layer 2.6: Handle specific patterns that might appear in news headlines with numbers
	// Look for patterns like "2,230%!S(MISSING)urge" and fix them
	// This is especially important for financial news articles with percentages

	// First pass: fix percent formatting issues
	if strings.Contains(sanitized, "%!") {
		// Find all instances where we have digits followed by '%!'
		// Typical pattern: "2,230%!S(MISSING)urge" should become "2,230% Surge"
		parts := strings.Split(sanitized, "%!")
		if len(parts) > 1 {
			// The first part will be before the '%!'
			result := parts[0] + "%"

			for i := 1; i < len(parts); i++ {
				part := parts[i]

				// Check if this part starts with a format specifier pattern
				if len(part) > 0 && strings.ContainsAny(string(part[0]), "SsdvfF") {
					// If it starts with a format specifier, take the rest of the string
					if len(part) > 1 {
						// Skip the format specifier character and any "MISSING" or parentheses
						idx := 1
						// Skip past (MISSING) if present
						if strings.HasPrefix(part[idx:], "(MISSING)") {
							idx += len("(MISSING)")
						}
						// Add a space if the next character is a letter (for readability)
						if idx < len(part) && ((part[idx] >= 'a' && part[idx] <= 'z') || (part[idx] >= 'A' && part[idx] <= 'Z')) {
							result += " " + part[idx:]
						} else {
							result += part[idx:]
						}
					}
				} else {
					// If it doesn't start with a format specifier, just concatenate as is
					result += part
				}
			}

			sanitized = result
		}
	}

	// Layer 3: Final validation to ensure we have valid UTF-8
	if !utf8.ValidString(sanitized) {
		// If we still have invalid UTF-8, replace all non-ASCII characters with spaces
		// This is a last resort that ensures we'll have valid data
		result := ""
		for _, r := range sanitized {
			if r < 128 && r >= 32 { // ASCII printable range
				result += string(r)
			} else {
				result += " " // Replace with space
			}
		}
		return result
	}

	// Remove any leading/trailing whitespace that might have been introduced
	sanitized = strings.TrimSpace(sanitized)

	// If the string was modified, log the change for debugging
	if sanitized != s {
		log.Printf("String sanitized: '%s' -> '%s'", s, sanitized)
	}

	return sanitized
}

// parseTranslationResponse extracts the JSON data from the Gemini response
func (t *TranslatorClient) parseTranslationResponse(response string) (*TranslationFields, error) {
	// Ensure response is valid UTF-8 first
	response = SanitizeUTF8(response)

	// Extract JSON content from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("could not find valid JSON in response: %s", response)
	}

	jsonContent := response[jsonStart : jsonEnd+1]

	// Parse the JSON into TranslationFields
	var translatedFields TranslationFields
	if err := json.Unmarshal([]byte(jsonContent), &translatedFields); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Sanitize all fields to ensure valid UTF-8
	translatedFields.Name = SanitizeUTF8(translatedFields.Name)
	translatedFields.Description = SanitizeUTF8(translatedFields.Description)
	translatedFields.Color = SanitizeUTF8(translatedFields.Color)
	translatedFields.Diet = SanitizeUTF8(translatedFields.Diet)
	translatedFields.Habitat = SanitizeUTF8(translatedFields.Habitat)
	translatedFields.FavoriteWeather = SanitizeUTF8(translatedFields.FavoriteWeather)
	translatedFields.ExistenceReason = SanitizeUTF8(translatedFields.ExistenceReason)
	translatedFields.Effect = SanitizeUTF8(translatedFields.Effect)
	translatedFields.PlayerEffect = SanitizeUTF8(translatedFields.PlayerEffect)

	// Sanitize stat effects
	for i := range translatedFields.StatEffects {
		translatedFields.StatEffects[i].Description = SanitizeUTF8(translatedFields.StatEffects[i].Description)
	}

	return &translatedFields, nil
}

// Close releases resources used by the translator client
func (t *TranslatorClient) Close() {
	if t.client != nil {
		t.client.Close()
		log.Println("Translator client closed")
	}
}
