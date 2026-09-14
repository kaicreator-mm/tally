// aiexecutor.go is the server composition-root wiring for the Intent proposal
// executor. Domain handlers only see airuntime.Executor; provider/model/endpoint
// configuration remains a transport concern.
package main

import (
	"errors"

	"tally/cmd/internal/airuntimelegacy"
	"tally/internal/airuntime"
)

var errAIRuntimeNotConfigured = errors.New(
	"AI Runtime 未配置:需要 TALLY_LLM_API_KEY(或 OPENAI_API_KEY),或显式 TALLY_LLM_BASE_URL")

func aiProposeExecutorFromEnv() (airuntime.Executor, error) {
	if !aiRuntimeConfigured() {
		return nil, errAIRuntimeNotConfigured
	}
	return airuntimelegacy.FromEnv(), nil
}

func aiRuntimeConfigured() bool {
	return envOr("TALLY_LLM_API_KEY", "") != "" ||
		envOr("OPENAI_API_KEY", "") != "" ||
		envOr("TALLY_LLM_BASE_URL", "") != ""
}
