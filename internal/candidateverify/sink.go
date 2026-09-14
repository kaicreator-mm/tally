// RunSink→runstore 适配(P1-38 端口的集成接线,P2-01 从 worker v1.3 组成迁入)。
package candidateverify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/verification"
)

// runStoreSink 把 verification.RunSink 窄端口接到 runstore 适配器(internal/runstore
// 反向依赖 verification 包,不能自己实现端口:适配归集成接线,本类型即该接线)。
// append-only:RecordRun/RecordFact 只写,ValidApplicableFacts 只读,端口上不存在
// 任何更新/删除/裁定语义(INV-FACT-01)。一个实例服务一次最终验证(revision 在
// 构造时绑定)。
type runStoreSink struct {
	runs     *runstore.Store
	revision int32
}

// NewRunStoreSink 构造生产事实面适配器:verification_run / evidence 落
// internal/runstore,精确绑定冻结契约 revision。
func NewRunStoreSink(runs *runstore.Store, revision int) verification.RunSink {
	return &runStoreSink{runs: runs, revision: int32(revision)}
}

// RecordRun 恰好落一行 verification_run(精确绑定 contract/change/environment
// 身份,§8.13)。
func (s *runStoreSink) RecordRun(ctx context.Context, spec verification.RunSpec) (int64, error) {
	id, err := s.runs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
		Project:           spec.Project,
		Task:              spec.Task,
		ContractRevision:  int32(spec.ContractRevision),
		ContractHash:      spec.ContractHash,
		ChangeSetID:       spec.ChangeSetID,
		EnvName:           spec.EnvironmentName,
		EnvDefinitionHash: spec.EnvironmentHash,
		VersionCombo:      spec.VersionCombo,
		Runtime:           spec.Runtime,
		StartedAt:         spec.StartedAt,
		FinishedAt:        spec.FinishedAt,
		Status:            runstoreStatus(spec.Status),
	})
	if err != nil {
		return 0, fmt.Errorf("candidateverify: 落 verification_run(task=%s): %w", spec.Task, err)
	}
	return id, nil
}

func runstoreStatus(s verification.RunStatus) runstore.VerificationStatus {
	switch s {
	case verification.RunStatusPass:
		return runstore.VerificationPass
	case verification.RunStatusFail:
		return runstore.VerificationFail
	default:
		return runstore.VerificationError
	}
}

// RecordFact 落一条 evidence 事实(append-only;fact_id 重复是数据库约束错误,
// 不是更新)。
func (s *runStoreSink) RecordFact(ctx context.Context, project core.ProjectID, fact verification.FactSpec) error {
	exit := int32(fact.ExitCode)
	rec := runstore.EvidenceRecord{
		FactID:           fact.FactID,
		RunID:            fact.RunID,
		Task:             fact.Task,
		Acceptance:       fact.Acceptance,
		BindingID:        fact.BindingID,
		VisibilityPolicy: fact.VisibilityPolicy,
		Verdict:          evidence.Verdict(fact.Verdict),
		Recorder:         fact.Recorder,
		Command:          commandSpec(fact.Command),
		ExitCode:         &exit,
		StdoutDigest:     fact.StdoutDigest,
		StderrDigest:     fact.StderrDigest,
		RecordedAt:       fact.RecordedAt,
	}
	if _, err := s.runs.RecordEvidence(ctx, project, rec); err != nil {
		return fmt.Errorf("candidateverify: 落 evidence %s: %w", fact.FactID, err)
	}
	return nil
}

// commandSpec 把冻结协议命令转成 command.CommandSpec:env 的 KEY=VALUE 列表
// 转 map,shell 恒为 false(结构化 argv,决策锁 6)。
func commandSpec(spec *verification.CommandSpec) *command.CommandSpec {
	if spec == nil {
		return nil
	}
	var env map[string]string
	for _, kv := range spec.Env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue // 非法项由 spec.Validate 在执行前拒绝;这里只做形状转换
		}
		if env == nil {
			env = make(map[string]string)
		}
		env[k] = v
	}
	return &command.CommandSpec{
		Argv:    append([]string(nil), spec.Argv...),
		Cwd:     spec.Dir,
		Env:     env,
		Timeout: time.Duration(spec.Timeout),
		Shell:   false,
	}
}

// ValidApplicableFacts 读回既往 valid 且精确适用的 evidence(§8.8.1/8.8.2)。
// error 判定无可用 verdict,适配层省略(不参与覆盖推导)。
func (s *runStoreSink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]verification.PriorFact, error) {
	rows, err := s.runs.ApplicableValidEvidence(ctx, scope, s.revision)
	if err != nil {
		return nil, fmt.Errorf("candidateverify: 读回既往 evidence: %w", err)
	}
	out := make([]verification.PriorFact, 0, len(rows))
	for _, r := range rows {
		switch evidence.Verdict(r.Verdict) {
		case evidence.VerdictPass:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusPass, RecordedAt: r.RecordedAt})
		case evidence.VerdictFail:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusFail, RecordedAt: r.RecordedAt})
		default:
			continue
		}
	}
	return out, nil
}
