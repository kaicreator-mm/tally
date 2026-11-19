/**
 * Pi provider spawn-environment hardening: per-attempt isolation directory
 * groups and the credential-stripping / forced-key spawn env construction.
 * Pure, testable functions — no process is spawned from this module.
 *
 * Evidence source (ported to Node from PR #469 `runtimes/pi/isolation.go`,
 * which encodes host observations of pi 0.85.1 on 2026-11-19; every list and
 * key below is transcribed from that evidence, nothing invented):
 *
 *   - 42 provider credential env vars observed from `pi --help` are stripped
 *     from the inherited environment case-insensitively (Windows normalizes
 *     env var names), so operator credentials never silently flow into
 *     candidate execution. The ONLY explicit injection channel is the Host
 *     supplied `context.env` (the Go candidate's `Config.Env`).
 *   - 6 pi config/behavior env switches are likewise stripped from the
 *     inherited environment and re-imposed by isolation semantics.
 *   - Six forced isolation keys (HOME, USERPROFILE, PI_CODING_AGENT_DIR,
 *     PI_CODING_AGENT_SESSION_DIR, PI_OFFLINE=1, PI_TELEMETRY=0) override
 *     both the inherited environment AND any same-named injected value
 *     (case-insensitive matching; case-variant duplicates never coexist).
 *   - Per-attempt temp directory group: root + agent-config + sessions +
 *     home. pi scaffolds empty auth.json/models-store.json into the
 *     redirected config dir (observed), the owner's ~/.pi is untouched
 *     (observed mtime unchanged), and `--no-session` plus the redirected
 *     session dir keep session files off the host (defense in depth).
 *   - PI_OFFLINE=1 / PI_TELEMETRY=0 are egress hardening (observed: offline
 *     mode does not block a real model turn).
 *   - A fully empty env is deliberately NOT used: unobserved survival of pi's
 *     tool subprocesses and Windows TLS (SystemRoot) under an empty env, so
 *     the OS functional surface (PATH etc.) is preserved — it is not an
 *     authentication channel.
 */

import {mkdtempSync, mkdirSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';

/**
 * All provider credential environment variables observed from the pi 0.85.1
 * help text (PR #469 isolation.go). Stripped from the inherited environment
 * on every spawn; explicit injection via Host `context.env` is the single
 * sanctioned exception channel.
 */
export const PI_CREDENTIAL_ENV_VARS = Object.freeze([
  'ANTHROPIC_AUTH_TOKEN',
  'ANTHROPIC_API_KEY',
  'ANTHROPIC_OAUTH_TOKEN',
  'ANT_LING_API_KEY',
  'OPENAI_API_KEY',
  'AZURE_OPENAI_API_KEY',
  'AZURE_OPENAI_BASE_URL',
  'AZURE_OPENAI_RESOURCE_NAME',
  'AZURE_OPENAI_API_VERSION',
  'AZURE_OPENAI_DEPLOYMENT_NAME_MAP',
  'DEEPSEEK_API_KEY',
  'NVIDIA_API_KEY',
  'GEMINI_API_KEY',
  'GROQ_API_KEY',
  'CEREBRAS_API_KEY',
  'XAI_API_KEY',
  'FIREWORKS_API_KEY',
  'TOGETHER_API_KEY',
  'BASETEN_API_KEY',
  'OPENROUTER_API_KEY',
  'AI_GATEWAY_API_KEY',
  'ZAI_API_KEY',
  'ZAI_CODING_CN_API_KEY',
  'MISTRAL_API_KEY',
  'MINIMAX_API_KEY',
  'MOONSHOT_API_KEY',
  'OPENCODE_API_KEY',
  'KIMI_API_KEY',
  'CLOUDFLARE_API_KEY',
  'CLOUDFLARE_ACCOUNT_ID',
  'CLOUDFLARE_GATEWAY_ID',
  'QWEN_TOKEN_PLAN_API_KEY',
  'QWEN_TOKEN_PLAN_CN_API_KEY',
  'XIAOMI_API_KEY',
  'XIAOMI_TOKEN_PLAN_CN_API_KEY',
  'XIAOMI_TOKEN_PLAN_AMS_API_KEY',
  'XIAOMI_TOKEN_PLAN_SGP_API_KEY',
  'AWS_PROFILE',
  'AWS_ACCESS_KEY_ID',
  'AWS_SECRET_ACCESS_KEY',
  'AWS_BEARER_TOKEN_BEDROCK',
  'AWS_REGION',
]);

/**
 * pi config/behavior environment switches observed from the pi 0.85.1 help
 * text: stripped from the inherited environment, then re-imposed per the
 * isolation semantics (offline/telemetry hardening; config/session directory
 * redirection; package dir discovery-face guard; share URL drift guard).
 */
export const PI_CONFIG_ENV_VARS = Object.freeze([
  'PI_CODING_AGENT_DIR',
  'PI_CODING_AGENT_SESSION_DIR',
  'PI_PACKAGE_DIR',
  'PI_OFFLINE',
  'PI_TELEMETRY',
  'PI_SHARE_VIEWER_URL',
]);

/** Forced isolation key values (evidence: PI_OFFLINE=1, PI_TELEMETRY=0). */
export const PI_OFFLINE_FORCED = '1';
export const PI_TELEMETRY_FORCED = '0';

const FORCED_ENV_KEYS = Object.freeze([
  'HOME',
  'USERPROFILE',
  'PI_CODING_AGENT_DIR',
  'PI_CODING_AGENT_SESSION_DIR',
  'PI_OFFLINE',
  'PI_TELEMETRY',
]);

const FROZEN_FORCED_ENV_KEYS = Object.freeze([...FORCED_ENV_KEYS]);

const foldCase = (name) => String(name).toLowerCase();

/**
 * True when `name` is on the strip list (42 credential vars + 6 pi config
 * switches). Case-insensitive: Windows env var names are not case-sensitive.
 */
export function isStrippedEnvName(name) {
  const folded = foldCase(name);
  return PI_CREDENTIAL_ENV_VARS.some((n) => foldCase(n) === folded) ||
    PI_CONFIG_ENV_VARS.some((n) => foldCase(n) === folded);
}

/** True when `name` is one of the six forced isolation keys. */
export function isForcedEnvName(name) {
  const folded = foldCase(name);
  return FORCED_ENV_KEYS.some((n) => foldCase(n) === folded);
}

/**
 * Remove any case-insensitive duplicate of `name` already present in `env`
 * and return the canonical key name (Windows env var names must not coexist
 * in case-variant forms).
 */
function canonicalKey(env, name) {
  const folded = foldCase(name);
  for (const key of Object.keys(env)) {
    if (foldCase(key) === folded) {
      delete env[key];
      break;
    }
  }
  return name;
}

/**
 * Create one attempt's isolation directory group: a fresh temp root with
 * three subdirectories (agent-config / sessions / home). Returns
 * `{root, agentConfig, sessions, home}`; throws when creation fails (no
 * half-initialized state; callers must not spawn without isolation).
 */
export function createIsolationDirs({parent} = {}) {
  const root = mkdtempSync(path.join(parent ?? tmpdir(), 'tally-pi-attempt-'));
  const agentConfig = path.join(root, 'agent-config');
  const sessions = path.join(root, 'sessions');
  const home = path.join(root, 'home');
  for (const dir of [agentConfig, sessions, home]) {
    mkdirSync(dir, {mode: 0o755});
  }
  return {root, agentConfig, sessions, home};
}

/**
 * Delete an isolation directory group (best effort: a failure does not block
 * attempt teardown; OS temp-dir cleanup is the backstop for orphans).
 * Returns true when a removal was attempted and succeeded.
 */
export function removeIsolationDirs(dirs) {
  if (!dirs || !dirs.root) return false;
  try {
    rmSync(dirs.root, {recursive: true, force: true});
    return true;
  } catch {
    return false;
  }
}

/**
 * Build the spawn environment for one pi process. Semantics (mirrors PR #469
 * `spawnEnv`):
 *   1. inherit `inheritedEnv` minus the stripped names (credentials + pi
 *      config switches, case-insensitive);
 *   2. force the six isolation keys (override everything, case-insensitive;
 *      values come from the per-attempt `dirs`);
 *   3. layer `injectedEnv` (the Host's explicit channel, e.g. the provider
 *      API key) — forced keys reject same-named injections.
 * Keys are returned in sorted order for deterministic env construction.
 */
export function buildPiSpawnEnv({inheritedEnv, injectedEnv, dirs} = {}) {
  if (!dirs || !dirs.root || !dirs.agentConfig || !dirs.sessions || !dirs.home) {
    throw new TypeError('PI_ISOLATION_DIRS_INCOMPLETE');
  }
  const env = {};
  for (const [name, value] of Object.entries(inheritedEnv ?? {})) {
    if (isStrippedEnvName(name) || isForcedEnvName(name)) continue;
    env[canonicalKey(env, name)] = value;
  }
  const forced = {
    HOME: dirs.home,
    USERPROFILE: dirs.home,
    PI_CODING_AGENT_DIR: dirs.agentConfig,
    PI_CODING_AGENT_SESSION_DIR: dirs.sessions,
    PI_OFFLINE: PI_OFFLINE_FORCED,
    PI_TELEMETRY: PI_TELEMETRY_FORCED,
  };
  for (const [name, value] of Object.entries(forced)) {
    env[canonicalKey(env, name)] = value;
  }
  for (const [name, value] of Object.entries(injectedEnv ?? {})) {
    if (isForcedEnvName(name)) continue;
    env[canonicalKey(env, name)] = value;
  }
  const sorted = {};
  for (const key of Object.keys(env).sort()) sorted[key] = env[key];
  return sorted;
}

export {FROZEN_FORCED_ENV_KEYS as PI_FORCED_ENV_KEYS};
