/**
 * Tally AgentProvider/1 Node adapter for the xAI official Grok Build CLI.
 *
 * Status: v2.1 candidate, **default DISABLED**. The registry default provider
 * is `codex`; `grok` is never in any default enabled list and this
 * deployment holds no real Grok credentials, so `probe` reports UNAVAILABLE
 * (GROK_NOT_AUTHENTICATED) and `execute` refuses to spawn. All CLI facts are
 * observations carried from PR #468 (grok 1.0.87 / commit 929e6dd187b1 /
 * win32-x64, observed 2026-11-19). See README.md for enablement conditions
 * and the honest NOT_RUN list.
 */
export {
  GROK_PROVIDER_ID,
  PINNED_GROK_VERSION,
  GROK_PERMISSION_MODES,
  GrokProviderRefusalError,
  parseGrokVersionLine,
  createGrokAgentProvider,
} from './grok-agent-provider.mjs';
