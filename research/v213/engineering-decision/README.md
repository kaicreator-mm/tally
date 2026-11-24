# Tally Engineering Decision — v2.1.3 T2/T3 native research Package

Status: IMPLEMENTED_AND_EXECUTED_SCOPED / INDEPENDENT_REVIEW_PENDING / NO_PRODUCT_INTEGRATION. Scope [#563](https://github.com/kaicreator-mm/tally/issues/563), frozen design [#564](https://github.com/kaicreator-mm/tally/pull/564) reviewed 06a2d33caad619b7d6e4839b80f0079612b8f697, merged c6d2c8404072f38f2bf946a7f77ce0c712621516 with zero file delta. No #286 authorization, no release or actual production business connection.

## Usage and responsibility

One native TypeScript Business Element `EngineeringDecisionElement` owns an exact knowledge projection (Git Blob f9657c07a72c134e6d21e6aa7523b9982ffb327f, source SHA 3c9241d7a08a39b8a9865b7b7d221339a0f145e3). Import the Package root after compilation, provide the exact text of docs/versions/v2.1.3/abstraction-knowledge-projection.json to its constructor. No duplicate corpus, no mutable registry, no new framework, no initialization resource or empty Init/Exit. The native SDK's BoundedQueryElement selects the existing DP-K02/DP-K03 or K05 references and related concepts. The actual decision logic is source-derived ordinary TypeScript code, not a generic rule interpreter. Graph edges are contextual knowledge, not automatic inference or permissions. Projection identity checking is business knowledge revision binding, not runtime Package validation.

`assessAbstraction(request, checkedContext, queryLimits?)` returns an actual advisory decision for supplied candidate-specific facts. `assessContractEvidence(request, checkedContext, queryLimits?)` evaluates the authorized obligation universe, including distinct scope cell IDs, target and environment; passing verified cells can jointly cover an explicitly declared obligation, failure coverage describes actual counterexample cells, not all tests run. Same-scope qualified contradictory observations stay CONFLICT; independent unconflicted failure outranks incomplete universe. `run(request, checkedContext, workflowLimits, queryLimits?, validation?)` uses SDK invokeOutcome and shares one root budget for both operations. COMPLETE is evaluation completion, not engineering QA PASS; no supported candidate returns UNKNOWN with a bounded JSON diagnostic assessment in reason.message, because the SDK Stop type exposes only code/message. No self-report has authority. Supplied normal observations and verified context are separate parameters; context records are provided exclusively by a trusted composition caller who has actually checked content and authority (not a string label, transport credential or signature). Checked records remain available even if an ordinary observation is omitted or replaced. Facts bind full candidate description/content as well as record and evidence refs. Scope cells and positive/negative factual values are prepared by the trusted caller; conflicts in facts are explicitly consolidated as CONFLICT. This is not an authentication service, automatic source-fact extraction, proof of correctness, or untrusted JSON network parser. Do not expose the confirmation parameter to an untrusted API. Current fixtures construct synthetic confirmed contexts for testing; future integration must bind actual GitHub/Host/Reviewer authority. Falsifier strings are proposed checks against the cited fact/contract, not executed tests or fully generated test code.

## Execution evidence (R4)

Node v22.16.0 Linux x64, TypeScript 5.8.3, locally available @types/node 22.19.7. Read the applicable DomainHarness 0.11.2 Skill and core/element-package/typescript-profile/composition-trust/engineering-quality specs and actual packed native API declarations at accepted train 18384259e36ba56d937416ed5262bac1de6689ae. Reused the [R63 pack and archived Linux dependency environment](../../../docs/versions/v2.1.3/sdk-preflight.md); independently rechecked SHA256 pack a3cd22d2a514fb2f6321218b1dba27b9bd5b17b6fac7bbe2daf00b02cc25a5ff and archive e63c173b1a2577d322eec4c6a09bbd039d890508faf777747a7864546220cb2e, SDK installed folder compared to separately extracted pack (diff exit 0). Original projection reconstructed exactly from GitHub and git hash-object verified. No SDK mock, no remote CI, no local Agent, no registry claim.

To reproduce in an environment already having the exact packed SDK and its dependencies, the selected TypeScript and Node typings, use this directory as working directory. The preserved archive contains dh-native-sdk-only-VOKXqJ/consumer/node_modules; only that subtree was restored, not the archived consumer program. Put that directory at node_modules (or link it), compare the SDK files with the pack before executing; supply a typeRoots directory containing the actual Node typings if not already installed:

```sh
tsc --typeRoots /path/to/type-roots --noEmit
tsc --typeRoots /path/to/type-roots
node --check dist/model.js
node --check dist/engineering.js
node --check engineering.test.mjs
node --test engineering.test.mjs
```

Final actual result: strict typecheck PASS, compile PASS, JS syntax 3/3 PASS, Node tests **57/57 PASS**. The 43 design IDs A01–A44 except A24 are exercised (A05 has nine independent dimension probes; several others run sub-probes). Six additional X01–X06 tests cover candidate content substitution, omission of checked failure, candidate-local conflict, projection tamper, independent failure cell and foreign target. A17 is an encoded representation of the archived R20 observation, not a new source execution or requalification. A24 actual new source consumption remains NOT_RUN until T4. Test coverage is scoped, not full product acceptance. No automatic extraction of qualified engineering facts and no real production authority binding.

Three finite targeted development mutations, each applied locally and restored: remove query completeness gate (A20 fails even when all matches fit), promote partial pass (A18/A43/A44 fail), remove Strategy's authorized replacement need (A09/A33 fail). All three runs exit 1; not a comprehensive mutation score. The combined command reached a tool timeout after mutations and restoration during trailing verification; separate complete final strict typecheck/build/syntax/test rerun exited 0, all 57 passed and original source hash matches. No mutation committed.

| Locally tested Git Blob | File |
|---|---|
| c74c66e5a4fa35d03aa8043288d41fafe1ac841c | package.json |
| 9df81d333d3b6ce6e2dcaaf5ada71ea50939f25c | tsconfig.json |
| 83a2cba461960a62f00532cfc0c20f0376be02eb | model.ts |
| 62f92ddb0808a52bde77a5079d27b435bb4265b7 | engineering.ts |
| 2dd54f56751343477e95eec9661927feaaabe47e | engineering.test.mjs |

Final emitted JS SHA256 engineering.js e9aeb4e718fb5bfd5bb2986fbf6206441c2cba8f904d4aca6f32e72f07ebf9fc; model.js 6727081647570fcaaad6a0f53037a8e50a298db651abdbb05d62eddabb4ab441. Emitted JS/typings, node_modules and tarballs are not committed.

## Limits and next step

This is isolated native SDK-only research, not registry/installable production delivery. SDK 0.11.2 has not been released; actual GitHub direct clone failed DNS in this environment. No new npm install or generated package-lock for this research Package; the archived SDK runtime dependency closure is the only verified current install. Dev tooling is locally installed; no claim of a new locked clean install or external package publication, full repo build/lint, Windows coverage, real KDP/Host/Release, complete DIS conformance or knowledge quality uplift. A fresh package-lock/clean install must be addressed when actual distribution is authorized; do not invent registry integrity or lock evidence. The research Package is not wired into the monorepo production workspaces. Review only this complete T2/T3 delta, its scoped tests and contract. Then select an authorized actual engineering decision (T4/A24) with source/contract/confirmation and behavior verification; no repeat local-Agent experiment.
