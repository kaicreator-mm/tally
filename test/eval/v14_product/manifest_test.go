package v14product

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// frozenManifestHash is the SHA-256 of docs/evals/v1.4/task_set.json at the
// moment of freeze. This constant is the mechanical freeze enforcement
// demanded by PRE_REGISTRATION §5 / PRD §30.2: any byte change to the
// manifest after this commit makes TestManifestFrozen fail. A frozen set is
// never edited; a new trial requires a NEW pre-registered manifest whose
// hash is pinned here in a fresh commit (see docs/evals/v1.4/TASK_SET.md
// "Task replacement policy").
//
// Freeze history:
//   - 2026-09-09 (P4-01, 8b9aad4): c4050efe1fe1d6017bcf310333698f2dcdf8a7ee76dd86561dad774c9c9f7420
//   - 2026-09-10 (freeze-owner adjudication, pre-trial): bf-01 baseline
//     snapshot was byte-pinned as FIXED code (all 6 oracle cases passed —
//     discovered by the P4-02 bare-runner self-check, 6799b09); the trial had
//     NOT started (no P4-03 run), so per TASK_SET.md §8 the freeze was
//     superseded by this re-freeze with the baseline repaired to the intended
//     off-by-one defect (fails exactly the frozen baseline_failures set;
//     acceptance/oracle/intent bytes unchanged). No results existed to
//     invalidate; the defect discovery and both hashes are recorded here and
//     in TASK_SET.md / PRE_REGISTRATION.md dated notes.
const frozenManifestHash = "bbf381be07a0859328f8f39a3a5cdef4ce453b6fa05a244163438e2b6bcd4e4d"

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func loadFrozenManifest(t *testing.T) (*Manifest, []byte, string) {
	t.Helper()
	root := repoRoot(t)
	m, raw, err := LoadManifest(root)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest violates preregistered schema semantics: %v", err)
	}
	return m, raw, root
}

// TestManifestFrozen mechanically enforces the freeze (P4-01 AC1): the
// manifest bytes on disk must equal the hash pinned at freeze time.
func TestManifestFrozen(t *testing.T) {
	_, raw, _ := loadFrozenManifest(t)
	sum := sha256.Sum256(raw)
	got := hex.EncodeToString(sum[:])
	if got != frozenManifestHash {
		t.Fatalf("task-set manifest changed after freeze: got %s, frozen %s.\n"+
			"The frozen set must never be edited. A different set is a NEW preregistration: "+
			"re-register under A0-05 rules and pin the new hash in a fresh commit, "+
			"or the trial is invalidated (PRE_REGISTRATION §5).", got, frozenManifestHash)
	}
}

// TestManifestSchemaConformance asserts the preregistered schema fields that
// Validate cannot express as one error (const version, date-time format,
// parity identity, freeze completeness: no task excluded at freeze).
func TestManifestSchemaConformance(t *testing.T) {
	m, _, _ := loadFrozenManifest(t)
	if m.ManifestVersion != 1 {
		t.Fatalf("manifest_version = %d, want 1", m.ManifestVersion)
	}
	if !strings.HasSuffix(m.FrozenAt, "Z") {
		t.Fatalf("frozen_at %q must be UTC RFC3339", m.FrozenAt)
	}
	if m.EnvironmentParity == nil {
		t.Fatal("environment_parity must be pinned (PRE_REGISTRATION §6)")
	}
	for i := range m.Tasks {
		if m.Tasks[i].Excluded != nil && *m.Tasks[i].Excluded {
			t.Fatalf("task %s excluded at freeze; exclusions apply only mid-trial with a §5 reason", m.Tasks[i].TaskID)
		}
	}
}

// TestTaskIDsUniqueAndCoverage checks manifest internal consistency: unique
// ids, minimum size, and both preregistered scenario types covered (P4-01).
func TestTaskIDsUniqueAndCoverage(t *testing.T) {
	m, _, _ := loadFrozenManifest(t)
	seen := map[string]bool{}
	for i := range m.Tasks {
		id := m.Tasks[i].TaskID
		if seen[id] {
			t.Errorf("duplicate task_id %q", id)
		}
		seen[id] = true
	}
	byType := m.TaskByHarness()
	if len(m.Tasks) < minTasks {
		t.Errorf("task set size %d below preregistered minimum %d", len(m.Tasks), minTasks)
	}
	if byType[HarnessBugFix] < minPerType {
		t.Errorf("bug_fix coverage %d below minimum %d", byType[HarnessBugFix], minPerType)
	}
	if byType[HarnessMigration] < minPerType {
		t.Errorf("database_migration coverage %d below minimum %d", byType[HarnessMigration], minPerType)
	}
	t.Logf("frozen set: %s", FreezeInfo(m))
}

// TestFixtureHashesMatchDisk pins every fixture: files exist and their
// content hashes equal the manifest values (P4-01, deterministic fixtures).
func TestFixtureHashesMatchDisk(t *testing.T) {
	m, _, root := loadFrozenManifest(t)
	for _, err := range VerifyFixtures(root, m) {
		t.Error(err)
	}
}

// TestNoArmLabelsInEvaluatorAssets enforces P4-01 AC3: neither the manifest
// nor any pinned fixture names a trial arm; the identical assets are served
// to every participant and judge.
func TestNoArmLabelsInEvaluatorAssets(t *testing.T) {
	m, raw, root := loadFrozenManifest(t)
	hits, err := ScanArmLabels(root, m, raw)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, h := range hits {
		t.Error(h)
	}
}

// acceptanceDoc is the frozen acceptance contract shape (YAML).
type acceptanceDoc struct {
	TaskID        string   `yaml:"task_id"`
	Harness       string   `yaml:"harness"`
	PolicyRef     string   `yaml:"policy_ref"`
	Requires      []string `yaml:"requires"`
	GateCondition struct {
		AllOf []string `yaml:"all_of"`
	} `yaml:"gate_condition"`
	Acceptance []struct {
		ID        string `yaml:"id"`
		Statement string `yaml:"statement"`
		OracleRef string `yaml:"oracle_ref,omitempty"`
	} `yaml:"acceptance"`
}

// Policy gate keys of the real catalog bundles the trial exercises.
var policyGates = map[string]map[string]bool{
	HarnessBugFix: {"bug_repro_failed_before": true, "regression_check_passes_after": true},
	HarnessMigration: {
		EvFresh: true, EvNMinus1: true, EvData: true,
	},
}

// TestAcceptanceContractsStableAndPolicyBound enforces P4-01 AC2: every task
// carries a parseable frozen acceptance contract whose gate is exactly the
// gate of the catalog policy it references, with at least one acceptance
// clause and a resolvable oracle reference.
func TestAcceptanceContractsStableAndPolicyBound(t *testing.T) {
	m, _, root := loadFrozenManifest(t)
	for i := range m.Tasks {
		task := &m.Tasks[i]
		path := filepath.Join(root, filepath.FromSlash(FixturePath(task.TaskID, "acceptance.yaml")))
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("task %s: frozen acceptance unreadable: %v", task.TaskID, err)
			continue
		}
		var doc acceptanceDoc
		if err := yaml.Unmarshal(b, &doc); err != nil {
			t.Errorf("task %s: acceptance.yaml does not parse: %v", task.TaskID, err)
			continue
		}
		if doc.TaskID != task.TaskID {
			t.Errorf("task %s: acceptance task_id %q mismatch", task.TaskID, doc.TaskID)
		}
		if doc.Harness != task.Harness {
			t.Errorf("task %s: acceptance harness %q mismatch", task.TaskID, doc.Harness)
		}
		if doc.PolicyRef == "" {
			t.Errorf("task %s: acceptance must reference its catalog policy", task.TaskID)
		}
		want := policyGates[task.Harness]
		got := map[string]bool{}
		for _, k := range doc.GateCondition.AllOf {
			got[k] = true
		}
		if len(got) != len(want) {
			t.Errorf("task %s: gate %v does not match policy gate %v", task.TaskID, doc.GateCondition.AllOf, doc.Requires)
		}
		for k := range got {
			if !want[k] {
				t.Errorf("task %s: gate key %q outside policy gate", task.TaskID, k)
			}
		}
		for _, k := range doc.Requires {
			if !want[k] {
				t.Errorf("task %s: requires key %q outside policy requires", task.TaskID, k)
			}
		}
		if len(doc.Acceptance) == 0 {
			t.Errorf("task %s: acceptance has no clauses", task.TaskID)
		}
		for _, cl := range doc.Acceptance {
			if strings.TrimSpace(cl.Statement) == "" {
				t.Errorf("task %s: clause %s has empty statement", task.TaskID, cl.ID)
			}
			if cl.OracleRef != "" {
				oracle := FixturePath(task.TaskID, cl.OracleRef)
				if _, ok := task.FixtureHashes[oracle]; !ok {
					t.Errorf("task %s: clause %s oracle %s not pinned in fixture_hashes", task.TaskID, cl.ID, oracle)
				}
			}
		}
	}
}

// TestReplacementPolicyDocumented enforces P4-01 AC4: the committed task-set
// document states the replacement policy and records the freeze hash of the
// manifest it describes.
func TestReplacementPolicyDocumented(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(taskSetDocRelPath)))
	if err != nil {
		t.Fatalf("%s missing: %v", taskSetDocRelPath, err)
	}
	doc := string(b)
	hasHeading := func(phrase string) bool {
		for _, line := range strings.Split(doc, "\n") {
			if strings.HasPrefix(line, "## ") && strings.Contains(line, phrase) {
				return true
			}
		}
		return false
	}
	if !hasHeading("Task replacement policy") {
		t.Error("TASK_SET.md must contain a 'Task replacement policy' section")
	}
	if !hasHeading("Exclusions") {
		t.Error("TASK_SET.md must contain an 'Exclusions' section")
	}
	if frozenManifestHash != "PENDING_FREEZE_HASH" && !strings.Contains(doc, frozenManifestHash) {
		t.Error("TASK_SET.md must record the frozen manifest sha256")
	}
}
