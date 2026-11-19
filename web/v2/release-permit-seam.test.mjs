/** #491 J07 — release qualification read seam tests (hermetic: injected
 * octokit/store fakes; no network, no host-node import, no GitHub). */
import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createReleaseQualificationRead} from './serve.mjs';

const SOURCE='3'.repeat(40);
const SOURCE_TREE='4'.repeat(40);
const TARGET='5'.repeat(40);
const TARGET_TREE='6'.repeat(40);
const key=(prefix,parts)=>prefix+createHash('sha256')
  .update(JSON.stringify(parts)).digest('hex');

function fakeOctokit({branchSha=TARGET,branchThrows=false,
  commitTrees={[TARGET]:TARGET_TREE,[SOURCE]:SOURCE_TREE}}={}){
  return {rest:{repos:{
    getBranch:async()=>{
      if(branchThrows)throw new Error('net');
      return {data:{commit:{sha:branchSha}}};},
    getCommit:async({ref})=>{
      if(!(ref in commitTrees))throw new Error('no commit');
      return {data:{commit:{tree:{sha:commitTrees[ref]}}}};}
  }}};
}
function fakeStore(rows){
  const openStore=async()=>({get:async k=>rows.get(k)??null,
    close:async()=>{}});
  return {openStore};
}
const pins=(paths={})=>({owner:'kaicreator-mm',repo:'tally',
  rqRegistryPath:paths.rq??'C:/reg/rq.sqlite',
  grantRegistryPath:paths.grant??'C:/reg/grant.sqlite',
  permitRegistryPath:paths.permit??'C:/reg/permit.sqlite'});

test('target readback: observed / head-unobservable / tree-missing / error',async()=>{
  const ok=createReleaseQualificationRead({octokit:fakeOctokit(),
    sourceSha:SOURCE,pins:pins()});
  const t1=await ok.readTarget();
  assert.equal(t1.targetState,'OBSERVED');
  assert.equal(t1.targetSha,TARGET);
  assert.equal(t1.targetTree,TARGET_TREE);
  const badHead=createReleaseQualificationRead({octokit:fakeOctokit(
    {branchSha:'not-a-sha'}),sourceSha:SOURCE,pins:pins()});
  assert.equal((await badHead.readTarget()).targetState,'UNKNOWN');
  const noTree=createReleaseQualificationRead({octokit:fakeOctokit(
    {commitTrees:{[SOURCE]:SOURCE_TREE}}),sourceSha:SOURCE,pins:pins()});
  const t3=await noTree.readTarget();
  assert.equal(t3.targetState,'UNKNOWN');
  assert.equal(t3.targetSha,null,'tree-missing target must not leak a half answer');
  const throws=createReleaseQualificationRead({octokit:fakeOctokit(
    {branchThrows:true}),sourceSha:SOURCE,pins:pins()});
  assert.equal((await throws.readTarget()).targetReason,
    'RELEASE_TARGET_READ_UNAVAILABLE');
  const none=createReleaseQualificationRead({sourceSha:SOURCE,pins:pins()});
  assert.equal((await none.readTarget()).targetState,'UNKNOWN');
});
test('authority: absent registries stay honest UNKNOWN with null permits',async()=>{
  const read=createReleaseQualificationRead({octokit:fakeOctokit(),
    openStore:null,sourceSha:SOURCE,pins:pins()});
  const a=await read.readAuthority();
  assert.equal(a.rqState,'UNKNOWN');
  assert.equal(a.ownerGrantState,'UNKNOWN');
  assert.equal(a.permits,null);
});
test('authority: fresh RQ + grant + live-target-bound permits surface',async()=>{
  const rqKey=key('tally:v2:release-rq:',
    [SOURCE,SOURCE_TREE,TARGET,TARGET_TREE,'MERGE']);
  const grantKey=key('tally:v2:release-grant:',
    [SOURCE,TARGET,'MERGE']);
  const permitKey=op=>key('tally:v2:release-permit:',
    [op,SOURCE,TARGET]);
  const rows=new Map([
    [rqKey,{value:{state:'QUALIFIED',verifierIndependentlyBound:true,
      sourceSha:SOURCE,targetSha:TARGET}}],
    [grantKey,{value:{state:'GRANTED',sourceSha:SOURCE,targetSha:TARGET}}],
    [permitKey('merge'),{value:{operation:'merge',state:'GRANTED',
      sourceSha:SOURCE,targetSha:TARGET,grantedAt:'2026-11-20T00:00:00Z',
      expiresAt:'2099-01-01T00:00:00Z'}}],
    [permitKey('tag'),{value:{operation:'tag',state:'DENIED',
      sourceSha:SOURCE,targetSha:TARGET}}]]);
  const read=createReleaseQualificationRead({octokit:fakeOctokit(),
    openStore:fakeStore(rows).openStore,sourceSha:SOURCE,pins:pins(),
    now:()=>'2026-11-20T12:00:00Z'});
  const a=await read.readAuthority();
  assert.equal(a.rqState,'QUALIFIED');
  assert.equal(a.ownerGrantState,'GRANTED');
  assert.deepEqual(a.permits.map(p=>[p.operation,p.state]),
    [['merge','GRANTED'],['tag','DENIED']]);
  assert.equal(a.targetSha,TARGET);
});
test('authority: stale bindings, wrong target, expired permits degrade',async()=>{
  // RQ row bound to a DIFFERENT target: the observed live target no longer
  // matches the qualified receipt → the UI must not show QUALIFIED.
  const staleRq=key('tally:v2:release-rq:',
    [SOURCE,SOURCE_TREE,'f'.repeat(40),TARGET_TREE,'MERGE']);
  const rows=new Map([
    [staleRq,{value:{state:'QUALIFIED',verifierIndependentlyBound:true,
      sourceSha:SOURCE,targetSha:'f'.repeat(40)}}]]);
  const read=createReleaseQualificationRead({octokit:fakeOctokit(),
    openStore:fakeStore(rows).openStore,sourceSha:SOURCE,pins:pins()});
  const a=await read.readAuthority();
  assert.equal(a.rqState,'UNKNOWN','no exact-target RQ row found');
  // Expired permit degrades to UNKNOWN state, never GRANTED.
  const permitKey=key('tally:v2:release-permit:',
    ['merge',SOURCE,TARGET]);
  const rows2=new Map([
    [permitKey,{value:{operation:'merge',state:'GRANTED',
      sourceSha:SOURCE,targetSha:TARGET,
      expiresAt:'2026-11-20T01:00:00Z'}}]]);
  const read2=createReleaseQualificationRead({octokit:fakeOctokit(),
    openStore:fakeStore(rows2).openStore,sourceSha:SOURCE,pins:pins(),
    now:()=>'2026-11-20T12:00:00Z'});
  const a2=await read2.readAuthority();
  assert.deepEqual(a2.permits.map(p=>[p.operation,p.state]),
    [['merge','UNKNOWN']]);
  // Unreadable store rows answer null → no permit entry at all.
  const read3=createReleaseQualificationRead({octokit:fakeOctokit(),
    openStore:async()=>{throw new Error('locked');},sourceSha:SOURCE,
    pins:pins()});
  const a3=await read3.readAuthority();
  assert.equal(a3.rqState,'UNKNOWN');
  assert.deepEqual(a3.permits,[]);
});
test('factory contract: bad source SHA refuses construction',()=>{
  assert.throws(()=>createReleaseQualificationRead({sourceSha:'nope'}),
    /RELEASE_QUALIFICATION_SOURCE_REQUIRED/);
});
