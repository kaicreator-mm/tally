package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"tally/internal/runner"
)

// isContextErr reports whether err is a cancellation or deadline of the
// command's own context (as opposed to a git-level failure).
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// gitClient executes the git CLI with structured argv (no shell, no implicit
// semantics) so every invocation is cancellable and unambiguous.
type gitClient struct {
	// run executes one structured command. Never nil after newGitClient;
	// replaceable in tests to simulate cancellation and failure paths.
	run func(context.Context, runner.Command) (runner.Result, error)
}

func newGitClient() *gitClient {
	local := runner.LocalRunner{}
	return &gitClient{run: local.Run}
}

// gitEnv isolates git from ambient machine configuration: no global/system
// config files, no credential prompts. Tally controls the configuration every
// command sees and no command can block on interactive input.
func gitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

// git runs one git command with dir as working directory. A non-zero exit
// code is an error carrying the captured stderr; executor-level errors
// (binary missing, context cancellation) pass through unchanged so callers
// can match errors.Is(err, context.Canceled).
func (g *gitClient) git(ctx context.Context, dir string, argv ...string) (runner.Result, error) {
	res, err := g.run(ctx, runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  gitEnv(),
	})
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("git %s failed (exit %d): %s",
			strings.Join(argv, " "), res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res, nil
}

// resolveRevision resolves rev to the full commit SHA it names. The caller's
// revision text is only ever used as a rev-parse input; worktrees are created
// from the resolved commit so no option-like revision can leak into argv
// positions interpreted as flags.
func (g *gitClient) resolveRevision(ctx context.Context, repoDir, rev string) (string, error) {
	res, err := g.git(ctx, repoDir, "rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		if isContextErr(err) {
			return "", err
		}
		return "", fmt.Errorf("%w: %q: %v", ErrUnknownRevision, rev, err)
	}
	return strings.TrimSpace(res.Stdout), nil
}

// worktreeAdd registers and checks out a new detached worktree at path for
// the exact commit. path and commit are manager-generated values (absolute
// path, resolved SHA), never caller-supplied free text.
func (g *gitClient) worktreeAdd(ctx context.Context, repoDir, path, commit string) error {
	_, err := g.git(ctx, repoDir, "worktree", "add", "--detach", path, commit)
	if err != nil {
		return fmt.Errorf("workspace: git worktree add %s at %s: %w", path, commit, err)
	}
	return nil
}

// worktreeRemove unregisters and deletes the worktree at path. It may fail
// because the worktree is already gone; callers verify observed state
// (directory absence, worktree list) instead of matching stderr text.
func (g *gitClient) worktreeRemove(ctx context.Context, repoDir, path string) error {
	_, err := g.git(ctx, repoDir, "worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("workspace: git worktree remove %s: %w", path, err)
	}
	return nil
}

// worktreePrune drops stale administrative metadata for worktrees whose
// directories no longer exist.
func (g *gitClient) worktreePrune(ctx context.Context, repoDir string) error {
	_, err := g.git(ctx, repoDir, "worktree", "prune")
	if err != nil {
		return fmt.Errorf("workspace: git worktree prune: %w", err)
	}
	return nil
}

// worktreeInfo is one entry of `git worktree list --porcelain`.
type worktreeInfo struct {
	// Path is the absolute worktree path as reported by git.
	Path string
	// Head is the commit the worktree has checked out (may be empty).
	Head string
	// Detached reports a detached HEAD (a disposable worktree is always
	// detached).
	Detached bool
}

// worktreeList returns the registered worktrees of the repository checked out
// at repoDir.
func (g *gitClient) worktreeList(ctx context.Context, repoDir string) ([]worktreeInfo, error) {
	res, err := g.git(ctx, repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(res.Stdout), nil
}

// parseWorktreeList parses `git worktree list --porcelain` output. Blocks are
// separated by blank lines and start with a "worktree <path>" line; paths may
// contain spaces.
func parseWorktreeList(out string) []worktreeInfo {
	var entries []worktreeInfo
	cur := worktreeInfo{}
	inBlock := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case line == "":
			if inBlock {
				entries = append(entries, cur)
				cur = worktreeInfo{}
				inBlock = false
			}
		case strings.HasPrefix(line, "worktree "):
			cur = worktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
			inBlock = true
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case line == "detached":
			cur.Detached = true
		case line == "bare":
			cur.Detached = true // no checkout to mutate; treat like detached
		}
	}
	if inBlock {
		entries = append(entries, cur)
	}
	return entries
}

// worktreeRegistered reports whether path is present in entries. Path
// comparison is filesystem-realistic (separators and case normalised) because
// git prints canonical forward-slash paths while the manager builds
// OS-native ones.
func worktreeRegistered(entries []worktreeInfo, path string) bool {
	want := normalizePath(path)
	for _, e := range entries {
		if normalizePath(e.Path) == want {
			return true
		}
	}
	return false
}

// normalizePath canonicalises a path for equality comparisons. Native paths
// use the host filesystem's semantics. Windows-style absolute/UNC paths are
// normalised lexically even when tests or persisted Git output are inspected
// on another OS, so forward-slash Git paths and backslash manager paths still
// compare as the same Windows location.
func normalizePath(p string) string {
	if runtime.GOOS == "windows" || isWindowsStylePath(p) {
		p = strings.ReplaceAll(p, `\`, "/")
		return strings.ToLower(path.Clean(p))
	}
	return filepath.ToSlash(filepath.Clean(p))
}

func isWindowsStylePath(p string) bool {
	if strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//") {
		return true
	}
	if len(p) < 3 || p[1] != ':' || (p[2] != '/' && p[2] != '\\') {
		return false
	}
	return (p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')
}

// samePath reports whether a and b name the same filesystem path.
func samePath(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}
