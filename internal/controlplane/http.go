package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

const maxAPIRequestBytes = 2 << 20

type ControlPlane interface {
	Snapshot(context.Context, engineering.ID) (ProjectSnapshot, error)
	AcceptRequirements(context.Context, engineering.ID, engineering.RequirementsBaseline) (ProjectSnapshot, error)
	AcceptPlan(context.Context, engineering.ID, engineering.ProjectPlan) (ProjectSnapshot, error)
	AcceptArchitecture(context.Context, engineering.ID, engineering.ArchitectureRevision) (ProjectSnapshot, error)
	ActivateDAG(context.Context, engineering.ID, engineering.TaskDAGProposal, map[engineering.ID]engineering.CompletionContract) (ProjectSnapshot, error)
	RecordAuthority(context.Context, engineering.ID, engineering.ID, engineering.AuthorityDecision) (ProjectSnapshot, error)
	RestartTask(context.Context, engineering.ID, engineering.ID) (ProjectSnapshot, error)
	Dispatch(context.Context, engineering.ID, taskexecution.TaskExecutionRequest) (DispatchResult, error)
}

type dagActivationRequest struct {
	Proposal  engineering.TaskDAGProposal                        `json:"proposal"`
	Contracts map[engineering.ID]engineering.CompletionContract `json:"completion_contracts"`
}

type HTTPHandler struct {
	service ControlPlane
	mux     *http.ServeMux
}

func NewHTTPHandler(service ControlPlane) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("controlplane: HTTP service is required")
	}
	h := &HTTPHandler{service: service, mux: http.NewServeMux()}
	// Thin browser console. The outer web router mounts this under the same
	// authenticated + CSRF-protected group as the JSON API.
	h.mux.HandleFunc("GET /v15/control-plane", h.console)

	h.mux.HandleFunc("GET /api/v15/projects/{project}/control-plane", h.snapshot)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/requirements", h.requirements)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/plan", h.plan)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/architecture", h.architecture)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/task-dag", h.taskDAG)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/tasks/{task}/authority", h.authority)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/tasks/{task}/restart", h.restart)
	h.mux.HandleFunc("POST /api/v15/projects/{project}/tasks/{task}/dispatch", h.dispatch)
	return h, nil
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) snapshot(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Snapshot(r.Context(), engineering.ID(r.PathValue("project")))
	h.respond(w, result, err)
}

func (h *HTTPHandler) requirements(w http.ResponseWriter, r *http.Request) {
	var input engineering.RequirementsBaseline
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.AcceptRequirements(r.Context(), engineering.ID(r.PathValue("project")), input)
	h.respond(w, result, err)
}

func (h *HTTPHandler) plan(w http.ResponseWriter, r *http.Request) {
	var input engineering.ProjectPlan
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.AcceptPlan(r.Context(), engineering.ID(r.PathValue("project")), input)
	h.respond(w, result, err)
}

func (h *HTTPHandler) architecture(w http.ResponseWriter, r *http.Request) {
	var input engineering.ArchitectureRevision
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.AcceptArchitecture(r.Context(), engineering.ID(r.PathValue("project")), input)
	h.respond(w, result, err)
}

func (h *HTTPHandler) taskDAG(w http.ResponseWriter, r *http.Request) {
	var input dagActivationRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.ActivateDAG(r.Context(), engineering.ID(r.PathValue("project")), input.Proposal, input.Contracts)
	h.respond(w, result, err)
}

func (h *HTTPHandler) authority(w http.ResponseWriter, r *http.Request) {
	var input engineering.AuthorityDecision
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.RecordAuthority(r.Context(), engineering.ID(r.PathValue("project")), engineering.ID(r.PathValue("task")), input)
	h.respond(w, result, err)
}

func (h *HTTPHandler) restart(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.RestartTask(r.Context(), engineering.ID(r.PathValue("project")), engineering.ID(r.PathValue("task")))
	h.respond(w, result, err)
}

func (h *HTTPHandler) dispatch(w http.ResponseWriter, r *http.Request) {
	var input taskexecution.TaskExecutionRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	taskID := engineering.ID(r.PathValue("task"))
	if input.TaskID == "" {
		input.TaskID = taskID
	} else if input.TaskID != taskID {
		writeAPIError(w, http.StatusBadRequest, "task id in body does not match path")
		return
	}
	result, err := h.service.Dispatch(r.Context(), engineering.ID(r.PathValue("project")), input)
	h.respond(w, result, err)
}

func (h *HTTPHandler) respond(w http.ResponseWriter, result any, err error) {
	if err != nil {
		status := http.StatusUnprocessableEntity
		switch {
		case errors.Is(err, ErrProjectNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrStageOrder), errors.Is(err, ErrImmutableStage), errors.Is(err, ErrDAGNotActive), errors.Is(err, ErrTaskNotReady):
			status = http.StatusConflict
		case errors.Is(err, ErrWorkflowUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, context.Canceled):
			status = http.StatusRequestTimeout
		}
		writeAPIError(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, maxAPIRequestBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "request body must contain exactly one JSON value")
		return false
	}
	return true
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
