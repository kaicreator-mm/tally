//go:build integration

package project

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

// TestClauseImplViewA:视图 A 全字段 —— 实现提交数、构建失败数、部署失败数、同一指纹次数、
// 条款年龄与变更次数、失败构成按指纹标注处置分类。
func TestClauseImplViewA(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)
	h1 := "sha256:" + run + "-v1"
	h2 := "sha256:" + run + "-v2"
	c1, c2, c3 := "c1"+run, "c2"+run, "c3"+run
	art := "sha256:art-" + run
	fpX, fpY := "fpx-"+run, "fpy-"+run

	// 条款两个版本:变更次数=2,首签于 base。
	seedClauseVersion(t, q, string(project), "REQ-12", h1, 1, base, true)
	seedClauseVersion(t, q, string(project), "REQ-12", h2, 2, base.Add(24*time.Hour), true)
	// 三个实现提交(挂到本条款的不同版本)。
	for _, cc := range []store.InsertCommitClauseParams{
		{ProjectID: string(project), CommitSha: c1, ClauseContentHash: h1},
		{ProjectID: string(project), CommitSha: c2, ClauseContentHash: h2},
		{ProjectID: string(project), CommitSha: c3, ClauseContentHash: h1},
	} {
		if err := q.InsertCommitClause(ctx, cc); err != nil {
			t.Fatalf("写 commit_clause: %v", err)
		}
	}

	// 构建:c1/c3 成功产制品,c2 失败无制品。
	_ = seedBuild(t, q, string(project), c1, art, base)
	bFail := seedBuild(t, q, string(project), c2, "", base)
	_ = seedBuild(t, q, string(project), c3, art, base)
	// 部署:用了制品 art。
	d1 := seedDeploy(t, q, string(project), art, 0, base)
	// 失败:c2 的构建失败两次(同一指纹 fpX),部署失败一次(fpY)。
	seedBuildFailure(t, q, string(project), bFail, fpX, base)
	seedBuildFailure(t, q, string(project), bFail, fpX, base.Add(time.Minute))
	seedDeployFailure(t, q, string(project), d1, fpY, base)

	// 给 fpX 一个处置分类(视图 A 失败构成要标注),fpY 不归类(应落 unresolved)。
	if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: fpX, FpFuzzy: fpX + "f", FirstSeen: pgtype.Timestamptz{Time: base, Valid: true}, Project: string(project),
	}); err != nil {
		t.Fatalf("upsert fpX: %v", err)
	}
	if err := q.UpdateFingerprintStatClassification(ctx, store.UpdateFingerprintStatClassificationParams{
		FpExact: fpX, Status: "flaky",
	}); err != nil {
		t.Fatalf("归类 fpX: %v", err)
	}

	rep, err := ClauseImpl(ctx, q, project, "REQ-12")
	if err != nil {
		t.Fatalf("ClauseImpl: %v", err)
	}
	if rep.ImplCommits != 3 {
		t.Errorf("实现提交数应 3, got %d", rep.ImplCommits)
	}
	if rep.BuildFailures != 2 {
		t.Errorf("构建失败数应 2, got %d", rep.BuildFailures)
	}
	if rep.DeployFailures != 1 {
		t.Errorf("部署失败数应 1, got %d", rep.DeployFailures)
	}
	if rep.SameFingerprint != 2 {
		t.Errorf("同一指纹最高复发应 2(fpX), got %d", rep.SameFingerprint)
	}
	if rep.ChangeCount != 2 {
		t.Errorf("变更次数应 2, got %d", rep.ChangeCount)
	}
	if rep.AgeDays < 0 {
		t.Errorf("条款年龄应 >=0, got %d", rep.AgeDays)
	}
	if rep.FirstSignedAt.IsZero() {
		t.Error("首签时刻不该为零")
	}

	// 失败构成:fpX×2(flaky), fpY×1(unresolved)。
	byFP := map[string]FailureComponent{}
	for _, f := range rep.Failures {
		byFP[string(f.Fingerprint)] = f
	}
	if x := byFP[fpX]; x.Count != 2 || x.Classification != "flaky" {
		t.Errorf("fpX 应 2 次/flaky, got %+v", x)
	}
	if y := byFP[fpY]; y.Count != 1 || y.Classification != "unresolved" {
		t.Errorf("fpY 应 1 次/unresolved, got %+v", y)
	}
}

// TestClauseImplNotFound:条款不存在 → ErrClauseNotFound。
func TestClauseImplNotFound(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	run := runID(t)
	_, err := ClauseImpl(ctx, st.Queries(), core.ProjectID("proj-"+run), "REQ-NOPE")
	if err != ErrClauseNotFound {
		t.Errorf("应 ErrClauseNotFound, got %v", err)
	}
}

// TestFingerprintImpactViewB:视图 B 全字段 —— 出现次数、影响项目数、关联条款、已验证修复及验证组合、是否漂移。
func TestFingerprintImpactViewB(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	fp := "fpz-" + run
	hZ := "sha256:" + run + "-hz"
	cZ := "cz" + run

	// 条款 + 挂钩,让指纹能关联到 REQ-9。
	seedClauseVersion(t, q, project, "REQ-9", hZ, 1, base, true)
	_ = q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: cZ, ClauseContentHash: hZ})
	b := seedBuild(t, q, project, cZ, "", base)
	seedBuildFailure(t, q, project, b, fp, base)

	// 指纹索引:出现 3 次、横跨 2 项目、已修复、验证后复发(漂移)。
	for _, p := range []string{"p1", "p2", "p1"} {
		if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
			FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: base.Add(10 * time.Hour), Valid: true}, Project: p,
		}); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	if err := q.UpdateFingerprintStatFix(ctx, store.UpdateFingerprintStatFixParams{
		FpExact:          fp,
		FixRef:           pgtype.Text{String: "deadbeef", Valid: true},
		FixVerifiedAt:    pgtype.Timestamptz{Time: base.Add(5 * time.Hour), Valid: true}, // 早于 last_seen(base+10h)→ 漂移
		FixVerifiedCombo: combo,
	}); err != nil {
		t.Fatalf("写修复索引: %v", err)
	}

	impact, err := FingerprintImpactView(ctx, q, core.Fingerprint(fp))
	if err != nil {
		t.Fatalf("FingerprintImpactView: %v", err)
	}
	if impact.Occurrences != 3 {
		t.Errorf("出现次数应 3, got %d", impact.Occurrences)
	}
	if impact.AffectedProjectCount != 2 {
		t.Errorf("影响项目数应 2, got %d", impact.AffectedProjectCount)
	}
	if impact.Status != "fixed" || !impact.HasFix || impact.FixRef != "deadbeef" {
		t.Errorf("应 fixed/deadbeef, got status=%s hasFix=%v ref=%s", impact.Status, impact.HasFix, impact.FixRef)
	}
	if impact.VerifiedCombo["go"] != "1.23.0" {
		t.Errorf("验证组合应回读 go=1.23.0, got %v", impact.VerifiedCombo)
	}
	if !impact.Drifted {
		t.Error("验证后复发应判漂移")
	}
	if len(impact.RelatedClauses) != 1 || impact.RelatedClauses[0].ClauseID != "REQ-9" {
		t.Errorf("应关联 REQ-9, got %+v", impact.RelatedClauses)
	}
}

// TestFingerprintImpactNoRequirementLayer:视图 B 的 LEFT JOIN —— 无需求层(推不出条款)时
// 关联条款为空,但行仍在、其余字段照常。
func TestFingerprintImpactNoRequirementLayer(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	fp := "fpw-" + run
	cW := "cw" + run

	// 有失败链到提交 cW,但 cW 没有 commit_clause(没有需求层)。
	b := seedBuild(t, q, project, cW, "", base)
	seedBuildFailure(t, q, project, b, fp, base)
	if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: base, Valid: true}, Project: project,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	impact, err := FingerprintImpactView(ctx, q, core.Fingerprint(fp))
	if err != nil {
		t.Fatalf("无需求层也该正常返回: %v", err)
	}
	if len(impact.RelatedClauses) != 0 {
		t.Errorf("无需求层时关联条款应为空, got %+v", impact.RelatedClauses)
	}
	if impact.Occurrences != 1 || impact.AffectedProjectCount != 1 {
		t.Errorf("其余字段照常:出现 1/影响 1, got %d/%d", impact.Occurrences, impact.AffectedProjectCount)
	}
	if impact.Drifted {
		t.Error("未修复不该判漂移")
	}
}

// TestFingerprintImpactNotFound:指纹不存在 → ErrFingerprintNotFound。
func TestFingerprintImpactNotFound(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	run := runID(t)
	_, err := FingerprintImpactView(ctx, st.Queries(), core.Fingerprint("nope-"+run))
	if err != ErrFingerprintNotFound {
		t.Errorf("应 ErrFingerprintNotFound, got %v", err)
	}
}
