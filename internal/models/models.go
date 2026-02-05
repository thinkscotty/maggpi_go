package models

import "time"

// Topic represents a user-defined topic for news aggregation
type Topic struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Source represents a web source for a topic
type Source struct {
	ID        int64     `json:"id"`
	TopicID   int64     `json:"topic_id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	IsManual  bool      `json:"is_manual"` // true if manually added by user
	CreatedAt time.Time `json:"created_at"`
}

// Story represents a summarized news story
type Story struct {
	ID          int64     `json:"id"`
	TopicID     int64     `json:"topic_id"`
	SourceID    *int64    `json:"source_id,omitempty"` // Nullable - may not map to a specific source
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	SourceURL   string    `json:"source_url"`
	SourceTitle string    `json:"source_title"`
	ImageURL    string    `json:"image_url,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Settings represents global application settings
type Settings struct {
	ID                       int64  `json:"id"`
	RefreshIntervalMinutes   int    `json:"refresh_interval_minutes"`
	StoriesPerTopic          int    `json:"stories_per_topic"`
	GlobalSourcingPrompt     string `json:"global_sourcing_prompt"`
	GlobalSummarizingPrompt  string `json:"global_summarizing_prompt"`
	PrimaryColor             string `json:"primary_color"`
	SecondaryColor           string `json:"secondary_color"`
	DarkMode                 bool   `json:"dark_mode"`
	GeminiAPIKey             string `json:"gemini_api_key"`
	DashboardTitle           string `json:"dashboard_title"`
	DashboardSubtitle        string `json:"dashboard_subtitle"`
}

// DefaultSettings returns the default application settings
func DefaultSettings() Settings {
	return Settings{
		RefreshIntervalMinutes:  120,
		StoriesPerTopic:         5,
		GlobalSourcingPrompt:    "Find reliable, reputable news sources that provide regular updates. Prefer sources with RSS feeds or well-structured HTML. Avoid paywalled content when possible.",
		GlobalSummarizingPrompt: "Summarize the news story in a clear, informative tone. Focus on the key facts and why this story matters. Keep the summary between 75-150 words.",
		PrimaryColor:            "#2563eb",
		SecondaryColor:          "#1e40af",
		DarkMode:                false,
		DashboardTitle:          "Dashboard",
		DashboardSubtitle:       "Your personalized news feed",
	}
}

// TopicWithStories combines a topic with its stories for display
type TopicWithStories struct {
	Topic   Topic   `json:"topic"`
	Stories []Story `json:"stories"`
}

// TopicWithSources combines a topic with its sources for management
type TopicWithSources struct {
	Topic   Topic    `json:"topic"`
	Sources []Source `json:"sources"`
}

// RefreshStatus tracks the status of topic refreshes
type RefreshStatus struct {
	TopicID      int64     `json:"topic_id"`
	LastRefresh  time.Time `json:"last_refresh"`
	NextRefresh  time.Time `json:"next_refresh"`
	Status       string    `json:"status"` // "pending", "in_progress", "completed", "failed"
	ErrorMessage string    `json:"error_message,omitempty"`
}

// APIResponse is the standard response format for the external API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
