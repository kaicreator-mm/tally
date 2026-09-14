// riskderive.go 是 v1.4.1 P1-02 的**权威风险输入推导**(A-02 / DL-05 / DL-06):
// 从冻结意图的确认事实(FrozenIntent)加上确定性任务事实(task_type 冻结词表)
// 推导 risk.Result,供合同编译消费(contract.Bind 的 riskFacts 输入,由 P1-03
// 从同一来源 bundle 装配)。
//
// 为什么推导入口在本包(FrozenIntent 域)而不是别处:internal/risk 刻意保持
// 标准库零依赖、basis 引用由调用方所有(见其包注释),不能反向依赖事实域;
// 本包已经依赖 risk(P0-01 裁决的 intentevidence → risk 边界),「哪些候选
// 进入推导」的裁定理应放在确认事实的 owner 侧。known/unknown 归类仍**完全**
// 由 risk.Derive 完成(DL-05:风险是派生物);本文件不新增任何归类语义。
//
// 信任边界(实现要求 1):候选 LLM 风险输出(intake.Draft.RiskSignals、推理
// prose)只有经 owner 确认并冻结进 risk_hints 才是证据(DL-06);本文件的推导
// 没有任何参数能让未确认候选旁路进入 —— 输入只有冻结证据本身与其 task_type
// 确定性事实。compile 侧(变更路径等执行期事实)不在这里:执行前不存在
// (INV-V14-02),候选变更上的信号复用属 P1-05 的匹配面。
//
// fail-safe(实现要求 2 / INV-V14-09):确认提示落在冻结信号枚举之外时,
// risk.Derive 以 Unknown 原样保真返回 —— 绝不丢弃、改名或按低风险解读;
// HasUnknown/AboveMinimal 使下游(策略匹配未覆盖 → assurance fail-safe 上调
// L3)不可能把 unknown 静默降级为最弱保证。
//
// 无持久化(实现要求 3 / DL-05):DeriveRisk 是纯函数,不落库、不建状态;
// 同一份证据(同 Revision)随时可从 pinned 提交重装载并确定性重算。
package intentevidence

import (
	"fmt"

	"tally/internal/risk"
)

// taskTypeBasisKind 是确定性任务事实候选的 basis kind(与 "intent:<task-id>"
// 同一 "<kind>:<id>" 约定)。与确认提示的 "intent:" 刻意区分:owner 确认的
// 声明与 task_type 词表的确定性推导保持可分辨的出处;两类候选产出同一信号时
// (如 db_migration 任务又确认了 schema_migration 提示),risk.Derive 合并为
// 排序去重的 basis 并集,两种出处都保留。
const taskTypeBasisKind = "tasktype"

// TaskTypeRiskCandidates 把冻结 task_type 的确定性风险事实投影为 risk.Derive
// 的输入候选(P1-02)。唯一的确定性映射:db_migration 闭环(PRD §24.2)按
// 定义就是数据库迁移任务 ⇒ 确定性蕴含 schema_migration 信号(PRD §13.1/
// §24.2,与 owner 是否额外确认该提示无关);bug_fix 不确定性地蕴含任何冻结
// 信号 ⇒ 零候选。basis 固定 "tasktype:<task-id>",出处可追溯到同一份冻结
// 证据。零值不是证据(返回 nil;DeriveRisk 对零值整体拒收)。
func (f FrozenIntent) TaskTypeRiskCandidates() []risk.Signal {
	if f.IsZero() || f.taskType != TaskTypeMigration {
		return nil
	}
	return []risk.Signal{{
		ID:    risk.SignalSchemaMigration,
		Basis: []string{taskTypeBasisKind + ":" + f.taskID},
	}}
}

// DeriveRisk 从冻结意图推导权威风险输入集(P1-02):确认风险提示
// (RiskSignalCandidates,basis "intent:<task-id>")加 task_type 确定性事实
// (TaskTypeRiskCandidates,basis "tasktype:<task-id>")一并交给 risk.Derive
// 做确定性归类、合并与排序。推导是纯函数:证据不可变,输入候选切片
// (RiskSignalCandidates 的返回值,本方法的私有拷贝)绝不被修改,任何输入
// 顺序都产出同一 Result;调用方改动返回值不影响证据、也不影响重算。
//
// 错误(fail-safe,调用方用 errors.Is 判类,绝不匹配文本):
//   - 零值接收者:零值不是证据(Freeze/Parse 的产物才携带内容),从非证据
//     推导出「无风险」就是静默低分类 —— 返回 ErrInvalidEvidence 拒收;
//   - risk.ErrMalformedSignal:任一候选结构性非法(缺/空 basis、空白信号
//     id)。合法证据的候选恒带 basis,该错误只可能来自构造 bug;推导链在
//     risk.Derive 入口按风险不变量**整体**拒收,绝不产出部分结果。
func (f FrozenIntent) DeriveRisk() (risk.Result, error) {
	if f.IsZero() {
		return risk.Result{}, fmt.Errorf("%w: 零值不是冻结意图证据,拒绝推导风险", ErrInvalidEvidence)
	}
	candidates := f.RiskSignalCandidates()
	candidates = append(candidates, f.TaskTypeRiskCandidates()...)
	return risk.Derive(candidates)
}
