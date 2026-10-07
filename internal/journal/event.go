package journal

import "github.com/wyw14/cry-161/internal/model"

func RunStarted(run model.Run) model.Event {
	return model.NewEvent("run.started", run.ID, map[string]any{"batch": run.Batch, "state": run.State})
}
func RunStateChanged(run model.Run, previous model.RunState) model.Event {
	return model.NewEvent("run.state", run.ID, map[string]any{"from": previous, "to": run.State})
}
func IncidentRaised(incident model.Incident) model.Event {
	return model.NewEvent("incident", incident.RunID, map[string]any{"severity": incident.Severity, "message": incident.Message})
}
func ShutterClosed(runID model.ID) model.Event {
	return model.NewEvent("shutter.closed", runID, map[string]any{"state": "closed"})
}
