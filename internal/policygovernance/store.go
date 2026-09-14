// store.go 拥有策略治理事实的 append-only 事件存储(P3-02):提案、批准/
// 否决、取代以事件行持久化,历史按序重放重建 —— 不建任何「当前 ACTIVE 策略」
// 可变权威(BI-01/DL-09;要求 2:运行时激活永不从 DB 标志推导)。
//
// 存储边界(改动前必读):
//   - Append 是唯一写路径:无 UPDATE/DELETE(DB 侧 0024 触发器同时兜底);
//     事件 kind 封闭(draft/submit/activate/reject/supersede),与域层四个
//     具名转移一一对应;
//   - 本存储只服务治理事实面:运行时策略装载走 artifacts.LoadPolicies
//     (pinned Git 快照),本表对运行时不可见;
//   - Reconstruct 按事件序重放域层转移重建最终提案:序列不一致(如 submit
//     之前的 activate)是类型化错误,绝不猜测修补(fail-safe);
//   - evidence_refs 序列化为 "kind:ref" 串数组,kind 闭合校验在读出侧同样
//     强制 —— 被污染的行读出即显式失败,不静默降级。

package policygovernance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateProposal:提案身份重复(P3-03 要求 2:重复身份 fail-closed)。
// 身份由首个 draft 事件铸造;同一 proposal_id 的第二个 draft 在写入处拒绝。
var ErrDuplicateProposal = errors.New("policygovernance: 提案身份重复")

// ErrUnknownProposal:重放时事件流中不存在该提案(Reconstruct)。
var ErrUnknownProposal = errors.New("policygovernance: 未知提案")

// ErrInconsistentHistory:事件流重放与域层封闭转移不一致(写入侧本应拦截;
// 读出侧仍强制 —— 历史被污染时显式失败,不猜测修补)。
var ErrInconsistentHistory = errors.New("policygovernance: 治理历史不一致")

// EventKind 是治理动作类型的封闭词表(与域层转移一一对应;0024 CHECK 同表)。
type EventKind string

const (
	EventDraft     EventKind = "draft"
	EventSubmit    EventKind = "submit"
	EventActivate  EventKind = "activate"
	EventReject    EventKind = "reject"
	EventSupersede EventKind = "supersede"
)

// Valid 报告 k 是否为封闭词表成员。
func (k EventKind) Valid() bool {
	switch k {
	case EventDraft, EventSubmit, EventActivate, EventReject, EventSupersede:
		return true
	default:
		return false
	}
}

// Event 是一条治理事实:提案在某时刻发生了某治理动作。
type Event struct {
	ID                int64 // DB identity;Append 后回填
	ProposalID        string
	Kind              EventKind
	CandidateRevision string
	CandidateYAML     string
	Evidence          []EvidenceRef
	ProposedBy        string
	ApprovedBy        string
	Supersedes        string
	Scope             string
	OccurredAt        time.Time
}

// Store 是治理事件面的 pgx 存储(append-only;每测试独立 schema 隔离)。
type Store struct {
	pool *pgxpool.Pool
}

// NewStore 构造治理事件存储。
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Append 落一条治理事件(fail-safe:kind/身份/时刻在写入前域侧校验,DB 侧
// CHECK/触发器兜底)。返回 DB identity。
func (s *Store) Append(ctx context.Context, e Event) (int64, error) {
	if !e.Kind.Valid() {
		return 0, fmt.Errorf("%w: 治理动作类型 %q 不在封闭词表", ErrInvalidProposal, e.Kind)
	}
	if err := validateToken(e.ProposalID); err != nil {
		return 0, fmt.Errorf("%w: 治理事件提案 ID: %w", ErrInvalidProposal, err)
	}
	if e.CandidateRevision == "" || e.CandidateYAML == "" {
		return 0, fmt.Errorf("%w: 治理事件必须携带候选内容修订与原始 YAML(所见即所批)", ErrInvalidProposal)
	}
	if e.ProposedBy == "" {
		return 0, fmt.Errorf("%w: 治理事件必须携带提案人", ErrInvalidProposal)
	}
	if e.OccurredAt.IsZero() {
		return 0, fmt.Errorf("%w: 治理事件必须显式携带发生时刻(不用 DB 缺省)", ErrInvalidProposal)
	}
	if e.Kind == EventActivate && e.ApprovedBy == "" {
		return 0, fmt.Errorf("%w: activate 事件必须携带批准人", ErrInvalidProposal)
	}
	// P3-03 写入边界:候选内容必须经既有解析器正典解析,且声称修订与解析
	// 修订一致(内容修订是解析语义,不信任调用方自报)。
	if _, err := VerifyCandidate([]byte(e.CandidateYAML), e.CandidateRevision); err != nil {
		return 0, err
	}
	// P3-03 要求 2:重复身份 fail-closed —— 身份由首个 draft 铸造,第二个
	// draft 在写入处拒绝(append-only 事件流不允许身份重铸)。
	if e.Kind == EventDraft {
		existing, err := s.EventsForProposal(ctx, e.ProposalID)
		if err != nil {
			return 0, err
		}
		for _, prev := range existing {
			if prev.Kind == EventDraft {
				return 0, fmt.Errorf("%w: 提案 %s 的 draft 已存在(事件 id=%d)", ErrDuplicateProposal, e.ProposalID, prev.ID)
			}
		}
	}
	refs := make([]string, 0, len(e.Evidence))
	for _, r := range e.Evidence {
		refs = append(refs, r.String())
	}

	var id int64
	err := s.pool.QueryRow(ctx, `
INSERT INTO policy_governance_event
  (proposal_id, action_kind, candidate_rev, candidate_yaml,
   evidence_refs, proposed_by, approved_by, supersedes, scope, occurred_at)
VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),$9,$10)
RETURNING id`,
		e.ProposalID, string(e.Kind), e.CandidateRevision, e.CandidateYAML,
		refs, e.ProposedBy, e.ApprovedBy, e.Supersedes, e.Scope, e.OccurredAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("policygovernance: 落治理事件: %w", err)
	}
	return id, nil
}

// EventsForProposal 返回该提案的全部治理事件(按 id 升序 = 发生序)。
func (s *Store) EventsForProposal(ctx context.Context, proposalID string) ([]Event, error) {
	return s.query(ctx, "WHERE proposal_id = $1 ORDER BY id ASC", proposalID)
}

// AllEvents 返回治理事件流的全部事件(按 id 升序)。
func (s *Store) AllEvents(ctx context.Context) ([]Event, error) {
	return s.query(ctx, "ORDER BY id ASC")
}

func (s *Store) query(ctx context.Context, suffix string, args ...any) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, proposal_id, action_kind, candidate_rev, candidate_yaml,
       evidence_refs, proposed_by, approved_by, supersedes, scope, occurred_at
FROM policy_governance_event `+suffix, args...)
	if err != nil {
		return nil, fmt.Errorf("policygovernance: 读治理事件: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var kind string
		var refs []string
		var approvedBy, supersedes *string
		if err := rows.Scan(&e.ID, &e.ProposalID, &kind, &e.CandidateRevision, &e.CandidateYAML,
			&refs, &e.ProposedBy, &approvedBy, &supersedes, &e.Scope, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("policygovernance: 扫描治理事件: %w", err)
		}
		e.Kind = EventKind(kind)
		if !e.Kind.Valid() {
			return nil, fmt.Errorf("%w: 未知治理动作类型 %q(id=%d)", ErrInconsistentHistory, e.Kind, e.ID)
		}
		for _, raw := range refs {
			ref, err := parseEvidenceRef(raw)
			if err != nil {
				return nil, fmt.Errorf("%w: 事件 id=%d: %w", ErrInconsistentHistory, e.ID, err)
			}
			e.Evidence = append(e.Evidence, ref)
		}
		e.ApprovedBy = deref(approvedBy)
		e.Supersedes = deref(supersedes)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("policygovernance: 遍历治理事件: %w", err)
	}
	return out, nil
}

// Reconstruct 按事件序重放域层封闭转移,重建该提案的最终治理状态(AC:
// 提案历史可重建)。事件流为空 → ErrUnknownProposal;重放与域层转移规则
// 冲突 → ErrInconsistentHistory(绝不猜测修补)。
func (s *Store) Reconstruct(ctx context.Context, proposalID string) (Proposal, error) {
	events, err := s.EventsForProposal(ctx, proposalID)
	if err != nil {
		return Proposal{}, err
	}
	if len(events) == 0 {
		return Proposal{}, fmt.Errorf("%w: %s", ErrUnknownProposal, proposalID)
	}

	// 候选与身份取自首个事件(候选内容出生即固定;后续事件不携带内容变化 ——
	// 候选不可变是域层不变量,若历史出现内容漂移即不一致)。
	first := events[0]
	cand, err := NewCandidate([]byte(first.CandidateYAML))
	if err != nil {
		return Proposal{}, fmt.Errorf("%w: 提案 %s 候选内容与正典解析不符: %w", ErrInconsistentHistory, proposalID, err)
	}
	if cand.Revision() != first.CandidateRevision {
		return Proposal{}, fmt.Errorf("%w: 提案 %s 候选修订漂移(记录 %s,解析 %s)", ErrInconsistentHistory, proposalID, first.CandidateRevision, cand.Revision())
	}

	p, err := NewDraft(first.ProposalID, cand, first.ProposedBy)
	if err != nil {
		return Proposal{}, fmt.Errorf("%w: 提案 %s 首事件非法: %w", ErrInconsistentHistory, proposalID, err)
	}
	if first.Kind != EventDraft && first.Kind != EventSubmit {
		return Proposal{}, fmt.Errorf("%w: 提案 %s 首事件类型 %s 非法(须 draft/submit)", ErrInconsistentHistory, proposalID, first.Kind)
	}
	if first.Kind == EventDraft {
		// draft 事件可携带提案时的初始证据引用(P3-08:提议动作引用本案)。
		p.Evidence = append(p.Evidence, first.Evidence...)
	}
	if first.Kind == EventSubmit {
		if p, err = p.Submit(first.Evidence); err != nil {
			return Proposal{}, fmt.Errorf("%w: 提案 %s: %w", ErrInconsistentHistory, proposalID, err)
		}
	}
	p.Scope, p.Supersedes = first.Scope, first.Supersedes

	for _, ev := range events[1:] {
		switch ev.Kind {
		case EventDraft:
			return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d:重复 draft", ErrInconsistentHistory, proposalID, ev.ID)
		case EventSubmit:
			if p, err = p.Submit(ev.Evidence); err != nil {
				return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d: %w", ErrInconsistentHistory, proposalID, ev.ID, err)
			}
		case EventActivate:
			if p, err = p.Activate(ev.ApprovedBy); err != nil {
				return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d: %w", ErrInconsistentHistory, proposalID, ev.ID, err)
			}
		case EventReject:
			if p, err = p.Reject(); err != nil {
				return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d: %w", ErrInconsistentHistory, proposalID, ev.ID, err)
			}
		case EventSupersede:
			if p, err = p.Supersede(); err != nil {
				return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d: %w", ErrInconsistentHistory, proposalID, ev.ID, err)
			}
		default:
			return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d:未知类型 %q", ErrInconsistentHistory, proposalID, ev.ID, ev.Kind)
		}
		// 候选内容漂移 = 历史被污染(候选不可变是域层不变量)。
		if ev.CandidateRevision != first.CandidateRevision || ev.CandidateYAML != first.CandidateYAML {
			return Proposal{}, fmt.Errorf("%w: 提案 %s 事件 id=%d:候选内容漂移", ErrInconsistentHistory, proposalID, ev.ID)
		}
	}
	return p, nil
}

// parseEvidenceRef 读出侧解析 "kind:ref" 串并强制封闭 kind 词表。
func parseEvidenceRef(raw string) (EvidenceRef, error) {
	kind, ref, ok := strings.Cut(raw, ":")
	if !ok {
		return EvidenceRef{}, fmt.Errorf("%w: 证据引用缺分隔符 %q", ErrInconsistentHistory, raw)
	}
	return NewEvidenceRef(EvidenceKind(kind), ref)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
