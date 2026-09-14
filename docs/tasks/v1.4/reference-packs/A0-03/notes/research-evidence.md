# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

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
