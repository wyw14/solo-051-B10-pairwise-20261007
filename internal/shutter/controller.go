package shutter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Controller struct {
	mu       sync.Mutex
	driver   *Driver
	feedback *Feedback
	commands []string
}

func NewController(driver *Driver, feedback *Feedback) *Controller {
	return &Controller{driver: driver, feedback: feedback, commands: make([]string, 0)}
}

func (c *Controller) Open(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := c.driver.Open(); err != nil {
		c.feedback.Update(StateUnknown)
		return fmt.Errorf("open shutter: %w", err)
	}
	return c.commit(StateOpen, c.driver.Opened)
}

func (c *Controller) Close(ctx context.Context) error {
	if err := CloseSafely(ctx, c.driver); err != nil {
		c.feedback.Update(StateUnknown)
		return fmt.Errorf("close shutter: %w", err)
	}
	return c.commit(StateClosed, func() bool { return !c.driver.Opened() })
}

// commit publishes a commanded state in a strict order: the audit trail is
// appended first, the device fact is re-verified as a security check, and
// only then is the feedback updated. Any failure leaves the feedback
// unknown so recovery flows re-confirm the shutter instead of trusting a
// stale state.
func (c *Controller) commit(state string, confirm func() bool) error {
	if err := c.record(state); err != nil {
		c.feedback.Update(StateUnknown)
		return fmt.Errorf("audit shutter %q: %w", state, err)
	}
	if !confirm() {
		c.feedback.Update(StateUnknown)
		return fmt.Errorf("shutter %q not confirmed by driver", state)
	}
	c.feedback.Update(state)
	return nil
}

func (c *Controller) record(value string) error {
	if value != StateOpen && value != StateClosed {
		return fmt.Errorf("invalid shutter state %q", value)
	}
	c.mu.Lock()
	c.commands = append(c.commands, fmt.Sprintf("%s:%s", time.Now().UTC().Format(time.RFC3339Nano), value))
	c.mu.Unlock()
	return nil
}

func (c *Controller) Commands() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.commands...)
}
