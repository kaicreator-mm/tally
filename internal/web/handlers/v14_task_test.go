package handlers_test

// P3-08 的闭环任务详情页测试(PRD §26.2)。
//
// 两层:
//   - 纯 HTTP 层(不需要数据库):数据源未接入空状态;
//   - 全链路层(临时 Git 制品仓库 + 独立 schema 的真实 Postgres,不可达即
//     SKIP):尝试链(修复尝试 = EscalatedFrom 父链接的新 Run,ADR-0001 §2)、
//     候选与验证计数、取代关系、INV-11 依据呈现、claim 标注与缺口可见性。
//
// 全部断言只针对派生/事实投影:页面没有任何可写状态。

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/measurement"
	"tally/internal/runstore"
)

func TestV14TaskUnavailableWithoutSources(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-0001")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/tasks/TASK-0001 = %d, want 200", status)
	}
	if !strings.Contains(body, "未接入") {
		t.Errorf("无数据源时应渲染未接入空状态")
	}
}

func TestV14TaskDetailShowsRepairChainBasisAndUncertainty(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	frozenHash := v13FrozenHash(t, repo, base)
	rs := runstore.New(st.Queries())

	when := time.Now().Add(-time.Hour)
	// 尝试 1(根):docs 候选,完成并固定候选。
	run1 := v14StartRun(t, ctx, rs, base, frozenHash, "ws-a1", 0, when)
	cs1 := v14FixCandidate(t, ctx, rs, base, docs, run1)
	// 尝试 2(修复):EscalatedFrom 指向尝试 1(ADR-0001 §2 父链接),code 候选。
	run2 := v14StartRun(t, ctx, rs, base, frozenHash, "ws-a2", run1, when.Add(10*time.Minute))
	cs2 := v14FixCandidate(t, ctx, rs, base, code, run2)

	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: run1, Status: runstore.RunCompleted, Iterations: 3,
		ResultPatchHash: string(cs1), AgentClaim: "第一版已经修好", FinishedAt: when.Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(run1): %v", err)
	}
	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: run2, Status: runstore.RunCompleted, Iterations: 2,
		ResultPatchHash: string(cs2), AgentClaim: "已修复会话残留", FinishedAt: when.Add(20 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(run2): %v", err)
	}
	// 取代事实:修复候选取代第一候选(append-only,不合并)。
	fact, err := changeset.NewSupersededFact(cs1, cs2, changeset.SupersedeRework, when.Add(21*time.Minute))
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := rs.AppendSupersededFact(ctx, fact); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}
	// 验证:尝试 1 的候选 FAIL,尝试 2 的候选 PASS。
	v14Verify(t, ctx, rs, frozenHash, cs1, runstore.VerificationFail, when.Add(6*time.Minute))
	v14Verify(t, ctx, rs, frozenHash, cs2, runstore.VerificationPass, when.Add(25*time.Minute))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-0001")
	if status != http.StatusOK {
		t.Fatalf("GET 任务详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"闭环任务 TASK-0001",
		"演示任务",
		"冻结验证契约",
		"首次",                       // 尝试 1
		"修复 #2",                    // 尝试 2:父链接推导的修复序号(AC4)
		"claim:已修复会话残留",            // Agent 自述永远标注为 claim(INV-AUTH)
		"当前候选",                     // 派生指针(AC4)
		"fail 1",                   // 尝试 1 候选的失败验证可见
		"pass 1",                   // 尝试 2 候选的通过验证可见
		"取代",                       // 取代关系 append-only 呈现
		"依据(Basis",                 // INV-11:结论必须带依据
		"fact:v13art:task_planned", // 依据引用落到事实 id
		"门槛缺口",                     // 不确定性可见(未达 VERIFIED 的原因)
		"没有暴露事实",
		"主线尚未达 VERIFIED",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("任务详情应包含 %q", want)
		}
	}
	if !strings.Contains(body, v13ShortCS(string(cs2))) {
		t.Errorf("当前候选应为第二次尝试的候选 %s", v13ShortCS(string(cs2)))
	}
	if strings.Contains(body, "需要人工关注") {
		t.Errorf("两尝试完成且无异常事实时不得派生异常态")
	}
}

func TestV14TaskNotFound(t *testing.T) {
	st := v13TestDB(t) // 需要 Postgres 才有事实面;不可达即 SKIP。
	repo, _, _, _ := v13Repo(t)
	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-9999")
	if status != http.StatusNotFound {
		t.Fatalf("未知任务应 404,得到 %d", status)
	}
	if !strings.Contains(body, "HEAD 快照里没有 TaskContract") {
		t.Errorf("应说明 Git 是唯一来源,body:\n%s", body)
	}
}

/* ---------- v1.4.1 P2-02:人类心智模型分节 + 验证结论依据面(PRD §7/§14/§29)---------- */

// v14TaskSections 是任务详情页的六个人类心智模型分节(PRD §7/§29:Overview /
// Plan / Changes / Verification / Reviews / Activity,全部是既有事实的
// projection,不改状态机,DL-19)。
var v14TaskSections = []string{
	"<h2>概览 Overview</h2>",
	"<h2>计划 Plan</h2>",
	"<h2>变更 Changes</h2>",
	"<h2>验证 Verification</h2>",
	"<h2>评审 Reviews</h2>",
	"<h2>动态 Activity</h2>",
}

// TestV14TaskCompiledBasisPlainVerified 全绿候选(零策略 + 零确认风险信号):
// 任务详情页在同一 pinned HEAD 提交消费编译权威(P1-03 生产 seam)与当前候选
// 覆盖(Change Detail 同一读路径),验证结论必须是普通 VERIFIED,依据携带
// compiled:<hash>,四维度给出事实值;同时钉住六个分节与「工作」高亮。
func TestV14TaskCompiledBasisPlainVerified(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t) // 需要 Postgres 才有事实面;不可达即 SKIP。
	repo, base, _, code := v14cIntentRepo(t, nil)
	rs := runstore.New(st.Queries())
	v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p202-plain", "fact-p202-1", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-0001")
	if status != http.StatusOK {
		t.Fatalf("GET 任务详情 = %d, want 200", status)
	}
	for _, want := range wantStrings(v14TaskSections, []string{
		"验证结论(VerifiedBasis,PRD §14)",              // 结论行渲染(P1-07 同一投影,不在本页重复实现)
		"验证门全部满足",                                  // 普通 VERIFIED 的说明分支(limited 分支的措辞不同)
		"compiled:sha256:",                         // 结论可追溯到已编译合同身份(INV-STATE-02;P1-07 遗留缺口闭合)
		"必需 L1",                                    // assurance 维度事实值:零信号默认地板
		"达成 L1",                                    // visible deterministic 证据的达成档
		"无未匹配风险信号",                                 // unmatched 维度事实值(命中策略 0)
		"针对冻结契约",                                   // 当前候选覆盖表(coverage 维度的逐条款事实)
		"fact:v13art:task_planned",                 // 依据列表照旧携带事实引用(INV-11)
		`href="/v13/tasks" aria-current="page">工作`, // 任务详情是 Work 域视图,高亮「工作」
	}) {
		if !strings.Contains(body, want) {
			t.Errorf("全绿任务详情应包含 %q", want)
		}
	}
	for _, banned := range []string{
		"VERIFIED_LIMITED", // 不得把普通 VERIFIED 降格展示
		"已编译合同未接入",         // 编译权威已接入时不得渲染未接入标注
	} {
		if strings.Contains(body, banned) {
			t.Errorf("全绿任务详情不得渲染 %q", banned)
		}
	}
}

// TestV14TaskCompiledBasisVerifiedLimitedOnUnmatchedRisk 确认风险提示
// auth_symbol_change 无任何命中策略声明:状态仍是 VERIFIED,但任务详情的验证
// 结论必须是 VERIFIED_LIMITED(不得显示成普通 VERIFIED),未匹配信号原样可见,
// 必需档被 fail-safe 上调 L3(PRD §14;INV-V14-09)。
func TestV14TaskCompiledBasisVerifiedLimitedOnUnmatchedRisk(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, _, code := v14cIntentRepo(t, nil, "auth_symbol_change")
	rs := runstore.New(st.Queries())
	v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p202-limited", "fact-p202-2", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-0001")
	if status != http.StatusOK {
		t.Fatalf("GET 任务详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"VERIFIED_LIMITED",           // limited 结论,不是普通 VERIFIED
		"不得显示成普通 VERIFIED",           // limited 结论的区分性说明(INV-07/PRD §14)
		"未匹配风险信号:auth_symbol_change", // 信号原样可见(verbatim)
		"必需 L3",            // fail-safe 上调的必需档
		"达成 L1",            // 最弱条款达成档诚实上报
		"compiled:sha256:", // 结论依据已编译合同身份
	} {
		if !strings.Contains(body, want) {
			t.Errorf("未匹配风险任务详情应包含 %q", want)
		}
	}
}

// TestV14TaskCompiledAbsentKeepsHonestBasis(负面)仓库没有 FrozenIntent 制品
// (LoadBundle 类型化失败):任务详情必须显式标注未接入及原因,且绝不渲染
// VerifiedBasis 结论、档位或 compiled 引用;六个分节与既有事实依据照常呈现
// —— 展示缺席不放宽任何验证门,依据哪里有就显示到哪里(AC-2)。
func TestV14TaskCompiledAbsentKeepsHonestBasis(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, _, code := v13Repo(t) // v1.3 布局:无 .tally/intents
	rs := runstore.New(st.Queries())
	v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p202-absent", "fact-p202-3", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/tasks/TASK-0001")
	if status != http.StatusOK {
		t.Fatalf("GET 任务详情 = %d, want 200", status)
	}
	for _, want := range wantStrings(v14TaskSections, []string{
		"已编译合同未接入",                 // 显式类型化原因(fail-safe 标注,不是静默缺席)
		"依据(Basis",                 // 既有事实依据照常呈现(INV-11)
		"fact:v13art:task_planned", // 依据引用落到事实 id
		"没有暴露事实",                   // 分节空状态如实呈现
	}) {
		if !strings.Contains(body, want) {
			t.Errorf("Compiled 缺席的任务详情应包含 %q", want)
		}
	}
	for _, banned := range []string{
		"验证结论(VerifiedBasis", // 不得渲染结论行
		"必需 L",               // 不得冒充必需档
		"达成 L",               // 不得冒充达成档
		"compiled:sha256:",   // 不得伪造编译权威引用(basis 必须真实)
	} {
		if strings.Contains(body, banned) {
			t.Errorf("Compiled 缺席时任务详情不得渲染 %q(不得冒充保证)", banned)
		}
	}
}

// wantStrings 把分组字串串接成一个期望列表(测试辅助,保持断言表扁平)。
func wantStrings(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

/* ---------- 种子辅助 ---------- */

// v14StartRun 开一条实现 Run:escalatedFrom > 0 表示修复尝试(父链接)。
func v14StartRun(t *testing.T, ctx context.Context, rs *runstore.Store, base, frozenHash, workspace string, escalatedFrom int64, startedAt time.Time) int64 {
	t.Helper()
	id, err := rs.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project: "demo-proj", Task: "TASK-0001", Role: "coding",
		Runtime: "fake", RuntimeVersion: "v0", Model: "demo-model", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneExecution, BaseCommit: base,
		TaskRevision: "sha256:taskrev", VerificationRevision: frozenHash, ContextPackHash: "sha256:cp",
		WorkspaceID:               workspace,
		RuntimeCapabilitySnapshot: []byte(`{"name":"fake","capabilities":[]}`),
		EscalatedFrom:             escalatedFrom,
		StartedAt:                 startedAt,
	})
	if err != nil {
		t.Fatalf("StartCodegenRun(%s): %v", workspace, err)
	}
	return id
}

// v14FixCandidate 为一次 Run 固定一个不可变候选快照(producer 回指 Run 行)。
func v14FixCandidate(t *testing.T, ctx context.Context, rs *runstore.Store, base, cand string, runID int64) changeset.ChangeSetID {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           "demo-proj",
		BaseRevision:      base,
		CandidateRevision: cand,
		ProducerRun:       strconv.FormatInt(runID, 10),
		CreatedAt:         time.Now().Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("构造候选(%s): %v", cand, err)
	}
	if err := rs.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("落候选快照: %v", err)
	}
	return cs.ID()
}

// v14Verify 给候选记一次验证 Run(观测结果)。
func v14Verify(t *testing.T, ctx context.Context, rs *runstore.Store, frozenHash string, cs changeset.ChangeSetID, status runstore.VerificationStatus, at time.Time) {
	t.Helper()
	if _, err := rs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
		Project:           "demo-proj",
		Task:              "TASK-0001",
		ContractRevision:  1,
		ContractHash:      frozenHash,
		ChangeSetID:       cs,
		EnvName:           "env@sha256:envdef",
		EnvDefinitionHash: "sha256:envdef",
		VersionCombo:      core.VersionCombo{"tool": "v1"},
		Runtime:           "fake",
		Status:            status,
		StartedAt:         at,
	}); err != nil {
		t.Fatalf("RecordVerificationRun: %v", err)
	}
}
