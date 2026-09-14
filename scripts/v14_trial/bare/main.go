// main.go wires the bare-arm subcommands:
//
//	go run ./scripts/v14_trial/bare self-check
//	go run ./scripts/v14_trial/bare provision --task <id>
//	go run ./scripts/v14_trial/bare run       --task <id> [--attempt N] [agent flags]
//	go run ./scripts/v14_trial/bare judge     --task <id> [--workspace <dir>] [db flags]
//	go run ./scripts/v14_trial/bare event     --task <id> --type <type> [--attempt N] [--note ...]
//	go run ./scripts/v14_trial/bare events    --task <id>
//
// The exact run procedure for BOTH trial arms is docs/evals/v1.4/RUN_PROTOCOL.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	case "provision":
		err = cmdProvision(args)
	case "run":
		err = cmdRun(args)
	case "judge":
		err = cmdJudge(args)
	case "event":
		err = cmdEvent(args)
	case "events":
		err = cmdEvents(args)
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
	fmt.Print(`bare — P4-02 bare-agent control runner (v1.4 product value experiment, arm A)

Deterministic trial tooling. It never imports or wires the Tally closed loop;
the judge is the arm-neutral external path both arms share (RUN_PROTOCOL.md).

Commands:
  self-check                  validate task-set loading, workspace provisioning
                              and judge wiring WITHOUT invoking codex (dry-run)
  provision --task <id>       materialize the per-task disposable workspace
  run       --task <id>       provision (attempt 1) + invoke the bare codex CLI,
                              recording argv, stdout JSONL, identity and timing
  judge     --task <id>       apply the frozen acceptance AFTER the run as the
                              external judge; writes judge-result.json
  event     --task <id>       append a human orchestration event to the journal
  events    --task <id>       print the journal with the frozen classifier

Common flags:
  --repo <dir>       repository root (default: walk up to the frozen manifest)
  --runs-root <dir>  trial records root (default: <os temp>/tally-v14-trial/bare;
                     records are local trial artifacts, never committed)
`)
}

func commonFlags(fs *flag.FlagSet) (repo *string, runsRoot *string) {
	repo = fs.String("repo", "", "repository root (default: walk up to the frozen manifest)")
	runsRoot = fs.String("runs-root", "", "trial records root (default: <os temp>/tally-v14-trial/bare)")
	return
}

func defaultRunsRoot() string {
	return filepath.Join(os.TempDir(), "tally-v14-trial", "bare")
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

func cmdSelfCheck(args []string) error {
	fs := flag.NewFlagSet("self-check", flag.ExitOnError)
	repo, _ := commonFlags(fs)
	psqlBin := fs.String("psql-bin", "psql", "PostgreSQL client binary (live-DB evidence; optional)")
	pg := fs.String("pg", envOr("TALLY_V14_PG", ""), "maintenance URI without db name; enables live-DB self-check when set")
	maintDB := fs.String("pg-maintenance-db", "postgres", "database used to create/drop the trial temp databases")
	if err := fs.Parse(args); err != nil {
		return err
	}
	set, _, err := loadSet(*repo)
	if err != nil {
		return err
	}
	ok, err := runSelfCheck(set, pgConfig{PsqlBin: *psqlBin, Maintenance: *pg, MaintDB: *maintDB})
	if err != nil {
		return err
	}
	if !ok {
		os.Exit(1)
	}
	return nil
}

func cmdProvision(args []string) error {
	fs := flag.NewFlagSet("provision", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	all := fs.Bool("all", false, "provision every frozen task")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *taskID == "" {
		return fmt.Errorf("--task <id> or --all is required")
	}
	set, _, err := loadSet(*repo)
	if err != nil {
		return err
	}
	root := orDefault(*runsRoot, defaultRunsRoot())
	count := 0
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if !*all && task.TaskID != *taskID {
			continue
		}
		ws := workspaceDir(root, task.TaskID, 1)
		if err := provisionTask(set, task, ws); err != nil {
			return err
		}
		fmt.Printf("provisioned %s -> %s\n", task.TaskID, ws)
		count++
	}
	if count == 0 {
		return fmt.Errorf("task %q not matched in the frozen set", *taskID)
	}
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	attempt := fs.Int("attempt", 1, "attempt number (1 provisions a fresh workspace; >1 reuses the existing workspace)")
	codexBin := fs.String("codex-bin", "codex", "codex CLI binary")
	model := fs.String("model", "", "model id passed as -m (MUST be the identical id both arms use)")
	sandbox := fs.String("sandbox", defaultSandbox, "codex -s mode (read-only | workspace-write | danger-full-access)")
	timeout := fs.Duration("timeout", defaultTimeout, "per-attempt wall-clock ceiling (equal budget both arms)")
	codexHome := fs.String("codex-home", "", "owner codex home for the auth.json handoff (default: CODEX_HOME or ~/.codex)")
	keepHome := fs.Bool("keep-home", false, "retain the per-attempt isolated CODEX_HOME (recorded, never hidden)")
	provider := &providerFlags{}
	fs.StringVar(&provider.ID, "provider-id", "", "ADR-0003 model_provider id (enables endpoint seeding)")
	fs.StringVar(&provider.Name, "provider-name", "", "ADR-0003 provider name")
	fs.StringVar(&provider.BaseURL, "provider-base-url", "", "ADR-0003 provider base URL")
	fs.StringVar(&provider.WireAPI, "provider-wire-api", "", "ADR-0003 wire api: responses | chat")
	fs.BoolVar(&provider.RequiresOpenAIAuth, "provider-requires-openai-auth", false, "ADR-0003: key arrives via the auth.json handoff")
	fs.StringVar(&provider.EnvKey, "provider-env-key", "", "ADR-0003 env var name carrying the API key (never the value)")
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
	task, err := set.task(*taskID)
	if err != nil {
		return err
	}
	cfg := invokeConfig{
		CodexBin:       *codexBin,
		Model:          *model,
		Sandbox:        *sandbox,
		Timeout:        *timeout,
		Provider:       provider,
		OwnerCodexHome: resolveOwnerCodexHome(*codexHome),
		KeepHome:       *keepHome,
	}
	return runBare(set, task, orDefault(*runsRoot, defaultRunsRoot()), *attempt, cfg)
}

func cmdJudge(args []string) error {
	fs := flag.NewFlagSet("judge", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	attempt := fs.Int("attempt", 1, "attempt whose workspace is judged")
	workspace := fs.String("workspace", "", "explicit workspace dir (overrides --attempt)")
	out := fs.String("out", "", "result path (default: <attempt dir>/judge-result.json)")
	psqlBin := fs.String("psql-bin", "psql", "PostgreSQL client binary")
	pg := fs.String("pg", envOr("TALLY_V14_PG", ""), "maintenance URI without db name (live-DB evidence; without it migration keys are NOT_RUN)")
	maintDB := fs.String("pg-maintenance-db", "postgres", "database used to create/drop the trial temp databases")
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
	task, err := set.task(*taskID)
	if err != nil {
		return err
	}
	root := orDefault(*runsRoot, defaultRunsRoot())
	ws := *workspace
	if ws == "" {
		ws = workspaceDir(root, task.TaskID, *attempt)
	}
	if _, err := os.Stat(ws); err != nil {
		return fmt.Errorf("workspace %s does not exist (run or provision first)", ws)
	}
	resultPath := *out
	if resultPath == "" {
		resultPath = filepath.Join(filepath.Dir(filepath.Clean(ws)), "judge-result.json")
	}
	res, err := runJudge(set, task, ws, resultPath, pgConfig{PsqlBin: *psqlBin, Maintenance: *pg, MaintDB: *maintDB})
	if err != nil {
		return err
	}
	for _, g := range res.Gate {
		fmt.Printf("gate %-24s %-7s %s\n", g.Key, g.Status, g.Detail)
	}
	verdict := "NOT_ACCEPTED (incomplete evidence — gate contains NOT_RUN)"
	if res.Accepted {
		verdict = "ACCEPTED (basis: frozen acceptance + oracle results in " + resultPath + ")"
	} else if res.Complete {
		verdict = "NOT_ACCEPTED (gate failed — basis recorded in " + resultPath + ")"
	}
	fmt.Printf("task %s: %s\n", task.TaskID, verdict)
	if _, err := appendEvent(root, task.TaskID, *attempt, evVerification, verdict); err != nil {
		return err
	}
	return nil
}

func cmdEvent(args []string) error {
	fs := flag.NewFlagSet("event", flag.ExitOnError)
	repo, runsRoot := commonFlags(fs)
	taskID := fs.String("task", "", "frozen task id")
	attempt := fs.Int("attempt", 1, "attempt the event belongs to")
	typeFlag := fs.String("type", "", "event type (closed vocabulary; see events --help)")
	note := fs.String("note", "", "free-text note (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return fmt.Errorf("--task <id> is required")
	}
	if _, err := parseEventType(*typeFlag); err != nil {
		return fmt.Errorf("%w\nlegal types: %s", err, eventTypeList())
	}
	set, _, err := loadSet(*repo)
	if err != nil {
		return err
	}
	if _, err := set.task(*taskID); err != nil {
		return err
	}
	ev, err := appendEvent(orDefault(*runsRoot, defaultRunsRoot()), *taskID, *attempt, eventType(*typeFlag), *note)
	if err != nil {
		return err
	}
	fmt.Printf("journalled %s (intervention=%v) for %s attempt %d\n", ev.Type, ev.Intervention, ev.TaskID, ev.Attempt)
	return nil
}

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
	return summarizeEvents(orDefault(*runsRoot, defaultRunsRoot()), *taskID)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
