# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MaggPi is a lightweight, local web application written in Go that allows users to input topics and displays AI-summarized news stories scraped from around the internet. Designed to run on minimal hardware (Raspberry Pi 3B+).

### Features

- User-manageable topics with AI-discovered sources
- AI-summarization of scraped stories via Gemini API
- Global AI instructions for source discovery and summarization
- Clean web UI with Dashboard, Topic Management, and Settings pages
- JSON API for external client devices (microcontrollers, displays)
- Automatic scheduled refreshes with staggered timing

## Development Commands

```bash
# Build for current platform
make build

# Build for Raspberry Pi (Linux ARM64)
make build-pi

# Run locally
make run

# Run with hot reload (requires air)
make dev

# Download dependencies
make deps

# Run tests
make test

# Clean build artifacts
make clean
```

## Architecture

### Directory Structure

```
maggpi_go/
├── cmd/maggpi/main.go       # Application entry point
├── internal/
│   ├── api/router.go        # Chi router configuration
│   ├── config/config.go     # JSON configuration loading
│   ├── database/database.go # SQLite database operations
│   ├── gemini/gemini.go     # Gemini AI API client
│   ├── handlers/handlers.go # HTTP request handlers
│   ├── models/models.go     # Data structures
│   ├── scheduler/scheduler.go # Background refresh scheduler
│   └── scraper/scraper.go   # Web scraping with Colly
├── web/
│   ├── templates/           # Go HTML templates
│   │   ├── base.html        # Base layout
│   │   ├── dashboard.html   # Main news display
│   │   ├── topics.html      # Topic management
│   │   └── settings.html    # App configuration
│   └── static/
│       ├── css/style.css    # Stylesheet
│       └── js/app.js        # Client-side JavaScript
├── data/                    # Runtime data (config.json, maggpi.db)
├── bin/                     # Build output
├── Makefile
└── README.md
```

### Tech Stack

| Component | Library | Version |
|-----------|---------|---------|
| Web Framework | github.com/go-chi/chi/v5 | v5.2.4 |
| Database | modernc.org/sqlite | v1.44.3 |
| AI API | google.golang.org/genai | v1.45.0 |
| Web Scraping | github.com/gocolly/colly/v2 | v2.3.0 |
| Templates | html/template | stdlib |

### Key Flows

1. **Topic Creation**: User creates topic → Gemini discovers sources → Sources saved to DB
2. **Story Refresh**: Scheduler triggers → Scraper fetches sources → Gemini summarizes → Stories saved
3. **Dashboard Display**: Handler fetches topics + stories → Template renders cards

### Database Schema

- `topics`: id, name, description, position, created_at, updated_at
- `sources`: id, topic_id, url, name, is_manual, created_at
- `stories`: id, topic_id, source_id, title, summary, source_url, source_title, image_url, published_at, created_at
- `settings`: Single row with all app settings including Gemini API key
- `refresh_status`: topic_id, last_refresh, next_refresh, status, error_message

### API Endpoints

**Internal (Web UI)**:
- `GET /` - Dashboard
- `GET /topics` - Topic management page
- `GET /settings` - Settings page
- `GET/POST/PUT/DELETE /api/topics/*` - Topic CRUD
- `GET/PUT /api/settings` - Settings management

**External (Client devices)**:
- `GET /v1/stories` - All topics with stories
- `GET /v1/topics` - List topics
- `GET /v1/topics/{id}/stories` - Stories for specific topic

## Important Notes

- **HARDWARE**: Raspberry Pi 3B+
- **OS**: Pi OS 64-bit (Debian 13 Trixie)
- **SERVER IP**: 192.168.0.101
- **PORT**: 7979
- **USERNAME**: scotty
- **GITHUB**: https://github.com/thinkscotty/maggpi_go.git
- **UI FONT**: Funnel Sans (Google Fonts)

### Performance Considerations

- SQLite with WAL mode for better concurrent access
- Limited parallel scraping (2 concurrent) to conserve memory
- Staggered topic refreshes to avoid API rate limits
- 30-second delay between topic refreshes
- Failed refreshes retry after 5 minutes

### Security Notes

- Gemini API key stored in SQLite (masked in UI responses)
- No authentication on web interface (designed for local network use)
- Input validation on all API endpoints
