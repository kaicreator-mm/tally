// journal.go owns the human orchestration journal (P4-02 AC3, PRE_REGISTRATION
// §3 M1/M3): in the bare arm the human does all orchestration, and every
// discrete orchestration action is appended here as a typed event. Metric
// computation (P4-04) consumes these recorded event logs only — nothing else.
//
// Diff viewing is journalable but is NOT an intervention (PRD §31.1,
// PRE_REGISTRATION §7): countsAsIntervention is the frozen classifier.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// eventType enumerates every recordable trial event. The set is closed: an
// unknown type is rejected rather than silently coerced (no stringly domain
// state). Agent attempts and verification runs are journaled by the runner
// itself; human events are journaled via the `event` subcommand.
type eventType string

const (
	// Human orchestration events (PRE_REGISTRATION §3 M1/M3, discrete).
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
	// Logged but NOT an intervention (PRD §31.1: viewing a diff is not a
	// failure event).
	evDiffViewed eventType = "diff_viewed"
	// Runner-recorded events.
	evAgentAttempt eventType = "agent_attempt"
	evVerification eventType = "verification"
)

// allEventTypes is the closed vocabulary, in stable display order.
var allEventTypes = []eventType{
	evClarification, evManualRepair, evManualRerun, evWaiver, evTradeoffDecision,
	evForcedRetry, evDirectCodeCorrection, evFailureCopyBack, evPromptReissue,
	evVerificationRestart, evDiffViewed, evAgentAttempt, evVerification,
}

func parseEventType(s string) (eventType, error) {
	for _, t := range allEventTypes {
		if eventType(s) == t {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown event type %q (legal: %s)", s, eventTypeList())
}

func eventTypeList() string {
	parts := make([]string, 0, len(allEventTypes))
	for _, t := range allEventTypes {
		parts = append(parts, string(t))
	}
	return strings.Join(parts, ", ")
}

// countsAsIntervention is the frozen M1/M3 classifier: every human
// orchestration event counts; diff viewing does not (PRD §31.1).
func (t eventType) countsAsIntervention() bool {
	return t != evDiffViewed
}

// journalEvent is one append-only journal line.
type journalEvent struct {
	RecordedAt string    `json:"recorded_at"`
	TaskID     string    `json:"task_id"`
	Attempt    int       `json:"attempt,omitempty"`
	Type       eventType `json:"type"`
	// Intervention is the frozen classifier applied at record time so the
	// log is self-describing; P4-04 recomputes nothing.
	Intervention bool   `json:"intervention"`
	Note         string `json:"note,omitempty"`
}

// journalPath is the per-task event log under the runs root.
func journalPath(runsRoot, taskID string) string {
	return filepath.Join(runsRoot, taskID, "events.jsonl")
}

// appendEvent appends one event to the task journal (create-if-missing).
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

// summarizeEvents prints the journal with the frozen classifier applied.
func summarizeEvents(runsRoot, taskID string) error {
	evs, err := readEvents(runsRoot, taskID)
	if err != nil {
		return err
	}
	if len(evs) == 0 {
		fmt.Printf("%s: no journal events\n", taskID)
		return nil
	}
	interventions := 0
	for _, ev := range evs {
		if ev.Intervention {
			interventions++
		}
		marker := "intervention"
		if !ev.Intervention {
			marker = "non-intervention (logged only)"
		}
		fmt.Printf("%s  attempt-%d  %-22s  %-28s  %s\n", ev.RecordedAt, ev.Attempt, ev.Type, marker, ev.Note)
	}
	fmt.Printf("%s: %d events, %d interventions (frozen classifier: diff_viewed excluded, PRD §31.1)\n",
		taskID, len(evs), interventions)
	return nil
}

// sortJournalEvents is a deterministic order for cross-task summaries.
func sortJournalEvents(evs []journalEvent) {
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].RecordedAt < evs[j].RecordedAt })
}
