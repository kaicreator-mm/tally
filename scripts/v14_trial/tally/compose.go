// compose.go owns the Tally arm's per-task Contract derivation (PRD §30.1 arm
// B: "Tally derives the Contract from the same intent"): the frozen
// acceptance.yaml is the composition source (its own header prescribes exactly
// this), and the verbatim frozen intent.md bytes are the objective the loop
// sends to the agent. Nothing is paraphrased by hand — every composed byte is
// a deterministic function of frozen bytes plus the trial-time probe location.
//
// Layout of the per-task fixture repository (a disposable Git repo under the
// runs root, never committed):
//
//	commit 1 (agent base, BaseRevision): agent-facing frozen fixtures at their
//	  repo-relative layout + .tally/project.yaml + .tally/requirements (for the
//	  pinned contracts' refs) + .tally/tasks/<contract>.yaml + .tally/policies/
//	  <catalog bundle verbatim>.
//	commit 2 (HEAD, artifacts.Load authority): + .tally/verifications/
//	  <contract>.yaml — the verification authority.
//
// The verification contract lives only in commit 2 so that NO agent-facing
// surface (execution worktree or one-shot verification workspace, both created
// from the base revision) ever contains it. This enforces the frozen role
// split of TASK_SET.md §4 inside arm B: oracle/** and reference repairs are
// verification-side and must never reach an implementing agent, and the frozen
// contract's binding commands (probe argv) are likewise implementation-invisible.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"tally/test/eval/v14_product"
)

// acceptanceDoc mirrors the frozen acceptance.yaml (verification-side frozen
// contract; the composition source for the Tally arm's Contract).
type acceptanceDoc struct {
	TaskID    string   `yaml:"task_id"`
	Harness   string   `yaml:"harness"`
	PolicyRef string   `yaml:"policy_ref"`
	Requires  []string `yaml:"requires"`
	GateCond  struct {
		AllOf []string `yaml:"all_of"`
	} `yaml:"gate_condition"`
	Acceptance []struct {
		ID        string `yaml:"id"`
		Statement string `yaml:"statement"`
		OracleRef string `yaml:"oracle_ref"`
	} `yaml:"acceptance"`
}

func loadAcceptanceDoc(repoRoot, taskID string) (*acceptanceDoc, error) {
	p := filepath.Join(repoRoot, filepath.FromSlash(v14product.FixturePath(taskID, "acceptance.yaml")))
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var doc acceptanceDoc
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("acceptance %s does not parse: %w", p, err)
	}
	if doc.TaskID != taskID {
		return nil, fmt.Errorf("acceptance %s declares task_id %q", p, doc.TaskID)
	}
	if len(doc.GateCond.AllOf) == 0 || len(doc.Acceptance) == 0 {
		return nil, fmt.Errorf("acceptance %s has empty gate_condition.all_of or acceptance", p)
	}
	return &doc, nil
}

// statementForGate deterministically maps a gate key to the frozen acceptance
// clause whose statement names it (bf: AC-03/AC-04; db: AC-01/02/03). Every
// frozen gate key must resolve to exactly one clause; otherwise composition
// fails loudly.
func statementForGate(acc *acceptanceDoc, key string) (string, error) {
	hits := []string{}
	for _, a := range acc.Acceptance {
		if strings.Contains(a.Statement, key) {
			hits = append(hits, a.Statement)
		}
	}
	if len(hits) != 1 {
		return "", fmt.Errorf("gate key %s resolves to %d acceptance clauses in the frozen acceptance (%v), want exactly 1", key, len(hits), hits)
	}
	return hits[0], nil
}

// contractTaskIDFor maps a frozen task id to the Tally-side contract task id.
// bf-01/bf-02 reuse their PINNED harness contracts (TASK-2410/TASK-2411 —
// agent-facing fixtures, single frozen source). Every other task composes
// TASK-9100+<frozen index> (trial-side, TASK-<digits> per PRD §6.1).
var pinnedContractTasks = map[string]string{
	"bf-01-token-bucket-offbyone":       "TASK-2410",
	"bf-02-logout-session-invalidation": "TASK-2411",
}

func contractTaskIDFor(set *frozenSet, taskID string) string {
	if id, ok := pinnedContractTasks[taskID]; ok {
		return id
	}
	return fmt.Sprintf("TASK-%d", 9100+set.taskIndex(taskID))
}

// ---- composed YAML shapes (field names match the production parsers) ----

type taskContractYAML struct {
	ID              string   `yaml:"id"`
	Profile         string   `yaml:"profile"`
	Title           string   `yaml:"title"`
	Objective       string   `yaml:"objective"`
	RequirementRefs []string `yaml:"requirement_refs"`
	Scope           struct {
		AllowedPaths []string `yaml:"allowed_paths"`
	} `yaml:"scope"`
	Acceptance []struct {
		ID                   string `yaml:"id"`
		Statement            string `yaml:"statement"`
		IndependenceRequired string `yaml:"independence_required"`
	} `yaml:"acceptance"`
}

type verificationYAML struct {
	TaskID      string `yaml:"task_id"`
	Revision    int    `yaml:"revision"`
	Acceptances []struct {
		ID                   string        `yaml:"id"`
		Statement            string        `yaml:"statement"`
		IndependenceRequired string        `yaml:"independence_required"`
		EvidenceBindings     []bindingYAML `yaml:"evidence_bindings"`
	} `yaml:"acceptances"`
}

type bindingYAML struct {
	ID                string       `yaml:"id"`
	AuthoringMode     string       `yaml:"authoring_mode"`
	VisibilityPolicy  string       `yaml:"visibility_policy"`
	Evidence          evidenceYAML `yaml:"evidence"`
	ExecutionProtocol protoYAML    `yaml:"execution_protocol"`
}

type evidenceYAML struct {
	Type string `yaml:"type"`
	Path string `yaml:"path,omitempty"`
}

type protoYAML struct {
	Repetitions int     `yaml:"repetitions"`
	Aggregation string  `yaml:"aggregation"`
	Command     cmdYAML `yaml:"command"`
}

type cmdYAML struct {
	Argv           []string `yaml:"argv"`
	Timeout        string   `yaml:"timeout"`
	InfraExitCodes []int    `yaml:"infra_exit_codes,omitempty"`
}

// migrationProbeSubcommands maps frozen migration gate keys to the probe's
// migration subcommands (scripts/v14_trial/tally/probe/main.go accepts only
// fresh/n1/integrity). P6-04 diagnosed the false-negative trigger: the gate
// key used verbatim as the probe subcommand deterministically exits 2
// ("unknown subcommand") for every candidate. The mapping is total and fails
// loudly on an unmapped key (frozen-set drift), so composition can never
// silently emit an unexecutable binding command again.
var migrationProbeSubcommands = map[string]string{
	"fresh_database":    "fresh",
	"n_minus_1_upgrade": "n1",
	"data_integrity":    "integrity",
}

// taskComposition is the derived per-task composition record (also embedded in
// the raw result row so every composed input is auditable).
type taskComposition struct {
	TaskID          string   `json:"task_id"`
	ContractTaskID  string   `json:"contract_task_id"`
	PinnedContract  bool     `json:"pinned_contract"`
	BundleID        string   `json:"bundle_id"`
	GateKeys        []string `json:"gate_keys"`
	Bindings        []string `json:"bindings"`
	AllowedPaths    []string `json:"allowed_paths"`
	RequirementRefs []string `json:"requirement_refs"`
	RiskSignals     []string `json:"risk_signals"`
	RiskBasis       []string `json:"risk_basis"`
	ObjectiveSHA256 string   `json:"objective_sha256"`
}

// composeInputs renders every composed byte for one task (used by both
// `run` and `self-check`). It returns the task contract YAML, verification
// contract YAML, requirement-file contents (path → bytes), and the composition
// record. Inputs: frozen set, frozen task, acceptance doc, verbatim intent,
// probe binary path, owner repo root, migration DSN base.
func composeInputs(set *frozenSet, task *v14product.ManifestTask, acc *acceptanceDoc, intent []byte,
	probePath, repoRoot, dsnBase string) (taskYAML []byte, verifYAML []byte, reqs map[string][]byte, rec *taskComposition, err error) {

	contractID := contractTaskIDFor(set, task.TaskID)
	rec = &taskComposition{
		TaskID:          task.TaskID,
		ContractTaskID:  contractID,
		PinnedContract:  pinnedContractTasks[task.TaskID] != "",
		GateKeys:        append([]string(nil), acc.GateCond.AllOf...),
		ObjectiveSHA256: sha256Hex(intent),
	}

	// ---- TaskContract -----------------------------------------------------
	tc := taskContractYAML{ID: contractID, Profile: "standard"}
	// Objective = the verbatim frozen intent bytes (arm B's contract is
	// generated from the same intent text arm A receives; RUN_PROTOCOL §4).
	tc.Objective = string(intent)
	tc.Title = "Tally closed-loop trial: " + task.TaskID
	if pinned := pinnedContractTasks[task.TaskID]; pinned != "" {
		// Pinned harness contract facts: requirement_refs axis comes from the
		// pinned contract fixture (read verbatim from the frozen bytes).
		refs, err := pinnedRequirementRefs(set, task.TaskID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		tc.RequirementRefs = refs
		rec.RequirementRefs = refs
		reqs = map[string][]byte{}
		for _, r := range refs {
			// Requirement file content = the verbatim frozen intent (the
			// requirement the intent states; no hand-written text).
			reqs[r] = append([]byte(nil), intent...)
		}
	}
	// scope.allowed_paths: the agent-facing baseline surface (deterministic
	// from the pinned fixture paths).
	tc.Scope.AllowedPaths = allowedPathsFor(task, acc)
	rec.AllowedPaths = tc.Scope.AllowedPaths
	for _, a := range acc.Acceptance {
		tc.Acceptance = append(tc.Acceptance, struct {
			ID                   string `yaml:"id"`
			Statement            string `yaml:"statement"`
			IndependenceRequired string `yaml:"independence_required"`
		}{ID: a.ID, Statement: a.Statement, IndependenceRequired: "none"})
	}

	// ---- VerificationContract --------------------------------------------
	// Bindings are exactly the frozen gate keys (acceptance.yaml
	// gate_condition.all_of == the catalog bundle's keys), each driven by the
	// probe through the same frozen acceptance/oracle semantics the arm-neutral
	// judge applies. All bindings are visible (L1 deterministic floor, Decision
	// Lock 21; no held-out content exists in this set — the oracle files stay
	// in the owner repository, reachable only through the probe).
	verif := verificationYAML{TaskID: contractID, Revision: 1}
	bug := acc.Harness == v14product.HarnessBugFix
	for i, key := range acc.GateCond.AllOf {
		stmt, err := statementForGate(acc, key)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		bnd := bindingYAML{
			ID:               key,
			AuthoringMode:    "pre_commit",
			VisibilityPolicy: "visible",
			ExecutionProtocol: protoYAML{
				Repetitions: 1,
				Aggregation: "all_pass",
				Command:     cmdYAML{Timeout: "300s"},
			},
		}
		if bug {
			baseRel := v14product.FixturePath(task.TaskID, "baseline")
			entries, err := os.ReadDir(filepath.Join(set.root, filepath.FromSlash(baseRel)))
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("baseline dir: %w", err)
			}
			var goFile string
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					goFile = baseRel + "/" + e.Name()
					break
				}
			}
			if goFile == "" {
				return nil, nil, nil, nil, fmt.Errorf("task %s: no baseline .go snapshot", task.TaskID)
			}
			bnd.Evidence = evidenceYAML{Type: "test", Path: goFile}
			// Deterministic probe subcommand per frozen key: repro keys are
			// properties of the frozen baseline; after keys judge the
			// delivered candidate (checkBugProbeKeys pins the key order).
			if key == "bug_repro_failed_before" {
				bnd.ExecutionProtocol.Command.Argv = []string{probePath, "repro", "-task", task.TaskID, "-repo", repoRoot}
			} else {
				bnd.ExecutionProtocol.Command.Argv = []string{probePath, "after", "-task", task.TaskID, "-repo", repoRoot}
			}
			// The probe's documented exit-code contract (probe/main.go header):
			// 2 = wiring/infra error. Declared in the frozen protocol so the
			// loop's verifier records it as needs_review, never a trusted FAIL
			// (P6-05 amplifier fix).
			bnd.ExecutionProtocol.Command.InfraExitCodes = []int{2}
		} else {
			bnd.Evidence = evidenceYAML{Type: "schema_check"}
			sub, ok := migrationProbeSubcommands[key]
			if !ok {
				return nil, nil, nil, nil, fmt.Errorf("migration gate key %q has no probe subcommand mapping (frozen-set drift; probe accepts fresh/n1/integrity)", key)
			}
			bnd.ExecutionProtocol.Command.Argv = []string{probePath, sub, "-task", task.TaskID, "-repo", repoRoot, "-dsn", dsnBase}
			bnd.ExecutionProtocol.Command.InfraExitCodes = []int{2}
		}
		rec.Bindings = append(rec.Bindings, key)
		av := struct {
			ID                   string        `yaml:"id"`
			Statement            string        `yaml:"statement"`
			IndependenceRequired string        `yaml:"independence_required"`
			EvidenceBindings     []bindingYAML `yaml:"evidence_bindings"`
		}{
			ID:                   fmt.Sprintf("AC-%02d", i+1),
			Statement:            stmt,
			IndependenceRequired: "none",
			EvidenceBindings:     []bindingYAML{bnd},
		}
		verif.Acceptances = append(verif.Acceptances, av)
	}

	// ---- risk facts (composition record only; the loop risk axis) --------
	rec.RiskSignals = riskSignalsFor(task)
	rec.RiskBasis = riskBasisFor(task)
	rec.BundleID = bundleFor(acc)

	if err := checkBugProbeKeys(acc, bug); err != nil {
		return nil, nil, nil, nil, err
	}

	taskYAML, err = yaml.Marshal(&tc)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	verifYAML, err = yaml.Marshal(&verif)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return taskYAML, verifYAML, reqs, rec, nil
}

// checkBugProbeKeys pins the repro/after assignment for bug_fix tasks: the
// frozen gate keys have fixed semantics (bug_repro_failed_before = frozen
// baseline property; regression_check_passes_after = candidate property).
func checkBugProbeKeys(acc *acceptanceDoc, bug bool) error {
	if !bug {
		return nil
	}
	if len(acc.GateCond.AllOf) != 2 {
		return fmt.Errorf("bug_fix acceptance must pin exactly 2 gate keys, got %v", acc.GateCond.AllOf)
	}
	if acc.GateCond.AllOf[0] != "bug_repro_failed_before" || acc.GateCond.AllOf[1] != "regression_check_passes_after" {
		return fmt.Errorf("unexpected bug_fix gate keys %v", acc.GateCond.AllOf)
	}
	return nil
}

// allowedPathsFor derives scope.allowed_paths deterministically from the
// agent-facing frozen fixtures (bf: the baseline dir; db: the migration dirs
// carrying the delivered chain).
func allowedPathsFor(task *v14product.ManifestTask, acc *acceptanceDoc) []string {
	seen := map[string]bool{}
	var out []string
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		out = append(out, dir+"/**")
	}
	for rel := range task.FixtureHashes {
		if !agentFacingFixture(rel) {
			continue
		}
		if strings.HasSuffix(rel, "intent.md") {
			continue
		}
		add(dirOf(rel))
	}
	sort.Strings(out)
	return out
}

func dirOf(rel string) string {
	i := strings.LastIndex(rel, "/")
	if i <= 0 {
		return ""
	}
	return rel[:i]
}

// riskSignalsFor returns the frozen risk-signal ids the task's facts derive:
// bf-02 carries the pinned auth signal (pinned facts fixture); every
// database_migration task derives the frozen schema_migration signal (the
// frozen oracle layout migrations are the basis); other bug_fix tasks derive
// none (low-risk functional defects, mirror of the pinned TASK-2410 facts).
func riskSignalsFor(task *v14product.ManifestTask) []string {
	switch {
	case task.TaskID == "bf-02-logout-session-invalidation":
		return []string{"auth_symbol_change"}
	case task.Harness == v14product.HarnessMigration:
		return []string{"schema_migration"}
	default:
		return nil
	}
}

// riskBasisFor returns deterministic basis refs for the derived signals
// (citing the frozen fixtures the signal derives from).
func riskBasisFor(task *v14product.ManifestTask) []string {
	if task.Harness == v14product.HarnessMigration {
		var refs []string
		for rel := range task.FixtureHashes {
			if strings.Contains(rel, "/migrations/") {
				refs = append(refs, "fixture:"+rel)
			}
		}
		sort.Strings(refs)
		return refs
	}
	if task.TaskID == "bf-02-logout-session-invalidation" {
		// Mirror of the pinned TASK-2411 facts basis (frozen fixture bytes).
		return []string{"intent:logout-session-still-valid", "fixture:test/eval/v14_product/testdata/tasks/bf-02-logout-session-invalidation/baseline/session_store.go"}
	}
	return nil
}

func bundleFor(acc *acceptanceDoc) string {
	if acc.Harness == v14product.HarnessBugFix {
		return "POL-BUG-FIX-REGRESSION"
	}
	return "database_migration"
}

// pinnedRequirementRefs extracts the requirement_refs axis from the pinned
// harness contract fixture (frozen bytes, read verbatim from the owner repo).
func pinnedRequirementRefs(set *frozenSet, taskID string) ([]string, error) {
	contractRel := pinnedContractFixture(taskID)
	b, err := os.ReadFile(filepath.Join(set.root, filepath.FromSlash(contractRel)))
	if err != nil {
		return nil, fmt.Errorf("pinned contract fixture: %w", err)
	}
	var doc struct {
		ID              string   `yaml:"id"`
		RequirementRefs []string `yaml:"requirement_refs"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("pinned contract %s does not parse: %w", contractRel, err)
	}
	if contractTaskIDFor(set, taskID) != doc.ID {
		return nil, fmt.Errorf("pinned contract %s declares id %q, want %s", contractRel, doc.ID, contractTaskIDFor(set, taskID))
	}
	return doc.RequirementRefs, nil
}

// pinnedContractFixture locates the pinned harness contract fixture for the
// two tasks that reuse P3-01 frozen corpus contracts.
func pinnedContractFixture(taskID string) string {
	switch taskID {
	case "bf-01-token-bucket-offbyone":
		return "test/fixtures/v14/bugfix/contracts/TASK-2410.yaml"
	case "bf-02-logout-session-invalidation":
		return "test/fixtures/v14/bugfix/contracts/TASK-2411.yaml"
	}
	return ""
}
