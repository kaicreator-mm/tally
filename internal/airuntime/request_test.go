package airuntime

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validRequest() Request {
	return Request{
		Operation:      "tally.requirement.interview",
		Intent:         Intent{Role: "requirement-interviewer", Instructions: "你是需求问询助手。"},
		Input:          Input{Prompt: "问:这个环境要部署到哪里?", OutputShape: json.RawMessage(`{"answer":""}`)},
		Budget:         Budget{MaxAttempts: 1, MaxCostCeilingUSD: 0.5, DeadlineMs: 60000, MaxTokens: 4096},
		IdempotencyKey: "task-42/interview/attempt-1",
	}
}

func TestRequestValidateAcceptsValidRequest(t *testing.T) {
	if err := validRequest().Validate(); err != nil {
		t.Fatalf("合法请求被拒绝: %v", err)
	}
}

func TestRequestValidateRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Request)
		want   error
	}{
		{"空操作 ID", func(r *Request) { r.Operation = "" }, ErrNoOperation},
		{"空白操作 ID", func(r *Request) { r.Operation = "   " }, ErrNoOperation},
		{"缺角色名", func(r *Request) { r.Intent.Role = "" }, ErrNoIntent},
		{"缺角色指令", func(r *Request) { r.Intent.Instructions = "" }, ErrNoIntent},
		{"缺幂等键", func(r *Request) { r.IdempotencyKey = "" }, ErrInvalidIdempotencyKey},
		{"幂等键超长", func(r *Request) { r.IdempotencyKey = strings.Repeat("k", 201) }, ErrInvalidIdempotencyKey},
		{"尝试次数为 0", func(r *Request) { r.Budget.MaxAttempts = 0 }, ErrInvalidBudget},
		{"时限为 0", func(r *Request) { r.Budget.DeadlineMs = 0 }, ErrInvalidBudget},
		{"token 上界为 0", func(r *Request) { r.Budget.MaxTokens = 0 }, ErrInvalidBudget},
		{"成本上界为负", func(r *Request) { r.Budget.MaxCostCeilingUSD = -0.01 }, ErrInvalidBudget},
		{"空提示词", func(r *Request) { r.Input.Prompt = "" }, ErrInvalidInput},
		{"输出形状非 JSON", func(r *Request) { r.Input.OutputShape = json.RawMessage(`{not-json`) }, ErrInvalidInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.mutate(&req)
			if err := req.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestRequestValidateRedactionGate(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Request)
	}{
		{"提示词带密钥", func(r *Request) { r.Input.Prompt = "用这个 key: sk-AbCdEfGhIjKlMnOpQrStUv" }},
		{"角色指令带密钥", func(r *Request) { r.Intent.Instructions = "Authorization: Bearer abc123secret" }},
		{"输出形状带密钥", func(r *Request) {
			r.Input.OutputShape = json.RawMessage(`{"token":"sk-AbCdEfGhIjKlMnOpQrStUv"}`)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.mutate(&req)
			if err := req.Validate(); !errors.Is(err, ErrRedacted) {
				t.Fatalf("want %v, got %v", ErrRedacted, err)
			}
		})
	}
	// 正常内容不误伤。
	if err := validRequest().Validate(); err != nil {
		t.Fatalf("正常内容被脱敏闸误伤: %v", err)
	}
}
