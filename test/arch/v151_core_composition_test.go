package arch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// v151CoreServerSource returns the concatenated production source of the
// cmd/v15server package. Composition may legitimately be split across several
// files in the package (e.g. v1.5.2 environment composition); the guards apply
// to the package as a whole.
func v151CoreServerSource(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir("../../cmd/v15server")
	if err != nil {
		t.Fatalf("read v1.5 Core server package: %v", err)
	}
	var b strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../cmd/v15server", name))
		if err != nil {
			t.Fatalf("read v1.5 Core server file %s: %v", name, err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestV151CoreServerDoesNotDependOnLegacyQueueOrPostgresStore(t *testing.T) {
	source := v151CoreServerSource(t)
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

func TestV151CoreServerMakesHarnessedLocalSchedulerAgentExecutorReachable(t *testing.T) {
	source := v151CoreServerSource(t)
	for _, required := range []string{
		"taskexecution.NewRuntimeExecutor",
		"controlplane.NewExecutionOnlyHarness",
		"controlplane.NewServiceWithHarness",
		"controlplane.NewGitTaskRequestFactory",
		"controlplane.NewLocalScheduler",
		"controlplane.NewSchedulerHTTPHandler",
		"TALLY_V15_SCHEDULER_MAX_CONCURRENT",
		"local_scheduler+harness+agent_executor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("v1.5 Core production execution path is missing %q", required)
		}
	}
	if strings.Contains(source, "NewBrownfieldHarness") {
		t.Fatal("SQLite-first v1.5 Core must not pull the legacy PostgreSQL-backed BrownfieldHarness into default production composition")
	}
}

func TestV151ExecutionOnlyHarnessIsFailClosed(t *testing.T) {
	data, err := os.ReadFile("../../internal/controlplane/execution_only_harness.go")
	if err != nil {
		t.Fatalf("read execution-only harness: %v", err)
	}
	source := string(data)
	for _, required := range []string{
		"AgentExecutor",
		"independent verification/review workflow unavailable",
		"ExecutionCompleted",
		"Accepted",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("execution-only harness is missing fail-closed guard %q", required)
		}
	}
	if strings.Contains(source, "Accepted: true") {
		t.Fatal("execution-only harness must never manufacture acceptance from an Agent claim")
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
		"pinnedCommit",
		"bundle.SourceCommit",
		"bundle.Task.Contract.RevisionHash()",
		"RequiredCapabilities",
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