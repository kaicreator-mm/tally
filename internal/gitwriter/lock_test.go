package gitwriter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// deadPID returns the pid of a spawned, killed and reaped process: provably
// not running. t.Skip guards against exotic pid-recycling on the host.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("git", "--version")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start probe process: %v", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if processAlive(pid) {
		t.Skip("pid reused too quickly to test dead-owner takeover")
	}
	return pid
}

func lockPathIn(t *testing.T, dir string) string {
	t.Helper()
	return filepath.Join(dir, ".git", lockFileName)
}

// writeLockFile simulates a lock file written by a (possibly foreign) owner.
func writeLockFile(t *testing.T, path string, info fileLockInfo) {
	t.Helper()
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal lock info: %v", err)
	}
	if err := os.WriteFile(path, b, 0o666); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
}

func TestRepoMutexSameInstancePerRepo(t *testing.T) {
	dir := t.TempDir()
	if repoMutexFor(dir) != repoMutexFor(filepath.Join(dir, "subdir", "..")) {
		t.Fatal("same repository resolved to different mutexes")
	}
}

func TestRepoMutexSerializesSameRepo(t *testing.T) {
	dir := t.TempDir()
	mu := repoMutexFor(dir)
	if err := mu.lock(context.Background()); err != nil {
		t.Fatalf("first lock: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if err := repoMutexFor(dir).lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second lock while held: got %v, want DeadlineExceeded", err)
	}

	mu.unlock()
	if err := repoMutexFor(dir).lock(context.Background()); err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	repoMutexFor(dir).unlock()
}

func TestRepoMutexCancelWhileWaiting(t *testing.T) {
	dir := t.TempDir()
	mu := repoMutexFor(dir)
	if err := mu.lock(context.Background()); err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer mu.unlock()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- repoMutexFor(dir).lock(ctx)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting lock: got %v, want Canceled", err)
	}

	// The waiter must not have consumed the slot.
	heldCtx, heldCancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer heldCancel()
	if err := repoMutexFor(dir).lock(heldCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("mutex still held after canceled waiter: %v", err)
	}
}

func TestRepoMutexDifferentReposIndependent(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	muA := repoMutexFor(dirA)
	if err := muA.lock(context.Background()); err != nil {
		t.Fatalf("lock A: %v", err)
	}
	defer muA.unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := repoMutexFor(dirB).lock(ctx); err != nil {
		t.Fatalf("repo B blocked by repo A: %v", err)
	}
	repoMutexFor(dirB).unlock()
}

func TestNormalizeRepoPathCaseInsensitiveOnWindows(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS != "windows" {
		if normalizeRepoPath(dir) == "" {
			t.Fatal("empty normalization")
		}
		return
	}
	if normalizeRepoPath(dir) != normalizeRepoPath(strings.ToUpper(dir)) {
		t.Fatal("case variants must map to one key on windows")
	}
}

func TestAcquireFileLockWritesOwnerInfo(t *testing.T) {
	dir := t.TempDir()
	path := lockPathIn(t, dir)
	before := time.Now().UTC()

	lk, err := acquireFileLock(context.Background(), path, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	var info fileLockInfo
	if err := json.Unmarshal(b, &info); err != nil {
		t.Fatalf("parse lock info %q: %v", b, err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("pid = %d, want %d", info.PID, os.Getpid())
	}
	if info.Host != localHostName() {
		t.Errorf("host = %q, want %q", info.Host, localHostName())
	}
	if info.CreatedAt.Before(before.Add(-time.Second)) || info.CreatedAt.After(time.Now().UTC().Add(time.Second)) {
		t.Errorf("created_at = %v, want close to %v", info.CreatedAt, before)
	}

	if err := lk.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after release: %v", err)
	}
}

func TestAcquireFileLockExclusiveWhileHeld(t *testing.T) {
	dir := t.TempDir()
	path := lockPathIn(t, dir)

	lk, err := acquireFileLock(context.Background(), path, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := acquireFileLock(ctx, path, 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second acquire while held: got %v, want DeadlineExceeded", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("original lock must survive failed take: %v", err)
	}

	if err := lk.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	lk2, err := acquireFileLock(context.Background(), path, 0)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := lk2.release(); err != nil {
		t.Fatalf("release 2: %v", err)
	}
}

func TestAcquireFileLockCanceledWhileWaiting(t *testing.T) {
	dir := t.TempDir()
	path := lockPathIn(t, dir)

	lk, err := acquireFileLock(context.Background(), path, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lk.release()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := acquireFileLock(ctx, path, 0)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled acquire: got %v, want Canceled", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("holder's lock must survive canceled waiter: %v", err)
	}
}

func TestAcquireFileLockTakeoverAfterStaleAge(t *testing.T) {
	dir := initRepo(t)
	path := lockPathIn(t, dir)
	writeLockFile(t, path, fileLockInfo{
		PID:       os.Getpid(), // alive: only the age rule may take over
		Host:      localHostName(),
		CreatedAt: time.Now().UTC().Add(-time.Hour),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lk, err := acquireFileLock(ctx, path, 30*time.Second)
	if err != nil {
		t.Fatalf("stale lock not taken over: %v", err)
	}
	if err := lk.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestAcquireFileLockTakeoverWhenHolderDead(t *testing.T) {
	dir := initRepo(t)
	path := lockPathIn(t, dir)
	writeLockFile(t, path, fileLockInfo{
		PID:       deadPID(t),
		Host:      localHostName(),
		CreatedAt: time.Now().UTC(), // fresh: only the dead-pid rule applies
	})

	lk, err := acquireFileLock(context.Background(), path, time.Hour)
	if err != nil {
		t.Fatalf("dead holder's lock not taken over: %v", err)
	}
	if err := lk.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestAcquireFileLockLiveHolderNotStolenBeforeAge(t *testing.T) {
	dir := initRepo(t)
	path := lockPathIn(t, dir)
	writeLockFile(t, path, fileLockInfo{
		PID:       os.Getpid(),
		Host:      localHostName(),
		CreatedAt: time.Now().UTC(),
	})
	t.Cleanup(func() { _ = os.Remove(path) })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := acquireFileLock(ctx, path, time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("live holder's lock was stolen or waiter misbehaved: %v", err)
	}
}

func TestAcquireFileLockRejectsBadPath(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireFileLock(context.Background(), filepath.Join(blocker, lockFileName), 0); err == nil {
		t.Fatal("acquire under a file path must fail")
	}
}

func TestFileLockReleaseIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := lockPathIn(t, dir)
	lk, err := acquireFileLock(context.Background(), path, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := lk.release(); err != nil {
		t.Fatalf("first release: %v", err)
	}
	if err := lk.release(); err != nil {
		t.Fatalf("second release: %v", err)
	}
	var nilLock *fileLock
	if err := nilLock.release(); err != nil {
		t.Fatalf("nil release: %v", err)
	}
}

func TestProcessAlive(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("own pid reported dead")
	}
	if processAlive(-1) || processAlive(0) {
		t.Fatal("non-positive pids must be reported dead")
	}
	if deadPID(t) <= 0 {
		t.Fatal("deadPID returned invalid pid")
	}
}
