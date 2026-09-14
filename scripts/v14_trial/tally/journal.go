// journal.go owns the Tally-arm trial journal: the SAME closed event
// vocabulary and on-disk shape as the bare arm's journal
// (scripts/v14_trial/bare/journal.go), so P4-04 consumes one record shape for
// both arms. In arm B the loop records agent attempts and verifications; human
// interventions arise only from escalations (each escalation is recorded as
// the typed human-decision event it demands, with the loop's escalation
// reason as the note — PRD §31 vocabulary).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type eventType string

const (
	evClarification        eventType = "clarification"
	evManualRepair         eventType = "manual_repair"
	evManualRerun          eventType = "manual_rerun"
	evWaiver               eventType = "waiver"
	evTradeoffDecision     eventType = "tradeoff_decision"
	evForcedRetry          eventType = "forced_retry"
	evDirectCodeCorrection eventType = "direct_code_correction"
	evFailureCopyBack      eventType = "failure_copy_back"
	evPromptReissue        eventType = "prompt_reissue"
	evVerificationRestart  eventType = "verification_restart"
	evDiffViewed           eventType = "diff_viewed"
	evAgentAttempt         eventType = "agent_attempt"
	evVerification         eventType = "verification"
	evEscalation           eventType = "escalation"
)

// countsAsIntervention is the frozen M1/M3 classifier (PRD §31.1: viewing a
// diff is not a failure event), extended with the runner-recorded escalation
// event: an escalation hands the task to a human decision, so it is an
// intervention event of the trial (the e2e bfLedger semantics).
func (t eventType) countsAsIntervention() bool {
	switch t {
	case evDiffViewed:
		return false
	case evAgentAttempt, evVerification:
		return false
	default:
		return true
	}
}

type journalEvent struct {
	RecordedAt   string    `json:"recorded_at"`
	TaskID       string    `json:"task_id"`
	Attempt      int       `json:"attempt,omitempty"`
	Type         eventType `json:"type"`
	Intervention bool      `json:"intervention"`
	Note         string    `json:"note,omitempty"`
}

func journalPath(runsRoot, taskID string) string {
	return filepath.Join(runsRoot, taskID, "events.jsonl")
}

// appendEvent appends one event to the task journal (create-if-missing).
// NOTE: the bare runner's judge appends to this exact file shape when the
// arm-neutral judge is delegated with --runs-root pointing at the Tally arm's
// runs root; the two journals are byte-compatible by construction.
func appendEvent(runsRoot, taskID string, attempt int, typ eventType, note string) (journalEvent, error) {
	ev := journalEvent{
		RecordedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		TaskID:       taskID,
		Attempt:      attempt,
		Type:         typ,
		Intervention: typ.countsAsIntervention(),
		Note:         strings.TrimSpace(note),
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return ev, err
	}
	path := journalPath(runsRoot, taskID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ev, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return ev, err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return ev, err
	}
	return ev, nil
}

// readEvents returns the task journal in file order.
func readEvents(runsRoot, taskID string) ([]journalEvent, error) {
	b, err := os.ReadFile(journalPath(runsRoot, taskID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []journalEvent
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev journalEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return out, fmt.Errorf("journal %s: corrupt line: %w", taskID, err)
		}
		out = append(out, ev)
	}
	return out, nil
}
