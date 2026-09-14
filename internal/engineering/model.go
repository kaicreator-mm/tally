package engineering

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ID string

func (id ID) Validate(kind string) error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("engineering: %s id is required", kind)
	}
	return nil
}

type Provenance struct {
	Actor    string    `json:"actor"`
	Source   string    `json:"source"`
	Revision string    `json:"revision,omitempty"`
	At       time.Time `json:"at"`
}

type ArtifactKind string

const (
	ArtifactRequirements ArtifactKind = "requirements"
	ArtifactProjectPlan  ArtifactKind = "project_plan"
	ArtifactArchitecture ArtifactKind = "architecture"
	ArtifactTaskDAG      ArtifactKind = "task_dag"
	ArtifactTaskContract ArtifactKind = "task_contract"
	ArtifactVerification ArtifactKind = "verification"
	ArtifactReview       ArtifactKind = "review"
	ArtifactAcceptance   ArtifactKind = "acceptance"
)

type ArtifactRef struct {
	ID       ID           `json:"id"`
	Kind     ArtifactKind `json:"kind"`
	URI      string       `json:"uri"`
	Revision string       `json:"revision,omitempty"`
	Digest   string       `json:"digest,omitempty"`
}

func (r ArtifactRef) Validate() error {
	if err := r.ID.Validate("artifact"); err != nil {
		return err
	}
	if r.Kind == "" {
		return errors.New("engineering: artifact kind is required")
	}
	if strings.TrimSpace(r.URI) == "" {
		return errors.New("engineering: artifact URI is required")
	}
	return nil
}

type EvidenceStatus string

const (
	EvidencePass       EvidenceStatus = "pass"
	EvidenceFail       EvidenceStatus = "fail"
	EvidenceNotRun     EvidenceStatus = "not_run"
	EvidenceEnvBlocked EvidenceStatus = "env_blocked"
)

type EvidenceRef struct {
	ID       ID             `json:"id"`
	Kind     string         `json:"kind"`
	URI      string         `json:"uri"`
	Revision string         `json:"revision,omitempty"`
	Status   EvidenceStatus `json:"status"`
}

func (r EvidenceRef) Validate() error {
	if err := r.ID.Validate("evidence"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Kind) == "" || strings.TrimSpace(r.URI) == "" {
		return errors.New("engineering: evidence kind and URI are required")
	}
	switch r.Status {
	case EvidencePass, EvidenceFail, EvidenceNotRun, EvidenceEnvBlocked:
		return nil
	}
	return fmt.Errorf("engineering: invalid evidence status %q", r.Status)
}

type ActionRecord struct {
	ID         ID         `json:"id"`
	NodeID     ID         `json:"node_id,omitempty"`
	Kind       string     `json:"kind"`
	Actor      string     `json:"actor"`
	At         time.Time  `json:"at"`
	Provenance Provenance `json:"provenance"`
}

func (r ActionRecord) Validate() error {
	if err := r.ID.Validate("action record"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Kind) == "" || strings.TrimSpace(r.Actor) == "" || r.At.IsZero() {
		return errors.New("engineering: action kind, actor and time are required")
	}
	return nil
}

var ErrDuplicateActionRecord = errors.New("engineering: duplicate action record")

type AuditLog struct {
	Records []ActionRecord `json:"records"`
}

func (l AuditLog) Append(r ActionRecord) (AuditLog, error) {
	if err := r.Validate(); err != nil {
		return l, err
	}
	for _, x := range l.Records {
		if x.ID == r.ID {
			return l, ErrDuplicateActionRecord
		}
	}
	out := AuditLog{Records: append([]ActionRecord(nil), l.Records...)}
	out.Records = append(out.Records, r)
	return out, nil
}

type Requirement struct {
	ID                 ID       `json:"id"`
	Statement          string   `json:"statement"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Scope              []string `json:"scope,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
	Risks              []string `json:"risks,omitempty"`
	Assumptions        []string `json:"assumptions,omitempty"`
}

func (r Requirement) validate() error {
	if err := r.ID.Validate("requirement"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Statement) == "" || len(r.AcceptanceCriteria) == 0 {
		return fmt.Errorf("engineering: requirement %s requires statement and acceptance criteria", r.ID)
	}
	for _, x := range r.AcceptanceCriteria {
		if strings.TrimSpace(x) == "" {
			return fmt.Errorf("engineering: requirement %s has empty acceptance criterion", r.ID)
		}
	}
	return nil
}

type RequirementsBaseline struct {
	ID           ID            `json:"id"`
	Revision     string        `json:"revision"`
	Requirements []Requirement `json:"requirements"`
	Artifact     ArtifactRef   `json:"artifact"`
	Provenance   Provenance    `json:"provenance"`
}

func NewRequirementsBaseline(id ID, revision string, reqs []Requirement, a ArtifactRef, p Provenance) (RequirementsBaseline, error) {
	if err := id.Validate("requirements baseline"); err != nil {
		return RequirementsBaseline{}, err
	}
	if strings.TrimSpace(revision) == "" || len(reqs) == 0 {
		return RequirementsBaseline{}, errors.New("engineering: requirements revision and at least one requirement are required")
	}
	if err := a.Validate(); err != nil {
		return RequirementsBaseline{}, err
	}
	seen := map[ID]bool{}
	cp := make([]Requirement, len(reqs))
	for i, r := range reqs {
		if err := r.validate(); err != nil {
			return RequirementsBaseline{}, err
		}
		if seen[r.ID] {
			return RequirementsBaseline{}, fmt.Errorf("engineering: duplicate requirement %s", r.ID)
		}
		seen[r.ID] = true
		cp[i] = r
		cp[i].AcceptanceCriteria = append([]string(nil), r.AcceptanceCriteria...)
		cp[i].Scope = append([]string(nil), r.Scope...)
		cp[i].Constraints = append([]string(nil), r.Constraints...)
		cp[i].Risks = append([]string(nil), r.Risks...)
		cp[i].Assumptions = append([]string(nil), r.Assumptions...)
	}
	return RequirementsBaseline{ID: id, Revision: revision, Requirements: cp, Artifact: a, Provenance: p}, nil
}

type PlanningRevision struct {
	ID             ID          `json:"id"`
	Number         uint64      `json:"number"`
	ParentRevision ID          `json:"parent_revision,omitempty"`
	Strategy       []string    `json:"strategy"`
	Artifact       ArtifactRef `json:"artifact"`
	Provenance     Provenance  `json:"provenance"`
}

func NewPlanningRevision(id ID, n uint64, parent ID, strategy []string, a ArtifactRef, p Provenance) (PlanningRevision, error) {
	if err := id.Validate("planning revision"); err != nil {
		return PlanningRevision{}, err
	}
	if n == 0 || len(strategy) == 0 {
		return PlanningRevision{}, errors.New("engineering: planning revision number and strategy are required")
	}
	for _, x := range strategy {
		if strings.TrimSpace(x) == "" {
			return PlanningRevision{}, errors.New("engineering: empty planning strategy step")
		}
	}
	if err := a.Validate(); err != nil {
		return PlanningRevision{}, err
	}
	return PlanningRevision{ID: id, Number: n, ParentRevision: parent, Strategy: append([]string(nil), strategy...), Artifact: a, Provenance: p}, nil
}

type ProjectPlan struct {
	ID               ID               `json:"id"`
	AcceptedRevision PlanningRevision `json:"accepted_revision"`
}

type ArchitectureKind string

const (
	ArchitectureNote   ArchitectureKind = "note"
	ArchitectureADR    ArchitectureKind = "adr"
	ArchitectureDesign ArchitectureKind = "design"
)

type ArchitectureConstraint struct {
	ID        ID     `json:"id"`
	Statement string `json:"statement"`
}
type ArchitectureRevision struct {
	ID          ID                       `json:"id"`
	Kind        ArchitectureKind         `json:"kind"`
	Summary     string                   `json:"summary"`
	Constraints []ArchitectureConstraint `json:"constraints,omitempty"`
	Artifact    ArtifactRef              `json:"artifact"`
	Provenance  Provenance               `json:"provenance"`
}

func NewArchitectureRevision(id ID, k ArchitectureKind, summary string, cs []ArchitectureConstraint, a ArtifactRef, p Provenance) (ArchitectureRevision, error) {
	if err := id.Validate("architecture revision"); err != nil {
		return ArchitectureRevision{}, err
	}
	if k != ArchitectureNote && k != ArchitectureADR && k != ArchitectureDesign {
		return ArchitectureRevision{}, fmt.Errorf("engineering: invalid architecture kind %q", k)
	}
	if strings.TrimSpace(summary) == "" {
		return ArchitectureRevision{}, errors.New("engineering: architecture summary is required")
	}
	if err := a.Validate(); err != nil {
		return ArchitectureRevision{}, err
	}
	seen := map[ID]bool{}
	cp := append([]ArchitectureConstraint(nil), cs...)
	for _, c := range cp {
		if err := c.ID.Validate("architecture constraint"); err != nil {
			return ArchitectureRevision{}, err
		}
		if strings.TrimSpace(c.Statement) == "" || seen[c.ID] {
			return ArchitectureRevision{}, fmt.Errorf("engineering: invalid/duplicate architecture constraint %s", c.ID)
		}
		seen[c.ID] = true
	}
	return ArchitectureRevision{ID: id, Kind: k, Summary: summary, Constraints: cp, Artifact: a, Provenance: p}, nil
}
