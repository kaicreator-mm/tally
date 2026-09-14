package requirement

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"
)

// specDefectSinkShape 用镜像接口守住与 failure.SpecDefectSink 的同形关系
// (业务包互不导入,形状由各自包内的镜像接口守住,上层按形状接线)。
// failure 侧若改签名,这里编译失败,接线人立刻知道。
var _ interface {
	ReportSpecDefect(ctx context.Context, projectID, clauseID, ambiguity string) error
} = SpecDefectSink{}

// reqSrc 是测试用的项目需求文档(真实解析器格式,REQ-1 / REQ-2 两条条款)。
const reqSrc = `# 草稿管理

## REQ-1 保存

WHEN 用户点击保存, 系统 shall 将草稿写入 git

验收: 自动化测试
- 保存后仓库出现新提交

## REQ-2 加载

WHEN 用户打开草稿, 系统 shall 展示最近一次内容

验收: 自动化测试
- 打开后显示内容

## 非目标

- 不做云端同步

## 术语表

- 草稿: 未签署的需求文本
`

// writeReqDocs 把项目需求文档写进仓库(.tally/requirements/myproj/draft.md)。
// 条款校验读的是磁盘上的解析结果,文件在即条款在。
func writeReqDocs(t *testing.T, repo string) {
	t.Helper()
	dir := filepath.Join(repo, ".tally", "requirements", "myproj")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建需求目录: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "draft.md"), []byte(reqSrc), 0o644); err != nil {
		t.Fatalf("写需求文档: %v", err)
	}
}

// 正例:条款存在 → 提案留痕进 git,钩在提案时条款的内容哈希上,状态待审。
func TestReportSpecDefectKnownClause(t *testing.T) {
	repo := initRepo(t)
	writeReqDocs(t, repo)
	sink := SpecDefectSink{RepoPath: repo, Identity: testIdentity()}

	if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-1", "写满磁盘时的异常分支未定义"); err != nil {
		t.Fatalf("ReportSpecDefect 出错: %v", err)
	}

	recs, err := LoadSpecDefects(repo)
	if err != nil {
		t.Fatalf("LoadSpecDefects 出错: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("提案数 = %d, 期望 1", len(recs))
	}
	rec := recs[0]
	if rec.Project != "myproj" || rec.ClauseID != "REQ-1" {
		t.Errorf("提案定位不符: %+v", rec)
	}
	if rec.Status != SpecDefectNeedsReview {
		t.Errorf("状态 = %q, 期望 %q(只留痕,不代替 owner 决策)", rec.Status, SpecDefectNeedsReview)
	}
	if rec.Ambiguity != "写满磁盘时的异常分支未定义" {
		t.Errorf("模糊点不符: %q", rec.Ambiguity)
	}
	// 确定性挂钩:哈希必须等于该条款当前内容的 ContentHash
	want := ContentHash(Parse(reqSrc).Clauses[0])
	if rec.Hash != want {
		t.Errorf("挂钩哈希 = %s, 期望 ContentHash(REQ-1) = %s", rec.Hash, want)
	}
	if !rec.ReportedAt.Equal(testIdentity().When) {
		t.Errorf("提案时间 = %v, 期望身份注入的 %v", rec.ReportedAt, testIdentity().When)
	}

	// 留痕进了 git(有提交,提交信息指向条款)
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("打开仓库: %v", err)
	}
	head, err := r.Head()
	if err != nil || head.Hash().IsZero() {
		t.Fatalf("提案应产生 git 提交: %v", err)
	}
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("取提交: %v", err)
	}
	if !strings.Contains(commit.Message, "REQ-1") {
		t.Errorf("提交信息应指向条款: %q", commit.Message)
	}
	if !strings.Contains(commit.Message, "clause-hash: REQ-1 "+string(want)) {
		t.Errorf("提交信息应记录内容哈希(同签署约定): %q", commit.Message)
	}
}

// 负例(AC1):条款不存在 → 错误,提案不成立,不留任何痕。
func TestReportSpecDefectUnknownClause(t *testing.T) {
	repo := initRepo(t)
	writeReqDocs(t, repo)
	sink := SpecDefectSink{RepoPath: repo, Identity: testIdentity()}

	if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-99", "编造的条款"); err == nil {
		t.Fatal("不存在的条款应报错(LLM 编不出不存在的条款)")
	}
	// 项目没有需求文档:同样无从校验,报错
	if err := sink.ReportSpecDefect(context.Background(), "other", "REQ-1", "别处条款"); err == nil {
		t.Fatal("项目没有需求文档应报错")
	}
	// 缺条款编号:确定性拒绝
	if err := sink.ReportSpecDefect(context.Background(), "myproj", "", "无编号"); err == nil {
		t.Fatal("缺条款编号应报错")
	}
	// 以上全部不留痕
	recs, err := LoadSpecDefects(repo)
	if err != nil {
		t.Fatalf("LoadSpecDefects 出错: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("被拒提案不应留痕, got %d 条", len(recs))
	}
	if _, err := os.Stat(filepath.Join(repo, ".tally", "specdefects.md")); !os.IsNotExist(err) {
		t.Errorf("留痕文件不应被创建: %v", err)
	}
}

// 确定性幂等:同一缺陷(同条款同内容)重复提案只留一条;不同条款各留一条。
func TestReportSpecDefectIdempotent(t *testing.T) {
	repo := initRepo(t)
	writeReqDocs(t, repo)
	sink := SpecDefectSink{RepoPath: repo, Identity: testIdentity()}

	for i := 0; i < 2; i++ {
		if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-1", "异常分支未定义"); err != nil {
			t.Fatalf("第 %d 次提案出错: %v", i+1, err)
		}
	}
	if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-2", "量级未写清"); err != nil {
		t.Fatalf("REQ-2 提案出错: %v", err)
	}

	recs, err := LoadSpecDefects(repo)
	if err != nil {
		t.Fatalf("LoadSpecDefects 出错: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("提案数 = %d, 期望 2(同缺陷去重,不同缺陷各一条)", len(recs))
	}
	if recs[0].ClauseID != "REQ-1" || recs[1].ClauseID != "REQ-2" {
		t.Errorf("提案顺序不符: %s / %s", recs[0].ClauseID, recs[1].ClauseID)
	}
}

// 失效触发(INV-4,无额外机制):提案钩在提案时的条款哈希上;owner 决策
// 更新需求产生新 revision 后,同一挂钩对照新版自动「依据已变更」。
func TestReportSpecDefectInvalidationOnSpecUpdate(t *testing.T) {
	repo := initRepo(t)
	writeReqDocs(t, repo)
	sink := SpecDefectSink{RepoPath: repo, Identity: testIdentity()}

	if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-1", "异常分支未定义"); err != nil {
		t.Fatalf("ReportSpecDefect 出错: %v", err)
	}
	recs, err := LoadSpecDefects(repo)
	if err != nil || len(recs) != 1 {
		t.Fatalf("提案读回: %v, %d 条", err, len(recs))
	}

	oldDoc := Parse(reqSrc)
	if got := ClassifyReferences([]Reference{recs[0].Reference()}, oldDoc)[0].Status; got != RefValid {
		t.Errorf("提案时点对照当版应有效, got %v", got)
	}

	// owner 决策路径:specification update → 条款内容变,哈希变
	updated := strings.Replace(reqSrc,
		"WHEN 用户点击保存, 系统 shall 将草稿写入 git",
		"WHEN 用户点击保存, 系统 shall 将草稿写入 git 并在写满时拒绝", 1)
	if updated == reqSrc {
		t.Fatal("测试前置:需求正文应已改变")
	}
	newDoc := Parse(updated)
	if got := ClassifyReferences([]Reference{recs[0].Reference()}, newDoc)[0].Status; got != RefChanged {
		t.Errorf("spec 更新后旧挂钩应显示依据已变更, got %v", got)
	}
}

// 负例:缺提交身份不留痕(留痕必须可追溯,同签署的身份要求)。
func TestReportSpecDefectRequiresIdentity(t *testing.T) {
	repo := initRepo(t)
	writeReqDocs(t, repo)
	sink := SpecDefectSink{RepoPath: repo}

	if err := sink.ReportSpecDefect(context.Background(), "myproj", "REQ-1", "缺身份"); err == nil {
		t.Fatal("缺提交身份应报错")
	}
}
