// v141_bundle_loader.go 是 v1.4.1 P1-01 的**同源执行 bundle 装载器**(DL-04:
// 一次任务的全部权威制品必须解析自同一个 pinned source commit):打开 git
// 仓库、把 rev 解析到**一个**提交,从该提交的树快照(不是工作区)一次性装载
// 同一任务身份的全部执行权威 —— FrozenIntent(.tally/intents/<task_id>.yaml)、
// TaskContract(.tally/tasks/<task_id>.yaml)、冻结 VerificationContract
// (.tally/verifications/<task_id>.yaml)与 EngineeringPolicies
// (.tally/policies/*.yaml)。
//
// 为什么不是组合既有入口(LoadIntent/LoadPolicies):入口各自解析 revision,
// 对分支名/HEAD 这类可动 revision,两次解析可能跨并发推进落在不同提交上 ——
// 那是混合来源。bundle 装载把「解析」收敛为一次,四类制品全部从同一棵提交树
// 读取,混合来源在构造上不可能;各制品的解析/校验/冻结纪律与既有装载器逐字
// 共享(loadIntentTree / loadPolicyTree,本文件与 v141_intent_loader.go /
// v14_policy_loader.go 同包同源,不存在第二套语义)。
//
// 裁决纪律(与既有装载器一致):
//   - 只读,读取走 git 对象库:工作区(含未提交改动/未跟踪文件)一律不可见
//     (INV-ART-01);
//   - pinned 提交里缺哪份制品就是哪份的类型化不存在(ErrIntentNotFound /
//     ErrTaskNotFound / ErrVerificationNotFound)—— 绝不自动升级到 HEAD 或
//     任何其他来源补读(混合提交是硬错误);任务/验证制品内容声明的身份与
//     请求身份不一致按身份漂移 fail-closed 拒收(单制品装载没有同侪可对照);
//   - 策略制品沿用 LoadPolicies 的快照纪律:无效策略逐文件记入
//     PolicySnapshot.Rejected(拒于运行集之外,不中断装载),同一 id 多文件
//     冲突整体拒绝 —— Runtime-active 策略就是 pinned 快照里的精确策略内容
//     (DL-09/BI-01),bundle 原样携带这份裁决,由消费方(P1-03 装配器)决定
//     拒收是否阻断;
//   - 装载顺序 intent → task → verification → policies 固定:缺多份时报告
//     的那份是确定的。
//
// bundle 身份(任务 AC):ExecutionBundle.Identity 是「一个来源提交 + 全部
// 装载修订」的确定性纯函数(sha256:<hex>) —— 同一提交重复装载得到同一身份;
// 任一制品换版或换提交,身份必变。修订一律从 Git blob 实际内容重算
// (intent.Revision / taskcontract.RevisionHash / verification 冻结哈希 /
// policy.Revision),不存在「声明的修订」进入 bundle 的通道(PRD §30
// desired/observed 纪律)。
package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"gopkg.in/yaml.v3"

	"tally/internal/intentevidence"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// task/verification 制品级失效的类型化哨兵(与 v141_intent_loader.go 的
// intent 三哨兵同构):调用方用 errors.Is 判类,不匹配错误文本。
var (
	// ErrTaskNotFound:请求身份对应的 TaskContract 制品不在 pinned 提交的树
	// 快照里(从未冻结、或该提交早于/晚于冻结提交)。
	ErrTaskNotFound = errors.New("artifacts: task 制品不存在")
	// ErrTaskInvalid:制品在提交树里,但未通过 taskcontract.Parse + 静态校验
	// (与 v13 装载同一裁决链)。
	ErrTaskInvalid = errors.New("artifacts: task 制品非法")
	// ErrTaskIdentityMismatch:制品内容声明的 id 与请求身份(即制品路径派生
	// 身份)不一致 —— 身份漂移按 fail-closed 拒收,不猜谁是真身。
	ErrTaskIdentityMismatch = errors.New("artifacts: task 制品身份不一致")
	// ErrVerificationNotFound / ErrVerificationInvalid /
	// ErrVerificationIdentityMismatch:冻结 VerificationContract 制品的对应
	// 三类裁决(解析、P1 standard 冻结准入、身份一致性)。
	ErrVerificationNotFound         = errors.New("artifacts: verification 制品不存在")
	ErrVerificationInvalid          = errors.New("artifacts: verification 制品非法")
	ErrVerificationIdentityMismatch = errors.New("artifacts: verification 制品身份不一致")
)

// ExecutionBundle 是一次同源装载的产出:pinned 提交树快照里,同一任务身份的
// 全部执行权威制品。不变量:
//   - SourceCommit 是**观测到的**唯一装载来源(完整 sha),四类制品全部解析
//     自这同一棵提交树(DL-04);不是调用方传入的 revision 表达式;
//   - Intent/Task/Verification 共享请求身份,且各自内容声明的身份已逐一对照
//     (intent 由 loadIntentTree、task/verification 由本文件裁决);
//   - Policies 是同一提交的策略快照(loadPolicyTree 原样裁决,含 Rejected);
//   - 同一提交重复装载产出逐字段相等、Identity 相同的 bundle(确定性)。
type ExecutionBundle struct {
	// TaskID 是 bundle 的请求/内容共同任务身份(已折叠首尾空白)。
	TaskID string
	// SourceCommit 是装载依据的唯一 Git 提交(完整 sha)。
	SourceCommit string
	// Intent 是 FrozenIntent 制品(含制品路径与来源提交)。
	Intent *IntentArtifact
	// Task 是 TaskContract 制品(Hash 已由 taskcontract.Parse 从内容填充)。
	Task TaskArtifact
	// Verification 是冻结 VerificationContract 制品(P1 standard path)。
	Verification VerificationArtifact
	// Policies 是同一提交的 EngineeringPolicy 快照(无效策略逐文件 Rejected)。
	Policies PolicySnapshot
	// ProjectID 是同一棵提交树里项目注册表(.tally/project.yaml)声明的项目
	// 身份(P1-04 增补的运行事实坐标:worker 的 codegen_run 事实需要项目身份,
	// 读取与四类制品同走这一棵提交树,绝不二次解析 revision)。注册表缺席时
	// 为空串 —— 装载器保持任务作用域:四类权威制品的既有裁决序(intent →
	// task → verification → policies)不因注册表改变;空身份由消费方(worker)
	// 在执行前 fail-closed,与 v1.3 全仓装载器「缺注册表整个装载失败」同一
	// 裁决强度,只是把触发点放在执行入口。
	ProjectID string

	identity string
}

// Revisions 返回 bundle 装载的全部内容修订标识(确定性升序):intent 修订、
// task 合同修订哈希、冻结验证契约哈希与 revision、每条已装载策略的内容修订。
// 修订都从 Git blob 实际内容重算(装载器没有读「声明修订」的通道)。
func (b *ExecutionBundle) Revisions() []string {
	revs := []string{
		"intent:" + b.Intent.Intent.Revision(),
		"task:" + b.Task.Contract.RevisionHash(),
		"verification:" + b.Verification.Frozen.Hash(),
		fmt.Sprintf("verification_revision:%d", b.Verification.Frozen.Revision()),
	}
	for _, a := range b.Policies.Policies {
		revs = append(revs, "policy:"+a.Policy.Revision)
	}
	sort.Strings(revs)
	return revs
}

// bundleCanonicalVersion 是 bundle 身份规范文本的版本前缀(与 intent/v1、
// taskcontract/v1、contract/compiled/v1 同一纪律):规范格式变化时递增,
// 避免哈希空间重叠。
const bundleCanonicalVersion = "artifacts/bundle/v1"

// canonicalBundle 是 bundle 身份哈希的规范投影:任务身份 + 唯一来源提交 +
// 全部装载修订(Revisions 已升序)。
type canonicalBundle struct {
	Version      string   `yaml:"version"`
	TaskID       string   `yaml:"task_id"`
	SourceCommit string   `yaml:"source_commit"`
	Revisions    []string `yaml:"revisions"`
}

// Identity 返回 bundle 身份(sha256:<hex>):「一个来源提交 + 全部装载修订 +
// 任务身份」的确定性纯函数。同一提交重复装载得到同一身份;任一制品换版或
// 来源提交变化,身份必变 —— 供下游(P1-03 装配 / P1-04 运行事实)把执行
// 钉到这份同源权威上。
func (b *ExecutionBundle) Identity() string { return b.identity }

// LoadBundle 打开 repoPath 的 git 仓库,把 rev 解析到**一个**提交,从该提交
// 的树快照同源装载请求身份的 FrozenIntent、TaskContract、冻结
// VerificationContract 与 EngineeringPolicies。与 Load / LoadIntent /
// LoadPolicies 同一 revision 语法(完整 sha、分支名、HEAD 等)。
//
// 错误分类:仓库/revision/对象读取失败是基础设施 error;任一制品不存在 /
// 内容非法 / 身份不一致,分别返回该制品类的类型化哨兵(intent 见
// v141_intent_loader.go,task/verification 见本文件)。装载顺序 intent →
// task → verification → policies 固定,缺多份时报告的那份是确定的。
func LoadBundle(repoPath, rev, taskID string) (*ExecutionBundle, error) {
	// 请求身份先做与冻结 schema 相同的排版折叠,再派生制品路径;身份本身
	// 非法(空、含内部空白)是 intentevidence 的类型化错误,原样穿透。
	wantID := strings.TrimSpace(taskID)
	if _, err := intentevidence.ArtifactPath(wantID); err != nil {
		return nil, fmt.Errorf("artifacts: %w", err)
	}
	tree, sourceCommit, err := openCommitTree(repoPath, rev)
	if err != nil {
		return nil, err
	}
	return loadBundleTree(tree, sourceCommit, wantID)
}

// openCommitTree 把 (repoPath, rev) 解析到唯一提交的树快照,返回树与观测
// 提交 sha。基础设施失败(打不开仓库、解析不了 revision、对象读不出)返回
// 普通 error —— 与既有装载器同一分类。
func openCommitTree(repoPath, rev string) (*object.Tree, string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, "", fmt.Errorf("artifacts: 打开 git 仓库 %s: %w", repoPath, err)
	}
	h, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return nil, "", fmt.Errorf("artifacts: 解析 revision %q: %w", rev, err)
	}
	commit, err := repo.CommitObject(*h)
	if err != nil {
		return nil, "", fmt.Errorf("artifacts: revision %q 不指向提交: %w", rev, err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, "", fmt.Errorf("artifacts: 读提交 %s 的树: %w", commit.Hash.String(), err)
	}
	return tree, commit.Hash.String(), nil
}

// loadBundleTree 从同一棵提交树按固定顺序装载四类制品,最后计算 bundle 身份。
// 任一 intent/task/verification 制品级失效都让整次装载失败(类型化哨兵):
// 执行权威要么完整同源、要么不存在,不存在「缺一角也能跑」的 bundle。
func loadBundleTree(tree *object.Tree, sourceCommit, wantID string) (*ExecutionBundle, error) {
	intent, err := loadIntentTree(tree, sourceCommit, wantID)
	if err != nil {
		return nil, err
	}
	task, err := loadTaskTree(tree, sourceCommit, wantID)
	if err != nil {
		return nil, err
	}
	verif, err := loadVerificationTree(tree, sourceCommit, wantID)
	if err != nil {
		return nil, err
	}
	policies, err := loadPolicyTree(tree, sourceCommit)
	if err != nil {
		return nil, err
	}
	// 项目注册表是同一棵提交树上的坐标(P1-04),排在四类权威制品之后读取:
	// 缺 task/verification 等既有类型化裁决序不变;注册表缺席不改变 bundle
	// 的权威内容集(空身份由 worker 执行前 fail-closed),注册表存在但非法
	// 则按 parseManifest 的严格裁决整体失败 —— 不给 ghost project 留门。
	projectID := ""
	if f, err := tree.File(ManifestPath); err == nil {
		contents, err := f.Contents()
		if err != nil {
			return nil, fmt.Errorf("artifacts: 读对象 %s: %w", ManifestPath, err)
		}
		id, err := parseManifest([]byte(contents))
		if err != nil {
			return nil, err
		}
		projectID = id
	} else if !errors.Is(err, object.ErrFileNotFound) {
		return nil, fmt.Errorf("artifacts: 读提交 %s 的对象 %s: %w", sourceCommit, ManifestPath, err)
	}
	b := &ExecutionBundle{
		TaskID:       wantID,
		SourceCommit: sourceCommit,
		Intent:       intent,
		Task:         task,
		Verification: verif,
		Policies:     *policies,
		ProjectID:    projectID,
	}
	id, err := bundleIdentity(b)
	if err != nil {
		return nil, err
	}
	b.identity = id
	return b, nil
}

// loadTaskTree 从提交树装载请求身份的 TaskContract 制品:存在性(派生路径
// .tally/tasks/<task_id>.yaml)→ taskcontract.Parse + ValidateStatic(与 v13
// 装载同一裁决链)→ 内容 id 与请求身份精确一致。身份以内容为准,文件名只是
// 布局惯例;单制品装载没有同侪可对照,任何身份漂移 fail-closed。
func loadTaskTree(tree *object.Tree, sourceCommit, wantID string) (TaskArtifact, error) {
	p := TasksDir + "/" + wantID + ".yaml"
	f, err := tree.File(p)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return TaskArtifact{}, fmt.Errorf("%w: %s 不在提交 %s 的树快照里", ErrTaskNotFound, p, sourceCommit)
		}
		return TaskArtifact{}, fmt.Errorf("artifacts: 读提交 %s 的对象 %s: %w", sourceCommit, p, err)
	}
	contents, err := f.Contents()
	if err != nil {
		return TaskArtifact{}, fmt.Errorf("artifacts: 读对象 %s: %w", p, err)
	}
	c, err := taskcontract.Parse([]byte(contents))
	if err != nil {
		return TaskArtifact{}, fmt.Errorf("%w: %s(提交 %s): %w", ErrTaskInvalid, p, sourceCommit, err)
	}
	if err := c.ValidateStatic(); err != nil {
		return TaskArtifact{}, fmt.Errorf("%w: %s(提交 %s): %w", ErrTaskInvalid, p, sourceCommit, err)
	}
	if c.ID != wantID {
		return TaskArtifact{}, fmt.Errorf("%w: %s 声明 id %q,请求身份为 %q(提交 %s)", ErrTaskIdentityMismatch, p, c.ID, wantID, sourceCommit)
	}
	return TaskArtifact{Path: p, Contract: c}, nil
}

// loadVerificationTree 从提交树装载请求身份的冻结 VerificationContract 制品:
// 存在性(派生路径 .tally/verifications/<task_id>.yaml)→ verification.Parse +
// FreezeContract(P1 standard path 零值策略,与 v13 装载同一裁决:装载进执行
// 权威的验证契约必须可冻结)→ 冻结契约 task_id 与请求身份精确一致。
func loadVerificationTree(tree *object.Tree, sourceCommit, wantID string) (VerificationArtifact, error) {
	p := VerificationsDir + "/" + wantID + ".yaml"
	f, err := tree.File(p)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return VerificationArtifact{}, fmt.Errorf("%w: %s 不在提交 %s 的树快照里", ErrVerificationNotFound, p, sourceCommit)
		}
		return VerificationArtifact{}, fmt.Errorf("artifacts: 读提交 %s 的对象 %s: %w", sourceCommit, p, err)
	}
	contents, err := f.Contents()
	if err != nil {
		return VerificationArtifact{}, fmt.Errorf("artifacts: 读对象 %s: %w", p, err)
	}
	c, err := verification.Parse([]byte(contents))
	if err != nil {
		return VerificationArtifact{}, fmt.Errorf("%w: %s(提交 %s): %w", ErrVerificationInvalid, p, sourceCommit, err)
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		return VerificationArtifact{}, fmt.Errorf("%w: %s(提交 %s): %w", ErrVerificationInvalid, p, sourceCommit, err)
	}
	if frozen.TaskID() != wantID {
		return VerificationArtifact{}, fmt.Errorf("%w: %s 声明 task_id %q,请求身份为 %q(提交 %s)", ErrVerificationIdentityMismatch, p, frozen.TaskID(), wantID, sourceCommit)
	}
	return VerificationArtifact{Path: p, Frozen: frozen}, nil
}

// bundleIdentity 计算 bundle 身份哈希(sha256:<hex>)。规范文本只含任务身份、
// 唯一来源提交与全部装载修订 —— 全部是内容派生的观测事实。
func bundleIdentity(b *ExecutionBundle) (string, error) {
	data, err := yaml.Marshal(&canonicalBundle{
		Version:      bundleCanonicalVersion,
		TaskID:       b.TaskID,
		SourceCommit: b.SourceCommit,
		Revisions:    b.Revisions(),
	})
	if err != nil {
		return "", fmt.Errorf("artifacts: bundle 身份规范化序列化: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
