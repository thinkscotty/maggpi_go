package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// Client wraps the Gemini API client
type Client struct {
	client *genai.Client
	model  string
}

// DiscoveredSource represents a source discovered by AI
type DiscoveredSource struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SummarizedStory represents a summarized story from AI
type SummarizedStory struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	SourceURL   string `json:"source_url"`
	SourceTitle string `json:"source_title"`
}

// New creates a new Gemini client
func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		client: client,
		model:  "gemini-2.0-flash",
	}, nil
}

// Close is a no-op as the genai client doesn't require explicit cleanup
func (c *Client) Close() error {
	return nil
}

// DiscoverSources uses AI to find relevant sources for a topic
func (c *Client) DiscoverSources(ctx context.Context, topicName, topicDescription, globalInstructions string) ([]DiscoveredSource, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant that discovers reliable web sources for news topics.

Topic: %s
Description: %s

%s

Find 4-8 reliable web sources (websites, RSS feeds, or APIs) that provide ongoing news and updates related to this topic. For each source, provide:
1. The URL (must be a real, working URL)
2. A short name for the source
3. A brief description of what content it provides

IMPORTANT: Return ONLY a valid JSON array with no additional text, markdown, or explanation. The response must be parseable JSON.

Format your response as a JSON array like this:
[
  {"url": "https://example.com/feed", "name": "Example News", "description": "Daily updates on topic"},
  {"url": "https://another.com", "name": "Another Source", "description": "Breaking news coverage"}
]`, topicName, topicDescription, globalInstructions)

	result, err := c.client.Models.GenerateContent(ctx, c.model,
		[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}},
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract text from response
	responseText := extractText(result)
	if responseText == "" {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Clean up the response - remove markdown code blocks if present
	responseText = cleanJSONResponse(responseText)

	var sources []DiscoveredSource
	if err := json.Unmarshal([]byte(responseText), &sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources JSON: %w (response: %s)", err, responseText)
	}

	return sources, nil
}

// SummarizeContent summarizes scraped content into news stories
func (c *Client) SummarizeContent(ctx context.Context, topicName string, scrapedContent []ScrapedContent, globalInstructions string, maxStories int) ([]SummarizedStory, error) {
	if len(scrapedContent) == 0 {
		return nil, nil
	}

	// Build content string from scraped data
	var contentBuilder strings.Builder
	for i, content := range scrapedContent {
		contentBuilder.WriteString(fmt.Sprintf("\n--- Source %d: %s ---\nURL: %s\n%s\n",
			i+1, content.SourceName, content.URL, content.Content))
	}

	prompt := fmt.Sprintf(`You are a news summarization assistant. Your task is to analyze the following scraped content and create clear, informative news summaries.

Topic: %s

%s

Scraped Content:
%s

From the content above, identify the %d most interesting and relevant news stories. For each story:
1. Create a compelling headline (title)
2. Write a summary of 75-150 words focusing on key facts and why this story matters
3. Include the source URL where the story was found
4. Include the source name/title

IMPORTANT: Return ONLY a valid JSON array with no additional text, markdown, or explanation. The response must be parseable JSON.

Format your response as a JSON array like this:
[
  {"title": "Headline Here", "summary": "Summary text here...", "source_url": "https://source.com/article", "source_title": "Source Name"}
]`, topicName, globalInstructions, contentBuilder.String(), maxStories)

	result, err := c.client.Models.GenerateContent(ctx, c.model,
		[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}},
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	responseText := extractText(result)
	if responseText == "" {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	responseText = cleanJSONResponse(responseText)

	var stories []SummarizedStory
	if err := json.Unmarshal([]byte(responseText), &stories); err != nil {
		return nil, fmt.Errorf("failed to parse stories JSON: %w (response: %s)", err, responseText)
	}

	return stories, nil
}

// ScrapedContent represents content scraped from a source
type ScrapedContent struct {
	URL        string
	SourceName string
	Content    string
}

// extractText extracts text from a Gemini response
func extractText(result *genai.GenerateContentResponse) string {
	if result == nil || len(result.Candidates) == 0 {
		return ""
	}

	var text strings.Builder
	for _, candidate := range result.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				text.WriteString(part.Text)
			}
		}
	}
	return text.String()
}

// cleanJSONResponse removes markdown code blocks and extra whitespace from JSON responses
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}
