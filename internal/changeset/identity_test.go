package changeset

import (
	"testing"
	"time"
)

func identitySpec() Spec {
	return Spec{
		Project:      "proj-1",
		BaseRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Files: []ChangedFile{
			{Path: "internal/changeset/types.go", Content: "package changeset\n", Executable: true},
			{Path: "docs/notes.md", Content: "# notes\n"},
		},
		ProducerRun: "run-1",
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func mustNew(t *testing.T, spec Spec) *ChangeSet {
	t.Helper()
	cs, err := New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return cs
}

// AC3:可执行代码一变,patch 身份与快照 ID 必变。
func TestIdentityChangesWhenExecutableCodeChanges(t *testing.T) {
	base := mustNew(t, identitySpec())

	mutated := identitySpec()
	mutated.Files[0].Content = "package changeset // 一行可执行代码改动\n"
	after := mustNew(t, mutated)

	if base.PatchIdentity() == after.PatchIdentity() {
		t.Error("可执行代码变了但 patch 身份未变")
	}
	if base.ID() == after.ID() {
		t.Error("可执行代码变了但 ChangeSet ID 未变")
	}
}

// 内容寻址:仅非可执行文件(文档)内容变化同样改变身份 ——
// 身份由全部改动内容决定,不存在"改了内容但身份不变"的路径。
func TestIdentityChangesWhenAnyContentChanges(t *testing.T) {
	base := mustNew(t, identitySpec())

	docs := identitySpec()
	docs.Files[1].Content = "# notes v2\n"
	after := mustNew(t, docs)

	if base.PatchIdentity() == after.PatchIdentity() {
		t.Error("文档内容变了但 patch 身份未变")
	}
}

// 可执行性标记属于身份的一部分:标记翻转即身份变化。
func TestIdentityChangesWhenExecutableFlagChanges(t *testing.T) {
	base := mustNew(t, identitySpec())

	flag := identitySpec()
	flag.Files[1].Executable = true
	after := mustNew(t, flag)

	if base.PatchIdentity() == after.PatchIdentity() {
		t.Error("可执行标记变了但 patch 身份未变")
	}
}

// 身份确定且与文件顺序无关:同一候选内容(乱序给出)得到同一 patch 身份与同一 ID。
func TestIdentityDeterministicAndOrderIndependent(t *testing.T) {
	a := mustNew(t, identitySpec())

	shuffled := identitySpec()
	shuffled.Files[0], shuffled.Files[1] = shuffled.Files[1], shuffled.Files[0]
	b := mustNew(t, shuffled)

	if a.PatchIdentity() != b.PatchIdentity() || a.ID() != b.ID() {
		t.Errorf("同一候选内容得到不同身份:ID %q vs %q", a.ID(), b.ID())
	}
}

// candidate revision 是身份输入:revision 变则 ID 变(patch 身份保持为空)。
func TestIdentityChangesWithCandidateRevision(t *testing.T) {
	s1 := identitySpec()
	s1.Files = nil
	s1.CandidateRevision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	s2 := s1
	s2.CandidateRevision = "cccccccccccccccccccccccccccccccccccccccc"

	a, b := mustNew(t, s1), mustNew(t, s2)
	if a.PatchIdentity() != "" || b.PatchIdentity() != "" {
		t.Errorf("无改动文件时 patch 身份应为空,%q / %q", a.PatchIdentity(), b.PatchIdentity())
	}
	if a.ID() == b.ID() {
		t.Error("candidate revision 变了但 ID 未变")
	}
}

// producer run 是身份输入:不同 Run 的快照互不相同,即使候选内容一致。
func TestIdentityChangesWithProducerRun(t *testing.T) {
	a := mustNew(t, identitySpec())
	spec := identitySpec()
	spec.ProducerRun = "run-2"
	b := mustNew(t, spec)

	if a.ID() == b.ID() {
		t.Error("不同 producer run 得到相同 ID,无法区分各 Run 的候选快照")
	}
	if a.PatchIdentity() != b.PatchIdentity() {
		t.Error("候选内容一致时 patch 身份应相同")
	}
}

// project/base 也是身份输入。
func TestIdentityChangesWithProjectAndBase(t *testing.T) {
	base := mustNew(t, identitySpec())

	otherProj := identitySpec()
	otherProj.Project = "proj-2"
	if base.ID() == mustNew(t, otherProj).ID() {
		t.Error("project 变了但 ID 未变")
	}

	otherBase := identitySpec()
	otherBase.BaseRevision = "dddddddddddddddddddddddddddddddddddddddd"
	if base.ID() == mustNew(t, otherBase).ID() {
		t.Error("base revision 变了但 ID 未变")
	}
}

// hunk 落点是身份输入;nil 与空 hunk 切片等价,不产生伪差异。
func TestHunksAffectIdentityAndNilEqualsEmpty(t *testing.T) {
	withHunk := identitySpec()
	withHunk.Files[0].Hunks = []Hunk{{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 2}}
	a := mustNew(t, withHunk)

	noHunk := identitySpec()
	b := mustNew(t, noHunk)
	if a.PatchIdentity() == b.PatchIdentity() {
		t.Error("hunk 落点变了但 patch 身份未变")
	}

	emptyHunk := identitySpec()
	emptyHunk.Files[0].Hunks = []Hunk{}
	c := mustNew(t, emptyHunk)
	if b.PatchIdentity() != c.PatchIdentity() {
		t.Error("nil 与空 hunk 切片被当成不同内容")
	}
}
