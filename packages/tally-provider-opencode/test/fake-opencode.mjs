#!/usr/bin/env node
/**
 * fake-opencode.mjs — test-double opencode CLI (Node stdlib only), launched
 * via `process.execPath fake-opencode.mjs ...` by the adapter tests. It
 * mimics ONLY the surface observed on opencode-ai@1.19.6 (PR #471 evidence;
 * behavior mirrors runtimes/opencode/testdata/fakeopencode/main.go):
 *
 *   - `--version` → bare version line "1.19.6", exit 0;
 *   - `run --format json [--model <p/m>] [--agent <a>] --pure <message>` →
 *     one-shot process: stdout = NDJSON event stream (step_start / text /
 *     step_finish, observed envelope), exit 0 = completed / non-zero =
 *     incomplete; config/data dirs follow the XDG_* env vars; a missing
 *     credential is NOT a failure (the CLI falls back to its keyless default
 *     model — no such failure branch is implemented here);
 *   - long-run marker "tally-conformance-long" → spawns a child node process
 *     and waits (tree-kill evidence: child inherits stdout/stderr as a
 *     pipe-holding orphan target; its PID goes into the record file);
 *
 * Test injection env (production never sets these):
 *   - TALLY_FAKE_OPENCODE_RECORD   JSONL observation record per run call
 *                                  (argv/cwd/PWD/OLDPWD/XDG4/TMP/TEMP/
 *                                  api_key_set/model/agent/message/child_pid);
 *   - TALLY_FAKE_OPENCODE_SCENARIO exitfail | emptyok | plainok | hang |
 *                                  leakstderr | versionfail | badmodel;
 *   - TALLY_FAKE_OPENCODE_VERSION  overrides the `--version` output
 *                                  (pin-gate negative case).
 */
import {spawn} from 'node:child_process';
import {appendFileSync, writeSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

const PINNED_VERSION = '1.19.6';
const LONG_MARKER = 'tally-conformance-long';
const SELF_PATH = fileURLToPath(import.meta.url);

const out = text => writeSync(1, text);
const errOut = text => writeSync(2, text);
const hangForever = () => {
  setInterval(() => {}, 60_000);
  setTimeout(() => process.exit(0), 5 * 60_000); // safety net for tests
};

function emitEvent(envType, partType, text, tokens) {
  const part = {type: partType, id: 'prt_fake', sessionID: 'ses_fake', messageID: 'msg_fake'};
  if (text) part.text = text;
  if (tokens) {
    part.reason = 'stop';
    part.tokens = {
      total: tokens[0] + tokens[1], input: tokens[0], output: tokens[1],
      reasoning: 0, cache: {write: 0, read: 0},
    };
    part.cost = 0.0001;
  }
  out(JSON.stringify({type: envType, timestamp: Date.now(), sessionID: 'ses_fake', part}) + '\n');
}

function writeRecord(extra = {}) {
  const path = process.env.TALLY_FAKE_OPENCODE_RECORD;
  if (!path) return;
  const line = JSON.stringify({
    argv: process.argv,
    cwd: process.cwd(),
    pwd_env: process.env.PWD ?? null,
    oldpwd_env: process.env.OLDPWD ?? null,
    xdg_config: process.env.XDG_CONFIG_HOME ?? null,
    xdg_data: process.env.XDG_DATA_HOME ?? null,
    xdg_cache: process.env.XDG_CACHE_HOME ?? null,
    xdg_state: process.env.XDG_STATE_HOME ?? null,
    tmp: process.env.TMP ?? null,
    temp: process.env.TEMP ?? null,
    api_key_set: (process.env.DEEPSEEK_API_KEY ?? '') !== '',
    ...extra,
  });
  appendFileSync(path, line + '\n');
}

function main() {
  const args = process.argv.slice(2);

  // Child mode: sleep until the whole tree is terminated.
  if (args[0] === '--tally-fake-child') {
    setInterval(() => {}, 60_000);
    setTimeout(() => process.exit(0), 10 * 60_000);
    return;
  }

  if (args[0] === '--version') {
    const override = process.env.TALLY_FAKE_OPENCODE_VERSION;
    if (override) {
      out(override + '\n');
    } else if (process.env.TALLY_FAKE_OPENCODE_SCENARIO === 'versionfail') {
      errOut('fake-opencode: --version refused\n');
      process.exit(3);
    } else {
      out(PINNED_VERSION + '\n');
    }
    return;
  }

  // Only the `run` subcommand is an observed surface; anything else refuses.
  if (args[0] !== 'run') {
    errOut('fake-opencode: unexpected argument\n');
    process.exit(2);
  }

  const rest = args.slice(1);
  let model = null;
  let agent = null;
  const message = [];
  for (let i = 0; i < rest.length; i += 1) {
    switch (rest[i]) {
      case '--format':
        i += 1;
        if (rest[i] !== 'json') {
          errOut(`fake-opencode: unsupported format ${JSON.stringify(rest[i])}\n`);
          process.exit(2);
        }
        break;
      case '--pure':
        break; // boolean flag, no value (observed shape)
      case '--model':
        i += 1;
        model = rest[i];
        break;
      case '--agent':
        i += 1;
        agent = rest[i];
        break;
      default:
        message.push(rest[i]);
    }
  }
  const instruction = message.join(' ');
  const baseRecord = {model, agent, message: instruction};

  // Long run: spawn a child (inherits stdout/stderr — the pipe-holding orphan
  // shape), record its PID, then hang until the tree is terminated.
  if (instruction.includes(LONG_MARKER)) {
    const child = spawn(process.execPath, [SELF_PATH, '--tally-fake-child'],
      {stdio: 'inherit', windowsHide: true});
    writeRecord({...baseRecord, child_pid: child.pid});
    hangForever();
    return;
  }

  writeRecord(baseRecord);

  switch (process.env.TALLY_FAKE_OPENCODE_SCENARIO) {
    case 'exitfail': {
      // Crash after partial events: the partial text must never surface as
      // the final answer of a FAILED claim.
      emitEvent('step_start', 'step-start', '', null);
      emitEvent('text', 'text', 'partial answer before crash', null);
      errOut('{"name":"UnknownError","data":{"message":"Unexpected server error. Check server logs for details.","ref":"err_fake"}}\n');
      process.exit(7);
    }
    case 'emptyok':
      // exit 0 with no output at all: contract unfulfilled → adapter must
      // fail safe, not guess success.
      process.exit(0);
    case 'plainok':
      // exit 0 with bare non-JSON text: degrade to the raw text.
      out('plain-ok\n');
      return;
    case 'leakstderr': {
      // Emit the injected credential into stderr: the adapter must redact it
      // before it can enter any observation.
      errOut(`provider error api_key=${process.env.DEEPSEEK_API_KEY ?? 'none'}\n`);
      process.exit(1);
    }
    case 'hang':
      // Hang until the adapter kills the tree (timeout/abort/cancel).
      hangForever();
      return;
    case 'badmodel': {
      // Observed bad-model shape: stderr structured error JSON, stdout empty.
      errOut('{"name":"UnknownError","data":{"message":"model not found","ref":"err_fake"}}\n');
      process.exit(1);
    }
    default:
      break;
  }

  // Happy path: observed NDJSON event shape (step_start → text → step_finish,
  // tokens carrying input/output).
  emitEvent('step_start', 'step-start', '', null);
  emitEvent('text', 'text', 'ok', null);
  emitEvent('step_finish', 'step-finish', '', [38, 2]);
}

main();
