package v14product

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// bugOracleDoc is the frozen oracle/oracle.json for a bug_fix task.
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

func readTaskAsset(t *testing.T, root, taskID, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(FixturePath(taskID, rel))))
	if err != nil {
		t.Fatalf("task %s: missing frozen asset %s: %v", taskID, rel, err)
	}
	return b
}

// TestBugFixOraclesReproduceFrozenExpectations executes every bug_fix oracle
// against its registered baseline/reference behaviors. It proves the two
// preregistration properties per task: the shipped baseline reproducibly
// fails (exactly the frozen failure set — bug_repro_failed_before) and the
// reference semantics pass every case (stable oracle, P4-01 AC2).
func TestBugFixOraclesReproduceFrozenExpectations(t *testing.T) {
	m, _, root := loadFrozenManifest(t)
	checked := 0
	for i := range m.Tasks {
		task := &m.Tasks[i]
		if task.Harness != HarnessBugFix {
			continue
		}
		oracle, ok := OracleForBugTask(task.TaskID)
		if !ok {
			t.Errorf("task %s: no deterministic oracle registered", task.TaskID)
			continue
		}
		var doc bugOracleDoc
		if err := json.Unmarshal(readTaskAsset(t, root, task.TaskID, "oracle/oracle.json"), &doc); err != nil {
			t.Fatalf("task %s: oracle.json does not parse: %v", task.TaskID, err)
		}
		if len(doc.Cases) < 2 {
			t.Errorf("task %s: unstable oracle: fewer than 2 frozen cases", task.TaskID)
		}
		names := map[string]bool{}
		var failed []string
		var failedAgain []string
		referenceAllPass := true
		referenceAllPassAgain := true
		for _, c := range doc.Cases {
			if names[c.Name] {
				t.Errorf("task %s: duplicate oracle case name %q", task.TaskID, c.Name)
			}
			names[c.Name] = true
			// Determinism: evaluate twice; results must be identical.
			b1, b2 := oracle.baseline(c.In), oracle.baseline(c.In)
			if b1 != b2 {
				t.Errorf("task %s: baseline nondeterministic on %q", task.TaskID, c.Name)
			}
			r1, r2 := oracle.reference(c.In), oracle.reference(c.In)
			if r1 != r2 {
				t.Errorf("task %s: reference nondeterministic on %q", task.TaskID, c.Name)
			}
			if b1 != c.Want {
				failed = append(failed, c.Name)
			}
			if r1 != c.Want {
				referenceAllPass = false
			}
			if b2 != c.Want {
				failedAgain = append(failedAgain, c.Name)
			}
			if r2 != c.Want {
				referenceAllPassAgain = false
			}
		}
		sort.Strings(failed)
		sort.Strings(failedAgain)
		if !reflect.DeepEqual(failed, failedAgain) || referenceAllPass != referenceAllPassAgain {
			t.Errorf("task %s: oracle evaluation not deterministic across runs", task.TaskID)
		}
		want := append([]string(nil), doc.Expected.BaselineFailures...)
		sort.Strings(want)
		if !reflect.DeepEqual(failed, want) {
			t.Errorf("task %s: baseline failures %v != frozen expectation %v (bug_repro_failed_before)", task.TaskID, failed, want)
		}
		if doc.Expected.ReferencePassesAll != referenceAllPass {
			t.Errorf("task %s: reference passes=%v != frozen expectation %v", task.TaskID, referenceAllPass, doc.Expected.ReferencePassesAll)
		}
		if !referenceAllPass {
			t.Errorf("task %s: oracle unstable — reference behavior fails frozen cases", task.TaskID)
		}
		checked++
	}
	if checked < minPerType {
		t.Fatalf("only %d bug_fix oracles checked; manifest coverage regressed", checked)
	}
}

// TestMigrationOraclesReproduceFrozenExpectations runs the deterministic
// static analyzer over every database_migration chain for both evidence
// paths and both candidates (shipped + reference repair), comparing the
// derived evidence keys and violation sets against the frozen expectations.
// The live-DB evidence execution belongs to the trial (P4-03); this proves
// the frozen oracle is coherent before any arm runs.
func TestMigrationOraclesReproduceFrozenExpectations(t *testing.T) {
	m, _, root := loadFrozenManifest(t)
	checked := 0
	for i := range m.Tasks {
		task := &m.Tasks[i]
		if task.Harness != HarnessMigration {
			continue
		}
		var doc MigrationOracle
		if err := json.Unmarshal(readTaskAsset(t, root, task.TaskID, "oracle/oracle.json"), &doc); err != nil {
			t.Fatalf("task %s: oracle.json does not parse: %v", task.TaskID, err)
		}
		// Layout integrity: referenced SQL must be the pinned fixtures.
		pinned := func(p string) {
			t.Helper()
			if _, ok := task.FixtureHashes[p]; !ok {
				t.Errorf("task %s: oracle references unpinned fixture %s", task.TaskID, p)
			}
		}
		for _, p := range doc.Layout.Migrations {
			pinned(p)
		}
		pinned(doc.Layout.Seed)
		pinned(doc.Layout.ReferenceRepair)

		baseDerived, err := doc.derive(root, false)
		if err != nil {
			t.Fatalf("task %s: baseline derivation failed: %v", task.TaskID, err)
		}
		refDerived, err := doc.derive(root, true)
		if err != nil {
			t.Fatalf("task %s: reference derivation failed: %v", task.TaskID, err)
		}
		// Determinism: derive twice, compare.
		baseAgain, _ := doc.derive(root, false)
		refAgain, _ := doc.derive(root, true)
		if !reflect.DeepEqual(baseDerived, baseAgain) || !reflect.DeepEqual(refDerived, refAgain) {
			t.Errorf("task %s: migration oracle nondeterministic across runs", task.TaskID)
		}

		for _, pair := range []struct {
			label     string
			got, want *pathEvidence
		}{
			{"baseline", baseDerived, doc.Expected.Baseline.normalized()},
			{"reference", refDerived, doc.Expected.Reference.normalized()},
		} {
			for _, key := range []string{EvFresh, EvNMinus1, EvData} {
				if got := pair.got.keys[key]; got != pair.want.keys[key] {
					t.Errorf("task %s %s: evidence %s = %q, frozen expectation %q (violations derived: %v)",
						task.TaskID, pair.label, key, got, pair.want.keys[key], pair.got.viols[key])
				}
				if !reflect.DeepEqual(pair.got.viols[key], pair.want.viols[key]) {
					t.Errorf("task %s %s: violations for %s = %v, frozen expectation %v",
						task.TaskID, pair.label, key, pair.got.viols[key], pair.want.viols[key])
				}
			}
		}
		// The shipped candidate must be repairable (else the task is not a
		// defect task), and the repair candidate must satisfy the full gate.
		if baseDerived.keys[EvFresh] != "fail" && baseDerived.keys[EvNMinus1] != "fail" && baseDerived.keys[EvData] != "fail" {
			t.Errorf("task %s: shipped chain passes every evidence key — no reproducible failing baseline", task.TaskID)
		}
		for _, key := range []string{EvFresh, EvNMinus1, EvData} {
			if refDerived.keys[key] != "pass" {
				t.Errorf("task %s: reference repair fails %s — unstable oracle", task.TaskID, key)
			}
		}
		checked++
	}
	if checked < minPerType {
		t.Fatalf("only %d database_migration oracles checked; manifest coverage regressed", checked)
	}
}
