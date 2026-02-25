package monitor

import (
	"fmt"
	"net/http"
	"time"
)

type HealthMonitor struct {
	name string
	url  string
}

func NewHealthMonitor(name, url string) *HealthMonitor {
	return &HealthMonitor{
		name: name,
		url:  url,
	}
}

func (m *HealthMonitor) Check() []Alert {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(m.url)
	if err != nil {
		return []Alert{
			{
				Type:     "health",
				Message:  fmt.Sprintf("🔴 Health check failed for %s (%s): %v", m.name, m.url, err),
				Severity: "critical",
			},
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Alert{
			{
				Type:     "health",
				Message:  fmt.Sprintf("🔴 Health check for %s (%s) returned status code %d (expected 200)", m.name, m.url, resp.StatusCode),
				Severity: "critical",
			},
		}
	}

	return nil
}
