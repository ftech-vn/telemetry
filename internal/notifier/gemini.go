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
	"time"
)

// GeminiNotifier sends Gemini responses to the monitor-app backend.
type GeminiNotifier struct {
	webhookURL string // Base webhook URL (e.g., http://localhost:5000/api/metrics)
	serverID   string
	serverKey  string
}

// geminiPayload is the payload format expected by the monitor-app backend.
type geminiPayload struct {
	ServerID  string `json:"server_id"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// NewGeminiNotifier creates a new GeminiNotifier.
// geminiWebhookURL is the full URL to the Gemini webhook endpoint (e.g., http://localhost:5000/api/gemini)
func NewGeminiNotifier(geminiWebhookURL string, serverID string, serverKey string) *GeminiNotifier {
	return &GeminiNotifier{
		webhookURL: geminiWebhookURL,
		serverID:   serverID,
		serverKey:  serverKey,
	}
}

// Notify sends a Gemini response to the backend for WebSocket broadcast.
func (n *GeminiNotifier) Notify(prompt string, response string, success bool, errMsg string) error {
	if n.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload := geminiPayload{
		ServerID:  n.serverID,
		Prompt:    prompt,
		Response:  response,
		Success:   success,
		Error:     errMsg,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
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
		return fmt.Errorf("failed to send gemini notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf(" Gemini response sent to backend (status: %s)", resp.Status)
		return nil
	}

	return fmt.Errorf("gemini notification failed with status: %s", resp.Status)
}
