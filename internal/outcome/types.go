// Package outcome 实现 PRD §23 OutcomeLog 的 v1.4 事件事实模型(P3-05):
// accepted / reverted / hotfixed / regression_found / human_correction 五类
// outcome 从发生时点起立即捕获为 append-only 事件 —— 这些数据事后无法完整
// 补采,一个 revert 发生过就是发生过,没有采集点它就永远消失。
//
// 边界(P3-05,决策锁 14):
//   - outcome 是 append-only 事件捕获,不是任务上的可变布尔/状态列;本包
//     只定义事件类型、封闭枚举与确定性校验,不存在任何「当前 outcome」的
//     可变权威 —— 当前结论永远是事件流的读侧投影,本包不维护、不缓存
//     (AC5:派生结论不得变成可变权威,同 INV-STATE-02 的证据先例)。
//   - 不建 Outcome UI、不做分析平台聚合(§23:v1.4 只做耐久捕获);跨事件
//     的报表推导归后续读侧任务,不在事实面上加列。
//   - WorkItem 重写按决策锁 13 推迟:事件归属用现有 TaskContract 身份
//     (task_id),changeset_id 指向被验收/回退/热修的候选。两层口径:
//     数据库(0023)两列都可空 —— 绑定可得时必填,采集点先于归档完善,
//     拒绝落行等于永久丢失无法补采的事实;本包(闭环写入路径)要求至少
//     绑定其一 —— 完全无归属的 outcome 无法参与任何闭环审计,静默收下
//     孤儿事件就是把不可审计的行伪装成已归属事实(fail-safe)。
package outcome

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kind 是一次 outcome 的事件类别(PRD §23 最小集,封闭枚举,AC1)。
// 迁移 0023 的 CHECK 在数据库层限定同一集合;未知 kind 的写入两侧都被
// 显式拒绝(fail-safe),不被静默折叠进任何一个已知 kind。
type Kind string

const (
	// KindAccepted:候选被接受(验收闭环终点)。
	KindAccepted Kind = "accepted"
	// KindReverted:已接受的候选被回退。
	KindReverted Kind = "reverted"
	// KindHotfixed:事故热修(绕过常规候选闭环的修复)。
	KindHotfixed Kind = "hotfixed"
	// KindRegressionFound:已接受的候选之后发现回归。
	KindRegressionFound Kind = "regression_found"
	// KindHumanCorrection:人工修正(人工改写了 agent 产出)。
	KindHumanCorrection Kind = "human_correction"
)

// ErrUnknownKind 表示遇到了 PRD §23 五个 kind 之外的 outcome 事件类别。
// 解析/重建未知 kind 一律 fail-safe 返回该哨兵(errors.Is 可判),绝不
// 静默映射到最弱的已知 kind —— 未知的 outcome 不是「大概是 accepted」。
var ErrUnknownKind = errors.New("outcome: unknown event kind")

// IsValid 报告该值是否是 PRD §23 定义的五个 kind 之一。空串非法:kind
// 必须显式写入,不存在「默认 outcome」。
func (k Kind) IsValid() bool {
	switch k {
	case KindAccepted, KindReverted, KindHotfixed, KindRegressionFound, KindHumanCorrection:
		return true
	}
	return false
}

// ParseKind 把外部输入(存储列、事件载荷等)解析为 Kind。未知值返回
// 包装 ErrUnknownKind 的错误(errors.Is 可判),不猜、不折叠。
func ParseKind(s string) (Kind, error) {
	k := Kind(strings.TrimSpace(s))
	if !k.IsValid() {
		return "", fmt.Errorf("outcome: kind %q 非法: %w (必须是 accepted|reverted|hotfixed|regression_found|human_correction 之一, PRD §23)", s, ErrUnknownKind)
	}
	return k, nil
}

// Event 是一次 outcome 事实(PRD §23 的 OutcomeLog 条目)。全部字段在
// 事件发生时写入,之后只读(append-only;migrations/0023 的语句级触发器
// 是数据库侧防线,本包不提供任何改写 API 是第一道防线)。
type Event struct {
	// TaskID 归属任务(现有 TaskContract 身份,决策锁 13;PRD §23 的
	// work_item_id 归 WorkItem 重写一并推迟)。可得时必填;上下文外事件
	// (如事故热修)允许为空 —— 先落事实,补齐归属好过丢事件。
	TaskID string
	// ChangesetID 指向被验收/回退/热修的候选 ChangeSet;可得时必填,可空。
	ChangesetID string
	// Kind 是 §23 的五个 outcome 类别之一(AC1,封闭枚举)。
	Kind Kind
	// RecordedAt 是事件写入时间(§23 的 timestamp)。必须显式给出:
	// 缺省时间会把「何时发生的」变成「何时补记的」的猜测。
	RecordedAt time.Time
}

// Validate 校验事件的硬约束:
//
//   - kind 必须是五个封闭枚举之一(AC1);
//   - RecordedAt 必须显式给出;
//   - task_id / changeset_id 可空(绑定可得时必填),但至少其一非空 ——
//     完全无归属的 outcome 无法参与任何闭环审计;且坐标不得是纯空白,
//     空白坐标与缺失同义,不得伪装成已归属。
//
// 事件没有「数值」类字段,因此不存在负值/假 0 问题;也不存在任何状态列。
func (e Event) Validate() error {
	if !e.Kind.IsValid() {
		return fmt.Errorf("outcome: event 的 kind %q 非法,必须是 PRD §23 定义的五个 outcome 之一", string(e.Kind))
	}
	taskID, changesetID := strings.TrimSpace(e.TaskID), strings.TrimSpace(e.ChangesetID)
	if taskID == "" && changesetID == "" {
		return fmt.Errorf("outcome: event 缺少 task_id 与 changeset_id(至少绑定其一;完全无归属的 outcome 无法参与任何闭环审计)")
	}
	if (e.TaskID != "" && taskID == "") || (e.ChangesetID != "" && changesetID == "") {
		return fmt.Errorf("outcome: event 的 task_id/changeset_id 不得为纯空白(空白坐标与缺失同义)")
	}
	if e.RecordedAt.IsZero() {
		return fmt.Errorf("outcome: event 缺少 recorded_at(outcome 事实必须可定位到发生时间)")
	}
	return nil
}
