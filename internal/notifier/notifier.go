package notifier

import (
	"log"
	"sync"
	
	"telemetry/internal/monitor"
)

type Notifier interface {
	Notify(alerts []monitor.Alert) error
}

type Registry struct {
	mu        sync.RWMutex
	notifiers map[string]Notifier
}

func NewRegistry() *Registry {
	return &Registry{
		notifiers: make(map[string]Notifier),
	}
}

func (r *Registry) Register(name string, n Notifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifiers[name] = n
}

func (r *Registry) NotifyAll(alerts []monitor.Alert) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for name, notifier := range r.notifiers {
		if err := notifier.Notify(alerts); err != nil {
			// CRITICAL: Log errors instead of silently swallowing them
			log.Printf("⚠️  Notifier '%s' failed to send %d alert(s): %v", name, len(alerts), err)
		}
	}
}
