package artifacts

// v13_loader 的单元测试:不连库,用 go-git 在临时目录构造真实 git 仓库,
// 验证装载快照的内容、Git 快照语义(工作区不可见、历史修订可装载)与
// 无效制品的逐文件拒收(AC1/AC2)。

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var fixtureTime = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// manifestYAML 是注册表夹具(PRD §6.4.1 的形态)。
const manifestYAML = `project: loader-proj
entities:
  - id: server
    type: service
    path: cmd/server
`

// taskYAML 生成一份合法的 standard TaskContract 夹具(通过 Parse + ValidateStatic)。
func taskYAML(id, objective string) string {
	return `id: ` + id + `
profile: standard
title: 装载器夹具
objective: ` + objective + `
product_refs:
  - .tally/product/PRD.md#loader
requirement_refs:
  - .tally/requirements/REQ-0001.md
scope:
  allowed_paths:
    - src/**
  forbidden_paths:
    - vendor/**
acceptance:
  - id: AC-01
    statement: 夹具验收
    independence_required: none
`
}

// brokenTaskYAML 缺 objective:taskcontract.Validate 必拒(AC2 的无效制品)。
const brokenTaskYAML = `id: TASK-0002
profile: light
scope:
  allowed_paths:
    - src/**
`

// verificationYAML 生成一份可冻结的 VerificationContract 夹具(P1 standard path)。
func verificationYAML(taskID string, revision int) string {
	return `task_id: ` + taskID + `
revision: ` + strconv.Itoa(revision) + `
acceptances:
  - id: AC-01
    statement: 夹具验收
    independence_required: none
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: loader-fixture-v1
        qualification:
          change_overlap: required
`
}

// brokenVerificationYAML 的 revision 为 0:verification.Validate 必拒。
const brokenVerificationYAML = `task_id: TASK-0003
revision: 0
acceptances:
  - id: AC-01
    statement: 夹具验收
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: loader-fixture-v1
`

// commitFiles 在临时目录 init 一个 git 仓库(首次调用)或复用既有仓库,
// 把 files 全部写入并提交,返回提交 sha。name 用斜杠风格仓库相对路径。
type repoFixture struct {
	dir  string
	repo *git.Repository
}

func initRepo(t *testing.T) *repoFixture {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init git 仓库: %v", err)
	}
	return &repoFixture{dir: dir, repo: repo}
}

func (rf *repoFixture) commit(t *testing.T, files map[string]string, msg string) string {
	t.Helper()
	wt, err := rf.repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(rf.dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("建目录 %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil {
			t.Fatalf("写 %s: %v", name, err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatalf("暂存 %s: %v", name, err)
		}
	}
	sig := &object.Signature{Name: "loader-test", Email: "loader@test", When: fixtureTime}
	h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, AllowEmptyCommits: true})
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	return h.String()
}

// removeFile 从工作区删除文件并提交删除(验证快照外的合同消失)。
func (rf *repoFixture) removeFile(t *testing.T, name, msg string) string {
	t.Helper()
	wt, err := rf.repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	if _, err := wt.Remove(name); err != nil {
		t.Fatalf("删除 %s: %v", name, err)
	}
	sig := &object.Signature{Name: "loader-test", Email: "loader@test", When: fixtureTime}
	h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, AllowEmptyCommits: true})
	if err != nil {
		t.Fatalf("提交删除: %v", err)
	}
	return h.String()
}

// rejectedPaths 提取拒收清单里的路径(便于集合断言)。
func rejectedPaths(snap *Snapshot) []string {
	paths := make([]string, 0, len(snap.Rejected))
	for _, r := range snap.Rejected {
		paths = append(paths, r.Path)
	}
	return paths
}

func pathsOf(tasks []TaskArtifact) []string {
	paths := make([]string, 0, len(tasks))
	for _, a := range tasks {
		paths = append(paths, a.Path)
	}
	return paths
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestLoadSnapshotAtHead:一次装载读全快照 —— 有效制品入列、无效制品逐文件
// 拒收、无关文件视而不见、身份来自注册表、来源提交可追溯(AC1/AC2)。
func TestLoadSnapshotAtHead(t *testing.T) {
	rf := initRepo(t)
	head := rf.commit(t, map[string]string{
		ManifestPath:                          manifestYAML,
		"src/main.go":                         "package main\n",
		".tally/README.md":                    "# 不是合同\n",
		".tally/tasks/notes.txt":              "不是 yaml",
		".tally/tasks/TASK-0400.yml":          "扩展名不对,不属于制品集\n",
		".tally/tasks/TASK-0001.yaml":         taskYAML("TASK-0001", "目标一"),
		".tally/tasks/TASK-0002.yaml":         brokenTaskYAML,
		".tally/verifications/TASK-0001.yaml": verificationYAML("TASK-0001", 1),
		".tally/verifications/TASK-0003.yaml": brokenVerificationYAML,
	}, "夹具提交")

	snap, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}

	if snap.ProjectID != "loader-proj" {
		t.Errorf("项目身份应来自 Git 注册表, got %q", snap.ProjectID)
	}
	if snap.SourceCommit != head {
		t.Errorf("来源提交应为 HEAD sha %s, got %s", head, snap.SourceCommit)
	}

	// 有效任务只有 TASK-0001;无关路径(src/、.tally/README.md、notes.txt、.yml)不入列。
	got := pathsOf(snap.Tasks)
	if len(got) != 1 || got[0] != ".tally/tasks/TASK-0001.yaml" {
		t.Fatalf("有效任务应只有 TASK-0001.yaml, got %v", got)
	}
	tc := snap.Tasks[0].Contract
	if tc.ID != "TASK-0001" || tc.Objective != "目标一" || tc.Profile != "standard" {
		t.Errorf("合同内容应原样装载, got %+v", tc)
	}
	if !strings.HasPrefix(tc.Hash, "sha256:") {
		t.Errorf("解析应填充内容修订哈希, got %q", tc.Hash)
	}

	// 有效验证契约:冻结修订元数据就位。
	if len(snap.Verifications) != 1 {
		t.Fatalf("有效验证契约应只有 1 份, got %d", len(snap.Verifications))
	}
	frozen := snap.Verifications[0].Frozen
	if frozen.TaskID() != "TASK-0001" || frozen.Revision() != 1 {
		t.Errorf("冻结元数据不对: task=%s rev=%d", frozen.TaskID(), frozen.Revision())
	}
	if !strings.HasPrefix(frozen.Hash(), "sha256:") {
		t.Errorf("冻结哈希应形如 sha256:<hex>, got %q", frozen.Hash())
	}

	// 无效制品逐文件拒收,理由可读。
	wantRejected := []string{
		".tally/tasks/TASK-0002.yaml",         // 缺 objective
		".tally/verifications/TASK-0003.yaml", // revision 0
	}
	gotRejected := rejectedPaths(snap)
	if len(gotRejected) != len(wantRejected) {
		t.Fatalf("拒收清单应为 %v, got %v", wantRejected, gotRejected)
	}
	for _, p := range wantRejected {
		if !containsString(gotRejected, p) {
			t.Errorf("拒收清单缺 %s, got %v", p, gotRejected)
		}
	}
	for _, r := range snap.Rejected {
		if r.Reason == "" {
			t.Errorf("%s 的拒收理由不应为空", r.Path)
		}
		if r.Kind != KindTask && r.Kind != KindVerification {
			t.Errorf("拒收应带类型, got %q", r.Kind)
		}
	}
}

// TestLoadRejectsDuplicateIDs:同一快照里同一合同 id 被多个文件声明时,冲突组
// 整体拒绝 —— 当前投影必须唯一,不按文件名猜真身(歧义由作者解决)。
func TestLoadRejectsDuplicateIDs(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		ManifestPath:                             manifestYAML,
		".tally/tasks/TASK-0001.yaml":            taskYAML("TASK-0001", "真身"),
		".tally/tasks/TASK-0001.dup.yaml":        taskYAML("TASK-0001", "重复声明"),
		".tally/verifications/TASK-0002.yaml":    verificationYAML("TASK-0002", 1),
		".tally/verifications/TASK-0002.v2.yaml": verificationYAML("TASK-0002", 2),
	}, "id 冲突夹具")

	snap, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}
	if got := pathsOf(snap.Tasks); len(got) != 0 {
		t.Errorf("id 冲突的任务应整体拒绝, got %v", got)
	}
	if len(snap.Verifications) != 0 {
		t.Errorf("task_id 冲突的验证契约应整体拒绝, got %d 份", len(snap.Verifications))
	}
	wantRejected := []string{
		".tally/tasks/TASK-0001.dup.yaml",
		".tally/tasks/TASK-0001.yaml",
		".tally/verifications/TASK-0002.v2.yaml",
		".tally/verifications/TASK-0002.yaml",
	}
	gotRejected := rejectedPaths(snap)
	if len(gotRejected) != len(wantRejected) {
		t.Fatalf("拒收清单应为 %v, got %v", wantRejected, gotRejected)
	}
	for _, p := range wantRejected {
		if !containsString(gotRejected, p) {
			t.Errorf("拒收清单缺 %s, got %v", p, gotRejected)
		}
	}
}

// TestLoadReadsGitSnapshotNotWorkingTree:Git 是唯一来源(AC1)。装载读的是
// 提交树快照:工作区未提交改动与未跟踪文件不可见;历史修订按提交可复装载。
func TestLoadReadsGitSnapshotNotWorkingTree(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "注册表")
	sha1 := rf.commit(t, map[string]string{
		".tally/tasks/TASK-0001.yaml": taskYAML("TASK-0001", "目标 v1"),
	}, "任务 v1")

	// 未提交:把工作区改成 v2,再加一个未跟踪的新合同文件。
	writeWorktreeFile(t, rf.dir, ".tally/tasks/TASK-0001.yaml", taskYAML("TASK-0001", "目标 v2 工作区未提交"))
	writeWorktreeFile(t, rf.dir, ".tally/tasks/TASK-0009.yaml", taskYAML("TASK-0009", "未跟踪文件"))

	snap, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载 HEAD: %v", err)
	}
	if got := snap.Tasks[0].Contract.Objective; got != "目标 v1" {
		t.Errorf("HEAD 装载应读提交内容 v1, 不看工作区, got %q", got)
	}
	if got := pathsOf(snap.Tasks); len(got) != 1 {
		t.Errorf("未跟踪文件不应入快照, got %v", got)
	}

	// 提交 v2 后:历史修订 sha1 仍装载 v1,HEAD 装载 v2。
	sha2 := rf.commit(t, map[string]string{
		".tally/tasks/TASK-0001.yaml": taskYAML("TASK-0001", "目标 v2"),
	}, "任务 v2")

	old, err := Load(rf.dir, sha1)
	if err != nil {
		t.Fatalf("装载历史修订: %v", err)
	}
	if got := old.Tasks[0].Contract.Objective; got != "目标 v1" {
		t.Errorf("历史修订应装载 v1, got %q", got)
	}
	if old.SourceCommit != sha1 {
		t.Errorf("历史装载的来源提交应为 %s, got %s", sha1, old.SourceCommit)
	}

	newer, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载新 HEAD: %v", err)
	}
	if got := newer.Tasks[0].Contract.Objective; got != "目标 v2" {
		t.Errorf("HEAD 应装载 v2, got %q", got)
	}
	if newer.SourceCommit != sha2 {
		t.Errorf("来源提交应为 %s, got %s", sha2, newer.SourceCommit)
	}
}

// TestLoadContractDisappearsAfterDelete:合同文件从快照删除后,装载结果里不再有它
// (重建语义的前提:装载 = 该提交快照的忠实投影,不是累积)。
func TestLoadContractDisappearsAfterDelete(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "注册表")
	rf.commit(t, map[string]string{
		".tally/tasks/TASK-0001.yaml": taskYAML("TASK-0001", "目标一"),
		".tally/tasks/TASK-0002.yaml": taskYAML("TASK-0002", "目标二"),
	}, "两个任务")
	rf.removeFile(t, ".tally/tasks/TASK-0002.yaml", "删掉任务二")

	snap, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}
	got := pathsOf(snap.Tasks)
	if len(got) != 1 || got[0] != ".tally/tasks/TASK-0001.yaml" {
		t.Errorf("删除后的快照应只剩 TASK-0001.yaml, got %v", got)
	}
}

// TestLoadMissingManifest:没有注册表就装不出项目身份 —— 基础设施级失败,返回 error
// (§33.3:身份装不出来就不能给 guard 供料)。
func TestLoadMissingManifest(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/tasks/TASK-0001.yaml": taskYAML("TASK-0001", "目标一"),
	}, "没有注册表")

	_, err := Load(rf.dir, "HEAD")
	if err == nil {
		t.Fatal("缺 .tally/project.yaml 应报错")
	}
	if !strings.Contains(err.Error(), ManifestPath) {
		t.Errorf("错误应点名注册表路径, got %v", err)
	}
}

// TestLoadInvalidManifest:注册表本身写坏(未知字段/身份空白/多文档)必须报错,
// 不静默取半截身份。
func TestLoadInvalidManifest(t *testing.T) {
	cases := map[string]string{
		"未知字段":  "project: loader-proj\nunknown_field: 1\n",
		"身份空白":  "project: \"  \"\n",
		"身份含空白": "project: loader proj\n",
		"多文档":   "project: a\n---\nproject: b\n",
	}
	for name, yamlText := range cases {
		t.Run(name, func(t *testing.T) {
			rf := initRepo(t)
			rf.commit(t, map[string]string{ManifestPath: yamlText}, "坏注册表")
			if _, err := Load(rf.dir, "HEAD"); err == nil {
				t.Fatal("坏注册表应报错")
			}
		})
	}
}

// TestLoadUnresolvableRevision:仓库打不开/revision 解析不了是装载失败,不是空快照。
func TestLoadUnresolvableRevision(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "注册表")

	if _, err := Load(rf.dir, "definitely-not-a-revision"); err == nil {
		t.Fatal("解析不了的 revision 应报错")
	}
	if _, err := Load(filepath.Join(rf.dir, "nope"), "HEAD"); err == nil {
		t.Fatal("打不开的仓库应报错")
	}
}

// writeWorktreeFile 只改工作区,不提交(测 Git 快照语义用)。
func writeWorktreeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写工作区文件 %s: %v", name, err)
	}
}
