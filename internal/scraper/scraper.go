package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/thinkscotty/maggpi_go/internal/gemini"
	"github.com/thinkscotty/maggpi_go/internal/models"
)

// Scraper handles web scraping operations
type Scraper struct {
	userAgent      string
	requestTimeout time.Duration
	parallelLimit  int
}

// New creates a new Scraper
func New() *Scraper {
	return &Scraper{
		userAgent:      "MaggPi/1.0 (Raspberry Pi News Aggregator; +https://github.com/thinkscotty/maggpi_go)",
		requestTimeout: 30 * time.Second,
		parallelLimit:  2, // Keep low for Raspberry Pi
	}
}

// ScrapeSource scrapes content from a single source
func (s *Scraper) ScrapeSource(ctx context.Context, source models.Source) (*gemini.ScrapedContent, error) {
	c := colly.NewCollector(
		colly.UserAgent(s.userAgent),
		colly.MaxDepth(1),
	)

	c.SetRequestTimeout(s.requestTimeout)

	var content strings.Builder
	var title string
	var mu sync.Mutex

	// Extract page title
	c.OnHTML("title", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		if title == "" {
			title = strings.TrimSpace(e.Text)
		}
	})

	// Extract main content - try common content selectors
	contentSelectors := []string{
		"article",
		"main",
		".content",
		".post",
		".article",
		".entry-content",
		"#content",
		"#main",
	}

	for _, selector := range contentSelectors {
		c.OnHTML(selector, func(e *colly.HTMLElement) {
			mu.Lock()
			defer mu.Unlock()
			text := cleanText(e.Text)
			if len(text) > 100 { // Only include substantial content
				content.WriteString(text)
				content.WriteString("\n\n")
			}
		})
	}

	// Extract headlines and links
	c.OnHTML("h1, h2, h3", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		text := cleanText(e.Text)
		if len(text) > 10 && len(text) < 200 {
			content.WriteString("HEADLINE: ")
			content.WriteString(text)
			content.WriteString("\n")
		}
	})

	// Extract paragraph text if we haven't found main content
	c.OnHTML("p", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		text := cleanText(e.Text)
		if len(text) > 50 && len(text) < 2000 {
			content.WriteString(text)
			content.WriteString("\n")
		}
	})

	// Handle RSS/Atom feeds
	c.OnHTML("item, entry", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		itemTitle := e.ChildText("title")
		itemDesc := e.ChildText("description, summary, content")
		itemLink := e.ChildAttr("link", "href")
		if itemLink == "" {
			itemLink = e.ChildText("link")
		}

		if itemTitle != "" {
			content.WriteString("ARTICLE: ")
			content.WriteString(itemTitle)
			content.WriteString("\n")
			if itemLink != "" {
				content.WriteString("LINK: ")
				content.WriteString(itemLink)
				content.WriteString("\n")
			}
			if itemDesc != "" {
				content.WriteString(cleanText(itemDesc))
				content.WriteString("\n\n")
			}
		}
	})

	// Error handling
	var scrapeErr error
	c.OnError(func(r *colly.Response, err error) {
		scrapeErr = fmt.Errorf("scrape error for %s: %w (status: %d)", source.URL, err, r.StatusCode)
	})

	// Visit the URL
	if err := c.Visit(source.URL); err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", source.URL, err)
	}

	c.Wait()

	if scrapeErr != nil {
		return nil, scrapeErr
	}

	contentStr := content.String()
	if len(contentStr) < 100 {
		return nil, fmt.Errorf("insufficient content scraped from %s", source.URL)
	}

	// Truncate if too long (to manage API costs and memory)
	maxLength := 10000
	if len(contentStr) > maxLength {
		contentStr = contentStr[:maxLength] + "..."
	}

	sourceName := source.Name
	if sourceName == "" {
		sourceName = title
	}
	if sourceName == "" {
		parsedURL, _ := url.Parse(source.URL)
		if parsedURL != nil {
			sourceName = parsedURL.Host
		}
	}

	return &gemini.ScrapedContent{
		URL:        source.URL,
		SourceName: sourceName,
		Content:    contentStr,
	}, nil
}

// ScrapeSources scrapes multiple sources concurrently
func (s *Scraper) ScrapeSources(ctx context.Context, sources []models.Source) []gemini.ScrapedContent {
	var results []gemini.ScrapedContent
	var mu sync.Mutex

	// Use a semaphore to limit concurrent scrapes
	sem := make(chan struct{}, s.parallelLimit)
	var wg sync.WaitGroup

	for _, source := range sources {
		select {
		case <-ctx.Done():
			return results
		default:
		}

		wg.Add(1)
		go func(src models.Source) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			content, err := s.ScrapeSource(ctx, src)
			if err != nil {
				// Log error but continue with other sources
				fmt.Printf("Warning: failed to scrape %s: %v\n", src.URL, err)
				return
			}

			mu.Lock()
			results = append(results, *content)
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return results
}

// cleanText removes extra whitespace and normalizes text
func cleanText(s string) string {
	// Replace multiple whitespace with single space
	s = strings.Join(strings.Fields(s), " ")
	// Remove common navigation/UI text
	s = strings.TrimSpace(s)
	return s
}

// ValidateURL checks if a URL is valid and accessible
func ValidateURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}
