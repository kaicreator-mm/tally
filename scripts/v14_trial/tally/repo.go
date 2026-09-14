// repo.go builds the per-task disposable fixture repository the Tally loop
// executes against (layout rationale in compose.go): two commits, agent base
// first, verification authority second (HEAD). All Git access is structured
// argv with an isolated environment (no host git config leaks).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/runner"
	"tally/test/eval/v14_product"
)

func gitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

func git(dir string, argv ...string) (string, error) {
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  gitEnv(),
	})
	if err != nil {
		return "", fmt.Errorf("git %v: %w", argv, err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("git %v: exit %d: %s", argv, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return strings.TrimSpace(res.Stdout), nil
}

func writeRepoFile(repo, rel string, content []byte) error {
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, content, 0o644)
}

const projectYAMLTemplate = `project: %s
entities:
  - id: trial-baseline
    type: fixture_baseline
    path: %s
`

// buildTrialRepo materializes the per-task fixture repository. Returns the
// agent base revision (commit 1) and the HEAD revision (commit 2, carrying the
// verification authority).
func buildTrialRepo(set *frozenSet, task *v14product.ManifestTask, acc *acceptanceDoc, rec *taskComposition,
	taskYAML, verifYAML []byte, reqs map[string][]byte, repoDir string) (baseRev, headRev string, err error) {

	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return "", "", err
	}
	// Provision FIRST (into the empty directory), then git init: provisioning
	// refuses any pre-existing content.
	if err := provisionTask(set, task, repoDir); err != nil {
		return "", "", fmt.Errorf("provision agent-facing fixtures: %w", err)
	}
	if _, err := git(repoDir, "init", "-b", "main"); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "config", "user.name", "tally-v14-trial"); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "config", "user.email", "tally-v14-trial@example.invalid"); err != nil {
		return "", "", err
	}

	// Commit 1: agent base — frozen agent-facing fixtures + desired-side
	// .tally artifacts (project identity, task contract, requirement files,
	// catalog policy bundle). The verification contract is NOT in this tree.
	entityPath := entityPathFor(task)
	if err := writeRepoFile(repoDir, ".tally/project.yaml", []byte(fmt.Sprintf(projectYAMLTemplate, projectIDFor(rec), entityPath))); err != nil {
		return "", "", err
	}
	contractRel := ".tally/tasks/" + rec.ContractTaskID + ".yaml"
	if err := writeRepoFile(repoDir, contractRel, taskYAML); err != nil {
		return "", "", err
	}
	for rel, content := range reqs {
		if err := writeRepoFile(repoDir, rel, content); err != nil {
			return "", "", err
		}
	}
	bundleRel, err := bundleRelFor(set, acc)
	if err != nil {
		return "", "", err
	}
	bundleBytes, err := os.ReadFile(filepath.Join(set.root, filepath.FromSlash(bundleRel)))
	if err != nil {
		return "", "", fmt.Errorf("catalog bundle: %w", err)
	}
	if err := writeRepoFile(repoDir, ".tally/policies/"+filepath.Base(bundleRel), bundleBytes); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "add", "."); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "commit", "-m", "v14 trial agent base ("+task.TaskID+")"); err != nil {
		return "", "", err
	}
	baseRev, err = git(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}

	// Commit 2 (HEAD): the verification authority. Implementation-invisible by
	// construction: both the execution worktree and the one-shot verification
	// workspace are created from the base revision.
	if err := writeRepoFile(repoDir, ".tally/verifications/"+rec.ContractTaskID+".yaml", verifYAML); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "add", "."); err != nil {
		return "", "", err
	}
	if _, err := git(repoDir, "commit", "-m", "v14 trial verification authority ("+task.TaskID+")"); err != nil {
		return "", "", err
	}
	headRev, err = git(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	return baseRev, headRev, nil
}

func projectIDFor(rec *taskComposition) string {
	n := strings.TrimPrefix(rec.ContractTaskID, "TASK-")
	return "proj-v14t-" + n
}

func entityPathFor(task *v14product.ManifestTask) string {
	for rel := range task.FixtureHashes {
		if agentFacingFixture(rel) && !strings.HasSuffix(rel, "intent.md") {
			return dirOf(rel)
		}
	}
	return "."
}

func bundleRelFor(set *frozenSet, acc *acceptanceDoc) (string, error) {
	if acc.Harness == v14product.HarnessBugFix {
		return "catalog/v14/policies/bug_fix.yaml", nil
	}
	if acc.Harness == v14product.HarnessMigration {
		return "catalog/v14/policies/database_migration.yaml", nil
	}
	return "", fmt.Errorf("harness %q outside the preregistered set", acc.Harness)
}

// nowUTC is a small deterministic timestamp helper for records.
func nowUTC() time.Time { return time.Now().UTC() }
