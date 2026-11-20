/**
 * @kaicreator/tally-provider-opencode — narrow entry.
 *
 * Tally v2.1 candidate: AgentProvider/1 adapter for the OpenCode CLI
 * (opencode-ai@1.19.6). Observation-only, default-off, no fallback, no
 * scheduler, no execution journal. CLI contract provenance: PR #471 field
 * evidence (Go observations are input evidence; this package is the Node
 * adapter).
 */
export {
  createOpenCodeAgentProvider,
  OPENCODE_PINNED_VERSION,
  OPENCODE_PROVIDER_ID,
} from './opencode-agent-provider.mjs';
