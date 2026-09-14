// v141_intent_loader.go 把 v13_loader.go / v14_policy_loader.go 的 Git-first
// 装载纪律(P1-32 / P1-03)扩展到 v1.4.1 的 FrozenIntent 证据(A-02 / DL-06,
// 任务 P0-02):打开 git 仓库、解析到一个提交,从**该提交的树快照**(不是
// 工作区)读取 .tally/intents/<task_id>.yaml,经 intentevidence.Parse 的
// 确定性解析/fail-closed 校验,产出绑定来源提交的冻结意图 —— 供后续风险
// 推导(P0-01 RiskSignalCandidates)与 confirm→freeze 事务(P0-08)在
// pinned 提交上确定性复装载(DL-04:一次任务的全部权威制品解析自同一个
// pinned source commit)。
//
// 制品路径与 schema 归 P0-01 裁决(seam owner):精确路径由
// intentevidence.ArtifactPath 派生(本文件不散落路径字面量),schema 校验
// 全部在 intentevidence.Parse(严格 KnownFields:未知字段、声明出来的
// revision、provider/model prose 一律 fail-closed 拒收)。本装载器只新增
// 两类裁决:
//   - 存在性:请求身份对应的制品不在提交树里 → ErrIntentNotFound;
//   - 身份一致性:制品内容声明的 task_id 必须与请求/路径身份精确一致
//     (身份以内容为准,文件名只是布局惯例 —— 与 policies 同一纪律),
//     不一致 → ErrIntentIdentityMismatch。快照式装载器靠「id 冲突整体
//     拒绝」裁决歧义;单制品装载没有同侪可对照,任何身份漂移都直接拒收。
//
// 与 v13/v14 装载相同的边界与纪律:
//   - 只读,读取走 git 对象库:工作区(含未提交改动/未跟踪文件)一律
//     不可见,Git 是唯一来源(INV-ART-01);
//   - 错误分类与既有装载器一致:基础设施级失败(打不开仓库、解析不了
//     revision、读不出对象)返回普通 error;制品级失效(不存在、内容
//     非法、身份不一致)返回本文件的类型化哨兵错误,调用方以 errors.Is
//     判类 —— 解析链的 intentevidence 哨兵(ErrInvalidEvidence /
//     ErrUnsupportedTaskType)以 %w 穿透,两层判类都成立。
package artifacts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/intentevidence"
)

// 制品级失效的类型化哨兵:调用方用 errors.Is 判类,不匹配错误文本。
var (
	// ErrIntentNotFound:请求身份对应的 intent 制品不在 pinned 提交的树快照里
	// (从未冻结、或该提交早于/晚于冻结提交)。
	ErrIntentNotFound = errors.New("artifacts: intent 制品不存在")
	// ErrIntentInvalid:制品在提交树里,但内容未通过 intentevidence.Parse 的
	// 确定性校验(YAML 解析失败、schema 外未知字段、词表外 task_type 等)。
	ErrIntentInvalid = errors.New("artifacts: intent 制品非法")
	// ErrIntentIdentityMismatch:制品内容声明的 task_id 与请求身份(即制品
	// 路径派生身份)不一致 —— 身份漂移是布局错误,按 fail-closed 拒收,
	// 不猜测谁是真身。
	ErrIntentIdentityMismatch = errors.New("artifacts: intent 制品身份不一致")
)

// IntentArtifact 是一份从 pinned Git 提交树快照装载并通过存在性/解析/身份
// 三重裁决的 FrozenIntent 制品。Intent.Revision 已由 intentevidence.Parse
// 从 Git blob 的实际内容计算(sha256:<hex> 形式的内容修订哈希),不存在
// 声明/观测漂移的通道。
type IntentArtifact struct {
	// Path 是 Git 内制品路径(.tally/intents/<task_id>.yaml),由请求身份
	// 经 intentevidence.ArtifactPath 派生。
	Path string
	// SourceCommit 是装载依据的 Git 提交(完整 sha)。同一次任务的全部
	// 权威制品必须解析自同一个提交(DL-04);这是**观测到的**提交身份,
	// 不是调用方传入的 revision 表达式。
	SourceCommit string
	// Intent 是冻结意图证据(不可变值;Revision 即其修订标识)。
	Intent intentevidence.FrozenIntent
}

// LoadIntent 打开 repoPath 的 git 仓库,把 rev 解析到一个提交,从该提交的
// 树快照装载请求身份对应的 intent 制品(.tally/intents/<task_id>.yaml)。
// 与 Load / LoadPolicies 同一 revision 语法(完整 sha、分支名、HEAD 等);
// 同一提交重复装载产出逐字段相等的结果(确定性)。
//
// 错误分类(见包注释):仓库/revision/对象读取失败返回基础设施 error;
// 制品不存在、内容非法、身份不一致分别返回 ErrIntentNotFound /
// ErrIntentInvalid(携带 intentevidence 类型化哨兵)/
// ErrIntentIdentityMismatch。
func LoadIntent(repoPath, rev, taskID string) (*IntentArtifact, error) {
	// 请求身份先做与冻结 schema 相同的排版折叠(首尾空白),再派生制品路径;
	// 身份本身非法(空、含内部空白)是 intentevidence 的类型化错误,原样穿透。
	wantID := strings.TrimSpace(taskID)
	if _, err := intentevidence.ArtifactPath(wantID); err != nil {
		return nil, fmt.Errorf("artifacts: %w", err)
	}
	tree, sourceCommit, err := openCommitTree(repoPath, rev)
	if err != nil {
		return nil, err
	}
	return loadIntentTree(tree, sourceCommit, wantID)
}

// loadIntentTree 从一棵提交树快照装载请求身份的 intent 制品。LoadIntent 与
// bundle 装载(v141_bundle_loader.go)共享本裁决,两个入口不存在语义漂移:
// 存在性 / intentevidence.Parse 校验 / 内容 task_id 与请求身份精确一致。
func loadIntentTree(tree *object.Tree, sourceCommit, wantID string) (*IntentArtifact, error) {
	p, err := intentevidence.ArtifactPath(wantID)
	if err != nil {
		return nil, fmt.Errorf("artifacts: %w", err)
	}

	f, err := tree.File(p)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return nil, fmt.Errorf("%w: %s 不在提交 %s 的树快照里", ErrIntentNotFound, p, sourceCommit)
		}
		return nil, fmt.Errorf("artifacts: 读提交 %s 的对象 %s: %w", sourceCommit, p, err)
	}
	contents, err := f.Contents()
	if err != nil {
		return nil, fmt.Errorf("artifacts: 读对象 %s: %w", p, err)
	}

	fi, err := intentevidence.Parse([]byte(contents))
	if err != nil {
		// 双重 %w:既可按装载层 ErrIntentInvalid 判类,也可穿透到
		// intentevidence 的类型化哨兵(ErrInvalidEvidence /
		// ErrUnsupportedTaskType)细分失效原因。
		return nil, fmt.Errorf("%w: %s(提交 %s): %w", ErrIntentInvalid, p, sourceCommit, err)
	}
	if fi.TaskID() != wantID {
		return nil, fmt.Errorf("%w: %s 声明 task_id %q,请求身份为 %q(提交 %s)", ErrIntentIdentityMismatch, p, fi.TaskID(), wantID, sourceCommit)
	}
	return &IntentArtifact{Path: p, SourceCommit: sourceCommit, Intent: fi}, nil
}
