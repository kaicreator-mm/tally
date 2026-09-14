// Package gitintegration 在既有 Git 权威(gitwriter 单一 Git 写边界 +
// changeset 不可变候选快照)之上提供受控的本地集成语义层
// (PRD v1.4.1 §"controlled local Git integration",DL-13:范围止于本地集成)。
//
// 本包是语义层,不是第二个 Git 子系统:
//
//   - 只暴露有名字、带类型不变量的操作(Integrate),绝不暴露
//     模型可控的裸 Git argv / 任意命令透传;
//   - 一切 Git 变更仍经由 internal/gitwriter 提交 —— 复用其
//     expected-base CAS 校验、进程内 + 跨进程锁与原子回滚,
//     本包不重复实现任何 Git plumbing(DL-03:Git 持久化权威不动);
//   - 只集成「已识别 + 已接受」的 ChangeSet:未识别 / 未接受 /
//     验收事实与候选身份不匹配 / 无 basis 的验收一律拒绝,失败安全;
//   - 集成目标必须是集成工作树,永远不是 owner 主工作树
//     (信任不变量:owner primary worktree 不是候选执行/集成沙箱)。
//
// 后续任务(P5-02 分支创建、P5-03 合并)在本包语义之上叠加流程,
// 不新增并行权威。
package gitintegration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tally/internal/changeset"
	"tally/internal/gitwriter"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrCandidateUnidentified 候选未识别:没有 ChangeSet,或缺少身份/base。
	ErrCandidateUnidentified = errors.New("gitintegration: 候选快照未识别")
	// ErrNotAccepted 候选未被接受:缺少验收事实,或验收事实缺 basis/时间。
	ErrNotAccepted = errors.New("gitintegration: 候选未被接受")
	// ErrAcceptanceMismatch 验收事实与候选身份不匹配。
	ErrAcceptanceMismatch = errors.New("gitintegration: 验收事实与候选身份不匹配")
	// ErrProtectedTarget 集成目标是受保护路径(owner 主工作树)。
	ErrProtectedTarget = errors.New("gitintegration: 集成目标是受保护的主工作树")
	// ErrNoExpectedBase 缺少 expected base:CAS 语义下失败安全,不猜测当前状态。
	ErrNoExpectedBase = errors.New("gitintegration: 缺少 expected base")
	// ErrStaleBase 目标当前 HEAD 已偏离 expected base(冲突是预期类型化结果,
	// 同时包着 gitwriter.ErrBaseRevision,两个词都可 errors.Is 命中)。
	ErrStaleBase = errors.New("gitintegration: 目标 HEAD 偏离 expected base")
	// ErrNoApplicableContent ChangeSet 没有可应用的改动文件。
	ErrNoApplicableContent = errors.New("gitintegration: 候选无可应用的改动文件")
	// ErrNoRepository 缺少集成目标路径。
	ErrNoRepository = errors.New("gitintegration: 缺少集成目标仓库路径")
)

// AcceptanceFact 是「该 ChangeSet 已被接受」的类型化验收事实。
//
// 它由闭环接受权威(closedloop accept 事件及其记录)产生,本包只消费
// 不裁定:缺少事实或事实与候选身份不匹配时,集成直接失败安全。
// Basis 必填 —— 每个 ACCEPTABLE/VERIFIED 类结果都必须携带 basis,
// 空_basis 的验收视为无效验收(ErrNotAccepted)。
type AcceptanceFact struct {
	// ChangeSetID 被接受的候选快照身份;必须与被集成的 ChangeSet 一致。
	ChangeSetID changeset.ChangeSetID
	// AcceptedAt 接受时刻(UTC);零值无效。
	AcceptedAt time.Time
	// Basis 验收依据的可追溯引用(如接受事件/记录标识);必填非空。
	Basis string
}

// valid 报告验收事实是否完整。零值事实一律无效。
func (f AcceptanceFact) valid() bool {
	return f.ChangeSetID != "" &&
		!f.AcceptedAt.IsZero() &&
		strings.TrimSpace(f.Basis) != ""
}

// Config 构造 Service 的配置。
type Config struct {
	// PrimaryWorktree 是 owner 主工作树路径;集成目标与之相同即拒绝
	// (信任不变量:主工作树永远不是集成沙箱)。必填。
	PrimaryWorktree string
	// Signature 是集成提交的兜底身份;零值用 gitwriter 包默认。
	Signature gitwriter.Signature
}

// Service 是本地 Git 集成语义服务。零值不可用,用 New 构造。
// 无自身可变状态,可并发使用(Git 串行化由 gitwriter 负责)。
type Service struct {
	primaryWorktree string
	signature       gitwriter.Signature
}

// New 构造集成服务。主工作树路径必填:不声明主工作树的服务
// 无法执行主工作树保护,失败安全起见拒绝构造。
func New(cfg Config) (*Service, error) {
	if strings.TrimSpace(cfg.PrimaryWorktree) == "" {
		return nil, errors.New("gitintegration: PrimaryWorktree 是必填项(主工作树保护)")
	}
	abs, err := filepath.Abs(cfg.PrimaryWorktree)
	if err != nil {
		return nil, fmt.Errorf("gitintegration: 解析主工作树路径: %w", err)
	}
	return &Service{
		primaryWorktree: filepath.Clean(abs),
		signature:       cfg.Signature,
	}, nil
}

// IntegrateRequest 描述一次受控本地集成。
type IntegrateRequest struct {
	// ChangeSet 被集成的不可变候选快照;必须已识别(非 nil 且有身份)。
	ChangeSet *changeset.ChangeSet
	// Acceptance 该候选的验收事实;必须完整且与候选身份一致。
	Acceptance AcceptanceFact
	// RepositoryPath 集成目标工作树路径(集成分支所在的工作树,
	// 不是 owner 主工作树)。
	RepositoryPath string
	// ExpectedBase 集成时刻目标 HEAD 必须等于的提交(CAS)。
	// 为空即拒绝:集成分支的 unborn/未知状态不是合法集成起点。
	ExpectedBase string
}

// IntegrationResult 报告一次成功的受控集成。
type IntegrationResult struct {
	// ChangeSetID 被集成的候选快照身份。
	ChangeSetID changeset.ChangeSetID
	// Base 通过 CAS 校验的 base 提交。
	Base string
	// Commit 本次集成产生的提交(gitwriter 单一写边界产出)。
	Commit string
}

// Integrate 把一个已接受、已识别的 ChangeSet 以 CAS 语义应用到集成目标:
// 校验候选身份与验收事实、拒绝主工作树与空 base、预检目标 HEAD,
// 然后经 gitwriter 以 expected-base 提交全部改动文件(原子、可回滚、跨进程串行)。
// 任何一步不满足都不产生提交 —— 冲突(基线过期)是预期类型化结果,不是异常。
func (s *Service) Integrate(ctx context.Context, req IntegrateRequest) (IntegrationResult, error) {
	// 1. 候选必须已识别:有快照、有身份、有固定 base。
	cs := req.ChangeSet
	if cs == nil || cs.ID() == "" || strings.TrimSpace(cs.BaseRevision()) == "" {
		return IntegrationResult{}, fmt.Errorf("%w: 需要 ChangeSet 及其身份与 base", ErrCandidateUnidentified)
	}

	// 2. 验收事实必须完整且指向同一候选。
	if !req.Acceptance.valid() {
		return IntegrationResult{}, fmt.Errorf("%w: 验收事实不完整(需 ChangeSetID/AcceptedAt/Basis)", ErrNotAccepted)
	}
	if req.Acceptance.ChangeSetID != cs.ID() {
		return IntegrationResult{}, fmt.Errorf("%w: 事实指向 %s,候选是 %s", ErrAcceptanceMismatch, req.Acceptance.ChangeSetID, cs.ID())
	}

	// 3. 必须有可应用内容:只含 candidate revision 引用而无文件内容的快照
	// 无法经 gitwriter 全量内容模型应用,失败安全。
	files := cs.Files()
	if len(files) == 0 {
		return IntegrationResult{}, fmt.Errorf("%w: ChangeSet %s", ErrNoApplicableContent, cs.ID())
	}

	// 4. 目标与基线校验。
	if strings.TrimSpace(req.RepositoryPath) == "" {
		return IntegrationResult{}, ErrNoRepository
	}
	if s.isProtected(req.RepositoryPath) {
		return IntegrationResult{}, fmt.Errorf("%w: %s", ErrProtectedTarget, req.RepositoryPath)
	}
	if strings.TrimSpace(req.ExpectedBase) == "" {
		return IntegrationResult{}, fmt.Errorf("%w: 集成必须显式声明 expected base", ErrNoExpectedBase)
	}

	writer, err := gitwriter.New(req.RepositoryPath)
	if err != nil {
		return IntegrationResult{}, err
	}
	if s.signature != (gitwriter.Signature{}) {
		writer.Signature = s.signature
	}

	// 5. 预检 CAS:提前把「基线过期」暴露为预期类型化结果,
	// 避免把注定失败的请求送进提交路径。gitwriter.Commit 在锁内
	// 会再次校验,预检与提交之间发生的移动仍由其 ErrBaseRevision 兜住。
	head, err := writer.Head(ctx)
	if err != nil {
		return IntegrationResult{}, err
	}
	if head != req.ExpectedBase {
		return IntegrationResult{}, fmt.Errorf("%w: 当前 %s,期望 %s: %w", ErrStaleBase, head, req.ExpectedBase, gitwriter.ErrBaseRevision)
	}

	// 6. 经单一写边界提交。提交信息是语义化、确定性的,
	// 不透传任何模型可控文本。
	updates := make([]gitwriter.FileUpdate, 0, len(files))
	for _, f := range files {
		updates = append(updates, gitwriter.FileUpdate{
			Path:    f.Path,
			Content: []byte(f.Content),
		})
	}
	res, err := writer.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: req.ExpectedBase,
		Message:      fmt.Sprintf("tally: integrate changeset %s", cs.ID()),
		Files:        updates,
	})
	if err != nil {
		if errors.Is(err, gitwriter.ErrBaseRevision) {
			return IntegrationResult{}, fmt.Errorf("%w: 提交时基线已移动: %w", ErrStaleBase, err)
		}
		return IntegrationResult{}, err
	}
	return IntegrationResult{
		ChangeSetID: cs.ID(),
		Base:        res.Base,
		Commit:      res.Commit,
	}, nil
}

// isProtected 报告目标路径是否就是 owner 主工作树。
// Windows 上路径大小写不敏感,用不区分大小写比较;其余平台精确比较。
func (s *Service) isProtected(repoPath string) bool {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(abs, s.primaryWorktree)
	}
	return abs == s.primaryWorktree
}
