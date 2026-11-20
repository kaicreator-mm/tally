/**
 * fake-grok.mjs - stand-in grok CLI for the tally-provider-grok tests,
 * ported 1:1 from #468 testdata/fakegrok/main.go.
 *
 * It mimics ONLY the surface observed on the official @xai-official/grok
 * 1.0.87 native binary (commit 929e6dd187b1); it invents no flags:
 *
 *   - `--version` -> first line "grok 1.0.87 (0000000fakegrok)" (observed
 *     shape "grok 1.0.87 (929e6dd187b1)", exit 0);
 *   - `--no-auto-update [-m <model>] [--permission-mode <mode>]
 *     --single=<prompt>` -> one-shot process: stdout = final answer (plain);
 *     exit 0 = completed / non-zero = not completed; cwd = process cwd
 *     (workspaceRoot = process.cwd(), inspect observation);
 *     missing XAI_API_KEY -> stderr "Not signed in. ...", exit 1 (observed
 *     keyless contract text);
 *   - long runs (prompt contains the tally-conformance-long marker) spawn a
 *     child process and wait on it - the whole-tree teardown evidence
 *     surface: the child pid is written to the record file; the child
 *     inherits stdout/stderr and sleeps until the tree is killed.
 *
 * Test injection (production adapter never sets these):
 *   - TALLY_FAKE_GROK_SCENARIO: versionfail | exitfail | emptyok | usage |
 *     keyless | leak | hang;
 *   - TALLY_FAKE_GROK_VERSION: override the --version output (pin-gate
 *     negative case);
 *   - TALLY_FAKE_GROK_RECORD: JSONL record of each observed call surface
 *     (argv / cwd / GROK_HOME / credential-presence booleans / model /
 *     permission-mode / child pid / prompt). Records carry env PRESENCE
 *     booleans only - never credential values.
 */
import {spawn} from 'node:child_process';
import {appendFileSync} from 'node:fs';
import process from 'node:process';

const PINNED_VERSION = '1.0.87';
const LONG_MARKER = 'tally-conformance-long';

const KEYLESS_STDERR = 'Not signed in. To authenticate without a browser, run:\n' +
  '  grok login --device-code\n\n' +
  'Alternatively, set the XAI_API_KEY environment variable or run `grok login` on a machine with a browser.';

function writeRecord(record) {
  const path = process.env.TALLY_FAKE_GROK_RECORD;
  if (!path) return;
  try {
    appendFileSync(path, `${JSON.stringify(record)}\n`);
  } catch { /* record sink is best effort */ }
}

function usageError(message) {
  process.stderr.write(`fakegrok: ${message}\n`);
  process.exit(2);
}

function childMode() {
  // Sleep until the adapter's whole-tree teardown lands (or the harness
  // gives up); holds the inherited stdout pipe while alive.
  setTimeout(() => {}, 10 * 60 * 1000);
}

async function main() {
  if (process.env.TALLY_FAKE_GROK_CHILD === '1') return childMode();

  const args = process.argv.slice(2);
  if (args.length === 0) process.exit(2);

  if (args[0] === '--version') {
    const override = process.env.TALLY_FAKE_GROK_VERSION;
    if (override) {
      process.stdout.write(`${override}\n`);
      return;
    }
    if (process.env.TALLY_FAKE_GROK_SCENARIO === 'versionfail') {
      process.stderr.write('fakegrok: --version refused\n');
      process.exit(3);
    }
    process.stdout.write(`grok ${PINNED_VERSION} (0000000fakegrok)\n`);
    return;
  }

  // Observed invocation shape: [bin] --no-auto-update [-m <m>]
  // [--permission-mode <p>] --single=<prompt>. Anything else is a usage
  // error (observed: unknown flag -> exit 2).
  if (args[0] !== '--no-auto-update') {
    usageError(`unexpected argument ${JSON.stringify(args[0])}`);
  }
  let model = '';
  let permission = '';
  let prompt = '';
  for (let i = 1; i < args.length; i += 1) {
    if (args[i] === '-m') {
      if (i + 1 >= args.length) process.exit(2);
      i += 1;
      model = args[i];
    } else if (args[i] === '--permission-mode') {
      if (i + 1 >= args.length) process.exit(2);
      i += 1;
      permission = args[i];
    } else if (args[i].startsWith('--single=')) {
      prompt = args[i].slice('--single='.length);
    } else {
      usageError(`unexpected argument ${JSON.stringify(args[i])}`);
    }
  }
  if (prompt === '') usageError('missing --single=<prompt>');

  const record = {
    argv: args,
    cwd: process.cwd(),
    grokHome: process.env.GROK_HOME ?? '',
    xaiApiKeySet: (process.env.XAI_API_KEY ?? '') !== '',
    deepseekKeySet: (process.env.DEEPSEEK_API_KEY ?? '') !== '',
    prompt,
  };
  if (model) record.model = model;
  if (permission) record.permissionMode = permission;

  // Long run: spawn a child that inherits our pipes and wait on it. When the
  // adapter tears the tree down both die; a single-process kill would leave
  // the child holding the pipes (the orphan surface the adapter hardens
  // against with its bounded pipe-grace settle).
  if (prompt.includes(LONG_MARKER)) {
    const child = spawn(process.execPath, [process.argv[1]], {
      env: {...process.env, TALLY_FAKE_GROK_CHILD: '1'},
      stdio: 'inherit',
    });
    record.childPid = child.pid;
    writeRecord(record);
    child.on('error', () => process.exit(3));
    // Stay alive while the child lives (an active child handle holds the
    // event loop); mirror its exit code once it is gone.
    child.on('close', code => process.exit(code ?? 0));
    return;
  }

  writeRecord(record);

  // Observed keyless contract (missing or scenario-forced credential):
  // stderr "Not signed in. ...", exit 1, empty stdout.
  const scenario = process.env.TALLY_FAKE_GROK_SCENARIO;
  if ((process.env.XAI_API_KEY ?? '') === '' || scenario === 'keyless') {
    process.stderr.write(`${KEYLESS_STDERR}\n`);
    process.exit(1);
  }

  if (scenario === 'exitfail') {
    // Crash after a partial answer: the partial text must never surface as
    // the final answer.
    process.stdout.write('partial answer before crash\n');
    process.exit(7);
  }
  if (scenario === 'emptyok') {
    // exit 0 with no final answer: contract unfulfilled; the adapter must
    // fail safe.
    process.exit(0);
  }
  if (scenario === 'usage') {
    process.stderr.write("fakegrok: error: unexpected argument '-bogus' found\n");
    process.exit(2);
  }
  if (scenario === 'leak') {
    // A CLI that echoes injected credentials into its diagnostics - the
    // adapter must redact before evidence crosses any durable boundary.
    process.stderr.write(`provider failed api_key=${process.env.XAI_API_KEY}\n`);
    process.exit(5);
  }

  // cwd-bound answer (inspect observation shape): `read <file>` prints the
  // file's content from the process cwd, proving workspace pinning via
  // spawn cwd.
  if (prompt.startsWith('read ')) {
    const {readFileSync} = await import('node:fs');
    try {
      process.stdout.write(readFileSync(prompt.slice('read '.length).trim(), 'utf8'));
    } catch (error) {
      process.stderr.write(`fakegrok: read failed: ${error.message}\n`);
      process.exit(1);
    }
    return;
  }

  process.stdout.write('ok\n');
}

main().catch(error => {
  process.stderr.write(`fakegrok: ${error?.message ?? error}\n`);
  process.exit(3);
});
