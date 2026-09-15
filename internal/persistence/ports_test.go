package persistence

import (
	"encoding/json"
	"testing"
	"time"

	"tally/internal/engineering"
)

func TestNewDocumentOwnsPayloadAndTime(t *testing.T) {
	payload := json.RawMessage(`{"state":"ready"}`)
	at := time.Date(2026, 9, 15, 10, 0, 0, 123, time.FixedZone("test", 8*60*60))
	doc, err := NewDocument(engineering.ID("project-1"), engineering.ID("record-1"), payload, at)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	payload[2] = 'X'
	if string(doc.Payload) != `{"state":"ready"}` {
		t.Fatalf("document retained caller payload alias: %s", doc.Payload)
	}
	if doc.RecordedAt.Location() != time.UTC {
		t.Fatalf("recorded time must be normalized to UTC: %v", doc.RecordedAt.Location())
	}
}

func TestDocumentRejectsInvalidEnvelope(t *testing.T) {
	cases := []Document{
		{ProjectID: "", RecordID: "record-1", Payload: json.RawMessage(`{}`), RecordedAt: time.Now()},
		{ProjectID: "project-1", RecordID: "", Payload: json.RawMessage(`{}`), RecordedAt: time.Now()},
		{ProjectID: "project-1", RecordID: "record-1", Payload: json.RawMessage(`not-json`), RecordedAt: time.Now()},
		{ProjectID: "project-1", RecordID: "record-1", Payload: json.RawMessage(`{}`)},
	}
	for i, tc := range cases {
		if err := tc.Verify(); err == nil {
			t.Fatalf("case %d: expected validation failure", i)
		}
	}
}
