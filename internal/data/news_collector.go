package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// NewsAPIResponse represents the API response from NewsAPI
type NewsAPIResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Source struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"source"`
		Author      string    `json:"author"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		URLToImage  string    `json:"urlToImage"`
		PublishedAt time.Time `json:"publishedAt"`
		Content     string    `json:"content"`
	} `json:"articles"`
}

// NewsCollector collects real news data from NewsAPI
type NewsCollector struct {
	apiKey        string
	categories    []string
	currentIndex  int
	lastHeadlines map[string]bool // Track recently used headlines to avoid duplicates
	client        *http.Client
}

// NewNewsCollector creates a new news data collector using the NewsAPI
func NewNewsCollector(apiKey string) *NewsCollector {
	return &NewsCollector{
		apiKey:        apiKey,
		categories:    []string{"business", "technology", "science", "health", "entertainment"},
		currentIndex:  0,
		lastHeadlines: make(map[string]bool),
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

// estimateSentiment performs a very simple sentiment analysis
// Returns a value between -1.0 (negative) and 1.0 (positive)
func estimateSentiment(headline string) float64 {
	headline = strings.ToLower(headline)

	positiveWords := []string{
		"win", "success", "grow", "gain", "rise", "increase", "improve", "boost",
		"breakthrough", "celebrate", "positive", "benefit", "advantage", "innovation",
		"progress", "achievement", "discovery", "advance", "revolutionize", "solution",
	}

	negativeWords := []string{
		"loss", "fail", "crash", "decline", "drop", "decrease", "collapse", "crisis",
		"cut", "danger", "threat", "risk", "fear", "concern", "warning", "disaster",
		"struggle", "problem", "conflict", "controversy", "attack", "damage", "died",
	}

	positiveCount := 0
	for _, word := range positiveWords {
		if strings.Contains(headline, word) {
			positiveCount++
		}
	}

	negativeCount := 0
	for _, word := range negativeWords {
		if strings.Contains(headline, word) {
			negativeCount++
		}
	}

	// Calculate sentiment score
	totalWords := positiveCount + negativeCount
	if totalWords == 0 {
		return 0.0 // Neutral
	}

	return (float64(positiveCount) - float64(negativeCount)) / float64(totalWords)
}

// extractKeywords extracts relevant keywords from a headline
func extractKeywords(headline string, category string) []string {
	headline = strings.ToLower(headline)
	words := strings.Fields(headline)

	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "with": true, "about": true, "from": true,
		"as": true, "by": true, "into": true, "like": true, "through": true,
	}

	var keywords []string
	for _, word := range words {
		word = strings.Trim(word, ".,;:!?\"'()[]{}") // Remove punctuation
		if len(word) > 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	// Add the category as a keyword
	keywords = append(keywords, category)

	return keywords
}

// Collect retrieves real news data from NewsAPI
func (c *NewsCollector) Collect(ctx context.Context) (*DataEvent, error) {
	// Cycle through categories
	category := c.categories[c.currentIndex]
	c.currentIndex = (c.currentIndex + 1) % len(c.categories)

	// Construct API URL
	url := fmt.Sprintf(
		"https://newsapi.org/v2/top-headlines?category=%s&language=en&pageSize=10&apiKey=%s",
		category, c.apiKey,
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	// Parse the response
	var newsAPIResponse NewsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsAPIResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Check if we got any articles
	if len(newsAPIResponse.Articles) == 0 {
		return nil, fmt.Errorf("no news articles returned")
	}

	// Pick a random article that hasn't been used recently
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		index := rand.Intn(len(newsAPIResponse.Articles))
		article := newsAPIResponse.Articles[index]

		// Skip if headline is empty or if we've used it recently
		if article.Title == "" || c.lastHeadlines[article.Title] {
			continue
		}

		// Mark this headline as used
		c.lastHeadlines[article.Title] = true

		// Limit the size of the lastHeadlines map to avoid memory growth
		if len(c.lastHeadlines) > 100 {
			// Clear the oldest half of the entries
			i := 0
			for headline := range c.lastHeadlines {
				if i > 50 {
					break
				}
				delete(c.lastHeadlines, headline)
				i++
			}
		}

		// Calculate sentiment
		sentiment := estimateSentiment(article.Title)

		// Extract keywords
		keywords := extractKeywords(article.Title, category)

		newsItem := &NewsItem{
			Headline:    article.Title,
			Source:      article.Source.Name,
			URL:         article.URL,
			Category:    category,
			Keywords:    keywords,
			PublishedAt: article.PublishedAt,
			Sentiment:   sentiment,
		}

		return &DataEvent{
			Type:      NewsData,
			Value:     newsItem,
			Timestamp: time.Now(),
			Source:    "newsapi-org",
			Raw:       article,
		}, nil
	}

	return nil, fmt.Errorf("could not find a new headline after %d attempts", maxAttempts)
}

// GetType returns the type of data collected
func (c *NewsCollector) GetType() DataType {
	return NewsData
}

// Start begins periodic collection of news data
func (c *NewsCollector) Start(ctx context.Context, interval time.Duration, eventCh chan<- *DataEvent) {
	log.Printf("Starting NewsAPI collector with interval %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	event, err := c.Collect(ctx)
	if err == nil {
		eventCh <- event
	} else {
		log.Printf("Error collecting news data: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			event, err := c.Collect(ctx)
			if err == nil {
				eventCh <- event
			} else {
				log.Printf("Error collecting news data: %v", err)
			}
		case <-ctx.Done():
			log.Println("News collector stopped")
			return
		}
	}
}
