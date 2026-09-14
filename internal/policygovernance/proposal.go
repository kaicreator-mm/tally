// Package policygovernance 拥有策略治理的提案域(P3-01):把「提议中的策略
// 内容 + 证据引用 + 治理身份/生命周期」表示为独立的治理事实 —— 绝不修改
// 运行时 EngineeringPolicy 模式(DL-08/BI-01)。
//
// 域边界(改动前必读):
//
//   - 候选内容不可变:Candidate 在构造时经 engineeringpolicy.Parse 正典解析
//     并校验(AC:提案引用可正典解析的策略候选),之后零 setter —— 内容与
//     内容修订(RevisionHash)从出生固定。治理字段(status/evidence_refs/
//     approved_by/supersedes/scope)全部住在本包的治理事实侧;engineeringpolicy.
//     Parse 的 KnownFields(true) 使混入候选 YAML 的治理字段结构性地解析失败
//     —— 运行时策略 YAML 面不存在治理词汇(要求 2)。
//   - 生命周期是治理状态,不是运行时策略状态(要求 2):运行时生效的永远是
//     pinned Git 快照里 .tally/policies/*.yaml 的确切内容(DL-09);本包的
//     Status 只是治理事实的封闭词表(BI-01:DRAFT/PROPOSED/ACTIVE/SUPERSEDED/
//     REJECTED),「runtime consumes ACTIVE」的语义由后续 P3-07 的 gitwriter
//     提交实现 —— 本包不落库(P3-02 的 append-only store)、不写 Git、不做
//     运行时激活。
//   - 转移封闭且确定性:Submit/Activate/Reject/Supersede 四个具名转移,其余
//     一律 ErrInvalidTransition(无回退、无跳级;supersession append-only)。
package policygovernance

import (
	"errors"
	"fmt"

	"tally/internal/engineeringpolicy"
)

// ErrInvalidProposal 是治理提案域的哨兵:构造输入不合法。用 errors.Is 判别。
var ErrInvalidProposal = errors.New("policygovernance: 非法提案")

// ErrInvalidTransition 是生命周期转移的哨兵:封闭词表之外的转移一律拒绝
// (治理状态无回退、无跳级)。
var ErrInvalidTransition = errors.New("policygovernance: 非法生命周期转移")

// Status 是治理生命周期的封闭词表(BI-01 指定词表;DL-08:绝不写入
// engineeringpolicy.Policy 的 YAML/struct,也绝不落进运行时激活判定 ——
// 运行时只消费 pinned Git 快照内容,DL-09)。
type Status string

const (
	// StatusDraft:候选内容已固定但尚未提交评审(证据可后补;Submit 进入 PROPOSED)。
	StatusDraft Status = "DRAFT"
	// StatusProposed:已提交评审 —— AC 生效面:此刻起提案必须携带 ≥1 条证据引用。
	StatusProposed Status = "PROPOSED"
	// StatusActive:已批准并激活。激活本身是 gitwriter 对正典策略 YAML 的提交
	// (P3-07/DL-09);本状态只是治理事实的记录面,不是运行时开关。
	StatusActive Status = "ACTIVE"
	// StatusSuperseded:被后续提案取代(supersession append-only;终态)。
	StatusSuperseded Status = "SUPERSEDED"
	// StatusRejected:评审否决(终态;重新提案 = 新提案,不改写旧事实)。
	StatusRejected Status = "REJECTED"
)

// Valid 报告 s 是否为封闭词表成员。
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusProposed, StatusActive, StatusSuperseded, StatusRejected:
		return true
	default:
		return false
	}
}

// terminal 报告 s 是否终态(REJECTED/SUPERSEDED 不可再转移)。
func (s Status) terminal() bool {
	return s == StatusRejected || s == StatusSuperseded
}

// EvidenceKind 是证据引用类型的封闭词表:治理提案的证据只能指向既有事实面
// (outcome 事件 / ChangeSet / 验证修订 / 任务),不发明第二证据域。
type EvidenceKind string

const (
	// EvidenceOutcome:outcome 事件(五类 closed enum,P6;ref 形如 <event_id>)。
	EvidenceOutcome EvidenceKind = "outcome"
	// EvidenceChangeSet:不可变候选变更集(ref 形如 <changeset_id>)。
	EvidenceChangeSet EvidenceKind = "changeset"
	// EvidenceVerification:冻结验证修订(ref 形如 <verification_revision>)。
	EvidenceVerification EvidenceKind = "verification"
	// EvidenceTask:任务事实(ref 形如 <task_id>)。
	EvidenceTask EvidenceKind = "task"
)

// Valid 报告 k 是否为封闭词表成员。
func (k EvidenceKind) Valid() bool {
	switch k {
	case EvidenceOutcome, EvidenceChangeSet, EvidenceVerification, EvidenceTask:
		return true
	default:
		return false
	}
}

// EvidenceRef 是一条证据引用:类型化 + 引用串,可追溯到既有事实。只读值。
type EvidenceRef struct {
	Kind EvidenceKind
	// Ref 是事实面内的引用标识(如 outcome 事件 id / ChangeSet id)。非空、
	// 无空白与控制字符 —— 引用是标识,不是描述。
	Ref string
}

// String 实现 fmt.Stringer(kind:ref 形态,便于日志/审计行)。
func (e EvidenceRef) String() string { return string(e.Kind) + ":" + e.Ref }

// NewEvidenceRef 构造并校验一条证据引用。
func NewEvidenceRef(kind EvidenceKind, ref string) (EvidenceRef, error) {
	if !kind.Valid() {
		return EvidenceRef{}, fmt.Errorf("%w: 证据类型 %q 不在封闭词表(outcome|changeset|verification|task)", ErrInvalidProposal, kind)
	}
	if err := validateToken(string(kind) + ":" + ref); err != nil {
		return EvidenceRef{}, fmt.Errorf("%w: 证据引用: %w", ErrInvalidProposal, err)
	}
	return EvidenceRef{Kind: kind, Ref: ref}, nil
}

// Candidate 是不可变的候选策略内容:原始字节 + 正典解析结果在构造时一次固定,
// 之后零 setter(要求 1)。原始字节保留以便后续 P3-07 的 gitwriter 提交
// 「所见即所批」;解析结果提供内容修订与 gate 词表。
type Candidate struct {
	raw    []byte
	policy engineeringpolicy.Policy
}

// NewCandidate 构造并校验候选:内容必须可被运行时解析器正典解析(AC)。解析
// 失败 = 该内容永远进不了运行时(DL-09),提案域在出生点即拒绝,绝不携带
// 不可解析候选。
func NewCandidate(raw []byte) (Candidate, error) {
	policy, err := engineeringpolicy.Parse(raw)
	if err != nil {
		return Candidate{}, fmt.Errorf("%w: 候选策略内容不可正典解析: %w", ErrInvalidProposal, err)
	}
	// 深拷贝原始字节:外部切片后续变更不得影响候选(不可变要求 1)。
	cp := make([]byte, len(raw))
	copy(cp, raw)
	return Candidate{raw: cp, policy: policy}, nil
}

// Raw 返回候选原始字节的副本(调用方修改不影响候选)。
func (c Candidate) Raw() []byte {
	out := make([]byte, len(c.raw))
	copy(out, c.raw)
	return out
}

// Policy 返回正典解析结果(内容修订已由 Parse 填充)。
func (c Candidate) Policy() engineeringpolicy.Policy { return c.policy }

// Revision 返回候选的内容修订引用(engineeringpolicy.RevisionHash,AC 的
// 「canonical parseable policy candidate」可追溯面)。
func (c Candidate) Revision() string { return c.policy.Revision }

// Proposal 是一条治理提案:候选内容 + 证据引用 + 治理身份与生命周期
// (BI-01 指定的治理元数据词表)。值语义 + 零导出可变面:一切「变更」都是
// 返回新值的具名转移。
type Proposal struct {
	// ID 是治理身份(调用方铸造;非空、无空白、≤128 字符)。同一 ID 的
	// append-only 持久化由 P3-02 的 store 负责。
	ID string
	// Candidate 是不可变候选内容。
	Candidate Candidate
	// Status 是治理生命周期状态(BI-01 词表)。
	Status Status
	// Evidence 是证据引用(≥1;PROPOSED 及之后的硬性要求,AC)。
	Evidence []EvidenceRef
	// ProposedBy 是提案人/来源身份(非空)。
	ProposedBy string
	// ApprovedBy 是激活批准人(仅 ACTIVE 非空;由 Activate 记录)。
	ApprovedBy string
	// Supersedes 是被本提案取代的前任提案 ID(可选;append-only 指针,不改动
	// 前任事实)。
	Supersedes string
	// Scope 是提案的适用范围说明(可选;治理元数据,不是运行时匹配语义 ——
	// 运行时匹配永远由策略内容自身决定)。
	Scope string
}

// NewProposal 构造一条已提交的治理提案(StatusProposed):候选必须可正典解析,
// 证据 ≥1(AC),身份字段非空。等价于 NewDraft + Submit 的最终态。
func NewProposal(id string, cand Candidate, evidence []EvidenceRef, proposedBy string) (Proposal, error) {
	p, err := NewDraft(id, cand, proposedBy)
	if err != nil {
		return Proposal{}, err
	}
	if len(evidence) == 0 {
		return Proposal{}, fmt.Errorf("%w: 提案 %s 必须携带 ≥1 条证据引用(AC:提案引用可正典解析候选与证据)", ErrInvalidProposal, id)
	}
	p.Evidence = append([]EvidenceRef(nil), evidence...)
	p.Status = StatusProposed
	return p, nil
}

// NewDraft 构造一条草稿提案(StatusDraft):候选内容出生即固定并解析校验,
// 证据可后补(Submit 前必须补足)。草稿是治理事实的前置形态,不参与激活。
func NewDraft(id string, cand Candidate, proposedBy string) (Proposal, error) {
	if err := validateToken(id); err != nil {
		return Proposal{}, fmt.Errorf("%w: 提案 ID: %w", ErrInvalidProposal, err)
	}
	if err := validateToken(proposedBy); err != nil {
		return Proposal{}, fmt.Errorf("%w: 提案人: %w", ErrInvalidProposal, err)
	}
	return Proposal{
		ID:         id,
		Candidate:  cand,
		Status:     StatusDraft,
		ProposedBy: proposedBy,
	}, nil
}

// Submit 把 DRAFT 提案提交评审(→ PROPOSED):提交时必须补足 ≥1 条证据
// 引用(AC)。其余状态拒绝。失败时原样返回接收者(调用方保留原提案)。
func (p Proposal) Submit(evidence []EvidenceRef) (Proposal, error) {
	if p.Status != StatusDraft {
		return p, fmt.Errorf("%w: 提案 %s 处于 %s,只有 DRAFT 可提交", ErrInvalidTransition, p.ID, p.Status)
	}
	if len(evidence) == 0 {
		return p, fmt.Errorf("%w: 提交提案 %s 必须携带 ≥1 条证据引用", ErrInvalidProposal, p.ID)
	}
	next := p
	next.Evidence = append([]EvidenceRef(nil), evidence...)
	next.Status = StatusProposed
	return next, nil
}

// Activate 把 PROPOSED 提案激活(→ ACTIVE)并记录批准人。批准人非空是硬性
// 要求(治理问责面;BI-01 approved_by)。注意:真正的运行时生效由 P3-07 把
// 正典候选 YAML 经 gitwriter 提交到 .tally/policies/(DL-09) —— 本转移只是
// 治理事实的记录,不是运行时开关。
func (p Proposal) Activate(approvedBy string) (Proposal, error) {
	if p.Status != StatusProposed {
		return p, fmt.Errorf("%w: 提案 %s 处于 %s,只有 PROPOSED 可激活", ErrInvalidTransition, p.ID, p.Status)
	}
	if err := validateToken(approvedBy); err != nil {
		return p, fmt.Errorf("%w: 批准人: %w", ErrInvalidProposal, err)
	}
	next := p
	next.Status = StatusActive
	next.ApprovedBy = approvedBy
	return next, nil
}

// Reject 把 PROPOSED 提案否决(→ REJECTED,终态)。否决不改写候选内容,
// 重新提案 = 新提案(append-only)。
func (p Proposal) Reject() (Proposal, error) {
	if p.Status != StatusProposed {
		return p, fmt.Errorf("%w: 提案 %s 处于 %s,只有 PROPOSED 可否决", ErrInvalidTransition, p.ID, p.Status)
	}
	next := p
	next.Status = StatusRejected
	return next, nil
}

// Supersede 把 ACTIVE 提案标记为被取代(→ SUPERSEDED,终态):supersession
// append-only,本转移不触碰前任提案的任何字段。
func (p Proposal) Supersede() (Proposal, error) {
	if p.Status != StatusActive {
		return p, fmt.Errorf("%w: 提案 %s 处于 %s,只有 ACTIVE 可被取代", ErrInvalidTransition, p.ID, p.Status)
	}
	next := p
	next.Status = StatusSuperseded
	return next, nil
}

// validateToken 校验标识串:非空、无空白与控制字符、≤128 字符。
func validateToken(s string) error {
	if s == "" {
		return fmt.Errorf("%w: 不能为空", ErrInvalidProposal)
	}
	if len(s) > 128 {
		return fmt.Errorf("%w: 超过 128 字符", ErrInvalidProposal)
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: 含空白或控制字符 %q", ErrInvalidProposal, s)
		}
	}
	return nil
}
