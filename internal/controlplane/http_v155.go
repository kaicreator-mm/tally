package controlplane

import (
	"context"
	"errors"
	"net/http"

	"tally/internal/engineering"
	"tally/internal/replan"
)

// ReplanControlPlane is the additive v1.5.5 HTTP seam. Keeping it separate
// from the frozen v1.5.0 ControlPlane interface preserves compatibility for
// existing callers while ensuring HTTP invokes the exact same governed
// ActivateReplan service path as in-process callers.
type ReplanControlPlane interface {
	ActivateReplan(context.Context, engineering.ID, replan.Proposal, []engineering.AuthorityDecision) (ProjectSnapshot, error)
}

type replanActivationRequest struct {
	Proposal  replan.Proposal                 `json:"proposal"`
	Authority []engineering.AuthorityDecision `json:"authority,omitempty"`
}

func (h *HTTPHandler) activateReplan(w http.ResponseWriter, r *http.Request) {
	service, ok := h.service.(ReplanControlPlane)
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "controlplane: re-plan service is unavailable")
		return
	}
	var input replanActivationRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := service.ActivateReplan(
		r.Context(),
		engineering.ID(r.PathValue("project")),
		input.Proposal,
		input.Authority,
	)
	if err != nil {
		status := http.StatusUnprocessableEntity
		switch {
		case errors.Is(err, replan.ErrStaleParent), errors.Is(err, replan.ErrCycleBudget):
			status = http.StatusConflict
		case errors.Is(err, replan.ErrAuthorityRequired), errors.Is(err, replan.ErrSelfAuthorization):
			status = http.StatusForbidden
		}
		writeAPIError(w, status, err.Error())
		return
	}
	h.respond(w, result, nil)
}
