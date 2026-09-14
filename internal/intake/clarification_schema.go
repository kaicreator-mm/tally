// clarification_schema.go 承载有界 Dynamic Clarification Schema(DL-11 /
// PRD §3.2 / §8):Clarification 的封闭类型词表、数量/长度上限、旧简单形态
// 的显式迁移与逐项校验。
//
// 边界与不变量:
//
//   - 这只是展示 schema(presentation),不是领域契约:字段值是纯文本数据,
//     没有任何标记/脚本通道(arbitrary HTML/JS 无承载面,PRD §8.4);权威
//     字段面仍由 Draft 与既有冻结路径承担。
//   - 全部规则 fail-safe:词表外的类型、类型不匹配的形状、超限的数量一律
//     类型化拒绝,绝不 best-effort 猜测、绝不静默截断。
package intake

import (
	"fmt"
	"strings"
)

// 有界上限(DL-11:bounded presentation schema):澄清轮是一页人能一口气
// 答完的问题,超限即形状漂移,直接拒绝而非静默截断。
const (
	// maxClarifications 是一次澄清请求的问题数上限。
	maxClarifications = 8
	// maxClarificationOptions 是单个选项类问题的候选数上限。
	maxClarificationOptions = 16
	// maxClarificationQuestionLen 是问题文本的字节长度上限(与 maxIntentLen
	// 同一量纲:字节;有界输入,防失控提示词)。
	maxClarificationQuestionLen = 500
	// maxClarificationOptionLen 是单个候选标签的字节长度上限。
	maxClarificationOptionLen = 200
	// maxClarificationReasonLen 是推荐理由的字节长度上限。
	maxClarificationReasonLen = 500
)

// validateClarifications 对解析后的澄清集合做有界 schema 校验与归一:
//   - 数量超限 → ErrInvalidProposal;
//   - kind 为空(旧简单形态)→ 显式迁移(AC:旧候选澄清保持兼容):
//     带 candidates → choice;不带 → free_text(自然语言逃生舱,PRD §8.1);
//   - kind 不在封闭词表 → ErrUnknownClarificationKind(DL-11 fail safe);
//   - 选项类类型(choice/multichoice/repository/branch/risk_choice)必须
//     带非空候选,候选数与标签长度有界;
//   - 无候选类型(boolean/number/confirmation/free_text)不得携带候选;
//   - Recommended/RecommendationReason 只属于选项类类型且下标必须在候选
//     范围内(advisory,PRD §8.3);
//   - number 的 min/max 若同时给出必须构成合法闭区间。
//
// 返回归一后的副本(Kind 已解析为词表值),不就地修改输入。
func validateClarifications(cs []Clarification) ([]Clarification, error) {
	if len(cs) > maxClarifications {
		return nil, fmt.Errorf("%w: 澄清问题数 %d 超上限 %d", ErrInvalidProposal, len(cs), maxClarifications)
	}
	out := make([]Clarification, 0, len(cs))
	for i, c := range cs {
		label := fmt.Sprintf("澄清问题 #%d", i+1)
		if strings.TrimSpace(c.Question) == "" {
			return nil, fmt.Errorf("%w: %s 缺少 question", ErrInvalidProposal, label)
		}
		if len(c.Question) > maxClarificationQuestionLen {
			return nil, fmt.Errorf("%w: %s 问题长度 %d 超上限 %d", ErrInvalidProposal, label, len(c.Question), maxClarificationQuestionLen)
		}
		for j, cand := range c.Candidates {
			if strings.TrimSpace(cand) == "" {
				return nil, fmt.Errorf("%w: %s 的候选答案 #%d 为空白", ErrInvalidProposal, label, j+1)
			}
		}

		kind := c.Kind
		if kind == "" {
			// 旧简单形态的显式迁移:按形态归一,不猜语义。
			if len(c.Candidates) > 0 {
				kind = KindChoice
			} else {
				kind = KindFreeText
			}
		} else if _, ok := clarificationKindOf(string(kind)); !ok {
			return nil, fmt.Errorf("%w: %s kind=%q(有界词表:choice/multichoice/boolean/number/repository/branch/risk_choice/confirmation/free_text)", ErrUnknownClarificationKind, label, string(kind))
		}

		options := kind == KindChoice || kind == KindMultiChoice || kind == KindRepository ||
			kind == KindBranch || kind == KindRiskChoice
		if options {
			if len(c.Candidates) == 0 {
				return nil, fmt.Errorf("%w: %s 类型 %q 必须带候选答案", ErrInvalidProposal, label, string(kind))
			}
			if len(c.Candidates) > maxClarificationOptions {
				return nil, fmt.Errorf("%w: %s 候选数 %d 超上限 %d", ErrInvalidProposal, label, len(c.Candidates), maxClarificationOptions)
			}
			for _, cand := range c.Candidates {
				if len(cand) > maxClarificationOptionLen {
					return nil, fmt.Errorf("%w: %s 候选标签长度 %d 超上限 %d", ErrInvalidProposal, label, len(cand), maxClarificationOptionLen)
				}
			}
		} else if len(c.Candidates) > 0 {
			return nil, fmt.Errorf("%w: %s 类型 %q 不得携带候选答案", ErrInvalidProposal, label, string(kind))
		}

		if c.Recommended != nil {
			if !options {
				return nil, fmt.Errorf("%w: %s 类型 %q 不支持 recommended", ErrInvalidProposal, label, string(kind))
			}
			if *c.Recommended < 0 || *c.Recommended >= len(c.Candidates) {
				return nil, fmt.Errorf("%w: %s recommended 下标 %d 越界(候选数 %d)", ErrInvalidProposal, label, *c.Recommended, len(c.Candidates))
			}
		}
		if strings.TrimSpace(c.RecommendationReason) != "" {
			if c.Recommended == nil {
				return nil, fmt.Errorf("%w: %s 推荐理由必须伴随 recommended 出现", ErrInvalidProposal, label)
			}
			if len(c.RecommendationReason) > maxClarificationReasonLen {
				return nil, fmt.Errorf("%w: %s 推荐理由长度 %d 超上限 %d", ErrInvalidProposal, label, len(c.RecommendationReason), maxClarificationReasonLen)
			}
		}
		if kind == KindNumber && c.Min != nil && c.Max != nil && *c.Min > *c.Max {
			return nil, fmt.Errorf("%w: %s 数值区间非法(min %v > max %v)", ErrInvalidProposal, label, *c.Min, *c.Max)
		}

		nc := c
		nc.Kind = kind
		out = append(out, nc)
	}
	return out, nil
}
