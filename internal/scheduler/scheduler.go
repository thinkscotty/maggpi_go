package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thinkscotty/maggpi_go/internal/database"
	"github.com/thinkscotty/maggpi_go/internal/gemini"
	"github.com/thinkscotty/maggpi_go/internal/models"
	"github.com/thinkscotty/maggpi_go/internal/scraper"
)

// Scheduler manages periodic topic refreshes
type Scheduler struct {
	db       *database.DB
	scraper  *scraper.Scraper
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// New creates a new Scheduler
func New(db *database.DB) *Scheduler {
	return &Scheduler{
		db:       db,
		scraper:  scraper.New(),
		interval: 120 * time.Minute, // Default, will be overwritten from settings
		stopCh:   make(chan struct{}),
	}
}

// Start begins the scheduled refresh process
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()
	log.Println("Scheduler started")
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()
	log.Println("Scheduler stopped")
}

// UpdateInterval updates the refresh interval
func (s *Scheduler) UpdateInterval(minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = time.Duration(minutes) * time.Minute
	log.Printf("Scheduler interval updated to %d minutes", minutes)
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	// Initial delay to let the server start
	time.Sleep(10 * time.Second)

	// Check for topics that need initial sources
	s.initializeTopics()

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		// Get settings for interval
		settings, err := s.db.GetSettings()
		if err == nil && settings != nil {
			s.mu.Lock()
			s.interval = time.Duration(settings.RefreshIntervalMinutes) * time.Minute
			s.mu.Unlock()
		}

		// Find topics that need refresh
		topics, err := s.db.GetTopics()
		if err != nil {
			log.Printf("Error getting topics: %v", err)
			time.Sleep(time.Minute)
			continue
		}

		// Stagger refreshes to avoid API overload
		topicsToRefresh := s.getTopicsNeedingRefresh(topics)
		for _, topic := range topicsToRefresh {
			select {
			case <-s.stopCh:
				return
			default:
			}

			s.refreshTopic(topic.ID)

			// Wait between topic refreshes to be gentle on the Pi
			time.Sleep(30 * time.Second)
		}

		// Sleep until next check
		select {
		case <-s.stopCh:
			return
		case <-time.After(time.Minute):
		}
	}
}

// initializeTopics discovers sources for topics that have none
func (s *Scheduler) initializeTopics() {
	topics, err := s.db.GetTopics()
	if err != nil {
		log.Printf("Error getting topics for initialization: %v", err)
		return
	}

	settings, err := s.db.GetSettings()
	if err != nil || settings == nil || settings.GeminiAPIKey == "" {
		log.Println("Gemini API key not configured, skipping topic initialization")
		return
	}

	for _, topic := range topics {
		sources, err := s.db.GetSourcesForTopic(topic.ID)
		if err != nil {
			log.Printf("Error getting sources for topic %d: %v", topic.ID, err)
			continue
		}

		if len(sources) == 0 {
			log.Printf("Discovering sources for topic: %s", topic.Name)
			s.discoverSources(topic.ID)
			time.Sleep(5 * time.Second) // Rate limit
		}
	}
}

// getTopicsNeedingRefresh returns topics whose refresh time has passed
func (s *Scheduler) getTopicsNeedingRefresh(topics []models.Topic) []models.Topic {
	var needRefresh []models.Topic
	now := time.Now()

	for _, topic := range topics {
		status, err := s.db.GetRefreshStatus(topic.ID)
		if err != nil {
			log.Printf("Error getting refresh status for topic %d: %v", topic.ID, err)
			continue
		}

		// If no status exists or refresh time has passed, need refresh
		if status == nil {
			needRefresh = append(needRefresh, topic)
		} else if now.After(status.NextRefresh) && status.Status != "in_progress" {
			needRefresh = append(needRefresh, topic)
		}
	}

	return needRefresh
}

// RefreshTopic manually triggers a topic refresh
func (s *Scheduler) RefreshTopic(topicID int64) error {
	return s.refreshTopic(topicID)
}

// refreshTopic performs the actual refresh for a topic
func (s *Scheduler) refreshTopic(topicID int64) error {
	topic, err := s.db.GetTopic(topicID)
	if err != nil || topic == nil {
		return fmt.Errorf("topic not found: %d", topicID)
	}

	settings, err := s.db.GetSettings()
	if err != nil || settings == nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if settings.GeminiAPIKey == "" {
		return fmt.Errorf("Gemini API key not configured")
	}

	// Update status to in_progress
	status := &models.RefreshStatus{
		TopicID: topicID,
		Status:  "in_progress",
	}
	s.db.UpdateRefreshStatus(status)

	log.Printf("Refreshing topic: %s", topic.Name)

	// Get sources for this topic
	sources, err := s.db.GetSourcesForTopic(topicID)
	if err != nil {
		return s.handleRefreshError(topicID, fmt.Errorf("failed to get sources: %w", err))
	}

	if len(sources) == 0 {
		// Try to discover sources first
		if err := s.discoverSources(topicID); err != nil {
			return s.handleRefreshError(topicID, fmt.Errorf("failed to discover sources: %w", err))
		}
		sources, _ = s.db.GetSourcesForTopic(topicID)
		if len(sources) == 0 {
			return s.handleRefreshError(topicID, fmt.Errorf("no sources available for topic"))
		}
	}

	// Scrape content from sources
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	scrapedContent := s.scraper.ScrapeSources(ctx, sources)
	if len(scrapedContent) == 0 {
		return s.handleRefreshError(topicID, fmt.Errorf("failed to scrape any content"))
	}

	// Summarize with Gemini
	geminiClient, err := gemini.New(settings.GeminiAPIKey)
	if err != nil {
		return s.handleRefreshError(topicID, fmt.Errorf("failed to create Gemini client: %w", err))
	}
	defer geminiClient.Close()

	stories, err := geminiClient.SummarizeContent(ctx, topic.Name, scrapedContent, settings.GlobalSummarizingPrompt, settings.StoriesPerTopic)
	if err != nil {
		return s.handleRefreshError(topicID, fmt.Errorf("failed to summarize content: %w", err))
	}

	// Store stories
	for _, story := range stories {
		dbStory := &models.Story{
			TopicID:     topicID,
			Title:       story.Title,
			Summary:     story.Summary,
			SourceURL:   story.SourceURL,
			SourceTitle: story.SourceTitle,
			PublishedAt: time.Now(),
		}
		if err := s.db.CreateStory(dbStory); err != nil {
			log.Printf("Error creating story: %v", err)
		}
	}

	// Clean up old stories (keep 3x the display count)
	s.db.DeleteOldStories(topicID, settings.StoriesPerTopic*3)

	// Update status to completed
	s.mu.Lock()
	interval := s.interval
	s.mu.Unlock()

	status = &models.RefreshStatus{
		TopicID:     topicID,
		LastRefresh: time.Now(),
		NextRefresh: time.Now().Add(interval),
		Status:      "completed",
	}
	s.db.UpdateRefreshStatus(status)

	log.Printf("Completed refresh for topic: %s (%d stories)", topic.Name, len(stories))
	return nil
}

// handleRefreshError updates status and schedules a retry
func (s *Scheduler) handleRefreshError(topicID int64, err error) error {
	log.Printf("Refresh error for topic %d: %v", topicID, err)

	status := &models.RefreshStatus{
		TopicID:      topicID,
		NextRefresh:  time.Now().Add(5 * time.Minute), // Retry in 5 minutes
		Status:       "failed",
		ErrorMessage: err.Error(),
	}
	s.db.UpdateRefreshStatus(status)

	return err
}

// DiscoverSources triggers source discovery for a topic
func (s *Scheduler) DiscoverSources(topicID int64) error {
	return s.discoverSources(topicID)
}

// discoverSources uses AI to find sources for a topic
func (s *Scheduler) discoverSources(topicID int64) error {
	topic, err := s.db.GetTopic(topicID)
	if err != nil || topic == nil {
		return fmt.Errorf("topic not found: %d", topicID)
	}

	settings, err := s.db.GetSettings()
	if err != nil || settings == nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if settings.GeminiAPIKey == "" {
		return fmt.Errorf("Gemini API key not configured")
	}

	geminiClient, err := gemini.New(settings.GeminiAPIKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer geminiClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sources, err := geminiClient.DiscoverSources(ctx, topic.Name, topic.Description, settings.GlobalSourcingPrompt)
	if err != nil {
		return fmt.Errorf("failed to discover sources: %w", err)
	}

	// Clear existing AI sources and add new ones
	s.db.ClearAISources(topicID)

	for _, source := range sources {
		if err := scraper.ValidateURL(source.URL); err != nil {
			log.Printf("Skipping invalid source URL %s: %v", source.URL, err)
			continue
		}

		_, err := s.db.AddSource(topicID, source.URL, source.Name, false)
		if err != nil {
			log.Printf("Error adding source: %v", err)
		}
	}

	log.Printf("Discovered %d sources for topic: %s", len(sources), topic.Name)
	return nil
}
