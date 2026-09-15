package persistencetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/persistence"
)

// Adapter is the complete document/transaction surface shared by v1.5
// persistence implementations. Concrete database mechanics are intentionally
// absent from this test contract.
type Adapter interface {
	persistence.ProjectStore
	persistence.RequirementStore
	persistence.PlanningStore
	persistence.TaskGraphStore
	persistence.TaskContractStore
	persistence.RunStore
	persistence.FactStore
	persistence.EvidenceStore
	persistence.ActionAuditStore
	persistence.OutcomeStore
	persistence.TransactionBoundary
}

// Run exercises the Core-visible persistence semantics that every adapter must
// satisfy. Adapter-specific migration/restart behavior belongs in adapter tests.
func Run(t *testing.T, adapter Adapter) {
	t.Helper()
	ctx := context.Background()
	at := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)

	mutable := []struct {
		name   string
		id     engineering.ID
		save   func(context.Context, persistence.Document) error
		load   func(context.Context, engineering.ID, engineering.ID) (persistence.Document, error)
		isProj bool
	}{
		{name: "project", id: "project-conformance", isProj: true, save: adapter.SaveProject, load: func(ctx context.Context, projectID, _ engineering.ID) (persistence.Document, error) { return adapter.LoadProject(ctx, projectID) }},
		{name: "requirements", id: "requirements-1", save: adapter.SaveRequirements, load: adapter.LoadRequirements},
		{name: "planning", id: "planning-1", save: adapter.SavePlanning, load: adapter.LoadPlanning},
		{name: "task-graph", id: "graph-1", save: adapter.SaveTaskGraph, load: adapter.LoadTaskGraph},
		{name: "task-contract", id: "contract-1", save: adapter.SaveTaskContract, load: adapter.LoadTaskContract},
		{name: "run", id: "run-1", save: adapter.SaveRun, load: adapter.LoadRun},
	}

	for _, tc := range mutable {
		tc := tc
		t.Run("mutable/"+tc.name, func(t *testing.T) {
			projectID := engineering.ID("project-conformance")
			recordID := tc.id
			if tc.isProj {
				recordID = projectID
			}
			doc := mustDocument(t, projectID, recordID, fmt.Sprintf(`{"kind":%q,"revision":1}`, tc.name), at)
			if err := tc.save(ctx, doc); err != nil {
				t.Fatalf("save: %v", err)
			}
			got, err := tc.load(ctx, projectID, recordID)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if string(got.Payload) != string(doc.Payload) || !got.RecordedAt.Equal(at) {
				t.Fatalf("round trip mismatch: got=%+v want=%+v", got, doc)
			}

			updated := mustDocument(t, projectID, recordID, fmt.Sprintf(`{"kind":%q,"revision":2}`, tc.name), at.Add(time.Second))
			if err := tc.save(ctx, updated); err != nil {
				t.Fatalf("update: %v", err)
			}
			got, err = tc.load(ctx, projectID, recordID)
			if err != nil {
				t.Fatalf("load updated: %v", err)
			}
			if string(got.Payload) != string(updated.Payload) {
				t.Fatalf("updated payload mismatch: got=%s want=%s", got.Payload, updated.Payload)
			}
		})
	}

	appendOnly := []struct {
		name   string
		id     engineering.ID
		append func(context.Context, persistence.Document) error
		load   func(context.Context, engineering.ID, engineering.ID) (persistence.Document, error)
	}{
		{name: "fact", id: "fact-1", append: adapter.AppendFact, load: adapter.LoadFact},
		{name: "evidence", id: "evidence-1", append: adapter.AppendEvidence, load: adapter.LoadEvidence},
		{name: "action", id: "action-1", append: adapter.AppendAction, load: adapter.LoadAction},
		{name: "outcome", id: "outcome-1", append: adapter.AppendOutcome, load: adapter.LoadOutcome},
	}

	for _, tc := range appendOnly {
		tc := tc
		t.Run("append-only/"+tc.name, func(t *testing.T) {
			projectID := engineering.ID("project-conformance")
			first := mustDocument(t, projectID, tc.id, `{"revision":1}`, at)
			if err := tc.append(ctx, first); err != nil {
				t.Fatalf("append: %v", err)
			}
			second := mustDocument(t, projectID, tc.id, `{"revision":2}`, at.Add(time.Second))
			if err := tc.append(ctx, second); !errors.Is(err, persistence.ErrImmutable) {
				t.Fatalf("duplicate append = %v, want ErrImmutable", err)
			}
			got, err := tc.load(ctx, projectID, tc.id)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if string(got.Payload) != string(first.Payload) {
				t.Fatalf("append-only record mutated: got=%s want=%s", got.Payload, first.Payload)
			}
		})
	}

	t.Run("transaction-rollback", func(t *testing.T) {
		rollback := errors.New("persistencetest rollback")
		doc := mustDocument(t, "project-conformance", "evidence-tx-rollback", `{"passed":true}`, at)
		err := adapter.WithinTransaction(ctx, func(stores persistence.Stores) error {
			if err := stores.Verify(); err != nil {
				return err
			}
			if err := stores.Evidence.AppendEvidence(ctx, doc); err != nil {
				return err
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("transaction = %v, want rollback sentinel", err)
		}
		if _, err := adapter.LoadEvidence(ctx, doc.ProjectID, doc.RecordID); !errors.Is(err, persistence.ErrNotFound) {
			t.Fatalf("rolled-back evidence load = %v, want ErrNotFound", err)
		}
	})
}

func mustDocument(t *testing.T, projectID, recordID engineering.ID, raw string, at time.Time) persistence.Document {
	t.Helper()
	doc, err := persistence.NewDocument(projectID, recordID, json.RawMessage(raw), at)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	return doc
}
