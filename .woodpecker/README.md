# Tally v2+ — Woodpecker CI (single clone, targeted tests)

This directory intentionally contains **one active YAML workflow**: `ci.yaml`.
Tally v1.x and its Go `verify.yaml` are retired. Do not add Go or rebuild
NILAR / CC-LAN200 for a Node/TypeScript Tally v2+ release. Historical YAML and
CI results remain discoverable through Git history and their old commit SHAs.

## Why one workflow

With Woodpecker 3.x, each YAML in `.woodpecker/` is a distinct workflow,
and distinct workflows clone the repository independently. All v2+ test steps
now share one workspace after **one** Woodpecker Git checkout per pipeline.

The built-in Git plugin uses a partial clone and fixed sparse checkout:
only current v2 app, selected packages, and necessary v2 experiment modules
are included. It excludes `docs/**`, `knowledge/**`, legacy `cmd/**`,
`internal/**`, Go files and entire Git history, except for Git metadata
needed to identify the exact source SHA. Submodules and LFS are disabled.
The sparse path set is intentionally static. A new v2+ module must be added
to both `clone.settings.sparse` and the relevant `when.path` rules.

- `main` and `version/*` can run the same Node24 checks. This includes
  `version/v2` now and later version branches without renaming CI files.
- The workflow triggers on actual relevant **code/config** modifications, not
  ordinary Markdown/product/architecture documentation changes. A docs-only PR
  therefore yields **CI NOT_RUN/SKIPPED**, not a fabricated PASS. Independent
  content review remains separate when required.
- Individual steps have their own changed-path filters. Unaffected units,
  front-end builds and provider suites are skipped.
- Node and npm dependencies are installed only in selected steps and reused
  through the same workspace where already present. Local npm caches may help,
  but no cross-agent cache or external artifact service is assumed.
- No step executes `git clone`, `git fetch` or a live GitHub observation.
  The workflow no longer repeatedly clones DomainHarness or ADS for historical
  T06/GitHub-only experiments. Any new external code dependency requires a
  separate explicit, scoped qualification decision.
- Exact HEAD is checked against `CI_COMMIT_SHA`. Ordinary head commits
  require a `Refs:` trailer; actual merge commits are identified from the
  fetched commit object without fetching extra Git history. Complete PR history
  is reviewed in PR workflow/review rather than downloaded separately in every
  CI step.
- CI is **source verification**, not a Windows Codex Host, RQ signature,
  GitHub write permission, real installed browser witness, or deployment.

## Scope of targeted steps

The same workflow covers v2 ALM/business/Host no-effect tests, React/Vite UI,
the shared AgentProvider contract and provider adapters, native GitHub adapter,
release delivery negative tests, guided intake, candidate publication, and
offline observation. Real GitHub/Host writes never occur in the CI pipeline.

## When validation fails

A red step is red. Never reinterpret infrastructure failures or skips as
SUCCESS. Prefer confirming the changed module locally first and invoking CI
only where independent execution, provider-specific environment or required
checks actually matter. Preserve exact SHA and step receipts; no empty commits
to force a rerun.

Prior proof records are still preserved under their historical commits,
including v2.0.0 tag `61ede389719dec72f633519918f4b9c5a4d33846`.
This change does not reexecute the release or alter installed Host effects.

Refs: #494 #392 #259 #208
