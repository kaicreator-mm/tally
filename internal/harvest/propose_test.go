package harvest

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/airuntime"
)

// fakeExecutor 是 airuntime.FakeExecutor 的便捷包装:预设 SUCCEEDED 结果,
// 并暴露收到的提示词/调用数(与旧 fakeProposer 的断言面等价)。
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

var testMaterial = Material{
	Service: Service{Name: "minio", Source: "coolify", Ref: "minio", Hints: "对象存储"},
	Raw:     "services:\n  minio:\n    image: quay.io/minio/minio:latest\n",
}

const goodProposal = `{
  "summary": "MinIO 对象存储",
  "compose": "services:\n  minio:\n    image: quay.io/minio/minio:latest\n",
  "health_checks": [
    {"name": "minio-ready", "type": "log-pattern", "service": "minio", "pattern": "ready"}
  ],
  "wait": {"timeout": "60s", "interval": "2s"}
}`

// 正例:含健康检查的提案通过校验;送出的 Request 过域侧闸且形状完整
// (operation/intent/幂等键/输出形状,DECISION_LOCKS #17 / PRD §28)。
func TestProposeOK(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	prop, err := Propose(context.Background(), fe, testMaterial, 1, nil)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(prop.HealthChecks) == 0 {
		t.Error("提案应含健康检查")
	}
	if !strings.Contains(fe.prompt(0), "minio") {
		t.Errorf("提示词应含服务信息: %q", fe.prompt(0))
	}
	if !strings.Contains(fe.prompt(0), "健康检查") {
		t.Error("提示词应强制要求健康检查")
	}

	reqs := fe.Requests()
	if len(reqs) != 1 {
		t.Fatalf("应恰好发出一次推理调用, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Operation != ProposeOperation {
		t.Errorf("operation 应为稳定 ID %q, got %q", ProposeOperation, req.Operation)
	}
	if req.Intent.Role == "" || req.Intent.Instructions == "" {
		t.Errorf("intent 必须显式声明角色与用途: %+v", req.Intent)
	}
	if err := req.Validate(); err != nil {
		t.Errorf("域构造的 Request 应过域侧闸: %v", err)
	}
	if !json.Valid(req.Input.OutputShape) {
		t.Errorf("OutputShape 应是合法 JSON: %s", req.Input.OutputShape)
	}
}

// 验收(铁律 1 配套):推理提案缺健康检查 → 提案失败,不进验证。
func TestProposeMissingHealthCheckFails(t *testing.T) {
	noHC := `{
      "summary": "缺健康检查",
      "compose": "services:\n  x:\n    image: alpine\n",
      "health_checks": []
    }`
	fe := newFakeExecutor(noHC)
	_, err := Propose(context.Background(), fe, testMaterial, 1, nil)
	if !errors.Is(err, ErrNoHealthCheck) {
		t.Fatalf("缺健康检查应提案失败(ErrNoHealthCheck), got %v", err)
	}
	if !errors.Is(err, ErrProposalInvalid) {
		t.Fatalf("应同时命中 ErrProposalInvalid, got %v", err)
	}
}

// 缺 compose 也失败。
func TestProposeMissingComposeFails(t *testing.T) {
	noCompose := `{"summary":"x","compose":"  ","health_checks":[{"name":"a","type":"tcp","host":"h","port":1}]}`
	fe := newFakeExecutor(noCompose)
	_, err := Propose(context.Background(), fe, testMaterial, 1, nil)
	if !errors.Is(err, ErrNoCompose) {
		t.Fatalf("缺 compose 应 ErrNoCompose, got %v", err)
	}
}

// 健康检查类型非法 → 失败。
func TestProposeBadHealthCheckType(t *testing.T) {
	bad := `{"summary":"x","compose":"services: {}","health_checks":[{"name":"a","type":"weird"}]}`
	fe := newFakeExecutor(bad)
	if _, err := Propose(context.Background(), fe, testMaterial, 1, nil); err == nil {
		t.Fatal("非法健康检查类型应失败")
	}
}

// 传输层失败(error 返回,无 Operation 结果)→ Propose 返回错误。
func TestProposeExecutorError(t *testing.T) {
	fe := newFakeExecutorErr(errors.New("模型不可达"))
	if _, err := Propose(context.Background(), fe, testMaterial, 1, nil); err == nil {
		t.Fatal("传输层失败应返回错误")
	}
}

// 重试(attempt>1)提示词应带上一轮失败原因,让推理针对性改。
func TestProposeRetryMentionsPrevError(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	if _, err := Propose(context.Background(), fe, testMaterial, 2, errors.New("健康检查超时")); err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if !strings.Contains(fe.prompt(0), "健康检查超时") {
		t.Error("重试提示词应带上一轮失败原因")
	}
}

// 幂等键(契约 Idempotency-Key):轮次进键 —— 同一服务同轮复用同一键,
// 下一轮(新逻辑调用)是新键;服务变,键也变。
func TestProposeIdempotencyKeyPerAttempt(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	if _, err := Propose(context.Background(), fe, testMaterial, 1, nil); err != nil {
		t.Fatalf("Propose 第 1 轮: %v", err)
	}
	if _, err := Propose(context.Background(), fe, testMaterial, 1, nil); err != nil {
		t.Fatalf("Propose 第 1 轮重试: %v", err)
	}
	if _, err := Propose(context.Background(), fe, testMaterial, 2, errors.New("不过")); err != nil {
		t.Fatalf("Propose 第 2 轮: %v", err)
	}
	reqs := fe.Requests()
	sameRound := []string{reqs[0].IdempotencyKey, reqs[1].IdempotencyKey}
	if sameRound[0] == "" || sameRound[0] != sameRound[1] {
		t.Errorf("同一服务同一轮应复用同一幂等键, got %q vs %q", sameRound[0], sameRound[1])
	}
	if reqs[2].IdempotencyKey == sameRound[0] {
		t.Errorf("第 2 轮是新逻辑调用,应产生新幂等键, got %q", reqs[2].IdempotencyKey)
	}
	for _, req := range reqs {
		if len(req.IdempotencyKey) > 200 {
			t.Errorf("幂等键超过契约上限 200: %q(len=%d)", req.IdempotencyKey, len(req.IdempotencyKey))
		}
	}
}

// 负例:UNCERTAIN/CANCELLED/FAILED 各归到可区分的类型化错误,不折叠成
// 一个"提案失败"(信任不变量:FAIL 与 ERROR/拿不准端到端可区分)。
func TestProposeStatusMapping(t *testing.T) {
	cases := []struct {
		name   string
		status airuntime.Status
		want   error
	}{
		{"uncertain", airuntime.StatusUncertain, ErrProposeUncertain},
		{"cancelled", airuntime.StatusCancelled, ErrProposeCancelled},
		{"failed", airuntime.StatusFailed, ErrProposeFailed},
	}
	all := []error{ErrProposeUncertain, ErrProposeCancelled, ErrProposeFailed}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fe := &fakeExecutor{FakeExecutor: &airuntime.FakeExecutor{
				Result: airuntime.Result{
					OperationID:       "op-1",
					Status:            c.status,
					TerminationReason: airuntime.TerminationProviderFailure,
				},
			}}
			_, err := Propose(context.Background(), fe, testMaterial, 1, nil)
			if !errors.Is(err, c.want) {
				t.Fatalf("status=%s 应归 %v, got %v", c.status, c.want, err)
			}
			for _, other := range all {
				if other != c.want && errors.Is(err, other) {
					t.Errorf("status=%s 误命中 %v:终态必须可区分", c.status, other)
				}
			}
		})
	}
}

// 负例:SUCCEEDED 但输出为空 → 明确报错,不返回空提案假装成功。
func TestProposeEmptyOutput(t *testing.T) {
	fe := &fakeExecutor{FakeExecutor: &airuntime.FakeExecutor{
		Result: airuntime.Result{Status: airuntime.StatusSucceeded},
	}}
	if _, err := Propose(context.Background(), fe, testMaterial, 1, nil); err == nil {
		t.Fatal("SUCCEEDED 但无输出应报错")
	}
}

// 脱敏闸:Hints 直进提示词(不预设脱敏),混进疑似密钥 → 域侧拒绝,
// 内容不到达传输(FakeExecutor 零调用)。
func TestProposeRedactedPromptRejected(t *testing.T) {
	mat := testMaterial
	mat.Service.Hints = "端口 9000,上游密钥 sk-abcdefghijklmnopqrst"
	fe := newFakeExecutor(goodProposal)
	_, err := Propose(context.Background(), fe, mat, 1, nil)
	if err == nil {
		t.Fatal("疑似密钥应被脱敏闸拒绝")
	}
	if n := fe.calls(); n != 0 {
		t.Errorf("被拒内容不得到达传输, got %d 次调用", n)
	}
}

// 原始材料(Raw)在提示词里先过 RedactFine 防御性脱敏:含密钥的 Raw
// 不触发拒绝闸,送出的是占位符形态(不信任上游)。
func TestProposeRawMaterialDefensivelyRedacted(t *testing.T) {
	mat := testMaterial
	mat.Raw = "services:\n  x:\n    image: alpine\n    environment:\n      - API_KEY=sk-abcdefghijklmnopqrst\n"
	fe := newFakeExecutor(goodProposal)
	if _, err := Propose(context.Background(), fe, mat, 1, nil); err != nil {
		t.Fatalf("防御性脱敏后的 Raw 不应触发拒绝: %v", err)
	}
	if got := fe.prompt(0); !strings.Contains(got, "[REDACTED]") || strings.Contains(got, "sk-abcdefghijklmnopqrst") {
		t.Errorf("Raw 中的疑似密钥应被替换成占位符: %q", got)
	}
}

// 验收:提案结构里没有判定类字段——LLM 只提候选,不判定(仿 requirement 的结构检查)。
func TestProposalSchemaHasNoVerdictFields(t *testing.T) {
	forbidden := map[string]bool{
		"verified": true, "passed": true, "pass": true, "ok": true, "success": true,
		"score": true, "judge": true, "judged": true, "valid": true, "validated": true,
		"result": true, "conclusion": true, "verdict": true, "healthy": true, "confirmed": true,
		"结论": true, "判定": true, "已验证": true,
	}
	var check func(rt reflect.Type)
	check = func(rt reflect.Type) {
		switch rt.Kind() {
		case reflect.Slice, reflect.Ptr, reflect.Array:
			check(rt.Elem())
			return
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
				if forbidden[name] {
					t.Errorf("提案结构出现判定字段 %q(json:%s):LLM 只提候选,不判定", f.Name, name)
				}
				check(f.Type)
			}
		}
	}
	check(reflect.TypeOf(proposalSchema))
}
