# Karta Queue Monitor

A Go application for monitoring queues on the DUW (Dolnośląski Urząd Wojewódzki) website with notifications via Telegram bot.

## Description

The application tracks the "odbiór karty" (card pickup) queue for Wrocław city through the DUW website's JSON API and sends notifications to Telegram bot users when data changes.

**API Endpoint:** `https://rezerwacje.duw.pl/app/webroot/status_kolejek/query.php?status`

## Features

- 🔍 **JSON API Parsing**: Fetches data through official API every 11 seconds
- 📱 **Telegram Bot**: Notifications and commands via Telegram
- 💾 **Database**: SQLite for storing users and history
- 🔔 **Smart Notifications**: Highlights changes in red
- ⏰ **Time Tracking**: Shows last change time
- 🚀 **High Performance**: Uses JSON API instead of HTML parsing
- 🎫 **Personal Ticket Tracking**: Users can register their ticket numbers for personalized wait time estimates
- 🇵🇱 **VPN Support**: Docker deployment with Polish VPN for geo-restricted access

## Installation and Setup

### Prerequisites

- Go 1.23 or higher
- Telegram Bot Token (get from @BotFather)
- Docker and Docker Compose (for containerized deployment)
- SurfShark VPN account (for Docker deployment)

### Local Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd karta
```

2. Install dependencies:
```bash
go mod download
```

3. Create `.env` file based on `.env.example`:
```bash
cp .env.example .env
```

4. Add your Telegram Bot Token to `.env`:
```bash
TELEGRAM_BOT_TOKEN=your_actual_bot_token_here
```

### Docker Installation (Recommended)

1. Copy the environment file:
```bash
cp .env.example .env
```

2. Fill in the variables in `.env`:
```bash
# Telegram Bot Token
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# SurfShark VPN Credentials
SURFSHARK_USER=your_surfshark_email
SURFSHARK_PASSWORD=your_surfshark_password
```

3. Create data directory:
```bash
mkdir -p data
```

4. Build and run:
```bash
docker compose up -d
```

### Getting Telegram Bot Token

1. Find @BotFather in Telegram
2. Send `/newbot` command
3. Follow instructions to create a bot
4. Copy the received token

### Running

#### Local Run
```bash
# Set environment variable
export TELEGRAM_BOT_TOKEN="your_bot_token_here"

# Run the application
go run cmd/main.go

# Or compile and run
go build -o karta cmd/main.go
./karta
```

#### Docker Run
```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f karta-bot

# Stop services
docker compose down
```

## Usage

1. Find your bot in Telegram
2. Send `/start` command
3. Get current queue information
4. **Register your ticket**: Send your ticket number (e.g., `K222`) to get personalized wait time estimates
5. Bot will automatically send updates when changes occur

### Ticket Tracking Feature

- Send your ticket number in format `K123` to register it
- Bot will calculate and show your estimated wait time
- Wait time calculation: `(your_ticket_number - current_ticket) × average_service_time ÷ number_of_workplaces`
- Example: If current ticket is K065, your ticket is K222, average service time is 6 min, and there are 3 workplaces:
  - Wait time = (222 - 65) × 6 ÷ 3 = 314 minutes = 5h 14min

## Project Structure

```
karta/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── bot/
│   │   └── telegram_bot.go     # Telegram bot
│   ├── database/
│   │   └── sqlite.go           # SQLite operations
│   ├── models/
│   │   └── queue.go            # Data models
│   └── parser/
│       └── queue_parser.go     # JSON API parser
├── docker-compose.yml          # Docker Compose configuration
├── Dockerfile                  # Docker build configuration
├── .env.example                # Environment variables example
├── go.mod                      # Go module
└── README.md                   # Documentation
```

## Tracked Data

The application tracks the following fields for the "odbiór karty" queue:
- Served clients
- Waiting clients
- Number of workplaces
- Last ticket number
- Tickets left
- Queue status
- Average service time
- Average wait time

## Bot Commands

- `/start` - Registration and get current queue data
- `K123` - Register your ticket number for personalized tracking

## Technical Details

- **Update interval**: 11 seconds
- **Database**: SQLite (file `karta.db` or `/data/karta.db` in Docker)
- **Message format**: Telegram MarkdownV2
- **Error handling**: Logging and graceful shutdown
- **History cleanup**: Automatic cleanup of data older than 7 days
- **SSL handling**: Bypasses SSL verification for problematic certificates
- **VPN**: Uses SurfShark VPN for Polish IP address in Docker deployment

## Docker Monitoring

```bash
# Check container status
docker compose ps

# View bot logs
docker compose logs karta-bot

# View VPN logs
docker compose logs surfshark

# Check IP address (should be Polish)
docker compose exec karta-bot wget -qO- ifconfig.me

# Restart services
docker compose restart
```

## Logging

The application maintains detailed logs:
- Data parsing status
- Queue changes
- User statistics
- Errors and warnings
- Ticket registration events

## Stopping the Application

### Local
Use `Ctrl+C` for graceful shutdown of all components.

### Docker
```bash
docker compose down
```

## Troubleshooting

### VPN Issues
- Check logs: `docker compose logs surfshark`
- Verify SurfShark credentials in `.env`
- Ensure Polish IP: `docker compose exec karta-bot wget -qO- ifconfig.me`

### Bot Issues
- Check logs: `docker compose logs karta-bot`
- Verify Telegram Bot Token in `.env`
- Ensure bot is not blocked by users

### SSL Issues
The application automatically bypasses SSL verification for the DUW website. If you encounter SSL errors, they should be automatically handled.
