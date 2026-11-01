// Issue #100: separate, opt-in source-qualified Windows/Ubuntu WAL-startup
// stress, physically running four *OS processes* on each fresh SQLite file.
// This does not reproduce/qualify the different legacy NodeSqliteRuntimeStore
// N08b #1385 path, but screens the actual public Host Store seam Tally uses.
import test from 'node:test';
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const execFileAsync = promisify(execFile);
const worker = fileURLToPath(new URL('./wal-open-child.mjs', import.meta.url));
const rounds = Number(process.env.WCX_WAL_ROUNDS ?? '5');
assert.ok(Number.isSafeInteger(rounds) && rounds >= 1 && rounds <= 30,
  'WCX_WAL_ROUNDS must be integer 1..30');

test('real public Host SQLite: simultaneous fresh-file WAL open across processes', async (t) => {
  assert.equal(Number(process.versions.node.split('.')[0]), 24, 'Node24 required');
  const dir = await mkdtemp(join(tmpdir(), 'tally-v2-wcx-wal-'));
  t.after(async () => { await rm(dir, { recursive: true, force: true }); });
  const failures = [];
  for (let round = 1; round <= rounds; round += 1) {
    // A fresh path forces each round to exercise WAL initial negotiation.
    const db = join(dir, `fresh-${round}.sqlite`);
    const runs = await Promise.allSettled(Array.from({ length: 4 }, (_, slot) =>
      execFileAsync(process.execPath, [worker, db, String(slot)], {
        timeout: 30000, maxBuffer: 64 * 1024,
      })
    ));
    for (let slot = 0; slot < runs.length; slot += 1) {
      const result = runs[slot];
      if (result.status === 'rejected') {
        const err = result.reason;
        failures.push({ round, slot,
          exit: typeof err?.code === 'number' ? err.code : null,
          stderr: String(err?.stderr ?? err?.message ?? 'unknown').slice(0, 1200),
        });
      } else {
        assert.match(result.value.stdout, /OPEN_OK slot=/);
      }
    }
  }
  console.log('W-Cx PUBLIC_HOST_WAL_CONCURRENCY', JSON.stringify({
    node: process.version, platform: process.platform, rounds, workersPerRound: 4,
    failedWorkers: failures.length, failures,
  }));
  assert.equal(failures.length, 0, `${failures.length} fresh-WAL child process(es) failed`);
});
