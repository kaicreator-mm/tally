package airuntime

import (
	"encoding/json"
	"testing"
)

// 本文件把 Request/Result 的 JSON 形状钉死在公共 AI Runtime 冻结契约上
// (api/openapi.yaml v0.4.0-m1-frozen,ADR-0002)。字段名漂移会在这里先炸,
// 而不是在 P0-10/P0-11 接真实 SDK 时才炸。

func TestRequestWireShapeMatchesContract(t *testing.T) {
	raw, err := json.Marshal(validRequest())
	if err != nil {
		t.Fatalf("序列化 Request: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("解析 Request 体: %v", err)
	}

	// OperationRequest required: [operation, intent, input, budget]。
	// 幂等键是头(Idempotency-Key),绝不能出现在请求体里。
	for _, key := range []string{"operation", "intent", "input", "budget"} {
		if _, ok := body[key]; !ok {
			t.Errorf("请求体缺契约必填字段 %q", key)
		}
	}
	for _, key := range []string{"idempotency_key", "IdempotencyKey", "idempotencyKey"} {
		if _, ok := body[key]; ok {
			t.Errorf("幂等键 %q 混进了请求体:它只能走 Idempotency-Key 头", key)
		}
	}

	var budget map[string]json.RawMessage
	if err := json.Unmarshal(body["budget"], &budget); err != nil {
		t.Fatalf("解析 budget: %v", err)
	}
	// budget.schema.json required: [max_attempts, max_cost_ceiling, deadline_ms, max_tokens]。
	for _, key := range []string{"max_attempts", "max_cost_ceiling", "deadline_ms", "max_tokens"} {
		if _, ok := budget[key]; !ok {
			t.Errorf("budget 缺契约必填字段 %q", key)
		}
	}
}

func TestIntentWireShapeIsOpenObject(t *testing.T) {
	// 契约 intent: {type: object, additionalProperties: true};
	// Tally 侧类型化字段必须能直接落在开放对象上。
	raw, err := json.Marshal(Intent{Role: "r", Instructions: "i"})
	if err != nil {
		t.Fatalf("序列化 Intent: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Intent 不是 JSON 对象: %v", err)
	}
	if m["role"] != "r" || m["instructions"] != "i" {
		t.Errorf("Intent 字段落点漂移: %s", raw)
	}
}

func TestEnumValuesMatchFrozenContract(t *testing.T) {
	statuses := map[Status]string{
		StatusSucceeded: "SUCCEEDED",
		StatusUncertain: "UNCERTAIN",
		StatusFailed:    "FAILED",
		StatusCancelled: "CANCELLED",
	}
	for got, want := range statuses {
		if string(got) != want {
			t.Errorf("Status 常量漂移: got %q, want %q", got, want)
		}
	}
	terminations := map[TerminationReason]string{
		TerminationAttemptsExhausted: "ATTEMPTS_EXHAUSTED",
		TerminationBudgetExhausted:   "BUDGET_EXHAUSTED",
		TerminationClientCancelled:   "CLIENT_CANCELLED",
		TerminationCompleted:         "COMPLETED",
		TerminationDeadlineExceeded:  "DEADLINE_EXCEEDED",
		TerminationPolicyRejected:    "POLICY_REJECTED",
		TerminationProviderFailure:   "PROVIDER_FAILURE",
	}
	for got, want := range terminations {
		if string(got) != want {
			t.Errorf("TerminationReason 常量漂移: got %q, want %q", got, want)
		}
	}
	strategies := map[StrategyAssignment]string{
		StrategyBaseline:      "BASELINE",
		StrategyCanary:        "CANARY",
		StrategyShadowPrimary: "SHADOW_PRIMARY",
	}
	for got, want := range strategies {
		if string(got) != want {
			t.Errorf("StrategyAssignment 常量漂移: got %q, want %q", got, want)
		}
	}
}
