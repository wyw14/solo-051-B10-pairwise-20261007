package shutter

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCloseCommitsFeedbackOnlyAfterConfirmation(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}

	observedDuringConfirm := ""
	err := controller.Close(context.Background(), func() error {
		// Device fact and audit are already committed; terminal feedback is not.
		observedDuringConfirm, _ = feedback.State()
		return nil
	})
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !driver.isClosed() {
		t.Fatal("driver device fact not closed")
	}
	if observedDuringConfirm != "pending:closed" {
		t.Fatalf("feedback during confirm = %q, want pending:closed", observedDuringConfirm)
	}
	if state, _ := feedback.State(); state != StateClosed {
		t.Fatalf("feedback after close = %q, want closed", state)
	}
}

func TestCloseConfirmFailureKeepsUnknown(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}

	confirmErr := errors.New("journal unavailable")
	err := controller.Close(context.Background(), func() error { return confirmErr })
	if err == nil {
		t.Fatal("close must fail when confirmation fails")
	}
	if !errors.Is(err, confirmErr) {
		t.Fatalf("error %v does not wrap confirm failure", err)
	}
	// Device already moved, but feedback must not claim the safe state.
	if state, _ := feedback.State(); state != StateUnknown {
		t.Fatalf("feedback after failed confirm = %q, want unknown", state)
	}
	if feedback.Confirmed(StateClosed) {
		t.Fatal("feedback must not report confirmed closed")
	}
}

func TestCloseDriverFailureKeepsUnknown(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}
	driver.SetAvailable(false)

	if err := controller.Close(context.Background()); err == nil {
		t.Fatal("close must fail when driver unavailable")
	}
	if state, _ := feedback.State(); state != StateUnknown {
		t.Fatalf("feedback after driver failure = %q, want unknown", state)
	}
	if len(controller.Commands()) != 1 {
		t.Fatalf("audit commands = %d, want only the open record (no closed audit on failure)", len(controller.Commands()))
	}
}

func TestCloseRetriedAfterFailureReconfirms(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}

	failOnce := true
	err := controller.Close(context.Background(), func() error {
		if failOnce {
			failOnce = false
			return errors.New("audit write failed")
		}
		return nil
	})
	if err == nil {
		t.Fatal("first close must fail")
	}
	if state, _ := feedback.State(); !strings.HasPrefix(state, StateUnknown) {
		t.Fatalf("feedback after first close = %q, want unknown", state)
	}

	// Recovery re-issues and re-confirms; it must not trust the prior attempt.
	if err := controller.Close(context.Background(), func() error { return nil }); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if state, _ := feedback.State(); state != StateClosed {
		t.Fatalf("feedback after retried close = %q, want closed", state)
	}
}

func TestCloseContextCancelledBeforeDriver(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := controller.Close(ctx); err == nil {
		t.Fatal("close must fail on cancelled context")
	}
	if state, _ := feedback.State(); state != StateUnknown {
		t.Fatalf("feedback after cancelled close = %q, want unknown", state)
	}
}

func (d *Driver) isClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.open
}
