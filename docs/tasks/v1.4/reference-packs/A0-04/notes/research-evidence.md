# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

## REF-SWE-RETRY — SWE-agent/SWE-agent
- Pin: `3ea751c087f32b16e039a2233dd6eefecef325d5`
- License: `MIT`
- URL: https://github.com/SWE-agent/SWE-agent/blob/3ea751c087f32b16e039a2233dd6eefecef325d5/sweagent/agent/reviewer.py
- Relevant paths:
  - `sweagent/agent/reviewer.py`
  - `sweagent/agent/agents.py`
- Why relevant: Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-TEMPORAL-ERR — temporalio/sdk-go
- Pin: `82c8087ecd03722e2b8e7c6c892784095fa94e73`
- License: `MIT`
- URL: https://github.com/temporalio/sdk-go/blob/82c8087ecd03722e2b8e7c6c892784095fa94e73/temporal/error.go
- Relevant paths:
  - `temporal/error.go`
  - `internal/error.go`
  - `internal/internal_task_handlers.go`
- Why relevant: Uses typed application errors plus explicit non-retryable classification and RetryPolicy; useful for deterministic routing without string-matching error messages.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-SPECBENCH — WecoAI/SpecBench
- Pin: `08607352adc8abd78be2193dd9f725f1f032b8f0`
- License: `Apache-2.0`
- URL: https://github.com/WecoAI/SpecBench
- Relevant paths:
  - `benchmarks/spec_bench/evaluation/runner.py`
  - `benchmarks/spec_bench/run_loop.py`
  - `benchmarks/spec_bench/adapter.py`
- Why relevant: Separates visible validation from held-out evaluation and measures the gap; useful for L1/L3 capture comparisons without leaking evaluation evidence.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-EVILGENIE — JonathanGabor/evilgenie_inspect
- Pin: `f5d8a2ca5d92ecf5fd1e857695c54e609312e1bd`
- License: `MIT`
- URL: https://github.com/JonathanGabor/evilgenie_inspect
- Relevant paths:
  - `categorical_scorer.py`
  - `tests/test_scorer.py`
  - `tests/test_split_determinism.py`
  - `reward_hacking.py`
- Why relevant: Combines held-out tests, LLM judge, and test-file mutation detection; tests scorer dimensions independently and keeps dataset splits deterministic.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.
