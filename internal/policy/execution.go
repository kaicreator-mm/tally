// Package policy 定义 ExecutionPolicy:Tally 控制面对执行边界的唯一约束权威
// (PRD §11.2)。retry 次数、token/墙钟预算、网络、shell、凭据、停止条件与
// runtime/model 升级策略都在这里定值。
//
// 边界(PRD §11.2 / §32):
//   - TaskContract / BehaviorManifest / AgentProfile 只能通过 Proposal 提出
//     需求或偏好;Resolve 只接受「收紧」,任何放宽都以 ErrWidening 拒绝。
//   - 硬下限(hard minima)对任何来源(含 owner 直写)生效:编码运行默认断网、
//     shell 结构化且默认关闭、预算必须有限、三条硬停止条件不可移除、凭据只有
//     capability-scoped 引用一种形态 —— P1 类型里不存在「生产凭据」开关,
//     生产凭据暴露在类型上就不可表达。
//
// 升级终止 CodegenRun / 新 Run 记录 escalated_from 的生命周期语义(PRD §11.2.1)
// 归 CodegenRun 域所有;本包只拥有「何时升级、升到哪」的策略值。
// 验证恢复阶梯(verification recovery stop boundary)按决策锁 12 属 P2,不在此建模。
//
// 本包只定义策略值类型与合并/校验规则,不执行任何进程、不发起任何网络请求。
package policy

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// 哨兵错误:调用方用 errors.Is 区分,包装文案只提供字段级上下文。
var (
	// ErrInvalidPolicy 表示策略值本身非法(预算非正、升级阈值超过重试预算、
	// 缺硬停止条件等)。对任何来源的策略都生效,是硬下限的执行点。
	ErrInvalidPolicy = errors.New("policy: invalid execution policy")
	// ErrWidening 表示 Proposal 试图放宽当前有效策略(tighten-only 被拒)。
	ErrWidening = errors.New("policy: proposal widens execution policy")
)

// NetworkMode 是执行的网络访问模式。
type NetworkMode string

const (
	// NetworkDeny 完全断网。编码运行的默认值与硬下限(PRD §32.1)。
	NetworkDeny NetworkMode = "deny"
	// NetworkAllowHosts 只允许显式列出的主机。
	NetworkAllowHosts NetworkMode = "allow_hosts"
)

// NetworkPolicy 约束一次执行能触达哪些网络端点。
type NetworkPolicy struct {
	Mode         NetworkMode
	AllowedHosts []string // 仅 Mode=NetworkAllowHosts 时有效;形如 "host" 或 "host:port"
}

// ShellPolicy 约束 shell 语义。结构化命令(argv,见 internal/command.CommandSpec)
// 是唯一默认形态;AllowShell=true 只能由 owner 直写在基础策略里显式开启,
// Proposal 永远不能开启(决策锁 6:shell 语义显式且默认拒绝)。
type ShellPolicy struct {
	AllowShell bool
}

// CredentialPolicy 约束一次执行可见的凭据。凭据只有 capability-scoped 引用一种
// 形态:控制面授予具名引用,执行侧拿不到原始 secret 值。「生产凭据不暴露给普通
// 编码 agent」(PRD §32.1)由类型形状保证 —— 没有可置位的生产作用域。
type CredentialPolicy struct {
	ScopedSecrets []string // capability-scoped secret 引用名;空 = 不授予任何凭据
}

// RetryPolicy 约束同一 CodegenRun 内的重试次数。必须有限(>=1)。
type RetryPolicy struct {
	MaxIterations int
}

// Budgets 约束一次执行的资源上限。两项都必须有限且为正。
type Budgets struct {
	MaxTokens   int64
	MaxWallTime time.Duration
}

// EscalationPolicy 描述失败迭代达到阈值后的 runtime/model/behavior 升级。
// 升级动作的生命周期语义(PRD §11.2.1:终止当前 Run、新 Run 记录 escalated_from)
// 归 CodegenRun 域;本包只拥有阈值与目标 profile。
type EscalationPolicy struct {
	AfterFailedIterations int
	TargetProfile         string
}

// StopCondition 是触发立即停止当前执行的条件名。三条硬停止条件(PRD §11.2 示例)
// 属于硬下限:任何策略都必须包含;自定义条件只允许追加,移除在 Proposal 类型上
// 不可表达。
type StopCondition string

const (
	StopProtectedEvidenceModified        StopCondition = "protected_evidence_modified"
	StopForbiddenPathModified            StopCondition = "forbidden_path_modified"
	StopGovernanceProfileUpgradeRequired StopCondition = "governance_profile_upgrade_required"
)

// hardStopConditions 是任何有效策略都必须包含的硬停止条件集合。
var hardStopConditions = []StopCondition{
	StopProtectedEvidenceModified,
	StopForbiddenPathModified,
	StopGovernanceProfileUpgradeRequired,
}

// ExecutionPolicy 是执行边界的唯一权威值类型(PRD §11.2)。按值使用;
// 构造后必须通过 Validate 才能视为有效。
type ExecutionPolicy struct {
	Network        NetworkPolicy
	Shell          ShellPolicy
	Credentials    CredentialPolicy
	Retry          RetryPolicy
	Budgets        Budgets
	Escalation     EscalationPolicy
	StopConditions []StopCondition
}

// DefaultExecutionPolicy 返回 P1 编码运行的保守默认策略,数值取 PRD §11.2 示例:
// 重试 4 次、180k token、30m 墙钟、失败 2 轮后升级到 strong-coding;
// 网络默认 deny(AC1)、shell 默认关闭(AC2)、不授予任何凭据、预算有限(AC3),
// 并包含全部硬停止条件。
func DefaultExecutionPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Network:        NetworkPolicy{Mode: NetworkDeny},
		Shell:          ShellPolicy{AllowShell: false},
		Credentials:    CredentialPolicy{},
		Retry:          RetryPolicy{MaxIterations: 4},
		Budgets:        Budgets{MaxTokens: 180000, MaxWallTime: 30 * time.Minute},
		Escalation:     EscalationPolicy{AfterFailedIterations: 2, TargetProfile: "strong-coding"},
		StopConditions: slices.Clone(hardStopConditions),
	}
}

// Validate 校验策略自身:预算有限且为正、升级阈值不超过重试预算、网络模式自洽、
// 停止条件包含全部硬条件。对任何来源的策略生效 —— 这是「策略可以收紧,但硬下限
// 不可削弱」(AC4)对 owner 直写路径的执行点。
func (p ExecutionPolicy) Validate() error {
	if err := validateNetwork(p.Network); err != nil {
		return err
	}
	if err := validateCredentials(p.Credentials); err != nil {
		return err
	}
	if p.Retry.MaxIterations < 1 {
		return fmt.Errorf("%w: retry.max_iterations=%d, 必须 >=1(预算必须有限)", ErrInvalidPolicy, p.Retry.MaxIterations)
	}
	if p.Budgets.MaxTokens < 1 {
		return fmt.Errorf("%w: budgets.max_tokens=%d, 必须 >=1", ErrInvalidPolicy, p.Budgets.MaxTokens)
	}
	if p.Budgets.MaxWallTime <= 0 {
		return fmt.Errorf("%w: budgets.max_wall_time=%v, 必须为正", ErrInvalidPolicy, p.Budgets.MaxWallTime)
	}
	if p.Escalation.AfterFailedIterations < 1 {
		return fmt.Errorf("%w: escalation.after_failed_iterations=%d, 必须 >=1", ErrInvalidPolicy, p.Escalation.AfterFailedIterations)
	}
	if p.Escalation.TargetProfile == "" {
		return fmt.Errorf("%w: escalation.target_profile 为空", ErrInvalidPolicy)
	}
	if p.Escalation.AfterFailedIterations > p.Retry.MaxIterations {
		return fmt.Errorf("%w: escalation.after_failed_iterations=%d 超过 retry.max_iterations=%d, 升级永远不会触发",
			ErrInvalidPolicy, p.Escalation.AfterFailedIterations, p.Retry.MaxIterations)
	}
	return validateStopConditions(p.StopConditions)
}

// Proposal 是 TaskContract / BehaviorManifest / AgentProfile 等提出的执行策略叠加
// 请求(PRD §11.2:它们只能提出需求或偏好,不能拥有或放宽同类限制)。指针字段为
// nil 表示不改动该项;非 nil 的子策略必须完整合法,且相对当前有效策略只能收紧。
// StopConditions 只允许追加 —— 移除在类型上不可表达,这是硬停止条件的形状。
type Proposal struct {
	Network        *NetworkPolicy
	Shell          *ShellPolicy
	Credentials    *CredentialPolicy
	Retry          *RetryPolicy
	Budgets        *Budgets
	Escalation     *EscalationPolicy
	StopConditions []StopCondition // 追加;与现有重复的项忽略
}

// Resolve 把 proposal 合并进当前策略,返回新的有效策略。任何放宽(预算上调、
// 网络放行或新增主机、开启 shell、扩大凭据、推迟升级)返回 ErrWidening;
// proposal 子策略非法返回 ErrInvalidPolicy;合并结果再过 Validate,
// 违反硬下限同样返回 ErrInvalidPolicy。出错时返回零值策略。
func (p ExecutionPolicy) Resolve(proposal Proposal) (ExecutionPolicy, error) {
	out := p

	if pr := proposal.Network; pr != nil {
		if err := validateNetwork(*pr); err != nil {
			return ExecutionPolicy{}, err
		}
		switch {
		case pr.Mode == NetworkDeny:
			// deny 相对任何当前模式都是收紧或持平,直接接受。
		case pr.Mode == NetworkAllowHosts && p.Network.Mode == NetworkAllowHosts:
			for _, h := range pr.AllowedHosts {
				if !slices.Contains(p.Network.AllowedHosts, h) {
					return ExecutionPolicy{}, fmt.Errorf("%w: 网络放行新增主机 %q", ErrWidening, h)
				}
			}
		default:
			return ExecutionPolicy{}, fmt.Errorf("%w: 网络 %q→%q", ErrWidening, p.Network.Mode, pr.Mode)
		}
		out.Network = *pr
	}

	if pr := proposal.Shell; pr != nil {
		if pr.AllowShell && !p.Shell.AllowShell {
			return ExecutionPolicy{}, fmt.Errorf("%w: shell 不允许由 proposal 开启", ErrWidening)
		}
		out.Shell = *pr
	}

	if pr := proposal.Credentials; pr != nil {
		if err := validateCredentials(*pr); err != nil {
			return ExecutionPolicy{}, err
		}
		for _, s := range pr.ScopedSecrets {
			if !slices.Contains(p.Credentials.ScopedSecrets, s) {
				return ExecutionPolicy{}, fmt.Errorf("%w: 凭据引用 %q 不在当前授予集合内", ErrWidening, s)
			}
		}
		out.Credentials = *pr
	}

	if pr := proposal.Retry; pr != nil {
		if pr.MaxIterations < 1 {
			return ExecutionPolicy{}, fmt.Errorf("%w: retry.max_iterations=%d, 必须 >=1", ErrInvalidPolicy, pr.MaxIterations)
		}
		if pr.MaxIterations > p.Retry.MaxIterations {
			return ExecutionPolicy{}, fmt.Errorf("%w: retry.max_iterations %d→%d 上调", ErrWidening, p.Retry.MaxIterations, pr.MaxIterations)
		}
		out.Retry = *pr
	}

	if pr := proposal.Budgets; pr != nil {
		if pr.MaxTokens < 1 {
			return ExecutionPolicy{}, fmt.Errorf("%w: budgets.max_tokens=%d, 必须 >=1", ErrInvalidPolicy, pr.MaxTokens)
		}
		if pr.MaxWallTime <= 0 {
			return ExecutionPolicy{}, fmt.Errorf("%w: budgets.max_wall_time=%v, 必须为正", ErrInvalidPolicy, pr.MaxWallTime)
		}
		if pr.MaxTokens > p.Budgets.MaxTokens {
			return ExecutionPolicy{}, fmt.Errorf("%w: budgets.max_tokens %d→%d 上调", ErrWidening, p.Budgets.MaxTokens, pr.MaxTokens)
		}
		if pr.MaxWallTime > p.Budgets.MaxWallTime {
			return ExecutionPolicy{}, fmt.Errorf("%w: budgets.max_wall_time %v→%v 上调", ErrWidening, p.Budgets.MaxWallTime, pr.MaxWallTime)
		}
		out.Budgets = *pr
	}

	if pr := proposal.Escalation; pr != nil {
		if pr.AfterFailedIterations < 1 {
			return ExecutionPolicy{}, fmt.Errorf("%w: escalation.after_failed_iterations=%d, 必须 >=1", ErrInvalidPolicy, pr.AfterFailedIterations)
		}
		if pr.TargetProfile == "" {
			return ExecutionPolicy{}, fmt.Errorf("%w: escalation.target_profile 为空", ErrInvalidPolicy)
		}
		if pr.AfterFailedIterations > p.Escalation.AfterFailedIterations {
			return ExecutionPolicy{}, fmt.Errorf("%w: escalation.after_failed_iterations %d→%d 推迟升级",
				ErrWidening, p.Escalation.AfterFailedIterations, pr.AfterFailedIterations)
		}
		out.Escalation = *pr
	}

	if len(proposal.StopConditions) > 0 {
		merged := slices.Clone(out.StopConditions)
		for _, c := range proposal.StopConditions {
			if c == "" {
				return ExecutionPolicy{}, fmt.Errorf("%w: 追加的停止条件为空", ErrInvalidPolicy)
			}
			if !slices.Contains(merged, c) {
				merged = append(merged, c)
			}
		}
		out.StopConditions = merged
	}

	if err := out.Validate(); err != nil {
		return ExecutionPolicy{}, err
	}
	return out, nil
}

func validateNetwork(n NetworkPolicy) error {
	switch n.Mode {
	case NetworkDeny:
		if len(n.AllowedHosts) > 0 {
			return fmt.Errorf("%w: network.mode=deny 时不得携带 allowed_hosts", ErrInvalidPolicy)
		}
	case NetworkAllowHosts:
		if len(n.AllowedHosts) == 0 {
			return fmt.Errorf("%w: network.mode=allow_hosts 时 allowed_hosts 不能为空(空洞的放行等于静默 deny)", ErrInvalidPolicy)
		}
		for i, h := range n.AllowedHosts {
			if h == "" {
				return fmt.Errorf("%w: allowed_hosts[%d] 为空", ErrInvalidPolicy, i)
			}
		}
		if containsDuplicate(n.AllowedHosts) {
			return fmt.Errorf("%w: allowed_hosts 存在重复主机", ErrInvalidPolicy)
		}
	default:
		return fmt.Errorf("%w: 未知 network.mode=%q", ErrInvalidPolicy, n.Mode)
	}
	return nil
}

func validateCredentials(c CredentialPolicy) error {
	for i, s := range c.ScopedSecrets {
		if s == "" {
			return fmt.Errorf("%w: credentials.scoped_secrets[%d] 为空", ErrInvalidPolicy, i)
		}
	}
	if containsDuplicate(c.ScopedSecrets) {
		return fmt.Errorf("%w: credentials.scoped_secrets 存在重复引用", ErrInvalidPolicy)
	}
	return nil
}

func validateStopConditions(cs []StopCondition) error {
	for i, c := range cs {
		if c == "" {
			return fmt.Errorf("%w: stop_conditions[%d] 为空", ErrInvalidPolicy, i)
		}
	}
	if containsDuplicate(cs) {
		return fmt.Errorf("%w: stop_conditions 存在重复项", ErrInvalidPolicy)
	}
	for _, hard := range hardStopConditions {
		if !slices.Contains(cs, hard) {
			return fmt.Errorf("%w: 缺少硬停止条件 %q(不可移除)", ErrInvalidPolicy, hard)
		}
	}
	return nil
}

func containsDuplicate[T comparable](xs []T) bool {
	seen := make(map[T]struct{}, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			return true
		}
		seen[x] = struct{}{}
	}
	return false
}
