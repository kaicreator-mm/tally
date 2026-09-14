package gitintegration

// Git 权限策略(P5-04,PRD §18 Git Control Plane / §18.2 Permission Levels):
//
//   - 权限决策留在 Tally 域/组合根:Decider 是纯值 + 纯函数,由组合根构造并
//     调用;绝不委托给 runtime 或模型,也不在 runtime 协议上暴露任何 Git 操作
//     词汇(那是 agentruntime 的既有事实,由包内边界测试钉住);
//   - 发布范围止于 L3(DL-13):本档位只有 L0-L3 四个保留值。L4 Remote 与
//     L5 Destructive 不实现、不建模 —— 枚举之外的一切档位(含 Level(4)/
//     Level(5))在决策时失败安全拒绝,"超出范围"没有可表达的操作形态;
//   - runtime 调用方的档位天花板是 L1(PRD §18.1「Agent produces code
//     changes, Tally owns durable repository state」;DL-03:执行 runtime 不拥有
//     Tally Git 持久化权威):L2 本地提交与 L3 受控集成对 runtime 一律拒绝,
//     且没有任何 Approval 能覆盖该拒绝 —— 这是结构边界,不是可协商的策略;
//   - 受保护分支必须经策略批准的本地集成:声明在 Decider 里的受保护 ref,
//     L3 授权要求一条与目标匹配、带 basis 的 Approval(由 Tally 组合根从治理
//     事实构造;本包只消费不裁定,与 AcceptanceFact 同一纪律)。
//     EngineeringPolicy 的 YAML/struct 不做任何扩展(DL-08/BI-01):受保护
//     分支集合是组合根注入的类型化值,不是策略 schema 字段;
//   - 档位轴与 assurance.Level(PRD §13 验收保证档 L0-L4)是两个不同的轴:
//     本档位描述「谁可以对 Git 做什么」,不描述验证强度,二者不互相换算。
//
// 授权令牌(Authorization)是不可伪造的结构边界:L3 操作(EnsureIntegrationBranch
// / Merge)要求请求携带 Decider.Authorize 签发的令牌;令牌字段不导出,包外
// (包括 runtimes/** 与 internal/agentruntime)只能拿到零值,而零值无效。
// runtime 侧既无法经协议表达 L3 请求(协议只有 Instruction 词汇),也无法在
// 假想的直连下构造出有效令牌 —— 两层结构防线都由测试钉住。

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// 权限层哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidPermissionRequest 权限请求本身非法:调用方类别未知、档位不在
	// L0-L3 保留值内(L4/L5 不可表示)、或 L3 请求缺少合法目标 ref。
	ErrInvalidPermissionRequest = errors.New("gitintegration: 权限请求非法")
	// ErrRuntimeAuthorityExceeded runtime 调用方请求超出其档位天花板
	// (L2 本地提交 / L3 受控集成):结构边界,任何 Approval 都不能覆盖。
	ErrRuntimeAuthorityExceeded = errors.New("gitintegration: 调用方超出 runtime 档位天花板(runtime 不拥有 Git 持久化/集成权威)")
	// ErrProtectedTargetDenied 受保护分支缺少策略批准:没有 Approval、Approval
	// 不完整、或 Approval 与目标不匹配。
	ErrProtectedTargetDenied = errors.New("gitintegration: 受保护分支缺少策略批准的本地集成")
	// ErrPermissionRequired L3 操作未携带集成授权(零值令牌):
	// 授权由 Decider.Authorize 签发,不存在绕过决策的操作入口。
	ErrPermissionRequired = errors.New("gitintegration: 缺少集成授权")
	// ErrPermissionDenied 携带的集成授权与本操作不匹配(绑定的是别的目标):
	// 令牌不可跨目标重用。
	ErrPermissionDenied = errors.New("gitintegration: 集成授权与操作目标不匹配")
)

// Level 是 Git 权限档位(PRD §18.2),单轴、可比较、发布范围封闭于 L0-L3。
// 与 assurance.Level(验证保证档)无任何换算关系。
type Level int

const (
	// LevelRead L0 Read:status / diff / log / show 等只读 Git 事实。
	LevelRead Level = iota
	// LevelWorkspace L1 Workspace:可丢弃 worktree 的创建/清理/恢复
	// (runtime 调用方的天花板:候选执行沙箱,不触及持久仓库状态)。
	LevelWorkspace
	// LevelCommit L2 Commit:expected-base 本地提交 / 回滚 / 不可变 ChangeSet
	// (本包的对应操作是 Integrate,自带验收事实管线)。
	LevelCommit
	// LevelIntegration L3 Integration:受控分支创建 / 合并 / 冲突工作流
	// (本包的对应操作是 EnsureIntegrationBranch 与 Merge)。
	// 发布上限:L4 Remote 与 L5 Destructive 不在范围内(DL-13)。
	LevelIntegration
)

// IsValid 报告 l 是否是 L0-L3 四个保留值之一。范围外的一切值(包括任何
// L4/L5 形态)都不可表示,决策层对其失败安全拒绝。
func (l Level) IsValid() bool { return l >= LevelRead && l <= LevelIntegration }

// String 返回档位规范名("L0".."L3");范围外值返回确定性诊断文本。
func (l Level) String() string {
	switch l {
	case LevelRead:
		return "L0"
	case LevelWorkspace:
		return "L1"
	case LevelCommit:
		return "L2"
	case LevelIntegration:
		return "L3"
	default:
		return fmt.Sprintf("Level(%d)", int(l))
	}
}

// CallerKind 是调用方类别:权限矩阵的两个保留行。
type CallerKind int

const (
	// CallerTallyDomain Tally 域/组合根:闭环、集成步骤等 Tally 自己的控制面。
	CallerTallyDomain CallerKind = iota
	// CallerAgentRuntime 编码 Agent runtime 协议一侧(internal/agentruntime
	// Runtime/Session 及其适配器)。该类别没有 L2/L3 权威。
	CallerAgentRuntime
)

// IsValid 报告 k 是否是两个保留类别之一。未知类别失败安全拒绝。
func (k CallerKind) IsValid() bool { return k == CallerTallyDomain || k == CallerAgentRuntime }

// String 返回类别规范名;未知值返回确定性诊断文本。
func (k CallerKind) String() string {
	switch k {
	case CallerTallyDomain:
		return "tally_domain"
	case CallerAgentRuntime:
		return "agent_runtime"
	default:
		return fmt.Sprintf("CallerKind(%d)", int(k))
	}
}

// Caller 标识权限请求的发起方。Descriptor 只是人读的来源描述(如
// "worker/integration-step"),供审计;绝不携带凭据等敏感材料。
type Caller struct {
	Kind       CallerKind
	Descriptor string
}

// TallyCaller 构造一个 Tally 域调用方描述。
func TallyCaller(descriptor string) Caller {
	return Caller{Kind: CallerTallyDomain, Descriptor: descriptor}
}

// RuntimeCaller 构造一个 runtime 协议侧调用方描述。
func RuntimeCaller(descriptor string) Caller {
	return Caller{Kind: CallerAgentRuntime, Descriptor: descriptor}
}

// Request 是一次权限请求:对哪个档位的操作、(ref 类操作)针对哪个目标。
type Request struct {
	// Level 请求的权限档位;必须是 L0-L3 保留值。
	Level Level
	// TargetRef ref 类操作的目标完整 ref 名(refs/heads/<name>);L3 请求必填。
	// 非 ref 档位(L0/L1)可空。
	TargetRef string
}

// Approval 是「受保护分支的本地集成已获策略批准」的类型化事实,由 Tally
// 组合根从治理事实构造(本包只消费不裁定,与 AcceptanceFact 同一纪律)。
// Basis 必填:每一条放行决策都必须携带可追溯依据。
type Approval struct {
	// TargetRef 批准所针对的目标 ref;必须与请求目标完全一致。
	TargetRef string
	// Basis 批准依据的可追溯引用(治理/审批记录标识);必填非空。
	Basis string
	// ApprovedAt 批准时刻(UTC);零值无效。
	ApprovedAt time.Time
}

// valid 报告批准事实是否完整。零值批准一律无效。
func (a Approval) valid() bool {
	return a.TargetRef != "" &&
		strings.TrimSpace(a.Basis) != "" &&
		!a.ApprovedAt.IsZero()
}

// Decision 是权限决策投影(PRD §18 的决策面):纯派生值,可审计。
// Allowed=true 时 Basis 必非空 —— 放行必须带依据;拒绝经 error 表达
// (sentinel 可 errors.Is),Decision 零值返回。
type Decision struct {
	// Allowed 是否放行。
	Allowed bool
	// Level 被决策的档位(原样回显请求档位)。
	Level Level
	// Basis 放行依据(机器可读的稳定标识;受保护分支放行时内嵌批准 basis)。
	Basis string
}

// 放行依据的稳定标识(决策是纯函数,同一输入永远得到同一依据)。
const (
	// basisWithinCeiling 请求档位在调用方天花板之内(L0/L1 任意调用方;
	// L2 仅 Tally 域)。
	basisWithinCeiling = "level_within_caller_ceiling"
	// basisControlledIntegration Tally 域 L3 到非受保护目标:受控集成流程本身
	// (候选识别 + 验收事实 + CAS)就是门禁,在操作入口强制。
	basisControlledIntegration = "controlled_integration_flow"
	// basisProtectedApprovedPrefix 受保护分支放行依据前缀,后接批准 basis。
	basisProtectedApprovedPrefix = "protected_branch_approved:"
)

// Authorization 是 Decider.Authorize 签发的 L3 集成授权令牌。
//
// 字段不导出:包外(包括 runtime 侧的一切代码)无法构造非零值,零值经
// valid 判无效 —— 这是「runtime 无法请求 L3 操作」的结构执行点之一。
// 令牌绑定目标 ref:只能用于恰好该目标的 L3 操作,不可跨目标重用。
type Authorization struct {
	targetRef string
	basis     string
}

// valid 报告令牌是否可能有效(零值令牌无效;目标绑定由操作入口复核)。
func (a Authorization) valid() bool { return strings.TrimSpace(a.targetRef) != "" }

// Decider 是 Tally 域的 Git 权限决策点:持有受保护分支集合,对请求做
// 确定性决策。构造后不可变、无自身可变状态,可并发使用。
// 零值不可用,用 NewDecider 构造( Evaluate 对 nil 接收者失败安全拒绝)。
type Decider struct {
	// protected 受保护 ref 集合(构造时校验并升序存放;只读)。
	protected []string
}

// NewDecider 构造权限决策点。protectedRefs 声明受保护的完整 ref 名
// (refs/heads/<name>);nil/空表示组合根未声明受保护分支(Tally 的
// refs/heads/tally/ 审计命名空间的保全不在这里 —— 它由合并目标校验
// 结构性拒绝,集成分支创建是 Tally 自己的确定性写)。ref 非法或重复
// 即拒绝构造:保护集合是安全输入,不清洗、不去重。
func NewDecider(protectedRefs []string) (*Decider, error) {
	seen := make(map[string]struct{}, len(protectedRefs))
	out := make([]string, 0, len(protectedRefs))
	for i, ref := range protectedRefs {
		if err := validHeadRef(ref); err != nil {
			return nil, fmt.Errorf("%w: protectedRefs[%d]: %w", ErrInvalidPermissionRequest, i, err)
		}
		if _, dup := seen[ref]; dup {
			return nil, fmt.Errorf("%w: protectedRefs[%d] %q 重复", ErrInvalidPermissionRequest, i, ref)
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Strings(out)
	return &Decider{protected: out}, nil
}

// IsProtected 报告目标 ref 是否属于声明的受保护集合(精确匹配;没有 glob、
// 没有前缀推断 —— 与 EngineeringPolicy 触发轴同一「精确值相等」纪律)。
func (d *Decider) IsProtected(targetRef string) bool {
	if d == nil {
		return false
	}
	i := sort.SearchStrings(d.protected, targetRef)
	return i < len(d.protected) && d.protected[i] == targetRef
}

// Evaluate 对请求做确定性权限决策(纯函数投影,不签发令牌):
//
//  1. 调用方类别未知 → ErrInvalidPermissionRequest(失败安全);
//  2. 档位不在 L0-L3 → ErrInvalidPermissionRequest:L4 Remote / L5 Destructive
//     在本发布范围不可表示,任何请求形态都失败安全拒绝;
//  3. runtime 调用方请求 L2/L3 → ErrRuntimeAuthorityExceeded:结构边界,
//     runtime 不拥有 Tally Git 持久化(DL-03)与集成权威,Approval 不可覆盖;
//  4. 档位低于 L3 → 放行(basis: level_within_caller_ceiling);
//  5. Tally 域 L3:目标非法 → ErrInvalidPermissionRequest;目标非受保护 →
//     放行(basis: controlled_integration_flow,门禁在操作入口);
//     目标受保护 → 必须有与目标匹配、完整的 Approval,否则
//     ErrProtectedTargetDenied;放行 basis 内嵌批准 basis。
func (d *Decider) Evaluate(c Caller, req Request, approval Approval) (Decision, error) {
	if d == nil {
		return Decision{}, fmt.Errorf("%w: 权限决策点未构造(用 NewDecider)", ErrInvalidPermissionRequest)
	}
	if !c.Kind.IsValid() {
		return Decision{}, fmt.Errorf("%w: 调用方类别 %s 未知", ErrInvalidPermissionRequest, c.Kind)
	}
	if !req.Level.IsValid() {
		return Decision{}, fmt.Errorf("%w: 档位 %s 不在 L0-L3 发布范围(L4 Remote/L5 Destructive 不实现)", ErrInvalidPermissionRequest, req.Level)
	}
	if c.Kind == CallerAgentRuntime && req.Level >= LevelCommit {
		return Decision{}, fmt.Errorf("%w: runtime 调用方(%s)请求 %s: L2 提交与 L3 集成是 Tally 域权威", ErrRuntimeAuthorityExceeded, c.Descriptor, req.Level)
	}
	if req.Level < LevelIntegration {
		return Decision{Allowed: true, Level: req.Level, Basis: basisWithinCeiling}, nil
	}

	// L3:目标必须被显式、合法地声明。
	if err := validHeadRef(req.TargetRef); err != nil {
		return Decision{}, fmt.Errorf("%w: L3 请求目标: %w", ErrInvalidPermissionRequest, err)
	}
	if !d.IsProtected(req.TargetRef) {
		return Decision{Allowed: true, Level: LevelIntegration, Basis: basisControlledIntegration}, nil
	}
	if !approval.valid() || approval.TargetRef != req.TargetRef {
		return Decision{}, fmt.Errorf("%w: %s 需要 Tally 域策略批准(Approval 与目标匹配且带 basis)", ErrProtectedTargetDenied, req.TargetRef)
	}
	return Decision{
		Allowed: true,
		Level:   LevelIntegration,
		Basis:   basisProtectedApprovedPrefix + approval.Basis,
	}, nil
}

// Authorize 对一个 L3 请求做决策,放行即签发绑定该目标的集成授权令牌。
// 低档请求不经此入口(低档没有令牌语义):用 Evaluate 做投影。
// runtime 调用方在本入口拿不到令牌(Evaluate 拒绝)—— 这就是「agent
// runtime 无法请求 L3 操作」的决策面:没有令牌就没有可调用的 L3 操作。
func (d *Decider) Authorize(c Caller, req Request, approval Approval) (Authorization, error) {
	if req.Level != LevelIntegration {
		return Authorization{}, fmt.Errorf("%w: Authorize 只签发 L3 集成授权,请求档位 %s 请用 Evaluate", ErrInvalidPermissionRequest, req.Level)
	}
	dec, err := d.Evaluate(c, req, approval)
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{targetRef: req.TargetRef, basis: dec.Basis}, nil
}

// requireAuthorization 在 L3 操作入口校验授权令牌(结构性执行点):
// 未携带/零值令牌 → ErrPermissionRequired;令牌绑定别的目标 →
// ErrPermissionDenied(令牌不可跨目标重用)。令牌字段不导出,包外无法
// 构造有效值;本包内的一切 L3 入口都必须经过这里。
func requireAuthorization(auth Authorization, targetRef string) error {
	if !auth.valid() {
		return fmt.Errorf("%w: 操作目标 %s 需要 Decider.Authorize 签发的 L3 集成授权", ErrPermissionRequired, targetRef)
	}
	if auth.targetRef != targetRef {
		return fmt.Errorf("%w: 授权绑定 %s,操作目标是 %s", ErrPermissionDenied, auth.targetRef, targetRef)
	}
	return nil
}
