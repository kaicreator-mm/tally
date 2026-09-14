// main.go wires the Tally-arm trial subcommands (P4-03):
//
//	go run ./scripts/v14_trial/tally self-check
//	go run ./scripts/v14_trial/tally env-check
//	go run ./scripts/v14_trial/tally run --task <id> [--all]
//	go run ./scripts/v14_trial/tally events --task <id>
//	go run ./scripts/v14_trial/tally rows --out <dir>
//
// The run procedure for BOTH arms is docs/evals/v1.4/RUN_PROTOCOL.md; the
// arm-neutral judge lives in scripts/v14_trial/bare (P4-02) and is delegated
// as the identical command arm A uses. Raw result rows are assembled by
// `rows` (no metric computation — P4-04 owns aggregation).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v14product "tally/test/eval/v14_product"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "self-check":
		err = cmdSelfCheck(args)
	case "env-check":
		err = cmdEnvCheck(args)
	case "run":
		err = cmdRun(args)
	case "events":
		err = cmdEvents(args)
	case "rows":
		err = cmdRows(args)
	case "rejudge":
		err = cmdRejudge(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`tally — P4-03 Tally-arm closed-loop trial runner (v1.4 product value experiment, arm B)

Runs the FROZEN task set through the REAL Tally closed loop (Intent → Contract →
Execute → Verify → bounded Repair → Accept/Escalate) driving the REAL codex
runtime, then applies the SAME arm-neutral judge command the bare arm uses.
RUN_PROTOCOL.md governs; raw rows only — no metric computation (P4-04).

Commands:
  self-check            dry validation (no codex binary): frozen-set loading,
                        contract composition under the production parsers,
                        fixture-repo composition, compiler seams, probe wiring
  env-check             trial environment surface (codex identity, credential
                        presence, facts DB) — records PASS/FAIL/NOT_RUN lines
  run --task <id>       execute one task's closed loop + external judge
  run --all             execute every frozen task sequentially
  events --task <id>    print the task journal (arm-B honesty surface)
  rows --out <dir>      assemble raw per-task rows for both arms from the
                        recorded artifacts (append-only trial record)

Common flags:
  --repo <dir>        repository root (default: walk up to the frozen manifest)
  --runs-root <dir>   Tally-arm records root (default: <temp>/tally-v14-trial/tally;
                      local trial artifacts, never committed)
`)
}

func commonFlags(fs *flag.FlagSet) (*string, *string) {
	repo := fs.String("repo", "", "repository root (default: walk up to the frozen manifest)")
	runsRoot := fs.String("runs-root", "", "Tally-arm records root (default: <temp>/tally-v14-trial/tally)")
	return repo, runsRoot
}

func defaultRunsRoot() string {
	return filepath.Join(os.TempDir(), "tally-v14-trial", "tally")
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func loadSet(repoFlag string) (*frozenSet, string, error) {
	root, err := findRepoRoot(repoFlag)
	if err != nil {
		return nil, "", err
	}
	set, err := loadFrozenSet(root)
	if err != nil {
		return nil, root, err
	}
	return set, root, nil
}

// trialConfigFromFlags assembles the requested configuration surface shared by
// run/env-check (parity values documented in RUN_PROTOCOL §4).
func trialConfigFromFlags(repo, runsRoot string) (trialConfig, *frozenSet, error) {
	set, root, err := loadSet(repo)
	if err != nil {
		return trialConfig{}, nil, err
	}
	cfg := trialConfig{
		Repo:           root,
		RunsRoot:       orDefault(runsRoot, defaultRunsRoot()),
		CodexBin:       envOr("TALLY_V14_CODEX_BIN", defaultCodexBin),
		Model:          envOr("TALLY_V14_MODEL", defaultTrialModel),
		Sandbox:        envOr("TALLY_V14_SANDBOX", "danger-full-access"),
		AttemptCeiling: 30 * time.Minute, // production ExecutionPolicy budget; equal per-attempt budget both arms
		TaskCeiling:    120 * time.Minute,
		DSN:            envOr("DATABASE_URL", defaultDSN),
		EnvName:        "p403-tally-trial",
	}
	return cfg, set, nil
}

const (
	defaultCodexBin = `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe`
	defaultDSN      = "postgres://postgres:test@localhost:5433/postgres?sslmode=disable"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ---------------------------------------------------------------------------
// env-check
// ---------------------------------------------------------------------------

func cmdEnvCheck(args []string) error {
	fs := flag.NewFlagSet("env-check", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	codexBin := fs.String("codex-bin", envOr("TALLY_V14_CODEX_BIN", defaultCodexBin), "codex CLI binary")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, set, err := trialConfigFromFlags(*repo, *runsRoot)
	if err != nil {
		return err
	}
	cfg.CodexBin = *codexBin
	ok := true
	line := func(status, name, detail string) {
		if status == "FAIL" {
			ok = false
		}
		fmt.Printf("env-check %s: %s — %s\n", status, name, detail)
	}

	line("PASS", "task-set", v14product.FreezeInfo(set.manifest)+" manifest="+shortSHA(set.sha))

	// codex CLI identity: one real --version probe (observed, never assumed).
	ctx, cancel := context2()
	defer cancel()
	rt := newCodexRuntime(cfg)
	snap, err := rt.Snapshot(ctx)
	if err != nil {
		line("FAIL", "codex runtime identity", err.Error())
	} else {
		line("PASS", "codex runtime identity",
			fmt.Sprintf("cli=%s adapter=%s binary=%s (observed)", snap.CLIVersion, snap.AdapterVersion, snap.BinaryPath))
	}

	// credential presence (never contents).
	home, present, err := prepareOwnerCodexHome(cfg.RunsRoot, resolveAuthFile(cfg.Repo))
	if err != nil {
		line("FAIL", "credential handoff home", err.Error())
	} else if present {
		line("PASS", "credential handoff home", "auth.json present in the minimal owner home "+home+" (contents never recorded)")
	} else {
		line("NOT_RUN", "credential handoff home", "auth file "+resolveAuthFile(cfg.Repo)+" absent — real runs would record not_run(missing credential)")
	}

	// facts DB.
	if err := checkDB(ctx, cfg.DSN); err != nil {
		line("FAIL", "facts database", err.Error())
	} else {
		line("PASS", "facts database", "connected: "+maintenanceURI(cfg.DSN))
	}

	// judge tooling builds.
	psqlsh, err := ensurePsqlsh(cfg.Tools())
	if err != nil {
		line("FAIL", "judge psql client (psqlsh)", err.Error())
	} else {
		line("PASS", "judge psql client (psqlsh)", psqlsh)
	}

	if !ok {
		os.Exit(1)
	}
	return nil
}

// tools returns the tools dir inside the runs root.
func (c *trialConfig) Tools() string { return filepath.Join(c.RunsRoot, "tools") }

// ---------------------------------------------------------------------------
// run
// ---------------------------------------------------------------------------

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	all := fs.Bool("all", false, "run every frozen task sequentially")
	codexBin := fs.String("codex-bin", envOr("TALLY_V14_CODEX_BIN", defaultCodexBin), "codex CLI binary")
	model := fs.String("model", envOr("TALLY_V14_MODEL", defaultTrialModel), "requested model id (-m; identical id both arms)")
	sandbox := fs.String("sandbox", envOr("TALLY_V14_SANDBOX", "danger-full-access"), "codex -s mode")
	taskCeiling := fs.Duration("task-ceiling", 120*time.Minute, "per-task wall-clock ceiling for the whole loop (documented per row)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *taskID == "" {
		return fmt.Errorf("--task <id> or --all is required")
	}
	cfg, set, err := trialConfigFromFlags(*repo, *runsRoot)
	if err != nil {
		return err
	}
	cfg.CodexBin = *codexBin
	cfg.Model = *model
	cfg.Sandbox = *sandbox
	cfg.TaskCeiling = *taskCeiling
	cfg.ToolsDir = cfg.Tools()

	// Credential handoff (Decision Lock 22): minimal owner home with ONLY
	// auth.json. skip-not-fail when absent → not_run rows, honest record.
	authFile := resolveAuthFile(cfg.Repo)
	home, present, err := prepareOwnerCodexHome(cfg.RunsRoot, authFile)
	if err != nil {
		return err
	}
	cfg.OwnerCodexHome = home
	cfg.AuthPresent = present

	// Tooling builds.
	if _, err := ensureProbe(cfg.Tools()); err != nil {
		return err
	}
	if _, err := ensurePsqlsh(cfg.Tools()); err != nil {
		return err
	}

	var failures int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if !*all && task.TaskID != *taskID {
			continue
		}
		if !cfg.AuthPresent {
			res := &trialResult{
				TaskID:         task.TaskID,
				ContractTaskID: contractTaskIDFor(set, task.TaskID),
				Outcome:        "not_run",
				OutcomeDetail:  "credential file absent: " + authFile + " (skip-not-fail; Decision Lock 22 handoff impossible)",
				ManifestSHA:    set.sha,
				Identity:       captureIdentity(),
			}
			if err := writeRunRecord(cfg.RunsRoot, task.TaskID, res); err != nil {
				return err
			}
			fmt.Printf("task %s: not_run (credential absent) — row recorded honestly\n", task.TaskID)
			failures++
			continue
		}
		fmt.Printf("=== task %s: closed loop starting (model=%s sandbox=%s) ===\n", task.TaskID, cfg.Model, cfg.Sandbox)
		res, err := runTask(set, task, cfg)
		if res != nil {
			if werr := writeRunRecord(cfg.RunsRoot, task.TaskID, res); werr != nil {
				return werr
			}
		}
		if err != nil {
			failures++
			fmt.Printf("task %s: %s — %v (%s)\n", task.TaskID, res.Outcome, err, res.OutcomeDetail)
			continue
		}
		fmt.Printf("task %s: %s — %s\n", task.TaskID, res.Outcome, res.OutcomeDetail)
	}
	if failures > 0 {
		fmt.Printf("completed with %d not_run/failed task rows (all retained in the record; never dropped)\n", failures)
	}
	return nil
}

// cmdRejudge re-runs the arm-neutral judge on an existing candidate workspace
// and updates the run record (completing the judge step after an infra defect
// in the original judge invocation; the loop facts are never touched).
func cmdRejudge(args []string) error {
	fs := flag.NewFlagSet("rejudge", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	reason := fs.String("reason", "infra", "short reason tag for the superseded judge result file")
	all := fs.Bool("all", false, "rejudge every task that has a run record")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *taskID == "" {
		return fmt.Errorf("--task <id> or --all is required")
	}
	cfg, set, err := trialConfigFromFlags(*repo, *runsRoot)
	if err != nil {
		return err
	}
	cfg.ToolsDir = cfg.Tools()
	if _, err := ensurePsqlsh(cfg.ToolsDir); err != nil {
		return err
	}
	var failures int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if !*all && task.TaskID != *taskID {
			continue
		}
		if _, err := os.Stat(filepath.Join(cfg.RunsRoot, task.TaskID, "run-record.json")); err != nil {
			fmt.Printf("rejudge %s: no run record, skipping\n", task.TaskID)
			continue
		}
		if err := rejudgeTask(set, cfg, task, *reason); err != nil {
			failures++
			fmt.Printf("rejudge %s: FAIL: %v\n", task.TaskID, err)
			continue
		}
		fmt.Printf("rejudge %s: judge step completed\n", task.TaskID)
	}
	if failures > 0 {
		return fmt.Errorf("%d rejudge failures", failures)
	}
	return nil
}

func writeRunRecord(runsRoot, taskID string, res *trialResult) error {
	p := filepath.Join(runsRoot, taskID, "run-record.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

// ---------------------------------------------------------------------------
// events
// ---------------------------------------------------------------------------

func cmdEvents(args []string) error {
	fs := flag.NewFlagSet("events", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return fmt.Errorf("--task <id> is required")
	}
	set, _, err := loadSet(*repo)
	if err != nil {
		return err
	}
	if _, err := set.task(*taskID); err != nil {
		return err
	}
	root := orDefault(*runsRoot, defaultRunsRoot())
	evs, err := readEvents(root, *taskID)
	if err != nil {
		return err
	}
	if len(evs) == 0 {
		fmt.Printf("%s: no journal events\n", *taskID)
		return nil
	}
	interventions := 0
	for _, ev := range evs {
		if ev.Intervention {
			interventions++
		}
		marker := "runner-event"
		if ev.Intervention {
			marker = "intervention"
		}
		fmt.Printf("%s  attempt-%d  %-16s  %-14s  %s\n", ev.RecordedAt, ev.Attempt, ev.Type, marker, ev.Note)
	}
	fmt.Printf("%s: %d events, %d intervention events (frozen classifier: agent attempts, verifications and diff viewing are not interventions; escalations are, PRD §31)\n",
		*taskID, len(evs), interventions)
	return nil
}
