package notifier

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"telemetry/internal/monitor"
	"time"
)

// WebhookNotifier sends alerts to a generic webhook URL.
type WebhookNotifier struct {
	webhookURL string
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
	}
}

// Notify sends the alert to the configured webhook URL.
func (n *WebhookNotifier) Notify(alerts []monitor.Alert) {
	if n.webhookURL == "" {
		return
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		log.Printf("❌ Failed to marshal alerts for webhook: %v", err)
		return
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("❌ Failed to create webhook request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to send webhook notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Webhook notification failed with status: %s", resp.Status)
	}
}
