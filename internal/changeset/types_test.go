package changeset

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tally/internal/core"
)

// validSpec 返回一个满足全部必填项的构造输入(带改动文件,patch 身份可推导)。
func validSpec() Spec {
	return Spec{
		Project:      "proj-1",
		BaseRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Files: []ChangedFile{
			{Path: "cmd/server/main.go", Content: "package main\n", Executable: true,
				Hunks: []Hunk{{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 2}}},
			{Path: "docs/notes.md", Content: "# notes\n"},
		},
		ProducerRun: "run-1",
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

// AC1:必填项 = project、base revision、candidate revision/patch 身份(此处由文件推导)。
// 全部满足时构造成功并固定各字段。
func TestNewFixesRequiredFields(t *testing.T) {
	cs, err := New(validSpec())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if cs.Project() != core.ProjectID("proj-1") {
		t.Errorf("Project = %q", cs.Project())
	}
	if cs.BaseRevision() != validSpec().BaseRevision {
		t.Errorf("BaseRevision = %q", cs.BaseRevision())
	}
	if cs.PatchIdentity() == "" {
		t.Error("PatchIdentity 为空:有改动文件时必须推导出 patch 身份")
	}
	if cs.ProducerRun() != "run-1" {
		t.Errorf("ProducerRun = %q", cs.ProducerRun())
	}
	if cs.CreatedAt() != time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) {
		t.Errorf("CreatedAt = %v", cs.CreatedAt())
	}
	if id := cs.ID(); !strings.HasPrefix(string(id), "sha256:") {
		t.Errorf("ID = %q,应为 sha256 前缀", id)
	}
	// 文件按路径排序归一。
	files := cs.Files()
	if len(files) != 2 || files[0].Path != "cmd/server/main.go" || files[1].Path != "docs/notes.md" {
		t.Errorf("Files 未按路径排序或内容不对: %+v", files)
	}
}

// AC1:只给 candidate revision(无改动文件)也是合法的候选身份。
func TestNewCandidateRevisionAloneSuffices(t *testing.T) {
	spec := validSpec()
	spec.Files = nil
	spec.CandidateRevision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	cs, err := New(spec)
	if err != nil {
		t.Fatalf("New(仅 candidate revision): %v", err)
	}
	if cs.CandidateRevision() == "" || cs.PatchIdentity() != "" {
		t.Errorf("CandidateRevision = %q, PatchIdentity = %q", cs.CandidateRevision(), cs.PatchIdentity())
	}
}

// AC1:缺任一必填项都必须报对应哨兵错误。
func TestNewRejectsMissingRequired(t *testing.T) {
	cases := []struct {
		name string
		mut  func(s *Spec)
		want error
	}{
		{"缺 project", func(s *Spec) { s.Project = "" }, ErrNoProject},
		{"缺 base revision", func(s *Spec) { s.BaseRevision = "  " }, ErrNoBaseRevision},
		{"缺 producer run", func(s *Spec) { s.ProducerRun = "" }, ErrNoProducerRun},
		// candidate revision 与改动文件都为空 => 没有任何候选身份。
		{"缺候选身份", func(s *Spec) { s.Files = nil }, ErrNoCandidateIdentity},
		{"文件条目空路径", func(s *Spec) { s.Files = []ChangedFile{{Path: "  ", Content: "x"}} }, ErrInvalidFile},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := validSpec()
			tc.mut(&spec)
			if _, err := New(spec); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// AC2:构造之后 API 不可变 —— 所有取值返回副本/值,改返回值改不动快照本体;
// 构造后改 Spec 输入也影响不了已固定的快照。
func TestImmutableByAPI(t *testing.T) {
	spec := validSpec()
	cs, err := New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 改返回的 Files 切片与内层 Hunk。
	files := cs.Files()
	files[0].Content = "package hacked\n"
	files[0].Hunks[0].NewLines = 999
	files = append(files, ChangedFile{Path: "zzz-injected.go"})
	if got := cs.Files(); got[0].Content != "package main\n" ||
		got[0].Hunks[0].NewLines != 2 ||
		len(got) != 2 {
		t.Fatalf("快照本体被通过返回值改动: %+v", got)
	}

	// 改返回的 Entities 切片。
	spec2 := validSpec()
	spec2.Entities = []string{"server"}
	cs2, err := New(spec2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	entities := cs2.Entities()
	entities[0] = "injected"
	entities = append(entities, "extra")
	if got := cs2.Entities(); len(got) != 1 || got[0] != "server" {
		t.Fatalf("Entities 本体被改动: %v", got)
	}

	// 构造后再改输入 Spec(含内层 Hunk)不影响快照。
	spec.Files[0].Content = "package mutated\n"
	spec.Files[0].Hunks[0].NewLines = 12345
	spec.Files = append(spec.Files, ChangedFile{Path: "aaa-late.go"})
	if got := cs.Files(); got[0].Content != "package main\n" ||
		got[0].Hunks[0].NewLines != 2 ||
		len(got) != 2 {
		t.Fatalf("快照被通过输入 Spec 改动: %+v", got)
	}
}

// AC4:取代不是 ChangeSet 上的可变状态,而是独立的 append-only 事实值。
func TestSupersessionIsSeparateAppendOnlyFact(t *testing.T) {
	cs, err := New(validSpec())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	replacement, err := New(func() Spec {
		s := validSpec()
		s.Files = []ChangedFile{{Path: "cmd/server/main.go", Content: "package main // v2\n", Executable: true}}
		return s
	}())
	if err != nil {
		t.Fatalf("New(替代快照): %v", err)
	}

	at := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	fact, err := NewSupersededFact(cs.ID(), replacement.ID(), SupersedeEscalation, at)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if fact.Old != cs.ID() || fact.New != replacement.ID() || fact.Reason != SupersedeEscalation || fact.At != at {
		t.Errorf("事实字段不符: %+v", fact)
	}
	// 原快照各取值不变:取代没有落在快照本体上。
	if cs.PatchIdentity() == replacement.PatchIdentity() {
		t.Error("新旧快照 patch 身份相同,取代事实失去意义")
	}

	fact2, err := NewSupersededFact(cs.ID(), replacement.ID(), SupersedeRework, at)
	if err != nil || fact2.Reason != SupersedeRework {
		t.Fatalf("同类事实应可继续追加(append-only),err=%v fact=%+v", err, fact2)
	}
}

func TestSupersededFactRejectsInvalid(t *testing.T) {
	a := ChangeSetID("sha256:a")
	b := ChangeSetID("sha256:b")
	at := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	cases := []struct {
		name   string
		oldNew [2]ChangeSetID
		reason SupersedeReason
	}{
		{"old 为空", [2]ChangeSetID{"", b}, SupersedeRework},
		{"new 为空", [2]ChangeSetID{a, ""}, SupersedeRework},
		{"old == new", [2]ChangeSetID{a, a}, SupersedeRework},
		{"未知原因", [2]ChangeSetID{a, b}, SupersedeReason("vibes")},
		{"原因为空", [2]ChangeSetID{a, b}, SupersedeReason("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewSupersededFact(tc.oldNew[0], tc.oldNew[1], tc.reason, at); !errors.Is(err, ErrBadSupersede) {
				t.Fatalf("err = %v, want ErrBadSupersede", err)
			}
		})
	}
}

func TestSupersededFactDefaultsTimestamp(t *testing.T) {
	a := ChangeSetID("sha256:a")
	b := ChangeSetID("sha256:b")
	fact, err := NewSupersededFact(a, b, SupersedeSpecDefect, time.Time{})
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if fact.At.IsZero() {
		t.Error("零时刻应被补为当下")
	}
	if _, offset := fact.At.Zone(); offset != 0 {
		t.Errorf("At 应为 UTC,offset = %d", offset)
	}
}

func TestCreatedAtDefaultsToNowUTC(t *testing.T) {
	spec := validSpec()
	spec.CreatedAt = time.Time{}
	cs, err := New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if cs.CreatedAt().IsZero() {
		t.Error("CreatedAt 为零值,应默认取当下")
	}
	if _, offset := cs.CreatedAt().Zone(); offset != 0 {
		t.Errorf("CreatedAt 应为 UTC,offset = %d", offset)
	}
}
