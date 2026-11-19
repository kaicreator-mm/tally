#!/usr/bin/env node
/**
 * fake-pi.mjs — Node stand-in pi CLI for the tally-provider-pi tests
 * (port of the PR #469 fakepi behavior; only the 0.85.1 observed surface is
 * imitated, no invented flags).
 *
 * Observed surface:
 *   - `--version` / `-v` → bare version line "0.85.1", exit 0.
 *   - `-p --mode <m> --no-session --session-dir <d> --no-extensions
 *     --no-skills --no-prompt-templates --no-themes --no-context-files
 *     --no-approve [--provider <p>] [--model <m>] [--thinking <t>] --
 *     <instruction>` → one-shot process; json mode stdout = NDJSON event
 *     stream (session line + assistant message_end with fixed self-reported
 *     provider/model/usage); text mode stdout = the final answer plain text;
 *     exit 0 = success / non-zero = incomplete. cwd = process cwd (the
 *     workspace-pinning effect surface); the `read <file>` instruction reads
 *     that file from cwd and echoes its content, proving cwd binding.
 *
 * Test injection points (test-only; the production adapter never sets them):
 *   - TALLY_FAKE_PI_VERSION   overrides the `--version` output (pin-gate
 *                             negative case).
 *   - TALLY_FAKE_PI_SCENARIO  versionfail (--version exits 3) |
 *                             exitfail (partial output, then exit 7) |
 *                             emptyok (exit 0, no output) |
 *                             leakstderr (stderr carries a secret-shaped
 *                             string, exit 1) |
 *                             hang (spawn a child that inherits stdout and
 *                             wait — the tree-termination proof surface).
 *   - TALLY_FAKE_PI_REQUIRE_KEY=1 → without a DEEPSEEK_API_KEY in the
 *                             environment: stderr "No API key found for
 *                             deepseek." + exit 1 (observed keyless shape;
 *                             also proves operator-env keys were stripped).
 *   - TALLY_FAKE_PI_RECORD    append one JSON observation line per invocation
 *                             (argv / cwd / mode / flags / forced + stripped
 *                             env values / credential NAMES seen / child
 *                             pid). Names only, never values.
 */

import {appendFileSync, readFileSync} from 'node:fs';
import {spawn} from 'node:child_process';
import {PI_CREDENTIAL_ENV_VARS} from '../src/pi-env.mjs';

const PINNED_VERSION = '0.85.1';
const FAKE_SESSION_ID = 'f0000000-0000-4000-8000-000000000001';
// Deliberately different from any plausible expected configuration, proving
// the adapter reports stream-observed values instead of backfilling wants.
const FAKE_PROVIDER = 'fake-provider';
const FAKE_MODEL = 'fake-model';

const foldedEnv = new Map(
  Object.entries(process.env).map(([k, v]) => [k.toLowerCase(), v]),
);
const envValue = (name) => foldedEnv.get(name.toLowerCase());
const isEnvSet = (name) => {
  const value = envValue(name);
  return value !== undefined && value !== '';
};

function writeRecord(rec) {
  const recordPath = envValue('TALLY_FAKE_PI_RECORD');
  if (!recordPath) return;
  try {
    appendFileSync(recordPath, JSON.stringify(rec) + '\n');
  } catch {
    // observation recording is best-effort
  }
}

function credentialsPresent() {
  return PI_CREDENTIAL_ENV_VARS.filter((name) => isEnvSet(name));
}

function childMode() {
  // Sleep until tree-killed / timeout: the tree-termination forensic target.
  setTimeout(() => {}, 10 * 60 * 1000);
}

function main() {
  if (isEnvSet('TALLY_FAKE_PI_CHILD')) {
    childMode();
    return;
  }
  const args = process.argv.slice(2);
  if (args.length === 0) process.exit(2);

  if (args[0] === '--version' || args[0] === '-v') {
    if (envValue('TALLY_FAKE_PI_SCENARIO') === 'versionfail') {
      process.stderr.write('fake-pi: --version refused\n');
      process.exit(3);
    }
    const version = envValue('TALLY_FAKE_PI_VERSION') ?? PINNED_VERSION;
    writeRecord({kind: 'version', version});
    process.stdout.write(version + '\n');
    return;
  }

  let mode = '';
  let sessionDir = '';
  let provider = '';
  let model = '';
  let thinking = '';
  let instruction = '';
  let sawSeparator = false;
  const flags = new Map();
  for (let i = 0; i < args.length; i++) {
    const a = args[i];
    if (sawSeparator) {
      instruction = a; // single positional after `--` (adapter contract)
    } else if (a === '--') {
      sawSeparator = true;
    } else if (a === '-p' || a === '--print') {
      // non-interactive print mode: one-shot process
    } else if (a === '--mode' && i + 1 < args.length) {
      mode = args[++i];
    } else if (a === '--session-dir' && i + 1 < args.length) {
      sessionDir = args[++i];
    } else if (a === '--provider' && i + 1 < args.length) {
      provider = args[++i];
    } else if (a === '--model' && i + 1 < args.length) {
      model = args[++i];
    } else if (a === '--thinking' && i + 1 < args.length) {
      thinking = args[++i];
    } else if (a.startsWith('--no-')) {
      flags.set(a, true);
    } else {
      process.stderr.write(`fake-pi: unexpected argument ${JSON.stringify(a)}\n`);
      process.exit(2);
    }
  }
  if (mode === '') mode = 'text'; // observed default output mode

  const cwd = process.cwd();
  const rec = {
    kind: 'print',
    argv: process.argv,
    cwd,
    mode,
    session_dir: sessionDir,
    provider,
    model,
    thinking,
    no_session: flags.get('--no-session') === true,
    no_extensions: flags.get('--no-extensions') === true,
    no_skills: flags.get('--no-skills') === true,
    no_prompt_templates: flags.get('--no-prompt-templates') === true,
    no_themes: flags.get('--no-themes') === true,
    no_context_files: flags.get('--no-context-files') === true,
    no_approve: flags.get('--no-approve') === true,
    home: envValue('HOME') ?? '',
    userprofile: envValue('USERPROFILE') ?? '',
    pi_agent_dir: envValue('PI_CODING_AGENT_DIR') ?? '',
    pi_session_dir_env: envValue('PI_CODING_AGENT_SESSION_DIR') ?? '',
    pi_package_dir: envValue('PI_PACKAGE_DIR') ?? '',
    pi_share_viewer_url: envValue('PI_SHARE_VIEWER_URL') ?? '',
    pi_offline: envValue('PI_OFFLINE') ?? '',
    pi_telemetry: envValue('PI_TELEMETRY') ?? '',
    credentials_present: credentialsPresent(),
    instruction,
  };

  // hang: spawn a child inheriting stdout and wait — killing only the parent
  // would leave the child holding the output pipes (the orphan the adapter's
  // tree termination + WaitDelay bound must cover).
  if (envValue('TALLY_FAKE_PI_SCENARIO') === 'hang') {
    const child = spawn(process.execPath, [process.argv[1]], {
      env: {...process.env, TALLY_FAKE_PI_CHILD: '1'},
      stdio: ['ignore', 'inherit', 'inherit'],
    });
    rec.child_pid = child.pid;
    writeRecord(rec);
    child.once('exit', (code) => process.exit(code ?? 0));
    child.once('error', () => process.exit(3));
    return;
  }

  writeRecord(rec);

  // Observed keyless contract: missing provider key → structured stderr,
  // exit 1, empty stdout. Gated for tests only; also proves the operator
  // environment's stripped key did not flow in when injection is absent.
  if (isEnvSet('TALLY_FAKE_PI_REQUIRE_KEY') && !isEnvSet('DEEPSEEK_API_KEY')) {
    process.stderr.write('No API key found for deepseek.\n');
    process.stderr.write('\n');
    process.stderr.write('Use /login to log into a provider via OAuth or API key.\n');
    process.exit(1);
  }

  switch (envValue('TALLY_FAKE_PI_SCENARIO')) {
    case 'exitfail': {
      // Crash after partial output: the partial text must never become the
      // final answer (json mode emits one valid session line + one truncated
      // tail line without a trailing newline).
      if (mode === 'json') {
        process.stdout.write(
          `{"type":"session","version":3,"id":"${FAKE_SESSION_ID}","cwd":${JSON.stringify(cwd)}}\n`,
        );
        process.stdout.write('{"type":"message_update","assistantMessageEvent":{"delta":"partial');
      } else {
        process.stdout.write('partial answer before crash\n');
      }
      process.exit(7);
      return;
    }
    case 'emptyok':
      // exit 0 but no output: contract not honored → the adapter must fail
      // safe instead of guessing success.
      process.exit(0);
      return;
    case 'stdinwait': {
      // Regression for the real-CLI hang: pi 0.85.1 `-p` reads stdin to EOF
      // before finishing its turn. The fake models that by answering only
      // after stdin ends. If the adapter ever spawns with a default `pipe`
      // stdin it never closes, this scenario never answers and the bounded
      // timeout turns the observation into UNKNOWN — the test asserts
      // COMPLETED, so the regression cannot pass silently.
      process.stdin.resume();
      process.stdin.once('end', () => {
        const answer = 'stdin closed, turn complete';
        if (mode === 'json') emitJSONStream(cwd, answer);
        else process.stdout.write(answer + '\n');
        process.exit(0);
      });
      process.stdin.once('error', () => process.exit(3));
      return;
    }
    case 'leakstderr':
      // Failure output with a secret-shaped string: the non-completed
      // summary must be redacted before entering an observation.
      process.stderr.write('provider error DEEPSEEK_API_KEY=sk-fake-secret-1234567890abcdef\n');
      process.exit(1);
      return;
    default:
      break;
  }

  // cwd binding task: `read <file>` → the file content under the process cwd
  // (proves workspace pinning takes effect via the process cwd).
  if (instruction.startsWith('read ')) {
    const file = instruction.slice('read '.length).trim();
    try {
      const data = readFileSync(file, 'utf8');
      if (mode === 'json') emitJSONStream(cwd, data.replace(/\n+$/, ''));
      else process.stdout.write(data);
    } catch (err) {
      process.stderr.write(`fake-pi: read failed: ${err?.message ?? err}\n`);
      process.exit(1);
    }
    return;
  }

  if (mode === 'json') {
    emitJSONStream(cwd, 'ok');
    return;
  }
  process.stdout.write('ok\n');
}

function emitJSONStream(cwd, text) {
  const sessionLine = {
    type: 'session', version: 3, id: FAKE_SESSION_ID,
    timestamp: '2026-11-19T00:00:00.000Z', cwd,
  };
  const messageEnd = {
    type: 'message_end',
    message: {
      role: 'assistant',
      content: [{type: 'text', text}],
      api: 'openai-completions',
      provider: FAKE_PROVIDER,
      model: FAKE_MODEL,
      usage: {input: 11, output: 3, cacheRead: 0, cacheWrite: 0, totalTokens: 14},
      stopReason: 'stop',
    },
  };
  for (const line of [
    JSON.stringify(sessionLine),
    JSON.stringify({type: 'agent_start'}),
    JSON.stringify(messageEnd),
    JSON.stringify({type: 'turn_end'}),
    JSON.stringify({type: 'agent_end'}),
    JSON.stringify({type: 'agent_settled'}),
  ]) {
    process.stdout.write(line + '\n');
  }
}

main();
