# Tally v2.0.1 cross-platform runtime status

## Runtime contract

The v2 web UI, GitHub source read and domain/ALM read models run on Node.js >=22.16.0. The supported application entrypoint is `node web/v2/serve.mjs` from a source checkout or a **platform-specific installed runtime with its dependencies already built**. Open the emitted `TALLY_PROJECT_SETUP` URL; the local HTTP listener remains on `127.0.0.1`. Do not supply global `TALLY_V2_REPOSITORY` / SHA variables for ordinary first-run setup.

| Platform | First-run configuration storage | Requirements |
|---|---|---|
| Windows | DPAPI CurrentUser; existing `.dpapi` files retained | Windows PowerShell, same logged-in user |
| Linux | Secret Service via `secret-tool`; catalog stores an opaque key reference | `libsecret-tools`, an active unlocked user Secret Service over D-Bus (GNOME Keyring / KWallet equivalent) |
| macOS | login Keychain via `/usr/bin/security`; catalog stores an opaque reference | unlocked user Keychain; allow any OS prompts |

On Linux the default catalog path is `${XDG_CONFIG_HOME:-$HOME/.config}/tally/v2`, on macOS `$HOME/Library/Application Support/Tally/v2`, and on Windows the existing `%LOCALAPPDATA%/Tally/v2`. An explicit `TALLY_V2_CONFIG_DIR` overrides this. The catalog JSON stores only repository identities, refs and observed SHAs, not tokens. Linux/macOS `credential-*.keyring` files contain references; deleting the OS keyring secret causes `CREDENTIAL_UNAVAILABLE`. Missing service/locked keyring causes `CREDENTIAL_STORE_UNAVAILABLE` and setup stays unconfigured: no plaintext fallback.

The Windows R145 ZIP (and its `better-sqlite3` native binary) is **not** a portable Linux/macOS package. Install/rebuild native dependencies on the target platform; never unpack Windows native `.node` modules as if they were Linux/macOS builds.

## Selected Agent Host: Linux support candidate (#516 / PR #517)

The existing sealed K/B/C, native GitHub Claim and frozen Write Set admission,
CAS resource grants, selected Provider, candidate tree/commit readback and
one-Builder Task semantics are reused. Windows keeps its previously qualified
SID/ACL/PowerShell execution path unchanged.

For Linux, the newly added `local-codex-linux-host.mjs` uses a **rootless
bubblewrap** namespace (user UID/GID 65534, PID, IPC and UTS), with exactly
one writable Git worktree mounted at `/workspace`, a dedicated private Builder
model-home mounted at `/home/tally-builder`, a separate empty Git config and
only readonly system runtime/certificate paths. The operator's home, private
GitHub/SSH credentials, hidden fixture and Host CAS database are **not mounted**.
A real namespace UID probe is mandatory before a new worktree is allocated.
The Host then probes filesystem confinement and Codex login inside the
sandbox before launching. Errors or unavailable namespaces deny execution.
No unsandboxed fallback exists.

Host-only Linux configuration (not browser-supplied):

- `TALLY_V2_AGENT_PLATFORM=linux`, `TALLY_V2_AGENT_DISPATCH=host-mounted`;
  all existing exact Claim/FSW/attempt/Task pins remain required.
- A freshly sealed **Linux-native** `TALLY_V2_AGENT_PREBUILT_CONSUMER_DIR`
  and complete `T06M_TARGET_REPOSITORY`, `T06M_TARGET_ISSUE_NUMBER`,
  `T06M_TARGET_VERSION_REF`, `T06M_TARGET_CANDIDATE_PATH` that match
  the mounted source/Issue; never reuse the earlier Windows R145 prebuild.
- `TALLY_LINUX_BWRAP_PATH=/usr/bin/bwrap`; `TALLY_CODEX_CLI_PATH` must
  be an absolute executable path within `/usr` (e.g. `/usr/local/bin/codex`).
  The host must be a **non-root** user on a Linux kernel allowing rootless
  user namespaces. The required user namespace is never silently skipped.
- A dedicated `TALLY_BUILDER_PROFILE` private (0700) directory,
  `TALLY_CODEX_HOME=$TALLY_BUILDER_PROFILE/.codex` with the *separate* model
  login, `TALLY_BUILDER_TEMP`, `TALLY_EMPTY_GIT_CONFIG`, and
  `TALLY_HIDDEN_FIXTURE`. Keep them outside the exact editable worktree and
  never point Builder home at the operator home itself.
- `TALLY_BUILDER_PRINCIPAL=linux-bwrap:65534` and legacy sealed-result field
  `TALLY_BUILDER_SID=linux-uid:65534` are **typed Linux namespace identity
  markers, not Windows security identifiers**. The verifier marker must differ.
  Tally intentionally retains the sealed C's process evidence *field name*
  while refusing to equate Linux UID with a Windows SID.
- The new Linux physical consumer runs a rebuilt sealed K app plus the **same**
  candidate/commit validation. `TALLY_GIT_PATH` should be a Linux Git
  executable; native SQLite and Node packages must be rebuilt for Linux/arch.

The Linux path currently supports the selected **Codex** Builder only. Other
AgentProviders and the Windows-only single-review consumer are not silently
promoted to Linux; unsupported positive routes stay fail-closed. A real
Builder -> candidate -> Draft PR requires one physical Linux Host/Claim
acceptance, rather than substituting an injected unit-test process. This is
still **NOT_RUN** until that Host's exact Linux install, namespace, login,
recompiled Consumer, CAS and GitHub readback receipts exist.

macOS has an OS-native *configuration keyring* adapter but **no qualified
Agent Host** yet; it is not a v2.0.1 Linux/Windows selected-Builder release
claim. Full installed UI J01-J08 acceptance (#513) and official Release
(#512) remain held until the actual required v2.0.1 scope is proven.
