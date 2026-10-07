package shutter

import (
	"sync"
	"time"
)

const (
	StateOpen    = "open"
	StateClosed  = "closed"
	StatePending = "pending"
	StateUnknown = "unknown"
)

type Feedback struct {
	mu      sync.RWMutex
	state   string
	changed time.Time
}

func NewFeedback() *Feedback { return &Feedback{state: StateClosed, changed: time.Now().UTC()} }

// Update commits a confirmed device fact. It must only be called after the
// device action and every dependent record have succeeded.
func (f *Feedback) Update(state string) {
	f.mu.Lock()
	f.state = state
	f.changed = time.Now().UTC()
	f.mu.Unlock()
}

// Pending marks a command that has been issued but not yet confirmed. Recovery
// flows must keep re-confirming a pending shutter rather than trusting it.
func (f *Feedback) Pending(action string) {
	f.mu.Lock()
	f.state = StatePending + ":" + action
	f.changed = time.Now().UTC()
	f.mu.Unlock()
}

// Unknown records that the physical shutter position cannot be trusted. It must
// never read as a safe terminal state.
func (f *Feedback) Unknown() {
	f.mu.Lock()
	f.state = StateUnknown
	f.changed = time.Now().UTC()
	f.mu.Unlock()
}

func (f *Feedback) State() (string, time.Time) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state, f.changed
}

// Confirmed reports whether the feedback holds the given terminal state.
func (f *Feedback) Confirmed(state string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state == state
}
