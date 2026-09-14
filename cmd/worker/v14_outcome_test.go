package main

// v14_outcome_test.go 固化 P6-01 的生产接线:一次耐久接受恰产出一条
// accepted outcome 事件(task/changeset 绑定),重试/重放不重复;出口写入
// 失败显式 outcome_unavailable 升级,绝不静默接受。
//
// 数据库层与 v14_test.go 同一隔离策略:每测试独立 schema、自跑迁移、
// cleanup DROP;PostgreSQL 不可达即 SKIP(绝不记 PASS)。

import (
	"context"
	"errors"
	"testing"

	"tally/internal/closedloop"
	"tally/internal/outcome"
)

// failingAcceptSink 模拟 accepted 出口的基础设施失败(P6-01 失败政策)。
type failingAcceptSink struct{ err error }

func (f *failingAcceptSink) RecordAccepted(ctx context.Context, fact closedloop.AcceptFact) error {
	return f.err
}

// 生产路径(仅注入 runtime):接受 → outcome_event 恰一条 accepted 事件,
// 绑定任务与被接受候选;同一事实经生产适配再投递(重试/重放)仍恰一条。
func TestV14AcceptedOutcomePersistsExactlyOnce(t *testing.T) {
	runs, st := v13TestStore(t)
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		queries: st, // 生产验证服务 + 全部事实出口(含 accepted outcome)就位
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted+basis", res.Phase, res.Basis)
	}

	events := outcome.New(st)
	got, err := events.ForTask(ctx0(), "TASK-1043")
	if err != nil || len(got) != 1 {
		t.Fatalf("accepted outcome 事件 = %+v err=%v, 期望恰 1 条", got, err)
	}
	if got[0].Kind != outcome.KindAccepted ||
		got[0].TaskID != "TASK-1043" ||
		got[0].ChangesetID != string(res.Basis.ChangeSet) {
		t.Fatalf("accepted 事件绑定 = %+v, 期望 {accepted, TASK-1043, %s}",
			got[0], res.Basis.ChangeSet)
	}

	// 幂等(真库):同一接受事实重试/重放(生产适配直投)不重复落行。
	sink, err := closedloop.NewStoreAcceptSink(events)
	if err != nil {
		t.Fatalf("NewStoreAcceptSink: %v", err)
	}
	replay := closedloop.AcceptFact{
		TaskID:     "TASK-1043",
		ChangeSet:  res.Basis.ChangeSet,
		VerifyRun:  res.Basis.VerifyRun,
		Attempt:    res.Basis.Attempt,
		RecordedAt: got[0].RecordedAt,
	}
	if err := sink.RecordAccepted(ctx0(), replay); err != nil {
		t.Fatalf("重放投递: %v", err)
	}
	after, err := events.ForTask(ctx0(), "TASK-1043")
	if err != nil || len(after) != 1 {
		t.Fatalf("重放后事件 = %d 条 err=%v, 期望仍恰 1 条(不重复)", len(after), err)
	}

	// 事实面完整性:其余闭环事实照常(不因新增出口而缺)。
	timeline, err := runs.CodegenRunsForTask(ctx0(), "proj-v13", "TASK-1043")
	if err != nil || len(timeline) != 1 {
		t.Fatalf("codegen_run = %d 行 err=%v, 期望 1", len(timeline), err)
	}
}

// 失败路径:出口写入失败 → outcome_unavailable 显式升级(无 basis、不进
// 修复),且不落任何 accepted 事件(绝不静默接受)。
func TestV14AcceptSinkFailureEscalates(t *testing.T) {
	_, st := v13TestStore(t)
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		queries: st,
		accept:  &failingAcceptSink{err: errors.New("outcome store unreachable")},
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil {
		t.Fatalf("终态 = %s escalation=%v, 期望升级", res.Phase, res.Escalation)
	}
	if res.Escalation.Reason != closedloop.EscalationOutcomeUnavailable {
		t.Fatalf("升级原因 = %s, 期望 %s", res.Escalation.Reason, closedloop.EscalationOutcomeUnavailable)
	}
	if res.Basis != nil {
		t.Fatal("outcome 写失败的接受不得携带 basis")
	}
	got, err := outcome.New(st).ForTask(ctx0(), "TASK-1043")
	if err != nil {
		t.Fatalf("读 outcome 事件: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("写失败的接受不得落任何 outcome 事件: %+v", got)
	}
}
