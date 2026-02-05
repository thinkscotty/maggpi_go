package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/thinkscotty/maggpi_go/internal/api"
	"github.com/thinkscotty/maggpi_go/internal/config"
	"github.com/thinkscotty/maggpi_go/internal/database"
	"github.com/thinkscotty/maggpi_go/internal/handlers"
	"github.com/thinkscotty/maggpi_go/internal/scheduler"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "./data/config.json", "Path to configuration file")
	flag.Parse()

	log.Println("Starting MaggPi...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed default topics if database is empty
	if err := seedDefaultTopics(db); err != nil {
		log.Printf("Warning: failed to seed default topics: %v", err)
	}

	// Create scheduler
	sched := scheduler.New(db)

	// Get executable directory for templates/static
	execDir, err := os.Executable()
	if err != nil {
		execDir = "."
	} else {
		execDir = filepath.Dir(execDir)
	}

	// Try multiple template locations
	templatesDir := findDir([]string{
		filepath.Join(execDir, "web", "templates"),
		"./web/templates",
		"/opt/maggpi/web/templates",
	})
	staticDir := findDir([]string{
		filepath.Join(execDir, "web", "static"),
		"./web/static",
		"/opt/maggpi/web/static",
	})

	if templatesDir == "" {
		log.Fatal("Could not find templates directory")
	}
	if staticDir == "" {
		log.Fatal("Could not find static directory")
	}

	log.Printf("Using templates from: %s", templatesDir)
	log.Printf("Using static files from: %s", staticDir)

	// Create handlers
	h, err := handlers.New(db, sched, templatesDir)
	if err != nil {
		log.Fatalf("Failed to create handlers: %v", err)
	}

	// Create router
	router := api.NewRouter(h, staticDir)

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start scheduler
	sched.Start()

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Stop scheduler
	sched.Stop()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// findDir returns the first directory that exists
func findDir(paths []string) string {
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

// seedDefaultTopics adds the default topics if the database is empty
func seedDefaultTopics(db *database.DB) error {
	topics, err := db.GetTopics()
	if err != nil {
		return err
	}

	// Only seed if no topics exist
	if len(topics) > 0 {
		return nil
	}

	defaultTopics := []struct {
		Name        string
		Description string
	}{
		{
			Name:        "World News",
			Description: "Major international news and current events from around the globe. Focus on significant political developments, international relations, and major world events.",
		},
		{
			Name:        "Formula 1",
			Description: "Formula 1 racing news including race results, driver standings, team updates, technical regulations, and breaking news from the F1 paddock.",
		},
		{
			Name:        "Science News",
			Description: "Latest scientific discoveries and research breakthroughs across all fields including physics, biology, astronomy, climate science, and medical research.",
		},
		{
			Name:        "Tech News",
			Description: "Technology industry news including product launches, company updates, software releases, AI developments, and emerging tech trends.",
		},
	}

	for _, t := range defaultTopics {
		if _, err := db.CreateTopic(t.Name, t.Description); err != nil {
			return fmt.Errorf("failed to create topic %s: %w", t.Name, err)
		}
		log.Printf("Created default topic: %s", t.Name)
	}

	return nil
}
