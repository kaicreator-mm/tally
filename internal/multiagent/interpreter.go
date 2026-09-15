package multiagent

import (
	"context"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type DecisionKind string

const (
	DecisionPass        DecisionKind = "pass"
	DecisionRepair      DecisionKind = "repair"
	DecisionBlock       DecisionKind = "block"
	DecisionInterrupted DecisionKind = "interrupted"
)

type RoleContext struct {
	TaskID     engineering.ID `json:"task_id"`
	Role       Role           `json:"role"`
	Specialist string         `json:"specialist,omitempty"`
	RunID      string         `json:"role_run_id"`
}

type Decision struct {
	Kind     DecisionKind `json:"kind"`
	Reason   string       `json:"reason,omitempty"`
	Findings []Finding    `json:"findings,omitempty"`
}

type Interpreter interface {
	Interpret(context.Context, RoleContext, taskexecution.TaskExecutionResult) (Decision, error)
}

// EvidenceInterpreter is deliberately conservative: implementation/repair
// completion remains an Agent claim and only advances to review; review,
// verification and specialists require Tally evidence. A role cannot convert
// its own summary into a project fact.
type EvidenceInterpreter struct{}

func (EvidenceInterpreter) Interpret(_ context.Context, rc RoleContext, result taskexecution.TaskExecutionResult) (Decision, error) {
	switch result.Status {
	case taskexecution.ExecutionInterrupted:
		return Decision{Kind: DecisionInterrupted, Reason: "role execution interrupted"}, nil
	case taskexecution.ExecutionFailed:
		return Decision{Kind: DecisionBlock, Reason: "role execution failed"}, nil
	case taskexecution.ExecutionCompleted:
	default:
		return Decision{}, fmt.Errorf("multiagent: unknown execution status %q", result.Status)
	}

	if rc.Role == RoleImplementer || rc.Role == RoleRepairer {
		return Decision{Kind: DecisionPass}, nil
	}
	if len(result.EvidenceRefs) == 0 {
		return Decision{Kind: DecisionBlock, Reason: "role produced no Tally evidence; Agent claim alone cannot satisfy review/verification"}, nil
	}

	var findings []Finding
	blocked := false
	for i, evidence := range result.EvidenceRefs {
		switch evidence.Status {
		case engineering.EvidencePass:
		case engineering.EvidenceFail:
			findings = append(findings, Finding{
				ID:            engineering.ID(fmt.Sprintf("%s:finding:%d", rc.RunID, i+1)),
				Summary:       evidenceSummary(evidence),
				EvidenceRefs:  []engineering.EvidenceRef{evidence},
				ProducerRole:  rc.Role,
				ProducerRunID: rc.RunID,
			})
		case engineering.EvidenceNotRun, engineering.EvidenceEnvBlocked:
			blocked = true
		default:
			return Decision{}, fmt.Errorf("multiagent: invalid evidence status %q", evidence.Status)
		}
	}
	if len(findings) > 0 {
		return Decision{Kind: DecisionRepair, Reason: "role produced failing evidence", Findings: findings}, nil
	}
	if blocked {
		return Decision{Kind: DecisionBlock, Reason: "required role evidence was not runnable in the current environment"}, nil
	}
	return Decision{Kind: DecisionPass}, nil
}

func evidenceSummary(e engineering.EvidenceRef) string {
	kind := strings.TrimSpace(e.Kind)
	if kind == "" {
		kind = "evidence"
	}
	return fmt.Sprintf("%s %s failed", kind, e.ID)
}
