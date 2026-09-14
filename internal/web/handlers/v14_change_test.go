package handlers_test

// P3-09 的 Change Detail 页测试(PRD §26.3)。
//
// 两层,与 v13_http_test.go / v14_task_test.go 同一模式:
//   - 纯 HTTP 层(不需要数据库):数据源未接入空状态;
//   - 全链路层(临时 Git 制品仓库 + 独立 schema 的真实 Postgres,不可达即
//     SKIP):改了什么(确定性 hunk 清单)、为什么可信(验证事实 + Basis
//     引用 + 依据四维度)、哪里不确定(门槛缺口 / 取代 / 暴露可见)、谱系
//     (取代链与祖先暴露,P0-07)。
//
// 全部断言只针对派生/事实投影:页面没有任何可写状态,默认没有 Agent 会话转录。

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/verification"
)

func TestV14ChangeUnavailableWithoutSources(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/abc")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/changes/abc = %d, want 200", status)
	}
	if !strings.Contains(body, "未接入") {
		t.Errorf("无数据源时应渲染未接入空状态")
	}
}

func TestV14ChangeNotFound(t *testing.T) {
	st := v13TestDB(t) // 需要 Postgres 才有事实面;不可达即 SKIP。
	repo, _, _, _ := v13Repo(t)
	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if status != http.StatusNotFound {
		t.Fatalf("未知候选应 404,得到 %d", status)
	}
	if !strings.Contains(body, "没有这份 ChangeSet") {
		t.Errorf("应说明候选身份由内容推导,body:\n%s", body)
	}
}

// TestV14ChangeDetailFourQuestionsAndLineage 走一条修复链:候选1(docs)被
// 候选2(code)以 rework 取代,候选1 上有一条祖先锚定的暴露事实 —— 当前候选
// 页面必须仍能看到它(P0-07/AC3)。页面按 PRD §26.3 回答四问。
func TestV14ChangeDetailFourQuestionsAndLineage(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	frozenHash := v13FrozenHash(t, repo, base)
	rs := runstore.New(st.Queries())

	when := time.Now().Add(-time.Hour)
	// 尝试 1(根)固定 docs 候选;尝试 2(修复)固定 code 候选并取代前者。
	run1 := v14StartRun(t, ctx, rs, base, frozenHash, "ws-p309-a", 0, when)
	cs1 := v14cCandidate(t, ctx, rs, base, docs, run1, when.Add(5*time.Minute))
	run2 := v14StartRun(t, ctx, rs, base, frozenHash, "ws-p309-b", run1, when.Add(10*time.Minute))
	cs2 := v14cCandidate(t, ctx, rs, base, code, run2, when.Add(20*time.Minute))

	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: run1, Status: runstore.RunCompleted, Iterations: 3,
		ResultPatchHash: string(cs1), AgentClaim: "第一版已经修好", FinishedAt: when.Add(6 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(run1): %v", err)
	}
	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: run2, Status: runstore.RunCompleted, Iterations: 2,
		ResultPatchHash: string(cs2), AgentClaim: "已修复会话残留", FinishedAt: when.Add(21 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(run2): %v", err)
	}
	// 取代事实:修复候选取代第一候选(append-only,不合并)。
	f, err := changeset.NewSupersededFact(cs1, cs2, changeset.SupersedeRework, when.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := rs.AppendSupersededFact(ctx, f); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}
	// 当前候选的验证:pass,并留一条可引用的证据事实(INV-11)。
	v14cVerifyEvidence(t, ctx, rs, frozenHash, cs2, runstore.VerificationPass, when.Add(25*time.Minute), "fact-p309-1")

	// 祖先暴露:锚定在已被取代的 cs1 上 —— 取代不重置暴露账。
	if _, err := rs.RecordExposure(ctx, runstore.ExposureRecord{
		Task:             "TASK-0001",
		ContractRevision: 1,
		Binding: evidence.BindingRef{
			Project:      core.ProjectID("demo-proj"),
			Verification: evidence.VerificationRevision(frozenHash),
			BindingID:    "EV-AC01-01",
		},
		Acceptance: evidence.AcceptanceID("AC-01"),
		ChangeSet:  evidence.ChangeSetID(cs1),
		Channel:    evidence.ExposureChannel("repair_workspace"),
		Reason:     "P3-09 祖先暴露投影测试",
		ExposedAt:  when.Add(30 * time.Minute),
	}); err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}

	srv := v14NewServer(t, st.Queries(), repo, nil)

	// ---- 当前候选页:改了什么 / 为什么可信 / 哪里不确定 / 谱系。
	status, body := v13Get(t, srv.URL, "/v14/changes/"+string(cs2))
	if status != http.StatusOK {
		t.Fatalf("GET 当前候选详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"Change Detail",
		// 改了什么:快照固定文件 + Git 确定性重建的 hunk 清单(§19.2)。
		"src/app/handler.go",
		"确定性 hunk 清单",
		"return 1",
		// 为什么可信:验证观测计数 + 证据 fact 引用 + 派生 Basis 引用(INV-11)。
		"pass 1",
		"fact-p309-1",
		"fact:v13art:task_planned",
		// 依据四维度(PRD §14):已接入/未接入如实标注(AC1),VERIFIED_LIMITED
		// 的 fail-safe 语义可见,展示缺席不冒充。
		"coverage",
		"assurance",
		"unmatched",
		"waiver",
		"未接入",
		"VERIFIED_LIMITED",
		// 哪里不确定:派生门槛缺口可见(功能/independence 覆盖报告缺失)。
		"门槛缺口",
		// 谱系:取代关系 + 祖先锚定的暴露可见(P0-07/AC3)+ 当前候选标记。
		"取代",
		"祖先 · 第 1 代",
		"条暴露锚定在取代祖先上",
		"当前候选",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("当前候选详情应包含 %q", want)
		}
	}
	// 默认无 Agent 会话转录;Agent 自述是 claim,不是本页依据。
	if strings.Contains(body, "claim:") {
		t.Errorf("Change Detail 不应包含 Agent 自述转录(INV-AUTH)")
	}

	// ---- 被取代的祖先候选页:取代可见、暴露可见、不得显示为当前。
	status, body = v13Get(t, srv.URL, "/v14/changes/"+string(cs1))
	if status != http.StatusOK {
		t.Fatalf("GET 祖先候选详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"docs/guide.md", // 改了什么:祖先候选自己的确定性清单
		"已被取代",          // 取代事实可见(修复谱系,AC2)
		"当前候选",          // 自身锚定的暴露行(锚点 = 本候选)
		"门槛缺口",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("祖先候选详情应包含 %q", want)
		}
	}
	if strings.Contains(body, "本候选即当前候选") {
		t.Errorf("被取代的候选不得显示为当前候选")
	}
}

/* ---------- v1.4.1 P1-07:编译权威读模型(compiled:<hash> basis)---------- */

// v14cIntentYAML 渲染 TASK-0001 的 FrozenIntent 制品文本(恰四字段冻结
// schema,DL-06;riskHints 非空时附 owner 确认的 risk_hints 声明)。
func v14cIntentYAML(riskHints ...string) string {
	b := "task_id: TASK-0001\ntask_type: bug_fix\nsource_ref: GitHub Issue #142\n"
	if len(riskHints) > 0 {
		b += "risk_hints:\n"
		for _, h := range riskHints {
			b += "- " + h + "\n"
		}
	}
	return b
}

// v14cIntentRepo 在 v13Repo 完全相同的布局上(同一 task/verification 制品与
// docs/code 候选提交),于 base 提交额外冻结一份 FrozenIntent
// (.tally/intents/TASK-0001.yaml),使 LoadBundle → CompileAuthority 生产
// seam 在本仓库可用;extra 里的文件在 base 提交前一并写入(非法策略制品等
// 负面场景)。
func v14cIntentRepo(t *testing.T, extra map[string]string, riskHints ...string) (dir, base, docs, code string) {
	t.Helper()
	dir = t.TempDir()
	r, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := r.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	when := time.Now().Add(-time.Hour)
	commit := func(msg string) string {
		t.Helper()
		h, err := wt.Commit(msg, &git.CommitOptions{
			Author:    &object.Signature{Name: "test", Email: "test@example.com", When: when},
			Committer: &object.Signature{Name: "test", Email: "test@example.com", When: when},
		})
		if err != nil {
			t.Fatalf("提交 %s: %v", msg, err)
		}
		return h.String()
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("建目录 %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("写 %s: %v", path, err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}

	write(".tally/project.yaml", "project: demo-proj\n")
	write(".tally/tasks/TASK-0001.yaml", v13TaskYAML)
	write(".tally/verifications/TASK-0001.yaml", v13VerifYAML)
	write(".tally/intents/TASK-0001.yaml", v14cIntentYAML(riskHints...))
	for p, c := range extra {
		write(p, c)
	}
	base = commit("base: .tally 制品(含 FrozenIntent)")

	write("docs/guide.md", "# guide\n\ndemo\n")
	docs = commit("docs: 演示文档改动")

	write("src/app/handler.go", "package app\n\nfunc Handler() int { return 1 }\n")
	code = commit("feat: 实现代码改动(规则无法归账的残差)")

	return dir, base, docs, code
}

// v14cPassCandidate 种一条完成的 Run + code 候选 + 一条 pass 验证证据,
// 返回候选身份(P1-07 编译权威读模型测试的公共夹具)。
func v14cPassCandidate(t *testing.T, ctx context.Context, rs *runstore.Store, repo, base, code, workspace, factID string, at time.Time) changeset.ChangeSetID {
	t.Helper()
	frozenHash := v13FrozenHash(t, repo, base)
	runID := v14StartRun(t, ctx, rs, base, frozenHash, workspace, 0, at)
	cs := v14cCandidate(t, ctx, rs, base, code, runID, at.Add(5*time.Minute))
	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: runID, Status: runstore.RunCompleted, Iterations: 1,
		ResultPatchHash: string(cs), AgentClaim: "我做完了(只是 claim)", FinishedAt: at.Add(6 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun: %v", err)
	}
	v14cVerifyEvidence(t, ctx, rs, frozenHash, cs, runstore.VerificationPass, at.Add(10*time.Minute), factID)
	return cs
}

// TestV14ChangeCompiledBasisPlainVerified 全绿候选(零策略 + 零确认风险信号):
// 验证结论必须是普通 VERIFIED,依据列表携带 compiled:<hash>,assurance 维度
// 给出必需/达成档事实值(P1-07 AC:结论区分 + basis 列表)。
func TestV14ChangeCompiledBasisPlainVerified(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t) // 需要 Postgres 才有事实面;不可达即 SKIP。
	repo, base, _, code := v14cIntentRepo(t, nil)
	rs := runstore.New(st.Queries())
	cs := v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p107-plain", "fact-p107-1", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/"+string(cs))
	if status != http.StatusOK {
		t.Fatalf("GET 候选详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"验证结论(VerifiedBasis",       // 结论行渲染(与 v1.3 状态行可区分)
		"compiled:sha256:",         // 结论可追溯到已编译合同身份(INV-STATE-02)
		"必需 L1",                    // assurance 维度事实值:零信号默认地板
		"达成 L1",                    // visible deterministic 证据的达成档
		"无未匹配风险信号",                 // unmatched 维度事实值(命中策略 0)
		"fact:v13art:task_planned", // 依据列表照旧携带事实引用(INV-11)
	} {
		if !strings.Contains(body, want) {
			t.Errorf("全绿候选详情应包含 %q", want)
		}
	}
	if strings.Contains(body, "VERIFIED_LIMITED") {
		t.Errorf("全绿候选不得显示 VERIFIED_LIMITED(不得把普通 VERIFIED 降格展示)")
	}
	if strings.Contains(body, "已编译合同未接入") {
		t.Errorf("编译权威已接入时不得渲染未接入标注")
	}
}

// TestV14ChangeCompiledBasisVerifiedLimitedOnUnmatchedRisk 确认风险提示
// auth_symbol_change 无任何命中策略声明:状态仍是 VERIFIED,但验证结论必须是
// VERIFIED_LIMITED(不得显示成普通 VERIFIED),未匹配信号原样可见,必需档
// 被 fail-safe 上调 L3(PRD §14;INV-V14-09)。
func TestV14ChangeCompiledBasisVerifiedLimitedOnUnmatchedRisk(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, _, code := v14cIntentRepo(t, nil, "auth_symbol_change")
	rs := runstore.New(st.Queries())
	cs := v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p107-limited", "fact-p107-2", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/"+string(cs))
	if status != http.StatusOK {
		t.Fatalf("GET 候选详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"VERIFIED_LIMITED",           // limited 结论,不是普通 VERIFIED
		"未匹配风险信号:auth_symbol_change", // 信号原样可见(verbatim)
		"必需 L3",                      // fail-safe 上调的必需档
		"达成 L1",                      // 最弱条款达成档诚实上报
		"compiled:sha256:",           // 结论依据已编译合同身份
	} {
		if !strings.Contains(body, want) {
			t.Errorf("未匹配风险候选详情应包含 %q", want)
		}
	}
}

// TestV14ChangeCompiledAbsentDoesNotClaimAssurance 仓库没有 FrozenIntent 制品
// (LoadBundle 类型化失败):页面必须显式标注未接入及其原因,且绝不渲染
// VerifiedBasis 结论、档位或 compiled 引用(P1-07 要求 2:Compiled 缺席不得
// 宣称任何保证)。
func TestV14ChangeCompiledAbsentDoesNotClaimAssurance(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, _, code := v13Repo(t) // v1.3 布局:无 .tally/intents
	rs := runstore.New(st.Queries())
	cs := v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p107-absent", "fact-p107-3", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/"+string(cs))
	if status != http.StatusOK {
		t.Fatalf("GET 候选详情 = %d, want 200", status)
	}
	for _, want := range []string{
		"已编译合同未接入", // 显式原因(fail-safe 标注,不是静默缺席)
		"assurance",
		"unmatched",
		"未接入",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Compiled 缺席时详情应包含 %q", want)
		}
	}
	for _, banned := range []string{
		"验证结论(VerifiedBasis", // 不得渲染结论行
		"必需 L",               // 不得冒充必需档
		"达成 L",               // 不得冒充达成档
		"compiled:sha256:",   // 不得伪造编译权威引用(basis 必须真实)
	} {
		if strings.Contains(body, banned) {
			t.Errorf("Compiled 缺席时不得渲染 %q(不得冒充保证)", banned)
		}
	}
}

// TestV14ChangeInvalidPolicyKeepsCompiledAbsent 快照里有非法策略制品:
// CompileAuthority 的类型化 ErrPolicyRejections 使编译权威缺席 —— 页面保持
// 未接入标注,绝不降级成「无策略的普通 VERIFIED」(P1-03 fail-safe 语义在
// 读路径的延续)。
func TestV14ChangeInvalidPolicyKeepsCompiledAbsent(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, _, code := v14cIntentRepo(t, map[string]string{
		".tally/policies/policy-bad.yaml": "rationale: 缺少 id/enforcement 的非法策略制品\n",
	})
	rs := runstore.New(st.Queries())
	cs := v14cPassCandidate(t, ctx, rs, repo, base, code, "ws-p107-badpol", "fact-p107-4", time.Now().Add(-time.Hour))

	srv := v14NewServer(t, st.Queries(), repo, nil)
	status, body := v13Get(t, srv.URL, "/v14/changes/"+string(cs))
	if status != http.StatusOK {
		t.Fatalf("GET 候选详情 = %d, want 200", status)
	}
	if !strings.Contains(body, "已编译合同未接入") {
		t.Errorf("策略拒绝阻断编译权威时应显式标注未接入")
	}
	for _, banned := range []string{"验证结论(VerifiedBasis", "必需 L", "达成 L", "compiled:sha256:"} {
		if strings.Contains(body, banned) {
			t.Errorf("编译权威被拒绝时不得渲染 %q", banned)
		}
	}
}

/* ---------- 种子辅助 ---------- */

// v14cCandidate 为一次 Run 固定不可变候选,创建时刻显式传入(保证取代链的
// 规范顺序确定,不依赖时钟分辨率)。
func v14cCandidate(t *testing.T, ctx context.Context, rs *runstore.Store, base, cand string, runID int64, createdAt time.Time) changeset.ChangeSetID {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           "demo-proj",
		BaseRevision:      base,
		CandidateRevision: cand,
		ProducerRun:       strconv.FormatInt(runID, 10),
		CreatedAt:         createdAt,
	})
	if err != nil {
		t.Fatalf("构造候选(%s): %v", cand, err)
	}
	if err := rs.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("落候选快照: %v", err)
	}
	return cs.ID()
}

// v14cVerifyEvidence 给候选记一次验证 Run 并留一条 pass/fail 证据事实
// (fact 引用在 Change Detail 的证据表可见,INV-11)。
func v14cVerifyEvidence(t *testing.T, ctx context.Context, rs *runstore.Store, frozenHash string, cs changeset.ChangeSetID, status runstore.VerificationStatus, at time.Time, factID string) {
	t.Helper()
	vrID, err := rs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
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
	})
	if err != nil {
		t.Fatalf("RecordVerificationRun: %v", err)
	}
	verdict := evidence.VerdictPass
	if status == runstore.VerificationFail {
		verdict = evidence.VerdictFail
	}
	if _, err := rs.RecordEvidence(ctx, core.ProjectID("demo-proj"), runstore.EvidenceRecord{
		FactID:           evidence.FactID(factID),
		RunID:            vrID,
		Task:             "TASK-0001",
		Acceptance:       "AC-01",
		BindingID:        "EV-AC01-01",
		VisibilityPolicy: verification.VisibilityVisible,
		Verdict:          verdict,
		Recorder:         evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "demo-v1"},
		RecordedAt:       at,
	}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
}
