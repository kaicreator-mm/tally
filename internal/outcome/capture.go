// capture.go 实现 P6-02:reverted / hotfixed / regression_found / human_correction
// 四类既有 outcome 事件(P3-05 封闭枚举,DL-12)的显式捕获 API。
//
// 缝的裁决(P6-02,先于代码定案):当前源里这四类事件**不存在自动生产钩子**,
// 只有读侧消费者(v14_metrics §31.4 rework / §31.5 guardrail 投影读它们)。
// 逐一核过的候选产地与否定证据:
//
//   - reverted:deploy 回滚(jobs.KindDeployRollback → deploy.Deployer.Rollback)
//     是 project/env 维度的 deploy_run 记账,载荷里没有 task/changeset 身份,
//     无法归属成一次「已接受候选被回退」;gitwriter.revert 是 apply 失败的
//     事务清理,不是产品语义的回退。按 L3 失败纪律:不从同名/通用工作流失败
//     推断 reverted,一律不接。
//   - hotfixed:仓库内不存在任何热修路径(hotfix 是绕过常规候选闭环的带外修复,
//     v1.4.1 源没有这个动作)。
//   - regression_found:verification 的 regression gate 是验收前的门
//     (coverage_independence),验收后对已接受候选的回归复测不存在。
//   - human_correction:human_approval/waiver(§19)是方向裁决证据,不是对
//     agent 产出的人工改写;project 追溯图显式「无人工标注字段」。
//
// 因此按任务合同落**显式捕获 API**(不是伪造自动检测):每一类事件各有一个
// 专用捕获入口,kind 由入口固定(调用方给不出错 kind),权威生产者在类型文档
// 里逐类写明 —— 满足 AC「每个受支持事件有被记录的权威生产者」。
//
// **accepted 刻意不在本面暴露**:它的唯一写入方是闭环 Engine 的 accept 转移
// (P6-01 的 AcceptSink 缝,验收裁决先于落事实)。本文件不提供、未来也不得
// 提供 accepted 捕获入口 —— 否则会制造第二个验收落账权威(TestCaptureAPI-
// NeverExposesAccepted 钉住)。
//
// 失败政策(与 P6-01 同一口径的显式 fail-safe):捕获失败原样返回带坐标的
// 错误,「事实未落成」对调用方可见 —— 生产者绝不把一次捕获失败解释成任何
// 域结论(不是候选失败、不是裁决翻转),也绝不静默吞掉无法补采的 §23 事实。
//
// 幂等/重放:同一捕获调用的重试(坐标与发生时刻完全相同的重放)幂等返回
// 成功,不重复落行;发生时刻不同 = 另一次真实发生,必然追加(事件面没有
// 载荷区分实例,发生时刻就是实例坐标 —— 不同发生必须给不同时刻)。检查-
// 追加的原子性与 P6-01 同源:单写者纪律(捕获入口的调用方串行投递同一事实)。
package outcome

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RevertFact 是一次 reverted 事件的捕获事实:某个已接受候选被回退。
//
// 权威生产者:Tally 控制面的「回退已接受候选」动作(操作者授权的显式撤销;
// 未来任务接入生产链)。deploy_run 的 rolled_back 是环境维度的部署记账,
// 与本事件互不推导 —— 从部署回滚推断 reverted 是伪造事实(L3 失败纪律)。
type RevertFact struct {
	// TaskID / ChangesetID:回退发生在哪个任务的哪个候选上(至少绑定其一;
	// 两者可得时都必须给 —— 回退天然指向具体候选)。
	TaskID      string
	ChangesetID string
	// RecordedAt 是回退发生的时刻(显式必填;补记时刻不是发生时刻)。
	RecordedAt time.Time
}

// HotfixFact 是一次 hotfixed 事件的捕获事实:一次绕过常规候选闭环的事故热修。
//
// 权威生产者:事故响应方的显式带外修复登记(操作者在处置事故时落账)。
// 热修天然先于归档完善:允许只有 changeset 绑定、没有任务上下文(§23 动机,
// types.go 的两层口径);拒绝落行等于永久丢失无法补采的事实。
type HotfixFact struct {
	// TaskID 可空(带外修复可能无任务上下文);ChangesetID 指向热修产出的
	// 候选,可得时必填;至少绑定其一。
	TaskID      string
	ChangesetID string
	// RecordedAt 是热修事实的发生时刻(显式必填)。
	RecordedAt time.Time
}

// RegressionFact 是一次 regression_found 事件的捕获事实:某个已接受候选之后
// 被发现回归。
//
// 权威生产者:针对已接受候选的验收后复测/验证运行,观察到回归门失败(未来
// 的验收后验证任务接入)。验证面里的 regression gate 是验收前的门,与本事件
// 无关;从失败名字或通用工作流失败推断回归是伪造事实(L3 失败纪律)。
type RegressionFact struct {
	// TaskID / ChangesetID:哪个任务的哪个已接受候选出现回归(至少绑定其一)。
	TaskID      string
	ChangesetID string
	// RecordedAt 是回归被发现的时刻(显式必填)。
	RecordedAt time.Time
}

// HumanCorrectionFact 是一次 human_correction 事件的捕获事实:人工改写了
// agent 的产出。
//
// 权威生产者:控制面的人工修正登记(人改写已产出候选内容时显式落账)。§19
// 的 human_approval / waiver 是方向裁决证据(归 approval 域),改写产出才
// 是本事件 —— 从审批事实推断人工修正是伪造事实。
type HumanCorrectionFact struct {
	// TaskID / ChangesetID:哪个任务的哪个候选被人工改写(至少绑定其一)。
	TaskID      string
	ChangesetID string
	// RecordedAt 是人工修正的发生时刻(显式必填)。
	RecordedAt time.Time
}

// Validate 校验捕获事实的硬约束(与 Event.Validate 同一口径的镜像):
// 至少绑定 task/changeset 其一、坐标不得是纯空白(空白与缺失同义)、
// 发生时刻必须显式。kind 不在事实面上 —— 由捕获入口固定。
func validateCaptureFact(taskID, changesetID string, at time.Time) error {
	t, c := strings.TrimSpace(taskID), strings.TrimSpace(changesetID)
	if (taskID != "" && t == "") || (changesetID != "" && c == "") {
		return fmt.Errorf("outcome: capture 事实的 task_id/changeset_id 不得为纯空白(空白坐标与缺失同义)")
	}
	if t == "" && c == "" {
		return fmt.Errorf("outcome: capture 事实缺少 task_id 与 changeset_id(至少绑定其一;完全无归属的 outcome 无法参与任何闭环审计)")
	}
	if at.IsZero() {
		return fmt.Errorf("outcome: capture 事实缺少 recorded_at(发生时刻必须显式,补记时刻不是发生时刻)")
	}
	return nil
}

// Validate 实现 RevertFact 的校验。
func (f RevertFact) Validate() error {
	return validateCaptureFact(f.TaskID, f.ChangesetID, f.RecordedAt)
}

// Validate 实现 HotfixFact 的校验。
func (f HotfixFact) Validate() error {
	return validateCaptureFact(f.TaskID, f.ChangesetID, f.RecordedAt)
}

// Validate 实现 RegressionFact 的校验。
func (f RegressionFact) Validate() error {
	return validateCaptureFact(f.TaskID, f.ChangesetID, f.RecordedAt)
}

// Validate 实现 HumanCorrectionFact 的校验。
func (f HumanCorrectionFact) Validate() error {
	return validateCaptureFact(f.TaskID, f.ChangesetID, f.RecordedAt)
}

// RecordReverted 捕获一次 reverted 事实(kind 由本入口固定)。同一事实的
// 重放(坐标与时刻完全相同)幂等返回成功;时刻不同 = 另一次真实回退,追加。
func (s *Store) RecordReverted(ctx context.Context, f RevertFact) error {
	return s.capture(ctx, Event{
		TaskID: f.TaskID, ChangesetID: f.ChangesetID,
		Kind: KindReverted, RecordedAt: f.RecordedAt,
	}, f)
}

// RecordHotfixed 捕获一次 hotfixed 事实(kind 由本入口固定)。允许带外绑定
// (仅 changeset)。同一事实重放幂等;时刻不同 = 另一次热修,追加。
func (s *Store) RecordHotfixed(ctx context.Context, f HotfixFact) error {
	return s.capture(ctx, Event{
		TaskID: f.TaskID, ChangesetID: f.ChangesetID,
		Kind: KindHotfixed, RecordedAt: f.RecordedAt,
	}, f)
}

// RecordRegressionFound 捕获一次 regression_found 事实(kind 由本入口固定)。
// 同一事实重放幂等;时刻不同 = 另一次回归发现,追加(同一候选的多次回归是
// 多行事实,不是可变状态)。
func (s *Store) RecordRegressionFound(ctx context.Context, f RegressionFact) error {
	return s.capture(ctx, Event{
		TaskID: f.TaskID, ChangesetID: f.ChangesetID,
		Kind: KindRegressionFound, RecordedAt: f.RecordedAt,
	}, f)
}

// RecordHumanCorrection 捕获一次 human_correction 事实(kind 由本入口固定)。
// 同一事实重放幂等;时刻不同 = 另一次人工改写,追加。
func (s *Store) RecordHumanCorrection(ctx context.Context, f HumanCorrectionFact) error {
	return s.capture(ctx, Event{
		TaskID: f.TaskID, ChangesetID: f.ChangesetID,
		Kind: KindHumanCorrection, RecordedAt: f.RecordedAt,
	}, f)
}

// capture 是四类捕获入口的共用追加路径:先校验事实(不触库)→ 由数据库唯一
// 约束 + INSERT ON CONFLICT 原子裁决重放。旧的「先读事件流再 INSERT」存在
// TOCTOU 窗口:两个并发 writer 都可能读到“不存在”后各自追加。现在同一完整
// 事实坐标(kind+task+changeset+recorded_at)最多一行;冲突是成功重放。
func (s *Store) capture(ctx context.Context, e Event, f interface{ Validate() error }) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if _, err := s.RecordOnce(ctx, e); err != nil {
		return fmt.Errorf("outcome: 捕获 kind=%s task=%q changeset=%q @%s: %w",
			e.Kind, e.TaskID, e.ChangesetID, e.RecordedAt.UTC().Format(time.RFC3339), err)
	}
	return nil
}
