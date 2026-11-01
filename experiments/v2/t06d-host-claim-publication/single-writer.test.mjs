import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createHostSingleWriterClaimController} from './single-writer.mjs';

const sourceSha='a'.repeat(40);
const req={owner:'kaicreator-mm',repo:'tally',issueNumber:125,
  sourceSha,actorRole:'builder',dispatchId:'dispatch:125:t06d:1',
  operatorKind:'codex',operatorId:'codex:builder1',expectedGeneration:0};
const goodFacts=i=>({
  sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',sourceQualified:true,
  complete:true,issueRef:i.issueRef,sourceSha:i.sourceSha,
  issueState:'open',workflowState:'state:ready',
  dependencyState:'CLEAR',activeIncompatibleClaimCount:0,
  dispatchSourceBound:true,publisherSourceBound:true
});
const exactReadback=(i,commentId)=>({
  sourceKind:'NATIVE_GITHUB_EXACT_READBACK',complete:true,edited:false,
  issueRef:i.issueRef,sourceSha:i.sourceSha,
  dispatchId:i.dispatchId,actorRole:i.actorRole,operatorKind:i.operatorKind,
  operatorId:i.operatorId,generation:i.generation,
  protectedClaimKey:i.protectedClaimKey,event:'DISPATCH_CLAIMED',
  dispatchState:'CLAIMED',commentId,
  commentRef:i.issueRef+'#issuecomment-'+commentId
});
const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',
  exclusiveHostControl:true,writerId:'tally-host:one'};
async function harness(fn){
  const dir=await mkdtemp(join(tmpdir(),'tally-v2-admission-cas-'));
  const path=join(dir,'cas.sqlite');
  let store=await openSqliteCasStore({path,busyTimeoutMs:1000});
  try{await fn({store,path,async reopen(){await store.close();
    store=await openSqliteCasStore({path,busyTimeoutMs:1000});
    return store;}});}
  finally{await store.close();await rm(dir,{recursive:true,force:true});}
}
function ports(overrides={}){
  const counters={writes:0,reads:0};
  const source={
    async readAdmissionFacts(i){counters.reads++;return goodFacts(i);},
    async readPublishedClaim(i,id){counters.reads++;return exactReadback(i,id);}
  };
  const publisher={async publishClaim(){counters.writes++;return {commentId:72620};}};
  Object.assign(source,overrides.source);
  Object.assign(publisher,overrides.publisher);
  return {source,publisher,counters,writerAuthority};
}
const controller=(store,p)=>createHostSingleWriterClaimController({store,...p});
test('actual DH SQLite CAS: one serialized publication, exact after-state, durable restart; no K Task PASS',async()=>{
  await harness(async({store,path,reopen})=>{
    const p=ports();
    const c=controller(store,p);
    const receipt=await c.reserveAndPublish(req);
    assert.equal(receipt.status,'RECORDED');
    assert.equal(receipt.claimPublication,'VERIFIED_PENDING_KERNEL_ADMISSION');
    assert.equal(receipt.taskPass,false);
    assert.equal(receipt.admitted,false);
    assert.equal(p.counters.writes,1);
    const entries=await store.list('tally:v2:claim-admission:');
    assert.equal(entries.length,1);
    const row=(await store.get(entries[0]))?.value;
    assert.equal(row.phase,'PUBLISHED_VERIFIED');
    assert.equal(row.generation,1);
    assert.equal(row.commentRef,'https://github.com/kaicreator-mm/tally/issues/125#issuecomment-72620');
    const duplicate=await c.reserveAndPublish(req);
    assert.equal(duplicate.reason,'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
    assert.equal(p.counters.writes,1);
    const real=await reopen();
    const after=(await real.get(entries[0]))?.value;
    assert.equal(after.phase,'PUBLISHED_VERIFIED');
    const different=await controller(real,ports()).reserveAndPublish({
      ...req,dispatchId:'dispatch:125:t06d:competing'});
    assert.equal(different.reason,'INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS');
  });
});
test('true DH SQLite CAS makes two competing concurrent reservations physically exclusive',async()=>{
  await harness(async({store})=>{
    const p=ports();
    const a=controller(store,p),b=controller(store,p);
    const [one,two]=await Promise.all([
      a.reserveAndPublish(req),
      b.reserveAndPublish({...req,dispatchId:'dispatch:125:other'})
    ]);
    assert.equal([one,two].filter(x=>x.status==='RECORDED').length,1);
    assert.equal(p.counters.writes,1);
    assert.ok([one.reason,two.reason].some(r=>
      ['CONCURRENT_CLAIM_RESERVATION_LOST','INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS'].includes(r)));
    assert.equal((await store.list('tally:v2:claim-admission:')).length,1);
  });
});
test('no GitHub mutation without qualified native source or generation',async()=>{
  await harness(async({store})=>{
    const p=ports({source:{readAdmissionFacts:async i=>({...goodFacts(i),
      activeIncompatibleClaimCount:1})}});
    const c=controller(store,p);
    assert.equal((await c.reserveAndPublish(req)).reason,'SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
    assert.equal((await c.reserveAndPublish({...req,expectedGeneration:1})).reason,'INVALID_CLAIM_REQUEST');
    assert.equal((await c.reserveAndPublish({...req,sourceSha:'bad'})).reason,'INVALID_CLAIM_REQUEST');
    assert.equal(p.counters.writes,0);
    assert.equal((await store.list('tally:v2:claim-admission:')).length,0);
  });
});
test('GitHub lost ACK leaves PENDING durable reservation; retry MUST NOT publish again',async()=>{
  await harness(async({store})=>{
    const p=ports({publisher:{publishClaim:async()=>{throw Error('GITHUB_TOKEN_PRIVATE');}}});
    const c=controller(store,p);
    const first=await c.reserveAndPublish(req);
    assert.equal(first.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
    assert.equal(JSON.stringify(first).includes('GITHUB_TOKEN_PRIVATE'),false);
    assert.equal((await c.reserveAndPublish(req)).reason,'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
    const keys=await store.list('tally:v2:claim-admission:');
    assert.equal((await store.get(keys[0])).value.phase,'PENDING_GITHUB_PUBLISH');
  });
});
test('edited/wrong native readback never produces an accepted receipt',async()=>{
  for(const change of [
    x=>({...x,edited:true}),
    x=>({...x,sourceSha:'b'.repeat(40)}),
    x=>({...x,operatorId:'untrusted-actor'}),
    x=>({...x,commentRef:'https://github.com/evil/repo/issues/4'})
  ]){
    await harness(async({store})=>{
      const p=ports({source:{readPublishedClaim:async(i,id)=>change(exactReadback(i,id))}});
      const c=controller(store,p);
      const result=await c.reserveAndPublish(req);
      assert.equal(result.reason,'GITHUB_CLAIM_READBACK_NOT_VERIFIED');
      const keys=await store.list('tally:v2:claim-admission:');
      assert.equal((await store.get(keys[0])).value.phase,'PENDING_GITHUB_PUBLISH');
      assert.equal(result.admitted,false);
    });
  }
});
test('publisher ports must be Host-owned; invalid constructor does not create a writer',()=>{
  assert.throws(()=>createHostSingleWriterClaimController({
    store:{get:()=>null,put:()=>{}},...ports(),
    writerAuthority:{mode:'INSECURE_MULTI_WRITER',exclusiveHostControl:true,writerId:'bad'}
  }),/TRUSTED_SINGLE_WRITER_HOST_PORTS_REQUIRED/);
});
