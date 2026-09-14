package gitwriter

// Serialization primitives required by PRD v1.3 §23.1: a per-repo in-process
// mutex and a cross-process lock file, so that Tally Server, Worker and the
// knowledge/docs writers can never mutate the same repository concurrently.
//
// Standard library only: the cross-process lock is a lock file created with
// O_CREATE|O_EXCL (atomic exclusive create on every supported platform) that
// records its owner pid/host/timestamp. There is intentionally no blocking
// kernel-level lock (flock/LockFileEx): take-over of abandoned locks would
// otherwise be impossible, and Writer.Commit validates the expected base
// revision under the lock, which remains the correctness backstop.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DefaultStaleLockAfter is the age at which a cross-process lock file is
// considered abandoned even when its recorded owner cannot be probed. Choose
// conservative values above the worst-case duration of the git operations a
// Writer performs; take-over of a live holder's lock after this age is a
// documented, deliberate trade-off (see lockIsStale).
const DefaultStaleLockAfter = 10 * time.Minute

// lockFileName is created inside the repository git dir (resolved via
// `git rev-parse --absolute-git-dir`), which keeps the lock out of the work
// tree, out of commits, and scoped per work tree for linked worktrees.
const lockFileName = "tally-gitwriter.lock"

// lockPollInterval is how long a blocked cross-process acquirer sleeps
// between attempts. A variable so tests can tighten it if needed.
var lockPollInterval = 20 * time.Millisecond

// errLocked is returned by tryAcquireFileLock when the lock file exists.
var errLocked = errors.New("gitwriter: lock held by another process")

// repoMutex serializes writers of one repository within this process.
//
// The channel has capacity one; lock selects on ctx so a waiter can give up
// on cancellation instead of blocking forever, and unlock is a bare receive
// (unlocking an unheld mutex panics, surfacing misuse instead of hiding it).
// Instances live in repoLocks for the process lifetime; the map grows by one
// entry per distinct repository, which is bounded in practice and keeps
// lookup O(1). Locks are not reentrant: never call Commit from within a
// locked section of the same Writer.
type repoMutex struct {
	slot chan struct{}
}

func newRepoMutex() *repoMutex {
	return &repoMutex{slot: make(chan struct{}, 1)}
}

// lock takes the mutex or returns ctx's error while waiting.
func (m *repoMutex) lock(ctx context.Context) error {
	select {
	case m.slot <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlock releases the mutex.
func (m *repoMutex) unlock() {
	<-m.slot
}

var (
	repoLocksMu sync.Mutex
	repoLocks   = map[string]*repoMutex{}
)

// repoMutexFor returns the process-wide mutex for one repository. Different
// repository paths get different mutexes and never block one another.
func repoMutexFor(repoPath string) *repoMutex {
	key := normalizeRepoPath(repoPath)
	repoLocksMu.Lock()
	defer repoLocksMu.Unlock()
	m, ok := repoLocks[key]
	if !ok {
		m = newRepoMutex()
		repoLocks[key] = m
	}
	return m
}

// normalizeRepoPath canonicalizes a repository path into a stable map key:
// absolute, slash-separated, and lower-cased where the filesystem is
// case-insensitive (Windows), so "C:\Repo" and "c:/repo" share one mutex.
func normalizeRepoPath(repoPath string) string {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = filepath.Clean(repoPath)
	}
	abs = filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		abs = strings.ToLower(abs)
	}
	return abs
}

// fileLockInfo is the ownership record stored in the lock file.
type fileLockInfo struct {
	PID       int       `json:"pid"`
	Host      string    `json:"host"`
	CreatedAt time.Time `json:"created_at"`
}

// fileLock is a held cross-process lock. Release must be called exactly once
// on every path that acquired it.
type fileLock struct {
	path string
}

// acquireFileLock creates path exclusively or waits until the current holder
// releases, the holder proves dead, or the lock ages past staleAfter
// (<= 0 selects DefaultStaleLockAfter). Returns ctx's error if ctx ends
// while waiting.
func acquireFileLock(ctx context.Context, path string, staleAfter time.Duration) (*fileLock, error) {
	if staleAfter <= 0 {
		staleAfter = DefaultStaleLockAfter
	}
	ticker := time.NewTicker(lockPollInterval)
	defer ticker.Stop()
	for {
		lk, err := tryAcquireFileLock(path)
		if err == nil {
			return lk, nil
		}
		if !errors.Is(err, errLocked) {
			return nil, err
		}
		takeoverStaleLock(path, staleAfter)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("gitwriter: timed out waiting for lock %s: %w", path, ctx.Err())
		case <-ticker.C:
		}
	}
}

// tryAcquireFileLock performs one exclusive-create attempt and writes the
// ownership record. Any failure while recording ownership removes the lock
// file again so another writer is not blocked by a half-written lock.
func tryAcquireFileLock(path string) (*fileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o666)
	if err != nil {
		if os.IsExist(err) {
			return nil, errLocked
		}
		return nil, err
	}
	info := fileLockInfo{PID: os.Getpid(), Host: localHostName(), CreatedAt: time.Now().UTC()}
	b, mErr := json.Marshal(info)
	if mErr == nil {
		_, mErr = f.Write(append(b, '\n'))
	}
	if cErr := f.Close(); mErr == nil {
		mErr = cErr
	}
	if mErr != nil {
		_ = os.Remove(path)
		return nil, mErr
	}
	return &fileLock{path: path}, nil
}

// release removes the lock file. Releasing an already released lock is a
// no-op returning nil; a failed release keeps the path so a retry can be
// attempted (Commit surfaces the error).
func (l *fileLock) release() error {
	if l == nil || l.path == "" {
		return nil
	}
	path := l.path
	l.path = ""
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		l.path = path
		return err
	}
	return nil
}

// takeoverStaleLock removes the lock file when it is provably stale.
// Removal is best effort: if the original holder re-created it in between,
// the next poll re-checks against the fresh file.
func takeoverStaleLock(path string, staleAfter time.Duration) {
	info, mtime, ok := readLockInfo(path)
	if !ok {
		return
	}
	if lockIsStale(info, mtime, staleAfter) {
		_ = os.Remove(path)
	}
}

// readLockInfo reads the ownership record plus the file mtime (used as the
// fallback timestamp for empty or unparsable files). ok is false when the
// file vanished mid-check; callers then simply wait for the next poll.
func readLockInfo(path string) (info fileLockInfo, mtime time.Time, ok bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return info, time.Time{}, false
	}
	if jErr := json.Unmarshal(b, &info); jErr != nil {
		info = fileLockInfo{}
	}
	st, sErr := os.Stat(path)
	if sErr != nil {
		return info, time.Time{}, false
	}
	return info, st.ModTime(), true
}

// lockIsStale decides whether a held lock may be taken over. Two rules, both
// conservative:
//
//  1. Dead holder, same host: the recorded pid cannot be probed alive. This
//     is the ordinary crash case and the reason pid and host are recorded.
//  2. Age: the lock is older than staleAfter regardless of owner state.
//     This covers holders on other hosts (not probeable) and pid reuse that
//     would otherwise make a dead holder look alive. Take-over after this
//     age can still hit a live-but-slow holder; set staleAfter above the
//     worst-case git operation time and rely on the base-revision check in
//     Writer.Commit as the correctness backstop.
func lockIsStale(info fileLockInfo, mtime time.Time, staleAfter time.Duration) bool {
	created := info.CreatedAt
	if created.IsZero() {
		created = mtime
	}
	if !created.IsZero() && time.Since(created) >= staleAfter {
		return true
	}
	return info.Host == localHostName() && info.PID > 0 && !processAlive(info.PID)
}

// processAlive reports whether pid names a running process. Best effort by
// design: on Windows a successfully opened handle implies existence; on unix
// signal 0 is probed, and any error other than a definitive "process gone"
// conservatively counts as alive so live holders are never stolen from.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false // Windows: the process does not exist.
	}
	if runtime.GOOS == "windows" {
		return true
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return false
		}
		if errors.Is(err, syscall.Errno(3)) { // ESRCH on linux/darwin
			return false
		}
	}
	return true
}

// localHostName caches os.Hostname for pid-probe eligibility checks.
var localHostName = sync.OnceValue(func() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
})
