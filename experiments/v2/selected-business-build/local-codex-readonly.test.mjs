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
