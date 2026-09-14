package failure

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"tally/internal/airuntime"
	"tally/internal/core"
)

// fakeExecutor 是 airuntime.FakeExecutor 的便捷包装(P0-10 同款):
// 预设 SUCCEEDED 结果,并暴露收到的提示词/调用数(与旧 countingProposer
// 的断言面等价);铁律 1 的测试基础 —— 所有「零 LLM」验收都靠断言 calls。
type fakeExecutor struct {
	*airuntime.FakeExecutor
}

func newFakeExecutor(out string) *fakeExecutor {
	return &fakeExecutor{FakeExecutor: &airuntime.FakeExecutor{
		Result: airuntime.Result{
			OperationID:       "op-test",
			Status:            airuntime.StatusSucceeded,
			TerminationReason: airuntime.TerminationCompleted,
			Output:            json.RawMessage(out),
		},
	}}
}

func newFakeExecutorErr(err error) *fakeExecutor {
	return &fakeExecutor{FakeExecutor: &airuntime.FakeExecutor{Err: err}}
}

// prompt 返回第 i 次调用收到的提示词(无调用时为空)。
func (f *fakeExecutor) prompt(i int) string {
	reqs := f.Requests()
	if i >= len(reqs) {
		return ""
	}
	return reqs[i].Input.Prompt
}

// calls 返回收到的调用数。
func (f *fakeExecutor) calls() int { return len(f.Requests()) }

const okCandidateJSON = `{"summary":"锁定依赖版本","root_cause":"上游破坏性变更","changes":[{"path":"go.mod","content":"module demo\n"}]}`

func TestDiagnoseParsesCandidate(t *testing.T) {
	fe := newFakeExecutor(okCandidateJSON)
	ev := Event{Source: "build", Excerpt: "make: *** [build] Error 1"}
	cand, err := Diagnose(context.Background(), fe, ev, nil)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if cand.Summary != "锁定依赖版本" || len(cand.Changes) != 1 || cand.Changes[0].Path != "go.mod" {
		t.Errorf("候选应解析完整: %+v", cand)
	}
	if fe.calls() != 1 {
		t.Errorf("应调用一次推理, got %d", fe.calls())
	}

	// 送出的 Request 过域侧闸且形状完整(operation/intent/幂等键/输出形状,
	// DECISION_LOCKS #17 / PRD §28)。
	reqs := fe.Requests()
	if reqs[0].Operation != DiagnoseOperation {
		t.Errorf("operation 应为稳定 ID %q, got %q", DiagnoseOperation, reqs[0].Operation)
	}
	if reqs[0].Intent.Role == "" || reqs[0].Intent.Instructions == "" {
		t.Errorf("intent 必须显式声明角色与用途: %+v", reqs[0].Intent)
	}
	if err := reqs[0].Validate(); err != nil {
		t.Errorf("域构造的 Request 应过域侧闸: %v", err)
	}
	if !json.Valid(reqs[0].Input.OutputShape) {
		t.Errorf("OutputShape 应是合法 JSON: %s", reqs[0].Input.OutputShape)
	}
}

// 验收:送进推理的内容先脱敏 —— 片段里的疑似密钥变成占位符。
func TestDiagnoseRedactsBeforePropose(t *testing.T) {
	fe := newFakeExecutor(okCandidateJSON)
	ev := Event{
		Source:  "build",
		Excerpt: "connect failed password=hunter2secret\nsee https://user:pass123@db.internal/x",
	}
	if _, err := Diagnose(context.Background(), fe, ev, nil); err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	got := fe.prompt(0)
	if strings.Contains(got, "hunter2secret") || strings.Contains(got, "pass123") {
		t.Errorf("提示词泄漏了密钥: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("脱敏后应留下占位符: %q", got)
	}
}

// 验收(脱敏边界):脱敏漏网的字段(这里是来源字段被污染)在送出前被拒,不调推理。
func TestDiagnoseRefusesUnredactedField(t *testing.T) {
	fe := newFakeExecutor(okCandidateJSON)
	ev := Event{
		Source:  "Authorization: Bearer ghp_abcdefghij1234567890",
		Excerpt: "build failed",
	}
	if _, err := Diagnose(context.Background(), fe, ev, nil); err == nil {
		t.Fatal("疑似密钥应拒绝送出")
	}
	if fe.calls() != 0 {
		t.Errorf("拒绝时不得调用推理, got %d 次", fe.calls())
	}
}

func TestDiagnoseProposerError(t *testing.T) {
	fe := newFakeExecutorErr(errors.New("模型不可用"))
	if _, err := Diagnose(context.Background(), fe, Event{Excerpt: "x"}, nil); err == nil {
		t.Error("传输层出错应返回错误")
	}
}

func TestDiagnoseBadJSON(t *testing.T) {
	fe := newFakeExecutor(`{不是JSON`)
	if _, err := Diagnose(context.Background(), fe, Event{Excerpt: "x"}, nil); err == nil {
		t.Error("非法 JSON 候选应报错")
	}
}

// 验收:Runtime 终态不得折叠 —— UNCERTAIN/CANCELLED/FAILED 各归到可区分的
// 类型化错误(FAIL 与 ERROR/拿不准端到端可区分)。
func TestDiagnoseRuntimeTerminalStates(t *testing.T) {
	cases := []struct {
		name    string
		status  airuntime.Status
		wantErr error
	}{
		{"uncertain", airuntime.StatusUncertain, ErrDiagnoseUncertain},
		{"cancelled", airuntime.StatusCancelled, ErrDiagnoseCancelled},
		{"failed", airuntime.StatusFailed, ErrDiagnoseFailed},
		{"unknown", airuntime.Status("SOMETHING_ELSE"), ErrDiagnoseUncertain},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fe := &fakeExecutor{FakeExecutor: &airuntime.FakeExecutor{Result: airuntime.Result{
				OperationID: "op-x", Status: tc.status, TerminationReason: airuntime.TerminationCompleted,
			}}}
			_, err := Diagnose(context.Background(), fe, Event{Excerpt: "x"}, nil)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("终态 %s 应归 %v, got %v", tc.status, tc.wantErr, err)
			}
		})
	}
}

// 验收:模糊命中的线索进提示词,且明确标注「仅供参考」。
func TestDiagnosePromptMarksLeads(t *testing.T) {
	fe := newFakeExecutor(okCandidateJSON)
	leads := []core.Lead{{ExactFP: "beef0000beef0000", Status: "fixed"}}
	if _, err := Diagnose(context.Background(), fe, Event{Excerpt: "x"}, leads); err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	got := fe.prompt(0)
	if !strings.Contains(got, "beef0000beef0000") {
		t.Errorf("提示词应含线索指纹: %q", got)
	}
	if !strings.Contains(got, "仅供参考") {
		t.Errorf("线索应明确标注仅供参考: %q", got)
	}
}

// 验收(铁律 1 的接口形状):LLM 候选结构里只有提案字段,没有判定字段 ——
// LLM 想声明「已验证/已通过」都没地方落。
func TestCandidateSchemaHasNoVerdictFields(t *testing.T) {
	forbidden := map[string]bool{
		"verified": true, "approved": true, "passed": true, "score": true,
		"confidence": true, "verdict": true, "judgment": true, "judged": true,
		"evaluated": true, "validated": true, "status": true,
	}
	var check func(rt reflect.Type)
	check = func(rt reflect.Type) {
		switch rt.Kind() {
		case reflect.Slice, reflect.Ptr, reflect.Array:
			check(rt.Elem())
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
				if forbidden[name] {
					t.Errorf("候选结构出现判定字段 %q:LLM 只能提案,不得自我判定", f.Name)
				}
				check(f.Type)
			}
		}
	}
	check(reflect.TypeOf(Candidate{}))
}

// 防止测试用的固定时间漂走。
func TestFixedNowIsUTC(t *testing.T) {
	if fixedNow.Location() != time.UTC {
		t.Error("测试时钟应为 UTC")
	}
}
