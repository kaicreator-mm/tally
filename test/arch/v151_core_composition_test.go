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
