// feedback_source.go 实现 P2-08 组合层的 FeedbackSource(feedback.go 的组合
// 层端口):从已记录的 FinalOutcome(verify.go 的 outcomeLedger)与冻结验证
// 契约,经 P2-04 的 FeedbackBuilder 构造本次可信 FAIL 的安全 RepairFeedback
// 投影 —— ADR-0001 §4 的 allowlist 全量、且仅有 allowlist:
//
//   - 失败主体:acceptance id + 冻结契约语义文本(执行前对实现侧本就可见),
//     或失败 required gate 名 —— 主体选择确定性(声明顺序下第一条失败);
//   - 失败类:P2-03 ClassifyFinalStatus(仅 repairable 可构造,Builder 结构
//     强制 —— 与 Router 裁决互为印证,非 repairable 材料绝不从修复通道流出);
//   - 安全诊断摘要:仅 id / 计数 / 类别的确定性组合(无原始输出、无摘要、
//     无 held-out 定位;经 Builder 的界界 + 摘要脱敏管线);
//   - 可见证据引用:仅 visible binding(实现侧本已可见),Path/Selector/
//     Validator 取自冻结契约的 visible evidence ref —— held-out 的 store/ref
//     不在索引里,结构上不可表达;
//   - held-out 投影:仅经 verification.RedactHoldoutRecord(P1-29 唯一出口)
//   - pass|fail 类别,binding 身份/内容/摘要/可识别 verdict 不存在;
//   - 尝试谱系:Builder 从 (task, attempt) 确定性推导。
//
// fail-safe(AC4):结论事实缺席、失败主体无法指认、材料越界/不安全 —— 一律
// 类型化 error,Engine 落 feedback_unavailable 升级;修复 Run 不启动,绝不
// 降格转发、绝不带着 allowlist 之外的材料修复(INV-V14-07)。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/verification"
)

// ErrFeedbackOutcomeUnavailable:失败验证运行的结论事实在账面缺席(接线断裂
// /身份漂移)。没有可信失败事实就没有安全反馈,落点只能是升级。
var ErrFeedbackOutcomeUnavailable = errors.New("closedloop: 失败验证运行的结论事实缺席,拒绝构造修复反馈")

// contractFeedbackSource 是 FeedbackSource 的组合实现(经 ServiceVerifier
// 的 FeedbackSource() 构造,与 Router 共享同一账面)。
type contractFeedbackSource struct {
	ledger   *outcomeLedger
	contract *verification.Frozen
}

// FeedbackSource 端口编译期钉住。
var _ FeedbackSource = (*contractFeedbackSource)(nil)

// RepairFeedback 实现 FeedbackSource。构造与分类是纯内存投影:ctx 仅承载
// 端口签名,取消纪律由 Engine 在调用前后机械执行(feedback.go 端口契约)。
func (s *contractFeedbackSource) RepairFeedback(ctx context.Context, failure VerifyFailure) (RepairFeedback, error) {
	_ = ctx
	out := s.ledger.lookup(failure.ChangeSet, failure.RunRef)
	if out == nil {
		return RepairFeedback{}, fmt.Errorf("%w: changeset=%s run=%s", ErrFeedbackOutcomeUnavailable, failure.ChangeSet, failure.RunRef)
	}
	// 只有 repairable 可构造(P2-03/04 结构强制):升级与 infra 材料绝不从
	// 修复通道流出 —— 先于主体指认显式拒绝,错误词面精确到失败类。
	cls := ClassifyFinalStatus(out.Status)
	if cls != FailureRepairable {
		return RepairFeedback{}, fmt.Errorf("%w: 失败类 %q", ErrFailureNotRepairable, string(cls))
	}
	b := NewFeedbackBuilder(failure.TaskID, failure.Attempt, failure.ChangeSet, failure.RunRef).
		WithFailure(cls)
	if err := s.fillSubject(b, out); err != nil {
		return RepairFeedback{}, err
	}
	s.fillVisibleEvidence(b, out)
	s.fillHoldout(b, out)
	b.WithDiagnosticSummary(diagnosticSummary(out))
	return b.Build()
}

// fillSubject 指认失败主体(确定性):声明顺序下第一条失败的 visible binding
// 指认其 acceptance(id + 冻结语义文本);没有失败 binding 时指认第一条失败
// required gate。结论里既无失败 binding 也无失败 gate(与 FinalFailed 语义
// 矛盾)时显式失败 —— 绝不虚构主体。
func (s *contractFeedbackSource) fillSubject(b *FeedbackBuilder, out *verification.FinalOutcome) error {
	accs := acceptanceIndex(s.contract)
	for _, bo := range out.Bindings {
		if bo.Status != verification.EvidenceStatusFail {
			continue
		}
		acc, ok := accs[bo.AcceptanceID]
		if !ok {
			return fmt.Errorf("%w: 失败 binding %s 指认的 acceptance %q 不在冻结契约中", ErrInvalidFeedback, bo.BindingID, bo.AcceptanceID)
		}
		b.ForAcceptance(acc.ID, acc.Statement)
		return nil
	}
	for _, g := range out.Gates {
		if g.Status != verification.VisibleStatusFail {
			continue
		}
		b.ForGate(g.Name)
		return nil
	}
	return fmt.Errorf("%w: 失败结论中没有可指认的 acceptance/gate 主体", ErrInvalidFeedback)
}

// fillVisibleEvidence 追加失败 visible binding 的证据引用事实(仅实现侧本已
// 可见;条数截到 Builder 的界,超量条目按确定性顺序丢弃 —— 界界报错会升级,
// 这里选择保留前 MaxFeedbackEvidenceFacts 条可定位证据,不因规模升级)。
// 定位字段只取 visible evidence ref:held-out binding 不在索引中,若结论
// 携带未知 binding id,Builder 按非法证据类型拒绝(fail-safe)。
func (s *contractFeedbackSource) fillVisibleEvidence(b *FeedbackBuilder, out *verification.FinalOutcome) {
	refs := visibleEvidenceIndex(s.contract)
	added := 0
	for _, bo := range out.Bindings {
		if bo.Status != verification.EvidenceStatusFail || added >= MaxFeedbackEvidenceFacts {
			continue
		}
		b.AddVisibleEvidence(VisibleEvidence{
			AcceptanceID: bo.AcceptanceID,
			BindingID:    bo.BindingID,
			EvidenceType: refs[bo.BindingID].Type,
			Path:         refs[bo.BindingID].Path,
			Selector:     refs[bo.BindingID].Selector,
			Validator:    refs[bo.BindingID].Validator,
			Status:       verification.EvidenceStatusFail,
		})
		added++
	}
}

// fillHoldout 追加 held-out 投影事实:唯一出口 RedactHoldoutRecord + pass|fail
// 类别;条数截到 Builder 的界。binding 身份/内容/摘要不在投影类型上。
func (s *contractFeedbackSource) fillHoldout(b *FeedbackBuilder, out *verification.FinalOutcome) {
	for i, h := range out.Holdout {
		if i >= MaxFeedbackHoldoutFacts {
			break
		}
		b.AddHoldoutOutcome(verification.RedactHoldoutRecord(h), h.Status)
	}
}

// acceptanceIndex 以冻结契约的 acceptance id 为键(声明顺序保留在切片里,
// 这里只做查找)。
func acceptanceIndex(f *verification.Frozen) map[string]verification.Acceptance {
	out := map[string]verification.Acceptance{}
	if f == nil {
		return out
	}
	for _, acc := range f.Contract().Acceptances {
		out[acc.ID] = acc
	}
	return out
}

// visibleEvidenceIndex 只索引 **visible** binding 的 evidence ref:held-out
// binding 的 store/ref 在本索引里根本不存在(类型层排除,INV-V14-07)。
func visibleEvidenceIndex(f *verification.Frozen) map[string]verification.EvidenceRef {
	out := map[string]verification.EvidenceRef{}
	if f == nil {
		return out
	}
	for _, acc := range f.Contract().Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy != verification.VisibilityVisible {
				continue
			}
			out[b.ID] = b.Evidence
		}
	}
	return out
}

// diagnosticSummary 组装安全诊断摘要(allowlist:"safe diagnostic summary",
// 权威侧脱敏的再防线):只含 id、计数与类别 —— 无原始输出、无内容/输出摘要、
// 无 held-out 定位(held-out 只报计数,binding 身份不进实现侧通道)。决定性:
// 相同结论恒得相同摘要;长度由 Builder 的界界管线兜底(sha256 形态被显式
// 标记替换)。
func diagnosticSummary(out *verification.FinalOutcome) string {
	var failedBindings []string
	for _, bo := range out.Bindings {
		if bo.Status == verification.EvidenceStatusFail {
			failedBindings = append(failedBindings, bo.BindingID)
		}
	}
	var failedGates []string
	for _, g := range out.Gates {
		if g.Status == verification.VisibleStatusFail {
			failedGates = append(failedGates, g.Name)
		}
	}
	failedHoldouts := 0
	for _, h := range out.Holdout {
		if h.Status == verification.EvidenceStatusFail {
			failedHoldouts++
		}
	}
	parts := []string{
		fmt.Sprintf("candidate %s rejected under frozen contract revision %d", out.ChangeSetID, out.ContractRevision),
		fmt.Sprintf("visible evidence failed: %s", joinOrNone(failedBindings)),
		fmt.Sprintf("required gates failed: %s", joinOrNone(failedGates)),
		fmt.Sprintf("held-out checks failed: %d", failedHoldouts),
	}
	return strings.Join(parts, "; ")
}

// joinOrNone 渲染 id 列表;空集显式写 none(摘要永远三段可读)。
func joinOrNone(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}
