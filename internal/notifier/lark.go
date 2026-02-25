package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"telemetry/internal/monitor"
)

type LarkNotifier struct {
	webhookURL string
}

func NewLarkNotifier(webhookURL string) *LarkNotifier {
	if webhookURL != "" && !isValidWebhookURL(webhookURL) {
		log.Printf("⚠️ Invalid Lark webhook URL (must be HTTPS), disabling notifier")
		return &LarkNotifier{webhookURL: ""}
	}
	return &LarkNotifier{webhookURL: webhookURL}
}

func isValidWebhookURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("⚠️ Failed to parse webhook URL: %v", err)
		return false
	}
	
	if parsed.Scheme != "https" {
		log.Printf("⚠️ Webhook URL must use HTTPS scheme, got: %s", parsed.Scheme)
		return false
	}
	
	host := strings.ToLower(parsed.Host)
	allowedHosts := []string{
		"open.larksuite.com",
		"open.feishu.cn",
		"open.feishu.com",
	}
	
	for _, allowed := range allowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	
	log.Printf("⚠️ Webhook URL must be a Lark domain (open.larksuite.com, open.feishu.cn, open.feishu.com), got: %s", host)
	return false
}

func (n *LarkNotifier) Notify(alerts []monitor.Alert) error {
	if n.webhookURL == "" {
		log.Println("⚠️ Lark webhook URL not configured, skipping notification")
		return nil
	}

	message := n.buildMessage(alerts)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Post(n.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("⚠️ Failed to send Lark notification: %v", err)
		return fmt.Errorf("failed to send notification (check logs for details)")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ Lark API returned status %d", resp.StatusCode)
		return fmt.Errorf("lark API returned non-OK status (check logs for details)")
	}

	log.Println("✅ Sent notification to Lark")
	return nil
}

func (n *LarkNotifier) buildMessage(alerts []monitor.Alert) map[string]interface{} {
	var lines []string
	
	serverName := "Unknown Server"
	if len(alerts) > 0 && alerts[0].ServerName != "" {
		serverName = alerts[0].ServerName
	}
	
	lines = append(lines, fmt.Sprintf("🚨 **System Alert from %s**", serverName))
	lines = append(lines, "")

	for _, alert := range alerts {
		emoji := "⚠️"
		if alert.Severity == "critical" {
			emoji = "🔴"
		}
		lines = append(lines, fmt.Sprintf("%s %s", emoji, alert.Message))
	}

	return map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": strings.Join(lines, "\n"),
		},
	}
}
