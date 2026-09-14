// judge.go owns the external judge: after the agent process has exited, the
// frozen acceptance is applied to the candidate workspace by the SAME path
// for BOTH arms (PRE_REGISTRATION §7 — a bare agent's self-declared done is a
// Claim, never a pass; PRD §30). The judge is arm-neutral: it reads the frozen
// acceptance + oracle from the REPOSITORY (verification-side) and the
// candidate's delivered bytes from the WORKSPACE, and it never writes anything
// back into the workspace.
//
// Per harness:
//   - bug_fix: compile the candidate package in a judge sandbox (a temp module
//     OUTSIDE the owner worktree) and drive every frozen oracle case through
//     the documented entry point; every case must pass
//     (regression_check_passes_after).
//   - database_migration: live-DB evidence per PRD §24.2 — apply the
//     delivered chain on a fresh database and on an N-1 upgraded+seeded
//     database, then run the per-task frozen data-integrity assertions. When
//     no PostgreSQL is available the keys are recorded NOT_RUN(reason), never
//     PASS (Decision Lock 20).
//
// Every ACCEPTED result carries a basis (trust invariant: no basis, no
// acceptance); an incomplete gate is never an acceptance.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"tally/test/eval/v14_product"
)

// gate statuses are typed strings; NOT_RUN is a first-class outcome and never
// collapses into pass or fail.
const (
	gatePass   = "pass"
	gateFail   = "fail"
	gateNotRun = "NOT_RUN"
)

type gateKeyResult struct {
	Key    string `json:"key"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type caseVerdict struct {
	Name string `json:"name"`
	Want string `json:"want"`
	Got  string `json:"got"`
	Pass bool   `json:"pass"`
}

type bugBasis struct {
	CompileOK     bool          `json:"compile_ok"`
	CompileDetail string        `json:"compile_detail,omitempty"`
	Cases         []caseVerdict `json:"cases"`
}

type migrationAssertion struct {
	Query  string `json:"query"`
	Want   string `json:"want"`
	Got    string `json:"got,omitempty"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

type migrationBasis struct {
	Assertions []migrationAssertion `json:"assertions"`
	DBVersion  string               `json:"db_version,omitempty"`
}

type judgeResult struct {
	TaskID           string          `json:"task_id"`
	Harness          string          `json:"harness"`
	Workspace        string          `json:"workspace"`
	Accepted         bool            `json:"accepted"`
	Complete         bool            `json:"complete"`
	Gate             []gateKeyResult `json:"gate"`
	Bug              *bugBasis       `json:"bug,omitempty"`
	Migration        *migrationBasis `json:"migration,omitempty"`
	JudgedAt         string          `json:"judged_at"`
	ManifestSHA256   string          `json:"manifest_sha256"`
	AcceptanceSHA256 string          `json:"acceptance_sha256"`
	Env              envIdentity     `json:"environment"`
	Notes            string          `json:"notes,omitempty"`
}

// acceptanceDoc mirrors the frozen acceptance.yaml contract (verification
// side). The judge resolves the gate from THIS file — never from runner-side
// defaults — so both arms are judged against exactly the frozen clauses.
type acceptanceDoc struct {
	TaskID        string   `yaml:"task_id"`
	Harness       string   `yaml:"harness"`
	Requires      []string `yaml:"requires"`
	GateCondition struct {
		AllOf []string `yaml:"all_of"`
	} `yaml:"gate_condition"`
}

func acceptancePath(repoRoot, taskID string) string {
	return filepath.Join(repoRoot, filepath.FromSlash(v14product.FixturePath(taskID, "acceptance.yaml")))
}

func loadAcceptance(repoRoot, taskID string) (*acceptanceDoc, string, error) {
	path := acceptancePath(repoRoot, taskID)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var doc acceptanceDoc
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, "", fmt.Errorf("acceptance %s does not parse: %w", path, err)
	}
	if doc.TaskID != taskID {
		return nil, "", fmt.Errorf("acceptance %s declares task_id %q", path, doc.TaskID)
	}
	if len(doc.GateCondition.AllOf) == 0 {
		return nil, "", fmt.Errorf("acceptance %s has an empty gate_condition.all_of", path)
	}
	return &doc, sha256Hex(b), nil
}

// pgConfig is the live-evidence database surface (trial-time; never committed).
type pgConfig struct {
	PsqlBin     string
	Maintenance string // URI WITHOUT a database name, e.g. postgres://user@host:5432
	MaintDB     string // database used to CREATE/DROP the temp databases
}

func (p pgConfig) available() bool { return strings.TrimSpace(p.Maintenance) != "" }

// runJudge applies the frozen acceptance to one candidate workspace and
// writes the result record next to it (never into it).
func runJudge(set *frozenSet, task *v14product.ManifestTask, ws, outPath string, pg pgConfig) (*judgeResult, error) {
	acc, accSHA, err := loadAcceptance(set.root, task.TaskID)
	if err != nil {
		return nil, err
	}
	res := &judgeResult{
		TaskID:           task.TaskID,
		Harness:          task.Harness,
		Workspace:        ws,
		JudgedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		ManifestSHA256:   set.sha,
		AcceptanceSHA256: accSHA,
		Env:              captureHostIdentity(),
	}
	switch {
	case task.Harness == v14product.HarnessBugFix:
		fillBugGate(set, task, ws, acc, res)
	case task.Harness == v14product.HarnessMigration:
		fillMigrationGate(set, task, ws, acc, pg, res)
	default:
		return nil, fmt.Errorf("harness %q is outside the preregistered set", task.Harness)
	}
	for _, g := range res.Gate {
		if g.Status != gatePass {
			res.Accepted = false
		}
		if g.Status == gateNotRun {
			res.Complete = false
		}
	}
	if allPass(res.Gate) {
		res.Accepted = true
	}
	res.Complete = !containsNotRun(res.Gate)

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return nil, err
	}
	return res, nil
}

func allPass(gate []gateKeyResult) bool {
	for _, g := range gate {
		if g.Status != gatePass {
			return false
		}
	}
	return len(gate) > 0
}

func containsNotRun(gate []gateKeyResult) bool {
	for _, g := range gate {
		if g.Status == gateNotRun {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// bug_fix judge
// ---------------------------------------------------------------------------

// bugAdapter is the verification-side driver knowledge for one bug_fix task:
// how a frozen oracle input string is driven through the delivered fix. It is
// derived entirely from the frozen baseline snapshots' documented input
// encodings, is never shipped to any agent, and is identical for both arms.
type bugAdapter struct {
	pkgDir  string   // workspace-relative package directory (the baseline dir)
	pkgName string   // package clause required in the delivered sources
	entry   string   // documented entry function
	arity   int      // parameters of the entry function
	imports []string // imports of the generated driver
	driveFn string   // full Go source of func v14Drive(in string) string
	stub    string   // deliberately-wrong compile stub used ONLY by self-check
}

var bugAdapters = map[string]bugAdapter{
	"bf-01-token-bucket-offbyone": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-01-token-bucket-offbyone/baseline",
		pkgName: "limiter",
		entry:   "Allow", arity: 3,
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
		stub: `func Allow(capacity, refilled, requestIndex int) bool { return true }`,
	},
	"bf-02-logout-session-invalidation": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-02-logout-session-invalidation/baseline",
		pkgName: "auth",
		entry:   "apply", arity: 1,
		imports: []string{"strings"},
		driveFn: `func v14Drive(in string) string {
	s := newStore()
	var out []string
	for _, op := range strings.Split(in, ";") {
		out = append(out, s.apply(op))
	}
	return strings.Join(out, ";")
}`,
		stub: `type store struct{ valid map[string]string }

func newStore() *store { return &store{valid: map[string]string{}} }

func (s *store) apply(op string) string {
	if strings.HasPrefix(op, "auth:") {
		return "ok:wrong-user"
	}
	return "logged-out"
}`,
	},
	"bf-03-pagination-partial-page": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-03-pagination-partial-page/baseline",
		pkgName: "web",
		entry:   "Page", arity: 3,
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
		stub: `func Page(total, pageSize, page int) string { return "out-of-range" }`,
	},
	"bf-04-date-validation-calendar": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-04-date-validation-calendar/baseline",
		pkgName: "ingest",
		entry:   "ValidateDate", arity: 1,
		driveFn: `func v14Drive(in string) string { return ValidateDate(in) }`,
		stub:    `func ValidateDate(s string) string { return "valid" }`,
	},
	"bf-05-currency-rounding-halfup": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-05-currency-rounding-halfup/baseline",
		pkgName: "billing",
		entry:   "FormatDollars", arity: 1,
		driveFn: `func v14Drive(in string) string { return FormatDollars(in) }`,
		stub:    `func FormatDollars(micros string) string { return "0.00" }`,
	},
	"bf-06-email-dedupe-casefold": {
		pkgDir:  "test/eval/v14_product/testdata/tasks/bf-06-email-dedupe-casefold/baseline",
		pkgName: "contacts",
		entry:   "UniqueEmails", arity: 1,
		driveFn: `func v14Drive(in string) string { return UniqueEmails(in) }`,
		stub:    `func UniqueEmails(in string) string { return in }`,
	},
}

// bugOracleDoc is the frozen oracle/oracle.json document for a bug_fix task
// (verification side; same JSON shape P4-01 pinned).
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
	b, err := os.ReadFile(filepath.Join(repoRoot,
		filepath.FromSlash(v14product.FixturePath(taskID, "oracle/oracle.json"))))
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

// fillBugGate evaluates the two frozen bug_fix gate keys for a candidate.
func fillBugGate(set *frozenSet, task *v14product.ManifestTask, ws string, acc *acceptanceDoc, res *judgeResult) {
	var gate []gateKeyResult
	// bug_repro_failed_before is a property of the FROZEN assets, not of the
	// candidate: the shipped baseline's failure set was registry-proven at
	// P4-01 (go test ./test/eval/v14_product) and the baseline bytes are
	// hash-pinned in the manifest — re-verified at every load.
	repro := gateKeyResult{
		Key:    "bug_repro_failed_before",
		Status: gatePass,
		Detail: "frozen baseline bytes hash-identical to the set P4-01 proved reproducibly failing " +
			"(manifest " + shortSHA(set.sha) + "); frozen failure set carried by oracle/oracle.json",
	}
	// regression_check_passes_after: every frozen case must pass against the
	// delivered fix, compiled and driven in a judge sandbox.
	basis, reg := judgeBugCandidate(set, task, ws)
	res.Bug = basis

	for _, key := range acc.GateCondition.AllOf {
		if key == "bug_repro_failed_before" {
			gate = append(gate, repro)
		} else if key == "regression_check_passes_after" {
			gate = append(gate, reg)
		} else {
			gate = append(gate, gateKeyResult{Key: key, Status: gateNotRun,
				Detail: "gate key has no judge implementation — refusing to fabricate a status"})
		}
	}
	res.Gate = gate
}

// judgeBugCandidate compiles the delivered package in a judge sandbox and
// drives every frozen case. The sandbox is a temp module OUTSIDE the owner
// worktree; the workspace itself is only read.
func judgeBugCandidate(set *frozenSet, task *v14product.ManifestTask, ws string) (*bugBasis, gateKeyResult) {
	notRun := func(reason string) (*bugBasis, gateKeyResult) {
		return nil, gateKeyResult{Key: "regression_check_passes_after", Status: gateNotRun, Detail: reason}
	}
	adapter, ok := bugAdapters[task.TaskID]
	if !ok {
		return notRun("no verification-side driver adapter registered for task (judge wiring gap)")
	}
	oracle, err := loadBugOracle(set.root, task.TaskID)
	if err != nil {
		return notRun(fmt.Sprintf("frozen oracle unreadable: %v", err))
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(adapter.pkgDir))); err != nil {
		return notRun(fmt.Sprintf("candidate package dir %s missing from workspace: %v", adapter.pkgDir, err))
	}

	sandbox, err := os.MkdirTemp("", "tally-v14-bare-judge-")
	if err != nil {
		return notRun(fmt.Sprintf("judge sandbox: %v", err))
	}
	defer os.RemoveAll(sandbox)
	if err := os.WriteFile(filepath.Join(sandbox, "go.mod"), []byte("module v14judgesandbox\n\ngo 1.23\n"), 0o644); err != nil {
		return notRun(fmt.Sprintf("judge sandbox go.mod: %v", err))
	}
	candDir := filepath.Join(sandbox, "candidate")
	if err := copyDir(filepath.Join(ws, filepath.FromSlash(adapter.pkgDir)), candDir); err != nil {
		return notRun(fmt.Sprintf("copy candidate package: %v", err))
	}
	driver := driverSource(adapter, oracle)
	if err := os.WriteFile(filepath.Join(candDir, "v14_judge_driver_test.go"), []byte(driver), 0o644); err != nil {
		return notRun(fmt.Sprintf("write judge driver: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	argv := []string{"go", "test", "./candidate", "-run", "TestV14FrozenOracleJudge", "-v", "-count=1", "-vet=off"}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = sandbox
	// Offline guard: the judge sandbox needs nothing beyond the standard
	// library; make any accidental network fetch fail loudly instead of
	// silently depending on it.
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, runErr := cmd.CombinedOutput()

	basis := &bugBasis{}
	verdicts, found := parseVerdicts(string(out))
	if found {
		basis.CompileOK = true
		for _, c := range oracle.Cases {
			got, ok := verdicts[c.Name]
			basis.Cases = append(basis.Cases, caseVerdict{Name: c.Name, Want: c.Want, Got: got, Pass: ok && got == c.Want})
		}
	} else {
		basis.CompileOK = false
		basis.CompileDetail = truncate(strings.TrimSpace(string(out)), 2000)
		if runErr != nil {
			basis.CompileDetail = truncate(runErr.Error()+": "+basis.CompileDetail, 2500)
		}
		for _, c := range oracle.Cases {
			basis.Cases = append(basis.Cases, caseVerdict{Name: c.Name, Want: c.Want, Got: "(no verdict: package did not compile/test)", Pass: false})
		}
	}
	if !basis.CompileOK {
		return basis, gateKeyResult{Key: "regression_check_passes_after", Status: gateFail,
			Detail: "candidate package did not compile or the judge driver produced no verdicts — a non-compiling delivery fails acceptance"}
	}
	for _, c := range basis.Cases {
		if !c.Pass {
			return basis, gateKeyResult{Key: "regression_check_passes_after", Status: gateFail,
				Detail: "frozen oracle case failed against the delivered fix (see bug.cases in the result record)"}
		}
	}
	return basis, gateKeyResult{Key: "regression_check_passes_after", Status: gatePass,
		Detail: fmt.Sprintf("all %d frozen oracle cases pass against the delivered fix (judge sandbox, go toolchain)", len(basis.Cases))}
}

// driverSource renders the in-package test driver: frozen cases in, verdicts
// out — the ONLY generated content the judge ever injects, into the sandbox
// copy only, never into the workspace.
func driverSource(a bugAdapter, oracle *bugOracleDoc) string {
	var b strings.Builder
	b.WriteString("package " + a.pkgName + "\n\n")
	b.WriteString("import (\n\t\"encoding/json\"\n")
	for _, imp := range a.imports {
		b.WriteString("\t\"" + imp + "\"\n")
	}
	b.WriteString("\t\"testing\"\n)\n\n")
	b.WriteString("var v14JudgeCases = []struct{ Name, In, Want string }{\n")
	for _, c := range oracle.Cases {
		b.WriteString("\t{Name: " + goQuote(c.Name) + ", In: " + goQuote(c.In) + ", Want: " + goQuote(c.Want) + "},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString(a.driveFn + "\n\n")
	b.WriteString(`func TestV14FrozenOracleJudge(t *testing.T) {
	verdicts := make(map[string]string, len(v14JudgeCases))
	for _, c := range v14JudgeCases {
		verdicts[c.Name] = v14Drive(c.In)
	}
	out, err := json.Marshal(verdicts)
	if err != nil {
		t.Fatalf("marshal verdicts: %v", err)
	}
	t.Logf("V14_ORACLE_VERDICTS %s", out)
}
`)
	return b.String()
}

// parseVerdicts extracts the last V14_ORACLE_VERDICTS payload from go test
// output.
func parseVerdicts(output string) (map[string]string, bool) {
	const marker = "V14_ORACLE_VERDICTS "
	var verdicts map[string]string
	found := false
	for _, line := range strings.Split(output, "\n") {
		if i := strings.Index(line, marker); i >= 0 {
			payload := strings.TrimSpace(line[i+len(marker):])
			var v map[string]string
			if json.Unmarshal([]byte(payload), &v) == nil {
				verdicts = v
				found = true
			}
		}
	}
	return verdicts, found
}

// ---------------------------------------------------------------------------
// database_migration judge
// ---------------------------------------------------------------------------

// dataIntegrityChecks is the verification-side assertion table for the frozen
// data_integrity clauses (acceptance AC-03 per db task, over the frozen seed
// fixtures). Verification-side only; identical for both arms. to_jsonb-based
// value-survival checks are robust to legitimate column renames in a fix.
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

// fillMigrationGate evaluates the three frozen database_migration gate keys
// by live-DB evidence (PRD §24.2) against the delivered chain in the
// workspace. Seed placement mirrors the frozen analyzer's semantics exactly:
// the seed applies right after the revision whose file base starts with
// seed_after_revision.
func fillMigrationGate(set *frozenSet, task *v14product.ManifestTask, ws string, acc *acceptanceDoc, pg pgConfig, res *judgeResult) {
	notRunDetail := "live-DB evidence (PRD §24.2) is the trial-time judge for database_migration; " +
		"PostgreSQL client/connection is not available in this environment; the frozen oracle's static coherence was proven at P4-01"
	oracle, err := loadMigrationOracle(set.root, task.TaskID)
	if err != nil {
		for _, key := range acc.GateCondition.AllOf {
			res.Gate = append(res.Gate, gateKeyResult{Key: key, Status: gateNotRun, Detail: "frozen oracle unreadable: " + err.Error()})
		}
		return
	}
	// The candidate chain must resolve inside the workspace: every layout
	// revision and the seed must be present at the pinned relative paths.
	for _, rel := range oracle.Layout.Migrations {
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
			for _, key := range acc.GateCondition.AllOf {
				res.Gate = append(res.Gate, gateKeyResult{Key: key, Status: gateFail,
					Detail: "delivered chain is incomplete: " + rel + " missing from workspace"})
			}
			return
		}
	}
	if oracle.Layout.Seed != "" {
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(oracle.Layout.Seed))); err != nil {
			for _, key := range acc.GateCondition.AllOf {
				res.Gate = append(res.Gate, gateKeyResult{Key: key, Status: gateNotRun,
					Detail: "seed fixture missing from workspace: " + err.Error()})
			}
			return
		}
	}
	if !pg.available() {
		for _, key := range acc.GateCondition.AllOf {
			res.Gate = append(res.Gate, gateKeyResult{Key: key, Status: gateNotRun, Detail: notRunDetail})
		}
		res.Notes = "database_migration acceptance was NOT evaluated: provide --pg (maintenance URI) and a PostgreSQL client to execute the live evidence path"
		return
	}
	basis, gate := executeMigrationEvidence(set, task, ws, acc, pg, oracle)
	res.Migration = basis
	res.Gate = gate
}

func loadMigrationOracle(repoRoot, taskID string) (*v14product.MigrationOracle, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot,
		filepath.FromSlash(v14product.FixturePath(taskID, "oracle/oracle.json"))))
	if err != nil {
		return nil, err
	}
	var doc v14product.MigrationOracle
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("oracle.json does not parse: %w", err)
	}
	if len(doc.Layout.Migrations) == 0 {
		return nil, fmt.Errorf("oracle.json pins no migration revisions")
	}
	return &doc, nil
}

// executeMigrationEvidence runs the fresh / N-1 / integrity paths against two
// disposable temp databases.
func executeMigrationEvidence(set *frozenSet, task *v14product.ManifestTask, ws string, acc *acceptanceDoc, pg pgConfig, oracle *v14product.MigrationOracle) (*migrationBasis, []gateKeyResult) {
	basis := &migrationBasis{}
	freshStatus, freshDetail := gateNotRun, "not executed"
	n1Status, n1Detail := gateNotRun, "not executed"
	dataStatus, dataDetail := gateNotRun, "not executed"

	abs := func(rel string) string { return filepath.Join(ws, filepath.FromSlash(rel)) }
	dbName := "v14trial_" + strings.NewReplacer("-", "_").Replace(task.TaskID) + fmt.Sprintf("_%d", time.Now().UnixNano())

	freshDB, n1DB := dbName+"_fresh", dbName+"_n1"
	defer func() {
		_ = psqlExec(context.Background(), pg, pg.MaintDB, "-c", "DROP DATABASE IF EXISTS "+freshDB)
		_ = psqlExec(context.Background(), pg, pg.MaintDB, "-c", "DROP DATABASE IF EXISTS "+n1DB)
	}()

	// fresh_database: the delivered chain applies cleanly to an empty DB.
	if err := psqlExec(context.Background(), pg, pg.MaintDB, "-c", "CREATE DATABASE "+freshDB); err != nil {
		freshDetail = "create temp database: " + err.Error()
	} else {
		ok := true
		var failDetail string
		for _, rel := range oracle.Layout.Migrations {
			if err := psqlExec(context.Background(), pg, freshDB, "-1", "-v", "ON_ERROR_STOP=1", "-q", "-f", abs(rel)); err != nil {
				ok = false
				failDetail = fmt.Sprintf("revision %s failed on a fresh database: %v", filepath.Base(rel), err)
				break
			}
		}
		if ok {
			freshStatus, freshDetail = gatePass, "delivered chain applies cleanly to an empty database"
		} else {
			freshStatus, freshDetail = gateFail, failDetail
		}
	}

	// n_minus_1_upgrade: upgrade from the seed-after revision with seeded rows.
	if err := psqlExec(context.Background(), pg, pg.MaintDB, "-c", "CREATE DATABASE "+n1DB); err != nil {
		n1Detail = "create temp database: " + err.Error()
	} else {
		seeded := false
		ok := true
		var failDetail string
		for _, rel := range oracle.Layout.Migrations {
			if err := psqlExec(context.Background(), pg, n1DB, "-1", "-v", "ON_ERROR_STOP=1", "-q", "-f", abs(rel)); err != nil {
				ok = false
				failDetail = fmt.Sprintf("revision %s failed on the upgraded path: %v", filepath.Base(rel), err)
				break
			}
			if !seeded && oracle.Layout.SeedAfterRevision != "" &&
				strings.HasPrefix(filepath.Base(rel), oracle.Layout.SeedAfterRevision) {
				if err := psqlExec(context.Background(), pg, n1DB, "-1", "-v", "ON_ERROR_STOP=1", "-q", "-f", abs(oracle.Layout.Seed)); err != nil {
					ok = false
					failDetail = "seed application failed: " + err.Error()
					break
				}
				seeded = true
			}
		}
		if ok && !seeded && oracle.Layout.Seed != "" {
			ok = false
			failDetail = "seed_after_revision " + oracle.Layout.SeedAfterRevision + " matched no revision base — N-1 path never seeded"
		}
		if ok {
			n1Status, n1Detail = gatePass, "delivered chain applies to the seeded N-1 upgrade path"
		} else {
			n1Status, n1Detail = gateFail, failDetail
		}
	}

	// data_integrity: frozen per-task assertions on the upgraded database.
	if n1Status == gatePass {
		checks := dataIntegrityChecks[task.TaskID]
		if len(checks) == 0 {
			dataStatus, dataDetail = gateNotRun, "no verification-side data-integrity assertions registered for task (judge wiring gap)"
		} else {
			allOK := true
			for i := range checks {
				c := &checks[i]
				got, qerr := psqlQuery(context.Background(), pg, n1DB, c.Query)
				c.Got = got
				if qerr != nil {
					c.Pass = false
					c.Detail = c.Detail + " (query error: " + qerr.Error() + ")"
					allOK = false
					continue
				}
				c.Pass = got == c.Want
				if !c.Pass {
					allOK = false
				}
			}
			basis.Assertions = checks
			if allOK {
				dataStatus, dataDetail = gatePass, "every frozen data-integrity assertion holds on the upgraded database"
			} else {
				dataStatus, dataDetail = gateFail, "seeded data was altered or destroyed (see migration.assertions in the result record)"
			}
		}
	} else {
		dataStatus, dataDetail = gateNotRun, "data_integrity runs on the N-1 database, which did not build: "+n1Detail
	}

	basis.DBVersion = probeDBVersion(pg)

	byKey := map[string]gateKeyResult{
		"fresh_database":    {Key: "fresh_database", Status: freshStatus, Detail: freshDetail},
		"n_minus_1_upgrade": {Key: "n_minus_1_upgrade", Status: n1Status, Detail: n1Detail},
		"data_integrity":    {Key: "data_integrity", Status: dataStatus, Detail: dataDetail},
	}
	var gate []gateKeyResult
	for _, key := range acc.GateCondition.AllOf {
		if r, ok := byKey[key]; ok {
			gate = append(gate, r)
		} else {
			gate = append(gate, gateKeyResult{Key: key, Status: gateNotRun, Detail: "gate key has no judge implementation — refusing to fabricate a status"})
		}
	}
	return basis, gate
}

// psqlExec runs one psql invocation against the named database.
func psqlExec(ctx context.Context, pg pgConfig, db string, args ...string) error {
	full := append([]string{"-X", "--dbname", pg.Maintenance + "/" + db}, args...)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, pg.PsqlBin, full...).CombinedOutput() //nolint:gosec // args are structural, not a shell string
	if err != nil {
		return fmt.Errorf("%v: %s", err, truncate(strings.TrimSpace(string(out)), 400))
	}
	return nil
}

// psqlQuery runs one scalar query and returns the trimmed single value.
func psqlQuery(ctx context.Context, pg pgConfig, db, query string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, pg.PsqlBin, "-X", "-A", "-t",
		"--dbname", pg.Maintenance+"/"+db, "-c", query).CombinedOutput() //nolint:gosec // query is from the frozen verification-side table
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, truncate(strings.TrimSpace(string(out)), 400))
	}
	return strings.TrimSpace(string(out)), nil
}

func probeDBVersion(pg pgConfig) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, pg.PsqlBin, "-X", "-A", "-t",
		"--dbname", pg.Maintenance+"/"+pg.MaintDB, "-c", "SHOW server_version").CombinedOutput() //nolint:gosec // static query
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ---------------------------------------------------------------------------
// shared sandbox helpers
// ---------------------------------------------------------------------------

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

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
