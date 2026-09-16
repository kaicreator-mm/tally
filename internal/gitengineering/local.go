package gitengineering

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"tally/internal/runner"
)

var (
	ErrGitCommand = errors.New("gitengineering: git command failed")
	ErrStaleHead  = errors.New("gitengineering: stale head")
	ErrUnsafeName = errors.New("gitengineering: unsafe git name")
)

// LocalGitAdapter performs structured, non-shell Git operations against a
// checked-out repository. It intentionally does not use ambient credentials or
// config prompting; provider/network failures are returned to the harness.
type LocalGitAdapter struct {
	Runner runner.Runner
}

func NewLocalGitAdapter(r runner.Runner) *LocalGitAdapter {
	if r == nil {
		r = runner.LocalRunner{}
	}
	return &LocalGitAdapter{Runner: r}
}

func localGitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

func (a *LocalGitAdapter) run(ctx context.Context, repoDir string, argv ...string) (runner.Result, error) {
	if a == nil || a.Runner == nil {
		return runner.Result{}, errors.New("gitengineering: local git runner is required")
	}
	res, err := a.Runner.Run(ctx, runner.Command{Argv: append([]string{"git"}, argv...), Dir: repoDir, Env: localGitEnv()})
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%w: git %s (exit %d): %s", ErrGitCommand, strings.Join(argv, " "), res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res, nil
}

func (a *LocalGitAdapter) ResolveRevision(ctx context.Context, repoDir, revision string) (string, error) {
	if strings.TrimSpace(revision) == "" {
		return "", errors.New("gitengineering: revision is required")
	}
	res, err := a.run(ctx, repoDir, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(res.Stdout)
	if sha == "" {
		return "", fmt.Errorf("%w: empty resolved revision", ErrGitCommand)
	}
	return sha, nil
}

func (a *LocalGitAdapter) validateBranch(ctx context.Context, repoDir, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" || strings.HasPrefix(branch, "-") || strings.ContainsAny(branch, "\r\n\x00") {
		return ErrUnsafeName
	}
	_, err := a.run(ctx, repoDir, "check-ref-format", "--branch", branch)
	if err != nil {
		return fmt.Errorf("%w: invalid branch %q: %v", ErrUnsafeName, branch, err)
	}
	return nil
}

func (a *LocalGitAdapter) CreateBranch(ctx context.Context, repoDir, branch, baseRevision string) (string, error) {
	if err := a.validateBranch(ctx, repoDir, branch); err != nil {
		return "", err
	}
	base, err := a.ResolveRevision(ctx, repoDir, baseRevision)
	if err != nil {
		return "", err
	}
	if _, err := a.run(ctx, repoDir, "branch", "--no-track", branch, base); err != nil {
		return "", err
	}
	return base, nil
}

func (a *LocalGitAdapter) Commit(ctx context.Context, repoDir, message string, paths []string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errors.New("gitengineering: commit message is required")
	}
	if len(paths) == 0 {
		return "", errors.New("gitengineering: explicit commit paths are required")
	}
	args := []string{"add", "--"}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" || strings.ContainsRune(p, '\x00') {
			return "", errors.New("gitengineering: invalid commit path")
		}
		args = append(args, p)
	}
	if _, err := a.run(ctx, repoDir, args...); err != nil {
		return "", err
	}
	if _, err := a.run(ctx, repoDir, "commit", "--no-gpg-sign", "-m", message); err != nil {
		return "", err
	}
	return a.ResolveRevision(ctx, repoDir, "HEAD")
}

func (a *LocalGitAdapter) Push(ctx context.Context, repoDir, remote, branch, expectedHeadRevision string) (string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" || strings.HasPrefix(remote, "-") || strings.ContainsAny(remote, "\r\n\x00") {
		return "", ErrUnsafeName
	}
	if err := a.validateBranch(ctx, repoDir, branch); err != nil {
		return "", err
	}
	head, err := a.ResolveRevision(ctx, repoDir, "HEAD")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(expectedHeadRevision) == "" || head != expectedHeadRevision {
		return "", fmt.Errorf("%w: expected %s, observed %s", ErrStaleHead, expectedHeadRevision, head)
	}
	refspec := "HEAD:refs/heads/" + branch
	if _, err := a.run(ctx, repoDir, "push", "--porcelain", remote, refspec); err != nil {
		return "", err
	}
	return head, nil
}
