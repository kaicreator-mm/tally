package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"tally/internal/engineering"
)

// NewSchedulerHTTPHandler exposes the bounded application-local scheduler as a
// separate control surface. It does not introduce a durable job queue: every
// request reloads the persisted Active TaskDAG and scheduling truth remains in
// the project snapshot.
func NewSchedulerHTTPHandler(scheduler *LocalScheduler) (http.Handler, error) {
	if scheduler == nil {
		return nil, errors.New("controlplane: scheduler HTTP service is required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v15/projects/{project}/scheduler/recover", func(w http.ResponseWriter, r *http.Request) {
		result, err := scheduler.RecoverStartup(r.Context(), engineering.ID(r.PathValue("project")))
		respondScheduler(w, result, err)
	})
	mux.HandleFunc("POST /api/v15/projects/{project}/scheduler/run-once", func(w http.ResponseWriter, r *http.Request) {
		result, err := scheduler.RunOnce(r.Context(), engineering.ID(r.PathValue("project")))
		respondScheduler(w, result, err)
	})
	mux.HandleFunc("POST /api/v15/projects/{project}/scheduler/run", func(w http.ResponseWriter, r *http.Request) {
		err := scheduler.Run(r.Context(), engineering.ID(r.PathValue("project")))
		respondScheduler(w, map[string]string{"status": "idle"}, err)
	})
	return mux, nil
}

func respondScheduler(w http.ResponseWriter, result any, err error) {
	if err != nil {
		status := http.StatusUnprocessableEntity
		switch {
		case errors.Is(err, ErrProjectNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrDAGNotActive), errors.Is(err, ErrTaskNotReady), errors.Is(err, ErrDispatchConflict):
			status = http.StatusConflict
		case errors.Is(err, ErrWorkflowUnavailable):
			// The AgentExecutor path is alive, but the optional legacy independent
			// verification/review capability is not composed in SQLite-first Core.
			status = http.StatusServiceUnavailable
		}
		writeAPIError(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
