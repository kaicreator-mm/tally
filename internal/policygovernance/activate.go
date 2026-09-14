// activate.go 拥有治理链的激活服务(P3-07):把人本批准的 RuleProposal 变成
// 运行时可观察的正典 .tally/policies/*.yaml —— 经既有 gitwriter 的期望基线
// 提交,并从确切新源提交重载验证。
//
// 激活顺序即不变量(AC:提案只有在其承诺策略可从源快照观察后才算运行时生效):
//
//  1. Reconstruct:提案必须处于 PROPOSED(DRAFT 未提交、终态不可激活);
//  2. Git 提交:候选原始 YAML 经 gitwriter 写入 .tally/policies/(期望基线
//     CAS;陈旧基线 → ErrBaseRevision,零部分写入,治理面不留激活事件);
//  3. 重载验证:artifacts.LoadPolicies 从确切新提交装载,必须能观察到同 id
//     且同内容修订的策略 —— 不可观察即类型化失败(提交回滚不做了:gitwriter
//     的 CAS 提交本身原子;此时 HEAD 已含该策略,运行时可观察性与治理记录
//     的取舍永远偏向 Git 权威面,DL-09);
//  4. 治理记录:最后落 activate 事件(approved_by 必填,BI-01)—— 它是
//     批准的证据记录,不是运行时开关(DL-09:运行时只消费 Git 快照内容)。
//
// 失败次序的取舍:第 4 步失败(治理库不可写)时 Git 面已生效 —— 返回类型化
// 错误并在 Detail 携带已生效提交 sha,治理记录按 append-only 纪律补记,绝不
// 回滚 Git(运行时权威不可因记录面故障而摆动)。
package policygovernance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/artifacts"
	"tally/internal/gitwriter"
)

// ErrInvalidActivation:激活前置条件不满足(提案不存在/状态不是 PROPOSED/
// 候选内容与治理历史漂移)。
var ErrInvalidActivation = errors.New("policygovernance: 激活前置条件不满足")

// ErrActivationNotObservable:提交后从确切新源提交装载,未能观察到同 id 且
// 同内容修订的策略(AC 的可观察性闸)。
var ErrActivationNotObservable = errors.New("policygovernance: 提交后策略不可从源快照观察")

// ActivationRequest 是一次激活的输入:谁批准、基于哪个基线提交、写进哪个仓库。
type ActivationRequest struct {
	// ProposalID 是要激活的治理提案(必须已 Reconstruct 且处于 PROPOSED)。
	ProposalID string
	// Approver 是批准人(BI-01 approved_by;非空)。
	Approver string
	// ExpectedBase 是 gitwriter 期望基线(CAS;空串断言 unborn HEAD)。
	ExpectedBase string
	// RepoPath 是内容仓库(与 artifacts.LoadPolicies 同一仓库)。
	RepoPath string
	// OccurredAt 是治理动作发生时刻;零值取 time.Now(服务便利;事件面仍显式)。
	OccurredAt time.Time
}

// ActivationResult 报告一次成功激活:Git 提交、策略路径与重载到的快照提交。
type ActivationResult struct {
	// Commit 是激活提交的完整 sha(运行时可观察的源快照)。
	Commit string
	// PolicyPath 是写入的策略制品路径(.tally/policies/*.yaml)。
	PolicyPath string
	// Proposal 是激活后的治理提案(ACTIVE,含批准人)。
	Proposal Proposal
}

// Activate 执行激活链:重放核验 → Git 提交 → 重载验证 → 治理记录。
// w 是该仓库的唯一 Git 写边界(调用方持有一例;DL-03)。
func (s *Store) Activate(ctx context.Context, req ActivationRequest, w *gitwriter.Writer) (ActivationResult, error) {
	if req.Approver == "" {
		return ActivationResult{}, fmt.Errorf("%w: 批准人必填(BI-01 approved_by)", ErrInvalidActivation)
	}
	if req.RepoPath == "" || w == nil {
		return ActivationResult{}, fmt.Errorf("%w: 仓库与 Git 写边界必填", ErrInvalidActivation)
	}

	// 1. 重放核验:提案存在、处于 PROPOSED、候选内容仍与治理历史一致。
	proposal, err := s.Reconstruct(ctx, req.ProposalID)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("%w: %w", ErrInvalidActivation, err)
	}
	if proposal.Status != StatusProposed {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s 处于 %s,只有 PROPOSED 可激活(且必须已携带证据引用)",
			ErrInvalidActivation, req.ProposalID, proposal.Status)
	}
	if len(proposal.Evidence) == 0 {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s 无证据引用,拒绝激活", ErrInvalidActivation, req.ProposalID)
	}
	// 候选内容必须仍可正典解析且修订一致(P3-03;治理面防御,内容出生已校验)。
	if _, err := VerifyCandidate(proposal.Candidate.Raw(), proposal.Candidate.Revision()); err != nil {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s: %w", ErrInvalidActivation, req.ProposalID, err)
	}

	policyPath, err := policyArtifactPath(proposal.Candidate.Policy().ID)
	if err != nil {
		return ActivationResult{}, err
	}

	// 2. Git 提交(期望基线 CAS;唯一写边界;消息确定性)。
	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	res, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: req.ExpectedBase,
		Message:      fmt.Sprintf("tally: activate policy %s (proposal %s)", proposal.Candidate.Policy().ID, proposal.ID),
		Files: []gitwriter.FileUpdate{{
			Path:    policyPath,
			Content: proposal.Candidate.Raw(),
		}},
	})
	if err != nil {
		// 陈旧基线/写失败:零部分写入,治理面不留激活事件(重放仍是 PROPOSED)。
		return ActivationResult{}, fmt.Errorf("policygovernance: 激活提交失败(提案 %s 保持 PROPOSED): %w", req.ProposalID, err)
	}

	// 3. 重载验证:从确切新提交装载,必须观察到同 id 且同修订的策略。
	snap, err := artifacts.LoadPolicies(req.RepoPath, res.Commit)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s 重载失败(提交 %s 已生效): %w",
			ErrActivationNotObservable, req.ProposalID, res.Commit, err)
	}
	var observedRevision string
	for i := range snap.Policies {
		if snap.Policies[i].Policy.ID == proposal.Candidate.Policy().ID {
			observedRevision = snap.Policies[i].Policy.Revision
			break
		}
	}
	if observedRevision == "" || observedRevision != proposal.Candidate.Revision() {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s 的策略(id=%s rev=%s)未在提交 %s 快照观察",
			ErrActivationNotObservable, req.ProposalID, proposal.Candidate.Policy().ID, proposal.Candidate.Revision(), res.Commit)
	}

	// 4. 治理记录:activate 事件(approved_by 必填)。失败时 Git 面已生效 ——
	//    类型化错误携带提交 sha,治理记录按 append-only 补记,不回滚 Git。
	act, err := proposal.Activate(req.Approver)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("%w: 提案 %s: %w", ErrInvalidActivation, req.ProposalID, err)
	}
	_, err = s.Append(ctx, Event{
		ProposalID:        proposal.ID,
		Kind:              EventActivate,
		CandidateRevision: proposal.Candidate.Revision(),
		CandidateYAML:     string(proposal.Candidate.Raw()),
		ProposedBy:        proposal.ProposedBy,
		ApprovedBy:        req.Approver,
		Scope:             proposal.Scope,
		OccurredAt:        occurredAt,
	})
	if err != nil {
		return ActivationResult{Commit: res.Commit, PolicyPath: policyPath, Proposal: act},
			fmt.Errorf("%w: 提案 %s 激活事件落库失败;Git 提交 %s 已生效(运行时权威,DL-09),治理记录需补记: %w",
				ErrInconsistentHistory, req.ProposalID, res.Commit, err)
	}
	return ActivationResult{Commit: res.Commit, PolicyPath: policyPath, Proposal: act}, nil
}

// policyArtifactPath 由策略 id 推导制品路径:.tally/policies/ 扁平布局
// (与装载器 isPolicyArtifactPath 同一判定);路径由策略 id 唯一决定(集合内
// id 唯一),提案 id 记录在治理面。id 必须路径安全(小写化后仅字母数字与
// ._-;含路径分隔符即拒绝 —— 不做转义器)。
func policyArtifactPath(policyID string) (string, error) {
	name := strings.ToLower(policyID)
	if name == "" {
		return "", fmt.Errorf("%w: 策略 id 为空,无法推导制品路径", ErrInvalidActivation)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return "", fmt.Errorf("%w: 策略 id %q 含路径不安全字符(允许:小写字母数字与 .-_)", ErrInvalidActivation, policyID)
		}
	}
	return artifacts.PoliciesDir + "/" + name + ".yaml", nil
}
