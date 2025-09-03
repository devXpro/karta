package models

import (
	"fmt"
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
	formatField("Всего времени", q.AvgWaitTime, "avg_wait_time")
	formatField("Последний билет", q.LastTicket, "last_ticket")
	formatField("Осталось билетов", q.TicketsLeft, "tickets_left")
	formatField("Статус очереди", q.Status, "status")

	// Show last sync time and last change time
	builder.WriteString(fmt.Sprintf("\n🔄 *Синхронизация:* %s", q.LastUpdated.Format("15:04:05")))
	if !q.LastChanged.IsZero() {
		builder.WriteString(fmt.Sprintf("\n⏰ *Последнее изменение:* %s", q.LastChanged.Format("15:04:05")))
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
