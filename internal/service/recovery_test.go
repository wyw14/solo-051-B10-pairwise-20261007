package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wyw14/cry-161/internal/journal"
	"github.com/wyw14/cry-161/internal/model"
)

func recoverFixture(t *testing.T, journalDir string) (*Service, *Runtime, model.ID) {
	t.Helper()
	svc := New()
	runtime := NewRuntime()
	runID := model.NewID()
	svc.Runs.Add(&model.Run{
		ID:        runID,
		Batch:     "batch",
		RecipeID:  model.NewID(),
		State:     model.Coating,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	if err := runtime.Shutter.Open(context.Background()); err != nil {
		t.Fatalf("open shutter: %v", err)
	}
	svc.Chamber.SetShutter("open")
	if journalDir != "" {
		svc.Journal = journal.New(journalDir)
	}
	return svc, runtime, runID
}

func TestRecoveryKeepsUnknownWhenAuditWriteFails(t *testing.T) {
	// A regular file as the journal directory makes every Append fail.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if f, err := os.Create(blocker); err != nil {
		t.Fatal(err)
	} else {
		f.Close()
	}

	svc, runtime, runID := recoverFixture(t, blocker)
	report := svc.Recover(context.Background(), runtime, runID)

	if report.Recovered {
		t.Fatal("report must not be recovered when the shutter audit is not durable")
	}
	if state, _ := runtime.Shutter.Feedback().State(); state != "unknown" {
		t.Fatalf("shutter feedback = %q, want unknown", state)
	}
	if got := svc.Chamber.Snapshot().Shutter; got != "unknown" {
		t.Fatalf("chamber shutter fact = %q, want unknown", got)
	}
	value, ok := svc.Runs.Get(runID)
	if !ok {
		t.Fatal("run missing")
	}
	if value.State != model.Coating {
		t.Fatalf("run state = %q, must stay %q pending shutter confirmation", value.State, model.Coating)
	}
}

func TestRecoveryConfirmsClosedWhenAuditDurable(t *testing.T) {
	svc, runtime, runID := recoverFixture(t, t.TempDir())
	report := svc.Recover(context.Background(), runtime, runID)

	if !report.Recovered {
		t.Fatalf("report should be recovered: %+v", report.Actions)
	}
	if state, _ := runtime.Shutter.Feedback().State(); state != "closed" {
		t.Fatalf("shutter feedback = %q, want closed", state)
	}
	if got := svc.Chamber.Snapshot().Shutter; got != "closed" {
		t.Fatalf("chamber shutter fact = %q, want closed", got)
	}
	value, ok := svc.Runs.Get(runID)
	if !ok {
		t.Fatal("run missing")
	}
	if value.State != model.Cooling {
		t.Fatalf("run state = %q, want cooling", value.State)
	}
	events, err := svc.Journal.Events(0)
	if err != nil {
		t.Fatal(err)
	}
	sawShutterEvent := false
	for _, event := range events {
		if event.Kind == "shutter.closed" {
			sawShutterEvent = true
		}
	}
	if !sawShutterEvent {
		t.Fatal("durable shutter.closed audit event missing")
	}
}
