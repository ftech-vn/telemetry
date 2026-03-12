package notifier

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"telemetry/internal/monitor"
	"time"
)

// WebhookNotifier sends metrics to the monitor-app backend webhook URL.
type WebhookNotifier struct {
	webhookURL string
	serverID   string
	serverKey  string
}

// webhookPayload is the payload format expected by the monitor-app backend.
type webhookPayload struct {
	ServerID string          `json:"server_id"`
	Data     []monitor.Alert `json:"data"`
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(webhookURL string, serverID string, serverKey string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		serverID:   serverID,
		serverKey:  serverKey,
	}
}

// Notify sends the metrics to the configured webhook URL with HMAC-SHA256 signature.
func (n *WebhookNotifier) Notify(alerts []monitor.Alert) {
	if n.webhookURL == "" {
		return
	}

	payload := webhookPayload{
		ServerID: n.serverID,
		Data:     alerts,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf(" Failed to marshal alerts for webhook: %v", err)
		return
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf(" Failed to create webhook request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request body with HMAC-SHA256
	if n.serverKey != "" {
		mac := hmac.New(sha256.New, []byte(n.serverKey))
		mac.Write(data)
		signature := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
		req.Header.Set("x-hub-signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf(" Failed to send webhook notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf(" Webhook metrics sent (status: %s)", resp.Status)
	} else {
		log.Printf(" Webhook notification failed with status: %s", resp.Status)
	}
}
