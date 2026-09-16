package controlplane

import (
	"strings"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

func TestV152NonLocalWorkspaceRootMustBeAbsolute(t *testing.T) {
	for _, root := range []string{"workspaces", "./workspaces", "../workspaces", "/"} {
		_, err := taskWorkspaceLocation(root, engineering.ID("TASK-1"), taskexecution.EnvironmentSSH)
		if err == nil || !strings.Contains(err.Error(), "absolute non-root") {
			t.Fatalf("root=%q err=%v", root, err)
		}
	}
	got, err := taskWorkspaceLocation("/srv/tally/workspaces", engineering.ID("TASK-1"), taskexecution.EnvironmentSSH)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/srv/tally/workspaces/TASK-1" {
		t.Fatalf("location=%q", got)
	}
}

func TestV152RemoteDSHImplicitCapabilitiesIncludeNetworkAndCredential(t *testing.T) {
	got := requiredEnvironmentCapabilities("dsh", taskexecution.EnvironmentSSH, nil)
	for _, want := range []taskexecution.EnvironmentCapability{
		taskexecution.CapabilityProcess,
		taskexecution.CapabilityCancel,
		taskexecution.CapabilityWorkspaceRead,
		taskexecution.CapabilityWorkspaceWrite,
		"git",
		"tool:dsh",
		"network",
		"secret:DEEPSEEK_API_KEY",
	} {
		if !hasEnvironmentCapability(got, want) {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}
