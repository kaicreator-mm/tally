// rows.go assembles the RAW trial record rows (docs/evals/v1.4/results/raw/**)
// from the recorded artifacts of BOTH arms. It performs NO metric computation
// and NO aggregation (P4-04 owns those): each row is the per-task factual
// record — arm, task, outcome, attempts, interventions, timings, identity,
// judge verdict, NOT_RUN reasons — exactly what PRE_REGISTRATION §8 and the
// task contract require.
//
// Arm A rows are read from the bare runner's records (attempt.json /
// judge-result.json / events.jsonl); arm B rows from the Tally arm's
// run-record.json (which embeds the identical judge-result.json produced by
// the same judge command). Interventions are classified with the PRD §31
// vocabulary on both sides: human orchestration events count; agent attempts,
// verifications and diff viewing do not; arm B escalations are the
// human-decision events the loop hands over (the e2e ledger semantics).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// rawRow is the shared per-task raw record shape for both arms.
type rawRow struct {
	Arm            string          `json:"arm"`
	TaskID         string          `json:"task_id"`
	ContractTaskID string          `json:"contract_task_id,omitempty"`
	Outcome        string          `json:"outcome"` // accepted | rejected | escalated | cancelled | not_run
	OutcomeDetail  string          `json:"outcome_detail,omitempty"`
	Attempts       int             `json:"attempts"`
	Interventions  []rawInterv     `json:"interventions"`
	Timing         rawTiming       `json:"timing"`
	Identity       json.RawMessage `json:"identity,omitempty"`
	Requested      json.RawMessage `json:"requested,omitempty"`
	Judge          json.RawMessage `json:"judge,omitempty"`
	JudgeNote      string          `json:"judge_note,omitempty"`
	LoopRecord     json.RawMessage `json:"loop_record,omitempty"`
	NOTRunReasons  []string        `json:"not_run_reasons,omitempty"`
	ManifestSHA    string          `json:"manifest_sha256,omitempty"`
	RecordedAt     string          `json:"recorded_at"`
}

type rawInterv struct {
	Type   string `json:"type"`
	Note   string `json:"note,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type rawTiming struct {
	StartedAt         string    `json:"started_at,omitempty"`
	FinishedAt        string    `json:"finished_at,omitempty"`
	WallSeconds       float64   `json:"wall_seconds,omitempty"`
	PerAttemptSeconds []float64 `json:"per_attempt_seconds,omitempty"`
}

// humanEventTypes is the PRD §31 intervention vocabulary (the events a HUMAN
// performs); runner-recorded attempts/verifications and diff viewing are never
// interventions on either arm.
var humanEventTypes = map[string]bool{
	"clarification": true, "manual_repair": true, "manual_rerun": true,
	"waiver": true, "tradeoff_decision": true, "forced_retry": true,
	"direct_code_correction": true, "failure_copy_back": true,
	"prompt_reissue": true, "verification_restart": true,
}

func cmdRows(args []string) error {
	fs := flag.NewFlagSet("rows", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	bareRunsRoot := fs.String("bare-runs-root", envOr("TALLY_V14_BARE_RUNS_ROOT", filepath.Join(os.TempDir(), "tally-v14-trial", "bare")), "bare-arm records root (arm A)")
	out := fs.String("out", "", "output directory under docs/evals/v1.4/results/raw (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*out) == "" {
		return fmt.Errorf("--out <dir> is required (raw rows only land under docs/evals/v1.4/results/raw)")
	}
	set, root, err := loadSet(*repo)
	if err != nil {
		return err
	}
	if !strings.Contains(filepath.ToSlash(*out), "docs/evals/v1.4/results/raw") {
		return fmt.Errorf("--out must be under docs/evals/v1.4/results/raw (append-only raw record), got %s", *out)
	}

	tallyRoot := orDefault(*runsRoot, defaultRunsRoot())
	bareRoot := *bareRunsRoot

	written := 0
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		// Arm B row.
		b, err := os.ReadFile(filepath.Join(tallyRoot, task.TaskID, "run-record.json"))
		if err == nil {
			row, err := tallyRow(set, task.TaskID, b, journalEvents(tallyRoot, task.TaskID))
			if err != nil {
				return fmt.Errorf("tally row %s: %w", task.TaskID, err)
			}
			// Provenance, not outcomes: preserved superseded infra-failure
			// artifacts inside the task record (AUDIT_RESUME.md §5).
			if _, serr := os.Stat(filepath.Join(tallyRoot, task.TaskID, "not_run-infra")); serr == nil {
				row.NOTRunReasons = append(row.NOTRunReasons,
					"provenance: superseded infra-failure artifacts preserved under the task record's not_run-infra/ (see AUDIT_RESUME.md §5)")
			}
			if err := writeRow(filepath.Join(*out, "tally", task.TaskID+".json"), row); err != nil {
				return err
			}
			written++
		} else if row, ok, ierr := interruptedTallyRow(tallyRoot, task.TaskID, set.sha); ierr != nil {
			return fmt.Errorf("tally interrupted row %s: %w", task.TaskID, ierr)
		} else if ok {
			// The loop started but the trial runner died before a terminal
			// state: materialize the honest interrupted row (never skipped,
			// never fabricated into a verdict).
			if err := writeRow(filepath.Join(*out, "tally", task.TaskID+".json"), row); err != nil {
				return err
			}
			written++
			fmt.Printf("rows: tally arm %s materialized as INTERRUPTED (journal preserved, no run record)\n", task.TaskID)
		} else {
			fmt.Printf("rows: tally arm has no records for %s (NOT_RUN row will note absence)\n", task.TaskID)
		}
		// Arm A row.
		if row, ok, err := bareRow(task.TaskID, bareRoot); err != nil {
			return fmt.Errorf("bare row %s: %w", task.TaskID, err)
		} else if ok {
			if err := writeRow(filepath.Join(*out, "bare", task.TaskID+".json"), row); err != nil {
				return err
			}
			written++
		} else {
			fmt.Printf("rows: bare arm has no records for %s\n", task.TaskID)
		}
	}

	// Trial-level environment record (shared identity surface; no metrics).
	envRec := map[string]any{
		"trial":           "P4-03 real Codex Tally closed-loop comparative trial",
		"date":            time.Now().UTC().Format("2006-01-02"),
		"manifest_sha256": set.sha,
		"frozen_at":       set.manifest.FrozenAt,
		"host_os":         hostOS(),
		"protocol":        "docs/evals/v1.4/RUN_PROTOCOL.md",
		"preregistration": "docs/evals/v1.4/PRE_REGISTRATION.md",
		"arms":            "A=bare (scripts/v14_trial/bare), B=tally closed loop (scripts/v14_trial/tally)",
		"judge":           "arm-neutral: go run ./scripts/v14_trial/bare judge (identical command both arms)",
		"model_policy":    "requested model recorded separately; observed model not self-reported by the codex exec JSONL surface (unknown, never backfilled)",
		"sandbox_mode":    "danger-full-access (both arms, identical)",
		"integrity_notes": []string{
			"DISCLOSED: codex-cli 0.153.0 on this host cannot enforce a writable-workspace sandbox; under the adjudicated danger-full-access mode agents in BOTH arms had read access to the whole host, including verification-side assets (frozen oracle, reference repairs, trial scripts). Agent stdout streams in both arms contain oracle/reference-repair path references (discovery evidence recorded in the runs roots).",
			"DISCLOSED (tally arm): under that mode the implementing agents of db-02, db-03 and db-05 discovered and executed the trial's own arm-neutral judge (go run ./scripts/v14_trial/bare judge) with custom --out paths during their attempts; every invocation is journalled and its output preserved (Temp/judge-db02-repair1.json, tally-v14-trial/judge-db03-repair1.json, tally-v14-trial/judge-db03-repair1-final.json, Temp/judge-db05-current.json, Temp/judge-db05-reworked.json). They did not and could not alter the loop verdicts (probe-based) or the recorded final judge results (the runner's own invocations at 23:00:04Z for db-02, 23:37:38Z for db-03). db-05's runner died before its runner judge; its row is interrupted.",
			"Row outcomes are loop-governed for arm B (accepted/escalated) with the external judge verdict recorded alongside; disagreements are retained, never resolved in favor of either signal.",
		},
		"sandbox_adjudication": "DEVIATION DISCLOSED: codex-cli 0.153.0 on this Windows host enforces -s workspace-write as READ-ONLY (every agent command rejected 'blocked by policy'; verified by minimal write test). Coordinator adjudicated 2026-09-10 pre-trial: both arms run -s danger-full-access (identical mode both arms, parity preserved; disposable-workspace discipline unchanged). The one task attempted under the broken mode (bf-06, arm A only, 3 attempts + 2 prompt_reissue) was VOIDED and is preserved untouched under <bare runs root>/superseded-environment-workspace-write-readonly-defect/ with its adjudication note; both arms re-ran the full set under the corrected environment.",
	}
	b, _ := json.MarshalIndent(envRec, "", "  ")
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "trial-environment.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("rows: wrote %d rows + trial-environment.json under %s\n", written, *out)
	_ = root
	return nil
}

func hostOS() string { return os.Getenv("OS") }

func writeRow(path string, row rawRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func journalEvents(runsRoot, taskID string) []journalEvent {
	evs, err := readEvents(runsRoot, taskID)
	if err != nil {
		return nil
	}
	return evs
}

// tallyRow converts a Tally-arm run-record into the raw row.
func tallyRow(set *frozenSet, taskID string, record []byte, evs []journalEvent) (rawRow, error) {
	var rec trialResult
	if err := json.Unmarshal(record, &rec); err != nil {
		return rawRow{}, err
	}
	row := rawRow{
		Arm:            "tally",
		TaskID:         taskID,
		ContractTaskID: rec.ContractTaskID,
		Outcome:        rec.Outcome,
		OutcomeDetail:  rec.OutcomeDetail,
		Attempts:       len(rec.Attempts),
		Timing: rawTiming{
			StartedAt:   rec.StartedAt,
			FinishedAt:  rec.FinishedAt,
			WallSeconds: rec.WallSeconds,
		},
		Judge:       rec.Judge,
		JudgeNote:   rec.JudgeNote,
		LoopRecord:  json.RawMessage(record),
		ManifestSHA: set.sha,
		RecordedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	for _, a := range rec.Attempts {
		row.Timing.PerAttemptSeconds = append(row.Timing.PerAttemptSeconds, a.WallSeconds)
	}
	for _, ev := range evs {
		if ev.Intervention && (humanEventTypes[string(ev.Type)] || ev.Type == evEscalation) {
			row.Interventions = append(row.Interventions, rawInterv{Type: string(ev.Type), Note: ev.Note})
		}
	}
	if rec.Outcome == "not_run" {
		row.NOTRunReasons = append(row.NOTRunReasons, rec.OutcomeDetail)
	}
	if rec.Judge == nil {
		if rec.JudgeNote != "" {
			row.NOTRunReasons = append(row.NOTRunReasons, "judge: "+rec.JudgeNote)
		} else {
			row.NOTRunReasons = append(row.NOTRunReasons, "judge: not executed")
		}
	}
	b, err := json.Marshal(rec.Identity)
	if err == nil {
		row.Identity = b
	}
	b, err = json.Marshal(rec.Requested)
	if err == nil {
		row.Requested = b
	}
	return row, nil
}

// interruptedTallyRow materializes the raw row for a tally task whose closed
// loop started but never reached a terminal trial state (the trial runner died
// before the loop verdict, candidate materialization and the external judge —
// no run-record.json exists). Facts come exclusively from the preserved
// journal, whose agent_attempt notes carry the loop's own recording format
// ("run <id>: status=%s iterations=%d wall=%.1fs claim=%q"). Nothing is
// fabricated: no terminal verdict, no judge result, no per-run identity — every
// absence is recorded in not_run_reasons (P4-03 AC2/AC3: no silent drops).
func interruptedTallyRow(tallyRoot, taskID, manifestSHA string) (rawRow, bool, error) {
	evs, err := readEvents(tallyRoot, taskID)
	if err != nil || len(evs) == 0 {
		return rawRow{}, false, err
	}
	row := rawRow{
		Arm:     "tally",
		TaskID:  taskID,
		Outcome: "interrupted",
		OutcomeDetail: "trial runner died before the closed loop reached a terminal state, before candidate " +
			"materialization and before the arm-neutral judge; partial artifacts preserved untouched under the " +
			"tally runs root (AUDIT_RESUME.md §3); the runner has no loop resumption, so per the resume discipline " +
			"the task was recorded as interrupted, not re-run",
		Attempts:    0,
		ManifestSHA: manifestSHA,
		RecordedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	for _, ev := range evs {
		if row.Timing.StartedAt == "" {
			row.Timing.StartedAt = ev.RecordedAt
		}
		row.Timing.FinishedAt = ev.RecordedAt
		if ev.Type == evAgentAttempt && strings.HasPrefix(ev.Note, "run ") {
			row.Attempts++
			if s := wallSecondsFromNote(ev.Note); s > 0 {
				row.Timing.PerAttemptSeconds = append(row.Timing.PerAttemptSeconds, s)
			}
		}
		if ev.Intervention && (humanEventTypes[string(ev.Type)] || ev.Type == evEscalation) {
			row.Interventions = append(row.Interventions, rawInterv{Type: string(ev.Type), Note: ev.Note})
		}
	}
	row.NOTRunReasons = append(row.NOTRunReasons,
		"loop terminal state: never reached (interrupted after the attempts recorded above)",
		"judge: never executed by the runner",
		"run-record.json: never written (runner death); per-attempt identity record therefore absent — trial-level environment identity is in trial-environment.json",
	)
	// On this path the runner's own judge never ran, so every verification
	// journal event is an agent-executed judge invocation during an attempt
	// (agent-produced evidence, disclosed, not a trial verdict).
	for _, ev := range evs {
		if ev.Type == evVerification {
			row.NOTRunReasons = append(row.NOTRunReasons,
				"disclosure: agent-executed arm-neutral judge during attempt (journalled; agent-produced evidence, not the trial judge): "+ev.Note)
		}
	}
	return row, true, nil
}

// wallSecondsFromNote extracts the wall=%.1fs value from the loop's own
// agent_attempt recording format (codegen.go appendEvent note).
func wallSecondsFromNote(note string) float64 {
	const marker = "wall="
	i := strings.Index(note, marker)
	if i < 0 {
		return 0
	}
	rest := note[i+len(marker):]
	if j := strings.Index(rest, "s"); j >= 0 {
		rest = rest[:j]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
	if err != nil {
		return 0
	}
	return v
}

// bareRow assembles the arm-A raw row from the bare runner's records.
func bareRow(taskID, bareRoot string) (rawRow, bool, error) {
	dir := filepath.Join(bareRoot, taskID)
	row := rawRow{
		Arm:        "bare",
		TaskID:     taskID,
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	// attempts
	for n := 1; ; n++ {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("attempt-%d", n), "attempt.json"))
		if err != nil {
			break
		}
		var att struct {
			WallSeconds float64         `json:"wall_seconds"`
			ExitStatus  string          `json:"exit_status"`
			Summary     string          `json:"summary"`
			Requested   json.RawMessage `json:"requested"`
			Observed    json.RawMessage `json:"observed"`
			Host        json.RawMessage `json:"host"`
			StartedAt   string          `json:"started_at"`
		}
		if err := json.Unmarshal(b, &att); err != nil {
			return row, false, fmt.Errorf("attempt %d record: %w", n, err)
		}
		row.Attempts++
		row.Timing.PerAttemptSeconds = append(row.Timing.PerAttemptSeconds, att.WallSeconds)
		row.Timing.WallSeconds += att.WallSeconds
		if n == 1 {
			row.Timing.StartedAt = att.StartedAt
			row.Identity = att.Host
			row.Requested = att.Requested
		}
		if att.ExitStatus != "completed" {
			row.NOTRunReasons = append(row.NOTRunReasons,
				fmt.Sprintf("attempt %d exit_status=%s (%s)", n, att.ExitStatus, att.Summary))
		}
	}
	if row.Attempts == 0 {
		return rawRow{}, false, nil
	}
	// judge
	judgePath := filepath.Join(dir, fmt.Sprintf("attempt-%d", row.Attempts), "judge-result.json")
	b, err := os.ReadFile(judgePath)
	if err == nil {
		row.Judge = json.RawMessage(b)
		var doc struct {
			Accepted bool `json:"accepted"`
			Complete bool `json:"complete"`
		}
		if json.Unmarshal(b, &doc) == nil {
			switch {
			case doc.Accepted:
				row.Outcome = "accepted"
			case doc.Complete:
				row.Outcome = "rejected"
				row.OutcomeDetail = "external judge NOT_ACCEPTED (gate failed; basis in judge result)"
			default:
				row.Outcome = "not_run"
				row.OutcomeDetail = "external judge incomplete (NOT_RUN keys retained)"
				row.NOTRunReasons = append(row.NOTRunReasons, "judge incomplete")
			}
		}
	} else {
		row.Outcome = "not_run"
		row.JudgeNote = "judge-result.json absent (judge never ran)"
		row.NOTRunReasons = append(row.NOTRunReasons, "judge: never ran")
	}
	// journal (bare's events.jsonl has the same shape)
	evs, err := readEvents(bareRoot, taskID)
	if err == nil {
		for _, ev := range evs {
			if humanEventTypes[string(ev.Type)] {
				row.Interventions = append(row.Interventions, rawInterv{Type: string(ev.Type), Note: ev.Note})
			}
		}
	}
	return row, true, nil
}
