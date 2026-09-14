// P6-06 — Safe project export (docs/tasks/v1.4.1/tasks/P6/P6-06_safe_project_export.md).
//
// Deterministic, commit-pinned source/docs export for external code review or
// agent analysis, without local runtime state or credentials:
//
//	git -C <repo> rev-parse --verify <ref>^{commit}   // pin the source commit
//	git -C <repo> archive --format=tar <commit>       // committed tree only
//	  → manifest/denylist scan of every tar entry (fail-safe)
//	  → archive + manifest written ONLY if the scan is clean
//
// Safety properties:
//   - content is read from the Git object database at one pinned commit, never
//     from the working tree, so ignored local runtime state (.codegraph/ /
//     .codex/ / .worktrees/ / .e2e/ / bin/ / caches) cannot enter the archive
//     by construction — this is the defect named in SOURCE_BASELINE.md #7
//     ("repository export currently includes local/runtime artifacts when the
//     entire worktree is archived");
//   - a second denylist scan runs over the produced archive and rejects denied
//     paths even when they are tracked (e.g. after a .gitignore change):
//     deny wins over ignore (L3 Failure Handling);
//   - only regular files and directories are allowed; symlinks, links and
//     special tar entry types are denied (the export is a source projection,
//     not a filesystem image);
//   - path traversal is denied (absolute paths, backslash separators, ".."
//     segments, duplicate entry names);
//   - secret backstop: private-key PEM markers are denied by content, on top of
//     the credential-filename denylist (.env*, id_rsa*, *.pem, *.key, …).
//     The content backstop covers private-key PEM markers only; it deliberately
//     does not scan for AWS-style key ids because the tracked logpipe redaction
//     fixture uses the public AWS documentation example key
//     (internal/logpipe/redact_test.go) as required source;
//   - on any denial nothing is published: the temporary archive is removed and
//     a typed *DeniedPathError is returned.
//
// Source-commit identity is preserved (DL-04 reading): the manifest pins the
// full resolved commit sha, and every entry carries size + sha256. For a given
// commit and git version the archive bytes are deterministic (git archive uses
// the commit timestamp for tar headers and the repo's eol=lf attribute), so the
// same commit always yields the same archive and manifest bytes.
package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestSchema is the machine-readable manifest structure version.
const ManifestSchema = "tally.export.manifest.v1"

// maxSecretMarkerLen bounds the inter-chunk carry of the streaming content
// scanner; every marker must be shorter than this.
const maxSecretMarkerLen = 64

// deniedDirSegments denies an entry whose ANY path segment equals one of these
// names — local runtime/tooling state, build output and dependency/bytecode
// caches. Validated against the tracked tree at HEAD (2026-09-12): zero
// tracked paths contain any of these segments, so the list is fail-safe and
// collision-free today; if a future tracked path collides, the export fails by
// design (deny wins over ignore) and the list must be revisited consciously.
var deniedDirSegments = map[string]string{
	".git":          "git metadata/objects",
	".worktrees":    "disposable candidate worktrees (runtime isolation state)",
	".e2e":          "local e2e runtime state",
	".codegraph":    "local code-graph index cache",
	".codex":        "local coding-agent session state",
	"bin":           "local build output",
	"node_modules":  "dependency cache",
	"__pycache__":   "bytecode cache",
	".pytest_cache": "test cache",
	".mypy_cache":   "type-check cache",
	".ruff_cache":   "linter cache",
	".cache":        "generic local cache",
	".venv":         "local virtualenv",
	"venv":          "local virtualenv",
	".gradle":       "build cache",
	".turbo":        "build cache",
	".next":         "build cache",
	".nuxt":         "build cache",
	".nyc_output":   "coverage cache",
	".ssh":          "local credentials",
	".aws":          "local credentials",
	".gnupg":        "local keyring",
}

// deniedBaseExact denies an entry whose base name equals one of these.
var deniedBaseExact = map[string]string{
	".env":      "local secrets/environment",
	".netrc":    "local credentials",
	".npmrc":    "may carry registry tokens",
	".pypirc":   "may carry upload tokens",
	".htpasswd": "local credentials",
	".DS_Store": "OS metadata junk",
	"Thumbs.db": "OS metadata junk",
}

// deniedBasePrefix denies an entry whose base name starts with one of these
// (matched after deniedBaseExact, so the .env.example exemption below wins).
var deniedBasePrefix = []struct {
	prefix string
	reason string
}{
	{".env.", "local secrets/environment"},
	{"id_rsa", "private key"},
	{"id_dsa", "private key"},
	{"id_ecdsa", "private key"},
	{"id_ed25519", "private key"},
}

// deniedBaseSuffix denies an entry whose base name ends with one of these.
var deniedBaseSuffix = []struct {
	suffix string
	reason string
}{
	{".pem", "certificate/key material"},
	{".key", "key material"},
	{".pfx", "key container"},
	{".p12", "key container"},
	{".jks", "java key container"},
	{".kdbx", "password database"},
	{".keystore", "key container"},
}

// envExampleExempt is the single documented exemption mirroring .gitignore's
// `!.env.example`: a field-name example file is not a secret.
const envExampleExempt = ".env.example"

// secretMarkers are denied by CONTENT in regular files (bounded streaming scan).
// PEM private-key markers only — see the package comment for why AWS-style key
// ids are intentionally not content-scanned.
var secretMarkers = [][]byte{
	privateKeyMarker("OPENSSH"),
	pgpPrivateKeyMarker(),
	privateKeyMarker("RSA"),
	privateKeyMarker("DSA"),
	privateKeyMarker("EC"),
	privateKeyMarker("ENCRYPTED"),
	privateKeyMarker(""),
}

// privateKeyMarker deliberately constructs the marker at runtime instead of
// embedding the full marker bytes in this source file. The exporter scans its
// own committed source, so a literal marker here would make a clean repository
// self-trigger the secret-content backstop.
func privateKeyMarker(kind string) []byte {
	if kind == "" {
		return []byte("-----BEGIN " + "PRIVATE KEY-----")
	}
	return []byte("-----BEGIN " + kind + " " + "PRIVATE KEY-----")
}

func pgpPrivateKeyMarker() []byte {
	return []byte("-----BEGIN PGP " + "PRIVATE KEY BLOCK-----")
}

// ManifestEntry is one regular file in the export.
type ManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Denial is one fail-safe trip: the path, the rule class and why.
type Denial struct {
	Path   string `json:"path"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
}

// DeniedPathError is the typed fail-safe trip: the source material contains
// denied content, so no archive or manifest is published.
type DeniedPathError struct {
	Denials []Denial
}

func (e *DeniedPathError) Error() string {
	paths := make([]string, 0, len(e.Denials))
	for _, d := range e.Denials {
		paths = append(paths, d.Path+" ("+d.Rule+")")
	}
	return fmt.Sprintf("export denylist tripped on %d path(s); no archive published: %s",
		len(e.Denials), strings.Join(paths, ", "))
}

// Manifest describes one clean export.
type Manifest struct {
	Schema        string          `json:"schema"`
	SourceCommit  string          `json:"source_commit"`
	RequestedRef  string          `json:"requested_ref"`
	Archive       string          `json:"archive"`
	ArchiveSHA256 string          `json:"archive_sha256"`
	ArchiveSize   int64           `json:"archive_size"`
	FileCount     int             `json:"file_count"`
	DeniedCount   int             `json:"denied_count"`
	Entries       []ManifestEntry `json:"entries"`
}

// ExportResult points at the published artifacts.
type ExportResult struct {
	Commit        string
	ArchivePath   string
	ManifestPath  string
	ArchiveSHA256 string
	FileCount     int
}

// RunExport produces the archive + manifest for the pinned commit.
func RunExport(ctx context.Context, repoDir, requestedRef, outDir string, logf func(string, ...any)) (*ExportResult, error) {
	if repoDir == "" {
		return nil, errors.New("repo dir is empty")
	}
	if requestedRef == "" {
		requestedRef = "HEAD"
	}
	logf = nilSafeLogf(logf)

	// HEAD is the release/default face. Refuse to export it from a dirty
	// worktree because that creates a misleading candidate identity: the caller
	// may think local changes are in the baseline while git archive can only see
	// the committed tree. Explicit immutable commit SHAs remain exportable from a
	// dirty checkout for forensic/reproducibility use.
	if requestedRef == "HEAD" {
		if err := ensureCleanWorktree(ctx, repoDir); err != nil {
			return nil, err
		}
	}

	commit, err := resolveCommit(ctx, repoDir, requestedRef)
	if err != nil {
		return nil, fmt.Errorf("resolve source commit: %w", err)
	}
	logf("source commit pinned: %s (requested ref %q)", commit, requestedRef)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	// Stage the archive in a temporary file first; it is only renamed to its
	// deterministic final name after the full denylist scan passes.
	tmp, err := os.CreateTemp(outDir, ".tally-export-*.tmp.tar")
	if err != nil {
		return nil, fmt.Errorf("create temp archive: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	logf("streaming git archive of %s …", commit[:12])
	if err := streamGitArchive(ctx, repoDir, commit, tmp); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("write temp archive: %w", err)
	}

	logf("scanning archive manifest against denylist …")
	archiveSHA, archiveSize, entries, denials, err := scanArchiveFile(tmpName)
	if err != nil {
		return nil, err
	}
	if len(denials) > 0 {
		return nil, &DeniedPathError{Denials: denials}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("export of %s contains no files; refusing to publish an empty archive", commit)
	}

	finalName := fmt.Sprintf("tally-src-%s.tar", commit[:12])
	finalPath := filepath.Join(outDir, finalName)
	if err := publishFile(tmpName, finalPath, archiveSize); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Schema:        ManifestSchema,
		SourceCommit:  commit,
		RequestedRef:  requestedRef,
		Archive:       finalName,
		ArchiveSHA256: archiveSHA,
		ArchiveSize:   archiveSize,
		FileCount:     len(entries),
		DeniedCount:   0,
		Entries:       entries,
	}
	manifestPath, err := writeManifestFile(outDir, finalName+".manifest.json", manifest)
	if err != nil {
		return nil, err
	}

	logf("published %s (%d files, sha256 %s…)", finalName, len(entries), archiveSHA[:12])
	return &ExportResult{
		Commit:        commit,
		ArchivePath:   finalPath,
		ManifestPath:  manifestPath,
		ArchiveSHA256: archiveSHA,
		FileCount:     len(entries),
	}, nil
}

// ensureCleanWorktree keeps the default HEAD export honest: source identity
// and working-copy identity must be the same candidate. Ignored runtime state
// is intentionally excluded by git status and by the archive denylist.
func ensureCleanWorktree(ctx context.Context, repoDir string) error {
	out, err := gitOutput(ctx, repoDir, "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return fmt.Errorf("inspect worktree before HEAD export: %w", err)
	}
	if dirty := strings.TrimSpace(string(out)); dirty != "" {
		lines := strings.Split(dirty, "\n")
		if len(lines) > 5 {
			lines = append(lines[:5], "...")
		}
		return fmt.Errorf("refuse HEAD export from dirty worktree; commit/stash changes or export an explicit commit SHA: %s", strings.Join(lines, "; "))
	}
	return nil
}

// resolveCommit pins requestedRef to one full commit sha.
func resolveCommit(ctx context.Context, repoDir, ref string) (string, error) {
	out, err := gitOutput(ctx, repoDir, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("ref %q: %w", ref, err)
	}
	commit := strings.TrimSpace(string(out))
	if !isFullCommitSHA(commit) {
		return "", fmt.Errorf("ref %q resolved to %q, which is not a full commit sha", ref, commit)
	}
	return commit, nil
}

// streamGitArchive writes `git archive --format=tar <commit>` to w using
// structured argv (no shell, no caller-controlled interpolation beyond the
// already-validated full commit sha).
func streamGitArchive(ctx context.Context, repoDir, commit string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, gitExe(), "-C", repoDir, "archive", "--format=tar", commit)
	cmd.Stdout = w
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w (stderr: %s)", commit[:12], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// scanArchiveFile hashes the whole archive bytes and walks every tar entry
// through the manifest/denylist scan.
func scanArchiveFile(path string) (archiveSHA string, size int64, entries []ManifestEntry, denials []Denial, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, nil, nil, fmt.Errorf("open archive for scan: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	size, err = io.Copy(hasher, f)
	if err != nil {
		return "", 0, nil, nil, fmt.Errorf("hash archive: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", 0, nil, nil, fmt.Errorf("rewind archive: %w", err)
	}

	entries, denials, err = inspectEntries(f)
	if err != nil {
		return "", 0, nil, nil, err
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, entries, denials, nil
}

// inspectEntries is the pure tar walk: it returns the regular-file manifest
// entries (sorted by path) and every denial found.
func inspectEntries(r io.Reader) ([]ManifestEntry, []Denial, error) {
	var (
		entries  []ManifestEntry
		denials  []Denial
		seen     = map[string]bool{}
		contents = newSecretScanner()
	)
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("walk archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue // pax global metadata, not a filesystem entry
		}
		if d := evaluateEntry(hdr); d != nil {
			denials = append(denials, *d)
			continue // never read denied content into the manifest
		}
		key, _ := canonicalArchivePath(hdr.Name) // evaluateEntry already validated it
		if seen[key] {
			denials = append(denials, Denial{Path: hdr.Name, Rule: "duplicate-entry",
				Reason: "archive contains the same canonical path twice"})
			continue
		}
		seen[key] = true
		if hdr.Typeflag != tar.TypeReg {
			continue // directories are validated but not manifested
		}

		fileHash := sha256.New()
		contents.reset()
		n, err := io.Copy(io.MultiWriter(fileHash, contents), tr)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", hdr.Name, err)
		}
		if n != hdr.Size {
			return nil, nil, fmt.Errorf("read %s: got %d bytes, header says %d", hdr.Name, n, hdr.Size)
		}
		if d := contents.denial(hdr.Name); d != nil {
			denials = append(denials, *d)
			continue
		}
		entries = append(entries, ManifestEntry{
			Path:   hdr.Name,
			Size:   hdr.Size,
			SHA256: hex.EncodeToString(fileHash.Sum(nil)),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, denials, nil
}

// canonicalArchivePath validates an archive path *before* cleaning it. Looking
// only at path.Clean(name) is insufficient because inputs such as a/../b can
// erase the traversal segment before policy sees it. A trailing slash is
// accepted for directory headers; all other non-canonical spellings are denied.
func canonicalArchivePath(name string) (string, *Denial) {
	if name == "" {
		return "", &Denial{Path: name, Rule: "path-safety", Reason: "empty path"}
	}
	if strings.Contains(name, "\\") {
		return "", &Denial{Path: name, Rule: "path-safety", Reason: "unexpected path separator"}
	}
	if strings.HasPrefix(name, "/") {
		return "", &Denial{Path: name, Rule: "path-safety", Reason: "absolute path"}
	}
	raw := strings.TrimSuffix(name, "/")
	if raw == "" {
		return "", &Denial{Path: name, Rule: "path-safety", Reason: "empty path"}
	}
	for _, seg := range strings.Split(raw, "/") {
		switch seg {
		case "", ".":
			return "", &Denial{Path: name, Rule: "path-safety", Reason: "non-canonical path segment"}
		case "..":
			return "", &Denial{Path: name, Rule: "path-safety", Reason: "parent-directory segment"}
		}
	}
	clean := path.Clean(raw)
	if clean == "." || clean == "" || clean != raw {
		return "", &Denial{Path: name, Rule: "path-safety", Reason: "non-canonical path"}
	}
	return clean, nil
}

// evaluateEntry applies the full denylist to one tar entry; nil means allowed.
func evaluateEntry(hdr *tar.Header) *Denial {
	name := hdr.Name

	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeDir:
	default:
		return &Denial{Path: name, Rule: "entry-type",
			Reason: fmt.Sprintf("non-regular entry type %q; export allows regular files and directories only", rune(hdr.Typeflag))}
	}

	clean, pathDenial := canonicalArchivePath(name)
	if pathDenial != nil {
		return pathDenial
	}
	segments := strings.Split(clean, "/")
	for _, seg := range segments {
		if reason, ok := deniedDirSegments[seg]; ok {
			return &Denial{Path: name, Rule: "denied-dir", Reason: reason}
		}
	}
	if hdr.Typeflag != tar.TypeReg {
		return nil
	}

	base := segments[len(segments)-1]
	if base == envExampleExempt {
		return nil
	}
	if reason, ok := deniedBaseExact[base]; ok {
		return &Denial{Path: name, Rule: "denied-file", Reason: reason}
	}
	for _, p := range deniedBasePrefix {
		if strings.HasPrefix(base, p.prefix) {
			return &Denial{Path: name, Rule: "denied-file", Reason: p.reason}
		}
	}
	for _, s := range deniedBaseSuffix {
		if strings.HasSuffix(base, s.suffix) {
			return &Denial{Path: name, Rule: "denied-file", Reason: s.reason}
		}
	}
	return nil
}

// secretScanner is a bounded-memory streaming scan for secretMarkers with a
// carry so a marker split across chunk boundaries is still found.
type secretScanner struct {
	carry []byte
	hit   string
}

func newSecretScanner() *secretScanner { return &secretScanner{} }

func (s *secretScanner) reset() { s.carry = nil; s.hit = "" }

func (s *secretScanner) Write(p []byte) (int, error) {
	if s.hit != "" {
		return len(p), nil
	}
	buf := make([]byte, 0, len(s.carry)+len(p))
	buf = append(buf, s.carry...)
	buf = append(buf, p...)
	for _, m := range secretMarkers {
		if bytes.Contains(buf, m) {
			s.hit = string(m)
			return len(p), nil
		}
	}
	if len(buf) >= maxSecretMarkerLen {
		s.carry = append(s.carry[:0], buf[len(buf)-maxSecretMarkerLen+1:]...)
	} else {
		s.carry = append(s.carry[:0], buf...)
	}
	return len(p), nil
}

func (s *secretScanner) denial(path string) *Denial {
	if s.hit == "" {
		return nil
	}
	return &Denial{Path: path, Rule: "secret-content",
		Reason: "content contains private-key marker " + s.hit}
}

// publishFile renames the scanned temp file to its final name; an existing
// identical artifact makes the export idempotent (byte-for-byte same commit →
// same output), a differing one is replaced.
func publishFile(tmpName, finalPath string, size int64) error {
	if existing, err := os.Stat(finalPath); err == nil {
		if existing.Size() == size {
			existingSHA, err := fileSHA256(finalPath)
			if err == nil {
				tmpSHA, err := fileSHA256(tmpName)
				if err == nil && existingSHA == tmpSHA {
					return nil // already published, byte-identical
				}
			}
		}
		if err := os.Remove(finalPath); err != nil {
			return fmt.Errorf("replace stale export artifact: %w", err)
		}
	}
	if err := os.Rename(tmpName, finalPath); err != nil {
		return fmt.Errorf("publish archive: %w", err)
	}
	return nil
}

func writeManifestFile(outDir, name string, manifest *Manifest) (string, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	data = append(data, '\n')
	finalPath := filepath.Join(outDir, name)
	tmp, err := os.CreateTemp(outDir, ".tally-manifest-*.tmp.json")
	if err != nil {
		return "", fmt.Errorf("create temp manifest: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("write temp manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("write temp manifest: %w", err)
	}
	if err := os.Rename(tmpName, finalPath); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("publish manifest: %w", err)
	}
	return finalPath, nil
}

func gitOutput(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, gitExe(), append([]string{"-C", repoDir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func gitExe() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	return "git"
}

func isFullCommitSHA(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func nilSafeLogf(logf func(string, ...any)) func(string, ...any) {
	if logf != nil {
		return logf
	}
	return func(string, ...any) {}
}
