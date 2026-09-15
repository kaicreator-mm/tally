package arch

import (
	"os"
	"strings"
	"testing"
)

func TestV151CoreServerDoesNotDependOnLegacyQueueOrPostgresStore(t *testing.T) {
	data, err := os.ReadFile("../../cmd/v15server/main.go")
	if err != nil {
		t.Fatalf("read v1.5 Core server: %v", err)
	}
	source := string(data)
	for _, forbidden := range []string{
		`"tally/internal/jobs"`,
		`"tally/internal/store"`,
		"DATABASE_URL",
		"SKIP LOCKED",
		"heartbeat",
		"lease",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("v1.5 Core composition contains forbidden legacy dependency %q", forbidden)
		}
	}
	if !strings.Contains(source, `TALLY_V15_PERSISTENCE`) || !strings.Contains(source, `profile.SQLite`) {
		t.Fatal("v1.5 Core server must select the explicit SQLite-first persistence profile")
	}
}

func TestV151CoreServerMakesLocalSchedulerAgentExecutorReachable(t *testing.T) {
	data, err := os.ReadFile("../../cmd/v15server/main.go")
	if err != nil {
		t.Fatalf("read v1.5 Core server: %v", err)
	}
	source := string(data)
	for _, required := range []string{
		"taskexecution.NewRuntimeExecutor",
		"controlplane.NewGitTaskRequestFactory",
		"controlplane.NewLocalScheduler",
		"controlplane.NewSchedulerHTTPHandler",
		"TALLY_V15_SCHEDULER_MAX_CONCURRENT",
		"local_scheduler+agent_executor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("v1.5 Core production execution path is missing %q", required)
		}
	}
	if strings.Contains(source, "NewBrownfieldHarness") {
		t.Fatal("SQLite-first v1.5 Core must not pull the legacy PostgreSQL-backed BrownfieldHarness into default production composition")
	}
}

func TestV151SchedulerRequestFactoryUsesPinnedExistingAuthority(t *testing.T) {
	data, err := os.ReadFile("../../internal/controlplane/git_request_factory.go")
	if err != nil {
		t.Fatalf("read scheduler request factory: %v", err)
	}
	source := string(data)
	for _, required := range []string{
		"artifacts.LoadBundle",
		"TaskExecutionRequest",
		"bundle.SourceCommit",
		"bundle.Task.Contract.RevisionHash()",
		"request.Validate()",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("request factory is missing frozen-authority guard %q", required)
		}
	}
	for _, forbidden := range []string{"jobs.Queue", "lease_until", "heartbeat", "SKIP LOCKED"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("request factory contains forbidden second-queue primitive %q", forbidden)
		}
	}
}

func TestV151SchedulerDoesNotOwnDatabaseQueueSemantics(t *testing.T) {
	data, err := os.ReadFile("../../internal/controlplane/scheduler.go")
	if err != nil {
		t.Fatalf("read scheduler: %v", err)
	}
	source := strings.ToLower(string(data))
	for _, forbidden := range []string{"skip locked", "heartbeat", "stale takeover", "lease_until", "select for update"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("local scheduler contains forbidden DB-queue correctness primitive %q", forbidden)
		}
	}
}