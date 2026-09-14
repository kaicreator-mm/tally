// P6-06 tests — archive inspection + fail-safe negative paths.
//
// Focused coverage for docs/tasks/v1.4.1/tasks/P6/P6-06_safe_project_export.md:
//   - the generated archive contains required source/docs and zero denied paths
//     (acceptance criterion), verified by re-opening the published tar and
//     re-checking every entry against the task-named denylist independently of
//     the scanner implementation;
//   - determinism: same commit → byte-identical archive and manifest;
//   - commit pinning: an explicit sha keeps resolving to that exact commit even
//     after history moves on (one pinned source commit, DL-04 reading);
//   - fail-safe: committed denied paths / secret content yield a typed
//     *DeniedPathError and publish nothing (deny wins over ignore);
//   - the tar walker rejects unsafe entries (symlink, traversal, backslash,
//     absolute, duplicate) and finds a private-key marker split across chunk
//     boundaries.
package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot locates the repository root from the package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Fatalf("repo root %s does not contain go.mod: %v", abs, err)
	}
	return abs
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return string(out)
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "-c", "user.name=tally-export-test", "-c", "user.email=test@example.invalid",
		"commit", "--no-gpg-sign", "-m", msg)
	return strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))
}

// initTempRepo creates a disposable git repository with an initial main branch.
func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	return dir
}

func mustFileSHA(t *testing.T, path string) string {
	t.Helper()
	sha, err := fileSHA256(path)
	if err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return sha
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func readManifest(t *testing.T, path string) *Manifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse manifest %s: %v", path, err)
	}
	return &m
}

// checkNoDeniedNames is an independent, scanner-agnostic re-check of the
// task-named denylist minimum, applied to raw tar entry names.
func checkNoDeniedNames(t *testing.T, where string, names []string) {
	t.Helper()
	deniedSegments := map[string]bool{
		".git": true, ".worktrees": true, ".e2e": true,
		".codegraph": true, ".codex": true, "bin": true, "node_modules": true,
	}
	for _, name := range names {
		if strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
			t.Fatalf("%s: unsafe entry name %q", where, name)
		}
		segs := strings.Split(path.Clean(name), "/")
		for _, seg := range segs {
			if seg == ".." {
				t.Fatalf("%s: traversal entry %q", where, name)
			}
			if deniedSegments[seg] {
				t.Fatalf("%s: denied segment in entry %q", where, name)
			}
		}
		if base := segs[len(segs)-1]; base == ".env" {
			t.Fatalf("%s: credential file entry %q", where, name)
		}
	}
}

func readArchiveEntries(t *testing.T, archivePath, wantCommit string) (allNames []string, regFiles map[string]bool) {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open archive %s: %v", archivePath, err)
	}
	defer f.Close()
	regFiles = map[string]bool{}
	tr := tar.NewReader(f)
	sawPaxCommit := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("walk archive %s: %v", archivePath, err)
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			// git archive's own pax metadata record ("pax_global_header");
			// Go parses it into PAXRecords and consumes the data section, and
			// its comment carries the pinned commit id — provenance, not a
			// filesystem entry. Assert it instead of failing on it.
			if hdr.Name != "pax_global_header" {
				t.Fatalf("%s: unexpected global header %q", archivePath, hdr.Name)
			}
			if hdr.PAXRecords["comment"] != wantCommit {
				t.Fatalf("pax global header does not pin commit %s: %v", wantCommit, hdr.PAXRecords)
			}
			sawPaxCommit = true
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeDir:
		default:
			t.Fatalf("%s: unexpected non-regular entry %q typeflag=%c", archivePath, hdr.Name, hdr.Typeflag)
		}
		allNames = append(allNames, hdr.Name)
		if hdr.Typeflag == tar.TypeReg {
			regFiles[hdr.Name] = true
		}
	}
	if !sawPaxCommit {
		t.Fatalf("%s: pax_global_header with commit id missing", archivePath)
	}
	return allNames, regFiles
}

// TestExportRealRepoContainsRequiredSourcesAndZeroDeniedPaths is the task
// acceptance criterion, executed against the real repository at HEAD.
func TestExportRealRepoContainsRequiredSourcesAndZeroDeniedPaths(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	res, err := RunExport(context.Background(), root, "HEAD", outDir, nil)
	if err != nil {
		t.Fatalf("RunExport: %v", err)
	}

	m := readManifest(t, res.ManifestPath)
	if m.Schema != ManifestSchema {
		t.Fatalf("manifest schema = %q, want %q", m.Schema, ManifestSchema)
	}
	if m.SourceCommit != res.Commit || len(m.SourceCommit) != 40 {
		t.Fatalf("manifest commit %q mismatch (result %q)", m.SourceCommit, res.Commit)
	}
	if m.DeniedCount != 0 || len(m.Entries) == 0 {
		t.Fatalf("manifest denied=%d entries=%d", m.DeniedCount, len(m.Entries))
	}

	// Manifest archive identity matches the published bytes.
	archiveSHA, err := fileSHA256(res.ArchivePath)
	if err != nil {
		t.Fatalf("hash archive: %v", err)
	}
	if archiveSHA != m.ArchiveSHA256 || archiveSHA != res.ArchiveSHA256 {
		t.Fatalf("archive sha mismatch: file=%s manifest=%s result=%s", archiveSHA, m.ArchiveSHA256, res.ArchiveSHA256)
	}

	// Archive inspection: re-open the tar, verify types and the exact
	// regular-file set against the manifest, and independently re-check
	// every entry name against the task-named denylist.
	allNames, regFiles := readArchiveEntries(t, res.ArchivePath, res.Commit)
	checkNoDeniedNames(t, res.ArchivePath, allNames)
	if len(regFiles) != m.FileCount || len(regFiles) != len(m.Entries) {
		t.Fatalf("tar regular files=%d, manifest files=%d entries=%d", len(regFiles), m.FileCount, len(m.Entries))
	}
	for _, e := range m.Entries {
		if !regFiles[e.Path] {
			t.Fatalf("manifest entry %q missing from archive", e.Path)
		}
	}

	// Required source/docs are present.
	for _, required := range []string{
		"go.mod",
		"go.sum",
		"Makefile",
		".gitattributes",
		"docs/tasks/v1.4.1/tasks/P6/P6-06_safe_project_export.md",
		"scripts/preflight/main.go",
	} {
		if !regFiles[required] {
			t.Fatalf("required file %q missing from export", required)
		}
	}
	for _, prefix := range []string{"cmd/", "internal/", "migrations/", "docs/", "scripts/", "web/", "test/", "runtimes/"} {
		found := false
		for _, e := range m.Entries {
			if strings.HasPrefix(e.Path, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("required source tree %q has no entries in export", prefix)
		}
	}
}

// TestExportIsDeterministicForSameCommit: same commit → byte-identical
// archive and manifest.
func TestExportIsDeterministicForSameCommit(t *testing.T) {
	root := repoRoot(t)
	out1, out2 := t.TempDir(), t.TempDir()

	res1, err := RunExport(context.Background(), root, "HEAD", out1, nil)
	if err != nil {
		t.Fatalf("first RunExport: %v", err)
	}
	res2, err := RunExport(context.Background(), root, "HEAD", out2, nil)
	if err != nil {
		t.Fatalf("second RunExport: %v", err)
	}

	sha1, err := fileSHA256(res1.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	sha2, err := fileSHA256(res2.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if sha1 != sha2 || sha1 != res1.ArchiveSHA256 || res1.ArchiveSHA256 != res2.ArchiveSHA256 {
		t.Fatalf("archive not deterministic: %s vs %s", sha1, sha2)
	}
	m1, err := os.ReadFile(res1.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := os.ReadFile(res2.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m1, m2) {
		t.Fatalf("manifest not deterministic for same commit:\n%s\n---\n%s", m1, m2)
	}
}

// TestExportPinsExactCommitFromRef: an explicit sha keeps exporting exactly
// that commit even after the branch moves on.
func TestExportPinsExactCommitFromRef(t *testing.T) {
	repo := initTempRepo(t)
	writeFile(t, repo, "src/a.txt", "a\n")
	sha1 := commitAll(t, repo, "first")
	out1 := t.TempDir()
	resA, err := RunExport(context.Background(), repo, sha1, out1, nil)
	if err != nil {
		t.Fatalf("export sha1: %v", err)
	}

	writeFile(t, repo, "src/b.txt", "b\n")
	commitAll(t, repo, "second")

	out2, out3 := t.TempDir(), t.TempDir()
	resB, err := RunExport(context.Background(), repo, "HEAD", out2, nil)
	if err != nil {
		t.Fatalf("export HEAD: %v", err)
	}
	resC, err := RunExport(context.Background(), repo, sha1, out3, nil)
	if err != nil {
		t.Fatalf("re-export sha1 after history moved: %v", err)
	}

	if resA.Commit != sha1 || resC.Commit != sha1 {
		t.Fatalf("pin violated: A=%s C=%s want %s", resA.Commit, resC.Commit, sha1)
	}
	if resB.Commit == sha1 {
		t.Fatalf("HEAD export did not advance to the second commit")
	}
	shaA := mustFileSHA(t, resA.ArchivePath)
	shaB := mustFileSHA(t, resB.ArchivePath)
	shaC := mustFileSHA(t, resC.ArchivePath)
	if shaA != shaC {
		t.Fatalf("explicit-sha export drifted: %s vs %s", shaA, shaC)
	}
	if shaB == shaA {
		t.Fatalf("HEAD export identical to first commit export; content should differ")
	}
	mA, err := os.ReadFile(resA.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	mC, err := os.ReadFile(resC.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mA, mC) {
		t.Fatalf("manifest for pinned commit not reproducible:\n%s\n---\n%s", mA, mC)
	}
}

// TestExportFailsSafeOnCommittedDeniedPaths: tracked denied material trips the
// denylist (deny wins over ignore) and nothing is published.
func TestExportFailsSafeOnCommittedDeniedPaths(t *testing.T) {
	repo := initTempRepo(t)
	writeFile(t, repo, "docs/readme.md", "# clean\n")
	writeFile(t, repo, "bin/tool.exe", "binary-ish\n")
	writeFile(t, repo, "sub/.env", "SECRET=1\n")
	writeFile(t, repo, ".e2e/state.json", "{}\n")
	commitAll(t, repo, "poison")

	outDir := t.TempDir()
	_, err := RunExport(context.Background(), repo, "HEAD", outDir, nil)
	if err == nil {
		t.Fatalf("RunExport succeeded despite committed denied paths")
	}
	var denied *DeniedPathError
	if !errors.As(err, &denied) {
		t.Fatalf("error is %T (%v), want *DeniedPathError", err, err)
	}
	got := map[string]string{}
	for _, d := range denied.Denials {
		got[d.Path] = d.Rule
	}
	for path, wantRule := range map[string]string{
		"bin/tool.exe":    "denied-dir",
		"sub/.env":        "denied-file",
		".e2e/state.json": "denied-dir",
	} {
		rule, ok := got[path]
		if !ok {
			t.Fatalf("denied path %q not reported; got %v", path, got)
		}
		if rule != wantRule {
			t.Fatalf("denied path %q rule=%s, want %s", path, rule, wantRule)
		}
	}
	if _, ok := got["docs/readme.md"]; ok {
		t.Fatalf("clean path docs/readme.md must not be denied")
	}

	// Nothing published: no archive, no manifest, no temp leftovers.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("fail-safe trip left artifacts in out dir: %v", entries)
	}
}

// TestExportFailsSafeOnSecretContentMarker: a private-key PEM block is denied
// by content even under an innocuous file name.
func TestExportFailsSafeOnSecretContentMarker(t *testing.T) {
	repo := initTempRepo(t)
	marker := string(privateKeyMarker("RSA"))
	writeFile(t, repo, "docs/notes.txt",
		"team notes\noops:\n"+marker+"\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n")
	commitAll(t, repo, "leak")

	outDir := t.TempDir()
	_, err := RunExport(context.Background(), repo, "HEAD", outDir, nil)
	var denied *DeniedPathError
	if !errors.As(err, &denied) {
		t.Fatalf("error is %T (%v), want *DeniedPathError", err, err)
	}
	found := false
	for _, d := range denied.Denials {
		if d.Path == "docs/notes.txt" && d.Rule == "secret-content" {
			found = true
		}
	}
	if !found {
		t.Fatalf("secret-content denial for docs/notes.txt missing; got %v", denied.Denials)
	}
}

// TestExportHeadRejectsDirtyWorktree pins candidate identity: the default HEAD
// export must not silently ignore local edits. An explicit immutable SHA remains
// available for forensic export from the same dirty checkout.
func TestExportHeadRejectsDirtyWorktree(t *testing.T) {
	repo := initTempRepo(t)
	writeFile(t, repo, "a.txt", "a\n")
	sha := commitAll(t, repo, "one")
	writeFile(t, repo, "local.txt", "uncommitted\n")

	if _, err := RunExport(context.Background(), repo, "HEAD", t.TempDir(), nil); err == nil || !strings.Contains(err.Error(), "dirty worktree") {
		t.Fatalf("dirty HEAD export should fail with candidate-identity error, got %v", err)
	}
	if _, err := RunExport(context.Background(), repo, sha, t.TempDir(), nil); err != nil {
		t.Fatalf("explicit immutable SHA should remain exportable from dirty checkout: %v", err)
	}
}

// TestRunExportRejectsUnknownRef: an unresolvable ref is an operational error,
// not a denial and not a published archive.
func TestRunExportRejectsUnknownRef(t *testing.T) {
	repo := initTempRepo(t)
	writeFile(t, repo, "a.txt", "a\n")
	commitAll(t, repo, "one")

	outDir := t.TempDir()
	_, err := RunExport(context.Background(), repo, "does-not-exist", outDir, nil)
	if err == nil {
		t.Fatalf("RunExport with unknown ref succeeded")
	}
	var denied *DeniedPathError
	if errors.As(err, &denied) {
		t.Fatalf("unknown ref must not be a denial: %v", err)
	}
	if !strings.Contains(err.Error(), "resolve source commit") {
		t.Fatalf("unexpected error shape: %v", err)
	}
}

// tarEntrySpec is one synthetic archive entry for the scanner unit test.
type tarEntrySpec struct {
	name     string
	typeflag byte
	body     []byte
	link     string
}

func buildTar(t *testing.T, entries []tarEntrySpec) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     0o644,
			Size:     int64(len(e.body)),
			Linkname: e.link,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if len(e.body) > 0 {
			if _, err := tw.Write(e.body); err != nil {
				t.Fatalf("write body %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	return buf.Bytes()
}

// TestInspectEntriesRejectsUnsafeArchiveEntries unit-covers the tar walker:
// denied names/types/traversal, duplicate entries, the .env.example exemption,
// benign prose, and a private-key marker split across read-chunk boundaries.
func TestInspectEntriesRejectsUnsafeArchiveEntries(t *testing.T) {
	marker := string(privateKeyMarker("RSA"))
	straddle := make([]byte, 32760+len(marker)+16)
	for i := range straddle {
		straddle[i] = 'A'
	}
	copy(straddle[32760:], marker) // marker straddles the 32KiB copy boundary

	data := buildTar(t, []tarEntrySpec{
		{name: "ok.txt", typeflag: tar.TypeReg, body: []byte("hello")},
		{name: "docs", typeflag: tar.TypeDir},
		{name: ".env.example", typeflag: tar.TypeReg, body: []byte("FIELD=\n")},
		{name: ".git/config", typeflag: tar.TypeReg, body: []byte("[core]\n")},
		{name: "bin/tool", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "sub/.env", typeflag: tar.TypeReg, body: []byte("SECRET=1")},
		{name: "id_rsa_backup", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "cert.pem", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "link", typeflag: tar.TypeSymlink, link: "/etc/passwd"},
		{name: "../escape.txt", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "a/../escape2.txt", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "./alias.txt", typeflag: tar.TypeReg, body: []byte("x")},
		{name: "dupe.txt", typeflag: tar.TypeReg, body: []byte("one")},
		{name: "dupe.txt", typeflag: tar.TypeReg, body: []byte("two")},
		{name: "straddle.bin", typeflag: tar.TypeReg, body: straddle},
		{name: "prose.txt", typeflag: tar.TypeReg, body: []byte("the private key material is stored in the vault")},
	})

	entries, denials, err := inspectEntries(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("inspectEntries: %v", err)
	}

	wantDenied := map[string]string{
		".git/config":      "denied-dir",
		"bin/tool":         "denied-dir",
		"sub/.env":         "denied-file",
		"id_rsa_backup":    "denied-file",
		"cert.pem":         "denied-file",
		"link":             "entry-type",
		"../escape.txt":    "path-safety",
		"a/../escape2.txt": "path-safety",
		"./alias.txt":      "path-safety",
		"dupe.txt":         "duplicate-entry",
		"straddle.bin":     "secret-content",
	}
	gotDenied := map[string]string{}
	for _, d := range denials {
		gotDenied[d.Path] = d.Rule
	}
	for p, rule := range wantDenied {
		if gotDenied[p] != rule {
			t.Fatalf("denial for %q = %q, want %q (all: %v)", p, gotDenied[p], rule, gotDenied)
		}
	}
	if len(gotDenied) != len(wantDenied) {
		t.Fatalf("unexpected extra denials: %v", gotDenied)
	}

	gotEntries := map[string]bool{}
	for _, e := range entries {
		gotEntries[e.Path] = true
	}
	for _, want := range []string{"ok.txt", ".env.example", "dupe.txt", "prose.txt"} {
		if !gotEntries[want] {
			t.Fatalf("clean entry %q missing from manifest: %v", want, gotEntries)
		}
	}
	if len(gotEntries) != 4 {
		t.Fatalf("manifest entries = %v, want exactly the 4 clean files", gotEntries)
	}
}

// TestEnvExampleExemptIsAllowed pins the single documented exemption.
func TestEnvExampleExemptIsAllowed(t *testing.T) {
	hdr := &tar.Header{Name: "config/.env.example", Typeflag: tar.TypeReg, Size: 0}
	if d := evaluateEntry(hdr); d != nil {
		t.Fatalf(".env.example must be exempt, got denial %+v", d)
	}
	hdr = &tar.Header{Name: "config/.env.production", Typeflag: tar.TypeReg, Size: 0}
	if d := evaluateEntry(hdr); d == nil || d.Rule != "denied-file" {
		t.Fatalf(".env.production must be denied-file, got %+v", d)
	}
}
