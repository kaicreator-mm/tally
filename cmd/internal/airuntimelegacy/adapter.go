// Package airuntimelegacy is the single composition-root adapter from the
// legacy OpenAI-compatible internal/llm transport to the provider-neutral
// internal/airuntime boundary.
//
// It is intentionally located under cmd/internal: domain packages cannot import
// it, while server/worker/CLI composition roots can share one transitional
// implementation. Replace this package wholesale when the real public AI
// Runtime SDK transport is wired.
package airuntimelegacy

import (
	"context"
	"encoding/json"
	"errors"

	"tally/internal/airuntime"
	"tally/internal/llm"
)

// ErrNilClient indicates a composition error. It is exported only so adapter
// tests/diagnostics can classify the failure; domain code must never depend on
// this legacy transport error.
var ErrNilClient = errors.New("airuntimelegacy: 未配置 llm 客户端(装配错误)")

// FromEnv constructs the transitional transport from the existing TALLY_LLM_*
// environment contract and wraps it with the common airuntime validation gate.
func FromEnv() airuntime.Executor {
	return airuntime.Validated(executor{client: llm.NewHTTPClient(llm.ConfigFromEnv())})
}

type executor struct{ client *llm.HTTPClient }

func (e executor) Execute(ctx context.Context, req airuntime.Request) (airuntime.Result, error) {
	if e.client == nil {
		return airuntime.Result{}, ErrNilClient
	}
	c := e.client.WithRole(llm.Role{Name: req.Intent.Role, Instructions: req.Intent.Instructions})
	raw, err := c.Propose(ctx, req.Input.Prompt, json.RawMessage(req.Input.OutputShape))
	if err != nil {
		return airuntime.Result{}, err
	}
	return airuntime.Result{
		Status:            airuntime.StatusSucceeded,
		TerminationReason: airuntime.TerminationCompleted,
		Output:            raw,
	}, nil
}

var _ airuntime.Executor = executor{}
