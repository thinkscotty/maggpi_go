# Maggpi - AI-Powered News Aggregator for Raspberry Pi

MaggPi is a very lightweight, self-hosted content aggregator/transformer and content server built with Go to run on minimal hardware. Rather than displaying bare links and text previews, it utilizes Gemini AI's free-tier API to both source content and transform it into bite-sized stories. The user can input ANY topic whatsoever. This makes the applicaiton suitable for 'at-a-glace' topic tracking via the web UI, and for serving to clients (such as smart-home dashboards or news tickers).

The **Maggpi** name is derived from two things. First, a portmanteau of "Mini AGGregator for PI". And second, the Magpie bird - a creature known for its love of collecting shiny objects. Maggpi collects shiny stories from around the internet. 

[![GitHub release](https://img.shields.io/github/v/release/thinkscotty/maggpi_go)](https://github.com/thinkscotty/maggpi_go/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Built to be Fast on Lightweight Hardware** - Built in the GO programming language, intentionally lightweight UI and featureset. Runs smoothly on Raspberry Pi 3 and later (requires just 1GB of RAM).
- **AI-Powered Source Discovery** - Input any topic whatsoever with a brief description and Gemini will add suitable sources, including relevant Reddit subreddits. You can, of course, also add your own sources.
- **Reddit Integration** - Automatically discovers and fetches content from relevant subreddits for niche topics. Filters for substantive text posts only.
- **Smart Summarization** - Each story intelligently summarized to 75-150 words (configurable)
- **Custom AI Instructions** - Determine how Gemini chooses sources and transforms stories. Set tone, focus, and more with simple English instructions.
- **Configurable UI** - Custom logo, dashboard title, and color theme.
- **Serve Stories to Other Devices** - The original purpose of this project was to build an application for serving updated, short, custom stories to microcontroller-based smart home displays. The web UI is made to allow full customization of what is served via simple JSON configs.

### Installation on Raspberry Pi

**NOTE:** You will need to use your Raspberry Pi's CLI (command line) to install this application. A 64-bit OS is required (Raspberry Pi OS 64-bit recommended).

#### Option A: Download Pre-Built Binary (Recommended)

1. **Download and extract the latest release**

   ```bash
   cd ~
   wget https://github.com/thinkscotty/maggpi_go/releases/latest/download/maggpi-linux-arm64.tar.gz
   tar -xzf maggpi-linux-arm64.tar.gz
   cd maggpi-release
   chmod +x maggpi
   ```

2. **Run the application**

   ```bash
   ./maggpi
   ```

#### Option B: Build from Source

1. **Install Go** (if not already installed)

   ```bash
   sudo apt update
   sudo apt install -y golang-go git
   ```

2. **Clone and build**

   ```bash
   cd ~
   git clone https://github.com/thinkscotty/maggpi_go.git
   cd maggpi_go
   go build -o maggpi ./cmd/maggpi
   ```

3. **Run the application**

   ```bash
   ./maggpi
   ```

#### After Installation

The application will start on port 7979.

1. **Find your Pi's IP address:**
   ```bash
   hostname -I
   ```

2. **Access the web interface** from any device on your local network:
   ```
   http://<your-pi-ip-address>:7979
   ```

3. **Configure your API key:**
   - Click **Settings** in the navigation menu
   - Enter your **Gemini API Key** ([get one free here](https://aistudio.google.com/apikey))
   - Customize refresh intervals and AI instructions as desired
   - Click **Save Settings**

That's it! MaggPi will automatically start discovering news sources and fetching stories.

## Running as a System Service

To have MaggPi start automatically on boot, create a systemd service:

1. **Create the service file**

   ```bash
   sudo nano /etc/systemd/system/maggpi.service
   ```

2. **Add the following content** (adjust paths and username as needed)

   ```ini
   [Unit]
   Description=MaggPi News Aggregator
   After=network.target

   [Service]
   Type=simple
   User=pi
   WorkingDirectory=/home/pi/maggpi-release
   ExecStart=/home/pi/maggpi-release/maggpi
   Restart=on-failure
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

   **Note:** If you built from source, use `/home/pi/maggpi_go` instead of `/home/pi/maggpi-release`.

3. **Enable and start the service**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable maggpi
   sudo systemctl start maggpi
   ```

4. **Check the status**

   ```bash
   sudo systemctl status maggpi
   ```

## Using MaggPi

### Adding Topics

1. Navigate to the **Topics** page
2. Enter a topic name (e.g., "Space Exploration")
3. Add a description to help the AI find relevant sources
4. Click **Add Topic**

The AI will automatically discover 4-8 relevant news sources for your topic.

### Managing Sources

- Click **Sources** on any topic to view and manage its news sources
- Manually add sources by entering a URL
- Delete unwanted sources with the X button
- AI-discovered sources are marked in blue, manual sources in green

### Customizing Appearance

In the **Settings** page, you can customize:

- **Dashboard Title & Subtitle** - Personalize your dashboard heading
- **Primary & Secondary Colors** - Choose your theme colors
- **Dark Mode** - Toggle dark mode on or off
- **Logo** - Replace the default logo with your own (see below)

### Custom Logo

To use your own logo:

1. Prepare a PNG image with transparent background (recommended size: 512x512px)
2. On your Raspberry Pi, navigate to the `web/static/images/` directory
3. Replace `maggpi-logo-white-512.png` with your logo file (keep the same filename)
4. Restart the application

The logo will automatically scale to 64px height in the navigation bar.

## External API

MaggPi provides a REST API for integrating with external devices like e-ink displays or microcontrollers.

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/stories` | GET | Get all topics with their stories |
| `/v1/topics` | GET | Get list of all topics |
| `/v1/topics/{id}/stories` | GET | Get stories for a specific topic |

### Example

Fetch all stories from the command line:

```bash
curl http://<your-pi-ip>:7979/v1/stories
```

Example response:

```json
{
  "success": true,
  "data": [
    {
      "topic": {
        "id": 1,
        "name": "World News",
        "description": "Major international news and events"
      },
      "stories": [
        {
          "id": 1,
          "title": "Breaking: Major Event Occurs",
          "summary": "A significant event happened today...",
          "source_url": "https://example.com/article",
          "source_title": "Example News",
          "published_at": "2026-02-05T12:00:00Z"
        }
      ]
    }
  ]
}
```

## Updating

To update to the latest version:

1. **Stop the service** (if running as a service)

   ```bash
   sudo systemctl stop maggpi
   ```

2. **Download and extract the new release**

   ```bash
   cd ~
   wget https://github.com/thinkscotty/maggpi_go/releases/latest/download/maggpi-linux-arm64.tar.gz
   tar -xzf maggpi-linux-arm64.tar.gz
   ```

3. **Replace the binary and web files** (adjust path to your installation)

   ```bash
   cp ~/maggpi-release/maggpi ~/maggpi-release/maggpi
   cp -r ~/maggpi-release/web ~/maggpi-release/
   ```

4. **Restart the service**

   ```bash
   sudo systemctl start maggpi
   ```

Your database and settings will be automatically preserved and migrated if needed.

**Alternative:** If you built from source, use `git pull` and rebuild instead:
```bash
cd ~/maggpi_go && git pull origin main && go build -o maggpi ./cmd/maggpi
```

## Configuration

### Configuration File

The application creates `./data/config.json` with these defaults:

```json
{
  "port": 7979,
  "host": "0.0.0.0",
  "data_dir": "./data",
  "database_path": "./data/maggpi.db",
  "debug": false
}
```

You can edit this file to change the port or other settings.

### Command Line Options

```bash
./maggpi -config /path/to/custom-config.json
```

## Troubleshooting

### Application Won't Start

1. Check the logs if running as a service:
   ```bash
   sudo journalctl -u maggpi -f
   ```

2. Ensure port 7979 is not already in use:
   ```bash
   sudo lsof -i :7979
   ```

3. Verify the binary has execute permissions:
   ```bash
   chmod +x maggpi
   ```

### No Stories Appearing

NOTE: The nature of this application requires some time for the AI to do its thing. It can take a few minutes for stories to appear once the Gemini API key is added and topics are refreshed.

1. Verify your Gemini API key is correctly entered in Settings
2. Check that sources were discovered for your topics (view in Topics page)
3. Wait for the refresh interval or manually click the refresh button on a topic
4. Check logs for API errors or rate limiting

### High Memory Usage

MaggPi is optimized for low-power devices. If experiencing memory issues:

- Reduce "Stories Per Topic" in Settings (default: 5)
- Increase the refresh interval (default: 120 minutes)
- Reduce the number of active topics

### API Rate Limiting

The free Gemini API tier has rate limits. If you encounter rate limiting:

- Increase the refresh interval between updates
- Reduce the number of topics
- Failed refreshes automatically retry after 5 minutes

## Building from Source

If you prefer to build from source:

```bash
# Install Go (1.21 or later)
sudo apt update
sudo apt install golang-go -y

# Clone the repository
git clone https://github.com/thinkscotty/maggpi_go.git
cd maggpi_go

# Build
go mod tidy
go build -o bin/maggpi ./cmd/maggpi

# Run
./bin/maggpi
```

## Project Structure

```
maggpi_go/
├── cmd/maggpi/          # Application entry point
├── internal/
│   ├── api/             # HTTP router
│   ├── database/        # SQLite database layer
│   ├── gemini/          # Gemini API client
│   ├── handlers/        # HTTP request handlers
│   ├── models/          # Data structures
│   ├── scheduler/       # Refresh scheduler
│   └── scraper/         # Web scraper (Colly)
├── web/
│   ├── templates/       # HTML templates
│   └── static/          # CSS, JavaScript, images
└── data/                # Database and config (created at runtime)
```

## Technology Stack

- **Backend**: Go 1.21+
- **Database**: SQLite with WAL mode
- **AI**: Google Gemini API
- **Web Scraping**: [Colly v2](https://github.com/gocolly/colly)
- **HTTP Router**: [Chi v5](https://github.com/go-chi/chi)
- **Templates**: Go html/template
- **Frontend**: Vanilla JavaScript, CSS Grid

## Contributing

Contributions are welcome. This is a personal project and I'm nothing close to an expert. Feel free to:

- Report bugs by opening an issue
- Suggest new features
- Submit pull requests

Please test your code thoroughtly before adding it. Thanks!

## License

MIT License - see [LICENSE](LICENSE) file for details. In short: it's open source and you can do what you want with the code. 

## Acknowledgments

- Built with [Go](https://golang.org/)
- AI powered by [Google Gemini](https://ai.google.dev/)
- Web scraping by [Colly](https://github.com/gocolly/colly)
- HTTP routing by [Chi](https://github.com/go-chi/chi)
- Database by [modernc.org/sqlite](https://modernc.org/sqlite)

## Support

- **Issues**: [GitHub Issues](https://github.com/thinkscotty/maggpi_go/issues)
- **Documentation**: See this README
- **Releases**: [GitHub Releases](https://github.com/thinkscotty/maggpi_go/releases)

**AI-GENERATED CODE NOTICE:** Some of this code was generated with Claude Code. Becaude I don't really know how to write Javascript, CSS, or HTML. And I'm pretty new to coding in Go. 

I hope you enjoy my little project! If you build something cool with it, show me!