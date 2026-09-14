// v14_confirm_freeze_test.go 钉住 P0-08 的 handler 侧冻结流:确认 POST →
// intakefreeze 冻结准入 → artifacts.FreezeCommit 单一期望基线原子提交 →
// PRG 303 回 Launch 页。覆盖任务合同 Verification 要求的四个场景中 handler
// 可达的三个(成功 + 读回、重复确认幂等、准入阻断),以及数据源未接入的
// 如实降级;stale-base 冲突在事务层以显式期望基线覆盖
// (internal/artifacts/v141_freeze_commit_test.go:handler 每次提交前读实时
// HEAD,HTTP 层无法无损注入并发推进窗口)。
//
// 装配说明:303 结局不渲染页面,可用 v13Repo(带 standard 任务)夹具;
// 400 退回渲染整页会派生任务行(需要真实事实面),因此准入阻断用例用
// EnsureRepo 的最小仓库(无 .tally 制品 → 列表区如实渲染装载失败,不触
// 事实面),仓库本身仍是真实的,HEAD 不变断言依然有效。

package handlers_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"tally/internal/artifacts"
	"tally/internal/gitwriter"
	"tally/internal/store"
	"tally/internal/web/handlers"
)

// v14ConfirmForm 是一份能通过冻结准入的完整确认表单(为 TASK-0042 冻结:
// 治理等级显式 standard、owner 勾选保留一条风险提示、唯一验收条款声明
// repo 路径型 test 证据)。
func v14ConfirmForm() url.Values {
	return url.Values{
		"task_type":       {"bug_fix"},
		"summary":         {"修复登出后会话残留"},
		"objective":       {"登出后销毁全部服务端会话状态"},
		"allowed_paths":   {"src/auth/**"},
		"forbidden_paths": {"vendor/**"},
		"acid.0":          {"AC-01"},
		"acstmt.0":        {"登出后旧会话 token 必须失效"},
		"ev_kind.0":       {"test"},
		"ev_path.0":       {"src/auth/logout_test.go"},
		"task_id":         {"TASK-0042"},
		"profile":         {"standard"},
		"source_ref":      {"GitHub Issue #142"},
		"risk":            {"auth_session"}, // 候选(隐藏域回传)
		"risk_hint":       {"auth_session"}, // owner 确认保留
	}
}

// v14FreezeRepo 是准入阻断用例的最小真实制品仓库(有 HEAD、无 .tally 制品)。
func v14FreezeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := handlers.EnsureRepo(dir, handlers.Identity{Name: "test", Email: "test@example.com"}); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	return dir
}

// v14RepoHead 返回制品仓库当前 HEAD sha。
func v14RepoHead(t *testing.T, dir string) string {
	t.Helper()
	w, err := gitwriter.New(dir)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	h, err := w.Head(context.Background())
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	return h
}

// v14FlashOf 解码 303 Location 里的 kind/flash(query 值是 URL 编码的)。
func v14FlashOf(t *testing.T, resp *http.Response) (string, string) {
	t.Helper()
	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("解析 Location %q: %v", resp.Header.Get("Location"), err)
	}
	q := u.Query()
	return q.Get("kind"), q.Get("flash")
}

func TestV14ConfirmFreezesTripleAtomicallyAndReloads(t *testing.T) {
	repo, _, _, _ := v13Repo(t)
	before := v14RepoHead(t, repo)
	srv := v14NewServer(t, &store.Queries{}, repo, nil)

	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/confirm", v14ConfirmForm())
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("确认冻结应 303(PRG),得到 %d", resp.StatusCode)
	}
	kind, flash := v14FlashOf(t, resp)
	if kind != "ok" {
		t.Fatalf("确认应成功(kind=ok),得到 kind=%s flash=%s", kind, flash)
	}
	if !strings.Contains(flash, "冻结完成") || !strings.Contains(flash, "TASK-0042") {
		t.Errorf("成功提示应说明任务已冻结,得到 %q", flash)
	}

	// 恰一个新提交;成功提示携带提交 sha。
	after := v14RepoHead(t, repo)
	if after == "" || after == before {
		t.Fatalf("确认应产生新提交,before=%s after=%s", before, after)
	}
	if !strings.Contains(flash, after[:12]) {
		t.Errorf("成功提示应携带提交 sha %q,得到 %q", after[:12], flash)
	}

	// AC2:成功确认可从提交的制品确定性读回 —— 任务合同 + 冻结验证契约经
	// Load,意图证据经 LoadIntent。
	snap, err := artifacts.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("Load(HEAD): %v", err)
	}
	found := false
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == "TASK-0042" {
			found = true
			if snap.Tasks[i].Contract.Profile != "standard" {
				t.Errorf("冻结 profile = %q", snap.Tasks[i].Contract.Profile)
			}
		}
	}
	if !found {
		t.Fatalf("HEAD 快照里没有 TASK-0042 的 TaskContract(任务 %v)", pathsOfTasks(snap))
	}
	if !v14HasVerificationFor(snap, "TASK-0042") {
		t.Fatalf("HEAD 快照里没有 TASK-0042 的冻结 VerificationContract")
	}
	ia, err := artifacts.LoadIntent(repo, "HEAD", "TASK-0042")
	if err != nil {
		t.Fatalf("LoadIntent(HEAD, TASK-0042): %v", err)
	}
	if ia.Intent.SourceRef() != "GitHub Issue #142" {
		t.Errorf("重载 intent SourceRef = %q", ia.Intent.SourceRef())
	}
	if hints := ia.Intent.RiskHints(); len(hints) != 1 || hints[0] != "auth_session" {
		t.Errorf("重载 intent RiskHints = %v,want owner 确认保留的 [auth_session]", hints)
	}
}

func TestV14ConfirmDuplicateIsIdempotent(t *testing.T) {
	repo, _, _, _ := v13Repo(t)
	srv := v14NewServer(t, &store.Queries{}, repo, nil)

	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/confirm", v14ConfirmForm())
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("首次确认应 303,得到 %d", resp.StatusCode)
	}
	if kind, _ := v14FlashOf(t, resp); kind != "ok" {
		t.Fatalf("首次确认应 kind=ok,得到 %s", kind)
	}
	first := v14RepoHead(t, repo)

	// 完全相同的确认再提交:幂等 —— 不产生新提交,提示如实说明。
	resp, _ = v14PostForm(t, srv.URL, "/v14/launch/confirm", v14ConfirmForm())
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("重复确认应 303,得到 %d", resp.StatusCode)
	}
	kind, flash := v14FlashOf(t, resp)
	if kind != "ok" || !strings.Contains(flash, "未产生新提交") {
		t.Errorf("重复确认应幂等并如实说明,得到 kind=%s flash=%q", kind, flash)
	}
	if now := v14RepoHead(t, repo); now != first {
		t.Fatalf("重复确认不得产生新提交,HEAD %s → %s", first, now)
	}
}

func TestV14ConfirmAdmissionFailuresBlockFreeze(t *testing.T) {
	repo := v14FreezeRepo(t)
	before := v14RepoHead(t, repo)
	srv := v14NewServer(t, &store.Queries{}, repo, nil)

	// 缺证据声明:遗漏防护类型化阻断,退回表单,不产生任何冻结提交。
	form := v14ConfirmForm()
	form.Del("ev_kind.0")
	form.Del("ev_path.0")
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺证据的确认应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "planned evidence") || !strings.Contains(body, "AC-01") {
		t.Errorf("应类型化列出缺失证据的条款(AC-01),body:\n%s", body[:min(len(body), 800)])
	}

	// 证据半声明(只有类型没有路径):表单级类型化拒绝。
	form = v14ConfirmForm()
	form.Del("ev_path.0")
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("证据半声明应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "必须同时声明") {
		t.Errorf("半声明应类型化说明证据类型与路径必须同时声明,body:\n%s", body[:min(len(body), 800)])
	}

	// 任务 id 非法(含空白):意图证据 fail-closed。
	form = v14ConfirmForm()
	form.Set("task_id", "TASK 0042")
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法任务 id 应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "含空白") {
		t.Errorf("应类型化说明任务 id 非法,body:\n%s", body[:min(len(body), 800)])
	}

	// 全部失败路径都不许动仓库。
	if now := v14RepoHead(t, repo); now != before {
		t.Fatalf("准入失败不得产生提交,HEAD %s → %s", before, now)
	}
}

func TestV14ConfirmWithoutRepoStatesUnavailable(t *testing.T) {
	// 制品仓库未接入:303 + kind=bad 如实说明,不假装冻结成功。
	srv := v14NewServer(t, nil, "", nil)
	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/confirm", v14ConfirmForm())
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("仓库未接入应 303,得到 %d", resp.StatusCode)
	}
	kind, flash := v14FlashOf(t, resp)
	if kind != "bad" || !strings.Contains(flash, "未能冻结任务") {
		t.Errorf("应如实说明未能冻结,得到 kind=%s flash=%q", kind, flash)
	}
}

// v14HasVerificationFor 报告快照里是否有目标任务 id 的冻结验证契约。
func v14HasVerificationFor(snap *artifacts.Snapshot, taskID string) bool {
	for i := range snap.Verifications {
		if snap.Verifications[i].Frozen.TaskID() == taskID {
			return true
		}
	}
	return false
}

// pathsOfTasks 是快照里任务 id 的列表(断言消息用)。
func pathsOfTasks(snap *artifacts.Snapshot) []string {
	out := []string{}
	for i := range snap.Tasks {
		out = append(out, snap.Tasks[i].Contract.ID)
	}
	return out
}
