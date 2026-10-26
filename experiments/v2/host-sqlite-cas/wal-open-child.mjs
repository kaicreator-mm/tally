// Issue #100: optional cross-process WAL *initialization* probe for the public
// DomainHarness Host SQLite adapter. This is not the legacy DH N08b oracle.
import assert from 'node:assert/strict';
import { openSqliteCasStore } from '@kaicreator/domain-harness-host-node';

const [file, slot] = process.argv.slice(2);
assert.ok(file && /^\d+$/.test(slot ?? ''), 'file path and numeric worker slot required');
let store;
try {
  store = await openSqliteCasStore({ path: file, busyTimeoutMs: 10000 });
  const key = `wcx:wal:child:${slot}`;
  const created = await store.put(key, { slot }, { mustBeAbsent: true });
  assert.equal(created.revision, 0);
  assert.deepEqual(await store.get(key), { value: { slot }, revision: 0 });
  process.stdout.write(`OPEN_OK slot=${slot}\n`);
} catch (error) {
  // Retain typed error identity for exact process evidence. No credentials/data.
  const name = error instanceof Error ? error.name : 'UnknownError';
  const code = typeof error?.code === 'string' ? error.code : 'NO_TYPED_CODE';
  const message = error instanceof Error ? error.message : String(error);
  process.stderr.write(`OPEN_FAIL slot=${slot} name=${name} code=${code} message=${message}\n`);
  process.exitCode = 1;
} finally {
  if (store) await store.close();
}
