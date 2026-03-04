package monitor

import (
	"sync"
)

type Alert struct {
	ServerName string
	Type       string
	Message    string
	Value      float64
	Severity   string
}

type Monitor interface {
	CheckMetrics() []Alert
	CheckAlerts(metrics []Alert) []Alert
}

type Registry struct {
	mu         sync.RWMutex
	monitors   map[string]Monitor
	names      []string
	serverName string
}

func NewRegistry(serverName string) *Registry {
	return &Registry{
		monitors:   make(map[string]Monitor),
		names:      []string{},
		serverName: serverName,
	}
}

func (r *Registry) Register(name string, m Monitor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitors[name] = m
	r.names = append(r.names, name)
}

func (r *Registry) CheckAlerts() []Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allAlerts []Alert
	for _, name := range r.names {
		monitor := r.monitors[name]
		metrics := monitor.CheckMetrics()
		for i := range metrics {
			metrics[i].ServerName = r.serverName
		}

		alerts := monitor.CheckAlerts(metrics)
		for i := range alerts {
			alerts[i].ServerName = r.serverName
		}
		allAlerts = append(allAlerts, alerts...)
	}
	return allAlerts
}

func (r *Registry) CheckMetrics() []Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allMetrics []Alert
	for _, name := range r.names {
		monitor := r.monitors[name]
		metrics := monitor.CheckMetrics()
		for i := range metrics {
			metrics[i].ServerName = r.serverName
		}
		allMetrics = append(allMetrics, metrics...)
	}
	return allMetrics
}
