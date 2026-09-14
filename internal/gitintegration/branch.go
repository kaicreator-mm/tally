package gitintegration

// 集成分支创建(P5-02):从已接受候选的固定 base 创建确定性、
// 归属于 Tally 的本地集成分支,作为 P5-03 安全合并的审计 ref 起点。
//
// 语义不变量:
//
//   - 分支名与分支起点都从 ChangeSet 确定性推导 —— 请求里没有一处
//     可自由命名的 ref 文本,模型可控内容永远到不了 Git argv/stdin;
//   - 创建是 CAS:git update-ref --stdin 的 create 原语只在 ref 不存在时
//     创建,已存在即失败 —— 「静默覆盖已有分支」没有结构性路径;
//   - 分支已存在且恰好指向 expected base 是幂等成功(同一确定性输入的
//     期望状态),不发生任何变更;base 漂移(指向别处)是预期的类型化冲突;
//   - 本操作只创建 ref,不移动 HEAD、不检出、不产生提交(提交仍是
//     Integrate 经 gitwriter 单一写边界的事);
//   - L3 集成授权(P5-04 权限层):操作要求 Decider.Authorize 就本操作
//     确定性 ref 签发的令牌;runtime 协议一侧构造不出有效令牌,
//     集成分支创建没有绕过 Tally 权限决策的入口。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/changeset"
	"tally/internal/gitwriter"
)

// 集成分支哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrIntegrationBranchExists 集成分支已存在且不指向 expected base:
	// 预期的类型化冲突 —— 绝不重置/覆盖已有分支。
	ErrIntegrationBranchExists = errors.New("gitintegration: 集成分支已存在且偏离 expected base")
	// ErrBaseNotFound 候选固定 base 无法在集成目标仓库解析为提交:
	// 在没有该 base 的仓库上创建分支失败安全,不猜测当前状态。
	ErrBaseNotFound = errors.New("gitintegration: expected base 不存在于集成目标仓库")
)

// integrationBranchPrefix 是 Tally 集成分支的 ref 命名空间:
// "tally/" 前缀声明所有权(这是 Tally 拥有的集成 ref,不是任何主分支),
// 后接候选身份的 sha256 十六进制 —— 对同一 ChangeSet 永远同名,
// 对不同 ChangeSet 永远不同名。不嵌入 project 等自由文本:ProjectID 是
// 任意字符串,直接进 ref 名可能违反 ref 格式,清洗则引入碰撞风险;
// 身份十六进制本身已编码 project/base/candidate/run。
const integrationBranchPrefix = "refs/heads/tally/integration/"

// IntegrationBranchRequest 描述一次受控集成分支创建。
// 分支名与起点都从 ChangeSet 推导,请求无需(也无法)指定 ref 文本。
type IntegrationBranchRequest struct {
	// ChangeSet 分支所服务的已接受候选快照;必须已识别(非 nil 且有身份)。
	ChangeSet *changeset.ChangeSet
	// Acceptance 该候选的验收事实;必须完整且与候选身份一致
	// (信任不变量:只有已接受的候选才进入受控本地集成流程)。
	Acceptance AcceptanceFact
	// RepositoryPath 集成目标工作树路径(不是 owner 主工作树)。
	RepositoryPath string
	// Authorization 是 Decider.Authorize 就本操作的确定性集成分支 ref
	// 签发的 L3 集成授权令牌(P5-04 权限层)。零值/绑定目标不符即拒绝:
	// runtime 协议一侧构造不出有效令牌,L3 操作没有绕过决策的入口。
	Authorization Authorization
}

// IntegrationBranchResult 报告一次集成分支创建(或幂等确认)。
type IntegrationBranchResult struct {
	// ChangeSetID 分支所服务的候选快照身份。
	ChangeSetID changeset.ChangeSetID
	// Branch 分支的完整 ref 名(refs/heads/tally/integration/<hex>)。
	Branch string
	// Base 分支指向的提交:候选固定 base 在目标仓库解析出的完整 OID
	// (解析后的不可变值,不是可移动的表达式)。
	Base string
	// Created 为 false 表示分支已存在且指向同一 base(幂等确认),
	// 本次没有发生任何变更。
	Created bool
}

// EnsureIntegrationBranch 从已接受候选的固定 base 创建确定性集成分支:
// 校验候选身份、拒绝主工作树目标(与 Integrate 同一条管线),校验就本操作
// 确定性 ref 签发的 L3 集成授权(权限决策在 Tally 域/组合根,runtime 侧
// 构造不出有效令牌),校验验收事实,把候选固定 base 解析为目标仓库中的
// 完整提交 OID,然后以 CAS 原语创建 refs/heads/tally/integration/<候选身份十六进制>。
//
// 分支已存在时:指向同一 base 即幂等成功(Created=false);指向别处即
// ErrIntegrationBranchExists 类型化冲突 —— 无论哪种情况都不重写已有 ref。
// 本操作不移动 HEAD、不检出、不产生提交。
func (s *Service) EnsureIntegrationBranch(ctx context.Context, req IntegrationBranchRequest) (IntegrationBranchResult, error) {
	// 1. 候选必须已识别:有快照、有身份、有固定 base。
	cs := req.ChangeSet
	if cs == nil || cs.ID() == "" || strings.TrimSpace(cs.BaseRevision()) == "" {
		return IntegrationBranchResult{}, fmt.Errorf("%w: 需要 ChangeSet 及其身份与 base", ErrCandidateUnidentified)
	}

	// 2. 确定性命名(目标识别):候选身份映射为固定 ref 名
	//    (身份格式异常即失败安全);授权令牌按该目标绑定校验。
	ref, err := integrationBranchRef(cs.ID())
	if err != nil {
		return IntegrationBranchResult{}, err
	}

	// 3. 目标校验:路径必填,且永远不是 owner 主工作树。
	if strings.TrimSpace(req.RepositoryPath) == "" {
		return IntegrationBranchResult{}, ErrNoRepository
	}
	if s.isProtected(req.RepositoryPath) {
		return IntegrationBranchResult{}, fmt.Errorf("%w: %s", ErrProtectedTarget, req.RepositoryPath)
	}

	// 4. L3 集成授权(结构性执行点):没有 Decider 签发且绑定本操作 ref 的
	//    令牌,操作在触碰任何仓库状态之前即被拒绝。runtime 协议一侧构造
	//    不出有效令牌 —— L3 没有绕过权限决策的入口。
	if err := requireAuthorization(req.Authorization, ref); err != nil {
		return IntegrationBranchResult{}, err
	}

	// 5. 验收事实必须完整且指向同一候选:未接受的候选不进入受控集成。
	if !req.Acceptance.valid() {
		return IntegrationBranchResult{}, fmt.Errorf("%w: 验收事实不完整(需 ChangeSetID/AcceptedAt/Basis)", ErrNotAccepted)
	}
	if req.Acceptance.ChangeSetID != cs.ID() {
		return IntegrationBranchResult{}, fmt.Errorf("%w: 事实指向 %s,候选是 %s", ErrAcceptanceMismatch, req.Acceptance.ChangeSetID, cs.ID())
	}

	// 6. 目标必须是真实仓库(gitwriter 以类型化 ErrNotRepository 报告,
	//    与 Integrate 的判定一致),ctx 取消原样上抛。
	writer, err := gitwriter.New(req.RepositoryPath)
	if err != nil {
		return IntegrationBranchResult{}, err
	}
	if _, err = writer.Head(ctx); err != nil {
		return IntegrationBranchResult{}, err
	}

	// 7. 候选固定 base 必须能在目标仓库解析为提交;解析结果是不可变 OID,
	//    作为分支起点保持确定性。解析不到即 ErrBaseNotFound,失败安全。
	base, err := resolveCommit(ctx, writer.RepoPath, cs.BaseRevision())
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return IntegrationBranchResult{}, err
		}
		return IntegrationBranchResult{}, fmt.Errorf("%w: base %s: %w", ErrBaseNotFound, cs.BaseRevision(), err)
	}

	// 8. CAS 创建。create 原语在 ref 已存在时失败 —— 失败后读回 ref 当前值
	//    归类结果:等于 base 即并发创建者建了同一确定性分支(幂等成功);
	//    指向别处即类型化冲突;仍不存在则 create 因别的原因失败,原样上抛。
	if err = createRef(ctx, writer.RepoPath, ref, base); err != nil {
		cur, rerr := readRef(ctx, writer.RepoPath, ref)
		if rerr != nil {
			return IntegrationBranchResult{}, errors.Join(err, rerr)
		}
		switch {
		case cur == base:
			return IntegrationBranchResult{ChangeSetID: cs.ID(), Branch: ref, Base: base, Created: false}, nil
		case cur != "":
			return IntegrationBranchResult{}, fmt.Errorf("%w: %s 当前指向 %s,期望 %s", ErrIntegrationBranchExists, ref, cur, base)
		default:
			return IntegrationBranchResult{}, err
		}
	}
	return IntegrationBranchResult{ChangeSetID: cs.ID(), Branch: ref, Base: base, Created: true}, nil
}

// integrationBranchRef 把候选身份映射为确定性集成分支 ref 名。
// 身份必须是 changeset 推导的 "sha256:<64 位小写十六进制>" —— 这是
// 身份推导的唯一产物格式;其他格式无法映射,失败安全拒绝,
// 绝不尝试把任意文本「清洗」进 ref 名。
func integrationBranchRef(id changeset.ChangeSetID) (string, error) {
	const sha256Prefix = "sha256:"
	hex := strings.TrimPrefix(string(id), sha256Prefix)
	if hex == string(id) || len(hex) != 64 || !isLowerHex(hex) {
		return "", fmt.Errorf("%w: 身份 %q 无法映射为确定性分支名", ErrCandidateUnidentified, id)
	}
	return integrationBranchPrefix + hex, nil
}

func isLowerHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// resolveCommit 把 revision 表达式解析为目标仓库中的完整提交 OID。
// 表达式来自 ChangeSet 固定数据:拒绝空值与选项形态(以 "-" 开头)的
// 输入,避免它被解析为 git 选项;解析不到时返回错误,不静默。
func resolveCommit(ctx context.Context, repoPath, rev string) (string, error) {
	if strings.TrimSpace(rev) == "" || strings.HasPrefix(rev, "-") {
		return "", errors.New("gitintegration: base revision 不是可解析的提交表达式")
	}
	out, err := runGit(ctx, repoPath, "", "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		return "", err
	}
	oid := strings.TrimSpace(out)
	if oid == "" {
		return "", errors.New("gitintegration: rev-parse 未返回 OID")
	}
	return oid, nil
}

// readRef 读回一个 ref 的当前完整 OID;ref 不存在时返回 ""(无错误)。
// 与 gitwriter 的 head 读法同理:--quiet 下不存在即非零退出。
func readRef(ctx context.Context, repoPath, ref string) (string, error) {
	out, err := runGit(ctx, repoPath, "", "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// createRef 用 `git update-ref --stdin` 的 create 原语做 CAS 创建:
// 只在 ref 不存在时把它指向 newOID;ref 已存在时 git 拒绝该事务 ——
// 「静默覆盖已有分支」没有结构性路径(EXT-GIT-UPDATE-REF 的
// compare-and-swap ref 语义)。ref 与 OID 都由本包推导,必为安全字面量。
func createRef(ctx context.Context, repoPath, ref, newOID string) error {
	_, err := runGit(ctx, repoPath, fmt.Sprintf("create %s %s\n", ref, newOID), "update-ref", "--stdin")
	return err
}

// runGit 转发到 gitwriter.RunGit(结构化 argv、无 shell、ctx 取消即杀进程、
// 剔除环境中的 GIT_* 变量、错误以 gitwriter.GitError 附带 argv 与输出)。
// gitwriter 是这条纪律的唯一实现方;本包只保留语义,不复制 Git 调用助手。
func runGit(ctx context.Context, dir, stdin string, args ...string) (string, error) {
	return gitwriter.RunGit(ctx, dir, stdin, args...)
}
