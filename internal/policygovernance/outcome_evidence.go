// outcome_evidence.go 拥有 Outcome → 治理证据的投影桥(P6-03):把已录得的
// append-only outcome 事件暴露为规则提案/重放的证据输入 —— 治理面读
// append-only 事件;运行时策略匹配绝不直接查询 Outcome(AC,边界守卫测试
// 钉死:engineeringpolicy 不得导入 outcome)。
//
// 引用格式与稳定性(改动前必读):
//   - 引用 = 事件自然键 "outcome:<绑定>/<kind>/<时刻>":绑定取 task_id(优先)
//     或 changeset_id,时刻为 UTC RFC3339Nano。P6-02 的精确重放抑制键正是
//     kind+坐标+时刻 —— 该三元组在 append-only 日志内唯一,可作为稳定引用;
//   - 投影是纯函数:消费 outcome.Event 值(调用方经 outcome.Store 的
//     ForTask/ForChangeset 等既有 append-only 读面取得),本包不持有、不
//     查询任何数据源;运行时匹配面(engineeringpolicy)与 Outcome 零耦合;
//   - fail-closed:kind 必须是五个封闭枚举之一、事件必须通过 outcome 自身
//     Validate(至少一个绑定 + 显式时刻)—— 不合法事件不产生证据引用。
package policygovernance

import (
	"fmt"
	"strings"
	"time"

	"tally/internal/outcome"
)

// OutcomeEvidenceRef 把一条已录得的 outcome 事件映射为治理证据引用
// (EvidenceOutcome kind)。事件必须合法(kind 封闭枚举 + 至少一个绑定 +
// 显式时刻)。
func OutcomeEvidenceRef(e outcome.Event) (EvidenceRef, error) {
	if err := e.Validate(); err != nil {
		return EvidenceRef{}, fmt.Errorf("%w: outcome 事件: %w", ErrInvalidProposal, err)
	}
	binding := strings.TrimSpace(e.TaskID)
	if binding == "" {
		binding = strings.TrimSpace(e.ChangesetID)
	}
	ref := fmt.Sprintf("%s/%s/%s", binding, e.Kind, e.RecordedAt.UTC().Format(time.RFC3339Nano))
	return NewEvidenceRef(EvidenceOutcome, ref)
}

// ProposalEvidence 把一批已录得的 outcome 事件投影为提案证据引用(保持事件
// 日志的先后次序;任一事件不合法即整体失败 —— 证据面不做静默裁剪)。
func ProposalEvidence(events []outcome.Event) ([]EvidenceRef, error) {
	out := make([]EvidenceRef, 0, len(events))
	for i, e := range events {
		ref, err := OutcomeEvidenceRef(e)
		if err != nil {
			return nil, fmt.Errorf("证据[%d]: %w", i, err)
		}
		out = append(out, ref)
	}
	return out, nil
}

// OutcomeImportBoundary 是边界守卫的断言常量(测试引用):运行时策略匹配
// 面禁止依赖 Outcome —— 治理经本桥读取事件,匹配面与 Outcome 零耦合。
const OutcomeImportBoundary = "tally/internal/outcome"
