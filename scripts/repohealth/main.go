// Command repohealth rejects tracked local state, build output and credential-shaped files.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var deniedSegments = map[string]bool{
	".e2e": true, ".codex": true, ".worktrees": true, ".codegraph": true,
	"bin": true, ".ssh": true, ".aws": true, ".gnupg": true,
}

func main() {
	cmd := exec.Command("git", "ls-files", "-z")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repohealth: git ls-files: %v\n", err)
		os.Exit(1)
	}
	var bad []string
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		name := filepath.ToSlash(string(raw))
		parts := strings.Split(name, "/")
		for _, p := range parts {
			if deniedSegments[p] {
				bad = append(bad, name+" (runtime/build segment "+p+")")
				break
			}
		}
		base := parts[len(parts)-1]
		lower := strings.ToLower(base)
		if base == ".env" || (strings.HasPrefix(base, ".env.") && base != ".env.example") ||
			strings.HasPrefix(lower, "id_rsa") || strings.HasPrefix(lower, "id_dsa") ||
			strings.HasPrefix(lower, "id_ecdsa") || strings.HasPrefix(lower, "id_ed25519") ||
			strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") ||
			strings.HasSuffix(lower, ".p12") || strings.HasSuffix(lower, ".pfx") {
			bad = append(bad, name+" (credential-shaped tracked file)")
		}
	}
	if len(bad) > 0 {
		for _, b := range bad {
			fmt.Fprintln(os.Stderr, "repohealth: DENY", b)
		}
		os.Exit(1)
	}
	fmt.Println("repohealth: PASS")
}
