# Tally v1.5.2 — Decision Locks

These locks are subordinate to the frozen v1.5.2 PRD and preserve the v1.5.0 execution contracts.

1. **Runtime and Environment stay orthogonal.** Runtime defines Agent protocol; Environment defines process/workspace placement and operations. No Runtime×Environment combination types.
2. **Frozen v1.5.0 interfaces are not redefined.** v1.5.2 behavior is introduced through additive optional interfaces and compatible fields.
3. **Local compatibility is preserved.** `LocalEnvironment` and traditional `Runtime.Start` remain the Local path. Non-Local execution never silently falls back to it.
4. **Workspace is environment-native.** Core may hold a locator but must not require an SSH/Docker workspace to be readable by the host filesystem.
5. **Pinned Git/base revision is execution authority.** Workspace resolution and restart re-verify the accepted base revision inside the selected environment.
6. **Lost workspace recovery is bounded.** A configured recreation seam may recreate once from accepted authority and then must re-verify. Session resume is not a correctness dependency.
7. **Capability mismatch is pre-start and typed.** Static and discovered OS/architecture/toolchain/Git/Docker/network/secret/resource requirements are checked before Agent start.
8. **Secret values do not enter profile/provenance.** Only logical availability/policy is represented.
9. **SSH host identity is fail-closed by default.** Explicit callback or known_hosts is preferred. Any insecure host-key bypass is explicit transport configuration, never an implicit default.
10. **Cancellation and cleanup are distinct.** Process cancellation propagates through the environment transport; workspace cleanup is an explicit environment operation with unsafe-path guards.
11. **Reuse existing runners.** Do not add a second SSH/Docker transport stack or new external dependency unless later evidence requires it.
12. **No CI gate.** GitHub Actions and all CI systems remain outside v1.5.2 development/release gates. Validation evidence is Local/Build Host plus external held-out validation.
13. **External absence is not implementation blockage.** Missing real SSH/Docker may be `ENV-BLOCKED` only with evidence; adapter/contract/fake/failure-path work still must complete.
14. **No scope expansion.** Fleet scheduling, Kubernetes, cloud provisioning, distributed queue, GitHub authority, Multi-Agent orchestration and re-plan remain out of scope.
15. **PR merge is not Release PASS.** A tag/baseline requires the explicit v1.5.2 release qualification ledger to have acceptable mandatory evidence.
