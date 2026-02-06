package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Client handles fetching posts from Reddit's JSON API
type Client struct {
	httpClient   *http.Client
	userAgent    string
	minWordCount int
	mu           sync.Mutex
	lastRequest  time.Time
	minInterval  time.Duration
}

// Post represents a filtered Reddit post
type Post struct {
	Title      string
	Body       string
	Permalink  string
	Subreddit  string
	Author     string
	Score      int
	CreatedUTC time.Time
}

// New creates a new Reddit client with rate limiting
func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent:    "MaggPi/1.0 (Raspberry Pi News Aggregator; +https://github.com/thinkscotty/maggpi_go)",
		minWordCount: 100,
		minInterval:  1100 * time.Millisecond, // ~54 req/min to stay under 60/min limit
	}
}

// FetchPosts fetches and filters posts from a subreddit
// Only returns text posts (self posts) with >100 words
func (c *Client) FetchPosts(ctx context.Context, subredditURL string, topicName string) ([]Post, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Extract subreddit name from URL
	subreddit, err := extractSubreddit(subredditURL)
	if err != nil {
		return nil, err
	}

	// Rate limit (context-aware)
	if err := c.waitForRateLimitWithContext(ctx); err != nil {
		return nil, err
	}

	// Build the JSON API URL
	apiURL := fmt.Sprintf("https://www.reddit.com/r/%s.json?limit=25", subreddit)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Reddit requires a User-Agent header
	req.Header.Set("User-Agent", c.userAgent)

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subreddit %s: %w", subreddit, err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, fmt.Errorf("subreddit r/%s not found", subreddit)
		case http.StatusForbidden:
			return nil, fmt.Errorf("subreddit r/%s is private or quarantined", subreddit)
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("Reddit rate limit exceeded")
		default:
			return nil, fmt.Errorf("Reddit API returned status %d", resp.StatusCode)
		}
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Reddit's JSON structure
	var listing redditListing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit JSON: %w", err)
	}

	// Filter and convert posts
	var posts []Post
	for _, child := range listing.Data.Children {
		post := child.Data

		// Only include self posts (text posts, not links/images)
		if !post.IsSelf {
			continue
		}

		// Check word count
		wordCount := countWords(post.Selftext)
		if wordCount < c.minWordCount {
			continue
		}

		// Add to results
		posts = append(posts, Post{
			Title:      post.Title,
			Body:       post.Selftext,
			Permalink:  post.Permalink,
			Subreddit:  post.Subreddit,
			Author:     post.Author,
			Score:      post.Score,
			CreatedUTC: time.Unix(int64(post.CreatedUTC), 0),
		})
	}

	return posts, nil
}

// waitForRateLimitWithContext ensures we don't exceed Reddit's rate limit while respecting context
func (c *Client) waitForRateLimitWithContext(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := time.Since(c.lastRequest)
	if elapsed < c.minInterval {
		waitTime := c.minInterval - elapsed
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Rate limit wait completed
		}
	}
	c.lastRequest = time.Now()
	return nil
}

// extractSubreddit extracts the subreddit name from various URL formats
func extractSubreddit(url string) (string, error) {
	// Handle various Reddit URL formats:
	// https://reddit.com/r/golang
	// https://www.reddit.com/r/golang
	// https://old.reddit.com/r/golang
	// https://reddit.com/r/golang/
	// r/golang

	// Pattern to match subreddit name
	patterns := []string{
		`reddit\.com/r/([a-zA-Z0-9_]+)`,
		`^r/([a-zA-Z0-9_]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract subreddit from URL: %s", url)
}

// countWords counts words in a string
func countWords(s string) int {
	return len(strings.Fields(s))
}

// IsRedditURL checks if a URL is a Reddit subreddit URL
func IsRedditURL(url string) bool {
	return strings.Contains(url, "reddit.com/r/") || strings.HasPrefix(url, "r/")
}

// Reddit JSON API structures

type redditListing struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	Title      string  `json:"title"`
	Selftext   string  `json:"selftext"`
	IsSelf     bool    `json:"is_self"`
	Permalink  string  `json:"permalink"`
	Subreddit  string  `json:"subreddit"`
	Author     string  `json:"author"`
	Score      int     `json:"score"`
	CreatedUTC float64 `json:"created_utc"`
}
