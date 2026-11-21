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

## Privileged Agent and Host boundary

Cross-platform first-run/project read support does not establish cross-platform *Agent execution*. The v2.0.1 selected physical Codex consumer `local-codex-windows-e2e.mjs` and its Host use Windows SID, ACL and `icacls` to establish privileged isolation. That positive execution qualification is Windows-only. Linux/macOS require an independently validated equivalent isolation-capable Host/driver before exposing a positive `Agent start` result. A browser-selected project, a Secret Service/Keychain credential, or a keyring unit test must never grant Agent/Task/Release authority.

Scope of current source tests: injected provider read/write, missing-provider refusal, persisted references, no tokens in helper arguments, and platform configuration directories. Their passing CI is not a physical Linux/macOS keyring/sandbox or full J01-J08 installed acceptance. Official Release remains held by #513 / #512 until those requirements are satisfied.