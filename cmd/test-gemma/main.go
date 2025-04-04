package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// FishResponse is a simple structure for the fish response
type FishResponse struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Appearance  string  `json:"appearance"`
	Effect      string  `json:"effect"`
	Rarity      string  `json:"rarity"`
	Size        float64 `json:"size"`
	SizeUnits   string  `json:"size_units"`
	Value       float64 `json:"value"`
}

func main() {
	// Load API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatalln("GEMINI_API_KEY environment variable not set")
	}

	fmt.Println("Testing Gemma Fish Generator...")
	fmt.Println("Using API key:", maskAPIKey(apiKey))

	ctx := context.Background()

	// Create client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	// Create model
	model := client.GenerativeModel("gemma-3-27b-it")

	// Configure model
	model.SetTemperature(0.9)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"

	// Start chat session
	session := model.StartChat()

	// Create a prompt for fish generation
	prompt := `You are a creative fish species designer for a fishing game. 
Create a unique and imaginative fish species inspired by the following news headline:

NEWS HEADLINE: "New technology breakthrough could revolutionize renewable energy"
NEWS CATEGORY: Technology
SENTIMENT: positive

The fish should have characteristics that reflect the theme, content, and sentiment of the news.
Respond with a JSON object containing ONLY the following fields:
{
  "name": "A unique and creative name for the fish species",
  "description": "A short description of the fish, including any relevant traits or abilities",
  "appearance": "A vivid description of the fish's physical appearance",
  "effect": "A gameplay effect that the fish provides to the player",
  "rarity": "One of: Common, Uncommon, Rare, Epic, Legendary",
  "size": "A number representing the size of the fish",
  "size_units": "meters",
  "value": "A number representing the market value of the fish in USD"
}

Be creative, thematic, and make sure the fish's characteristics reflect the news content.
The more unusual or significant the news, the more rare and valuable the fish should be.
Return ONLY the valid JSON object with no additional text.`

	// Send the message
	fmt.Println("Sending request to Gemma model...")
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}

	// Extract response text
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if str, ok := part.(genai.Text); ok {
			responseText += string(str)
		}
	}

	fmt.Println("\nResponse from model:")
	fmt.Println("--------------------")
	fmt.Println(responseText)
	fmt.Println("--------------------")

	// Try to parse as JSON
	fmt.Println("\nAttempting to parse response as JSON...")

	// Extract JSON if needed
	jsonStart := -1
	jsonEnd := -1
	for i := 0; i < len(responseText); i++ {
		if responseText[i] == '{' {
			jsonStart = i
			break
		}
	}
	if jsonStart >= 0 {
		bracketCount := 1
		for i := jsonStart + 1; i < len(responseText); i++ {
			if responseText[i] == '{' {
				bracketCount++
			} else if responseText[i] == '}' {
				bracketCount--
				if bracketCount == 0 {
					jsonEnd = i
					break
				}
			}
		}
	}

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := responseText[jsonStart : jsonEnd+1]
		var fish FishResponse
		err = json.Unmarshal([]byte(jsonStr), &fish)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
		} else {
			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("🤖 AI-GENERATED FISH (GEMMA MODEL) 🤖")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Printf("Name: %s\n", fish.Name)
			fmt.Printf("Rarity: %s\n", fish.Rarity)
			fmt.Printf("Size: %.2f %s\n", fish.Size, fish.SizeUnits)
			fmt.Printf("Value: $%.2f\n", fish.Value)
			fmt.Printf("Description: %s\n", fish.Description)
			fmt.Printf("Appearance: %s\n", fish.Appearance)
			fmt.Printf("Effect: %s\n", fish.Effect)
			fmt.Println("Note: Game stat effects will be applied when this fish is integrated into the game.")
			fmt.Println(strings.Repeat("=", 50))
		}
	} else {
		fmt.Println("Could not find valid JSON in response")
	}
}

// maskAPIKey returns a masked version of the API key for logging
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
