// v14_policy_loader.go 把 v13_loader.go 的 Git-first 装载纪律(P1-32)扩展到
// v1.4 EngineeringPolicy(PRD §8,任务 P1-03):打开 git 仓库、解析到一个提交,
// 从**该提交的树快照**(不是工作区)读取 .tally/policies/*.yaml,经
// engineeringpolicy.Parse 的确定性解析/校验,产出带来源提交的策略快照,供
// 策略匹配(P1-05)与合同编译(P1-07)消费。
//
// 与 v13 装载相同的边界与纪律:
//   - 只读,读取走 git 对象库:工作区(含未提交改动/未跟踪文件)一律不可见,
//     Git 是唯一来源(INV-ART-01,AC1);
//   - 制品内容失效不中断整次装载:逐文件记入 PolicySnapshot.Rejected
//     (确定性拒收,AC2);基础设施级失败(打不开仓库、解析不了 revision、
//     读不出对象)才返回 error;
//   - 同一快照里同一策略 id 只允许一份当前投影:冲突组整体拒绝,不猜真身
//     (AC3)。策略 id 的集合层唯一性归加载方,正是 engineeringpolicy.Policy
//     注释里留给本文件(P1-03)的职责。
//
// 修订哈希即防篡改校验(PRD §30 desired/observed 纪律:声明与观测不混淆):
// Policy.Revision 不是制品里声明出来的,而是 engineeringpolicy.Parse 从 Git
// blob 的**实际内容**重算的内容修订哈希(schema 层面 YAML 里的 revision 字段
// 直接按未知字段拒收)。装载结果里的 revision 永远绑定本次提交的精确内容
// —— 内容被篡改必然得到不同 revision,任何「声称的 revision」都无法通过
// 装载进入下游,不存在声明/观测漂移的通道。
package artifacts

import (
	"errors"
	"fmt"
	"io"
	"path"
	"sort"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/engineeringpolicy"
)

// PoliciesDir EngineeringPolicy 制品目录(.tally/policies)。布局与 v1.3 的
// tasks/ 同构:扁平,每份策略一个 *.yaml,文件名只是布局惯例 —— 策略身份以
// 内容(id + 内容修订哈希)为准,不来自文件名。
const PoliciesDir = ".tally/policies"

// KindPolicy Rejection.Kind 的取值:被拒于装载结果之外的 EngineeringPolicy 制品。
const KindPolicy = "policy"

// PolicyArtifact 是一份通过解析与校验的 EngineeringPolicy 制品。
// Policy.Revision 已由 engineeringpolicy.Parse 从 Git blob 内容填充
// (sha256:<hex> 形式的内容修订哈希)。
type PolicyArtifact struct {
	Path   string                   // Git 内制品路径(.tally/policies/*.yaml)
	Policy engineeringpolicy.Policy // 已校验策略内容;Revision 即其修订标识
}

// PolicySnapshot 是一次策略装载的产出:Git 某提交快照里的全部有效 EngineeringPolicy
// 与被拒绝的制品。不变量:
//   - SourceCommit 是装载依据的 Git 提交(完整 sha),结果可追溯到快照;
//   - Policies/Rejected 各自按 Path 排序(确定性输出);
//   - Policies 内 id 唯一:同一快照里同一策略的当前投影必须唯一,冲突文件
//     整体拒绝,不猜谁是真身(AC3)。
type PolicySnapshot struct {
	SourceCommit string // 装载依据的 Git 提交(完整 sha)
	Policies     []PolicyArtifact
	Rejected     []Rejection
}

// LoadPolicies 打开 repoPath 的 git 仓库,把 rev 解析到一个提交,从该提交的树
// 快照装载 .tally/policies/*.yaml。与 v13 Load 同一 revision 语法(完整 sha、
// 分支名、HEAD 等);项目身份仍由 v13 装载路径的注册表负责,本入口只裁决策略
// 制品本身。
func LoadPolicies(repoPath, rev string) (*PolicySnapshot, error) {
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
	return loadPolicyTree(tree, commit.Hash.String())
}

// isPolicyArtifactPath 按 .tally/policies 的扁平布局给路径分类:只有该目录的
// 直接子 *.yaml 文件属于策略制品集;其余路径(子目录里的文件、.yml、非 YAML)
// 不属于制品集,装载器视而不见 —— 与 v13 classify 同一纪律。
func isPolicyArtifactPath(p string) bool {
	if path.Dir(p) != PoliciesDir {
		return false
	}
	ok, _ := path.Match("*.yaml", path.Base(p))
	return ok
}

// loadPolicyTree 遍历提交树,逐文件装载策略;最后做 id 唯一性裁决与确定性排序。
func loadPolicyTree(tree *object.Tree, sourceCommit string) (*PolicySnapshot, error) {
	snap := &PolicySnapshot{SourceCommit: sourceCommit}

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
		if !isPolicyArtifactPath(f.Name) {
			continue
		}
		contents, err := f.Contents()
		if err != nil {
			return nil, fmt.Errorf("artifacts: 读对象 %s: %w", f.Name, err)
		}
		p, err := engineeringpolicy.Parse([]byte(contents))
		if err != nil {
			// 一份写坏的策略不该让整个策略集装载不出来:逐文件拒收,
			// Reason 是确定性校验链的原始错误文本(含 engineeringpolicy
			// 的哨兵错误文案,errors.Is 语义在 Parse 处保留)。
			snap.Rejected = append(snap.Rejected, Rejection{Path: f.Name, Kind: KindPolicy, Reason: err.Error()})
			continue
		}
		snap.Policies = append(snap.Policies, PolicyArtifact{Path: f.Name, Policy: p})
	}

	snap.Policies, snap.Rejected = rejectDuplicatePolicies(snap.Policies, snap.Rejected)
	sort.Slice(snap.Policies, func(i, j int) bool { return snap.Policies[i].Path < snap.Policies[j].Path })
	sort.Slice(snap.Rejected, func(i, j int) bool { return snap.Rejected[i].Path < snap.Rejected[j].Path })
	return snap, nil
}

// rejectDuplicatePolicies 裁决策略 id 冲突(AC3):同一快照里同一策略 id 只允许
// 一份当前投影。冲突组整体拒绝并说明全部涉事文件 —— 不按文件名排序猜真身,
// 歧义由作者解决;即使两份内容完全相同(同 Revision)也照拒:重复声明本身是
// 布局错误,静默去重会掩盖真实来源。
func rejectDuplicatePolicies(policies []PolicyArtifact, rejected []Rejection) ([]PolicyArtifact, []Rejection) {
	byID := map[string][]PolicyArtifact{}
	for _, a := range policies {
		byID[a.Policy.ID] = append(byID[a.Policy.ID], a)
	}
	var kept []PolicyArtifact
	for _, a := range policies {
		if len(byID[a.Policy.ID]) == 1 {
			kept = append(kept, a)
		}
	}
	for _, a := range policies {
		if len(byID[a.Policy.ID]) > 1 {
			rejected = append(rejected, Rejection{
				Path:   a.Path,
				Kind:   KindPolicy,
				Reason: fmt.Sprintf("engineeringpolicy: 策略 id %s 由多个文件声明(%s);当前投影必须唯一,冲突文件整体拒绝", a.Policy.ID, joinPaths(policyArtifactPaths(byID[a.Policy.ID]))),
			})
		}
	}
	return kept, rejected
}

func policyArtifactPaths(policies []PolicyArtifact) []string {
	paths := make([]string, 0, len(policies))
	for _, a := range policies {
		paths = append(paths, a.Path)
	}
	return paths
}
