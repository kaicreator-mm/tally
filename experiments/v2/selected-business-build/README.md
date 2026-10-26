# Tally v2 T02b — real selected BUILD producer (experiment)

Owner [#110](https://github.com/kaicreator-mm/tally/issues/110) / parent #105. This test is **not** a new runtime or a second Kernel. It consumes public, separately npm-packed DomainHarness v0.10.2 K/S/Contracts/BUILD roots and **real Tally Business source** from #109, at exact Git SHAs. It does not deep-import upstream source; it reads only selected installed Domain Package public export-map bytes.

- DH release tag commit: `cee14399d9d0af4e20e34b9111861b16a465b690` (conditional RQ).
- T02a source: `30a469398f4e50ca3185acea22c0961da854d274` (R2 review pending; separate checkouts prevent pretending this code has already merged).
- Run Ubuntu Node24: `DH=/path/to/dh TALLY=/path/to/tally-branch TALLY_B=/path/to/t02a bash "$TALLY/experiments/v2/selected-business-build/run-clean.sh"`. It packs public tarballs, prints their true SHA256 values, installs clean producer and invokes `buildV10App` with genuine K/S/Tally B source bytes.
- Positive: exactly one K, one S, one selected Tally B, sealed source hash/digest and load order. Negatives: duplicate K, B wrong ABI, B digest forgery. All negative build failures are genuine public BUILD refusals.
- Out of scope **by design**: K `send()`, real B Effect `resource:null` support, admission/Claim authorization, journal replay, GitHub writes, selected C, Host SQLite, Hidden tests, Release RQ. Those remain `NOT_RUN` until a separate clean consumer is added to this task. CI success here is static BUILD acceptance **only**. No private task-meta integration.

CI workflow uses `contents:read`, no secrets or privileged GitHub mutations. **Never label a BUILD-only PASS as a full DomainHarness integrated Task PASS.**
