package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"karta/internal/database"
	"karta/internal/models"
)

// TelegramBot represents the Telegram bot instance
type TelegramBot struct {
	api      *tgbotapi.BotAPI
	db       *database.Database
	userMsgs sync.Map // map[int64]int - stores chat_id -> message_id for updates
}

// NewTelegramBot creates a new Telegram bot instance
func NewTelegramBot(token string, db *database.Database) (*TelegramBot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &TelegramBot{
		api: api,
		db:  db,
	}, nil
}

// Start starts the bot and handles incoming messages
func (b *TelegramBot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Telegram bot started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Telegram bot stopped")
			return nil
		case update := <-updates:
			if update.Message != nil {
				go b.handleMessage(update.Message)
			}
		}
	}
}

// handleMessage processes incoming messages
func (b *TelegramBot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	username := message.From.UserName

	log.Printf("Received message from %s (ID: %d): %s", username, chatID, message.Text)

	switch message.Command() {
	case "start":
		b.handleStartCommand(chatID, username)
	default:
		if message.Text != "" {
			b.sendMessage(chatID, "Используйте команду /start для получения информации о очереди.")
		}
	}
}

// handleStartCommand handles the /start command
func (b *TelegramBot) handleStartCommand(chatID int64, username string) {
	// Add user to database
	if err := b.db.AddUser(chatID, username); err != nil {
		log.Printf("Failed to add user to database: %v", err)
		b.sendMessage(chatID, "Произошла ошибка при регистрации. Попробуйте позже.")
		return
	}

	// Get latest queue data
	queueData, err := b.db.GetLatestQueueData()
	if err != nil {
		log.Printf("Failed to get latest queue data: %v", err)
		b.sendMessage(chatID, "Добро пожаловать! Данные о очереди будут доступны после первого обновления.")
		return
	}

	if queueData == nil {
		b.sendMessage(chatID, "Добро пожаловать! Данные о очереди пока недоступны. Ожидайте первого обновления.")
		return
	}

	// Send current queue data
	message := queueData.FormatTelegramMessage(nil)
	msgID := b.sendMessage(chatID, message)

	// Store message ID for future updates
	if msgID != 0 {
		b.userMsgs.Store(chatID, msgID)
	}
}

// sendMessage sends a message to a chat and returns message ID
func (b *TelegramBot) sendMessage(chatID int64, text string) int {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.DisableWebPagePreview = true

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
		return 0
	}

	return sentMsg.MessageID
}

// updateMessage updates an existing message
func (b *TelegramBot) updateMessage(chatID int64, messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.DisableWebPagePreview = true

	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Failed to update message for %d: %v", chatID, err)
		return err
	}

	return nil
}

// BroadcastQueueUpdate sends queue updates to all active users
func (b *TelegramBot) BroadcastQueueUpdate(queueData *models.QueueData, changes *models.QueueChanges) error {
	users, err := b.db.GetActiveUsers()
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	if len(users) == 0 {
		log.Println("No active users to broadcast to")
		return nil
	}

	message := queueData.FormatTelegramMessage(changes)

	log.Printf("Broadcasting queue update to %d users", len(users))

	var successCount, errorCount int

	for _, user := range users {
		// Try to update existing message first
		if msgIDInterface, exists := b.userMsgs.Load(user.ChatID); exists {
			if msgID, ok := msgIDInterface.(int); ok {
				if err := b.updateMessage(user.ChatID, msgID, message); err == nil {
					successCount++
					continue
				}
				// If update fails, remove stored message ID and send new message
				b.userMsgs.Delete(user.ChatID)
			}
		}

		// Send new message
		msgID := b.sendMessage(user.ChatID, message)
		if msgID != 0 {
			b.userMsgs.Store(user.ChatID, msgID)
			successCount++
		} else {
			errorCount++
			// Deactivate user if message sending fails (user might have blocked the bot)
			if err := b.db.DeactivateUser(user.ChatID); err != nil {
				log.Printf("Failed to deactivate user %d: %v", user.ChatID, err)
			}
		}

		// Small delay to avoid hitting rate limits
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("Broadcast completed: %d successful, %d errors", successCount, errorCount)
	return nil
}

// GetStats returns bot statistics
func (b *TelegramBot) GetStats() (map[string]interface{}, error) {
	userCount, err := b.db.GetUserCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}

	stats := map[string]interface{}{
		"active_users":    userCount,
		"stored_messages": b.getStoredMessageCount(),
		"bot_username":    b.api.Self.UserName,
	}

	return stats, nil
}

// getStoredMessageCount returns the number of stored message IDs
func (b *TelegramBot) getStoredMessageCount() int {
	count := 0
	b.userMsgs.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Stop gracefully stops the bot
func (b *TelegramBot) Stop() {
	b.api.StopReceivingUpdates()
	log.Println("Telegram bot stopped receiving updates")
}
