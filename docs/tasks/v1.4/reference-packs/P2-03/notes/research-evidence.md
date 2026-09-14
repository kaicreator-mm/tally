# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

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
