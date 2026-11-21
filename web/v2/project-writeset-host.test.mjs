import test from 'node:test';
import assert from 'node:assert/strict';
import {createProjectWriteSetHost,selectedWriteSetIdentity}
  from './project-writeset-host.mjs';
const sha='a'.repeat(40);
const settings={repository:'org/new',versionId:'main',expectedSha:sha};
const env={TALLY_V2_WRITESET_FREEZE:'host-mounted',
  T06M_TARGET_REPOSITORY:'org/new',T06M_TARGET_VERSION_REF:'main',
  T06M_TARGET_ISSUE_NUMBER:'27',T06M_TARGET_CANDIDATE_PATH:'src/health.mjs',
  TALLY_V2_CLAIM_REPOSITORY:'org/new',TALLY_V2_CLAIM_VERSION_REF:'main',
  TALLY_V2_CLAIM_SOURCE_SHA:sha,TALLY_V2_CLAIM_ISSUE_NUMBER:'27',
  TALLY_V2_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
  TALLY_V2_AGENT_DISPATCH:'host-mounted',
  TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite'};
function db(){
  const rows=new Map();let puts=0;
  return {rows,async get(key){return rows.get(key)??null;},
    async put(key,value,{mustBeAbsent}={}){
      puts++;if(mustBeAbsent&&rows.has(key))throw Error('CAS_CONFLICT');
      const row={revision:0,value};rows.set(key,row);return row;},
    async close(){},get puts(){return puts;}};
}
test('unmounted FSW does not expose Host',()=>{
  assert.equal(createProjectWriteSetHost({settings,env:{}}),null);
});
test('exact selected Host pins and one path, no request-controlled path',async()=>{
  const store=db();
  const host=createProjectWriteSetHost({settings,env,openStore:async()=>store});
  assert.equal(host.issueNumber,27);
  assert.equal(host.candidateRelativePath,'src/health.mjs');
  const blocked=await host.freeze({issueNumber:28,sourceSha:sha});
  assert.equal(blocked.status,'BLOCKED');assert.equal(store.puts,0);
  const first=await host.freeze({issueNumber:27,sourceSha:sha});
  assert.equal(first.status,'FROZEN_VERIFIED');assert.equal(store.puts,1);
  assert.equal(first.authorized,false);
  const replay=await host.freeze({issueNumber:27,sourceSha:sha});
  assert.equal(replay.status,'ALREADY_FROZEN_VERIFIED');assert.equal(store.puts,1);
  const value=[...store.rows.values()][0].value;
  assert.equal(value.document,JSON.stringify(['src/health.mjs']));
  assert.equal(value.writeSetSha256,first.writeSetSha256);
});
test('foreign project/ref/Issue/registry and unsafe path cannot mount',()=>{
  for(const bad of [
    {T06M_TARGET_REPOSITORY:'org/other'},
    {T06M_TARGET_VERSION_REF:'version/v2'},
    {T06M_TARGET_ISSUE_NUMBER:'28'},
    {T06M_TARGET_CANDIDATE_PATH:'../secret'},
    {TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/foreign.sqlite'},
    {TALLY_V2_CLAIM_SOURCE_SHA:'b'.repeat(40)}
  ])assert.throws(()=>selectedWriteSetIdentity({settings,env:{...env,...bad}}),
    TypeError);
});
test('drifted or occupied CAS record cannot be overwritten',async()=>{
  const store=db(),host=createProjectWriteSetHost({settings,env,
    openStore:async()=>store});
  await host.freeze({issueNumber:27,sourceSha:sha});
  const [key,row]=[...store.rows][0];
  store.rows.set(key,{...row,value:{...row.value,sourceSha:'b'.repeat(40)}});
  const refused=await host.freeze({issueNumber:27,sourceSha:sha});
  assert.equal(refused.reason,'SELECTED_WRITESET_EXISTING_ROW_MISMATCH');
  assert.equal(store.puts,1);
});
test('CAS write-ack or readback failure never reports FROZEN_VERIFIED',async()=>{
  const store=db(),host=createProjectWriteSetHost({settings,env,
    openStore:async()=>({...store,put:async()=>{throw Error('ACK_LOST')}})});
  const failed=await host.freeze({issueNumber:27,sourceSha:sha});
  assert.equal(failed.status,'BLOCKED');
  assert.equal(failed.reason,'SELECTED_WRITESET_CAS_UNKNOWN');
});
