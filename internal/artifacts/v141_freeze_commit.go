// v141_freeze_commit.go 是 v141_intent_loader.go 的**写侧**对应物(v1.4.1
// P0-08):把一次执行就绪冻结的完整产物(intakefreeze.FrozenArtifacts:
// FrozenIntent 意图证据 + TaskContract + 冻结 VerificationContract,共享同一
// 任务身份)作为**一个**原子 Git 事务写进制品仓库 —— 用户确认(P0-04 表单)
// → 冻结准入(P0-07 Freeze)之后、「提交后可经 LoadIntent 从 pinned 提交读回」
// 之前的落库跳。
//
// 事务语义(任务合同实现要求 1/2/3):
//   - 单一期望基线 CAS 边界:整个三件套只经**一次** gitwriter.Writer.Commit
//     落库,ExpectedBase 由调用方显式给定(读基线与提交之间 HEAD 被并发推进
//     时,gitwriter 返回类型化 ErrBaseRevision —— 冲突而不是静默改基,AC1)。
//   - 同一提交语义:三份制品路径全部由同一任务身份派生,且只出现在这一个
//     提交里;不存在「部分冻结」的快照状态(DL-04 的写侧对应)。
//   - 失败不留半套制品:三份文件先在内存里全部渲染成功,才交给 gitwriter;
//     gitwriter 在锁内原子应用文件并在任何失败路径回滚工作区与索引
//     (writer.go 包注释的契约)。本文件不引入第二条 Git 写路径(DL-03:
//     internal/gitwriter 是唯一写边界)。
//
// 错误分类:提交阶段失败原样透传 gitwriter 的类型化哨兵 —— ErrBaseRevision
// (基线冲突)、ErrNoChanges(内容与 HEAD 一致,重复确认的幂等语义,不产生
// 新提交);本包只新增一个制品级哨兵 ErrFreezeIdentityMismatch(三件套任务
// 身份不一致,拒绝装配任何提交)。
package artifacts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"tally/internal/gitwriter"
	"tally/internal/intakefreeze"
	"tally/internal/intentevidence"
)

// ErrFreezeIdentityMismatch:冻结三件套的任务身份不一致(FrozenIntent /
// TaskContract / VerificationContract 必须共享同一 task_id)。intakefreeze.Freeze
// 的产物按构造保证一致;本包对任何手工装配的值同样 fail-closed,绝不猜谁是真身。
var ErrFreezeIdentityMismatch = errors.New("artifacts: 冻结三件套任务身份不一致")

// FreezeCommitRequest 是一次冻结提交的输入。ExpectedBase 是调用方作者化时
// 依据的 HEAD(gitwriter.Writer.Head 读取);空串断言仓库还没有任何提交。
type FreezeCommitRequest struct {
	// ExpectedBase 是本次提交的期望基线(单一 CAS 边界,见包注释)。
	ExpectedBase string
	// Artifacts 是冻结准入(P0-07 intakefreeze.Freeze)的完整产物。
	Artifacts intakefreeze.FrozenArtifacts
	// Message 是提交信息(gitwriter 要求非空)。
	Message string
	// Signature 覆盖 gitwriter 的默认提交身份;nil 用默认。
	Signature *gitwriter.Signature
}

// FreezeCommit 把冻结三件套渲染为三份 .tally 制品文件,并经 internal/gitwriter
// 以单一期望基线原子提交。成功返回提交结果;基线冲突返回(包住)
// gitwriter.ErrBaseRevision,内容与 HEAD 一致(重复确认)返回
// gitwriter.ErrNoChanges —— 两者都绝无部分写入。
func FreezeCommit(ctx context.Context, repoPath string, req FreezeCommitRequest) (gitwriter.CommitResult, error) {
	files, err := freezeCommitFiles(req.Artifacts)
	if err != nil {
		return gitwriter.CommitResult{}, err
	}
	w, err := gitwriter.New(repoPath)
	if err != nil {
		return gitwriter.CommitResult{}, fmt.Errorf("artifacts: 定位制品仓库: %w", err)
	}
	// 一次提交 = 一次基线校验 = 一个不可再分的冻结快照。gitwriter 的类型化
	// 哨兵(ErrBaseRevision / ErrNoChanges / ErrNotRepository)原样透传,
	// 调用方以 errors.Is 判类。
	return w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: req.ExpectedBase,
		Message:      req.Message,
		Files:        files,
		Signature:    req.Signature,
	})
}

// freezeCommitFiles 渲染三份制品的 FileUpdate:先做任务身份一致性门,再把
// 三份内容**全部**渲染成功后才返回 —— 任何渲染失败都不会触碰仓库。
func freezeCommitFiles(fa intakefreeze.FrozenArtifacts) ([]gitwriter.FileUpdate, error) {
	taskID := strings.TrimSpace(fa.Task.ID)
	switch {
	case taskID == "":
		return nil, fmt.Errorf("%w: TaskContract 缺少 id", ErrFreezeIdentityMismatch)
	case fa.Intent.TaskID() != taskID:
		return nil, fmt.Errorf("%w: intent 声明 %q,task 声明 %q", ErrFreezeIdentityMismatch, fa.Intent.TaskID(), taskID)
	case fa.Verification.TaskID() != taskID:
		return nil, fmt.Errorf("%w: verification 声明 %q,task 声明 %q", ErrFreezeIdentityMismatch, fa.Verification.TaskID(), taskID)
	}

	intentPath, err := intentevidence.ArtifactPath(fa.Intent.TaskID())
	if err != nil {
		return nil, fmt.Errorf("artifacts: 派生 intent 制品路径: %w", err)
	}
	intentBytes, err := fa.Intent.Render()
	if err != nil {
		return nil, fmt.Errorf("artifacts: 渲染 intent 制品: %w", err)
	}

	// TaskContract:yaml.Marshal 与 taskcontract.Parse 严格互逆(Hash 字段
	// yaml:"-" 不落盘,解析侧从内容重算);确定性金测试钉住往返。
	taskBytes, err := yaml.Marshal(fa.Task)
	if err != nil {
		return nil, fmt.Errorf("artifacts: 渲染 task 制品: %w", err)
	}

	// VerificationContract:Frozen.Contract() 是深拷贝;yaml.Marshal 即该域
	// 哈希的同一规范形态(hash.go),verification.Parse 严格互逆。
	contract := fa.Verification.Contract()
	verifBytes, err := yaml.Marshal(&contract)
	if err != nil {
		return nil, fmt.Errorf("artifacts: 渲染 verification 制品: %w", err)
	}

	return []gitwriter.FileUpdate{
		{Path: intentPath, Content: intentBytes},
		{Path: TasksDir + "/" + taskID + ".yaml", Content: taskBytes},
		{Path: VerificationsDir + "/" + taskID + ".yaml", Content: verifBytes},
	}, nil
}
