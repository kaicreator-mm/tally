// Tally v2 W-Cx, Issue #100: public DomainHarness Host SQLite CAS smoke.
// Experimental clean consumer ONLY; neither a Tally Kernel nor a Release gate.
import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { openSqliteCasStore, probeTrueCasStore } from '@kaicreator/domain-harness-host-node';

const isCode = (name) => (error) => {
  assert.equal(error?.code, name, `expected ${name}, got ${error?.code ?? String(error)}`);
  return true;
};

test('real public Host SQLite: CAS, conflict, concurrent connections, restart', async (t) => {
  assert.equal(Number(process.versions.node.split('.')[0]), 24, 'qualified experimental lane requires Node24');
  const dir = await mkdtemp(join(tmpdir(), 'tally-v2-wcx-sqlite-'));
  const path = join(dir, 'host.sqlite');
  const key = 'tally:wcx:public-smoke';
  const stores = [];
  t.after(async () => {
    for (const store of stores.reverse()) {
      try { await store.close(); } catch { /* already closed */ }
    }
    await rm(dir, { recursive: true, force: true });
  });

  const first = await openSqliteCasStore({ path, busyTimeoutMs: 1000 });
  stores.push(first);
  assert.equal(await first.get(key), null);
  const created = await first.put(key, { phase: 'created' }, { mustBeAbsent: true });
  assert.equal(created.revision, 0);
  assert.deepEqual(await first.get(key), { revision: 0, value: { phase: 'created' } });

  await assert.rejects(first.put(key, { phase: 'unsafe' }, {}), isCode('HOST_PUT_CONDITION_REQUIRED'));
  await assert.rejects(first.put(key, { phase: 'duplicate' }, { mustBeAbsent: true }), isCode('HOST_KEY_EXISTS'));
  const updated = await first.put(key, { phase: 'updated' }, { mustMatchRevision: 0 });
  assert.equal(updated.revision, 1);
  await assert.rejects(first.put(key, { phase: 'stale' }, { mustMatchRevision: 0 }), isCode('HOST_CAS_CONFLICT'));

  const second = await openSqliteCasStore({ path, busyTimeoutMs: 1000 });
  stores.push(second);
  assert.deepEqual(await second.get(key), { revision: 1, value: { phase: 'updated' } });
  const raced = await Promise.allSettled([
    first.put(key, { phase: 'winner-A' }, { mustMatchRevision: 1 }),
    second.put(key, { phase: 'winner-B' }, { mustMatchRevision: 1 }),
  ]);
  const winners = raced.filter((x) => x.status === 'fulfilled');
  const losers = raced.filter((x) => x.status === 'rejected');
  assert.equal(winners.length, 1, 'one physical CAS winner is required');
  assert.equal(losers.length, 1, 'one stale revision must conflict');
  assert.equal(winners[0].value.revision, 2);
  assert.equal(losers[0].reason?.code, 'HOST_CAS_CONFLICT');

  const winner = await first.get(key);
  assert.equal(winner.revision, 2);
  assert.ok(['winner-A', 'winner-B'].includes(winner.value.phase));
  assert.deepEqual(await second.get(key), winner);
  await first.close();
  await second.close();

  const reopened = await openSqliteCasStore({ path, busyTimeoutMs: 1000 });
  stores.push(reopened);
  assert.deepEqual(await reopened.get(key), winner, 'restart must preserve revision and value');
  await probeTrueCasStore(reopened);
  assert.deepEqual(await reopened.get(key), winner, 'probe cannot mutate the user key');
  console.log('W-Cx PUBLIC_HOST_CAS_SMOKE=PASS', JSON.stringify({
    node: process.version, platform: process.platform, architecture: process.arch,
    revision: winner.revision, phase: winner.value.phase,
  }));
});
