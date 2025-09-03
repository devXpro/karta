package parser

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"karta/internal/models"
)

const (
	DUWStatusURL = "https://rezerwacje.duw.pl/app/webroot/status_kolejek/query.php?status"
	UserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// APIResponse represents the JSON response from DUW API
type APIResponse struct {
	Result map[string][]QueueItem `json:"result"`
}

// QueueItem represents a single queue item from the API
type QueueItem struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	TicketCount        int    `json:"ticket_count"`
	TicketsServed      int    `json:"tickets_served"`
	Workplaces         int    `json:"workplaces"`
	AverageWaitTime    int    `json:"average_wait_time"`    // in seconds
	AverageServiceTime int    `json:"average_service_time"` // in seconds
	TicketValue        string `json:"ticket_value"`
	TicketsLeft        int    `json:"tickets_left"`
	Enabled            bool   `json:"enabled"`
	Active             bool   `json:"active"`
	Location           string `json:"location"`
}

// QueueParser handles parsing of DUW queue status page
type QueueParser struct {
	client *http.Client
}

// NewQueueParser creates a new queue parser instance
func NewQueueParser() *QueueParser {
	// Create HTTP client with insecure TLS config for problematic SSL certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &QueueParser{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: tr,
		},
	}
}

// ParseQueueData fetches and parses queue data from DUW API
func (p *QueueParser) ParseQueueData(ctx context.Context) (*models.QueueData, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", DUWStatusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	queueData, err := p.extractQueueDataFromAPI(&apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to extract queue data: %w", err)
	}

	queueData.LastUpdated = time.Now()
	return queueData, nil
}

// extractQueueDataFromAPI extracts queue data from the API response
func (p *QueueParser) extractQueueDataFromAPI(apiResponse *APIResponse) (*models.QueueData, error) {
	// Look for Wrocław queues
	wroclawQueues, exists := apiResponse.Result["Wrocław"]
	if !exists {
		return nil, fmt.Errorf("Wrocław section not found in API response")
	}

	// Find the "odbiór karty" queue
	for _, queue := range wroclawQueues {
		if queue.Name == "odbiór karty" {
			log.Printf("Found 'odbiór karty' queue: %+v", queue)

			// Convert time from seconds to human-readable format
			avgServiceTime := formatTime(queue.AverageServiceTime)
			avgWaitTime := formatTime(queue.AverageWaitTime)

			// Determine status
			status := "Dostępna"
			if !queue.Enabled || !queue.Active {
				status = "Zamknięta"
			}

			queueData := &models.QueueData{
				Name:           queue.Name,
				ServedClients:  strconv.Itoa(queue.TicketsServed),
				WaitingClients: strconv.Itoa(queue.TicketCount),
				Workplaces:     strconv.Itoa(queue.Workplaces),
				AvgServiceTime: avgServiceTime,
				AvgWaitTime:    avgWaitTime,
				LastTicket:     queue.TicketValue,
				TicketsLeft:    strconv.Itoa(queue.TicketsLeft),
				Status:         status,
			}

			log.Printf("Extracted queue data: %+v", queueData)
			return queueData, nil
		}
	}

	return nil, fmt.Errorf("queue 'odbiór karty' not found in Wrocław section")
}

// formatTime converts seconds to human-readable format
func formatTime(seconds int) string {
	if seconds <= 0 {
		return "N/A"
	}

	if seconds < 60 {
		return fmt.Sprintf("%d s.", seconds)
	}

	minutes := seconds / 60
	return fmt.Sprintf("%d min.", minutes)
}

// StartMonitoring starts continuous monitoring of queue data
func (p *QueueParser) StartMonitoring(ctx context.Context, interval time.Duration, callback func(*models.QueueData, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting queue monitoring with %v interval", interval)

	// Parse immediately on start
	go func() {
		data, err := p.ParseQueueData(ctx)
		callback(data, err)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Queue monitoring stopped")
			return
		case <-ticker.C:
			data, err := p.ParseQueueData(ctx)
			callback(data, err)
		}
	}
}

// ValidateQueueData performs basic validation on parsed queue data
func ValidateQueueData(data *models.QueueData) error {
	if data == nil {
		return fmt.Errorf("queue data is nil")
	}

	if data.Name == "" {
		return fmt.Errorf("queue name is empty")
	}

	if !strings.Contains(strings.ToLower(data.Name), "odbiór karty") {
		return fmt.Errorf("invalid queue name: %s", data.Name)
	}

	return nil
}
