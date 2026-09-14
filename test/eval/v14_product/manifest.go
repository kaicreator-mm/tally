// Package v14product is the P4-01 validation harness for the FROZEN v1.4
// product value experiment task set (docs/tasks/v1.4/P4/P4-01.md).
//
// It is not production code and it runs no agents. It mechanically enforces,
// before any arm of the trial executes (A0-05 preregistration, PRD §30):
//
//   - the manifest docs/evals/v1.4/task_set.json conforms to the
//     preregistered schema docs/evals/v1.4/task-set-manifest.schema.json
//     (strict field sets, patterns, enums, minimum size, both harness types);
//   - the manifest is byte-frozen: its SHA-256 equals frozenManifestHash
//     (any change after freeze fails TestManifestFrozen, mirroring the
//     corpus freeze discipline of test/eval/v14_assurance);
//   - every pinned fixture exists on disk and hashes to its recorded value;
//   - every task carries stable acceptance/oracle assets whose deterministic
//     expectations actually hold (bug-fix oracles and a deterministic
//     PostgreSQL-dialect static migration analyzer; the live-DB evidence
//     paths of PRD §24.2 are executed at trial time by P4-03, not here);
//   - no evaluator-facing asset embeds trial-arm labels (P4-01 AC3);
//   - the replacement policy of PRE_REGISTRATION §5 is documented in
//     docs/evals/v1.4/TASK_SET.md (P4-01 AC4).
package v14product

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// manifestRelPath is the frozen manifest, relative to the repository root.
const manifestRelPath = "docs/evals/v1.4/task_set.json"

// taskSetDocRelPath documents rationale, exclusions and the replacement
// policy; relative to the repository root.
const taskSetDocRelPath = "docs/evals/v1.4/TASK_SET.md"

// Manifest mirrors docs/evals/v1.4/task-set-manifest.schema.json exactly.
// Unknown fields are rejected (schema additionalProperties:false at the root
// and per task; the validator is deliberately stricter than the schema's
// allowance for extra environment_parity properties — this frozen manifest
// uses none).
type Manifest struct {
	ManifestVersion   int                `json:"manifest_version"`
	FrozenAt          string             `json:"frozen_at"`
	TrialRef          string             `json:"trial_ref"`
	EnvironmentParity *EnvironmentParity `json:"environment_parity,omitempty"`
	Tasks             []ManifestTask     `json:"tasks"`
}

// EnvironmentParity records what must be identical across the trial
// (PRE_REGISTRATION §6); observed values are recorded per result row at
// trial time (Decision Lock 15: actual != desired runtime identity).
type EnvironmentParity struct {
	Runtime      string `json:"runtime"`
	Model        string `json:"model"`
	RepoBaseline string `json:"repo_baseline"`
	OS           string `json:"os,omitempty"`
	GoVersion    string `json:"go_version,omitempty"`
	DBVersion    string `json:"db_version,omitempty"`
}

// ManifestTask is one frozen evaluation task.
type ManifestTask struct {
	TaskID             string            `json:"task_id"`
	Harness            string            `json:"harness"`
	Difficulty         string            `json:"difficulty"`
	InclusionRationale string            `json:"inclusion_rationale"`
	FixtureHashes      map[string]string `json:"fixture_hashes"`
	AcceptanceFrozen   string            `json:"acceptance_frozen"`
	IntentTextSHA256   string            `json:"intent_text_sha256,omitempty"`
	Excluded           *bool             `json:"excluded,omitempty"`
	ExclusionReason    string            `json:"exclusion_reason,omitempty"`
}

// Harness type values (PRD §30.2 first round — no scope expansion).
const (
	HarnessBugFix    = "bug_fix"
	HarnessMigration = "database_migration"
)

var (
	taskIDRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	sha256Re    = regexp.MustCompile(`^[a-f0-9]{64}$`)
	minTasks    = 10 // PRE_REGISTRATION §5 minimum first-round size
	minPerType  = 5  // PRE_REGISTRATION §5: ≥5 per harness type
	rationaleRe = regexp.MustCompile(`^.{10,}$`)
)

// LoadManifest reads and strictly decodes the manifest at repoRoot.
func LoadManifest(repoRoot string) (*Manifest, []byte, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(manifestRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, nil, fmt.Errorf("manifest does not conform to the preregistered schema: %w", err)
	}
	// Reject trailing data (schema: single object).
	if dec.More() {
		return nil, nil, errors.New("manifest contains trailing data after the JSON object")
	}
	return &m, raw, nil
}

// Validate enforces the preregistered schema semantics beyond decoding.
func (m *Manifest) Validate() error {
	if m.ManifestVersion != 1 {
		return fmt.Errorf("manifest_version must be 1, got %d", m.ManifestVersion)
	}
	if _, err := time.Parse(time.RFC3339, m.FrozenAt); err != nil {
		return fmt.Errorf("frozen_at is not RFC3339 date-time: %w", err)
	}
	if strings.TrimSpace(m.TrialRef) == "" {
		return errors.New("trial_ref must record the git commit the manifest was frozen at")
	}
	if m.EnvironmentParity != nil {
		for _, k := range []struct{ name, v string }{
			{"runtime", m.EnvironmentParity.Runtime},
			{"model", m.EnvironmentParity.Model},
			{"repo_baseline", m.EnvironmentParity.RepoBaseline},
		} {
			if strings.TrimSpace(k.v) == "" {
				return fmt.Errorf("environment_parity.%s is required and must be non-empty", k.name)
			}
		}
	}
	if len(m.Tasks) < minTasks {
		return fmt.Errorf("task set below preregistered minimum: %d < %d (PRE_REGISTRATION §5)", len(m.Tasks), minTasks)
	}
	seen := map[string]bool{}
	perType := map[string]int{}
	for i := range m.Tasks {
		t := &m.Tasks[i]
		if seen[t.TaskID] {
			return fmt.Errorf("duplicate task_id %q", t.TaskID)
		}
		seen[t.TaskID] = true
		if !taskIDRe.MatchString(t.TaskID) {
			return fmt.Errorf("task %d: task_id %q violates schema pattern", i, t.TaskID)
		}
		if t.Harness != HarnessBugFix && t.Harness != HarnessMigration {
			return fmt.Errorf("task %s: harness %q outside preregistered set (PRD §30.2 first round)", t.TaskID, t.Harness)
		}
		perType[t.Harness]++
		switch t.Difficulty {
		case "S", "M", "L":
		default:
			return fmt.Errorf("task %s: difficulty %q not in S/M/L", t.TaskID, t.Difficulty)
		}
		if !rationaleRe.MatchString(t.InclusionRationale) {
			return fmt.Errorf("task %s: inclusion_rationale below schema minLength 10", t.TaskID)
		}
		if len(t.FixtureHashes) < 1 {
			return fmt.Errorf("task %s: fixture_hashes must pin at least one fixture", t.TaskID)
		}
		for p, h := range t.FixtureHashes {
			if !sha256Re.MatchString(h) {
				return fmt.Errorf("task %s: fixture hash for %s is not a sha256 hex digest", t.TaskID, p)
			}
		}
		if !sha256Re.MatchString(t.AcceptanceFrozen) {
			return fmt.Errorf("task %s: acceptance_frozen is not a sha256 hex digest", t.TaskID)
		}
		if t.IntentTextSHA256 != "" && !sha256Re.MatchString(t.IntentTextSHA256) {
			return fmt.Errorf("task %s: intent_text_sha256 is not a sha256 hex digest", t.TaskID)
		}
		if t.Excluded != nil && *t.Excluded && strings.TrimSpace(t.ExclusionReason) == "" {
			return fmt.Errorf("task %s: excluded=true requires a PRE_REGISTRATION §5 exclusion_reason", t.TaskID)
		}
	}
	if perType[HarnessBugFix] < minPerType || perType[HarnessMigration] < minPerType {
		return fmt.Errorf("preregistration requires ≥%d tasks per harness type, got %v", minPerType, perType)
	}
	return nil
}

// TaskByHarness counts tasks per harness type.
func (m *Manifest) TaskByHarness() map[string]int {
	out := map[string]int{}
	for i := range m.Tasks {
		out[m.Tasks[i].Harness]++
	}
	return out
}

// FixturePath returns the conventional per-task fixture path.
func FixturePath(taskID, rel string) string {
	return "test/eval/v14_product/testdata/tasks/" + taskID + "/" + rel
}

// HashFile returns the sha256 of the file at repoRoot/rel (slash path).
func HashFile(repoRoot, rel string) (string, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
