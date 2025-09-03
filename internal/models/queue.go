package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// QueueData represents the queue information from the DUW website
type QueueData struct {
	Name           string    `json:"name"`
	ServedClients  string    `json:"served_clients"`
	WaitingClients string    `json:"waiting_clients"`
	Workplaces     string    `json:"workplaces"`
	AvgServiceTime string    `json:"avg_service_time"`
	AvgWaitTime    string    `json:"avg_wait_time"`
	LastTicket     string    `json:"last_ticket"`
	TicketsLeft    string    `json:"tickets_left"`
	Status         string    `json:"status"`
	LastUpdated    time.Time `json:"last_updated"`
	LastChanged    time.Time `json:"last_changed"`
}

// QueueChanges represents changes between two queue states
type QueueChanges struct {
	HasChanges    bool
	ChangedFields map[string]bool
	PreviousData  *QueueData
	CurrentData   *QueueData
}

// CompareQueues compares two QueueData instances and returns changes
// Excludes AvgServiceTime and AvgWaitTime from comparison as per requirements
func CompareQueues(previous, current *QueueData) *QueueChanges {
	if previous == nil {
		return &QueueChanges{
			HasChanges:    true,
			ChangedFields: make(map[string]bool),
			PreviousData:  nil,
			CurrentData:   current,
		}
	}

	changes := &QueueChanges{
		HasChanges:    false,
		ChangedFields: make(map[string]bool),
		PreviousData:  previous,
		CurrentData:   current,
	}

	// Compare fields excluding AvgServiceTime and AvgWaitTime
	if previous.Name != current.Name {
		changes.HasChanges = true
		changes.ChangedFields["name"] = true
	}
	if previous.ServedClients != current.ServedClients {
		changes.HasChanges = true
		changes.ChangedFields["served_clients"] = true
	}
	if previous.WaitingClients != current.WaitingClients {
		changes.HasChanges = true
		changes.ChangedFields["waiting_clients"] = true
	}
	if previous.Workplaces != current.Workplaces {
		changes.HasChanges = true
		changes.ChangedFields["workplaces"] = true
	}
	if previous.LastTicket != current.LastTicket {
		changes.HasChanges = true
		changes.ChangedFields["last_ticket"] = true
	}
	if previous.TicketsLeft != current.TicketsLeft {
		changes.HasChanges = true
		changes.ChangedFields["tickets_left"] = true
	}
	if previous.Status != current.Status {
		changes.HasChanges = true
		changes.ChangedFields["status"] = true
	}

	return changes
}

// FormatTelegramMessage formats queue data for Telegram message
func (q *QueueData) FormatTelegramMessage(changes *QueueChanges) string {
	return q.FormatTelegramMessageWithTicket(changes, "")
}

// FormatTelegramMessageWithTicket formats queue data for Telegram message with user ticket info
func (q *QueueData) FormatTelegramMessageWithTicket(changes *QueueChanges, userTicket string) string {
	var builder strings.Builder

	builder.WriteString("🏢 *Очередь: odbiór karty \\(Wrocław\\)*\n\n")

	// Helper function to format field with emoji indicator
	formatField := func(label, value, fieldKey string) {
		var emoji string
		if changes != nil && changes.ChangedFields[fieldKey] {
			emoji = "🟢" // Changed
		} else {
			emoji = "⚪" // Unchanged
		}
		builder.WriteString(fmt.Sprintf("%s *%s:* %s\n", emoji, label, escapeMarkdown(value)))
	}

	formatField("Обслужено", q.ServedClients, "served_clients")
	formatField("Ожидает", q.WaitingClients, "waiting_clients")
	formatField("Стоек", q.Workplaces, "workplaces")
	formatField("Среднее время", q.AvgServiceTime, "avg_service_time")
	formatField("Последний билет", q.LastTicket, "last_ticket")
	formatField("Осталось билетов", q.TicketsLeft, "tickets_left")
	formatField("Статус очереди", q.Status, "status")

	// Show user's estimated wait time if ticket is provided
	if userTicket != "" {
		waitTime, err := q.CalculateWaitTime(userTicket)
		if err == nil && waitTime > 0 {
			hours := waitTime / 60
			minutes := waitTime % 60

			var timeStr string
			if hours > 0 {
				timeStr = fmt.Sprintf("%d ч\\. %d мин\\.", hours, minutes)
			} else {
				timeStr = fmt.Sprintf("%d мин\\.", minutes)
			}

			builder.WriteString(fmt.Sprintf("\n🎫 *Ваш билет %s \\- осталось:* %s", escapeMarkdown(userTicket), timeStr))
		} else if err == nil && waitTime == 0 {
			builder.WriteString(fmt.Sprintf("\n🎫 *Ваш билет %s \\- ваша очередь\\!*", escapeMarkdown(userTicket)))
		}
	}

	// Show last sync time and last change time
	builder.WriteString(fmt.Sprintf("\n🔄 *Синхронизация:* %s", q.LastUpdated.Format("15:04:05")))
	if !q.LastChanged.IsZero() {
		builder.WriteString(fmt.Sprintf("\n⏰ *Изменение:* %s", q.LastChanged.Format("15:04:05")))
	}

	return builder.String()
}

// escapeMarkdown escapes special characters for Telegram MarkdownV2
func escapeMarkdown(text string) string {
	// Characters that need to be escaped in MarkdownV2: _*[]()~`>#+-=|{}.!
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// IsEmpty checks if queue data is empty/invalid
func (q *QueueData) IsEmpty() bool {
	return q.Name == "" && q.ServedClients == "" && q.WaitingClients == ""
}

// Clone creates a deep copy of QueueData
func (q *QueueData) Clone() *QueueData {
	if q == nil {
		return nil
	}
	return &QueueData{
		Name:           q.Name,
		ServedClients:  q.ServedClients,
		WaitingClients: q.WaitingClients,
		Workplaces:     q.Workplaces,
		AvgServiceTime: q.AvgServiceTime,
		AvgWaitTime:    q.AvgWaitTime,
		LastTicket:     q.LastTicket,
		TicketsLeft:    q.TicketsLeft,
		Status:         q.Status,
		LastUpdated:    q.LastUpdated,
		LastChanged:    q.LastChanged,
	}
}

// CalculateWaitTime calculates estimated wait time for a user's ticket
func (q *QueueData) CalculateWaitTime(userTicket string) (int, error) {
	if userTicket == "" || q.LastTicket == "" {
		return 0, fmt.Errorf("missing ticket information")
	}

	// Extract numbers from ticket strings (e.g., "K222" -> 222)
	userNum, err := extractTicketNumber(userTicket)
	if err != nil {
		return 0, fmt.Errorf("invalid user ticket format: %w", err)
	}

	currentNum, err := extractTicketNumber(q.LastTicket)
	if err != nil {
		return 0, fmt.Errorf("invalid current ticket format: %w", err)
	}

	// Calculate tickets remaining
	ticketsRemaining := userNum - currentNum
	if ticketsRemaining <= 0 {
		return 0, nil // User's turn has passed or is current
	}

	// Parse average service time (e.g., "6 min." -> 6)
	avgServiceMinutes, err := parseServiceTime(q.AvgServiceTime)
	if err != nil {
		return 0, fmt.Errorf("invalid service time format: %w", err)
	}

	// Parse number of workplaces (e.g., "3" -> 3)
	workplaces, err := parseWorkplaces(q.Workplaces)
	if err != nil {
		return 0, fmt.Errorf("invalid workplaces format: %w", err)
	}

	// Ensure we have at least 1 workplace to avoid division by zero
	if workplaces <= 0 {
		workplaces = 1
	}

	// Calculate estimated wait time considering parallel workplaces
	// Total time = (tickets remaining * service time per ticket) / number of parallel workplaces
	totalServiceTime := ticketsRemaining * avgServiceMinutes
	estimatedWaitMinutes := totalServiceTime / workplaces

	return estimatedWaitMinutes, nil
}

// extractTicketNumber extracts the numeric part from a ticket string
func extractTicketNumber(ticket string) (int, error) {
	// Remove non-digit characters and parse
	numStr := ""
	for _, char := range ticket {
		if char >= '0' && char <= '9' {
			numStr += string(char)
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in ticket: %s", ticket)
	}

	return strconv.Atoi(numStr)
}

// parseServiceTime extracts minutes from service time string
func parseServiceTime(serviceTime string) (int, error) {
	// Extract number from strings like "6 min." or "5 minutes"
	numStr := ""
	for _, char := range serviceTime {
		if char >= '0' && char <= '9' {
			numStr += string(char)
		} else if numStr != "" {
			break // Stop at first non-digit after finding digits
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in service time: %s", serviceTime)
	}

	return strconv.Atoi(numStr)
}

// parseWorkplaces extracts number from workplaces string
func parseWorkplaces(workplaces string) (int, error) {
	// Extract number from strings like "3" or "5 стоек"
	numStr := ""
	for _, char := range workplaces {
		if char >= '0' && char <= '9' {
			numStr += string(char)
		} else if numStr != "" {
			break // Stop at first non-digit after finding digits
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in workplaces: %s", workplaces)
	}

	return strconv.Atoi(numStr)
}
