//go:build integration

package deps

// 集成测试:真 Postgres(不 mock 数据库,docs/05-test-standards.md)+ 真知识仓库
// (go-git 在临时目录 init)+ fake 执行器(执行抽象离线可控),走退休检查全链。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/knowledge"
	"tally/internal/store"
)

// retireProof 实现 core.VerificationProof,供测试把一条修复存进知识库。
// 真证据只有 failure 四道闸产得出;这里只用于铺集成测试的底。
type retireProof struct {
	at    time.Time
	combo core.VersionCombo
}

func (p retireProof) VerifiedAt() time.Time            { return p.at }
func (p retireProof) VerifiedCombo() core.VersionCombo { return p.combo }
func (p retireProof) GateRecord() string               { return "g1=5/5red g2=5/5green g3=5/5green g4=scope-ok" }
func (p retireProof) Valid() bool                      { return !p.at.IsZero() }

// seedVerifiedFix 在真知识仓库存一条已验证修复,并把索引面置成 fixed(模拟修复入库)。
func seedVerifiedFix(t *testing.T, st *store.Store, kb *knowledge.Store, project string, fp core.Fingerprint, combo core.VersionCombo) {
	t.Helper()
	ctx := context.Background()
	fix := core.Fix{Repro: "make build", Summary: "集成测试修复", Changes: nil}
	fixRef, err := kb.Store(ctx, fp, fix, retireProof{at: time.Now().UTC(), combo: combo})
	if err != nil {
		t.Fatalf("存已验证修复: %v", err)
	}
	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	if _, err := st.Queries().UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: string(fp), FpFuzzy: string(fp) + "f", FirstSeen: now, Project: project,
	}); err != nil {
		t.Fatalf("建索引行: %v", err)
	}
	comboJSON, err := json.Marshal(combo)
	if err != nil {
		t.Fatalf("序列化验证组合: %v", err)
	}
	if err := st.Queries().UpdateFingerprintStatFix(ctx, store.UpdateFingerprintStatFixParams{
		FpExact: string(fp), FixRef: pgtype.Text{String: fixRef, Valid: true},
		FixVerifiedAt: now, FixVerifiedCombo: comboJSON,
	}); err != nil {
		t.Fatalf("置索引为 fixed: %v", err)
	}
}

// seedFailureEvent 记一条带复现上下文的失败事件(来源外键配一行 environment_run)。
// withCtx=false 时留空 workdir/repro,模拟旧失败没带上下文。
func seedFailureEvent(t *testing.T, st *store.Store, project string, fp core.Fingerprint, withCtx bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	envID, err := st.Queries().InsertEnvironmentRun(ctx, store.InsertEnvironmentRunParams{
		ProjectID: project, EnvName: "dev", DefinitionHash: "sha256:def",
		StartedAt: pgtype.Timestamptz{Time: now, Valid: true}, Status: "failed",
		VersionCombo: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("记 environment_run: %v", err)
	}
	reproCommand := []byte("null")
	if withCtx {
		reproCommand = []byte(`{"argv":["make","build"],"cwd":"/tmp/demo","env":{},"shell":false}`)
	}
	if _, err := st.Queries().InsertFailureEvent(ctx, store.InsertFailureEventParams{
		ProjectID: project, Source: "environment",
		EnvironmentRunID: pgtype.Int8{Int64: envID, Valid: true},
		FpExact:          string(fp), FpFuzzy: string(fp) + "f",
		PipelineVersion: "1", OccurredAt: pgtype.Timestamptz{Time: now, Valid: true},
		VersionCombo: []byte(`{}`), OtherTargetCommands: []byte("[]"),
		ReproCommand: reproCommand,
	}); err != nil {
		t.Fatalf("记 failure_event: %v", err)
	}
}

// candidatesFromKB 从知识库列已验证修复并转成退休判定视图(与 cmd 同路数)。
func candidatesFromKB(t *testing.T, kb *knowledge.Store) []FixCandidate {
	t.Helper()
	entries, err := kb.ListVerified(context.Background())
	if err != nil {
		t.Fatalf("列已验证修复: %v", err)
	}
	out := make([]FixCandidate, 0, len(entries))
	for _, e := range entries {
		out = append(out, FixCandidate{
			Fingerprint:   core.Fingerprint(e.Fingerprint),
			VerifiedCombo: e.VerifiedCombo,
			Retired:       e.Retired,
		})
	}
	return out
}

// TestRetireIntegrationNoLongerReproduces:升级碰过验证组合 + 复现不再复现(有绿)
// → 退休:RETIRED.md 落盘、索引面回 unresolved 且 fix_ref 空、LookupExact 降级 stale。
func TestRetireIntegrationNoLongerReproduces(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	repoDir, _ := initRepo(t)
	kb := knowledge.NewStore(repoDir, st.Queries())
	project := "deps-retire-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())
	fp := core.Fingerprint("e1a9d2b834590c11")
	combo := core.VersionCombo{pkgPGX: "v5.5.1", "go": "1.23.0"}

	seedVerifiedFix(t, st, kb, project, fp, combo)
	seedFailureEvent(t, st, project, fp, true)

	rep, err := RetirePass(ctx, RetireOptions{
		ProjectID: project,
		Changed:   []string{pkgPGX},
		Fixes:     candidatesFromKB(t, kb),
		Runner:    &fakeReproRunner{green: true}, // 不再复现
		Retirer:   kb,
		Store:     NewRetireStore(st),
	})
	if err != nil {
		t.Fatalf("RetirePass: %v", err)
	}
	if !contains(rep.Retired, fp) {
		t.Fatalf("应退休 %s,得到 %+v", fp, rep)
	}

	// RETIRED.md 落盘(与 STALE.md 同构,记录保留)。
	if _, err := readFile(repoDir, "knowledge/"+string(fp)+"/RETIRED.md"); err != nil {
		t.Errorf("退休标记 RETIRED.md 未落盘: %v", err)
	}

	// 索引面被清:status 回 unresolved、fix_ref 空、诊断次数归零。
	stat, err := st.Queries().GetFingerprintStat(ctx, string(fp))
	if err != nil {
		t.Fatalf("读索引行: %v", err)
	}
	if stat.Status != "unresolved" {
		t.Errorf("退休后 status = %q,想要 unresolved", stat.Status)
	}
	if stat.FixRef.Valid {
		t.Errorf("退休后 fix_ref 应清空,得到 %q", stat.FixRef.String)
	}
	if stat.DiagnoseAttempts != 0 {
		t.Errorf("退休后诊断次数应归零,得到 %d", stat.DiagnoseAttempts)
	}

	// LookupExact 命中已退休条目 → 按 stale 处理(降级线索)。
	hit, ok, err := kb.LookupExact(ctx, fp, combo)
	if err != nil || !ok {
		t.Fatalf("LookupExact: ok=%v err=%v", ok, err)
	}
	if !hit.Stale {
		t.Errorf("已退休条目 LookupExact 应 Stale=true")
	}
}

// TestRetireIntegrationStillReproduces:仍稳定复现(全红)→ 不退休,索引面保持 fixed。
func TestRetireIntegrationStillReproduces(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	repoDir, _ := initRepo(t)
	kb := knowledge.NewStore(repoDir, st.Queries())
	project := "deps-retire-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())
	fp := core.Fingerprint("b834590c11e1a9d2")
	combo := core.VersionCombo{pkgPGX: "v5.5.1"}

	seedVerifiedFix(t, st, kb, project, fp, combo)
	seedFailureEvent(t, st, project, fp, true)

	rep, err := RetirePass(ctx, RetireOptions{
		ProjectID: project,
		Changed:   []string{pkgPGX},
		Fixes:     candidatesFromKB(t, kb),
		Runner:    &fakeReproRunner{green: false}, // 仍复现
		Retirer:   kb,
		Store:     NewRetireStore(st),
	})
	if err != nil {
		t.Fatalf("RetirePass: %v", err)
	}
	if !contains(rep.StillRepro, fp) {
		t.Fatalf("仍复现应不动 %s,得到 %+v", fp, rep)
	}
	if contains(rep.Retired, fp) {
		t.Errorf("仍复现不应退休")
	}

	stat, err := st.Queries().GetFingerprintStat(ctx, string(fp))
	if err != nil {
		t.Fatalf("读索引行: %v", err)
	}
	if stat.Status != "fixed" {
		t.Errorf("仍复现应保持 fixed,得到 %q", stat.Status)
	}
}

// TestRetireIntegrationMissingContextSkips:旧失败没带 workdir/repro → 跳过不误退。
func TestRetireIntegrationMissingContextSkips(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	repoDir, _ := initRepo(t)
	kb := knowledge.NewStore(repoDir, st.Queries())
	project := "deps-retire-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())
	fp := core.Fingerprint("590c11e1a9d2b834")
	combo := core.VersionCombo{pkgPGX: "v5.5.1"}

	seedVerifiedFix(t, st, kb, project, fp, combo)
	seedFailureEvent(t, st, project, fp, false) // 不带上下文

	rep, err := RetirePass(ctx, RetireOptions{
		ProjectID: project,
		Changed:   []string{pkgPGX},
		Fixes:     candidatesFromKB(t, kb),
		Runner:    &fakeReproRunner{green: true},
		Retirer:   kb,
		Store:     NewRetireStore(st),
	})
	if err != nil {
		t.Fatalf("RetirePass: %v", err)
	}
	if !contains(rep.Skipped, fp) {
		t.Fatalf("缺上下文应跳过 %s,得到 %+v", fp, rep)
	}
	if contains(rep.Retired, fp) {
		t.Errorf("缺上下文不应退休(不误退)")
	}
	if _, err := readFile(repoDir, "knowledge/"+string(fp)+"/RETIRED.md"); err == nil {
		t.Errorf("缺上下文不应写 RETIRED.md")
	}
}

// readFile 读知识仓库里某文件(相对仓库根)。
func readFile(repoDir, rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(rel)))
}
