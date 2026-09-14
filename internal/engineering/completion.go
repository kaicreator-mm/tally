package engineering

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ParticipationMode string

const (
	ModeAuto              ParticipationMode = "AUTO"
	ModeSupervised        ParticipationMode = "SUPERVISED"
	ModeAuthorityRequired ParticipationMode = "AUTHORITY_REQUIRED"
)

type AuthorityStatus string

const (
	AuthorityPending    AuthorityStatus = "pending"
	AuthorityAuthorized AuthorityStatus = "authorized"
	AuthorityRejected   AuthorityStatus = "rejected"
)

type AuthorityRequirement struct {
	ID     ID                `json:"id"`
	Mode   ParticipationMode `json:"mode"`
	Scope  string            `json:"scope"`
	Reason string            `json:"reason,omitempty"`
}

func (r AuthorityRequirement) Validate() error {
	if err := r.ID.Validate("authority requirement"); err != nil {
		return err
	}
	if r.Mode != ModeAuto && r.Mode != ModeSupervised && r.Mode != ModeAuthorityRequired {
		return fmt.Errorf("engineering: invalid participation mode %q", r.Mode)
	}
	if strings.TrimSpace(r.Scope) == "" {
		return errors.New("engineering: authority scope is required")
	}
	return nil
}

type AuthorityDecision struct {
	RequirementID ID              `json:"requirement_id"`
	Status        AuthorityStatus `json:"status"`
	Actor         string          `json:"actor"`
	At            time.Time       `json:"at"`
	Provenance    Provenance      `json:"provenance"`
}

func (d AuthorityDecision) Validate() error {
	if err := d.RequirementID.Validate("authority requirement"); err != nil {
		return err
	}
	if d.Status != AuthorityPending && d.Status != AuthorityAuthorized && d.Status != AuthorityRejected {
		return fmt.Errorf("engineering: invalid authority status %q", d.Status)
	}
	if strings.TrimSpace(d.Actor) == "" || d.At.IsZero() {
		return errors.New("engineering: authority actor and decision time are required")
	}
	return nil
}

type NodeState string

const (
	NodePending     NodeState = "pending"
	NodeReady       NodeState = "ready"
	NodeRunning     NodeState = "running"
	NodeBlocked     NodeState = "blocked"
	NodeInterrupted NodeState = "interrupted"
	NodeAccepted    NodeState = "accepted"
	NodeComplete    NodeState = "complete"
)

type EngineeringNode struct {
	ID         ID                 `json:"id"`
	Kind       string             `json:"kind"`
	State      NodeState          `json:"state"`
	Completion CompletionContract `json:"completion"`
}
type CheckStatus string

const (
	CheckPass       CheckStatus = "pass"
	CheckFail       CheckStatus = "fail"
	CheckNotRun     CheckStatus = "not_run"
	CheckEnvBlocked CheckStatus = "env_blocked"
)

type CheckResult struct {
	ID     ID          `json:"id"`
	Status CheckStatus `json:"status"`
	Basis  string      `json:"basis,omitempty"`
}
type GitOperationRecord struct {
	ID        ID     `json:"id"`
	Operation string `json:"operation"`
	Revision  string `json:"revision,omitempty"`
}
type CompletionContract struct {
	RequiredState         NodeState `json:"required_state"`
	RequiredArtifacts     []ID      `json:"required_artifacts,omitempty"`
	RequiredEvidence      []ID      `json:"required_evidence,omitempty"`
	RequiredChecks        []ID      `json:"required_checks,omitempty"`
	RequiredRecords       []ID      `json:"required_records,omitempty"`
	RequiredAuthority     []ID      `json:"required_authority,omitempty"`
	RequiredGitOperations []ID      `json:"required_git_operations,omitempty"`
}
type CompletionFacts struct {
	State         NodeState            `json:"state"`
	Artifacts     []ArtifactRef        `json:"artifacts,omitempty"`
	Evidence      []EvidenceRef        `json:"evidence,omitempty"`
	Checks        []CheckResult        `json:"checks,omitempty"`
	Records       []ActionRecord       `json:"records,omitempty"`
	Authority     []AuthorityDecision  `json:"authority,omitempty"`
	GitOperations []GitOperationRecord `json:"git_operations,omitempty"`
}
type MissingKind string

const (
	MissingState        MissingKind = "state"
	MissingArtifact     MissingKind = "artifact"
	MissingEvidence     MissingKind = "evidence"
	MissingCheck        MissingKind = "check"
	MissingRecord       MissingKind = "record"
	MissingAuthority    MissingKind = "authority"
	MissingGitOperation MissingKind = "git_operation"
)

type MissingRequirement struct {
	Kind   MissingKind `json:"kind"`
	ID     ID          `json:"id,omitempty"`
	Reason string      `json:"reason"`
}
type CompletionResult struct {
	Satisfied bool                 `json:"satisfied"`
	Missing   []MissingRequirement `json:"missing,omitempty"`
}

func (c CompletionContract) Evaluate(f CompletionFacts) CompletionResult {
	var m []MissingRequirement
	if c.RequiredState != "" && f.State != c.RequiredState {
		m = append(m, MissingRequirement{Kind: MissingState, Reason: fmt.Sprintf("requires state %s, got %s", c.RequiredState, f.State)})
	}
	arts := map[ID]bool{}
	for _, x := range f.Artifacts {
		arts[x.ID] = true
	}
	ev := map[ID]EvidenceStatus{}
	for _, x := range f.Evidence {
		ev[x.ID] = x.Status
	}
	checks := map[ID]CheckStatus{}
	for _, x := range f.Checks {
		checks[x.ID] = x.Status
	}
	records := map[ID]bool{}
	for _, x := range f.Records {
		records[x.ID] = true
	}
	auth := map[ID]AuthorityStatus{}
	for _, x := range f.Authority {
		auth[x.RequirementID] = x.Status
	}
	git := map[ID]bool{}
	for _, x := range f.GitOperations {
		git[x.ID] = true
	}
	for _, id := range uniqueSorted(c.RequiredArtifacts) {
		if !arts[id] {
			m = append(m, MissingRequirement{Kind: MissingArtifact, ID: id, Reason: "required artifact missing"})
		}
	}
	for _, id := range uniqueSorted(c.RequiredEvidence) {
		if ev[id] != EvidencePass {
			m = append(m, MissingRequirement{Kind: MissingEvidence, ID: id, Reason: "required PASS evidence missing"})
		}
	}
	for _, id := range uniqueSorted(c.RequiredChecks) {
		if checks[id] != CheckPass {
			m = append(m, MissingRequirement{Kind: MissingCheck, ID: id, Reason: "required PASS check missing"})
		}
	}
	for _, id := range uniqueSorted(c.RequiredRecords) {
		if !records[id] {
			m = append(m, MissingRequirement{Kind: MissingRecord, ID: id, Reason: "required action record missing"})
		}
	}
	for _, id := range uniqueSorted(c.RequiredAuthority) {
		if auth[id] != AuthorityAuthorized {
			m = append(m, MissingRequirement{Kind: MissingAuthority, ID: id, Reason: "required authorized authority missing"})
		}
	}
	for _, id := range uniqueSorted(c.RequiredGitOperations) {
		if !git[id] {
			m = append(m, MissingRequirement{Kind: MissingGitOperation, ID: id, Reason: "required git operation missing"})
		}
	}
	return CompletionResult{Satisfied: len(m) == 0, Missing: m}
}
func uniqueSorted(in []ID) []ID {
	seen := map[ID]bool{}
	out := make([]ID, 0, len(in))
	for _, id := range in {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
