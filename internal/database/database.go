package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thinkscotty/maggpi_go/internal/models"
	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate runs database migrations
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic_id INTEGER NOT NULL,
		url TEXT NOT NULL,
		name TEXT NOT NULL,
		is_manual BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS stories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic_id INTEGER NOT NULL,
		source_id INTEGER,
		title TEXT NOT NULL,
		summary TEXT NOT NULL,
		source_url TEXT NOT NULL,
		source_title TEXT,
		image_url TEXT,
		published_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
		FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		refresh_interval_minutes INTEGER DEFAULT 120,
		stories_per_topic INTEGER DEFAULT 5,
		global_sourcing_prompt TEXT,
		global_summarizing_prompt TEXT,
		primary_color TEXT DEFAULT '#2563eb',
		secondary_color TEXT DEFAULT '#1e40af',
		dark_mode BOOLEAN DEFAULT FALSE,
		gemini_api_key TEXT
	);

	CREATE TABLE IF NOT EXISTS refresh_status (
		topic_id INTEGER PRIMARY KEY,
		last_refresh DATETIME,
		next_refresh DATETIME,
		status TEXT DEFAULT 'pending',
		error_message TEXT,
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_stories_topic_id ON stories(topic_id);
	CREATE INDEX IF NOT EXISTS idx_sources_topic_id ON sources(topic_id);
	CREATE INDEX IF NOT EXISTS idx_stories_created_at ON stories(created_at DESC);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Topic operations

// GetTopics returns all topics ordered by position
func (db *DB) GetTopics() ([]models.Topic, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, description, position, created_at, updated_at
		FROM topics ORDER BY position ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []models.Topic
	for rows.Next() {
		var t models.Topic
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// GetTopic returns a single topic by ID
func (db *DB) GetTopic(id int64) (*models.Topic, error) {
	var t models.Topic
	err := db.conn.QueryRow(`
		SELECT id, name, description, position, created_at, updated_at
		FROM topics WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Description, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateTopic creates a new topic
func (db *DB) CreateTopic(name, description string) (*models.Topic, error) {
	// Get max position
	var maxPos sql.NullInt64
	db.conn.QueryRow("SELECT MAX(position) FROM topics").Scan(&maxPos)
	position := 0
	if maxPos.Valid {
		position = int(maxPos.Int64) + 1
	}

	result, err := db.conn.Exec(`
		INSERT INTO topics (name, description, position) VALUES (?, ?, ?)
	`, name, description, position)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetTopic(id)
}

// UpdateTopic updates an existing topic
func (db *DB) UpdateTopic(id int64, name, description string) error {
	_, err := db.conn.Exec(`
		UPDATE topics SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, description, id)
	return err
}

// DeleteTopic deletes a topic and all its related data
func (db *DB) DeleteTopic(id int64) error {
	_, err := db.conn.Exec("DELETE FROM topics WHERE id = ?", id)
	return err
}

// ReorderTopics updates the position of topics
func (db *DB) ReorderTopics(topicIDs []int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range topicIDs {
		if _, err := tx.Exec("UPDATE topics SET position = ? WHERE id = ?", i, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Source operations

// GetSourcesForTopic returns all sources for a topic
func (db *DB) GetSourcesForTopic(topicID int64) ([]models.Source, error) {
	rows, err := db.conn.Query(`
		SELECT id, topic_id, url, name, is_manual, created_at
		FROM sources WHERE topic_id = ?
	`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var s models.Source
		if err := rows.Scan(&s.ID, &s.TopicID, &s.URL, &s.Name, &s.IsManual, &s.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

// AddSource adds a new source to a topic
func (db *DB) AddSource(topicID int64, url, name string, isManual bool) (*models.Source, error) {
	result, err := db.conn.Exec(`
		INSERT INTO sources (topic_id, url, name, is_manual) VALUES (?, ?, ?, ?)
	`, topicID, url, name, isManual)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	var s models.Source
	err = db.conn.QueryRow(`
		SELECT id, topic_id, url, name, is_manual, created_at FROM sources WHERE id = ?
	`, id).Scan(&s.ID, &s.TopicID, &s.URL, &s.Name, &s.IsManual, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSource removes a source
func (db *DB) DeleteSource(id int64) error {
	_, err := db.conn.Exec("DELETE FROM sources WHERE id = ?", id)
	return err
}

// ClearAISources removes all AI-generated sources for a topic
func (db *DB) ClearAISources(topicID int64) error {
	_, err := db.conn.Exec("DELETE FROM sources WHERE topic_id = ? AND is_manual = FALSE", topicID)
	return err
}

// Story operations

// GetStoriesForTopic returns recent stories for a topic
func (db *DB) GetStoriesForTopic(topicID int64, limit int) ([]models.Story, error) {
	rows, err := db.conn.Query(`
		SELECT id, topic_id, source_id, title, summary, source_url, source_title, image_url, published_at, created_at
		FROM stories WHERE topic_id = ?
		ORDER BY created_at DESC LIMIT ?
	`, topicID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []models.Story
	for rows.Next() {
		var s models.Story
		var sourceID sql.NullInt64
		var sourceTitle, imageURL sql.NullString
		var publishedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.TopicID, &sourceID, &s.Title, &s.Summary, &s.SourceURL, &sourceTitle, &imageURL, &publishedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		if sourceID.Valid {
			id := sourceID.Int64
			s.SourceID = &id
		}
		if sourceTitle.Valid {
			s.SourceTitle = sourceTitle.String
		}
		if imageURL.Valid {
			s.ImageURL = imageURL.String
		}
		if publishedAt.Valid {
			s.PublishedAt = publishedAt.Time
		}
		stories = append(stories, s)
	}
	return stories, rows.Err()
}

// CreateStory creates a new story
func (db *DB) CreateStory(story *models.Story) error {
	result, err := db.conn.Exec(`
		INSERT INTO stories (topic_id, source_id, title, summary, source_url, source_title, image_url, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, story.TopicID, story.SourceID, story.Title, story.Summary, story.SourceURL, story.SourceTitle, story.ImageURL, story.PublishedAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	story.ID = id
	story.CreatedAt = time.Now()
	return nil
}

// DeleteOldStories removes stories older than the given duration for a topic
func (db *DB) DeleteOldStories(topicID int64, keepCount int) error {
	_, err := db.conn.Exec(`
		DELETE FROM stories WHERE topic_id = ? AND id NOT IN (
			SELECT id FROM stories WHERE topic_id = ? ORDER BY created_at DESC LIMIT ?
		)
	`, topicID, topicID, keepCount)
	return err
}

// Settings operations

// GetSettings returns the application settings
func (db *DB) GetSettings() (*models.Settings, error) {
	var s models.Settings
	var sourcingPrompt, summarizingPrompt, apiKey sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, refresh_interval_minutes, stories_per_topic, global_sourcing_prompt,
		       global_summarizing_prompt, primary_color, secondary_color, dark_mode, gemini_api_key
		FROM settings WHERE id = 1
	`).Scan(&s.ID, &s.RefreshIntervalMinutes, &s.StoriesPerTopic, &sourcingPrompt,
		&summarizingPrompt, &s.PrimaryColor, &s.SecondaryColor, &s.DarkMode, &apiKey)

	if err == sql.ErrNoRows {
		// Insert default settings
		defaults := models.DefaultSettings()
		_, err = db.conn.Exec(`
			INSERT INTO settings (id, refresh_interval_minutes, stories_per_topic, global_sourcing_prompt,
			                      global_summarizing_prompt, primary_color, secondary_color, dark_mode)
			VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		`, defaults.RefreshIntervalMinutes, defaults.StoriesPerTopic, defaults.GlobalSourcingPrompt,
			defaults.GlobalSummarizingPrompt, defaults.PrimaryColor, defaults.SecondaryColor, defaults.DarkMode)
		if err != nil {
			return nil, err
		}
		return &defaults, nil
	}
	if err != nil {
		return nil, err
	}

	if sourcingPrompt.Valid {
		s.GlobalSourcingPrompt = sourcingPrompt.String
	}
	if summarizingPrompt.Valid {
		s.GlobalSummarizingPrompt = summarizingPrompt.String
	}
	if apiKey.Valid {
		s.GeminiAPIKey = apiKey.String
	}

	return &s, nil
}

// UpdateSettings updates the application settings
func (db *DB) UpdateSettings(s *models.Settings) error {
	_, err := db.conn.Exec(`
		UPDATE settings SET
			refresh_interval_minutes = ?,
			stories_per_topic = ?,
			global_sourcing_prompt = ?,
			global_summarizing_prompt = ?,
			primary_color = ?,
			secondary_color = ?,
			dark_mode = ?,
			gemini_api_key = ?
		WHERE id = 1
	`, s.RefreshIntervalMinutes, s.StoriesPerTopic, s.GlobalSourcingPrompt,
		s.GlobalSummarizingPrompt, s.PrimaryColor, s.SecondaryColor, s.DarkMode, s.GeminiAPIKey)
	return err
}

// Refresh status operations

// GetRefreshStatus returns refresh status for a topic
func (db *DB) GetRefreshStatus(topicID int64) (*models.RefreshStatus, error) {
	var rs models.RefreshStatus
	var lastRefresh, nextRefresh sql.NullTime
	var errorMsg sql.NullString

	err := db.conn.QueryRow(`
		SELECT topic_id, last_refresh, next_refresh, status, error_message
		FROM refresh_status WHERE topic_id = ?
	`, topicID).Scan(&rs.TopicID, &lastRefresh, &nextRefresh, &rs.Status, &errorMsg)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastRefresh.Valid {
		rs.LastRefresh = lastRefresh.Time
	}
	if nextRefresh.Valid {
		rs.NextRefresh = nextRefresh.Time
	}
	if errorMsg.Valid {
		rs.ErrorMessage = errorMsg.String
	}

	return &rs, nil
}

// UpdateRefreshStatus updates or inserts refresh status for a topic
func (db *DB) UpdateRefreshStatus(rs *models.RefreshStatus) error {
	_, err := db.conn.Exec(`
		INSERT INTO refresh_status (topic_id, last_refresh, next_refresh, status, error_message)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(topic_id) DO UPDATE SET
			last_refresh = excluded.last_refresh,
			next_refresh = excluded.next_refresh,
			status = excluded.status,
			error_message = excluded.error_message
	`, rs.TopicID, rs.LastRefresh, rs.NextRefresh, rs.Status, rs.ErrorMessage)
	return err
}

// GetAllRefreshStatuses returns all refresh statuses
func (db *DB) GetAllRefreshStatuses() ([]models.RefreshStatus, error) {
	rows, err := db.conn.Query(`
		SELECT topic_id, last_refresh, next_refresh, status, error_message
		FROM refresh_status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.RefreshStatus
	for rows.Next() {
		var rs models.RefreshStatus
		var lastRefresh, nextRefresh sql.NullTime
		var errorMsg sql.NullString

		if err := rows.Scan(&rs.TopicID, &lastRefresh, &nextRefresh, &rs.Status, &errorMsg); err != nil {
			return nil, err
		}

		if lastRefresh.Valid {
			rs.LastRefresh = lastRefresh.Time
		}
		if nextRefresh.Valid {
			rs.NextRefresh = nextRefresh.Time
		}
		if errorMsg.Valid {
			rs.ErrorMessage = errorMsg.String
		}

		statuses = append(statuses, rs)
	}
	return statuses, rows.Err()
}

// GetTopicsWithStories returns all topics with their recent stories
func (db *DB) GetTopicsWithStories(storiesPerTopic int) ([]models.TopicWithStories, error) {
	topics, err := db.GetTopics()
	if err != nil {
		return nil, err
	}

	var result []models.TopicWithStories
	for _, topic := range topics {
		stories, err := db.GetStoriesForTopic(topic.ID, storiesPerTopic)
		if err != nil {
			return nil, err
		}
		result = append(result, models.TopicWithStories{
			Topic:   topic,
			Stories: stories,
		})
	}
	return result, nil
}

// GetTopicsWithSources returns all topics with their sources
func (db *DB) GetTopicsWithSources() ([]models.TopicWithSources, error) {
	topics, err := db.GetTopics()
	if err != nil {
		return nil, err
	}

	var result []models.TopicWithSources
	for _, topic := range topics {
		sources, err := db.GetSourcesForTopic(topic.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.TopicWithSources{
			Topic:   topic,
			Sources: sources,
		})
	}
	return result, nil
}
