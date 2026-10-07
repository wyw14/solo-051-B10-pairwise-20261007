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

func (c *Controller) Feedback() *Feedback { return c.feedback }

// Open issues the command and publishes "open" feedback only after every stage
// commits: device action, device-fact verification, audit record, then the
// caller's confirmation checks. Any failure leaves the feedback pending or
// unknown so recovery must re-issue and re-confirm the command.
func (c *Controller) Open(ctx context.Context, confirms ...func() error) error {
	if err := ctx.Err(); err != nil {
		c.feedback.Unknown()
		return err
	}
	c.feedback.Pending(StateOpen)
	if err := c.driver.Open(); err != nil {
		c.feedback.Unknown()
		return fmt.Errorf("open shutter: %w", err)
	}
	if !c.driver.Opened() {
		c.feedback.Unknown()
		return fmt.Errorf("open shutter: device fact unconfirmed")
	}
	if err := c.record(StateOpen); err != nil {
		c.feedback.Unknown()
		return fmt.Errorf("open shutter: record audit: %w", err)
	}
	for _, confirm := range confirms {
		if confirm == nil {
			continue
		}
		if err := confirm(); err != nil {
			c.feedback.Unknown()
			return fmt.Errorf("open shutter: confirm: %w", err)
		}
	}
	c.feedback.Update(StateOpen)
	return nil
}

// Close drives the shutter closed and publishes "closed" feedback only after
// every stage commits: device action, device-fact verification, audit record,
// then the caller's confirmation checks (durable journal write, downstream
// safety checks). A failure at any stage leaves the feedback pending or
// unknown, never "closed", so recovery cannot skip re-confirmation.
func (c *Controller) Close(ctx context.Context, confirms ...func() error) error {
	if err := ctx.Err(); err != nil {
		c.feedback.Unknown()
		return err
	}
	c.feedback.Pending(StateClosed)
	if err := CloseSafely(ctx, c.driver); err != nil {
		c.feedback.Unknown()
		return fmt.Errorf("close shutter: %w", err)
	}
	if c.driver.Opened() {
		c.feedback.Unknown()
		return fmt.Errorf("close shutter: device fact unconfirmed")
	}
	if err := c.record(StateClosed); err != nil {
		c.feedback.Unknown()
		return fmt.Errorf("close shutter: record audit: %w", err)
	}
	for _, confirm := range confirms {
		if confirm == nil {
			continue
		}
		if err := confirm(); err != nil {
			c.feedback.Unknown()
			return fmt.Errorf("close shutter: confirm: %w", err)
		}
	}
	c.feedback.Update(StateClosed)
	return nil
}

func (c *Controller) record(value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commands = append(c.commands, fmt.Sprintf("%s:%s", time.Now().UTC().Format(time.RFC3339Nano), value))
	return nil
}

func (c *Controller) Commands() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.commands...)
}
