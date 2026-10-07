package shutter

import (
	"sync"
	"time"
)

// Published shutter states. StateUnknown means the device fact could not
// be confirmed and the shutter must be re-confirmed before it is trusted.
const (
	StateOpen    = "open"
	StateClosed  = "closed"
	StateUnknown = "unknown"
)

type Feedback struct {
	mu      sync.RWMutex
	state   string
	changed time.Time
}

func NewFeedback() *Feedback { return &Feedback{state: StateClosed, changed: time.Now().UTC()} }
func (f *Feedback) Update(state string) {
	f.mu.Lock()
	f.state = state
	f.changed = time.Now().UTC()
	f.mu.Unlock()
}
func (f *Feedback) State() (string, time.Time) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state, f.changed
}
