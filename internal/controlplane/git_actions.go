package controlplane

import (
	"context"
	"errors"
	"fmt"

	"tally/internal/engineering"
)

// GitActionResult is the durable control-plane result of recording one
// authority-gated Git action. Completion is populated when the action changes
// the task's CompletionContract evaluation.
type GitActionResult struct {
	Record     engineering.GitActionRecord   `json:"record"`
	Completion *engineering.CompletionResult `json:"completion,omitempty"`
	Snapshot   ProjectSnapshot               `json:"snapshot"`
}

// RecordGitAction persists a Git action produced by the Git Engineering
// Authority service. It never trusts command success alone: the record must be
// internally valid and already contain the authority decision that governed the
// operation. Successful operations are projected into the existing
// CompletionContract facts; blocked/failed actions remain audit facts only.
func (s *Service) RecordGitAction(ctx context.Context, projectID, taskID engineering.ID, record engineering.GitActionRecord) (GitActionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := record.Validate(); err != nil {
		return GitActionResult{}, err
	}
	if record.Trace.ProjectID != projectID {
		return GitActionResult{}, errors.New("controlplane: git action project does not match path")
	}
	if record.Trace.TaskNodeID != taskID {
		return GitActionResult{}, errors.New("controlplane: git action task does not match path")
	}

	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return GitActionResult{}, err
	}
	if snap.ActiveDAG == nil {
		return GitActionResult{}, ErrDAGNotActive
	}
	node, ok := findNode(*snap.ActiveDAG, taskID)
	if !ok {
		return GitActionResult{}, fmt.Errorf("controlplane: unknown task %s", taskID)
	}
	if _, ok := snap.Contracts[taskID]; !ok {
		return GitActionResult{}, fmt.Errorf("controlplane: task %s has no CompletionContract", taskID)
	}

	nextLog, err := snap.GitActions.Append(record)
	if err != nil {
		return GitActionResult{}, err
	}
	snap.GitActions = nextLog
	if err := s.record(&snap, "git."+string(record.Action)+"."+string(record.Status), taskID, "git_engineering_authority"); err != nil {
		return GitActionResult{}, err
	}

	result := GitActionResult{Record: record}
	if _, ok := record.CompletionOperation(); ok {
		if snap.CompletionFacts == nil {
			snap.CompletionFacts = map[engineering.ID]engineering.CompletionFacts{}
		}
		facts := snap.CompletionFacts[taskID]
		facts = engineering.AddGitActionToCompletion(facts, record)
		snap.CompletionFacts[taskID] = facts

		// A task accepted by verification but waiting for required Git actions is
		// represented by the existing TaskAccepted state. Only a successful,
		// authorized action that makes the frozen CompletionContract satisfied may
		// advance it to TaskDone.
		if node.State == engineering.TaskAccepted {
			completion := snap.Contracts[taskID].Evaluate(facts)
			result.Completion = &completion
			if completion.Satisfied {
				done, err := snap.ActiveDAG.Transition(taskID, engineering.TaskDone, "")
				if err != nil {
					return GitActionResult{}, err
				}
				snap.ActiveDAG = &done
				if err := s.record(&snap, "task.completed", taskID, "completion_contract:git_action"); err != nil {
					return GitActionResult{}, err
				}
			}
		}
	}

	if err := s.save(ctx, &snap); err != nil {
		return GitActionResult{}, err
	}
	result.Snapshot = snap
	return result, nil
}

func applyPersistedGitOperations(facts engineering.CompletionFacts, log engineering.GitActionLog, taskID engineering.ID) engineering.CompletionFacts {
	for _, record := range log.Records {
		if record.Trace.TaskNodeID != taskID {
			continue
		}
		facts = engineering.AddGitActionToCompletion(facts, record)
	}
	return facts
}

func onlyGitOperationsMissing(result engineering.CompletionResult) bool {
	if result.Satisfied || len(result.Missing) == 0 {
		return false
	}
	for _, missing := range result.Missing {
		if missing.Kind != engineering.MissingGitOperation {
			return false
		}
	}
	return true
}
