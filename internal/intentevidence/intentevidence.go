// Package intentevidence 承载 v1.4.1 的 FrozenIntent 证据(A-02 / DL-06):
// owner 确认的 intake 事实,以**不可变**形式穿过 Freeze 存活下来,供下游风险
// 推导确定性消费(P0-01:immutable FrozenIntent evidence)。
//
// 边界与不变量:
//
//   - 证据只记录「确认了什么」:任务身份、任务类型、来源引用、确认声明的
//     风险提示四类字段,此外无任何字段(DL-06;任务合同实现要求 1)。
//     provider/model/推理 prose 一律不是权威,不得进入 schema(要求 3):
//     schema 之外的字段在解析时按未知字段 fail-closed 拒收(artifact.go)。
//   - FrozenIntent 在**类型层面**不可变:字段全部非导出,没有任何 setter;
//     唯一构造路径是 Freeze(从确认事实)与 Parse(从制品字节),两者都先
//     做 fail-closed 校验、再从内容计算修订哈希。零值不是证据(见 IsZero)。
//     冻结之后不存在任何改写内容的通道 —— 修订只能来自新一次确认/新冻结。
//   - 风险仍然是派生物(DL-05):本包只原样保存确认的声明提示(risk hints),
//     不做已知/未知判定;RiskSignalCandidates 把提示逐条投影成 risk.Signal
//     候选(固定 "intent:<task-id>" basis),已知/未知归类由 risk.Derive
//     完成。枚举之外的提示原样保留、绝不静默消失(INV-V14-09 fail-safe)。
//     权威风险输入推导入口(P1-02 DeriveRisk,含 task_type 确定性事实投影
//     TaskTypeRiskCandidates)也在本包:它只组装候选,归类仍完全归
//     risk.Derive(见 riskderive.go)。
//   - Git 制品身份(seam 裁决,owner = 本任务 P0-01):精确路径
//     .tally/intents/<task_id>.yaml —— 扁平布局、每任务一份 *.yaml,与
//     .tally/tasks、.tally/verifications、.tally/policies 同构;身份以内容
//     为准(task_id + 内容修订哈希),不来自文件名。schema 见 artifact.go。
//
// 依赖保持最小:标准库 + yaml.v3 + internal/risk(纯派生模型,无内部依赖);
// 不依赖 intake/airuntime —— 确认事实经 Facts 显式进入本包,Draft→Facts 的
// 映射归 P0-06,Git 装载归 P0-02。
package intentevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tally/internal/risk"
)

// TaskType 是 FrozenIntent 冻结的任务类型词表,恰为 intake 的两个 MVP 闭环
// 类型(PRD §24.1 / §24.2,AC4)。取值是冻结证据 schema 的一部分,保持稳定;
// 词表之外的值(含空串、大小写变体)在 Freeze/Parse 时一律 fail-closed 拒绝,
// 不猜测、不折叠。刻意不复用 intake.TaskType:本包不依赖 AI Runtime 输入侧,
// 两侧词表的一致性由测试钉住(intentevidence_test.go)。
type TaskType string

const (
	// TaskTypeBugFix 是缺陷修复闭环(PRD §24.1)。
	TaskTypeBugFix TaskType = "bug_fix"
	// TaskTypeMigration 是数据库迁移闭环(PRD §24.2)。
	TaskTypeMigration TaskType = "db_migration"
)

// IsValid 报告该值是否是冻结词表的两个合法值之一。精确匹配:无默认值、
// 不做大小写/空白归一(那是 Freeze/Parse 校验失败,不是合法数据)。
func (t TaskType) IsValid() bool {
	switch t {
	case TaskTypeBugFix, TaskTypeMigration:
		return true
	}
	return false
}

// Facts 是进入冻结的确认事实:owner 确认后、Freeze 之前的唯一输入形状。
// 与 FrozenIntent 一一对应,但字段可变、未校验 —— 只有经 Freeze 校验并计算
// 修订哈希后才成为证据。RiskHints 顺序不承载语义(规范化为排序去重)。
type Facts struct {
	// TaskID 是目标任务身份(如 TASK-0023),必填,不得含空白字符。
	TaskID string
	// TaskType 是冻结的任务类型,必须是两个 MVP 值之一。
	TaskType TaskType
	// SourceRef 是可选的外部来源引用(与 intake.Intent.SourceRef 同语义);
	// 空/纯空白视为未声明。
	SourceRef string
	// RiskHints 是确认声明保留的风险提示原文(如 schema_migration)。
	// 允许为空(确认了零条提示是合法证据);空白条目是写错了,不是数据。
	RiskHints []string
}

// FrozenIntent 是一份不可变的冻结意图证据(A-02 / DL-06)。字段全部非导出,
// 只能经 Freeze / Parse 构造;所有取值方法是只读投影,集合类取值返回副本,
// 外部改动不可能穿透回证据本身。
type FrozenIntent struct {
	taskID    string
	taskType  TaskType
	sourceRef string
	riskHints []string
	revision  string // 内容修订哈希("sha256:<hex>"),Freeze/Parse 时计算
}

// 类型化错误的哨兵值:调用方用 errors.Is/errors.As 判类,不匹配错误文本。
var (
	// ErrInvalidEvidence:证据内容结构性非法(任务身份缺失/含空白、空白提示
	// 条目、未知 schema 字段、多文档 YAML 等)。fail closed:宁可拒绝,不猜测。
	ErrInvalidEvidence = errors.New("intentevidence: 冻结意图证据非法")
	// ErrUnsupportedTaskType:任务类型不在冻结词表(bug_fix/db_migration)内。
	// 与 intake 的 AC4 同纪律:超出集合一律拒绝,不静默降级。
	ErrUnsupportedTaskType = errors.New("intentevidence: 任务类型不在冻结词表内(bug_fix/db_migration)")
)

// Freeze 校验确认事实并冻结为不可变证据。同一份事实无论以什么顺序、什么
// 排版给出,产出同一份证据(同一 Revision);任何实质差异都会改变 Revision。
// 校验失败返回类型化错误,绝不产出「部分冻结」的中间态。
func Freeze(f Facts) (FrozenIntent, error) {
	fi, err := normalize(f.TaskID, f.TaskType, f.SourceRef, f.RiskHints)
	if err != nil {
		return FrozenIntent{}, err
	}
	fi.revision = fi.computeRevision()
	return fi, nil
}

// normalize 应用冻结 schema 的确定性规范化并做 fail-closed 校验。
// 规范化只折叠「排版」(首尾空白、内部连续空白),不折叠语义:
//
//   - task_id:去首尾空白后必填,且不得含任何内部空白字符(身份里带空白
//     是写错了,不是数据);
//   - task_type:精确匹配冻结词表,无默认值;
//   - source_ref:折叠空白;空/纯空白 = 未声明(排版,不是内容丢失);
//   - risk_hints:逐条折叠空白;折叠后为空的条目报错;去重 + 排序(确认
//     同一条提示两次 = 同一条事实)。
func normalize(taskID string, taskType TaskType, sourceRef string, riskHints []string) (FrozenIntent, error) {
	fi := FrozenIntent{}

	id := strings.TrimSpace(taskID)
	switch {
	case id == "":
		return FrozenIntent{}, fmt.Errorf("%w: 缺少 task_id", ErrInvalidEvidence)
	case strings.ContainsAny(id, " \t\r\n"):
		return FrozenIntent{}, fmt.Errorf("%w: task_id %q 含空白字符", ErrInvalidEvidence, taskID)
	}
	fi.taskID = id

	tt := TaskType(strings.TrimSpace(string(taskType)))
	if !tt.IsValid() {
		return FrozenIntent{}, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, taskType)
	}
	fi.taskType = tt

	fi.sourceRef = collapseSpace(sourceRef)

	hints := make([]string, 0, len(riskHints))
	seen := make(map[string]bool, len(riskHints))
	for _, h := range riskHints {
		c := collapseSpace(h)
		if c == "" {
			return FrozenIntent{}, fmt.Errorf("%w: risk_hints 含空白条目", ErrInvalidEvidence)
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		hints = append(hints, c)
	}
	sort.Strings(hints)
	if len(hints) == 0 {
		hints = nil
	}
	fi.riskHints = hints
	return fi, nil
}

// TaskID 返回冻结的任务身份。
func (f FrozenIntent) TaskID() string { return f.taskID }

// TaskType 返回冻结的任务类型。
func (f FrozenIntent) TaskType() TaskType { return f.taskType }

// SourceRef 返回冻结的外部来源引用;未声明时为空串。
func (f FrozenIntent) SourceRef() string { return f.sourceRef }

// RiskHints 返回确认声明的风险提示(规范化后的排序去重副本)。调用方改动
// 返回值不影响证据本身。
func (f FrozenIntent) RiskHints() []string {
	if len(f.riskHints) == 0 {
		return nil
	}
	out := make([]string, len(f.riskHints))
	copy(out, f.riskHints)
	return out
}

// Revision 返回内容修订哈希("sha256:<hex>")。它由证据内容唯一决定
// (见 Canonical),不来自制品声明;内容任何实质变化都会改变它。
// 零值的 Revision 为空串。
func (f FrozenIntent) Revision() string { return f.revision }

// IsZero 报告该值是否是零值。零值不是证据:只有 Freeze/Parse 的返回值才
// 携带内容与修订哈希。
func (f FrozenIntent) IsZero() bool { return f.revision == "" }

// Equal 报告两份证据内容是否相同。逐字段比较,不依赖哈希碰撞假设。
func (f FrozenIntent) Equal(o FrozenIntent) bool {
	if f.taskID != o.taskID || f.taskType != o.taskType || f.sourceRef != o.sourceRef {
		return false
	}
	if len(f.riskHints) != len(o.riskHints) {
		return false
	}
	for i := range f.riskHints {
		if f.riskHints[i] != o.riskHints[i] {
			return false
		}
	}
	return true
}

// RiskSignalCandidates 把确认的风险提示逐条投影为 risk.Derive 的输入候选,
// basis 固定为 "intent:<task-id>"(risk 包约定的 "<kind>:<id>" 格式)——
// 下游风险推导的 basis 引用格式由本包统一裁定,消费方不再各自发明。
// 提示原样进入候选:是否属于冻结信号枚举由 risk.Derive 判定(DL-05),
// 枚举之外的提示以 Unknown 形式保持可见,绝不静默消失或被当作低风险
// (INV-V14-09)。返回切片与内部状态隔离。
func (f FrozenIntent) RiskSignalCandidates() []risk.Signal {
	if len(f.riskHints) == 0 {
		return nil
	}
	basis := []string{"intent:" + f.taskID}
	out := make([]risk.Signal, 0, len(f.riskHints))
	for _, h := range f.riskHints {
		out = append(out, risk.Signal{ID: risk.SignalID(h), Basis: basis})
	}
	return out
}

// collapseSpace 把字符串内所有空白折叠为单个空格并去首尾(与 taskcontract
// 同一纪律):同一引用的多种等价排版(YAML 折叠、换行、多空格)是排版,
// 不是内容差异。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// computeRevision 计算 "sha256:<hex>" 形式的内容修订哈希。唯一输入是
// Canonical 文本,与时间、进程、机器、YAML 排版无关。
func (f FrozenIntent) computeRevision() string {
	sum := sha256.Sum256([]byte(f.Canonical()))
	return "sha256:" + hex.EncodeToString(sum[:])
}
