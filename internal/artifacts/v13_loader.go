// Package artifacts 装载 Git 里的 v1.3 工程制品(PRD §6.1 Git-first Artifact)。
//
// v13_loader.go 只做一件事:打开一个 git 仓库、解析到一个提交,从**该提交的树
// 快照**(不是工作区)读取 .tally 下的 TaskContract / VerificationContract 与
// 项目注册表,全部经各自领域的确定性校验/冻结,产出一份带来源提交的装载快照,
// 供 v1.3 派生索引重建(internal/project)与后续执行链使用(PRD §33.3)。
//
// 边界(P1):
//   - 只读。任何 Git 写入都不在本包(生产 Git 写归 internal/gitwriter,DECISION_LOCKS 5);
//   - 不建 DB 索引:装载与派生入库分离,task_index/verification_index 的重建在
//     internal/project;
//   - 制品内容失效不中断整次装载:逐文件记入 Snapshot.Rejected(无效制品被拒于
//     当前索引之外,不产生任何索引行);基础设施级失败(打不开仓库、解析不了
//     revision、读不出对象、注册表缺失/非法)才返回 error —— 项目身份是
//     ghost-project guard(§33.3)的输入,身份装不出来就不能往下走。
package artifacts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v3"

	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// .tally 制品布局(PRD §6.1)。PRD 固定了项目注册表与 tasks/ 目录;
// VerificationContract 的目录 PRD 未钉死,这里采用与 tasks/ 同构的镜像约定:
// 每个任务一份 TASK-*.yaml(契约内容自带 task_id,文件名只是布局惯例,索引键以内容为准)。
const (
	// ManifestPath 项目注册表(PRD §6.4.1):声明项目身份(project: <id>)。
	ManifestPath = ".tally/project.yaml"
	// TasksDir TaskContract 制品目录(PRD §6.1 布局,.tally/tasks/TASK-*.yaml)。
	TasksDir = ".tally/tasks"
	// VerificationsDir VerificationContract 制品目录(.tally/verifications/TASK-*.yaml)。
	VerificationsDir = ".tally/verifications"

	// contractFileGlob 合同文件名形态(PRD §6.1 的 TASK-001.yaml 惯例)。
	// 只认 .yaml;布局是扁平的,子目录里的同名文件不属于制品集。
	contractFileGlob = "TASK-*.yaml"
)

// Rejection.Kind 的取值。
const (
	KindTask         = "task"
	KindVerification = "verification"
)

// TaskArtifact 是一份通过解析与静态校验的 TaskContract 制品。
// Contract.Hash 已由 taskcontract.Parse 填充(内容修订哈希,sha256:<hex>)。
type TaskArtifact struct {
	Path     string                    // Git 内制品路径(.tally/tasks/TASK-*.yaml)
	Contract taskcontract.TaskContract // 已校验合同内容;Hash 即其修订标识
}

// VerificationArtifact 是一份通过解析与 P1 standard freeze 的 VerificationContract
// 制品。Frozen 只读:Revision()/Hash() 即派生索引需要的冻结修订元数据(P1-02/15)。
type VerificationArtifact struct {
	Path   string
	Frozen *verification.Frozen
}

// Rejection 记录一份被拒于索引之外的无效制品(AC2)。Reason 是确定性校验的原始
// 错误文本,供上游呈现;装载与索引都不再使用该制品的任何内容。
type Rejection struct {
	Path   string
	Kind   string // KindTask / KindVerification
	Reason string
}

// Snapshot 是一次装载的产出:Git 某提交快照里的全部有效 v1.3 制品与被拒绝的制品。
//
// 不变量:
//   - ProjectID 来自 Git 注册表(.tally/project.yaml),不是调用方猜的;
//   - Tasks/Verifications/Rejected 各自按 Path 排序(确定性输出);
//   - Tasks 内 id 唯一、Verifications 内 task_id 唯一:同一快照里同一合同的
//     当前投影必须唯一,冲突文件整体拒绝,不猜谁是真身;
//   - 没有任何 lifecycle/status 字段:状态是证据可追溯的派生事实(INV-STATE-02),
//     装载快照只描述 Git 声明了什么。
type Snapshot struct {
	ProjectID     string
	SourceCommit  string // 装载依据的 Git 提交(完整 sha);派生索引的 source_commit 即它
	Tasks         []TaskArtifact
	Verifications []VerificationArtifact
	Rejected      []Rejection
}

// ProjectManifest 是 .tally/project.yaml 的注册表投影(PRD §6.4.1 示例的形态)。
// 解析是严格的:未知字段报错。entities 是注册表声明的一部分,本任务只消费
// project 身份,entities 原样保留为类型化数据,不做进一步解释。
type ProjectManifest struct {
	Project  string   `yaml:"project"`
	Entities []Entity `yaml:"entities,omitempty"`
}

// Entity 是注册表声明的一个 SoftwareEntity(PRD §6.4.1)。
type Entity struct {
	ID   string `yaml:"id"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

// Load 打开 repoPath 的 git 仓库,把 rev 解析到一个提交,从该提交的树快照装载
// .tally v1.3 制品。读取走 git 对象库,工作区(含未提交改动/未跟踪文件)一律
// 不可见 —— Git 是唯一来源(INV-ART-01)。rev 接受 go-git 的 revision 语法
// (完整 sha、分支名、HEAD 等)。
func Load(repoPath, rev string) (*Snapshot, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("artifacts: 打开 git 仓库 %s: %w", repoPath, err)
	}
	h, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return nil, fmt.Errorf("artifacts: 解析 revision %q: %w", rev, err)
	}
	commit, err := repo.CommitObject(*h)
	if err != nil {
		return nil, fmt.Errorf("artifacts: revision %q 不指向提交: %w", rev, err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("artifacts: 读提交 %s 的树: %w", commit.Hash.String(), err)
	}
	return loadTree(tree, commit.Hash.String())
}

// treeKind 是提交树里一个路径的制品分类。
type treeKind int

const (
	kindNone treeKind = iota
	kindManifest
	kindTask
	kindVerification
)

// classify 按 PRD §6.1 的扁平布局给路径分类:只认 .tally 下的注册表与两个
// 合同目录的直接子文件(TASK-*.yaml),其余路径与 v1.3 制品集无关。
func classify(p string) treeKind {
	if p == ManifestPath {
		return kindManifest
	}
	dir := path.Dir(p)
	base := path.Base(p)
	ok, _ := path.Match(contractFileGlob, base)
	if !ok {
		return kindNone
	}
	switch dir {
	case TasksDir:
		return kindTask
	case VerificationsDir:
		return kindVerification
	}
	return kindNone
}

// loadTree 遍历提交树,逐文件分类装载;最后做 id 唯一性裁决与确定性排序。
func loadTree(tree *object.Tree, sourceCommit string) (*Snapshot, error) {
	snap := &Snapshot{SourceCommit: sourceCommit}
	projectID := ""
	manifestSeen := false

	iter := tree.Files()
	defer iter.Close()
	for {
		f, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("artifacts: 遍历提交 %s 的文件: %w", sourceCommit, err)
		}
		contents, err := f.Contents()
		if err != nil {
			return nil, fmt.Errorf("artifacts: 读对象 %s: %w", f.Name, err)
		}
		data := []byte(contents)
		switch classify(f.Name) {
		case kindManifest:
			id, err := parseManifest(data)
			if err != nil {
				return nil, err
			}
			projectID = id
			manifestSeen = true
		case kindTask:
			a, reason := loadTask(f.Name, data)
			if reason != "" {
				snap.Rejected = append(snap.Rejected, Rejection{Path: f.Name, Kind: KindTask, Reason: reason})
				continue
			}
			snap.Tasks = append(snap.Tasks, a)
		case kindVerification:
			v, reason := loadVerification(f.Name, data)
			if reason != "" {
				snap.Rejected = append(snap.Rejected, Rejection{Path: f.Name, Kind: KindVerification, Reason: reason})
				continue
			}
			snap.Verifications = append(snap.Verifications, v)
		case kindNone:
			// 与 v1.3 制品集无关的仓库内容(src/ 等),装载器视而不见。
		}
	}
	if !manifestSeen {
		return nil, fmt.Errorf("artifacts: 提交 %s 缺少 %s(项目身份必须由 Git 注册表声明,PRD §6.4.1/§33.3)", sourceCommit, ManifestPath)
	}
	snap.ProjectID = projectID

	snap.Tasks, snap.Rejected = rejectDuplicateTasks(snap.Tasks, snap.Rejected)
	snap.Verifications, snap.Rejected = rejectDuplicateVerifications(snap.Verifications, snap.Rejected)
	sort.Slice(snap.Tasks, func(i, j int) bool { return snap.Tasks[i].Path < snap.Tasks[j].Path })
	sort.Slice(snap.Verifications, func(i, j int) bool { return snap.Verifications[i].Path < snap.Verifications[j].Path })
	sort.Slice(snap.Rejected, func(i, j int) bool { return snap.Rejected[i].Path < snap.Rejected[j].Path })
	return snap, nil
}

// loadTask 解析并静态校验一份 TaskContract(PRD §7.4 的 Tally static validation:
// 结构硬约束 + ID/refs 语法 + scope 矛盾)。失败返回拒收理由,不返回 error:
// 一份写坏的合同不该让整个项目装载不出来。
func loadTask(p string, data []byte) (TaskArtifact, string) {
	c, err := taskcontract.Parse(data)
	if err != nil {
		return TaskArtifact{}, err.Error()
	}
	if err := c.ValidateStatic(); err != nil {
		return TaskArtifact{}, err.Error()
	}
	return TaskArtifact{Path: p, Contract: c}, ""
}

// loadVerification 解析并冻结一份 VerificationContract。freeze 用 P1 standard
// path 的零值策略(AllowPostHoc=false):装载进索引的冻结修订元数据(revision/hash)
// 必须是 P1 治理语义下可冻结的契约;冻结失败的契约被拒于索引之外。
func loadVerification(p string, data []byte) (VerificationArtifact, string) {
	c, err := verification.Parse(data)
	if err != nil {
		return VerificationArtifact{}, err.Error()
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		return VerificationArtifact{}, err.Error()
	}
	return VerificationArtifact{Path: p, Frozen: frozen}, ""
}

// parseManifest 严格解析项目注册表,返回项目身份。身份是 guard 的输入:
// 缺失/空白/含空白字符都直接报错,不给 ghost project 留门。
func parseManifest(data []byte) (string, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var m ProjectManifest
	if err := dec.Decode(&m); err != nil {
		return "", fmt.Errorf("artifacts: 解析 %s: %w", ManifestPath, err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("artifacts: %s 只接受单文档 YAML", ManifestPath)
	}
	if strings.TrimSpace(m.Project) == "" {
		return "", fmt.Errorf("artifacts: %s 缺少 project 字段(项目身份必须由注册表声明,PRD §6.4.1)", ManifestPath)
	}
	if strings.ContainsAny(m.Project, " \t\r\n") {
		return "", fmt.Errorf("artifacts: %s 的 project %q 含空白字符", ManifestPath, m.Project)
	}
	return m.Project, nil
}

// rejectDuplicateTasks 裁决 task id 冲突:同一快照里同一合同 id 只允许一份当前
// 投影(0018 partial unique index 也只容一行 current)。冲突组整体拒绝并说明
// 全部涉事文件 —— 不按文件名排序猜真身,歧义由作者解决。
func rejectDuplicateTasks(tasks []TaskArtifact, rejected []Rejection) ([]TaskArtifact, []Rejection) {
	byID := map[string][]TaskArtifact{}
	for _, a := range tasks {
		byID[a.Contract.ID] = append(byID[a.Contract.ID], a)
	}
	var kept []TaskArtifact
	for _, a := range tasks {
		if len(byID[a.Contract.ID]) == 1 {
			kept = append(kept, a)
		}
	}
	for _, a := range tasks {
		if len(byID[a.Contract.ID]) > 1 {
			rejected = append(rejected, Rejection{
				Path:   a.Path,
				Kind:   KindTask,
				Reason: fmt.Sprintf("taskcontract: task id %s 由多个文件声明(%s);当前投影必须唯一,冲突文件整体拒绝", a.Contract.ID, joinPaths(taskArtifactPaths(byID[a.Contract.ID]))),
			})
		}
	}
	return kept, rejected
}

// rejectDuplicateVerifications 与 rejectDuplicateTasks 同构,按冻结契约的
// task_id 裁决。
func rejectDuplicateVerifications(verifs []VerificationArtifact, rejected []Rejection) ([]VerificationArtifact, []Rejection) {
	byID := map[string][]VerificationArtifact{}
	for _, v := range verifs {
		byID[v.Frozen.TaskID()] = append(byID[v.Frozen.TaskID()], v)
	}
	var kept []VerificationArtifact
	for _, v := range verifs {
		if len(byID[v.Frozen.TaskID()]) == 1 {
			kept = append(kept, v)
		}
	}
	for _, v := range verifs {
		if len(byID[v.Frozen.TaskID()]) > 1 {
			rejected = append(rejected, Rejection{
				Path:   v.Path,
				Kind:   KindVerification,
				Reason: fmt.Sprintf("verification: task_id %s 由多个文件声明(%s);当前投影必须唯一,冲突文件整体拒绝", v.Frozen.TaskID(), joinPaths(verificationArtifactPaths(byID[v.Frozen.TaskID()]))),
			})
		}
	}
	return kept, rejected
}

func taskArtifactPaths(tasks []TaskArtifact) []string {
	paths := make([]string, 0, len(tasks))
	for _, a := range tasks {
		paths = append(paths, a.Path)
	}
	return paths
}

func verificationArtifactPaths(verifs []VerificationArtifact) []string {
	paths := make([]string, 0, len(verifs))
	for _, v := range verifs {
		paths = append(paths, v.Path)
	}
	return paths
}

// joinPaths 把涉事文件路径排序后拼成一段可读文本(拒绝理由的一部分)。
func joinPaths(paths []string) string {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}
