// v141_contract_assembler.go 是 v1.4.1 P1-03 的**生产合同编译装配缝**(PRD §16:
// Frozen Contract → Compiler → Policy Match → Risk → Assurance →
// CompiledContract → Execution,Worker 不允许绕过 Compiler):把同源执行
// bundle(P1-01 LoadBundle 的产出)装配成权威的已编译合同 —— 复用既有权威,
// 不新建任何平行体系(DL-18):
//
//	contract.Compile  聚合钉住修订的 TaskContract + 冻结 VerificationContract
//	                  (internal/contract 的 pin 从内容推导,AC4 权威);
//	DeriveRisk        从冻结意图推导权威风险输入(P1-02 的 seam);
//	contract.Bind     编译策略匹配/保证档/必需证据/requires 上下文
//	                  (既有编译器,DL-04:不重写权威,只做纯投影)。
//
// fail-safe 裁决(P1-01 装载器显式留给 P1-03 的部分):
//   - 策略快照含**任何**被拒收的策略制品 → ErrPolicyRejections,停止执行。
//     依据:Runtime-active 策略就是 pinned 快照里的精确策略内容(DL-09/BI-01),
//     快照里存在却解析失败的策略意味着本次执行看到的规则集不完整 —— 带着
//     缺口执行就是策略假阴性(与 match 纪律同一裁决:非法策略绝不静默跳过)。
//     缺省裁决是阻断,绝不带拒收策略编译;
//   - 风险推导失败按 P1-02 的委托原样类型化穿透(零值意图
//     intentevidence.ErrInvalidEvidence / 候选缺 basis risk.ErrMalformedSignal):
//     invalid risk basis 停止执行,绝不从非证据推导出「无风险」。经装载器
//     进入的 bundle 不可能触发(Parse 已拒绝非法意图),该分支是装配器对
//     手工拼装 bundle 的防御边界;
//   - 来源提交不一致 → ErrSourceMismatch:bundle 携带的策略快照若来自另一
//     提交,DL-04 的单一来源在装配入口复核,混合来源拒绝编译(装载器产物
//     恒等,这里防的是绕过装载器的手工构造);
//   - 未覆盖 / unknown 风险信号**不**阻断:既有编译器语义(INV-V14-09)原样
//     保留 —— 信号经 Compiled.Coverage() 保持可见,全部条款的必需保证档
//     fail-safe 上调 L3,绝不静默选择最弱保证。零策略集合同理是合法的
//     ConfidenceNone 投影:任务带风险信号时必然未覆盖 → 同一 fail-safe
//     上调;不带信号时维持 L1 默认地板。阻断零策略仓库不是任何权威文档的
//     要求,本装配器不发明它。
//
// 确定性(任务 AC:inputs are reproducible from source commit):装配是
// bundle 的纯函数 —— 同一 bundle 重复装配得到同一 Compiled 哈希;同一提交
// 重新 LoadBundle 后装配同样可复现。任务 AC「No supported task starts
// execution without Compiled contract」在本缝的形态:唯一从 bundle 到执行
// 权威的入口就是 CompileAuthority,任何 fail-safe 触发都不产出 Compiled;
// Worker 侧的调用时序(先编译后开 workspace/runtime)归 P1-04。
package artifacts

import (
	"errors"
	"fmt"

	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
)

// 装配级类型化哨兵:调用方用 errors.Is 判类。contract.Compile/Bind 与
// DeriveRisk 的哨兵以 %w 原样穿透(两层判类都成立)。
var (
	// ErrPolicyRejections:bundle 的策略快照含被拒收的策略制品(fail-safe:
	// 拒收策略阻断执行,绝不带规则缺口编译)。
	ErrPolicyRejections = errors.New("artifacts: 策略快照含被拒收的策略制品")
	// ErrSourceMismatch:bundle 携带的策略快照来源提交与 bundle 来源提交
	// 不一致 —— 混合来源在装配入口 fail-closed(DL-04)。
	ErrSourceMismatch = errors.New("artifacts: 策略快照来源提交与 bundle 来源提交不一致")
)

// CompiledAuthority 是装配产出(PRD §16 的 CompiledContract 节点):一份从
// 单一同源提交装配出的已编译合同,加它的全部同源出处。P1-04 的 Worker 从
// 这里取执行权威并把身份/依据记入运行事实;不存在绕过编译器的执行入口。
type CompiledAuthority struct {
	// TaskID 是与 bundle 一致的任务身份。
	TaskID string
	// SourceCommit 是全部输入解析自的唯一 Git 提交(观测 sha,DL-04)。
	SourceCommit string
	// BundleIdentity 是被装配 bundle 的身份(ExecutionBundle.Identity)。
	BundleIdentity string
	// Risk 是权威风险输入(DeriveRisk 的派生结果;DL-05:派生值,不落库、
	// 不建状态,随时可从 pinned 提交确定性重算)。
	Risk risk.Result
	// Compiled 是已编译合同(contract.Bind 的产出):身份哈希
	// (Compiled.Hash)、pin(Compiled.Pins)、策略覆盖(Coverage)、每条款
	// 必需证据与保证档(Clauses)、冻结门禁(FrozenGates)、执行前 requires
	// 上下文(Requires/GateKeys)全部经它取用。
	Compiled *contract.Compiled
}

// CompileAuthority 把同源执行 bundle 装配成权威已编译合同(P1-03 生产缝):
// 来源复核 → 拒收策略裁决 → 权威风险推导 → 合同聚合(contract.Compile)→
// 合同编译(contract.Bind)。任一 fail-safe 触发都不产出 Compiled;错误按
// 上面的哨兵与穿透规则分类,调用方以 errors.Is 判类。
func (b *ExecutionBundle) CompileAuthority() (*CompiledAuthority, error) {
	// bundle 缺 intent 制品是手工构造错误(装载器产物恒有):类型化报错,
	// 不 panic。
	if b.Intent == nil {
		return nil, fmt.Errorf("artifacts: bundle 缺 intent 制品,拒绝装配")
	}
	// fail-safe:来源一致性(DL-04 入口复核)。装载器产物恒等,这里防的是
	// 绕过装载器手工拼装的混合来源 bundle。
	if b.Policies.SourceCommit != b.SourceCommit {
		return nil, fmt.Errorf("%w: bundle %s vs 策略快照 %s", ErrSourceMismatch, b.SourceCommit, b.Policies.SourceCommit)
	}
	// fail-safe:拒收策略阻断(实现要求 2;P1-01 留给本装配器的裁决)。
	if n := len(b.Policies.Rejected); n > 0 {
		first := b.Policies.Rejected[0]
		return nil, fmt.Errorf("%w: %d 份(首份 %s: %s)", ErrPolicyRejections, n, first.Path, first.Reason)
	}
	// 权威风险输入(P1-02):确认事实 + task_type 确定性事实 → risk.Derive。
	// 类型化失败原样穿透:invalid risk basis 停止执行。
	riskFacts, err := b.Intent.Intent.DeriveRisk()
	if err != nil {
		return nil, fmt.Errorf("artifacts: 权威风险推导: %w", err)
	}
	// 聚合(contract 权威,DL-18 复用):task/verification 来自同一提交树且
	// 身份已由装载器逐一对照;Compile 的内容校验(Validate/任务一致性)是
	// 第二道防线,其类型化哨兵原样穿透。
	snap, err := contract.Compile(b.Task.Contract, b.Verification.Frozen)
	if err != nil {
		return nil, fmt.Errorf("artifacts: 合同聚合: %w", err)
	}
	// 编译(既有 contract.Bind):策略集按装载顺序(Path 升序)传入,每条
	// 都带 Parse 填充的内容修订 pin;unmatched/unknown 风险经其 INV-V14-09
	// 语义处理(可见 + L3 上调),不在装配层改写。
	policies := make([]engineeringpolicy.Policy, 0, len(b.Policies.Policies))
	for _, a := range b.Policies.Policies {
		policies = append(policies, a.Policy)
	}
	compiled, err := contract.Bind(snap, policies, riskFacts)
	if err != nil {
		return nil, fmt.Errorf("artifacts: 合同编译: %w", err)
	}
	return &CompiledAuthority{
		TaskID:         b.TaskID,
		SourceCommit:   b.SourceCommit,
		BundleIdentity: b.Identity(),
		Risk:           riskFacts,
		Compiled:       compiled,
	}, nil
}
