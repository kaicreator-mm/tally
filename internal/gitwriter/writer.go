// Package gitwriter is the single Git-write boundary for Tally production
// repositories (PRD v1.3 §23.1 "Single Git Writer", DECISION_LOCKS #5).
//
// Every commit made through this package:
//
//   - serializes against all other writers for the same repository, both
//     in-process (per-repo mutex) and cross-process (a lock file inside the
//     repository git dir), while different repositories never block one
//     another;
//   - validates an explicit expected base revision before mutating anything;
//   - applies its files atomically: on any failure the work tree and index
//     are rolled back to their pre-call state and no commit is created;
//   - releases both locks on every return path, including context
//     cancellation and errors.
//
// Git is invoked as the git CLI with structured argv, cwd and context (no
// shell, no implicit semantics), so commands are cancellable and their
// inputs unambiguous.
//
// P0-06 scope: this is the foundation only. Existing git-writing modules
// (harvest catalog, knowledge store, requirement signing, content store) do
// not route through it yet; migrating their callers is future work. Until
// that migration lands, those modules keep writing git unserialized.
package gitwriter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Sentinel errors. Match with errors.Is, never by string.
var (
	// ErrNotRepository: the target path is not inside a git repository.
	ErrNotRepository = errors.New("gitwriter: path is not a git repository")
	// ErrBaseRevision: the repository HEAD moved away from the base the
	// caller authored against. Rebase the change and retry with the new base.
	ErrBaseRevision = errors.New("gitwriter: head does not match the expected base revision")
	// ErrNoChanges: the requested files match HEAD; no commit was created.
	ErrNoChanges = errors.New("gitwriter: nothing to commit")
)

// Signature is a commit identity. It is always supplied explicitly (per
// request, per writer, or via the package default) so commits never depend
// on ambient git configuration.
type Signature struct {
	Name  string
	Email string
}

// defaultSignature is used when neither the request nor the writer sets an
// identity: a fixed local identity instead of failing on machines without
// git user config.
var defaultSignature = Signature{Name: "tally", Email: "tally@tally.local"}

// FileUpdate is one repository-relative path included in a Commit.
type FileUpdate struct {
	// Path is relative to the repository root, slash-separated, and must
	// stay inside the repository (".." escapes, absolute paths and drive
	// letters are rejected).
	Path string
	// Content is the full file content written before staging.
	Content []byte
	// Delete removes Path from the work tree and the commit instead of
	// writing Content. Removing an already absent path is not an error.
	Delete bool
}

// CommitRequest describes one atomic, base-validated commit.
type CommitRequest struct {
	// ExpectedBase is the HEAD commit the caller authored the change
	// against. The commit is rejected with ErrBaseRevision unless the
	// repository's current HEAD equals it at commit time. The empty string
	// asserts the repository has no commits yet (unborn HEAD). Use
	// Writer.Head to read the current value.
	ExpectedBase string

	// Message is required and stored verbatim as the commit message.
	Message string

	// Files is the set of paths this commit touches; at least one is
	// required, and each path may appear at most once.
	Files []FileUpdate

	// Signature overrides the writer's commit identity when non-nil.
	Signature *Signature
}

// CommitResult reports the validated base and the created commit.
type CommitResult struct {
	// Base is the validated base revision ("" when the repository had no
	// commits yet).
	Base string
	// Commit is the full sha of the new HEAD.
	Commit string
}

// Writer is the single Git-write boundary for one repository work tree.
// The zero value is not usable; construct with New. A Writer is stateless
// besides its configuration and safe for concurrent use: all mutations
// serialize on the per-repo locks. Writers for different repositories are
// fully independent.
type Writer struct {
	// RepoPath is the absolute path of the repository work tree.
	RepoPath string

	// Signature is the fallback commit identity for commits that do not
	// carry their own. The zero value falls back to the package default.
	Signature Signature

	// StaleLockAfter overrides how long a cross-process lock file may remain
	// before take-over (see DefaultStaleLockAfter). Zero uses the default.
	StaleLockAfter time.Duration
}

// New returns a Writer for repoPath, resolved to an absolute path.
func New(repoPath string) (*Writer, error) {
	if strings.TrimSpace(repoPath) == "" {
		return nil, errors.New("gitwriter: repository path is required")
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("gitwriter: resolving repository path: %w", err)
	}
	return &Writer{RepoPath: filepath.Clean(abs)}, nil
}

// Head resolves the repository's current HEAD commit, or "" when the
// repository has no commits yet. It is a read-only helper callers use to
// pick the base revision a change is authored against.
func (w *Writer) Head(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := w.gitDir(ctx); err != nil {
		return "", err
	}
	return w.head(ctx)
}

// Commit validates the expected base revision, applies the requested files
// and records them as one commit, all while holding the repository's
// in-process and cross-process locks. On any failure the work tree and
// index are rolled back and the error is returned with both locks released.
func (w *Writer) Commit(ctx context.Context, req CommitRequest) (res CommitResult, err error) {
	if err = ctx.Err(); err != nil {
		return CommitResult{}, err
	}
	files, err := normalizeFiles(req.Files)
	if err != nil {
		return CommitResult{}, err
	}
	if strings.TrimSpace(req.Message) == "" {
		return CommitResult{}, errors.New("gitwriter: commit message is required")
	}
	sig, err := w.signature(req.Signature)
	if err != nil {
		return CommitResult{}, err
	}

	// Serialize in-process first, then locate the git dir to place the
	// cross-process lock file. Both acquisitions respect ctx.
	mu := repoMutexFor(w.RepoPath)
	if err = mu.lock(ctx); err != nil {
		return CommitResult{}, fmt.Errorf("gitwriter: acquiring per-repo lock: %w", err)
	}
	gitDir, gerr := w.gitDir(ctx)
	if gerr != nil {
		mu.unlock()
		return CommitResult{}, gerr
	}
	fl, ferr := acquireFileLock(ctx, filepath.Join(gitDir, lockFileName), w.staleLockAfter())
	if ferr != nil {
		mu.unlock()
		return CommitResult{}, fmt.Errorf("gitwriter: acquiring cross-process lock: %w", ferr)
	}
	// Guaranteed release on every return path below: file lock first, then
	// in-process mutex.
	defer func() {
		if rerr := fl.release(); rerr != nil && err == nil {
			err = fmt.Errorf("gitwriter: releasing lock: %w", rerr)
		}
		mu.unlock()
	}()

	base, err := w.head(ctx)
	if err != nil {
		return CommitResult{}, err
	}
	if base != req.ExpectedBase {
		return CommitResult{}, fmt.Errorf("gitwriter: current head %q does not match expected base %q: %w", base, req.ExpectedBase, ErrBaseRevision)
	}

	rels := relPaths(files)
	applied, aerr := w.apply(ctx, files)
	if aerr != nil {
		return CommitResult{}, errors.Join(aerr, w.revert(ctx, applied, rels, base))
	}

	if _, serr := runGit(ctx, w.RepoPath, "", append([]string{"add", "-A", "--"}, rels...)...); serr != nil {
		return CommitResult{}, errors.Join(serr, w.revert(ctx, applied, rels, base))
	}

	// No -q: a no-op commit must keep its "nothing to commit" stderr so
	// ErrNoChanges can be surfaced as a typed error. The success summary
	// goes to stdout and is discarded.
	commitArgs := []string{
		"-c", "user.name=" + sig.Name,
		"-c", "user.email=" + sig.Email,
		"commit", "--no-verify", "-m", req.Message,
	}
	if base != "" {
		// Partial commit: only the named paths' work tree state is recorded,
		// so pre-existing unrelated index entries cannot leak into this
		// commit. On an unborn HEAD a plain commit of the (empty) index is
		// the initial commit and contains exactly the staged files.
		commitArgs = append(commitArgs, "--")
		commitArgs = append(commitArgs, rels...)
	}
	if _, cerr := runGit(ctx, w.RepoPath, "", commitArgs...); cerr != nil {
		commitErr := cerr
		if isNothingToCommit(cerr) {
			commitErr = ErrNoChanges
		}
		return CommitResult{}, errors.Join(commitErr, w.revert(ctx, applied, rels, base))
	}

	sha, rerr := runGit(ctx, w.RepoPath, "", "rev-parse", "HEAD")
	if rerr != nil {
		return CommitResult{}, errors.Join(rerr, w.revert(ctx, applied, rels, base))
	}
	return CommitResult{Base: base, Commit: strings.TrimSpace(sha)}, nil
}

func (w *Writer) staleLockAfter() time.Duration {
	if w.StaleLockAfter > 0 {
		return w.StaleLockAfter
	}
	return DefaultStaleLockAfter
}

func (w *Writer) signature(override *Signature) (Signature, error) {
	sig := defaultSignature
	if w.Signature != (Signature{}) {
		sig = w.Signature
	}
	if override != nil {
		sig = *override
	}
	if strings.TrimSpace(sig.Name) == "" || strings.TrimSpace(sig.Email) == "" {
		return Signature{}, errors.New("gitwriter: commit signature requires name and email")
	}
	return sig, nil
}

// gitDir resolves the repository's git dir via `git rev-parse
// --absolute-git-dir`. This validates that the path is a repository and
// yields the right, per-work-tree git dir for linked worktrees (where .git
// is a file). Cancellation is returned unchanged.
func (w *Writer) gitDir(ctx context.Context) (string, error) {
	out, err := runGit(ctx, w.RepoPath, "", "rev-parse", "--absolute-git-dir")
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		return "", fmt.Errorf("%w (%s): %v", ErrNotRepository, w.RepoPath, err)
	}
	return strings.TrimSpace(out), nil
}

// head returns the current HEAD sha, or "" for an unborn HEAD (a quiet
// `git rev-parse HEAD` exits non-zero exactly when HEAD resolves to no
// commit). A corrupt-but-present HEAD is also reported as "", which the
// base-revision comparison then rejects for any caller-supplied base.
func (w *Writer) head(ctx context.Context) (string, error) {
	out, err := runGit(ctx, w.RepoPath, "", "rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// stagedFile remembers the pre-call state of one applied path so revert can
// restore it.
type stagedFile struct {
	rel      string
	abs      string
	prior    []byte
	hadPrior bool
}

// apply writes/deletes the requested paths, recording prior content before
// the first mutation. On failure it returns the paths mutated so far so the
// caller can roll them back.
func (w *Writer) apply(ctx context.Context, files []FileUpdate) ([]stagedFile, error) {
	applied := make([]stagedFile, 0, len(files))
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return applied, err
		}
		st := stagedFile{
			rel: f.Path,
			abs: filepath.Join(w.RepoPath, filepath.FromSlash(f.Path)),
		}
		if prior, rerr := os.ReadFile(st.abs); rerr == nil {
			st.prior, st.hadPrior = prior, true
		} else if !errors.Is(rerr, fs.ErrNotExist) {
			return applied, rerr
		}
		if f.Delete {
			if rmErr := os.Remove(st.abs); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
				return applied, rmErr
			}
		} else {
			if mkErr := os.MkdirAll(filepath.Dir(st.abs), 0o755); mkErr != nil {
				return applied, mkErr
			}
			if wErr := os.WriteFile(st.abs, f.Content, 0o644); wErr != nil {
				return applied, wErr
			}
		}
		applied = append(applied, st)
	}
	return applied, nil
}

// revert undoes apply's mutations: work tree contents go back to their
// pre-call state and the index entries staged for these paths are reset.
// Best effort by design — errors are collected and joined into the commit
// error instead of masking it. If ctx is already cancelled the index cleanup
// cannot run and a later Writer.Commit re-validates the base before
// committing anything, so a stale index entry cannot be committed silently.
func (w *Writer) revert(ctx context.Context, applied []stagedFile, rels []string, base string) error {
	var errs []error
	for _, st := range applied {
		var err error
		if st.hadPrior {
			err = os.WriteFile(st.abs, st.prior, 0o644)
		} else {
			err = os.Remove(st.abs)
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if len(applied) == 0 {
		return errors.Join(errs...)
	}
	var idxErr error
	if base != "" {
		_, idxErr = runGit(ctx, w.RepoPath, "", append([]string{"reset", "-q", "HEAD", "--"}, rels...)...)
	} else {
		_, idxErr = runGit(ctx, w.RepoPath, "", append([]string{"rm", "-r", "--cached", "-f", "-q", "--ignore-unmatch", "--"}, rels...)...)
	}
	if idxErr != nil {
		errs = append(errs, idxErr)
	}
	return errors.Join(errs...)
}

// GitError is a failed git command with its argv and output attached.
type GitError struct {
	Args   []string
	Err    error
	Stdout string
	Stderr string
}

func (e *GitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		msg = strings.TrimSpace(e.Stdout)
	}
	if msg == "" {
		return fmt.Sprintf("%s: %v", strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("%s: %v: %s", strings.Join(e.Args, " "), e.Err, msg)
}

func (e *GitError) Unwrap() error { return e.Err }

// RunGit executes one git command with structured argv in dir, optionally
// with fixed stdin text, under the same discipline as this package's own
// invocations: no shell, ctx cancellation kills the process, ambient GIT_*
// variables are stripped, and failures carry a GitError with argv and output.
//
// It exists so the semantic layer above this boundary (internal/gitintegration)
// can perform the ref/merge plumbing gitwriter does not expose without
// duplicating this helper; it grants no write authority beyond what git
// itself enforces.
func RunGit(ctx context.Context, dir, stdin string, args ...string) (string, error) {
	return runGit(ctx, dir, stdin, args...)
}

// runGit executes one git command with structured argv in dir. No shell is
// involved; ctx cancellation kills the process and is returned (joined with
// the GitError) so callers can errors.Is against context errors.
func runGit(ctx context.Context, dir, stdin string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = sanitizedGitEnv()
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		gerr := &GitError{
			Args:   append([]string{"git"}, args...),
			Err:    err,
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return stdout.String(), errors.Join(ctxErr, gerr)
		}
		return stdout.String(), gerr
	}
	return stdout.String(), nil
}

// sanitizedGitEnv strips ambient GIT_* variables (GIT_DIR, GIT_WORK_TREE,
// GIT_INDEX_FILE, ...) so they cannot redirect the controlled commands to
// another repository or index.
func sanitizedGitEnv() []string {
	env := os.Environ()
	out := env[:0]
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// isNothingToCommit recognizes git's no-op commit failure so callers get
// the typed ErrNoChanges. git reports it with variant wordings ("nothing to
// commit", "nothing added to commit") and, for partial commits, on stdout;
// both streams are checked. Matching CLI output is a pragmatic boundary; the
// messages are stable across supported git versions.
func isNothingToCommit(err error) bool {
	var gerr *GitError
	if !errors.As(err, &gerr) {
		return false
	}
	for _, out := range []string{gerr.Stderr, gerr.Stdout} {
		if strings.Contains(out, "nothing to commit") || strings.Contains(out, "nothing added to commit") {
			return true
		}
	}
	return false
}

// normalizeFiles validates and canonicalizes every path up front, before
// any lock is taken or file touched.
func normalizeFiles(in []FileUpdate) ([]FileUpdate, error) {
	out := make([]FileUpdate, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, f := range in {
		rel, err := cleanRelPath(f.Path)
		if err != nil {
			return nil, fmt.Errorf("gitwriter: file %q: %w", f.Path, err)
		}
		if seen[rel] {
			return nil, fmt.Errorf("gitwriter: duplicate path %q", rel)
		}
		seen[rel] = true
		f.Path = rel
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil, errors.New("gitwriter: commit requires at least one file")
	}
	return out, nil
}

// cleanRelPath normalizes a requested path to a slash-separated path safely
// inside the repository root.
func cleanRelPath(p string) (string, error) {
	switch {
	case p == "":
		return "", errors.New("path is empty")
	case strings.ContainsRune(p, 0):
		return "", errors.New("path contains NUL")
	case strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\"):
		return "", errors.New("path must be relative to the repository root")
	case len(p) >= 2 && p[1] == ':':
		return "", errors.New("absolute paths are not allowed")
	}
	np := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if np == "." || np == ".." || strings.HasPrefix(np, "../") {
		return "", errors.New("path escapes the repository")
	}
	return np, nil
}

func relPaths(files []FileUpdate) []string {
	rels := make([]string, 0, len(files))
	for _, f := range files {
		rels = append(rels, f.Path)
	}
	return rels
}
