package assurance

import (
	"errors"
	"fmt"
)

// Status — 单个验收条款必需保证的结论状态。satisfied 与 waived 是两个
// 永不合并的不同语义(PRD §13.5 / AC2):waived 不是弱化的 satisfied,
// satisfied 也不是"免检的 waived"。
type Status string

const (
	// StatusSatisfied — 达成档位满足必需档位(严格满足)。
	StatusSatisfied Status = "satisfied"
	// StatusWaived — 由 policy waiver 显式豁免;Required/Achieved 事实
	// 原样保留,waiver 不改写、不掩盖任何档位事实。
	StatusWaived Status = "waived"
	// StatusUnmet — 达成档位不满足必需档位且无 waiver。
	StatusUnmet Status = "unmet"
)

// Waiver — 针对单个验收条款保证要求的显式豁免(身份 + 理由)。
// waiver 本身是 policy 事实;其存储与 append-only 记录归 P1-08 投影。
type Waiver struct {
	// ID — waiver 身份(必填;taskstate 侧以 approval:<id>/waiver ref 追溯)。
	ID string
	// Reason — 豁免理由(必填,面向审计)。
	Reason string
}

// 结论类型化错误(errors.Is 可判)。
var (
	// ErrWaiverIdentity — waiver 缺少 ID 或 Reason。
	ErrWaiverIdentity = errors.New("assurance: waiver 必须有 id 与 reason")
	// ErrWaiverNotNeeded — 条款已严格满足时不需要也不接受 waiver
	// (satisfied 与 waived 必须保持互斥的清晰语义,PRD §13.5)。
	ErrWaiverNotNeeded = errors.New("assurance: 条款已严格满足,不接受 waiver")
)

// Requirement — 单个验收条款的完整保证结论:required 与 achieved 两个
// 独立档位值(AC1)+ 二者各自的依据(AC4)+ satisfied/waived/unmet 状态
// (AC2)。全部 typed 字段,无 map[string]any。
type Requirement struct {
	// Selection — 必需档位及其决策链(规则 1-4 的输出)。
	Selection Selection
	// AchievedLevel — 达成档位(独立于 Required 的类型化值)。
	AchievedLevel Level
	// AchievedBasis — 达成档位的事实依据。
	AchievedBasis AchievedBasis
	// Status — 结论状态;Waived 时 Waiver 非 nil,其余情况恒为 nil。
	Status Status
	// Waiver — 仅 Status == StatusWaived 时非 nil。
	Waiver *Waiver
}

// Satisfied 报告条款是否严格满足(与 waived 互斥)。
func (r Requirement) Satisfied() bool { return r.Status == StatusSatisfied }

// Waived 报告条款是否被豁免(与 satisfied 互斥)。
func (r Requirement) Waived() bool { return r.Status == StatusWaived }

// Evaluate 把选择结果、达成事实与(可选)waiver 组合成条款结论。
//
// 规则(确定性):
//   - achieved >= required → StatusSatisfied;此时传入 waiver 是调用方错误
//     (ErrWaiverNotNeeded)——satisfied 条款不允许被同时标记为 waived;
//   - achieved < required 且 waiver != nil → StatusWaived(waiver 必须带
//     身份与理由);Required/Achieved 档位事实原样保留;
//   - achieved < required 且 waiver == nil → StatusUnmet。
func (s Selection) Evaluate(achieved Level, achievedBasis AchievedBasis, waiver *Waiver) (Requirement, error) {
	if !achieved.IsValid() {
		return Requirement{}, fmt.Errorf("assurance: 条款 %s achieved %s: %w", s.ClauseID, achieved, ErrInvalidLevel)
	}
	if waiver != nil && (waiver.ID == "" || waiver.Reason == "") {
		return Requirement{}, fmt.Errorf("assurance: 条款 %s: %w", s.ClauseID, ErrWaiverIdentity)
	}

	r := Requirement{
		Selection:     s,
		AchievedLevel: achieved,
		AchievedBasis: achievedBasis,
	}

	switch {
	case achieved.Satisfies(s.Required):
		if waiver != nil {
			return Requirement{}, fmt.Errorf("assurance: 条款 %s: %w", s.ClauseID, ErrWaiverNotNeeded)
		}
		r.Status = StatusSatisfied
	case waiver != nil:
		r.Status = StatusWaived
		r.Waiver = &Waiver{ID: waiver.ID, Reason: waiver.Reason}
	default:
		r.Status = StatusUnmet
	}
	return r, nil
}
