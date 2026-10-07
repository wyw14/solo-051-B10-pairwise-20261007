package shutter

import (
	"context"
	"strings"
	"testing"
)

func TestCloseCommitsFeedbackAfterAudit(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := controller.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
	state, _ := feedback.State()
	if state != StateClosed {
		t.Fatalf("feedback = %q, want %q", state, StateClosed)
	}
	commands := controller.Commands()
	if len(commands) != 2 || !strings.HasSuffix(commands[1], ":"+StateClosed) {
		t.Fatalf("audit trail = %v, want open then closed entries", commands)
	}
}

func TestCloseFailureLeavesFeedbackUnknown(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}
	driver.SetAvailable(false)
	if err := controller.Close(context.Background()); err == nil {
		t.Fatal("close should fail when the driver is unavailable")
	}
	state, _ := feedback.State()
	if state != StateUnknown {
		t.Fatalf("feedback = %q, want %q after failed close", state, StateUnknown)
	}
	for _, entry := range controller.Commands() {
		if strings.HasSuffix(entry, ":"+StateClosed) {
			t.Fatalf("audit trail %v must not record closed for a failed close", controller.Commands())
		}
	}
}

func TestOpenFailureLeavesFeedbackUnknown(t *testing.T) {
	driver := NewDriver()
	driver.SetAvailable(false)
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err == nil {
		t.Fatal("open should fail when the driver is unavailable")
	}
	state, _ := feedback.State()
	if state != StateUnknown {
		t.Fatalf("feedback = %q, want %q after failed open", state, StateUnknown)
	}
	if len(controller.Commands()) != 0 {
		t.Fatalf("audit trail = %v, want no entries for a failed open", controller.Commands())
	}
}

func TestCloseCancelledContextDoesNotClaimClosed(t *testing.T) {
	driver := NewDriver()
	feedback := NewFeedback()
	controller := NewController(driver, feedback)
	if err := controller.Open(context.Background()); err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := controller.Close(ctx); err == nil {
		t.Fatal("close should fail with a cancelled context")
	}
	state, _ := feedback.State()
	if state == StateClosed {
		t.Fatalf("feedback = %q, must not claim closed after cancelled close", state)
	}
}
