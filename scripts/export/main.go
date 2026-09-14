// P6-06 — Safe project export CLI (docs/tasks/v1.4.1/tasks/P6/P6-06_safe_project_export.md).
//
// Usage:
//
//	go run ./scripts/export [-repo <dir>] [-ref <commit-ish>] [-out <dir>]
//
// Flags:
//
//	-repo  repository to export (default ".")
//	-ref   commit-ish to pin the export to (default "HEAD"; resolved to a full
//	       commit sha and recorded in the manifest)
//	-out   output directory (default "<repo>/bin/export"; a build-output
//	       location that is itself on the export denylist)
//
// Outputs (only if the manifest/denylist scan is clean):
//
//	<out>/tally-src-<commit12>.tar                deterministic source archive
//	<out>/tally-src-<commit12>.tar.manifest.json  pinned manifest (entries + sha256)
//
// Exit codes:
//
//	0  export published
//	1  operational error (git failure, IO failure)
//	2  fail-safe trip: denylist/secret scan denied the export; nothing published
//	3  usage error
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	repo := flag.String("repo", ".", "repository directory to export")
	ref := flag.String("ref", "HEAD", "commit-ish to pin the export to (recorded in the manifest)")
	out := flag.String("out", "", "output directory (default <repo>/bin/export)")
	flag.Parse()

	logf := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "[export] "+format+"\n", args...)
	}

	outDir := *out
	if outDir == "" {
		absRepo, err := filepath.Abs(*repo)
		if err != nil {
			logf("resolve repo dir: %v", err)
			os.Exit(1)
		}
		outDir = filepath.Join(absRepo, "bin", "export")
	}

	res, err := RunExport(context.Background(), *repo, *ref, outDir, logf)
	if err != nil {
		var denied *DeniedPathError
		if errors.As(err, &denied) {
			logf("FAIL-SAFE: export denied, nothing published")
			for _, d := range denied.Denials {
				logf("  denied %-50s rule=%s reason=%s", d.Path, d.Rule, d.Reason)
			}
			os.Exit(2)
		}
		logf("export failed: %v", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n%s\ncommit=%s files=%d archive_sha256=%s\n",
		res.ArchivePath, res.ManifestPath, res.Commit, res.FileCount, res.ArchiveSHA256)
}
