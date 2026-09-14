// wiring.go 实现 P2-08 的包内组合根:把已落地的闭环域件组合成一次 Run 的
// 生产引擎 ——
//
//	WorkspaceExecutor(P2-08 执行适配:一次性 worktree + RepairBase 物化
//	  + 不透明内循环)
//	→ ServiceVerifier(P2-08 验证适配:candidateverify → verdict + 有界
//	  ERROR 复验 + FinalOutcome 账面)
//	→ FailureRouter(P2-03 NewFailureRouter,分类源 = FinalOutcome 账面
//	  经 ClassifyFinalStatus)
//	→ FeedbackSource(P2-08 反馈构造:P2-04 FeedbackBuilder,材料源 =
//	  FinalOutcome 账面 + 冻结契约)
//	→ RepairPolicy(P2-07 预算:位置式,默认 2)
//	→ LineageSink(P2-05 存储适配,可选)+ ExposureSink/种子(P2-06,可选)
//
// 全部域不变量(候选不可变、取代先于复验、预算位置式、取消一等、FAIL≠ERROR、
// unknown→升级、暴露谱系单调)仍由 Engine 与各域件机械强制:组合只接线,
// 不绕过、不复制任何裁决逻辑。没有编排框架、没有 DSL:一次 NewLoop = 一个
// 一次性 Engine(恰好驱动一个任务的一次闭环 Run;P2-09 负责把本组合接入
// cmd/worker 作业)。
package closedloop

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/contract"
	"tally/internal/evidence"
	"tally/internal/verification"
	"tally/internal/workspace"
)

// ErrInvalidLoopConfig:组合输入不合法(依赖缺席/修复策略非法)。
// errors.Is 判因,不做字符串匹配。
var ErrInvalidLoopConfig = errors.New("closedloop: 闭环组合配置不合法")

// LoopConfig 是 NewLoop 的装配面。Codegen / Verify / Contract / Runtime /
// Worktrees / BaseRevision / Repair 必填(缺席拒绝构造半成品闭环);
// Lineage / Exposure / Exposures / InfraAttempts / Now 的缺席落点与 Engine
// 的 Config 同语义(确定性 fail-safe,绝不静默放宽)。
type LoopConfig struct {
	// Codegen 是内层 codegen 执行缝(生产面:workflow.TaskExecution 的小
	// 适配,归 P2-09 接线)。
	Codegen CodegenRunner
	// Verify 是候选验证服务缝(生产面 *candidateverify.Service)。
	Verify CandidateVerification
	// Contract 是冻结验证契约:执行前已冻结,验证/复验/修复反馈共用同一份
	// (契约 pin 逐字不变,INV-02/03)。
	Contract *verification.Frozen
	// Runtime 是执行后端身份(candidateverify 的 verification_run 观测坐标;
	// 期望配置,不是观测事实)。
	Runtime string
	// Worktrees 是一次性 worktree 表(与执行侧共享一个 workspace.Manager)。
	Worktrees *workspace.RunTable
	// BaseRevision 解析首次尝试的工作区基线;修复尝试的基线由冻结前候选携带。
	BaseRevision BaseRevisionFunc
	// Repair 是外层修复预算(P2-07;必须合法 —— 零值/越界拒绝构造)。
	Repair RepairPolicy
	// Lineage 是取代事实出口;nil = 修复结构性不可用(Engine fail-safe)。
	Lineage LineageSink
	// Exposure 是蓄意披露事实的持久化出口;nil = Run 在途记账被拒绝
	// (exposure sink unavailable),种子事实仍参与 accept 守卫。
	Exposure ExposureSink
	// Exposures 是 Run 开始前已成立的暴露事实(append-only 投影,组合根
	// 从存储读出后注入)。
	Exposures []evidence.Exposure
	// Authority 是已编译合同权威(P1-06 验收守卫的必需档/未匹配风险来源;
	// 与 Contract 同一执行 bundle 的产物,DL-04)。非 nil 时组合根**恒**接线
	// CompiledAcceptanceGuard:FinalVerified 不再单独授权普通 Accept ——
	// 保证缺口/未匹配风险/豁免条款 fail-safe 升级(DL-07/A-03)。
	// nil = 未绑定验收权威(无编译权威的测试/试验组合保持 v1.4 行为);
	// 生产组合根(cmd/worker)自 P1-04 起编译权威必达,恒非 nil,生产路径
	// 的 no-bypass 由 P1-08 的编译权威门与本任务的 worker 接线测试钉住。
	Authority *contract.Compiled
	// AcceptSink 是 accepted outcome 事实的 append-only 出口(P6-01);非 nil
	// 时 accept 转移落成之前追加 kind=accepted 事件(写入失败显式升级,
	// 绝不静默接受)。nil = 未绑定出口(v1.4 行为;生产组合根 cmd/worker
	// 在有事实存储时恒绑定)。
	AcceptSink AcceptSink
	// InfraAttempts 是验证 ERROR 的有界复验次数(0 = 默认 2;界见 verify.go)。
	InfraAttempts int
	// Now 返回取代事实的时刻;nil 则用 time.Now。
	Now func() time.Time
}

// NewLoop 组装一个恰好可执行一次闭环 Run 的 Engine(P2-08 组合根;P2-09 把
// 本组合接入 worker 作业)。任何必填依赖缺席或修复策略非法都拒绝构造
// (ErrInvalidLoopConfig / ErrInvalidRepairPolicy / ErrInvalidExecutor /
// ErrInvalidVerifier),绝不产出半成品闭环。
func NewLoop(cfg LoopConfig) (*Engine, error) {
	switch {
	case cfg.Codegen == nil:
		return nil, fmt.Errorf("%w: Codegen 为 nil", ErrInvalidLoopConfig)
	case cfg.Verify == nil:
		return nil, fmt.Errorf("%w: Verify 为 nil", ErrInvalidLoopConfig)
	case cfg.Contract == nil:
		return nil, fmt.Errorf("%w: 冻结验证契约为 nil", ErrInvalidLoopConfig)
	case strings.TrimSpace(cfg.Runtime) == "":
		return nil, fmt.Errorf("%w: Runtime 为空", ErrInvalidLoopConfig)
	case cfg.Worktrees == nil:
		return nil, fmt.Errorf("%w: Worktrees 为 nil", ErrInvalidLoopConfig)
	case cfg.BaseRevision == nil:
		return nil, fmt.Errorf("%w: BaseRevision 解析为 nil", ErrInvalidLoopConfig)
	}
	if err := cfg.Repair.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLoopConfig, err)
	}

	exec, err := NewWorkspaceExecutor(WorkspaceExecutorConfig{
		Worktrees:    cfg.Worktrees,
		Runner:       cfg.Codegen,
		BaseRevision: cfg.BaseRevision,
	})
	if err != nil {
		return nil, err
	}
	verifier, err := NewServiceVerifier(ServiceVerifierConfig{
		Service:       cfg.Verify,
		Contract:      cfg.Contract,
		Runtime:       cfg.Runtime,
		InfraAttempts: cfg.InfraAttempts,
	})
	if err != nil {
		return nil, err
	}
	// 验收守卫(P1-06):编译权威在位 ⇒ 恒接线(构造失败拒绝组合,绝不产出
	// 半成品闭环);权威缺席 = 未绑定,Engine 侧守卫为 nil(v1.4 行为)。
	var guard AcceptGuard
	if cfg.Authority != nil {
		g, gErr := NewCompiledAcceptanceGuard(cfg.Authority, cfg.Contract)
		if gErr != nil {
			return nil, gErr
		}
		guard = g
	}
	return NewEngine(Config{
		Executor:       exec,
		Verifier:       verifier,
		Router:         verifier.FailureRouter(),
		RepairBudget:   cfg.Repair,
		LineageSink:    cfg.Lineage,
		FeedbackSource: verifier.FeedbackSource(),
		ExposureSink:   cfg.Exposure,
		Exposures:      cfg.Exposures,
		AcceptGuard:    guard,
		AcceptSink:     cfg.AcceptSink,
		Now:            cfg.Now,
	})
}
