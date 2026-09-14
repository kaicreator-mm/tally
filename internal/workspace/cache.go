package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
)

// Sentinel errors for the shared dependency cache boundary. Match with
// errors.Is, never by string.
var (
	// ErrInvalidCacheKey: a cache key component is empty or contains
	// characters that cannot be used verbatim as a path element.
	ErrInvalidCacheKey = errors.New("workspace: invalid dependency cache key")
	// ErrCacheBoundary: the dependency cache root nests with the disposable
	// source workspace in a forbidden direction. Shared caches and mutable
	// source worktrees must never contain one another, so neither side's
	// lifecycle can destroy the other (PRD v1.3 §13.4).
	ErrCacheBoundary = errors.New("workspace: dependency cache root must not nest with the source workspace")
)

// CacheKey names one shared dependency cache entry along exactly the PRD
// v1.3 §13.4 dimensions: toolchain + dependency lock hash + platform.
type CacheKey struct {
	// Toolchain is the dependency toolchain family, e.g. "go", "node",
	// "python", "rust".
	Toolchain string
	// LockHash is a stable hash of the toolchain's dependency lock input
	// (for example a content hash of go.mod+go.sum or pnpm-lock.yaml). The
	// caller chooses the hash function; it only has to map identical lock
	// inputs to identical strings.
	LockHash string
	// Platform is the execution platform the cache is valid for, e.g.
	// "linux-amd64" or "windows-amd64".
	Platform string
}

// validate refuses keys whose components cannot be used verbatim as path
// elements: empty, too long, or containing anything outside a conservative
// [A-Za-z0-9._-] charset (so no separators, no "..", no whitespace).
func (k CacheKey) validate() error {
	parts := []struct{ name, val string }{
		{"toolchain", k.Toolchain},
		{"lock hash", k.LockHash},
		{"platform", k.Platform},
	}
	for _, p := range parts {
		if !safeCacheComponent(p.val) {
			return fmt.Errorf("%w: %s %q must be 1-64 characters of [A-Za-z0-9._-] starting with a letter or digit",
				ErrInvalidCacheKey, p.name, p.val)
		}
	}
	return nil
}

// safeCacheComponent reports whether s can be embedded verbatim as one path
// element of a cache directory.
func safeCacheComponent(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			// Allowed everywhere; a leading letter or digit is enforced below.
		case c == '.' || c == '_' || c == '-':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// DependencyCache allocates shared dependency cache directories that live
// outside every disposable source worktree (PRD v1.3 §13.4): a disposable
// worktree must not force every run to re-download all dependencies, but the
// shared cache must never share a filesystem prefix with the mutable source
// workspace. NewDependencyCache therefore guarantees mutual non-containment
// between the cache root and both the owner checkout and the disposable
// worktree root: worktree teardown cannot reach cache entries and cache
// eviction cannot reach a source worktree or the owner checkout.
type DependencyCache struct {
	root string
}

// NewDependencyCache roots a dependency cache at cacheRoot for the worktrees
// managed by m. The root must lie strictly outside the owner checkout (it is
// Tally-owned shared state, not part of any workspace) and must not nest with
// the disposable worktree root in either direction.
func (m *Manager) NewDependencyCache(cacheRoot string) (*DependencyCache, error) {
	if cacheRoot == "" {
		return nil, fmt.Errorf("%w: cache root must not be empty", ErrCacheBoundary)
	}
	absRoot, err := filepath.Abs(cacheRoot)
	if err != nil {
		return nil, fmt.Errorf("workspace: resolve dependency cache root: %w", err)
	}
	if samePath(absRoot, m.repoDir) || insidePath(m.repoDir, absRoot) {
		return nil, fmt.Errorf("%w: dependency cache root %s is inside owner checkout %s",
			ErrPrimaryWorktree, absRoot, m.repoDir)
	}
	if insidePath(absRoot, m.repoDir) {
		return nil, fmt.Errorf("%w: dependency cache root %s contains owner checkout %s",
			ErrCacheBoundary, absRoot, m.repoDir)
	}
	if samePath(absRoot, m.root) || insidePath(m.root, absRoot) {
		return nil, fmt.Errorf("%w: dependency cache root %s is inside disposable worktree root %s",
			ErrCacheBoundary, absRoot, m.root)
	}
	if insidePath(absRoot, m.root) {
		return nil, fmt.Errorf("%w: dependency cache root %s contains disposable worktree root %s",
			ErrCacheBoundary, absRoot, m.root)
	}
	return &DependencyCache{root: absRoot}, nil
}

// Root returns the absolute cache root directory.
func (c *DependencyCache) Root() string { return c.root }

// Path returns the deterministic shared cache directory for key: the same
// key always yields the same directory and different keys never share one.
// Nested elements keep the key dimensions unambiguous (no concatenation
// collisions). Path is pure — it creates nothing — so handing out a cache
// path can never mutate the filesystem; the toolchain that consumes the
// cache creates its directory on demand.
func (c *DependencyCache) Path(key CacheKey) (string, error) {
	if err := key.validate(); err != nil {
		return "", err
	}
	return filepath.Join(c.root, key.Toolchain, key.Platform, key.LockHash), nil
}
