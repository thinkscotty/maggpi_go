# MaggPi - AI-Powered News Aggregator

MaggPi is a lightweight, self-hosted news aggregator that uses Google's Gemini AI to discover relevant news sources and summarize stories based on your interests. Designed to run on minimal hardware like a Raspberry Pi.

## Features

- **User-Defined Topics**: Add any topic you're interested in with a description
- **AI Source Discovery**: Gemini AI automatically finds relevant news sources for each topic
- **Smart Summarization**: Stories are intelligently summarized (75-150 words each)
- **Clean Web Interface**: Simple, responsive dashboard to browse your news
- **External API**: JSON API for feeding news to other devices (displays, microcontrollers)
- **Low Resource Usage**: Designed for Raspberry Pi 3B+ and similar hardware
- **Customizable**: Global AI instructions, color themes, and refresh intervals

## Screenshots

(Coming soon)

## Requirements

- **Hardware**: Raspberry Pi 3B+ or better (or any Linux/macOS/Windows system)
- **OS**: Debian-based Linux recommended (tested on Debian 13 Trixie)
- **API Key**: Free Google Gemini API key ([Get one here](https://aistudio.google.com/apikey))

## Quick Start

### Option 1: Build on Mac and Publish as GitHub Release (Recommended)

This option builds the binary on your Mac, publishes it to GitHub as a release, then downloads it on the Pi.

#### Step 1A: Build the Binary on Your Mac

**On your Mac (development machine):**

1. **Install Go** (if not already installed):
   ```bash
   # Check if Go is installed
   go version

   # If not installed, download from https://go.dev/dl/
   # Or install via Homebrew:
   brew install go
   ```

2. **Clone the repository**:
   ```bash
   cd ~/code_projects  # Or your preferred directory
   git clone https://github.com/thinkscotty/maggpi_go.git
   cd maggpi_go
   ```

3. **Download dependencies**:
   ```bash
   go mod tidy
   ```

4. **Build the binary for Raspberry Pi** (ARM64):
   ```bash
   GOOS=linux GOARCH=arm64 go build -o maggpi-linux-arm64 ./cmd/maggpi
   ```

   This creates a file called `maggpi-linux-arm64` in your current directory.

5. **Verify the binary was created**:
   ```bash
   ls -lh maggpi-linux-arm64
   ```

   You should see a file around 30-35 MB.

#### Step 1B: Create a Release Package

**On your Mac:**

1. **Create a release directory**:
   ```bash
   mkdir -p release/maggpi
   ```

2. **Copy files to release directory**:
   ```bash
   # Copy the binary
   cp maggpi-linux-arm64 release/maggpi/maggpi

   # Copy the web directory (templates and static files)
   cp -r web release/maggpi/

   # Copy README for reference
   cp README.md release/maggpi/
   ```

3. **Create a compressed archive**:
   ```bash
   cd release
   tar -czf maggpi-v1.0.0-linux-arm64.tar.gz maggpi/
   cd ..
   ```

4. **Verify the archive**:
   ```bash
   ls -lh release/maggpi-v1.0.0-linux-arm64.tar.gz
   ```

#### Step 1C: Publish to GitHub as a Release

**On your Mac:**

1. **Install GitHub CLI** (if not already installed):
   ```bash
   brew install gh
   ```

2. **Authenticate with GitHub** (first time only):
   ```bash
   gh auth login
   ```

   Follow the prompts:
   - Choose "GitHub.com"
   - Choose "HTTPS"
   - Authenticate via web browser

3. **Tag your code** (if not already tagged):
   ```bash
   git tag -a v1.0.0 -m "Initial release of MaggPi"
   git push origin v1.0.0
   ```

4. **Create a GitHub release with the binary**:
   ```bash
   gh release create v1.0.0 \
     release/maggpi-v1.0.0-linux-arm64.tar.gz \
     --title "MaggPi v1.0.0" \
     --notes "Initial release of MaggPi - AI-Powered News Aggregator

   ## Features
   - AI source discovery via Gemini API
   - Smart story summarization
   - Web UI with dashboard, topics, and settings
   - JSON API for external clients
   - Designed for Raspberry Pi 3B+

   ## Installation
   1. Download maggpi-v1.0.0-linux-arm64.tar.gz
   2. Extract on your Raspberry Pi
   3. Run ./maggpi
   4. Open http://192.168.0.101:7979 in your browser
   5. Configure your Gemini API key in Settings

   See README.md for full documentation."
   ```

5. **Verify the release**:
   - Visit: https://github.com/thinkscotty/maggpi_go/releases
   - You should see v1.0.0 with the attached binary file

#### Step 1D: Download and Run on Raspberry Pi

**On your Raspberry Pi:**

1. **Download the release**:
   ```bash
   cd ~
   wget https://github.com/thinkscotty/maggpi_go/releases/download/v1.0.0/maggpi-v1.0.0-linux-arm64.tar.gz
   ```

2. **Extract the archive**:
   ```bash
   tar -xzf maggpi-v1.0.0-linux-arm64.tar.gz
   cd maggpi
   ```

3. **Make the binary executable**:
   ```bash
   chmod +x maggpi
   ```

4. **Run the application**:
   ```bash
   ./maggpi
   ```

5. **Open in your browser** (from any device on the network):
   - Navigate to: `http://192.168.0.101:7979`

6. **Configure the application**:
   - Click **Settings** in the navigation
   - Enter your **Gemini API Key** (get one at https://aistudio.google.com/apikey)
   - Adjust refresh intervals and AI instructions as desired
   - Click **Save Settings**

The application will start discovering sources and fetching stories for the default topics.

---

### Option 2: Build Directly on Raspberry Pi

**On Raspberry Pi:**

#### Prerequisites

Install Go 1.21 or later:

```bash
sudo apt update
sudo apt install golang-go

# Verify installation
go version
```

#### Build Steps

1. **Clone the repository**:
   ```bash
   cd ~
   git clone https://github.com/thinkscotty/maggpi_go.git
   cd maggpi_go
   ```

2. **Download dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the application**:
   ```bash
   go build -o bin/maggpi ./cmd/maggpi
   ```

   *Note: Building on the Pi may take 5-10 minutes.*

4. **Run the application**:
   ```bash
   ./bin/maggpi
   ```

5. **Configure** (same as Option 1, step 6 above)

---

### Option 3: Quick Transfer via SCP (No GitHub Release)

**On your Mac:**

1. **Build the binary** (follow Option 1, Step 1A above)

2. **Transfer to Pi**:
   ```bash
   # Create directory on Pi
   ssh scotty@192.168.0.101 "mkdir -p ~/maggpi"

   # Transfer binary
   scp maggpi-linux-arm64 scotty@192.168.0.101:~/maggpi/maggpi

   # Transfer web directory
   scp -r web scotty@192.168.0.101:~/maggpi/
   ```

**On Raspberry Pi:**

3. **Run the application**:
   ```bash
   cd ~/maggpi
   chmod +x maggpi
   ./maggpi
   ```

## Deployment on Raspberry Pi

This guide assumes you're deploying to a Raspberry Pi 3B+ running Pi OS 64-bit (Debian 13 Trixie).

### Step 1: Prepare the Pi

**On Raspberry Pi:**

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install git (if not already installed)
sudo apt install git -y
```

### Step 2: Clone and Build

**On Raspberry Pi:**

```bash
# Clone the repository
cd ~
git clone https://github.com/thinkscotty/maggpi_go.git
cd maggpi_go

# Install Go if not already installed
sudo apt install golang-go -y

# Build
go mod tidy
go build -o bin/maggpi ./cmd/maggpi
```

### Step 3: Install as a Service

**On Raspberry Pi:**

Create a systemd service file:

```bash
sudo nano /etc/systemd/system/maggpi.service
```

Add the following content:

```ini
[Unit]
Description=MaggPi News Aggregator
After=network.target

[Service]
Type=simple
User=scotty
WorkingDirectory=/home/scotty/maggpi_go
ExecStart=/home/scotty/maggpi_go/bin/maggpi
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable maggpi
sudo systemctl start maggpi
```

Check status:
```bash
sudo systemctl status maggpi
```

### Step 4: Configure the Application

**From any device on your network:**

1. Open a web browser and navigate to `http://192.168.0.101:7979`

2. Go to **Settings** page

3. Enter your **Gemini API Key**
   - Get a free key from [Google AI Studio](https://aistudio.google.com/apikey)

4. Customize refresh intervals and AI instructions as desired

5. Click **Save Settings**

The default topics (World News, Formula 1, Science News, Tech News) will automatically begin discovering sources and fetching stories.

## Configuration

### Configuration File

The application creates a configuration file at `./data/config.json`:

```json
{
  "port": 7979,
  "host": "0.0.0.0",
  "data_dir": "./data",
  "database_path": "./data/maggpi.db",
  "debug": false
}
```

### Command Line Options

```bash
./maggpi -config /path/to/config.json
```

## External API

MaggPi provides a JSON API for external clients like microcontroller displays:

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/stories` | GET | Get all topics with their stories |
| `/v1/topics` | GET | Get list of all topics |
| `/v1/topics/{id}/stories` | GET | Get stories for a specific topic |

### Example Response

```json
{
  "success": true,
  "data": [
    {
      "topic": {
        "id": 1,
        "name": "World News",
        "description": "Major international news..."
      },
      "stories": [
        {
          "id": 1,
          "title": "Breaking: Major Event Happens",
          "summary": "A significant event occurred today...",
          "source_url": "https://example.com/article",
          "source_title": "Example News"
        }
      ]
    }
  ]
}
```

### Example: Fetch Stories with curl

**From any device on your network:**

```bash
curl http://192.168.0.101:7979/v1/stories
```

## Managing Topics

### Adding a Topic

1. Navigate to **Topics** page
2. Enter a **Topic Name** (e.g., "Space Exploration")
3. Enter a **Description** that helps the AI find relevant sources
4. Click **Add Topic**

The AI will automatically discover 4-8 relevant sources for your topic.

### Managing Sources

- Click **Sources** on any topic to view its sources
- Manually add sources using the URL form
- Delete sources by clicking the X button
- AI-discovered sources are marked in blue; manual sources in green

### Reordering Topics

Drag and drop topics to change their display order on the dashboard.

## Updating

**On Raspberry Pi:**

To update to the latest version:

```bash
cd ~/maggpi_go
git pull
go mod tidy
go build -o bin/maggpi ./cmd/maggpi
sudo systemctl restart maggpi
```

## Troubleshooting

### Application Won't Start

**On Raspberry Pi:**

1. Check the logs:
   ```bash
   sudo journalctl -u maggpi -f
   ```

2. Verify the configuration file is valid JSON

3. Ensure port 7979 is not in use:
   ```bash
   sudo lsof -i :7979
   ```

### No Stories Appearing

1. Verify your Gemini API key is set correctly in Settings
2. Check if sources were discovered for your topics
3. Wait for the refresh interval or manually click the refresh button
4. Check logs for API errors

### High Memory Usage

The application is designed for limited hardware. If memory is an issue:
- Reduce "Stories Per Topic" in Settings
- Increase the refresh interval
- Reduce the number of topics

### API Rate Limiting

The free tier Gemini API has rate limits. MaggPi automatically staggers requests, but if you hit limits:
- Increase the refresh interval
- Reduce the number of topics
- Failed refreshes automatically retry after 5 minutes

## Development

### Running in Development Mode

**On your development machine:**

```bash
# Install air for hot reload
go install github.com/air-verse/air@latest

# Run with hot reload
make dev
```

### Project Structure

```
maggpi_go/
├── cmd/maggpi/          # Application entry point
├── internal/
│   ├── api/             # HTTP router
│   ├── config/          # Configuration management
│   ├── database/        # SQLite database layer
│   ├── gemini/          # Gemini API client
│   ├── handlers/        # HTTP handlers
│   ├── models/          # Data models
│   ├── scheduler/       # Refresh scheduler
│   └── scraper/         # Web scraper
├── web/
│   ├── templates/       # HTML templates
│   └── static/          # CSS and JavaScript
├── data/                # Database and config storage
├── Makefile
└── README.md
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Credits

- Built with [Go](https://golang.org/)
- AI powered by [Google Gemini](https://ai.google.dev/)
- Web scraping by [Colly](https://github.com/gocolly/colly)
- Routing by [Chi](https://github.com/go-chi/chi)
- Database by [modernc.org/sqlite](https://modernc.org/sqlite)
