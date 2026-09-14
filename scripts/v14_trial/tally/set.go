// set.go loads the FROZEN v1.4 task set for the Tally arm (P4-03) with the
// exact discipline of the bare arm (scripts/v14_trial/bare/trialset.go): the
// manifest is byte-pinned to the operative freeze hash, schema-validated and
// fixture-verified via the P4-01 validator package, and arm neutrality is
// re-checked at every load. Any drift refuses to operate on the set.
//
// Freeze history: c4050efe… (P4-01 8b9aad4) superseded pre-trial by the
// freeze-owner adjudication of 2026-09-10 (bf-01 baseline); bbf381be… is the
// operative freeze (TASK_SET.md §11, PRE_REGISTRATION dated note).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tally/test/eval/v14_product"
)

const (
	manifestRelPath   = "docs/evals/v1.4/task_set.json"
	frozenManifestSHA = "bbf381be07a0859328f8f39a3a5cdef4ce453b6fa05a244163438e2b6bcd4e4d"
)

// frozenSet is the loaded, integrity-checked frozen task set.
type frozenSet struct {
	manifest *v14product.Manifest
	raw      []byte
	sha      string
	root     string // repository root (manifest paths are repo-relative)
}

func findRepoRoot(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		if _, err := os.Stat(filepath.Join(explicit, filepath.FromSlash(manifestRelPath))); err != nil {
			return "", fmt.Errorf("--repo %s does not contain %s: %w", explicit, manifestRelPath, err)
		}
		return filepath.Abs(explicit)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(manifestRelPath))); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s (no %s above it)", dir, manifestRelPath)
		}
		dir = parent
	}
}

// loadFrozenSet loads the manifest, pins its freeze hash, validates it against
// the preregistered schema and verifies every pinned fixture on disk.
func loadFrozenSet(repoRoot string) (*frozenSet, error) {
	m, raw, err := v14product.LoadManifest(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("frozen manifest invalid: %w", err)
	}
	sum := sha256.Sum256(raw)
	sha := hex.EncodeToString(sum[:])
	if sha != frozenManifestSHA {
		return nil, fmt.Errorf("task-set manifest changed after freeze: got %s, frozen %s; "+
			"a frozen set is never edited (PRE_REGISTRATION §5, TASK_SET.md §8) — the trial on this set is invalidated", sha, frozenManifestSHA)
	}
	if issues := v14product.VerifyFixtures(repoRoot, m); len(issues) > 0 {
		return nil, fmt.Errorf("frozen fixtures drifted: %v", issues)
	}
	hits, err := v14product.ScanArmLabels(repoRoot, m, raw)
	if err != nil {
		return nil, err
	}
	if len(hits) > 0 {
		return nil, fmt.Errorf("evaluator-facing assets are no longer arm-neutral (P4-01 AC3): %v", hits)
	}
	return &frozenSet{manifest: m, raw: raw, sha: sha, root: repoRoot}, nil
}

func (s *frozenSet) task(taskID string) (*v14product.ManifestTask, error) {
	for i := range s.manifest.Tasks {
		if s.manifest.Tasks[i].TaskID == taskID {
			return &s.manifest.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %q is not in the frozen set", taskID)
}

// taskIndex returns the frozen position of a task (for deterministic trial-side
// contract id derivation).
func (s *frozenSet) taskIndex(taskID string) int {
	for i := range s.manifest.Tasks {
		if s.manifest.Tasks[i].TaskID == taskID {
			return i
		}
	}
	return -1
}

// agentFacingFixture classifies one pinned fixture path (TASK_SET.md §4,
// frozen roles): everything pinned is agent-facing EXCEPT the verification-side
// assets — the frozen acceptance contract, the frozen oracle assets, and the
// db-01 reference repair. Identical predicate to the bare arm.
func agentFacingFixture(rel string) bool {
	if filepath.Base(rel) == "acceptance.yaml" {
		return false
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "oracle" || seg == "repair" {
			return false
		}
	}
	return true
}

// provisionTask materializes the agent-facing frozen fixtures, byte-identical,
// at their repo-relative layout (the shared workspace provisioning parity
// guarantee, RUN_PROTOCOL §4) into wsRoot. It refuses to write into an
// existing non-empty workspace and fails if any verification-side asset would
// land in the workspace.
func provisionTask(set *frozenSet, task *v14product.ManifestTask, wsRoot string) error {
	if entries, err := os.ReadDir(wsRoot); err == nil {
		// An existing EMPTY directory is allowed (the fixture-repo builder
		// runs git init first); any existing content refuses.
		if len(entries) != 0 {
			return fmt.Errorf("workspace %s already exists — provision into a fresh directory", wsRoot)
		}
	}
	if err := os.MkdirAll(wsRoot, 0o755); err != nil {
		return err
	}
	paths := make([]string, 0, len(task.FixtureHashes))
	for p := range task.FixtureHashes {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		want := task.FixtureHashes[rel]
		if !agentFacingFixture(rel) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(set.root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("task %s: read pinned fixture %s: %w", task.TaskID, rel, err)
		}
		if got := sha256Hex(b); got != want {
			return fmt.Errorf("task %s: fixture %s drifted (manifest %s, disk %s)", task.TaskID, rel, want, got)
		}
		dst := filepath.Join(wsRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return fmt.Errorf("task %s: materialize %s: %w", task.TaskID, rel, err)
		}
	}
	return verifyWorkspaceIsolation(set, task, wsRoot)
}

// verifyWorkspaceIsolation proves the shared workspace discipline: every
// agent-facing pinned fixture is present and byte-identical; NO
// verification-side asset exists anywhere in the workspace.
func verifyWorkspaceIsolation(set *frozenSet, task *v14product.ManifestTask, wsRoot string) error {
	for rel, want := range task.FixtureHashes {
		if !agentFacingFixture(rel) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(wsRoot, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("task %s: agent-facing fixture %s missing from workspace: %w", task.TaskID, rel, err)
		}
		if got := sha256Hex(b); got != want {
			return fmt.Errorf("task %s: workspace fixture %s is not byte-identical to the frozen set", task.TaskID, rel)
		}
	}
	err := filepath.Walk(wsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(wsRoot, path)
		if relErr != nil {
			return relErr
		}
		slash := filepath.ToSlash(rel)
		if !agentFacingFixture(slash) {
			return fmt.Errorf("task %s: verification-side asset %s must never reach a workspace", task.TaskID, slash)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// The verbatim instruction bytes must hash to the frozen intent digest.
	_, err = intentBytes(set, task, wsRoot)
	return err
}

// intentBytes reads the workspace intent.md and verifies it is the verbatim
// frozen intent (manifest intent_text_sha256).
func intentBytes(set *frozenSet, task *v14product.ManifestTask, wsRoot string) ([]byte, error) {
	rel := v14product.FixturePath(task.TaskID, "intent.md")
	b, err := os.ReadFile(filepath.Join(wsRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("task %s: read workspace intent: %w", task.TaskID, err)
	}
	pinned := task.FixtureHashes[rel]
	want := task.IntentTextSHA256
	if want == "" {
		want = pinned
	}
	if got := sha256Hex(b); got != want {
		return nil, fmt.Errorf("task %s: intent bytes %s are not the frozen verbatim intent (%s)", task.TaskID, got, want)
	}
	return b, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
