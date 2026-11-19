import test from 'node:test';
import assert from 'node:assert/strict';
import {existsSync} from 'node:fs';
import {mkdtemp,readFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createLocalCodexHost,LOCAL_CODEX_CONSTANTS} from './local-codex-host.mjs';

const sourceSha='a'.repeat(40);
const issueRef='https://github.com/kaicreator-mm/tally/issues/264';
const attemptId='attempt:264:readonly:1';
const attemptKey=LOCAL_CODEX_CONSTANTS.attemptKeyPrefix+attemptId;

test('readAttemptRecord cannot create storage/schema and enforces full identity',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-codex-readonly-'));
  try{
    const host=createLocalCodexHost({spawnFn:()=>{
      throw new Error('read-only reconcile must never spawn');
    }});
    const missingPath=join(dir,'missing.sqlite');
    const missing=await host.readAttemptRecord({issueRef,sourceSha,attemptId,
      hostStorePath:missingPath});
    assert.equal(missing.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(missing.reason,'ATTEMPT_RECORD_STORE_UNAVAILABLE');
    for(const suffix of ['', '-wal', '-shm'])
      assert.equal(existsSync(missingPath+suffix),false,
        'missing storage and sidecars must not be created');

    // An existing zero-byte/schema-less DB must not acquire v8_host_docs or
    // change bytes merely because reconcile was asked to read it.
    const emptyPath=join(dir,'empty.sqlite');
    const Database=(await import('better-sqlite3')).default;
    new Database(emptyPath).close();
    const before=await readFile(emptyPath);
    const empty=await host.readAttemptRecord({issueRef,sourceSha,attemptId,
      hostStorePath:emptyPath});
    assert.equal(empty.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(empty.reason,'ATTEMPT_RECORD_READ_FAILED');
    assert.deepEqual(await readFile(emptyPath),before);
    const inspect=new Database(emptyPath,{readonly:true,fileMustExist:true});
    assert.equal(inspect.prepare("SELECT count(*) AS n FROM sqlite_master WHERE type='table' AND name='v8_host_docs'").get().n,0);
    inspect.close();

    const hostStorePath=join(dir,'host.sqlite');
    const store=await openSqliteCasStore({path:hostStorePath,busyTimeoutMs:1000});
    await store.put(attemptKey,{phase:'CANDIDATE_READY',attemptId,sourceSha,issueRef,
      providerId:'codex',candidateTree:'b'.repeat(40),candidateCommit:'c'.repeat(40),
      process:null},{mustBeAbsent:true});
    await store.close();
    const config={issueRef,sourceSha,attemptId,hostStorePath};
    const wrongTask=await host.readAttemptRecord({...config,
      issueRef:'https://github.com/kaicreator-mm/tally/issues/999'});
    assert.equal(wrongTask.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(wrongTask.reason,'ATTEMPT_IDENTITY_MISMATCH');
    const wrongProvider=await host.readAttemptRecord({...config,providerId:'zcode'});
    assert.equal(wrongProvider.state,'LOCAL_ATTEMPT_UNKNOWN');
    assert.equal(wrongProvider.reason,'ATTEMPT_IDENTITY_MISMATCH');
    const accepted=await host.readAttemptRecord(config);
    assert.equal(accepted.state,'LOCAL_CANDIDATE_READY');
    assert.equal(accepted.readOnly,true);
    assert.equal(accepted.candidateCommit,'c'.repeat(40));
  }finally{await rm(dir,{recursive:true,force:true});}
});

// ---------------------------------------------------------------------------
// R107 real-acceptance regression: the sealed shim's denied() helper calls
// common(input, null, ...) — the merged providerId spread must be null-safe
// or EVERY real denial crashes the sealed C (replay lanes only exercise
// REPLAY_COMPLETE and never see this). Found by the real two-provider
// parallel acceptance (null.providerId TypeError inside the sealed
// capability).
// ---------------------------------------------------------------------------
test('shim denial helper never dereferences a null observation (R107 P1)',async()=>{
  const {createRequire}=await import('node:module');
  const {fileURLToPath}=await import('node:url');
  const require=createRequire(import.meta.url);
  // Load the sealed shim exactly as producer seals it (marker replaced).
  const {readFileSync}=await import('node:fs');
  const source=readFileSync(new URL('./local-codex-shim.mjs',import.meta.url),'utf8')
    .replace('__PROVIDER_SHA256__','0'.repeat(64));
  // The shim's bare bundle import cannot resolve from a data: URL in the
  // bare test env — substitute a stub exporting the same names as throwing
  // functions (denial paths under test never call into the native bundle).
  const importRe=/import\s*\{([^}]+)\}\s*from\s*'@kaicreator\/tally-local-codex-host-bundle';/;
  const names=importRe.exec(source)[1].split(',')
    .map(x=>x.trim()).map(x=>x.split(' as ').pop().trim());
  const stub='data:text/javascript;base64,'+Buffer.from(
    names.map(n=>`export const ${n}=async()=>{throw Error('STUB_NOT_CALLED');}`).join(';'),
    'utf8').toString('base64');
  const moduleUrl='data:text/javascript;base64,'+
    Buffer.from(source.replace(importRe,
      `import{${names.join(',')}}from '${stub}';`),'utf8').toString('base64');
  const shim=await import(moduleUrl);
  // Drive one denial end-to-end: an input with a malformed sourceSha must
  // produce a NOT_ADMITTED observation, never a TypeError.
  const observed=await shim.runLocalCodexAttempt({sourceRef:
    'https://github.com/kaicreator-mm/tally/issues/172',issueNumber:172,
    sourceSha:'not-a-sha',attemptId:'attempt:172:r107:reg:1'});
  assert.equal(observed.outcome.output.state,'NOT_ADMITTED');
  assert.equal(observed.outcome.output.reason,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  // The denied observation carries no providerId and never crashes.
  assert.equal(observed.outcome.output.providerId,undefined);
  assert.equal(observed.outcome.output.taskPass,false);
});
