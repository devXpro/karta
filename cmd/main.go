package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"karta/internal/bot"
	"karta/internal/database"
	"karta/internal/models"
	"karta/internal/parser"
)

const (
	DefaultDatabasePath    = "karta.db"
	MonitoringInterval     = 11 * time.Second
	HistoryCleanupInterval = 24 * time.Hour
	HistoryRetentionPeriod = 7 * 24 * time.Hour // Keep 7 days of history
)

// getDatabasePath returns database path from environment or default
func getDatabasePath() string {
	if path := os.Getenv("DATABASE_PATH"); path != "" {
		return path
	}
	return DefaultDatabasePath
}

// Application represents the main application
type Application struct {
	db          *database.Database
	bot         *bot.TelegramBot
	parser      *parser.QueueParser
	lastData    *models.QueueData
	lastChanged time.Time
	lastChanges *models.QueueChanges // Store last changes to show red circles
	mu          sync.RWMutex
}

func main() {
	log.Println("Starting Karta Queue Monitor...")

	// Get Telegram bot token from environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	// Initialize database
	db, err := database.NewDatabase(getDatabasePath())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Telegram bot
	telegramBot, err := bot.NewTelegramBot(botToken, db)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	// Initialize queue parser
	queueParser := parser.NewQueueParser()

	// Create application instance
	app := &Application{
		db:          db,
		bot:         telegramBot,
		parser:      queueParser,
		lastChanged: time.Now(),
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start components
	var wg sync.WaitGroup

	// Start Telegram bot
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := telegramBot.Start(ctx); err != nil {
			log.Printf("Telegram bot error: %v", err)
		}
	}()

	// Start queue monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.startQueueMonitoring(ctx)
	}()

	// Start periodic cleanup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.startPeriodicCleanup(ctx)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Application started successfully. Press Ctrl+C to stop.")

	<-sigChan
	log.Println("Shutdown signal received, stopping application...")

	// Cancel context to stop all goroutines
	cancel()

	// Stop bot
	telegramBot.Stop()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		log.Println("Application stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Shutdown timeout, forcing exit")
	}
}

// startQueueMonitoring starts the queue monitoring process
func (app *Application) startQueueMonitoring(ctx context.Context) {
	log.Printf("Starting queue monitoring with %v interval", MonitoringInterval)

	app.parser.StartMonitoring(ctx, MonitoringInterval, func(queueData *models.QueueData, err error) {
		if err != nil {
			log.Printf("Failed to parse queue data: %v", err)
			return
		}

		if err := parser.ValidateQueueData(queueData); err != nil {
			log.Printf("Invalid queue data: %v", err)
			return
		}

		app.processQueueUpdate(queueData)
	})
}

// processQueueUpdate processes new queue data and sends notifications if needed
func (app *Application) processQueueUpdate(newData *models.QueueData) {
	app.mu.Lock()
	defer app.mu.Unlock()

	log.Printf("Processing queue update: %+v", newData)

	// Save to database
	if err := app.db.SaveQueueHistory(newData); err != nil {
		log.Printf("Failed to save queue history: %v", err)
	}

	// Compare with previous data
	changes := models.CompareQueues(app.lastData, newData)

	if app.lastData == nil {
		// First run - set initial change time
		app.lastChanged = time.Now()
		newData.LastChanged = app.lastChanged
		app.lastChanges = nil // No changes to highlight on first run
		log.Printf("First queue data received")
	} else if changes.HasChanges {
		// Data changed - update change time and store changes
		app.lastChanged = time.Now()
		newData.LastChanged = app.lastChanged
		app.lastChanges = changes // Store changes to show red circles
		log.Printf("Queue data changed: %+v", changes.ChangedFields)
	} else {
		// No changes - keep previous change time and previous changes for red circles
		newData.LastChanged = app.lastChanged
		// Keep showing red circles from last change
	}

	// Send notifications to users (always update to show sync time)
	// Use stored changes to keep showing red circles until next change
	changesToShow := app.lastChanges
	if changes.HasChanges {
		changesToShow = changes // Show new changes
	}

	if err := app.bot.BroadcastQueueUpdate(newData, changesToShow); err != nil {
		log.Printf("Failed to broadcast queue update: %v", err)
	}

	// Update last data
	app.lastData = newData.Clone()

	// Log statistics
	if stats, err := app.bot.GetStats(); err == nil {
		log.Printf("Bot stats: %+v", stats)
	}
}

// startPeriodicCleanup starts periodic database cleanup
func (app *Application) startPeriodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(HistoryCleanupInterval)
	defer ticker.Stop()

	log.Printf("Starting periodic cleanup with %v interval", HistoryCleanupInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Periodic cleanup stopped")
			return
		case <-ticker.C:
			if err := app.db.CleanOldHistory(HistoryRetentionPeriod); err != nil {
				log.Printf("Failed to clean old history: %v", err)
			}
		}
	}
}
