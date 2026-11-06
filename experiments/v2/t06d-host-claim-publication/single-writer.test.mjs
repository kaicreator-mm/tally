import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createHostSingleWriterClaimController} from './single-writer.mjs';
import {TARGET,readDispatchStateEventsFromClient} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';
import {terminalEventForReservation,createHostBoundNativeGitHubClaimPublisher} from '../../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';

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
  exclusiveHostControl:true,writerId:'tally-host:one',
  repository:'kaicreator-mm/tally',githubWriterLogin:'tally-host[bot]'};
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

const historical={phase:'PUBLISHED_VERIFIED',repository:TARGET.repository,issueRef:TARGET.issueRef,
  sourceSha:TARGET.sourceSha,dispatchId:TARGET.dispatchId,protectedClaimKey:TARGET.protectedClaimKey,
  generation:1,actorRole:'builder',operatorKind:'other',operatorId:'zcode:candidate-publication:1',
  writerId:'tally-host:win32:candidate-publication:1',commentId:TARGET.claimCommentId,commentRef:TARGET.claimCommentRef};
async function seedFixture(store,value=historical){
  await store.put(TARGET.casKey,{...value,phase:'PENDING_GITHUB_PUBLISH'},{mustBeAbsent:true});
  await store.put(TARGET.casKey,value,{mustMatchRevision:0});
}
function terminalPorts({preflight,readback,publish}={}){
  const p=ports();let posts=0;
  p.source.readTerminalizationPreflight=preflight??(async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
    complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',occurredAt:new Date().toISOString()}));
  p.source.readTerminalEvent=readback??(async()=>({status:'EXACT',complete:true,edited:false,commentId:900,
    commentRef:TARGET.issueRef+'#issuecomment-900'}));
  p.publisher.publishDispatchStateChange=async r=>{posts++;return publish?publish(r):
    {commentId:900,commentRef:TARGET.issueRef+'#issuecomment-900'};};
  return {...p,posts:()=>posts};
}
test('terminal transition uses public revision CAS, preserves Claim and survives reopen',async()=>{
  await harness(async({store,reopen})=>{
    await seedFixture(store);const p=terminalPorts();
    const result=await controller(store,p).terminalizePublishedClaim();
    assert.equal(result.claimTerminality,'VERIFIED');assert.equal(p.posts(),1);
    for(const field of ['authorized','admitted','taskPass','releasePass'])assert.equal(result[field],false);
    const reopened=await reopen(),row=await reopened.get(TARGET.casKey);
    assert.equal(row.revision,3);assert.equal(row.value.phase,'TERMINAL_VERIFIED');
    assert.equal(row.value.commentRef,TARGET.claimCommentRef);assert.equal(row.value.terminalCommentId,900);
    assert.equal((await controller(reopened,p).terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
    assert.equal(p.posts(),1);
  });
});
test('wrong identity, missing proof, lost ACK and unverified native readback remain fail closed',async()=>{
  for(const change of [{sourceSha:'b'.repeat(40)},{dispatchId:'wrong'},
    {protectedClaimKey:'wrong'},{generation:2},{commentId:1}])await harness(async({store})=>{
      await seedFixture(store,{...historical,...change});const p=terminalPorts();
      assert.equal((await controller(store,p).terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
      assert.equal(p.posts(),0);assert.equal((await store.get(TARGET.casKey)).revision,1);
    });
  for(const options of [{preflight:async()=>({complete:false})},
    {publish:async()=>{throw Error('lost');},readback:async()=>({status:'UNKNOWN'})},
    {readback:async()=>({status:'UNKNOWN'})}])await harness(async({store})=>{
      await seedFixture(store);const p=terminalPorts(options),c=controller(store,p);
      assert.equal((await c.terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
      const row=await store.get(TARGET.casKey);
      assert.equal(row.value.phase,options.preflight?'PUBLISHED_VERIFIED':'PENDING_GITHUB_TERMINALIZATION');
      await c.terminalizePublishedClaim();assert.ok(p.posts()<=1);
    });
});
test('lost ACK with the exact event already appended reconciles read-only; exactly one POST',async()=>{
  await harness(async({store,reopen})=>{
    await seedFixture(store);
    // End-to-end append-then-throw: the real native transport appends the
    // exact terminal event to GitHub-visible comments and then loses the
    // ACK. Reconciliation must use read-only exact native pagination only.
    const comments=[];let posts=0;
    const octokit={rest:{issues:{
      async get(){return {data:{id:2030,number:203,html_url:TARGET.issueRef,state:'open',
        comments:comments.length,updated_at:'now',body:`version/v2@${TARGET.sourceSha}`}};},
      async listComments(){throw Error('pagination required');},
      async getComment({comment_id}){return {data:comments.find(c=>c.id===comment_id)};},
      async createComment({body}){
        posts++;
        comments.push({id:901,body,html_url:TARGET.issueRef+'#issuecomment-901',
          issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/203',
          user:{login:'kaicreator-mm'},created_at:'now',updated_at:'now'});
        throw Error('NATIVE_GITHUB_ACK_LOST');
      }},
      repos:{async getCommit(){return {data:{sha:TARGET.sourceSha}};}}},
      paginate:{async *iterator(){yield {data:[...comments]};}}};
    const terminalAuthority={...writerAuthority,githubWriterLogin:'kaicreator-mm'};
    const p=ports();
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async reservation=>readDispatchStateEventsFromClient({
      octokit,writerLogin:terminalAuthority.githubWriterLogin,
      expectedEvent:terminalEventForReservation(reservation,terminalAuthority)});
    const publisher=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority:terminalAuthority});
    const c=createHostSingleWriterClaimController({store,source:p.source,publisher,
      writerAuthority:terminalAuthority});
    const result=await c.terminalizePublishedClaim();
    assert.equal(result.claimTerminality,'VERIFIED',JSON.stringify(result));
    assert.equal(result.terminalCommentRef,TARGET.issueRef+'#issuecomment-901');
    assert.equal(posts,1);
    const reopened=await reopen(),row=await reopened.get(TARGET.casKey);
    assert.equal(row.revision,3);assert.equal(row.value.phase,'TERMINAL_VERIFIED');
    assert.equal(row.value.terminalCommentId,901);
    assert.equal(row.value.terminalEvidence,'durable proof');
    assert.equal((await c.terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
    assert.equal(posts,1);
  });
});
test('stranded pending reservation recovers read-only once pagination is complete; exactly one POST total',async()=>{
  await harness(async({store,reopen})=>{
    await seedFixture(store);
    // End-to-end stranded reservation: the POST appended the exact terminal
    // event and lost its ACK, and the first readback saw incomplete native
    // pagination. Later calls must reconcile read-only — complete pagination
    // proving the exact unedited event, public CAS mustMatchRevision=2, and
    // never a second POST — while every earlier failure stays PENDING.
    const comments=[];let posts=0,paginationComplete=false;
    const octokit={rest:{issues:{
      async get(){return {data:{id:2030,number:203,html_url:TARGET.issueRef,state:'open',
        comments:comments.length,updated_at:'now',body:`version/v2@${TARGET.sourceSha}`}};},
      async listComments(){throw Error('pagination required');},
      async getComment({comment_id}){return {data:comments.find(c=>c.id===comment_id)};},
      async createComment({body}){
        posts++;
        comments.push({id:901,body,html_url:TARGET.issueRef+'#issuecomment-901',
          issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/203',
          user:{login:'kaicreator-mm'},created_at:'now',updated_at:'now'});
        throw Error('NATIVE_GITHUB_ACK_LOST');
      }},
      repos:{async getCommit(){return {data:{sha:TARGET.sourceSha}};}}},
      paginate:{async *iterator(){
        if(!paginationComplete){yield {data:null};return;}
        yield {data:[...comments]};
      }}};
    const terminalAuthority={...writerAuthority,githubWriterLogin:'kaicreator-mm'};
    const p=ports();
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async reservation=>readDispatchStateEventsFromClient({
      octokit,writerLogin:terminalAuthority.githubWriterLogin,
      expectedEvent:terminalEventForReservation(reservation,terminalAuthority)});
    p.publisher.publishDispatchStateChange=async r=>
      createHostBoundNativeGitHubClaimPublisher({octokit,
        owner:'kaicreator-mm',repo:'tally',writerAuthority:terminalAuthority})
        .publishDispatchStateChange(r);
    const c=createHostSingleWriterClaimController({store,source:p.source,
      publisher:p.publisher,writerAuthority:terminalAuthority});
    const first=await c.terminalizePublishedClaim();
    assert.equal(first.reason,'TERMINAL_PUBLICATION_AMBIGUOUS');
    assert.equal(posts,1);
    let row=await store.get(TARGET.casKey);
    assert.equal(row.revision,2);assert.equal(row.value.phase,'PENDING_GITHUB_TERMINALIZATION');
    const stillBroken=await c.terminalizePublishedClaim();
    assert.equal(stillBroken.reason,'TERMINAL_READBACK_NOT_VERIFIED');
    assert.equal(posts,1);
    row=await store.get(TARGET.casKey);
    assert.equal(row.revision,2);assert.equal(row.value.phase,'PENDING_GITHUB_TERMINALIZATION');
    paginationComplete=true;
    const recovered=await c.terminalizePublishedClaim();
    assert.equal(recovered.claimTerminality,'VERIFIED',JSON.stringify(recovered));
    assert.equal(recovered.reason,'PENDING_RESERVATION_RECONCILED_READ_ONLY');
    assert.equal(recovered.terminalCommentRef,TARGET.issueRef+'#issuecomment-901');
    assert.equal(posts,1);
    const reopened=await reopen();row=await reopened.get(TARGET.casKey);
    assert.equal(row.revision,3);assert.equal(row.value.phase,'TERMINAL_VERIFIED');
    assert.equal(row.value.terminalCommentId,901);
    assert.equal(row.value.terminalEvidence,'durable proof');
    assert.equal(row.value.commentRef,TARGET.claimCommentRef);
    assert.equal((await createHostSingleWriterClaimController({store:reopened,source:p.source,
      publisher:p.publisher,writerAuthority:terminalAuthority}).terminalizePublishedClaim())
      .reason,'TARGET_CLAIM_NOT_EXACT');
    assert.equal(posts,1);
  });
});
test('CAS contention during pending read-only reconciliation never verifies and preserves the row',async()=>{
  await harness(async({store})=>{
    await seedFixture(store);let posts=0,reads=0;
    const p=ports();
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async()=>{
      reads++;
      if(reads===1)return {status:'ABSENT',complete:true};
      // A competing writer moves the CAS row between the exact read-only
      // readback and the mustMatchRevision=2 confirm.
      const current=await store.get(TARGET.casKey);
      await store.put(TARGET.casKey,{...current.value,externalChange:true},
        {mustMatchRevision:current.revision});
      return {status:'EXACT',complete:true,edited:false,commentId:901,
        commentRef:TARGET.issueRef+'#issuecomment-901'};
    };
    p.publisher.publishDispatchStateChange=async()=>{posts++;throw Error('ACK lost');};
    const c=controller(store,p);
    assert.equal((await c.terminalizePublishedClaim()).reason,'TERMINAL_PUBLICATION_AMBIGUOUS');
    const contended=await c.terminalizePublishedClaim();
    assert.equal(contended.claimTerminality,'UNKNOWN');
    assert.equal(contended.reason,'TERMINAL_CONFIRMATION_AMBIGUOUS');
    assert.equal(posts,1);
    const row=await store.get(TARGET.casKey);
    assert.equal(row.revision,3);
    assert.equal(row.value.externalChange,true);
    assert.notEqual(row.value.phase,'TERMINAL_VERIFIED');
  });
});
test('an inexact or invalid pending reservation row is never reconciled; zero POSTs, row preserved',async()=>{
  const pending={...historical,phase:'PENDING_GITHUB_TERMINALIZATION',
    terminalWriterId:writerAuthority.writerId,terminalEvidence:'durable proof',
    terminalOccurredAt:new Date().toISOString(),terminalReservationRevision:2};
  for(const change of [{terminalWriterId:'other-host'},{dispatchId:'wrong'},
    {operatorId:'someone-else:1'},{terminalEvidence:''},
    {terminalEvidence:'x'.repeat(16385)},{terminalReservationRevision:3}])
    await harness(async({store})=>{
      await seedFixture(store);
      await store.put(TARGET.casKey,{...pending,...change},{mustMatchRevision:1});
      const p=terminalPorts();
      const result=await controller(store,p).terminalizePublishedClaim();
      assert.equal(result.claimTerminality,'UNKNOWN');
      assert.ok(['PENDING_TERMINAL_RESERVATION_NOT_EXACT',
        'TERMINAL_EVENT_NOT_VALID_BEFORE_RESERVATION'].includes(result.reason),result.reason);
      assert.equal(p.posts(),0);
      const row=await store.get(TARGET.casKey);
      assert.equal(row.revision,2);
      assert.equal(row.value.phase,'PENDING_GITHUB_TERMINALIZATION');
      assert.equal(row.value.terminalEvidence,({...pending,...change}).terminalEvidence);
    });
});
test('absent, incomplete, edited or conflicting readback after a lost ACK stays PENDING with no second POST',async()=>{
  const readbacks=[
    {status:'ABSENT',complete:true},
    {status:'UNKNOWN'},
    {status:'UNKNOWN',reason:'INCOMPLETE_PAGINATION'},
    {status:'EXACT',complete:true,edited:true,commentId:901,
      commentRef:TARGET.issueRef+'#issuecomment-901'},
    {status:'EXACT',complete:true,edited:false,commentId:901,
      commentRef:'https://github.com/evil/repo/issues/1#issuecomment-901'}];
  for(const observed of readbacks)await harness(async({store})=>{
    await seedFixture(store);let posts=0;
    const p=ports();
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async()=>structuredClone(observed);
    p.publisher.publishDispatchStateChange=async()=>{posts++;throw Error('ACK lost');};
    const c=controller(store,p);
    const result=await c.terminalizePublishedClaim();
    assert.equal(result.claimTerminality,'UNKNOWN');
    assert.equal(result.reason,'TERMINAL_PUBLICATION_AMBIGUOUS');
    assert.equal(posts,1);
    const row=await store.get(TARGET.casKey);
    assert.equal(row.revision,2);
    assert.equal(row.value.phase,'PENDING_GITHUB_TERMINALIZATION');
    // Later reconciliation is read-only: the still-negative readback leaves
    // the pending reservation untouched and never issues a second POST.
    const again=await c.terminalizePublishedClaim();
    assert.equal(again.claimTerminality,'UNKNOWN');
    assert.equal(again.reason,'TERMINAL_READBACK_NOT_VERIFIED');
    assert.equal(posts,1);
    const rowAfter=await store.get(TARGET.casKey);
    assert.equal(rowAfter.revision,2);
    assert.equal(rowAfter.value.phase,'PENDING_GITHUB_TERMINALIZATION');
  });
});
test('oversized evidence or an unbound writer identity is rejected before any CAS write or POST',async()=>{
  await harness(async({store})=>{
    await seedFixture(store);
    const p=terminalPorts({preflight:async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'x'.repeat(16385),
      occurredAt:new Date().toISOString()})});
    const oversized=await controller(store,p).terminalizePublishedClaim();
    assert.equal(oversized.claimTerminality,'UNKNOWN');
    assert.equal(oversized.reason,'TERMINAL_EVENT_NOT_VALID_BEFORE_RESERVATION');
    assert.equal(p.posts(),0);
    let row=await store.get(TARGET.casKey);
    assert.equal(row.revision,1);assert.equal(row.value.phase,'PUBLISHED_VERIFIED');
    const unbound=await createHostSingleWriterClaimController({store,...terminalPorts(),
      writerAuthority:{...writerAuthority,repository:'other/repo'}}).terminalizePublishedClaim();
    assert.equal(unbound.reason,'TERMINAL_EVENT_NOT_VALID_BEFORE_RESERVATION');
    row=await store.get(TARGET.casKey);
    assert.equal(row.revision,1);assert.equal(row.value.phase,'PUBLISHED_VERIFIED');
  });
});
test('competing expected-revision reservations publish at most once',async()=>{
  await harness(async({store})=>{
    await seedFixture(store);const p=terminalPorts();
    const results=await Promise.all([controller(store,p).terminalizePublishedClaim(),controller(store,p).terminalizePublishedClaim()]);
    assert.equal(results.filter(x=>x.claimTerminality==='VERIFIED').length,1);assert.equal(p.posts(),1);
  });
});
test('a changed fence or reservation before POST and changed revision before final CAS never verify',async()=>{
  for(const kind of ['fence','beforePost','beforeFinal'])await harness(async({store})=>{
    await seedFixture(store);let preflightCalls=0;const p=terminalPorts();
    const originalPreflight=p.source.readTerminalizationPreflight;
    p.source.readTerminalizationPreflight=async row=>{
      const facts=await originalPreflight(row);preflightCalls++;
      if(preflightCalls===2&&kind==='fence')return {...facts,fenceId:'other'};
      if(preflightCalls===2&&kind==='beforePost'){
        const current=await store.get(TARGET.casKey);
        await store.put(TARGET.casKey,{...current.value,externalChange:true},{mustMatchRevision:current.revision});
      }
      return facts;
    };
    if(kind==='beforeFinal')p.source.readTerminalEvent=async()=>{
      const current=await store.get(TARGET.casKey);
      await store.put(TARGET.casKey,{...current.value,externalChange:true},{mustMatchRevision:current.revision});
      return {status:'EXACT',complete:true,edited:false,commentId:900,commentRef:TARGET.issueRef+'#issuecomment-900'};
    };
    const result=await controller(store,p).terminalizePublishedClaim();
    assert.equal(result.claimTerminality,'UNKNOWN');assert.equal(p.posts(),kind==='beforeFinal'?1:0);
    assert.equal((await store.get(TARGET.casKey)).value.phase,'PENDING_GITHUB_TERMINALIZATION');
  });
});
