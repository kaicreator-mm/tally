package handlers_test

// P2-03 Runs 投影测试:真实 run fixtures(临时 Git 制品仓库 + 独立 schema 的
// 真实 Postgres;数据库不可达时 SKIP,不产假失败)。
//
// 覆盖:
//   - 运行时间线原样呈现 codegen_run 行事实(status / 迭代 / 用量 / 升级链 /
//     失败原因 / 固定候选),期望坐标与 runtime 自报实际身份分开呈现;
//   - 页面导航高亮(运行 aria-current)与任务/候选链接接线;
//   - 负路径:无运行时诚实说明缺席,不出现任何状态/候选/失败原因的编造渲染。

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/measurement"
	"tally/internal/runstore"
	"tally/internal/store"
)

// v13RunsSeed 在事实面里种一条确定的运行时间线(TASK-0001,时刻递增):
//
//	r1 completed(固定 docsCS,带用量)
//	r2 escalated(升级去 strong;升级即换 Run)
//	r3 completed(EscalatedFrom=r2;固定 codeCS;带 runtime 自报实际身份)
//	r4 failed(runtime_error,带用量)
//	r5 running(未终态、用量未上报)
//
// 返回 r1..r3 的行 id 与两个候选身份。
func v13RunsSeed(
	t *testing.T, ctx context.Context, q *store.Queries, base, docs, code string,
) (r1, r2, r3 int64, docsCS, codeCS changeset.ChangeSetID) {
	t.Helper()
	rs := runstore.New(q)
	t0 := time.Now().Add(-2 * time.Hour).Truncate(time.Second)

	start := func(at time.Time, escalatedFrom int64, observed bool) int64 {
		t.Helper()
		spec := runstore.StartRunSpec{
			Project:                   "demo-proj",
			Task:                      "TASK-0001",
			Role:                      "coder",
			Runtime:                   "fake",
			RuntimeVersion:            "v0",
			Model:                     "demo-model",
			BehaviorRevision:          "b1",
			CostPlane:                 measurement.CostPlaneExecution,
			BaseCommit:                base,
			TaskRevision:              "sha256:taskrev",
			VerificationRevision:      "sha256:verifrev",
			ContextPackHash:           "sha256:cp",
			WorkspaceID:               "ws-" + strconv.FormatInt(at.UnixMilli(), 10),
			RuntimeCapabilitySnapshot: []byte(`{"runtime":"fake"}`),
			EscalatedFrom:             escalatedFrom,
			StartedAt:                 at,
		}
		if observed {
			spec.ObservedRuntimeName = "codex-cli"
			spec.ObservedRuntimeVersion = "0.42.0"
			spec.ObservedModel = "actual-model-x"
		}
		id, err := rs.StartCodegenRun(ctx, spec)
		if err != nil {
			t.Fatalf("StartCodegenRun: %v", err)
		}
		return id
	}
	finish := func(id int64, status runstore.RunStatus, at time.Time, in, out int64, patch, reason string) {
		t.Helper()
		ok, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
			ID:              id,
			Status:          status,
			FinishedAt:      at,
			Iterations:      3,
			InputTokens:     &in,
			OutputTokens:    &out,
			ResultPatchHash: patch,
			FailureReason:   reason,
		})
		if err != nil || !ok {
			t.Fatalf("FinishCodegenRun(%d,%s): ok=%v err=%v", id, status, ok, err)
		}
	}
	fix := func(producer int64, cand string) changeset.ChangeSetID {
		t.Helper()
		cs, err := changeset.New(changeset.Spec{
			Project:           "demo-proj",
			BaseRevision:      base,
			CandidateRevision: cand,
			ProducerRun:       strconv.FormatInt(producer, 10),
		})
		if err != nil {
			t.Fatalf("构造候选(%s): %v", cand, err)
		}
		if err := rs.SaveChangeSet(ctx, cs); err != nil {
			t.Fatalf("落候选快照: %v", err)
		}
		return cs.ID()
	}

	min := time.Minute
	r1 = start(t0, 0, false)
	docsCS = fix(r1, docs)
	finish(r1, runstore.RunCompleted, t0.Add(10*min), 100, 200, string(docsCS), "")

	r2 = start(t0.Add(20*min), 0, false)
	finish(r2, runstore.RunEscalated, t0.Add(25*min), 30, 40, "", "escalated_to:strong")

	r3 = start(t0.Add(30*min), r2, true)
	codeCS = fix(r3, code)
	finish(r3, runstore.RunCompleted, t0.Add(38*min), 11, 22, string(codeCS), "")

	r4 := start(t0.Add(40*min), 0, false)
	finish(r4, runstore.RunFailed, t0.Add(42*min), 5, 7, "", "runtime_error")

	start(t0.Add(50*min), 0, false) // r5:保持 running,无终态、无用量
	return r1, r2, r3, docsCS, codeCS
}

// TestV13RunsPageShowsRunTimelineFacts:运行页原样呈现行的过程事实 —— 状态、
// 升级链、失败原因、固定候选链接、期望坐标与自报实际身份分开;新到在前。
func TestV13RunsPageShowsRunTimelineFacts(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	_, r2, _, docsCS, _ := v13RunsSeed(t, ctx, st.Queries(), base, docs, code)

	srv := v13NewServer(t, st.Queries(), repo)
	status, body := v13Get(t, srv.URL, "/v13/runs")
	if status != http.StatusOK {
		t.Fatalf("GET /v13/runs = %d, want 200", status)
	}
	for _, want := range []string{
		"<h1>运行</h1>",
		"共 5 次运行",                                             // 汇总由行推导
		">completed<", ">escalated<", ">failed<", ">running<", // 行上状态原样
		"escalated_to:strong", // 升级 Run 的失败原因(升级去向)原样
		"runtime_error",       // 失败原因原样
		"升级自 Run #" + strconv.FormatInt(r2, 10), // 升级链来源
		`href="/v13/changes/` + string(docsCS),  // completed Run 固定的候选可点开
		`href="/v14/tasks/TASK-0001"`,           // 每行可回到任务(Work → Task)
		"期望坐标(开跑写定)",                            // 期望坐标与实际身份分开(Decision Lock 15)
		"实际 codex-cli / actual-model-x",         // runtime 自报实际身份
		"实际身份未观察",                               // 未观察的行如实说明,不用期望值补齐
		"in 100 · out 200",                      // 用量事实
		"3 次迭代",
		"用量未上报", // running 且未上报的行
	} {
		if !strings.Contains(body, want) {
			t.Errorf("运行页应包含 %q", want)
		}
	}

	// 新到在前:晚开始的行先渲染(r4 用量在 r1 之前)。
	iLate := strings.Index(body, "in 5 · out 7")
	iEarly := strings.Index(body, "in 100 · out 200")
	if iLate < 0 || iEarly < 0 || iLate > iEarly {
		t.Errorf("运行时间线应新到在前(r4 在 r1 之前):late=%d early=%d", iLate, iEarly)
	}

	// 未种 cancelled:状态色板里的其余值不得凭空出现(状态来自行,不是清单)。
	if strings.Contains(body, ">cancelled<") {
		t.Errorf("事实面没有 cancelled 行,页面不得出现 cancelled 状态")
	}

	// 导航高亮:运行条目是当前项,工作不是。
	nav := navRail(t, body)
	if line := navHrefLine(t, nav, "/v13/runs"); !strings.Contains(line, `aria-current="page"`) {
		t.Errorf("运行页应高亮运行条目,得到行:%s", line)
	}
	if line := navHrefLine(t, nav, "/v13/tasks"); strings.Contains(line, `aria-current="page"`) {
		t.Errorf("运行页不得同时高亮工作条目,得到行:%s", line)
	}
}

// TestV13RunsPageWithoutRunsStatesAbsenceHonestly(负路径):没有运行事实时
// 页面诚实说明缺席,绝不渲染任何状态/候选/失败原因(不编造、不占位)。
func TestV13RunsPageWithoutRunsStatesAbsenceHonestly(t *testing.T) {
	st := v13TestDB(t)
	repo, _, _, _ := v13Repo(t)

	srv := v13NewServer(t, st.Queries(), repo)
	status, body := v13Get(t, srv.URL, "/v13/runs")
	if status != http.StatusOK {
		t.Fatalf("GET /v13/runs = %d, want 200", status)
	}
	if !strings.Contains(body, "还没有任何实现 Run") {
		t.Errorf("无运行时应诚实说明缺席")
	}
	for _, banned := range []string{
		">completed<", ">failed<", ">escalated<", ">cancelled<", ">running<",
		"升级自", "escalated_to", "runtime_error",
		`href="/v13/changes/`, // 没有运行就没有候选,不得出现候选链接
		"读取失败",
	} {
		if strings.Contains(body, banned) {
			t.Errorf("无运行时不得出现 %q(缺席要诚实,不编造事实)", banned)
		}
	}
}
