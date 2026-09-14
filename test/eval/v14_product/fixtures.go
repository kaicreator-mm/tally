package v14product

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// FixtureIssue describes one frozen-fixture integrity violation.
type FixtureIssue struct {
	TaskID string
	Detail string
}

func (i FixtureIssue) Error() string { return i.TaskID + ": " + i.Detail }

// VerifyFixtures checks, for every frozen task:
//   - every pinned fixture path exists on disk and hashes to its recorded
//     sha256 (no fixture drift after freeze);
//   - the conventional intent.md / acceptance.yaml assets are pinned and
//     their recorded hashes (intent_text_sha256 / acceptance_frozen) match;
//   - intent and acceptance live under the task's own testdata directory.
func VerifyFixtures(repoRoot string, m *Manifest) []error {
	var issues []error
	for i := range m.Tasks {
		t := &m.Tasks[i]
		for p, want := range t.FixtureHashes {
			got, err := HashFile(repoRoot, p)
			if err != nil {
				issues = append(issues, FixtureIssue{t.TaskID, fmt.Sprintf("fixture %s unreadable: %v", p, err)})
				continue
			}
			if got != want {
				issues = append(issues, FixtureIssue{t.TaskID, fmt.Sprintf("fixture %s hash drift: manifest %s, disk %s", p, want, got)})
			}
		}
		intentRel := FixturePath(t.TaskID, "intent.md")
		acceptRel := FixturePath(t.TaskID, "acceptance.yaml")
		intentHash, okIntent := t.FixtureHashes[intentRel]
		acceptHash, okAccept := t.FixtureHashes[acceptRel]
		if !okIntent {
			issues = append(issues, FixtureIssue{t.TaskID, "verbatim intent text not pinned at " + intentRel})
		} else if t.IntentTextSHA256 != "" && t.IntentTextSHA256 != intentHash {
			issues = append(issues, FixtureIssue{t.TaskID, "intent_text_sha256 does not match pinned intent.md hash"})
		}
		if !okAccept {
			issues = append(issues, FixtureIssue{t.TaskID, "frozen acceptance not pinned at " + acceptRel})
		} else if t.AcceptanceFrozen != acceptHash {
			issues = append(issues, FixtureIssue{t.TaskID, "acceptance_frozen does not match pinned acceptance.yaml hash"})
		}
	}
	return issues
}

// armLabelRe matches trial-arm labels (PRD §30.1: Bare vs Same Agent + Tally
// closed loop) inside evaluator-facing assets. Assets must be arm-neutral:
// the same bytes are served to every trial participant and judge.
var (
	armLabelRe = regexp.MustCompile(`(?i)\barms?\s*[-_ \t]\s*[ab]\b`)
	bareArmRe  = regexp.MustCompile(`(?i)\bbare\s+(agent|arm)\b`)
)

// ScanArmLabels returns every evaluator-facing asset (all pinned fixtures
// plus the manifest itself) that embeds an arm label, with the matched text.
// docs/evals/v1.4/TASK_SET.md is deliberately out of scope: it is trial
// documentation, not an evaluator-facing asset.
func ScanArmLabels(repoRoot string, m *Manifest, manifestRaw []byte) ([]string, error) {
	var hits []string
	scan := func(name string, content []byte) {
		for _, re := range []*regexp.Regexp{armLabelRe, bareArmRe} {
			for _, match := range re.FindAllString(string(content), -1) {
				hits = append(hits, fmt.Sprintf("%s: arm label %q", name, match))
			}
		}
	}
	scan(manifestRelPath, manifestRaw)
	for i := range m.Tasks {
		t := &m.Tasks[i]
		paths := make([]string, 0, len(t.FixtureHashes))
		for p := range t.FixtureHashes {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(p)))
			if err != nil {
				return nil, fmt.Errorf("arm-label scan: %v", err)
			}
			scan(p, b)
		}
	}
	return hits, nil
}

// FreezeInfo summarizes the frozen set for reporting.
func FreezeInfo(m *Manifest) string {
	byType := m.TaskByHarness()
	return fmt.Sprintf("frozen_at=%s trial_ref=%s tasks=%d (bug_fix=%d, database_migration=%d)",
		m.FrozenAt, m.TrialRef, len(m.Tasks), byType[HarnessBugFix], byType[HarnessMigration])
}
