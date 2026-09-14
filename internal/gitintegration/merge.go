package gitintegration

// 安全本地合并(P5-03):把已接受候选的确定性集成分支
// (P5-02 的 refs/heads/tally/integration/<hex>)合并进调用方声明的目标
// ref。冲突是预期的类型化结果,绝无自主解决、强制或历史重写。
//
// 语义不变量(对应任务验收标准):
//
//   - 冲突失败安全:先用 `git merge-tree --write-tree` 做只读预检
//     (不触碰任何工作树/索引),有冲突即返回携带冲突文件列表的类型化
//     结果,不落任何 ref、不留部分隐藏状态;
//   - 无强制/无历史重写:目标 ref 只经 `git update-ref --stdin` 的
//     `update <ref> <new> <old>` 原语前进 —— old 不匹配即失败,这是
//     结构性 CAS,没有 force/delete/reset 路径;
//   - 保留审计 ref:本操作只写目标 ref,集成分支只读不写、永不删除;
//     Tally 自己的 tally/ ref 命名空间禁止作为合并目标;
//   - base 漂移类型化:目标 ref 或集成分支偏离调用方声明的期望值
//     (ExpectedTarget / ExpectedBranchCommit)即 ErrStaleBase,
//     绝不静默 rebase/重试;
//   - 真合并经 plumbing 生成确定性双亲合并提交(commit-tree + 两个
//     -p),消息固定为 "tally: merge changeset <id>",不透传模型文本;
//     目标本就是分支祖先时按快进前进(只移动 ref,不造新提交);
//   - 目标 ref 不得是任何工作树(含本仓库全部 linked worktree)当前
//     检出的分支 —— 在检出的分支下面移动 ref 会让该工作树与索引
//     静默失真,失败安全拒绝;
//   - L3 集成授权(P5-04 权限层):操作要求 Decider.Authorize 就本目标
//     ref 签发的令牌;受保护目标的令牌以策略批准(Approval)为前提。
//     runtime 协议一侧构造不出有效令牌 —— 合并没有绕过 Tally 权限
//     决策、也没有绕过受保护分支策略的入口。

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"tally/internal/changeset"
	"tally/internal/gitwriter"
)

// 合并哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrMergeConflict 合并存在冲突:预期类型化结果,绝无自主解决。
	// 具体冲突文件经 *MergeConflictError(errors.As)获取。
	ErrMergeConflict = errors.New("gitintegration: 合并存在冲突")
	// ErrInvalidTargetRef 目标 ref 不满足格式约束(必须是受控的完整
	// refs/heads/ 名字,且不在 Tally 审计命名空间内)。
	ErrInvalidTargetRef = errors.New("gitintegration: 目标 ref 非法")
	// ErrTargetCheckedOut 目标 ref 是某个工作树当前检出的分支:
	// 在其下移动 ref 会造成工作树/索引失真,失败安全拒绝。
	ErrTargetCheckedOut = errors.New("gitintegration: 目标 ref 已被工作树检出")
	// ErrIntegrationBranchMissing 候选的确定性集成分支不存在:
	// 没有可合并的分支是不明状态,失败安全,不猜测。
	ErrIntegrationBranchMissing = errors.New("gitintegration: 集成分支不存在")
	// ErrAlreadyMerged 集成分支顶端已包含于目标(重放已成功的合并):
	// 类型化非成功,目标保持不动。
	ErrAlreadyMerged = errors.New("gitintegration: 候选已包含于目标")
)

// MergeConflictError 携带 merge-tree 预检报告的冲突文件列表。
// errors.Is(err, ErrMergeConflict) 与 errors.As(err, &MergeConflictError{})
// 都可判别;不携带任何冲突内容细节(预检树被丢弃,不落任何 ref)。
type MergeConflictError struct {
	// Paths 冲突文件的仓库相对路径(POSIX 风格),至少一个。
	Paths []string
}

func (e *MergeConflictError) Error() string {
	return fmt.Sprintf("%s: %s", ErrMergeConflict.Error(), strings.Join(e.Paths, ", "))
}

func (e *MergeConflictError) Unwrap() error { return ErrMergeConflict }

// MergeRequest 描述一次受控本地合并:把候选的确定性集成分支合并进
// 目标 ref。目标 ref 由调用方声明(它是用户拥有的分支,不是 Tally
// 推导的),但必须通过严格格式校验;两侧期望值都必填(CAS)。
type MergeRequest struct {
	// ChangeSet 被合并候选的快照;其身份推导集成分支名,
	// 其 BaseRevision 校验分支确实承载该候选的基线。必须已识别。
	ChangeSet *changeset.ChangeSet
	// Acceptance 该候选的验收事实;必须完整且与候选身份一致。
	Acceptance AcceptanceFact
	// RepositoryPath 目标仓库的工作树路径(不是 owner 主工作树)。
	RepositoryPath string
	// TargetRef 合并目标 ref,必须是完整的 refs/heads/<name> 且不在
	// refs/heads/tally/ 审计命名空间内(审计 ref 只读)。
	TargetRef string
	// ExpectedTarget 合并时刻目标 ref 必须指向的提交(CAS);
	// 为空即拒绝。集成分支的 unborn/未知状态不是合法合并起点。
	ExpectedTarget string
	// ExpectedBranchCommit 合并时刻集成分支必须指向的提交
	// (即 Integrate 产出的提交);为空即拒绝。分支在观察后漂移
	// 是类型化冲突,绝不静默合并未知内容。
	ExpectedBranchCommit string
	// Authorization 是 Decider.Authorize 就本目标 ref 签发的 L3 集成授权
	// 令牌(P5-04 权限层)。零值/绑定目标不符即拒绝:runtime 协议一侧
	// 构造不出有效令牌;受保护目标还要求携带策略批准(Approval)才拿得到令牌。
	Authorization Authorization
}

// MergeResult 报告一次成功的受控合并。
type MergeResult struct {
	// ChangeSetID 被合并的候选快照身份。
	ChangeSetID changeset.ChangeSetID
	// TargetRef 被前进的目标 ref(原样返回调用方声明的合法值)。
	TargetRef string
	// OldTarget 合并前目标 ref 指向的提交(通过 CAS 校验的期望值)。
	OldTarget string
	// Commit 合并后目标 ref 指向的提交:快进时是集成分支顶端,
	// 真合并时是新造的双亲合并提交。
	Commit string
	// FastForward 为 true 表示目标是集成分支祖先,本次没有创建
	// 新提交,只把目标 ref 前进到集成分支顶端。
	FastForward bool
}

// Merge 把已接受候选的确定性集成分支安全合并进目标 ref:
// 校验候选/目标管线与 L3 集成授权(受保护目标要求策略批准过的令牌),
// 校验验收事实,拒绝被检出的目标与 tally/ 审计 ref,CAS 校验目标与
// 集成分支的当前值,只读预检合并冲突,最后以 compare-and-swap 原语前进
// 目标 ref。任何一步不满足都不移动任何 ref —— 冲突与 base 漂移都是
// 预期类型化结果,不是异常。
func (s *Service) Merge(ctx context.Context, req MergeRequest) (MergeResult, error) {
	// 1. 候选必须已识别:有快照、有身份、有固定 base。
	cs := req.ChangeSet
	if cs == nil || cs.ID() == "" || strings.TrimSpace(cs.BaseRevision()) == "" {
		return MergeResult{}, fmt.Errorf("%w: 需要 ChangeSet 及其身份与 base", ErrCandidateUnidentified)
	}

	// 2. 目标校验:路径必填、永远不是 owner 主工作树,目标 ref 格式合法。
	if strings.TrimSpace(req.RepositoryPath) == "" {
		return MergeResult{}, ErrNoRepository
	}
	if s.isProtected(req.RepositoryPath) {
		return MergeResult{}, fmt.Errorf("%w: %s", ErrProtectedTarget, req.RepositoryPath)
	}
	if err := validateTargetRef(req.TargetRef); err != nil {
		return MergeResult{}, fmt.Errorf("%w: %w", ErrInvalidTargetRef, err)
	}

	// 3. L3 集成授权(结构性执行点):没有 Decider 就本目标 ref 签发且绑定
	//    该目标的令牌,操作在触碰任何仓库状态之前即被拒绝。runtime 协议
	//    一侧构造不出有效令牌 —— 受保护分支的合并令牌还以策略批准为前提。
	if err := requireAuthorization(req.Authorization, req.TargetRef); err != nil {
		return MergeResult{}, err
	}

	// 4. 验收事实必须完整且指向同一候选:未接受的候选不进入合并。
	if !req.Acceptance.valid() {
		return MergeResult{}, fmt.Errorf("%w: 验收事实不完整(需 ChangeSetID/AcceptedAt/Basis)", ErrNotAccepted)
	}
	if req.Acceptance.ChangeSetID != cs.ID() {
		return MergeResult{}, fmt.Errorf("%w: 事实指向 %s,候选是 %s", ErrAcceptanceMismatch, req.Acceptance.ChangeSetID, cs.ID())
	}

	// 5. 两侧期望值必填(CAS)。
	if strings.TrimSpace(req.ExpectedTarget) == "" || strings.TrimSpace(req.ExpectedBranchCommit) == "" {
		return MergeResult{}, fmt.Errorf("%w: 合并必须显式声明目标与集成分支的期望值", ErrNoExpectedBase)
	}

	writer, err := gitwriter.New(req.RepositoryPath)
	if err != nil {
		return MergeResult{}, err
	}
	// 6. 目标必须是真实仓库(类型化 ErrNotRepository,与既有管线一致)。
	if _, err = writer.Head(ctx); err != nil {
		return MergeResult{}, err
	}
	repo := writer.RepoPath

	// 7. 目标 ref 不得被任何工作树检出:在其下移动 ref 会让那个工作树
	//    与索引静默失真。这是结构性防护,先于一切读写在最前面拒绝。
	if err = s.rejectCheckedOutTarget(ctx, repo, req.TargetRef); err != nil {
		return MergeResult{}, err
	}

	// 8. 目标 ref CAS:当前值必须等于调用方声明的期望值。
	cur, err := readRef(ctx, repo, req.TargetRef)
	if err != nil {
		return MergeResult{}, err
	}
	if cur == "" {
		return MergeResult{}, fmt.Errorf("%w: 目标 ref %s 不存在", ErrNoExpectedBase, req.TargetRef)
	}
	if cur != req.ExpectedTarget {
		return MergeResult{}, fmt.Errorf("%w: 目标当前 %s,期望 %s", ErrStaleBase, cur, req.ExpectedTarget)
	}

	// 9. 集成分支 CAS:确定性命名,当前值必须等于观察到的分支顶端。
	ref, err := integrationBranchRef(cs.ID())
	if err != nil {
		return MergeResult{}, err
	}
	tip, err := readRef(ctx, repo, ref)
	if err != nil {
		return MergeResult{}, err
	}
	if tip == "" {
		return MergeResult{}, fmt.Errorf("%w: %s", ErrIntegrationBranchMissing, ref)
	}
	if tip != req.ExpectedBranchCommit {
		return MergeResult{}, fmt.Errorf("%w: 集成分支当前 %s,期望 %s", ErrStaleBase, tip, req.ExpectedBranchCommit)
	}

	// 10. 分支必须确实承载候选的固定 base(防御:ref 被外部移动过),
	//    否则这个分支与该候选没有可证关系,失败安全拒绝。
	base, err := resolveCommit(ctx, repo, cs.BaseRevision())
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return MergeResult{}, err
		}
		return MergeResult{}, fmt.Errorf("%w: base %s: %w", ErrBaseNotFound, cs.BaseRevision(), err)
	}
	carries, err := isAncestor(ctx, repo, base, tip)
	if err != nil {
		return MergeResult{}, err
	}
	if !carries {
		return MergeResult{}, fmt.Errorf("%w: 集成分支 %s 不承载候选 base %s", ErrStaleBase, ref, base)
	}

	// 11. 已包含:分支顶端是目标祖先(含相等)→ 重放已成功的合并,
	//    类型化非成功,目标保持不动。
	if tip == cur {
		return MergeResult{}, fmt.Errorf("%w: %s 已在 %s 中", ErrAlreadyMerged, tip, req.TargetRef)
	}
	contained, err := isAncestor(ctx, repo, tip, cur)
	if err != nil {
		return MergeResult{}, err
	}
	if contained {
		return MergeResult{}, fmt.Errorf("%w: %s 已包含于 %s", ErrAlreadyMerged, tip, req.TargetRef)
	}
	ff, err := isAncestor(ctx, repo, cur, tip)
	if err != nil {
		return MergeResult{}, err
	}

	// 12. 快进:目标是集成分支祖先 → 只经 CAS 把目标 ref 前进到分支
	//     顶端,不创建新提交。这不重写历史(旧提交仍是新顶端的祖先)。
	if ff {
		if err = casUpdateRef(ctx, repo, req.TargetRef, tip, cur); err != nil {
			return MergeResult{}, err
		}
		return MergeResult{
			ChangeSetID: cs.ID(),
			TargetRef:   req.TargetRef,
			OldTarget:   cur,
			Commit:      tip,
			FastForward: true,
		}, nil
	}

	// 13. 真合并:先只读预检(merge-tree 不触碰工作树/索引),
	//     冲突即携带文件列表的类型化结果,不落任何 ref。
	tree, paths, err := mergePreflight(ctx, repo, cur, tip)
	if err != nil {
		return MergeResult{}, err
	}
	if len(paths) > 0 {
		return MergeResult{}, &MergeConflictError{Paths: paths}
	}

	// 14. 无冲突:经 plumbing 生成确定性双亲合并提交(不碰工作树),
	//     再以 CAS 前进目标 ref。任何失败都不移动目标。
	sig := s.signatureOrFallback()
	commit, err := runGit(ctx, repo, "",
		"-c", "user.name="+sig.Name,
		"-c", "user.email="+sig.Email,
		"commit-tree", tree,
		"-p", cur, "-p", tip,
		"-m", fmt.Sprintf("tally: merge changeset %s", cs.ID()),
	)
	if err != nil {
		return MergeResult{}, err
	}
	mergeCommit := strings.TrimSpace(commit)
	if mergeCommit == "" {
		return MergeResult{}, errors.New("gitintegration: commit-tree 未返回 OID")
	}
	if err = casUpdateRef(ctx, repo, req.TargetRef, mergeCommit, cur); err != nil {
		return MergeResult{}, err
	}
	return MergeResult{
		ChangeSetID: cs.ID(),
		TargetRef:   req.TargetRef,
		OldTarget:   cur,
		Commit:      mergeCommit,
	}, nil
}

// signatureOrFallback 返回合并提交身份:服务配置的身份,未配置时用
// 与 gitwriter 包默认一致的固定身份(绝不依赖机器上的 ambient git 配置)。
func (s *Service) signatureOrFallback() gitwriter.Signature {
	if s.signature != (gitwriter.Signature{}) {
		return s.signature
	}
	return gitwriter.Signature{Name: "tally", Email: "tally@tally.local"}
}

// validateTargetRef 校验合并目标 ref 的约束。目标由调用方声明
// (用户拥有的分支),因此必须结构化校验而不是信任:
//   - 必须是合法的完整 refs/heads/<name>(validHeadRef);
//   - 且不得进入 refs/heads/tally/ 审计命名空间(本包创建的集成 ref 只读)。
func validateTargetRef(ref string) error {
	if err := validHeadRef(ref); err != nil {
		return err
	}
	if strings.HasPrefix(ref, "refs/heads/tally/") {
		return errors.New("tally/ 审计命名空间不可作为合并目标")
	}
	return nil
}

// validHeadRef 校验完整 refs/heads/<name> 的格式约束(权限层与合并目标
// 校验共用;不限制 tally/ 命名空间 —— 那是 validateTargetRef 的语义):
//   - 必须是完整 refs/heads/<name>,非空、不以 "-" 开头(防选项注入);
//   - 不得包含 ref 危险字元(空格、控制字符、~^:?*[]\、..、@{、
//     双斜杠、尾部 /、尾部 .lock)。
func validHeadRef(ref string) error {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return fmt.Errorf("必须是完整 %q ref 名", prefix)
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("分支名非法: %q", name)
	}
	switch {
	case strings.ContainsAny(name, " \t\r\n\x00~^:?*[]\\"),
		strings.Contains(name, ".."),
		strings.Contains(name, "//"),
		strings.Contains(name, "@{"),
		strings.HasSuffix(name, "/"),
		strings.HasSuffix(name, ".lock"):
		return fmt.Errorf("分支名非法: %q", name)
	}
	return nil
}

// rejectCheckedOutTarget 在目标 ref 是任何工作树(主工作树与 linked
// worktree)当前检出的分支时拒绝合并。porcelain 输出按块解析,只在
// 值精确等于目标 ref 时命中,不做子串猜测。
func (s *Service) rejectCheckedOutTarget(ctx context.Context, repo, targetRef string) error {
	out, err := runGit(ctx, repo, "", "worktree", "list", "--porcelain")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(out, "\n") {
		const branchField = "branch "
		if strings.HasPrefix(line, branchField) && strings.TrimSpace(strings.TrimPrefix(line, branchField)) == targetRef {
			return fmt.Errorf("%w: %s", ErrTargetCheckedOut, targetRef)
		}
	}
	return nil
}

// casUpdateRef 以 `git update-ref --stdin` 的 update 原语做 CAS 前进:
// 仅当 ref 当前仍等于 old 时才移到 new;被并发移动即失败。失败后读回
// 当前值归类:不再是 old 即类型化 base 漂移,否则原样上抛。
// 这是本操作唯一的 ref 写路径 —— 没有 force/delete/reset 结构。
func casUpdateRef(ctx context.Context, repo, ref, newOID, oldOID string) error {
	_, err := runGit(ctx, repo, fmt.Sprintf("update %s %s %s\n", ref, newOID, oldOID), "update-ref", "--stdin")
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return err
	}
	cur, rerr := readRef(ctx, repo, ref)
	if rerr != nil {
		return errors.Join(err, rerr)
	}
	if cur != oldOID {
		return fmt.Errorf("%w: %s 提交时已移动到 %s: %w", ErrStaleBase, ref, cur, err)
	}
	return err
}

// isAncestor 报告 a 是否 b 的祖先(含相等)。merge-base --is-ancestor
// 以退出码 0/1 表达是/否;其余退出码是真实错误(取消优先原样上抛)。
func isAncestor(ctx context.Context, repo, a, b string) (bool, error) {
	_, err := runGit(ctx, repo, "", "merge-base", "--is-ancestor", a, b)
	if err == nil {
		return true, nil
	}
	if code, ok := gitExitCode(err); ok && code == 1 && ctx.Err() == nil {
		return false, nil
	}
	return false, err
}

// mergePreflight 用 `git merge-tree --write-tree -z --name-only` 做
// 只读合并预检(EXT-GIT-MERGE-TREE):不触碰任何工作树/索引,退出码
// 0 = 干净(首字段为合并树 OID),1 = 冲突(首字段后、空字段前是
// 冲突文件列表),其余 = 真实错误原样上抛。冲突时返回的树被丢弃,
// 不作为任何 ref 的输入。参数都是本包解析出的 OID 字面量。
func mergePreflight(ctx context.Context, repo, ours, theirs string) (tree string, conflictPaths []string, err error) {
	out, err := runGit(ctx, repo, "", "merge-tree", "--write-tree", "-z", "--name-only", ours, theirs)
	if err == nil {
		fields := strings.Split(out, "\x00")
		if len(fields) == 0 || strings.TrimSpace(fields[0]) == "" {
			return "", nil, errors.New("gitintegration: merge-tree 未返回树 OID")
		}
		return fields[0], nil, nil
	}
	if ctx.Err() != nil {
		return "", nil, err
	}
	if code, ok := gitExitCode(err); ok && code == 1 {
		// -z 输出: <tree> NUL <冲突文件>* NUL NUL <信息性消息>。
		// 冲突文件是首字段后到第一个空字段之间的全部字段。
		fields := strings.Split(out, "\x00")
		var paths []string
		for _, f := range fields[1:] {
			if f == "" {
				break
			}
			paths = append(paths, f)
		}
		if len(paths) == 0 {
			return "", nil, fmt.Errorf("gitintegration: merge-tree 报冲突但未列出文件: %w", err)
		}
		return "", paths, nil
	}
	return "", nil, err
}

// gitExitCode 从 gitwriter.GitError 中提取 git 进程退出码,
// 供按退出码语义分类的调用方(merge-tree / merge-base)使用。
func gitExitCode(err error) (int, bool) {
	var gerr *gitwriter.GitError
	if !errors.As(err, &gerr) {
		return -1, false
	}
	var ee *exec.ExitError
	if !errors.As(gerr.Err, &ee) {
		return -1, false
	}
	return ee.ExitCode(), true
}
