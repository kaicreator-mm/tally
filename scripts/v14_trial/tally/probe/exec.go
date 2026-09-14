// exec.go: the probe's one local process runner (go test for the bug_fix
// driver sandbox). Structured argv, no shell; stdout/stderr captured verbatim.
package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

type execResult struct {
	stdout string
	stderr string
}

func runLocal(ctx context.Context, argv []string, dir string) (execResult, error) {
	c := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // argv is structural, never a shell string
	c.Dir = dir
	c.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	runErr := c.Run()
	return execResult{stdout: stdout.String(), stderr: stderr.String()}, runErr
}
