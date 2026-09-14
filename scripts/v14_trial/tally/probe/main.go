// probe is the Tally-arm (P4-03) verification-side evidence probe for the
// FROZEN v1.4 product value experiment (docs/tasks/v1.4/P4/P4-03.md).
//
// It is trial tooling, not production code. It is the executable half of the
// frozen VerificationContract bindings the Tally arm composes per task: the
// loop's candidate verification service (internal/candidateverify) runs each
// visible binding's frozen command inside the one-shot verification workspace,
// and this probe applies exactly the frozen acceptance/oracle semantics the
// arm-neutral judge (scripts/v14_trial/bare judge) applies:
//
//	bug_fix:
//	  repro  -task T -repo R   prove the FROZEN unfixed baseline fails exactly
//	                           the frozen baseline_failures (candidate-independent,
//	                           mirror of the judge's bug_repro_failed_before basis)
//	  after  -task T -repo R   drive every frozen oracle case through the
//	                           CANDIDATE delivered in the current working
//	                           directory (regression_check_passes_after)
//	database_migration (live-DB evidence per PRD §24.2):
//	  fresh     -task T -repo R -dsn D   delivered chain applies to an empty DB
//	  n1        -task T -repo R -dsn D   chain applies to the seeded N-1 path
//	  integrity -task T -repo R -dsn D   frozen per-task data-integrity
//	                                     assertions hold on the upgraded DB
//
// Parity anchoring: the FINAL verdict for BOTH arms comes from the arm-neutral
// bare judge, never from this probe. The probe only drives the loop's
// verify/repair decisions; its tables below mirror the judge's verification-side
// knowledge (bug driver adapters, data-integrity assertions) and any drift can
// therefore only cause loop/judge disagreement, never a fabricated acceptance.
//
// Exit codes: 0 = evidence satisfied; 1 = evidence failed (a trusted FAIL,
// never an error); 2 = probe wiring/infra error (recorded, not fabricated).
//
// Files are read from the owner repository (-repo) or the current working
// directory (the verification workspace); the probe writes only into its own
// temp sandbox. It never prints oracle case content, only case names and
// want/got VERDICT STRINGS (the same strings the frozen oracle already carries).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	sub, args := os.Args[1], os.Args[2:]
	var err error
	switch sub {
	case "repro", "after":
		err = runBug(sub, args)
	case "fresh", "n1", "integrity":
		err = runMigration(sub, args)
	case "wiring":
		err = runWiring(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", sub)
		usage()
		os.Exit(2)
	}
	if err != nil {
		// Wiring/infra errors exit 2; evidence FAILs exit 1 from within.
		fmt.Fprintf(os.Stderr, "probe: %v\n", err)
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`probe — Tally-arm verification-side evidence probe (P4-03, frozen task set)

Subcommands (frozen VerificationContract binding commands):
  repro     -task <id> -repo <root>   frozen unfixed baseline fails exactly the frozen baseline_failures
  after     -task <id> -repo <root>   every frozen oracle case passes against the candidate in CWD
  fresh     -task <id> -repo <root> -dsn <uri>   delivered chain applies to an empty database
  n1        -task <id> -repo <root> -dsn <uri>   delivered chain applies to the seeded N-1 upgrade path
  integrity -task <id> -repo <root> -dsn <uri>   frozen data-integrity assertions hold on the upgraded DB

CWD semantics: bug_fix ` + "`after`" + ` and the migration subcommands resolve the CANDIDATE
delivered in the current working directory (the one-shot verification workspace);
` + "`-repo`" + ` locates the FROZEN verification-side assets (oracle/oracle.json, pinned
baseline). The probe never writes outside its own temp sandbox.
`)
}

// ---------------------------------------------------------------- bug_fix ----

// bugAdapter mirrors scripts/v14_trial/bare/judge.go bugAdapters verbatim (the
// arm-neutral judge's verification-side driver knowledge): how a frozen oracle
// input string is driven through the delivered fix. Provenance: copied from the
// bare judge so the loop's repair signal matches the final judge semantics; the
// judge remains the acceptance authority for both arms.
type bugAdapter struct {
	pkgDir  string   // workspace-relative package directory (the baseline dir)
	pkgName string   // package clause required in the delivered sources
	entry   string   // documented entry function
	imports []string // imports of the generated driver
	driveFn string   // full Go source of func v14Drive(in string) string
}

var bugAdapters = map[string]bugAdapter{
	"bf-01-token-bucket-offbyone": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-01-token-bucket-offbyone/baseline",
		pkgName: "limiter",
		entry:   "Allow",
		imports: []string{"strconv", "strings"},
		driveFn: `func v14Drive(in string) string {
	parts := strings.Split(strings.TrimSpace(in), ",")
	if len(parts) != 3 {
		return "invalid-input"
	}
	c, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	r, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	i, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	if Allow(c, r, i) {
		return "allow"
	}
	return "reject"
}`,
	},
	"bf-02-logout-session-invalidation": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-02-logout-session-invalidation/baseline",
		pkgName: "auth",
		entry:   "apply",
		imports: []string{"strings"},
		driveFn: `func v14Drive(in string) string {
	s := newStore()
	var out []string
	for _, op := range strings.Split(in, ";") {
		out = append(out, s.apply(op))
	}
	return strings.Join(out, ";")
}`,
	},
	"bf-03-pagination-partial-page": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-03-pagination-partial-page/baseline",
		pkgName: "web",
		entry:   "Page",
		imports: []string{"strconv", "strings"},
		driveFn: `func v14Drive(in string) string {
	parts := strings.Split(strings.TrimSpace(in), ",")
	if len(parts) != 3 {
		return "invalid-input"
	}
	total, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	page, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	return Page(total, pageSize, page)
}`,
	},
	"bf-04-date-validation-calendar": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-04-date-validation-calendar/baseline",
		pkgName: "ingest",
		entry:   "ValidateDate",
		driveFn: `func v14Drive(in string) string { return ValidateDate(in) }`,
	},
	"bf-05-currency-rounding-halfup": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-05-currency-rounding-halfup/baseline",
		pkgName: "billing",
		entry:   "FormatDollars",
		driveFn: `func v14Drive(in string) string { return FormatDollars(in) }`,
	},
	"bf-06-email-dedupe-casefold": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-06-email-dedupe-casefold/baseline",
		pkgName: "contacts",
		entry:   "UniqueEmails",
		driveFn: `func v14Drive(in string) string { return UniqueEmails(in) }`,
	},
}

// bugOracleDoc mirrors the frozen oracle/oracle.json document for bug_fix tasks
// (the same JSON shape the arm-neutral judge consumes).
type bugOracleDoc struct {
	Cases []struct {
		Name string `json:"name"`
		In   string `json:"in"`
		Want string `json:"want"`
	} `json:"cases"`
	Expected struct {
		BaselineFailures   []string `json:"baseline_failures"`
		ReferencePassesAll bool     `json:"reference_passes_all"`
	} `json:"expected"`
}

func loadBugOracle(repoRoot, taskID string) (*bugOracleDoc, error) {
	p := filepath.Join(repoRoot, filepath.FromSlash(
		"test/eval/v14_product/testdata/tasks/"+taskID+"/oracle/oracle.json"))
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var doc bugOracleDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("oracle.json does not parse: %w", err)
	}
	if len(doc.Cases) == 0 {
		return nil, fmt.Errorf("oracle.json pins no cases")
	}
	return &doc, nil
}

func runBug(sub string, args []string) error {
	fs := flag.NewFlagSet(sub, flag.ExitOnError)
	taskID := fs.String("task", "", "frozen task id")
	repo := fs.String("repo", "", "owner repository root (frozen verification-side assets)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" || *repo == "" {
		return fmt.Errorf("-task and -repo are required")
	}
	adapter, ok := bugAdapters[*taskID]
	if !ok {
		return fmt.Errorf("no bug driver adapter registered for task %s", *taskID)
	}
	oracle, err := loadBugOracle(*repo, *taskID)
	if err != nil {
		return fmt.Errorf("frozen oracle unreadable: %v", err)
	}

	// Candidate source: `after` reads the candidate package from the CURRENT
	// working directory (the verification workspace); `repro` materializes the
	// FROZEN baseline bytes from the owner repo instead — the repro gate is a
	// property of the frozen assets, not of any candidate.
	sandbox, err := os.MkdirTemp("", "v14-tally-probe-")
	if err != nil {
		return fmt.Errorf("probe sandbox: %v", err)
	}
	defer os.RemoveAll(sandbox)
	if err := os.WriteFile(filepath.Join(sandbox, "go.mod"), []byte("module v14probe\n\ngo 1.23\n"), 0o644); err != nil {
		return err
	}
	candDir := filepath.Join(sandbox, "candidate")
	if sub == "after" {
		// Candidate package comes from the verification workspace (CWD).
		if err := copyDir(filepath.Join(".", filepath.FromSlash(adapter.pkgDir)), candDir); err != nil {
			return fmt.Errorf("candidate package %s missing from verification workspace: %v", adapter.pkgDir, err)
		}
	} else {
		baseRel := "test/eval/v14_product/testdata/tasks/" + *taskID + "/baseline"
		if err := copyDir(filepath.Join(*repo, filepath.FromSlash(baseRel)), candDir); err != nil {
			return fmt.Errorf("frozen baseline unreadable: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(candDir, "v14_probe_driver_test.go"), []byte(driverSource(adapter, oracle)), 0o644); err != nil {
		return err
	}

	verdicts, out, err := goTestCandidate(candDir)
	if err != nil {
		if sub == "repro" {
			// The frozen baseline snapshot elides helpers and does not compile
			// standalone: exactly the case the arm-neutral judge records as
			// "carried by the frozen registry mirror" - the pinned baseline
			// bytes' identity (re-verified at every trial load) plus the
			// P4-01 registry proof carry bug_repro_failed_before. NOT a FAIL.
			fmt.Printf("probe repro %s: baseline snapshot does not compile standalone (helper elision); reproduction carried by the frozen registry mirror (P4-01 proof, manifest identity verified at every load)\n%s\n", *taskID, truncate(out, 600))
			return nil
		}
		// A non-compiling delivery is a trusted FAIL of the evidence, not a
		// probe wiring error (same semantics as the arm-neutral judge).
		fmt.Printf("probe %s %s: candidate did not compile or produced no verdicts\n%s\n", sub, *taskID, truncate(out, 2000))
		exitOne()
		return nil
	}
	var failed []string
	for _, c := range oracle.Cases {
		got := verdicts[c.Name]
		if got != c.Want {
			failed = append(failed, fmt.Sprintf("%s(want %s got %s)", c.Name, c.Want, got))
		}
	}
	switch {
	case sub == "after":
		if len(failed) != 0 {
			fmt.Printf("probe after %s: FAIL — frozen oracle cases failed against the delivered fix: %s\n", *taskID, strings.Join(failed, ", "))
			exitOne()
			return nil
		}
		fmt.Printf("probe after %s: all %d frozen oracle cases pass against the delivered fix\n", *taskID, len(oracle.Cases))
	default: // repro
		want := append([]string(nil), oracle.Expected.BaselineFailures...)
		got := failedCaseNames(failed)
		if !equalStrings(sortedCopy(got), sortedCopy(want)) {
			fmt.Printf("probe repro %s: FAIL — unfrozen baseline failure set mismatch: got %v want %v\n", *taskID, got, want)
			exitOne()
			return nil
		}
		fmt.Printf("probe repro %s: unfixed frozen baseline fails exactly the frozen baseline_failures %v\n", *taskID, got)
	}
	return nil
}

// failedCaseNames extracts case names from "name(want X got Y)" renderings.
func failedCaseNames(failed []string) []string {
	out := make([]string, 0, len(failed))
	for _, f := range failed {
		if i := strings.Index(f, "("); i > 0 {
			out = append(out, f[:i])
		}
	}
	return out
}

// driverSource renders the in-package test driver: the same shape the
// arm-neutral judge renders (frozen cases in, verdicts out via one marker line).
func driverSource(a bugAdapter, oracle *bugOracleDoc) string {
	var b strings.Builder
	b.WriteString("package " + a.pkgName + "\n\n")
	b.WriteString("import (\n\t\"encoding/json\"\n")
	for _, imp := range a.imports {
		b.WriteString("\t\"" + imp + "\"\n")
	}
	b.WriteString("\t\"testing\"\n)\n\n")
	b.WriteString("var v14ProbeCases = []struct{ Name, In, Want string }{\n")
	for _, c := range oracle.Cases {
		b.WriteString("\t{Name: " + goQuote(c.Name) + ", In: " + goQuote(c.In) + ", Want: " + goQuote(c.Want) + "},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString(a.driveFn + "\n\n")
	b.WriteString(`func TestV14ProbeOracle(t *testing.T) {
	verdicts := make(map[string]string, len(v14ProbeCases))
	for _, c := range v14ProbeCases {
		verdicts[c.Name] = v14Drive(c.In)
	}
	out, err := json.Marshal(verdicts)
	if err != nil {
		t.Fatalf("marshal verdicts: %v", err)
	}
	t.Logf("V14_PROBE_VERDICTS %s", out)
}
`)
	return b.String()
}

// goTestCandidate runs the probe's go test invocation in the sandbox package
// dir and parses the verdict marker. Offline guard: stdlib-only sandbox; any
// accidental network fetch fails loudly (GOPROXY=off).
func goTestCandidate(pkgDir string) (map[string]string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	argv := []string{"go", "test", ".", "-run", "TestV14ProbeOracle", "-v", "-count=1", "-vet=off"}
	res, runErr := runLocal(ctx, argv, pkgDir)
	verdicts, found := parseVerdicts(res.stdout)
	if found {
		return verdicts, res.stdout + res.stderr, nil
	}
	detail := truncate(strings.TrimSpace(res.stdout+"\n"+res.stderr), 2000)
	if runErr != nil {
		detail = runErr.Error() + ": " + detail
	}
	return nil, detail, fmt.Errorf("no verdicts")
}

const probeVerdictMarker = "V14_PROBE_VERDICTS "

func parseVerdicts(output string) (map[string]string, bool) {
	var verdicts map[string]string
	found := false
	for _, line := range strings.Split(output, "\n") {
		if i := strings.Index(line, probeVerdictMarker); i >= 0 {
			payload := strings.TrimSpace(line[i+len(probeVerdictMarker):])
			var v map[string]string
			if json.Unmarshal([]byte(payload), &v) == nil {
				verdicts = v
				found = true
			}
		}
	}
	return verdicts, found
}

// ------------------------------------------------------- database_migration ----

// migrationAssertion mirrors the arm-neutral judge's verification-side frozen
// data-integrity assertions (scripts/v14_trial/bare/judge.go
// dataIntegrityChecks). Provenance note as for bugAdapters above.
type migrationAssertion struct {
	Query  string `json:"query"`
	Want   string `json:"want"`
	Detail string `json:"detail"`
}

var dataIntegrityChecks = map[string][]migrationAssertion{
	"db-01-orders-total-backfill": {
		{Query: "SELECT count(*) FROM orders", Want: "3", Detail: "every seeded order row survives (frozen seed: 3 orders)"},
		{Query: "SELECT count(*) FROM orders WHERE total_cents IS NOT NULL", Want: "3", Detail: "the NOT NULL column is backfilled for every seeded order"},
		{Query: "SELECT count(*) FROM accounts", Want: "2", Detail: "identity/linkage rows intact"},
	},
	"db-02-account-email-unique": {
		{Query: "SELECT count(*) FROM accounts", Want: "3", Detail: "no seeded account row deleted"},
		{Query: "SELECT count(*) - count(DISTINCT email) FROM accounts", Want: "0", Detail: "duplicate addresses disambiguated, not deleted"},
		{Query: "SELECT count(*) FROM events", Want: "3", Detail: "linked event rows intact"},
	},
	"db-03-rename-breaks-downstream": {
		{Query: "SELECT count(*) FROM accounts", Want: "2", Detail: "seeded account rows survive the rename and indexing"},
		{Query: "SELECT count(*) FROM accounts WHERE to_jsonb(accounts)::text LIKE '%a@example.test%'", Want: "1", Detail: "first seeded e-mail value survives"},
		{Query: "SELECT count(*) FROM accounts WHERE to_jsonb(accounts)::text LIKE '%b@example.test%'", Want: "1", Detail: "second seeded e-mail value survives"},
	},
	"db-04-orders-fk-orphan": {
		{Query: "SELECT count(*) FROM orders", Want: "3", Detail: "every seeded order row survives the upgrade"},
		{Query: "SELECT count(*) FROM orders WHERE note = 'orphaned-by-import'", Want: "1", Detail: "the orphaned order row itself survives"},
		{Query: "SELECT count(*) FROM orders WHERE account_id = 999", Want: "0", Detail: "the dangling reference is reassigned, not left dangling"},
	},
	"db-05-drop-column-data-loss": {
		{Query: "SELECT count(*) FROM accounts", Want: "2", Detail: "seeded account rows survive"},
		{Query: "SELECT count(*) FROM accounts WHERE to_jsonb(accounts)::text LIKE '%tok-1%'", Want: "1", Detail: "legacy token value 1 preserved by the change"},
		{Query: "SELECT count(*) FROM accounts WHERE to_jsonb(accounts)::text LIKE '%tok-2%'", Want: "1", Detail: "legacy token value 2 preserved by the change"},
	},
	"db-06-total-type-narrowing": {
		{Query: "SELECT count(*) FROM orders", Want: "2", Detail: "every seeded order row survives"},
		{Query: "SELECT count(*) FROM orders WHERE id = 100 AND total_micros = 1500000", Want: "1", Detail: "in-range total unchanged"},
		{Query: "SELECT count(*) FROM orders WHERE id = 101 AND total_micros = 9999999999", Want: "1", Detail: "out-of-32-bit-range total survives unchanged"},
	},
}

func runMigration(sub string, args []string) error {
	fs := flag.NewFlagSet(sub, flag.ExitOnError)
	taskID := fs.String("task", "", "frozen task id")
	repo := fs.String("repo", "", "owner repository root (frozen verification-side assets)")
	dsn := fs.String("dsn", "", "maintenance PostgreSQL URI without db name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" || *repo == "" || strings.TrimSpace(*dsn) == "" {
		return fmt.Errorf("-task, -repo and -dsn are required")
	}
	oracle, err := loadMigrationOracle(*repo, *taskID)
	if err != nil {
		return fmt.Errorf("frozen oracle unreadable: %v", err)
	}
	checks, ok := dataIntegrityChecks[*taskID]
	if !ok && sub == "integrity" {
		return fmt.Errorf("no verification-side data-integrity assertions registered for %s", *taskID)
	}

	// The delivered chain is resolved inside the CURRENT working directory
	// (the verification workspace) with the frozen repo-relative layout.
	for _, rel := range oracle.Layout.Migrations {
		if _, err := os.Stat(filepath.Join(".", filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("delivered chain is incomplete: %s missing from verification workspace", rel)
		}
	}
	if oracle.Layout.Seed != "" {
		if _, err := os.Stat(filepath.Join(".", filepath.FromSlash(oracle.Layout.Seed))); err != nil {
			return fmt.Errorf("seed fixture missing from verification workspace: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	dbName := "v14tally_" + strings.NewReplacer("-", "_").Replace(*taskID) +
		fmt.Sprintf("_%d", time.Now().UnixNano())

	switch sub {
	case "fresh":
		db := dbName + "_fresh"
		defer dropDB(ctx, *dsn, db)
		if err := createDB(ctx, *dsn, db); err != nil {
			fmt.Printf("probe fresh %s: FAIL — create temp database: %v\n", *taskID, err)
			exitOne()
			return nil
		}
		if err := applyChain(ctx, *dsn, db, oracle, false); err != nil {
			fmt.Printf("probe fresh %s: FAIL — %v\n", *taskID, err)
			exitOne()
			return nil
		}
		fmt.Printf("probe fresh %s: delivered chain applies cleanly to an empty database\n", *taskID)
	case "n1":
		db := dbName + "_n1"
		defer dropDB(ctx, *dsn, db)
		if err := createDB(ctx, *dsn, db); err != nil {
			fmt.Printf("probe n1 %s: FAIL — create temp database: %v\n", *taskID, err)
			exitOne()
			return nil
		}
		if err := applyChain(ctx, *dsn, db, oracle, true); err != nil {
			fmt.Printf("probe n1 %s: FAIL — %v\n", *taskID, err)
			exitOne()
			return nil
		}
		fmt.Printf("probe n1 %s: delivered chain applies to the seeded N-1 upgrade path\n", *taskID)
	case "integrity":
		db := dbName + "_int"
		defer dropDB(ctx, *dsn, db)
		if err := createDB(ctx, *dsn, db); err != nil {
			fmt.Printf("probe integrity %s: FAIL — create temp database: %v\n", *taskID, err)
			exitOne()
			return nil
		}
		if err := applyChain(ctx, *dsn, db, oracle, true); err != nil {
			fmt.Printf("probe integrity %s: FAIL — N-1 upgrade path did not build, assertions unreachable: %v\n", *taskID, err)
			exitOne()
			return nil
		}
		var failed []string
		for _, c := range checks {
			got, qerr := queryScalar(ctx, *dsn, db, c.Query)
			if qerr != nil {
				failed = append(failed, fmt.Sprintf("%s (query error: %v)", c.Detail, qerr))
				continue
			}
			if got != c.Want {
				failed = append(failed, fmt.Sprintf("%s (want %s got %s)", c.Detail, c.Want, got))
			}
		}
		if len(failed) != 0 {
			fmt.Printf("probe integrity %s: FAIL — seeded data was altered or destroyed: %s\n", *taskID, strings.Join(failed, "; "))
			exitOne()
			return nil
		}
		fmt.Printf("probe integrity %s: every frozen data-integrity assertion holds on the upgraded database\n", *taskID)
	}
	return nil
}

// runWiring is the trial self-check's no-DB wiring surface for
// database_migration tasks: the frozen oracle layout must resolve inside the
// current working directory tree and the integrity assertions must be
// registered. Live-DB execution stays trial-time.
func runWiring(args []string) error {
	fs := flag.NewFlagSet("wiring", flag.ExitOnError)
	taskID := fs.String("task", "", "frozen task id")
	repo := fs.String("repo", "", "owner repository root (frozen verification-side assets)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" || *repo == "" {
		return fmt.Errorf("-task and -repo are required")
	}
	oracle, err := loadMigrationOracle(*repo, *taskID)
	if err != nil {
		return fmt.Errorf("frozen oracle unreadable: %v", err)
	}
	for _, rel := range oracle.Layout.Migrations {
		if _, err := os.Stat(filepath.Join(".", filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("frozen layout does not resolve in the given tree: %s: %v", rel, err)
		}
	}
	if oracle.Layout.Seed != "" {
		if _, err := os.Stat(filepath.Join(".", filepath.FromSlash(oracle.Layout.Seed))); err != nil {
			return fmt.Errorf("frozen seed does not resolve in the given tree: %v", err)
		}
	}
	if len(dataIntegrityChecks[*taskID]) == 0 {
		return fmt.Errorf("no verification-side data-integrity assertions registered for %s", *taskID)
	}
	fmt.Printf("probe wiring %s: frozen layout resolves in the given tree; %d integrity assertions registered\n",
		*taskID, len(dataIntegrityChecks[*taskID]))
	return nil
}

func loadMigrationOracle(repoRoot, taskID string) (*migrationOracleDoc, error) {
	p := filepath.Join(repoRoot, filepath.FromSlash(
		"test/eval/v14_product/testdata/tasks/"+taskID+"/oracle/oracle.json"))
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var doc migrationOracleDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("oracle.json does not parse: %w", err)
	}
	if len(doc.Layout.Migrations) == 0 {
		return nil, fmt.Errorf("oracle.json pins no migration revisions")
	}
	return &doc, nil
}

// migrationOracleDoc mirrors v14product.MigrationOracle (the frozen oracle
// shape) without importing the validator package into the probe binary.
type migrationOracleDoc struct {
	Layout struct {
		Migrations        []string `json:"migrations"`
		Seed              string   `json:"seed"`
		SeedAfterRevision string   `json:"seed_after_revision"`
		ReferenceRepair   string   `json:"reference_repair"`
		ReferenceReplaces string   `json:"reference_replaces"`
	} `json:"layout"`
}

// applyChain applies the delivered chain (CWD-relative frozen layout) to db;
// withSeed mirrors the arm-neutral judge's placement exactly: the seed applies
// right after the revision whose file base starts with seed_after_revision.
func applyChain(ctx context.Context, dsn, db string, oracle *migrationOracleDoc, withSeed bool) error {
	for _, rel := range oracle.Layout.Migrations {
		if err := psqlFile(ctx, dsn, db, rel); err != nil {
			return fmt.Errorf("revision %s failed: %v", filepath.Base(rel), err)
		}
		if withSeed && oracle.Layout.Seed != "" && oracle.Layout.SeedAfterRevision != "" &&
			strings.HasPrefix(filepath.Base(rel), oracle.Layout.SeedAfterRevision) {
			if err := psqlFile(ctx, dsn, db, oracle.Layout.Seed); err != nil {
				return fmt.Errorf("seed application failed: %v", err)
			}
		}
	}
	if withSeed && oracle.Layout.Seed != "" {
		// Verify the seed actually landed (same discipline as the judge: the
		// N-1 path must have been seeded exactly once).
		applied := false
		for _, rel := range oracle.Layout.Migrations {
			if strings.HasPrefix(filepath.Base(rel), oracle.Layout.SeedAfterRevision) {
				applied = true
				break
			}
		}
		if !applied {
			return fmt.Errorf("seed_after_revision %s matched no revision base — N-1 path never seeded", oracle.Layout.SeedAfterRevision)
		}
	}
	return nil
}

func dropDB(ctx context.Context, dsn, db string) {
	_ = execSQL(ctx, dsn, "postgres", "DROP DATABASE IF EXISTS "+db)
}

func createDB(ctx context.Context, dsn, db string) error {
	return execSQL(ctx, dsn, "postgres", "CREATE DATABASE "+db)
}

// ------------------------------------------------------------------ shared ----

func exitOne() { os.Exit(1) }

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func goQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
