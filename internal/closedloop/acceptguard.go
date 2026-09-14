// acceptguard.go 实现 P1-06 的验收守卫缝(seam 裁决:守卫的**强制点**在
// Closed Loop Engine 的 accept 转移,不在 ServiceVerifier —— DL-01:闭环保有
// Accept 权威;P2-06 暴露守卫已确立「accept 时守卫 + fail-safe 升级」的先例;
// verdictForStatus 是验证权威的事实保真投影,把 FinalVerified 改映射为
// failed/inconclusive 会伪造候选失败/ERROR 事实)。守卫的**语义评估**是
// taskstate 的唯一纯评估器(EvaluateAcceptanceGuard),required/achieved 计算原
// 样委托 internal/assurance —— 本包只做权威绑定与裁决转译,不复制保证数学
// (Forbidden boundary)。
//
// 裁决语义(A-03 / DL-07):FinalVerified 只是验证事实,不是 Acceptance。
// 编译权威要求更高保证档(achieved < required)、存在未匹配风险信号或被豁免
// 条款时,候选不得成为普通 Accepted —— fail-safe 升级(EscalationAssurance-
// Insufficient),绝不静默保持、绝不自动豁免、绝不作为可修复候选失败重跑。
package closedloop

import (
	"context"
	"errors"
	"fmt"

	"tally/internal/changeset"
	"tally/internal/contract"
	"tally/internal/taskstate"
	"tally/internal/verification"
)

// ErrInvalidAcceptanceGuard:验收守卫构造输入不合法(已编译合同/冻结验证契约
// 缺席)。errors.Is 判因,不做字符串匹配。
var ErrInvalidAcceptanceGuard = errors.New("closedloop: 验收守卫构造不合法")

// AcceptGuardRequest 是 accept 转移上交给守卫的一次裁决请求:绑定被验收候选
// 与确认它的验证运行(basis 面),并携带验证编排记录的功能覆盖观察(事实面;
// 缺席 = 无观察事实,评估器按 L0 fail-safe 处理)。请求里没有任何 held-out
// 内容 —— 覆盖观察只有 acceptance/binding id 与布尔覆盖面。
type AcceptGuardRequest struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// Attempt 是产出该候选的尝试序号。
	Attempt AttemptNo
	// ChangeSet 是被验收的不可变候选快照身份。
	ChangeSet changeset.ChangeSetID
	// VerifyRun 是确认该候选的验证运行引用(与 Accepted basis 的 VerifyRun
	// 同一来源)。
	VerifyRun string
	// Coverage 是 FinalOutcome.Functional.Acceptances 的原样投影(验证事实,
	// 不是裁决;裁决归评估器)。
	Coverage []verification.AcceptanceCoverage
}

// AcceptGuardRuling 是守卫的类型化裁决:Allowed=false 时 Reason 必须非空
// (原样透传为升级明细,人类可读;输入里不存在 held-out 材料,不构成泄漏面)。
type AcceptGuardRuling struct {
	// Allowed 报告候选可否成为普通 Accepted。
	Allowed bool
	// Reason 是拒绝时的确定性明细(保证缺口/未匹配信号/豁免条款词面)。
	Reason string
}

// AcceptGuard 是验收守卫缝(小接口由消费方定义):Engine 在 VerdictVerified
// 的 accept 转移上、暴露守卫之后、EventVerified 之前咨询一次。实现必须是
// 纯函数(无 I/O、无时钟、确定性):守卫没有可取消的执行,ctx 仅随端口纪律
// 出现在签名上。缺席(nil)= 组合根未绑定验收权威(Engine 不替组合根裁决;
// 生产组合根 NewLoop 在编译权威在位时恒绑定,见 wiring.go)。
type AcceptGuard interface {
	AllowAccept(ctx context.Context, req AcceptGuardRequest) AcceptGuardRuling
}

// CompiledAcceptanceGuard 是验收守卫的生产实现:把已编译合同(必需档 +
// 未匹配风险视图,P1-03 CompileAuthority 产物)与冻结验证契约(binding 可见性
// 权威)绑定为一次纯验收裁决。immutable:构造后没有任何修改路径。
type CompiledAcceptanceGuard struct {
	compiled   *contract.Compiled
	visibility map[string]verification.VisibilityPolicy
}

// AcceptGuard 端口编译期钉住。
var _ AcceptGuard = (*CompiledAcceptanceGuard)(nil)

// NewCompiledAcceptanceGuard 构造编译权威验收守卫;输入不合法拒绝构造
// (ErrInvalidAcceptanceGuard)。frozen 提供 binding 可见性(与 compiled 必须
// 同一快照 —— 生产组合根两者都来自同一 ExecutionBundle,DL-04)。
func NewCompiledAcceptanceGuard(compiled *contract.Compiled, frozen *verification.Frozen) (*CompiledAcceptanceGuard, error) {
	switch {
	case compiled == nil:
		return nil, fmt.Errorf("%w: 已编译合同为 nil", ErrInvalidAcceptanceGuard)
	case frozen == nil:
		return nil, fmt.Errorf("%w: 冻结验证契约为 nil", ErrInvalidAcceptanceGuard)
	}
	vis := map[string]verification.VisibilityPolicy{}
	fc := frozen.Contract()
	for i := range fc.Acceptances {
		for _, b := range fc.Acceptances[i].EvidenceBindings {
			vis[b.ID] = b.VisibilityPolicy
		}
	}
	return &CompiledAcceptanceGuard{compiled: compiled, visibility: vis}, nil
}

// AllowAccept 实现 AcceptGuard:唯一语义评估器(taskstate)的转译 ——
// VERIFIED_LIMITED 语义(任何未消化缺口)一律不得普通接受。评估器错误
// (权威缺席等 misuse)按拒绝处理(fail-safe),绝不默认放行。
func (g *CompiledAcceptanceGuard) AllowAccept(_ context.Context, req AcceptGuardRequest) AcceptGuardRuling {
	d, err := taskstate.EvaluateAcceptanceGuard(taskstate.AcceptanceGuardInput{
		Compiled:   g.compiled,
		Coverage:   req.Coverage,
		Visibility: g.visibility,
		// v1.4.1 闭环内没有豁免通道:Waived 恒空(读模型侧的 waiver 投影
		// 不进入运行时裁决;豁免语义在评估器里保持单一属主)。
	})
	if err != nil {
		return AcceptGuardRuling{Allowed: false, Reason: "acceptance guard unavailable: " + err.Error()}
	}
	if d.Acceptable {
		return AcceptGuardRuling{Allowed: true}
	}
	return AcceptGuardRuling{Allowed: false, Reason: d.Detail}
}
