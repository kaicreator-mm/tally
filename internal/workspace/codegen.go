package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// RunID identifies one CodegenRun. The identifier is owned by the codegen
// orchestration; this package only requires it to be non-empty and treats it
// as opaque.
type RunID string

// Sentinel errors for the CodegenRun source-workspace lifecycle. Match with
// errors.Is, never by string.
var (
	// ErrEmptyRunID: the caller passed an empty run identifier.
	ErrEmptyRunID = errors.New("workspace: codegen run id must not be empty")
	// ErrRunTerminated: the run already terminated its source worktree. A
	// terminated run never gets a second worktree: one run owns exactly one
	// disposable source worktree for its whole lifetime (PRD v1.3 §13.2).
	ErrRunTerminated = errors.New("workspace: codegen run already terminated its source worktree")
	// ErrRunBaseConflict: an Open for a run that already owns a worktree
	// named a different base revision. Iterations of one run must agree on
	// the base commit; silently keeping the old base would hide the drift.
	ErrRunBaseConflict = errors.New("workspace: run base revision conflicts with the run's source worktree")
)

// RunSources is the single disposable source worktree owned by one CodegenRun
// (PRD v1.3 §13.2, INV-EXEC-01): every iteration of the run executes in this
// same worktree, and the worktree is destroyed only when the run terminates.
// The owner primary worktree can never become a RunSources target: the
// backing Manager refuses to create disposable worktrees inside the owner
// checkout and refuses to destroy the primary worktree.
type RunSources struct {
	run RunID
	wt  *Worktree
	m   *Manager

	mu         sync.Mutex
	terminated bool
}

// Run returns the run the sources belong to.
func (r *RunSources) Run() RunID { return r.run }

// Path returns the absolute directory of the run's disposable source
// worktree. It is stable for the run's whole lifetime: iterations reuse the
// same directory and never re-create it.
func (r *RunSources) Path() string { return r.wt.Path }

// BaseCommit returns the exact commit the worktree was created at.
func (r *RunSources) BaseCommit() string { return r.wt.BaseCommit }

// Terminate destroys the run's source worktree. This is the only teardown
// path for a run workspace: it removes exactly this disposable worktree and
// nothing else — never a shared dependency cache and never the owner primary
// worktree. Terminate is idempotent and completes even on an already
// cancelled context (teardown does not inherit caller cancellation; see
// Manager.Cleanup). A failure keeps the worktree unterminated so the call can
// be retried.
func (r *RunSources) Terminate(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.terminated {
		return nil
	}
	if err := r.m.Cleanup(ctx, r.wt); err != nil {
		return err
	}
	r.terminated = true
	return nil
}

// hasTerminated reports whether the run already terminated its worktree.
func (r *RunSources) hasTerminated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.terminated
}

// RunTable tracks the source worktrees of CodegenRuns created through one
// Manager. It enforces the §13.2 ownership rule: one run owns at most one
// disposable source worktree, every Open of that run returns the same one,
// and a terminated run cannot grow a replacement. It is safe for concurrent
// use. Terminated runs are kept as tombstones so a late Open fails loudly
// instead of silently minting a second worktree for the same run.
type RunTable struct {
	m    *Manager
	mu   sync.Mutex
	runs map[RunID]*RunSources
}

// NewRunTable returns a RunTable handing out run source worktrees through m.
func NewRunTable(m *Manager) *RunTable {
	return &RunTable{m: m, runs: make(map[RunID]*RunSources)}
}

// Open returns the source worktree of run, creating it detached at exactly
// baseRevision on first use. Every later Open for the same run returns the
// same worktree: the iterations of one CodegenRun reuse one disposable
// worktree and never re-create it (PRD v1.3 §13.2).
//
// Open fails with ErrRunTerminated when the run already terminated its
// worktree, with ErrRunBaseConflict when the run already owns a worktree but
// baseRevision resolves to a different commit, and with ErrUnknownRevision
// when baseRevision does not resolve at all.
func (t *RunTable) Open(ctx context.Context, run RunID, baseRevision string) (*RunSources, error) {
	if run == "" {
		return nil, ErrEmptyRunID
	}
	// Resolve the caller's revision text before touching the table: the
	// worktree is always created from a fully resolved commit, and reuse
	// decisions compare resolved commits, not moving refs.
	commit, err := t.m.git.resolveRevision(ctx, t.m.repoDir, baseRevision)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, ok := t.runs[run]; ok {
		if existing.hasTerminated() {
			return nil, fmt.Errorf("%w: %s", ErrRunTerminated, run)
		}
		if !strings.EqualFold(existing.BaseCommit(), commit) {
			return nil, fmt.Errorf("%w: run %s owns worktree at %s, requested %s (resolves to %s)",
				ErrRunBaseConflict, run, existing.BaseCommit(), baseRevision, commit)
		}
		return existing, nil
	}
	wt, err := t.m.Create(ctx, commit)
	if err != nil {
		return nil, err
	}
	rs := &RunSources{run: run, wt: wt, m: t.m}
	t.runs[run] = rs
	return rs, nil
}

// Terminate destroys the named run's source worktree and retires the run.
// It is a no-op success for a run that never opened a worktree or already
// terminated it, so defer-based teardown paths need no bookkeeping.
func (t *RunTable) Terminate(ctx context.Context, run RunID) error {
	t.mu.Lock()
	rs, ok := t.runs[run]
	t.mu.Unlock()
	if !ok {
		return nil
	}
	return rs.Terminate(ctx)
}
