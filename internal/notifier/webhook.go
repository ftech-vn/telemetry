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
	webhookURL      string
	serverID        string
	serverKey       string
	cpuThreshold    *float64
	memoryThreshold *float64
	diskThreshold   *float64
	// Track last sent thresholds to detect changes
	lastSentCPU    *float64
	lastSentMemory *float64
	lastSentDisk   *float64
	thresholdsSent bool // flag to track if we've ever sent thresholds
}

// webhookPayload is the payload format expected by the monitor-app backend.
type webhookPayload struct {
	ServerID        string          `json:"server_id"`
	Data            []monitor.Alert `json:"data"`
	CPUThreshold    *float64        `json:"cpu_threshold,omitempty"`
	MemoryThreshold *float64        `json:"memory_threshold,omitempty"`
	DiskThreshold   *float64        `json:"disk_threshold,omitempty"`
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(webhookURL string, serverID string, serverKey string, cpuThreshold *float64, memoryThreshold *float64, diskThreshold *float64) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL:      webhookURL,
		serverID:        serverID,
		serverKey:       serverKey,
		cpuThreshold:    cpuThreshold,
		memoryThreshold: memoryThreshold,
		diskThreshold:   diskThreshold,
	}
}

// thresholdChanged checks if a threshold value has changed from last sent
func thresholdChanged(current, lastSent *float64) bool {
	if current == nil && lastSent == nil {
		return false
	}
	if current == nil || lastSent == nil {
		return true
	}
	return *current != *lastSent
}

// copyThreshold creates a copy of a threshold pointer
func copyThreshold(t *float64) *float64 {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

// Notify sends the metrics to the configured webhook URL with HMAC-SHA256 signature.
func (n *WebhookNotifier) Notify(alerts []monitor.Alert) {
	if n.webhookURL == "" {
		return
	}

	// Detect if any threshold has changed since last send
	cpuChanged := thresholdChanged(n.cpuThreshold, n.lastSentCPU)
	memoryChanged := thresholdChanged(n.memoryThreshold, n.lastSentMemory)
	diskChanged := thresholdChanged(n.diskThreshold, n.lastSentDisk)
	hasChanges := !n.thresholdsSent || cpuChanged || memoryChanged || diskChanged

	payload := webhookPayload{
		ServerID: n.serverID,
		Data:     alerts,
	}

	// Only include thresholds if they've changed (or first time sending)
	if hasChanges {
		if cpuChanged || !n.thresholdsSent {
			payload.CPUThreshold = n.cpuThreshold
		}
		if memoryChanged || !n.thresholdsSent {
			payload.MemoryThreshold = n.memoryThreshold
		}
		if diskChanged || !n.thresholdsSent {
			payload.DiskThreshold = n.diskThreshold
		}
		log.Printf("📊 Threshold change detected, including in payload: cpu=%v, memory=%v, disk=%v",
			n.cpuThreshold, n.memoryThreshold, n.diskThreshold)
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
		// Update last sent thresholds after successful send
		if hasChanges {
			n.lastSentCPU = copyThreshold(n.cpuThreshold)
			n.lastSentMemory = copyThreshold(n.memoryThreshold)
			n.lastSentDisk = copyThreshold(n.diskThreshold)
			n.thresholdsSent = true
		}
	} else {
		log.Printf(" Webhook notification failed with status: %s", resp.Status)
	}
}

// UpdateThresholds updates the threshold values (called on config reload)
func (n *WebhookNotifier) UpdateThresholds(cpuThreshold, memoryThreshold, diskThreshold *float64) {
	n.cpuThreshold = cpuThreshold
	n.memoryThreshold = memoryThreshold
	n.diskThreshold = diskThreshold
}
