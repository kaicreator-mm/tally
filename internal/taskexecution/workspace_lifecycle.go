package taskexecution

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"tally/internal/runner"
)

func unsafeCleanupPath(kind EnvironmentKind, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if kind == EnvironmentLocal {
		clean := filepath.Clean(raw)
		if !filepath.IsAbs(clean) {
			return true
		}
		volume := filepath.VolumeName(clean)
		root := string(filepath.Separator)
		if volume != "" {
			root = volume + string(filepath.Separator)
		}
		return strings.EqualFold(clean, root)
	}
	clean := path.Clean(raw)
	return !path.IsAbs(clean) || clean == "/"
}

// NewGitWorkspaceRecreator builds the v1.5.2 lost-workspace recovery action.
// repo must be meaningful inside the selected environment (for example a Git
// URL or environment-visible path). Credentials must be supplied by that
// environment's normal Git credential mechanism, not embedded in provenance.
func NewGitWorkspaceRecreator(exec runner.Runner, repo string) (WorkspaceRecreator, error) {
	if exec == nil {
		return nil, errors.New("taskexecution: workspace recreator requires a runner")
	}
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, errors.New("taskexecution: workspace recreator requires an environment-visible Git repo")
	}
	return func(ctx context.Context, w WorkspaceRef) error {
		if err := w.Validate(); err != nil {
			return err
		}
		if unsafeCleanupPath(w.Environment, w.Path) {
			return &EnvironmentError{Kind: FailureCleanup, Environment: w.Environment, Target: w.Target, Operation: "workspace.recreate", Err: fmt.Errorf("refusing unsafe workspace path %q", w.Path)}
		}
		if err := runWorkspaceCommand(ctx, exec, w, "cleanup", []string{"rm", "-rf", "--", w.Path}); err != nil {
			return err
		}
		if err := runWorkspaceCommand(ctx, exec, w, "clone", []string{"git", "clone", "--no-checkout", "--", repo, w.Path}); err != nil {
			return err
		}
		if err := runWorkspaceCommand(ctx, exec, w, "checkout", []string{"git", "-C", w.Path, "checkout", "--detach", w.BaseRevision}); err != nil {
			_, _ = exec.Run(context.WithoutCancel(ctx), runner.Command{Argv: []string{"rm", "-rf", "--", w.Path}})
			return err
		}
		return nil
	}, nil
}

func runWorkspaceCommand(ctx context.Context, exec runner.Runner, w WorkspaceRef, operation string, argv []string) error {
	res, err := exec.Run(ctx, runner.Command{Argv: argv})
	if err != nil {
		// Runner errors mean the environment/transport did not execute the
		// command. Do not misclassify an SSH disconnect or missing Docker daemon
		// as a lost workspace; callers must not recreate again in response.
		return &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: w.Environment, Target: w.Target, Operation: "workspace.recreate." + operation, Err: err}
	}
	if res.ExitCode != 0 {
		// A command that actually ran but could not clean/clone/checkout the
		// accepted workspace is a workspace-level recovery failure.
		return &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: w.Environment, Target: w.Target, Operation: "workspace.recreate." + operation, Err: fmt.Errorf("exit %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))}
	}
	return nil
}
