import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash,generateKeyPairSync,randomBytes} from 'node:crypto';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createHostSingleWriterClaimController} from './single-writer.mjs';
import {HOST_RESOURCE_FENCE_SCHEMA,buildFrozenResourceSet,computeDispatchId,
  createHostResourceFenceAuthority,createHostResourceFenceClient,
  createHostResourceFencePort} from './host-resource-fence.mjs';
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
// --- Real shared-Host fence fixture (mirrors host-resource-fence.test.mjs) ---
const FENCE_WRITER_SID='S-1-5-21-2839165489-1832998833-1574237796-1004';
const FENCE_REGISTRY_ID='fence-registry-t06d-wiring-1';
const FENCE_AUTHORITY_KEY=generateKeyPairSync('rsa',{modulusLength:2048});
const FENCE_KEY_A=generateKeyPairSync('rsa',{modulusLength:2048});
const FENCE_KEY_B=generateKeyPairSync('rsa',{modulusLength:2048});
const sha256Hex=value=>createHash('sha256').update(value).digest('hex');
function fenceIdentity({issueNumber=125,files=['experiments/v2/t06d-host-claim-publication/single-writer.test.mjs'],
  reviewId=1000000001,ownerId=1000000002}={}){
  const repository='kaicreator-mm/tally';
  const protectedClaimKey=`${repository}#${issueNumber}:builder:__default__`;
  const issueRef=`https://github.com/${repository}/issues/${issueNumber}`;
  const reviewRef=`https://github.com/${repository}/pull/376#pullrequestreview-${reviewId}`;
  const ownerApprovalRef=`https://github.com/${repository}/issues/${issueNumber}#issuecomment-${ownerId}`;
  const writeSetSha256=sha256Hex(JSON.stringify([...files].sort()));
  return {
    schema:HOST_RESOURCE_FENCE_SCHEMA,repository,issueRef,protectedClaimKey,
    generation:1,sourceSha,
    dispatchId:computeDispatchId({repository,issueNumber,sourceSha,
      taskPackBlobSha:'b'.repeat(40),reviewRef,ownerApprovalRef,writeSetSha256}),
    taskPackRef:`https://github.com/${repository}/blob/${sourceSha}/`+
      'docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json',
    taskPackBlobSha:'b'.repeat(40),reviewRef,ownerApprovalRef,writeSetSha256,
    exclusiveResources:[
      `claim-key:${protectedClaimKey}`,
      `cas:tally:v2:claim-admission:${sha256Hex(protectedClaimKey)}`,
      `cas:tally:v2:task-writeset:${issueNumber}`,
      `github-issue-comment:${repository}#${issueNumber}`,
      ...files.map(file=>`file:${file}`)
    ]
  };
}
let fenceCounter=0;
async function openFence(dir,specs){
  const seq=++fenceCounter;
  const store=await openSqliteCasStore({path:join(dir,`fence-${seq}.sqlite`),
    busyTimeoutMs:1000});
  const anchorPath=join(dir,`fence-anchor-${seq}.ndjson`);
  const entries=specs.map(spec=>({
    repository:spec.identity.repository,
    protectedClaimKey:spec.identity.protectedClaimKey,
    writerId:spec.writerId,sourceSha:spec.identity.sourceSha,
    dispatchId:spec.identity.dispatchId,
    frozenResourceSetSha256:buildFrozenResourceSet(spec.identity).frozenResourceSetSha256,
    publicKeyPem:spec.keyPair.publicKey.export({type:'spki',format:'pem'})
  }));
  const authority=createHostResourceFenceAuthority({store,registryId:FENCE_REGISTRY_ID,
    anchorPath,writerSid:FENCE_WRITER_SID,
    authorityPrivateKey:FENCE_AUTHORITY_KEY.privateKey,authorizedClients:entries});
  await authority.initialize();
  const clients=specs.map(spec=>createHostResourceFenceClient({
    privateKey:spec.keyPair.privateKey,authorityPublicKeyPem:authority.publicKeyPem,
    registryId:authority.registryId,writerId:spec.writerId,identity:spec.identity,
    nonce:randomBytes(32).toString('hex'),send:request=>authority.handle(request)}));
  const ports=specs.map(spec=>createHostResourceFencePort({
    store,registryId:authority.registryId,anchorPath,
    authorityPublicKeyPem:authority.publicKeyPem,writerSid:FENCE_WRITER_SID,
    repository:spec.identity.repository,writerId:spec.writerId,
    protectedClaimKey:spec.identity.protectedClaimKey,
    sourceSha:spec.identity.sourceSha,
    frozenResourceSetSha256:buildFrozenResourceSet(spec.identity).frozenResourceSetSha256,
    expectedHead:()=>authority.anchorHead,withReadLock:task=>authority.withReadLock(task)}));
  return {store,anchorPath,authority,clients,ports,specs};
}
const builderSpec=({keyPair=FENCE_KEY_A,writerId=writerAuthority.writerId,...identity}={})=>
  ({identity:fenceIdentity(identity),keyPair,writerId});
async function harness(fn){
  const dir=await mkdtemp(join(tmpdir(),'tally-v2-admission-cas-'));
  const path=join(dir,'cas.sqlite');
  const fences=[];
  let store=await openSqliteCasStore({path,busyTimeoutMs:1000});
  try{await fn({store,path,
    async openFence(specs){const fence=await openFence(dir,specs);fences.push(fence);return fence;},
    async reopen(){await store.close();
      store=await openSqliteCasStore({path,busyTimeoutMs:1000});
      return store;}});}
  finally{for(const fence of fences){try{await fence.store.close();}catch{}}
    await store.close();await rm(dir,{recursive:true,force:true});}
}
function ports(fence,overrides={}){
  const counters={writes:0,reads:0};
  const source={
    async readAdmissionFacts(i){counters.reads++;return goodFacts(i);},
    async readPublishedClaim(i,id){counters.reads++;return exactReadback(i,id);}
  };
  const publisher={async publishClaim(){counters.writes++;return {commentId:72620};}};
  Object.assign(source,overrides.source);
  Object.assign(publisher,overrides.publisher);
  return {source,publisher,counters,writerAuthority,fenceClient:fence.clients[0]};
}
const controller=(store,p)=>createHostSingleWriterClaimController({store,...p});
test('actual DH SQLite CAS: one serialized publication, exact after-state, durable restart; no K Task PASS',async()=>{
  await harness(async({store,path,reopen,openFence})=>{
    const fence=await openFence([builderSpec()]);
    const p=ports(fence);
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
    const different=await controller(real,ports(await openFence([builderSpec()])))
      .reserveAndPublish({...req,dispatchId:'dispatch:125:t06d:competing'});
    assert.equal(different.reason,'INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS');
  });
});
test('true DH SQLite CAS makes two competing concurrent reservations physically exclusive',async()=>{
  await harness(async({store,openFence})=>{
    const p=ports(await openFence([builderSpec()]));
    const a=controller(store,p),b=controller(store,p);
    const [one,two]=await Promise.all([
      a.reserveAndPublish(req),
      b.reserveAndPublish({...req,dispatchId:'dispatch:125:other'})
    ]);
    assert.equal([one,two].filter(x=>x.status==='RECORDED').length,1);
    assert.equal(p.counters.writes,1);
    assert.ok([one.reason,two.reason].some(r=>
      ['CONCURRENT_CLAIM_RESERVATION_LOST','INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS',
        'FENCE_OPERATION_FAILED'].includes(r)));
    assert.equal((await store.list('tally:v2:claim-admission:')).length,1);
  });
});
test('no GitHub mutation without qualified native source or generation',async()=>{
  await harness(async({store,openFence})=>{
    const p=ports(await openFence([builderSpec()]),{source:{readAdmissionFacts:async i=>({...goodFacts(i),
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
  await harness(async({store,openFence})=>{
    const p=ports(await openFence([builderSpec()]),{publisher:{publishClaim:async()=>{throw Error('GITHUB_TOKEN_PRIVATE');}}});
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
    await harness(async({store,openFence})=>{
      const p=ports(await openFence([builderSpec()]),{source:{readPublishedClaim:async(i,id)=>change(exactReadback(i,id))}});
      const c=controller(store,p);
      const result=await c.reserveAndPublish(req);
      assert.equal(result.reason,'GITHUB_CLAIM_READBACK_NOT_VERIFIED');
      const keys=await store.list('tally:v2:claim-admission:');
      assert.equal((await store.get(keys[0])).value.phase,'PENDING_GITHUB_PUBLISH');
      assert.equal(result.admitted,false);
    });
  }
});
test('publisher ports must be Host-owned; invalid constructor does not create a writer',async()=>{
  await harness(async({openFence})=>{
    const good=ports(await openFence([builderSpec()]));
    assert.throws(()=>createHostSingleWriterClaimController({
      store:{get:()=>null,put:()=>{}},...good,
      writerAuthority:{mode:'INSECURE_MULTI_WRITER',exclusiveHostControl:true,writerId:'bad'}
    }),/TRUSTED_SINGLE_WRITER_HOST_PORTS_REQUIRED/);
    const {fenceClient:_,...withoutFence}=ports(await openFence([builderSpec()]));
    assert.throws(()=>createHostSingleWriterClaimController({
      store:{get:()=>null,put:()=>{}},...withoutFence
    }),/TRUSTED_SINGLE_WRITER_HOST_PORTS_REQUIRED/);
  });
});

const historical={phase:'PUBLISHED_VERIFIED',repository:TARGET.repository,issueRef:TARGET.issueRef,
  sourceSha:TARGET.sourceSha,dispatchId:TARGET.dispatchId,protectedClaimKey:TARGET.protectedClaimKey,
  generation:1,actorRole:'builder',operatorKind:'other',operatorId:'zcode:candidate-publication:1',
  writerId:'tally-host:win32:candidate-publication:1',commentId:TARGET.claimCommentId,commentRef:TARGET.claimCommentRef};
async function seedFixture(store,value=historical){
  await store.put(TARGET.casKey,{...value,phase:'PENDING_GITHUB_PUBLISH'},{mustBeAbsent:true});
  await store.put(TARGET.casKey,value,{mustMatchRevision:0});
}
function terminalPorts(fence,{preflight,readback,publish}={}){
  const p=ports(fence);let posts=0;
  p.source.readTerminalizationPreflight=preflight??(async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
    complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',occurredAt:new Date().toISOString()}));
  p.source.readTerminalEvent=readback??(async()=>({status:'EXACT',complete:true,edited:false,commentId:900,
    commentRef:TARGET.issueRef+'#issuecomment-900'}));
  p.publisher.publishDispatchStateChange=async r=>{posts++;return publish?publish(r):
    {commentId:900,commentRef:TARGET.issueRef+'#issuecomment-900'};};
  return {...p,posts:()=>posts};
}
test('terminal transition uses public revision CAS, preserves Claim and survives reopen',async()=>{
  await harness(async({store,reopen,openFence})=>{
    await seedFixture(store);const p=terminalPorts(await openFence([builderSpec()]));
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
    {protectedClaimKey:'wrong'},{generation:2},{commentId:1}])await harness(async({store,openFence})=>{
      await seedFixture(store,{...historical,...change});
      const p=terminalPorts(await openFence([builderSpec()]));
      assert.equal((await controller(store,p).terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
      assert.equal(p.posts(),0);assert.equal((await store.get(TARGET.casKey)).revision,1);
    });
  for(const options of [{preflight:async()=>({complete:false})},
    {publish:async()=>{throw Error('lost');},readback:async()=>({status:'UNKNOWN'})},
    {readback:async()=>({status:'UNKNOWN'})}])await harness(async({store,openFence})=>{
      await seedFixture(store);const p=terminalPorts(await openFence([builderSpec()]),options),c=controller(store,p);
      assert.equal((await c.terminalizePublishedClaim()).claimTerminality,'UNKNOWN');
      const row=await store.get(TARGET.casKey);
      assert.equal(row.value.phase,options.preflight?'PUBLISHED_VERIFIED':'PENDING_GITHUB_TERMINALIZATION');
      await c.terminalizePublishedClaim();assert.ok(p.posts()<=1);
    });
});
test('lost ACK with the exact event already appended reconciles read-only; exactly one POST',async()=>{
  await harness(async({store,reopen,openFence})=>{
    await seedFixture(store);
    const fence=await openFence([builderSpec()]);
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
    const p=ports(fence);
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async reservation=>readDispatchStateEventsFromClient({
      octokit,writerLogin:terminalAuthority.githubWriterLogin,
      expectedEvent:terminalEventForReservation(reservation,terminalAuthority)});
    const publisher=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority:terminalAuthority,
      fencePort:fence.ports[0]});
    const c=createHostSingleWriterClaimController({store,source:p.source,publisher,
      writerAuthority:terminalAuthority,fenceClient:fence.clients[0]});
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
  await harness(async({store,reopen,openFence})=>{
    await seedFixture(store);
    const fence=await openFence([builderSpec()]);
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
    const p=ports(fence);
    p.source.readTerminalizationPreflight=async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'durable proof',
      occurredAt:new Date().toISOString()});
    p.source.readTerminalEvent=async reservation=>readDispatchStateEventsFromClient({
      octokit,writerLogin:terminalAuthority.githubWriterLogin,
      expectedEvent:terminalEventForReservation(reservation,terminalAuthority)});
    const terminalPublisher=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority:terminalAuthority,
      fencePort:fence.ports[0]});
    p.publisher.publishDispatchStateChange=async r=>
      terminalPublisher.publishDispatchStateChange(r);
    const c=createHostSingleWriterClaimController({store,source:p.source,
      publisher:p.publisher,writerAuthority:terminalAuthority,fenceClient:fence.clients[0]});
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
      publisher:p.publisher,writerAuthority:terminalAuthority,
      fenceClient:fence.clients[0]}).terminalizePublishedClaim())
      .reason,'TARGET_CLAIM_NOT_EXACT');
    assert.equal(posts,1);
  });
});
test('CAS contention during pending read-only reconciliation never verifies and preserves the row',async()=>{
  await harness(async({store,openFence})=>{
    await seedFixture(store);let posts=0,reads=0;
    const p=ports(await openFence([builderSpec()]));
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
    await harness(async({store,openFence})=>{
      await seedFixture(store);
      await store.put(TARGET.casKey,{...pending,...change},{mustMatchRevision:1});
      const p=terminalPorts(await openFence([builderSpec()]));
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
  for(const observed of readbacks)await harness(async({store,openFence})=>{
    await seedFixture(store);let posts=0;
    const p=ports(await openFence([builderSpec()]));
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
  await harness(async({store,openFence})=>{
    await seedFixture(store);
    const p=terminalPorts(await openFence([builderSpec()]),{preflight:async()=>({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',
      complete:true,targetRevision:1,fenceId:'fence:one',evidence:'x'.repeat(16385),
      occurredAt:new Date().toISOString()})});
    const oversized=await controller(store,p).terminalizePublishedClaim();
    assert.equal(oversized.claimTerminality,'UNKNOWN');
    assert.equal(oversized.reason,'TERMINAL_EVENT_NOT_VALID_BEFORE_RESERVATION');
    assert.equal(p.posts(),0);
    let row=await store.get(TARGET.casKey);
    assert.equal(row.revision,1);assert.equal(row.value.phase,'PUBLISHED_VERIFIED');
    const unbound=await createHostSingleWriterClaimController({store,
      ...terminalPorts(await openFence([builderSpec()])),
      writerAuthority:{...writerAuthority,repository:'other/repo'}}).terminalizePublishedClaim();
    assert.equal(unbound.reason,'TERMINAL_EVENT_NOT_VALID_BEFORE_RESERVATION');
    row=await store.get(TARGET.casKey);
    assert.equal(row.revision,1);assert.equal(row.value.phase,'PUBLISHED_VERIFIED');
  });
});
test('competing expected-revision reservations publish at most once',async()=>{
  await harness(async({store,openFence})=>{
    await seedFixture(store);const p=terminalPorts(await openFence([builderSpec()]));
    const results=await Promise.all([controller(store,p).terminalizePublishedClaim(),controller(store,p).terminalizePublishedClaim()]);
    assert.equal(results.filter(x=>x.claimTerminality==='VERIFIED').length,1);assert.equal(p.posts(),1);
  });
});
test('a changed fence or reservation before POST and changed revision before final CAS never verify',async()=>{
  for(const kind of ['fence','beforePost','beforeFinal'])await harness(async({store,openFence})=>{
    await seedFixture(store);let preflightCalls=0;
    const p=terminalPorts(await openFence([builderSpec()]));
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

// --- #383 fence wiring join tests (real authority, real clients, real port) ---
const gate=()=>{
  let open;
  const promise=new Promise(resolve=>{open=resolve;});
  return {promise,open};
};
const claimOctokit=({onCreate}={})=>{
  const posted=[];
  return {posted,octokit:{rest:{issues:{
    async createComment(input){
      posted.push(input);
      if(onCreate)return onCreate(input);
      const id=90100+posted.length;
      return {data:{id,body:input.body,
        html_url:`https://github.com/kaicreator-mm/tally/issues/125#issuecomment-${id}`,
        issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/125',
        created_at:'now',updated_at:'now',user:{login:'tally-host[bot]'}}};
    },
    async getComment({comment_id}){
      const found=posted[comment_id-90100-1];
      return {data:{id:comment_id,body:found.body,
        html_url:`https://github.com/kaicreator-mm/tally/issues/125#issuecomment-${comment_id}`,
        issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/125',
        created_at:'now',updated_at:'now',user:{login:'tally-host[bot]'}}};
    }
  }}}};
};
const claimKeyCas=(issueNumber,role='builder')=>
  'tally:v2:claim-admission:'+createHash('sha256')
    .update(`kaicreator-mm/tally#${issueNumber}:${role}:__default__`).digest('hex');
test('two disjoint Builders join one authority+store: serialized POSTs, both RECORDED, both leases released',async()=>{
  await harness(async({store,openFence})=>{
    const specA=builderSpec({issueNumber:301,
      files:['experiments/v2/t06d-join/builder-a.mjs']});
    const specB=builderSpec({issueNumber:302,keyPair:FENCE_KEY_B,
      files:['experiments/v2/t06d-join/builder-b.mjs']});
    const fence=await openFence([specA,specB]);
    // Genuinely interleaved sections: A parks inside its admission read
    // holding lease #301; B acquires lease #302 and parks inside its own
    // admission read (both leases coexist); B then finishes its whole flow
    // (POST + release) before A resumes and POSTs.
    const aInRead=gate(),bInRead=gate(),bMayProceed=gate(),bFinished=gate();
    let posts=0,inFlight=0,maxInFlight=0;
    const sharedPublisher={async publishClaim(){posts++;inFlight++;
      maxInFlight=Math.max(maxInFlight,inFlight);
      await new Promise(resolve=>setTimeout(resolve,5));
      inFlight--;return {commentId:90100+posts};}};
    const sourceA={async readAdmissionFacts(i){aInRead.open();
      await bFinished.promise;return goodFacts(i);},
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const sourceB={async readAdmissionFacts(i){bInRead.open();
      await bMayProceed.promise;return goodFacts(i);},
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const cA=createHostSingleWriterClaimController({store,source:sourceA,
      publisher:sharedPublisher,writerAuthority,fenceClient:fence.clients[0]});
    const cB=createHostSingleWriterClaimController({store,source:sourceB,
      publisher:sharedPublisher,writerAuthority,fenceClient:fence.clients[1]});
    const runA=cA.reserveAndPublish({...req,issueNumber:301,
      dispatchId:'dispatch:301:join:1'});
    await aInRead.promise;
    const runB=cB.reserveAndPublish({...req,issueNumber:302,
      dispatchId:'dispatch:302:join:1'});
    await bInRead.promise;
    // Both disjoint leases coexist on the one shared registry right now.
    assert.equal((await fence.clients[0].observe()).leases.length,2);
    bMayProceed.open();
    const receiptB=await runB;
    bFinished.open();
    const receiptA=await runA;
    assert.equal(receiptA.status,'RECORDED',JSON.stringify(receiptA));
    assert.equal(receiptB.status,'RECORDED',JSON.stringify(receiptB));
    assert.equal(posts,2);
    assert.equal(maxInFlight,1);
    assert.equal((await store.list('tally:v2:claim-admission:')).length,2);
    // Both leases are durably released: the shared registry is empty again.
    assert.equal((await fence.clients[0].observe()).leases.length,0);
  });
});
test('conflicting frozen resource sets: the second acquire is refused, loser denies with zero POST',async()=>{
  await harness(async({store,openFence})=>{
    const specA=builderSpec({issueNumber:301,
      files:['experiments/v2/t06d-join/builder-a.mjs']});
    // Distinct claim key and distinct CAS resources, but a file path INSIDE
    // Builder A's frozen file set: the two sets physically conflict.
    const specC=builderSpec({issueNumber:303,keyPair:FENCE_KEY_B,
      files:['experiments/v2/t06d-join/builder-a.mjs/child.mjs']});
    const fence=await openFence([specA,specC]);
    const aInRead=gate(),aMayFinish=gate();
    let posts=0;
    const sourceA={async readAdmissionFacts(i){aInRead.open();
      await aMayFinish.promise;return goodFacts(i);},
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    let loserReads=0;
    const sourceC={async readAdmissionFacts(i){loserReads++;return goodFacts(i);},
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const publisher={async publishClaim(){posts++;return {commentId:90100+posts};}};
    const cA=createHostSingleWriterClaimController({store,source:sourceA,
      publisher,writerAuthority,fenceClient:fence.clients[0]});
    const cC=createHostSingleWriterClaimController({store,source:sourceC,
      publisher,writerAuthority,fenceClient:fence.clients[1]});
    const runA=cA.reserveAndPublish({...req,issueNumber:301,
      dispatchId:'dispatch:301:conflict:1'});
    await aInRead.promise;
    const loser=await cC.reserveAndPublish({...req,issueNumber:303,
      dispatchId:'dispatch:303:conflict:1'});
    assert.equal(loser.status,'BLOCKED');
    assert.equal(loser.reason,'FENCE_OPERATION_FAILED');
    assert.equal(loser.fenceCode,'HOST_RESOURCE_FENCE_CONFLICT');
    assert.equal(loser.claimPublication,'NOT_ACCEPTED');
    assert.equal(loserReads,0);
    assert.equal(posts,0);
    aMayFinish.open();
    assert.equal((await runA).status,'RECORDED');
    assert.equal(posts,1);
    assert.equal((await store.list('tally:v2:claim-admission:')).length,1);
  });
});
test('publisher refuses without the Host fence port at construction; forged witnesses never reach the POST',async()=>{
  await harness(async({openFence})=>{
    const {octokit,posted}=claimOctokit();
    const reservation={repository:'kaicreator-mm/tally',issueNumber:125,
      issueRef:'https://github.com/kaicreator-mm/tally/issues/125',
      sourceSha,actorRole:'builder',operatorKind:'codex',
      operatorId:'codex:builder1',dispatchId:'dispatch:125:t06d:1',generation:1,
      protectedClaimKey:'kaicreator-mm/tally#125:builder:__default__',
      writerId:writerAuthority.writerId};
    assert.throws(()=>createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority}),
      /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
    // A real port whose lease is simply NOT held: typed NOT_HELD, zero POST.
    const fence=await openFence([builderSpec()]);
    const held=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority,fencePort:fence.ports[0]});
    await assert.rejects(()=>held.publishClaim(reservation),
      error=>error.code==='HOST_RESOURCE_FENCE_NOT_HELD');
    // Caller-crafted proofs: publishClaim accepts NO witness parameter (an
    // extra argument is ignored); the only path is the Host-injected port,
    // and these forged readbacks fail the inlined validator pre-POST.
    const realProof=async context=>({
      sourceKind:'TRUSTED_HOST_RESOURCE_FENCE_READBACK',complete:true,fenceHeld:true,
      repository:'kaicreator-mm/tally',writerId:writerAuthority.writerId,
      protectedClaimKey:context.protectedClaimKey,sourceSha:context.sourceSha,
      frozenResourceSetSha256:fence.ports[0].frozenResourceSetSha256,
      fenceId:'fence-forged:one',fenceRevision:1,
      observedAt:new Date().toISOString()});
    for(const forge of [
      {readHeldFence:async context=>({...await realProof(context),
        frozenResourceSetSha256:'e'.repeat(64)})},
      {readHeldFence:async context=>({...await realProof(context),
        observedAt:new Date(Date.now()-60000).toISOString()})}
    ]){
      const forgedPort={sourceKind:'TRUSTED_HOST_RESOURCE_FENCE_PORT',
        repository:'kaicreator-mm/tally',writerId:writerAuthority.writerId,
        frozenResourceSetSha256:fence.ports[0].frozenResourceSetSha256,...forge};
      const forged=createHostBoundNativeGitHubClaimPublisher({octokit,
        owner:'kaicreator-mm',repo:'tally',writerAuthority,fencePort:forgedPort});
      await assert.rejects(()=>forged.publishClaim(reservation),
        /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
      await assert.rejects(()=>forged.publishClaim(reservation,
        {fenceHeld:true,witness:'caller'}),
        /HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED/);
    }
    assert.equal(posted.length,0);
  });
});
test('real publisher + real port under the held lease; a lease removed underneath denies with zero POST',async()=>{
  await harness(async({store,openFence})=>{
    const {octokit,posted}=claimOctokit();
    const fence=await openFence([builderSpec()]);
    const publisher=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority,fencePort:fence.ports[0]});
    const source={readAdmissionFacts:async i=>goodFacts(i),
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const c=createHostSingleWriterClaimController({store,source,publisher,
      writerAuthority,fenceClient:fence.clients[0]});
    const good=await c.reserveAndPublish(req);
    assert.equal(good.status,'RECORDED',JSON.stringify(good));
    assert.equal(posted.length,1);
    // Lease removed underneath: this admission read itself releases the
    // client's lease before the POST point. The real port readback then
    // proves NOT_HELD; the controller denies with the distinct fence reason,
    // terminally, with no retry — the reservation stays PENDING.
    const fence2=await openFence([builderSpec({issueNumber:126})]);
    const publisher2=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority,fencePort:fence2.ports[0]});
    let reads=0;
    const source2={async readAdmissionFacts(i){reads++;
      await fence2.clients[0].release();
      return goodFacts(i);},
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const c2=createHostSingleWriterClaimController({store,source:source2,
      publisher:publisher2,writerAuthority,fenceClient:fence2.clients[0]});
    const denied=await c2.reserveAndPublish({...req,issueNumber:126,
      dispatchId:'dispatch:126:underneath:1'});
    assert.equal(denied.status,'BLOCKED');
    assert.equal(denied.reason,'FENCE_OPERATION_FAILED');
    assert.equal(denied.fenceCode,'HOST_RESOURCE_FENCE_NOT_HELD');
    assert.equal(reads,1);
    assert.equal(posted.length,1);
    const row=await store.get(claimKeyCas(126));
    assert.equal(row.value.phase,'PENDING_GITHUB_PUBLISH');
  });
});
test('authority restart over the same store and anchor is refused at initialization',async()=>{
  await harness(async({openFence})=>{
    const spec=builderSpec();
    const fence=await openFence([spec]);
    const restarted=createHostResourceFenceAuthority({
      store:fence.store,registryId:fence.authority.registryId,
      anchorPath:fence.anchorPath,writerSid:FENCE_WRITER_SID,
      authorityPrivateKey:FENCE_AUTHORITY_KEY.privateKey,
      authorizedClients:[{
        repository:spec.identity.repository,
        protectedClaimKey:spec.identity.protectedClaimKey,
        writerId:spec.writerId,sourceSha:spec.identity.sourceSha,
        dispatchId:spec.identity.dispatchId,
        frozenResourceSetSha256:buildFrozenResourceSet(spec.identity).frozenResourceSetSha256,
        publicKeyPem:spec.keyPair.publicKey.export({type:'spki',format:'pem'})
      }]});
    await assert.rejects(()=>restarted.initialize(),
      error=>error.code==='HOST_RESOURCE_FENCE_AUTHORITY_RESTART_REFUSED');
  });
});
test('one ambiguous real POST stays PENDING; the same key can only reconcile natively',async()=>{
  await harness(async({store,openFence})=>{
    const {octokit,posted}=claimOctokit({onCreate(){throw Error('ACK_LOST_ONCE');}});
    const fence=await openFence([builderSpec()]);
    const publisher=createHostBoundNativeGitHubClaimPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority,fencePort:fence.ports[0]});
    const source={readAdmissionFacts:async i=>goodFacts(i),
      readPublishedClaim:async(i,id)=>exactReadback(i,id)};
    const c=createHostSingleWriterClaimController({store,source,publisher,
      writerAuthority,fenceClient:fence.clients[0]});
    const first=await c.reserveAndPublish(req);
    assert.equal(first.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
    assert.equal(JSON.stringify(first).includes('ACK_LOST_ONCE'),false);
    assert.equal(posted.length,1);
    const keys=await store.list('tally:v2:claim-admission:');
    assert.equal((await store.get(keys[0])).value.phase,'PENDING_GITHUB_PUBLISH');
    const second=await c.reserveAndPublish(req);
    assert.equal(second.reason,'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
    assert.equal(posted.length,1);
  });
});
test('registry drift is retried inside the same lease; exhaustion denies without any write',async()=>{
  const driftFacts=i=>({...goodFacts(i),sourceQualified:false,complete:false,
    reason:'HOST_RESERVATION_REGISTRY_DRIFT'});
  await harness(async({store,openFence})=>{
    const fence=await openFence([builderSpec()]);
    let reads=0;
    const p=ports(fence,{source:{async readAdmissionFacts(i){reads++;
      // Drift on the first two reads only; the third read is clean.
      return reads<3?driftFacts(i):goodFacts(i);}}});
    const receipt=await controller(store,p).reserveAndPublish(req);
    assert.equal(receipt.status,'RECORDED',JSON.stringify(receipt));
    assert.equal(reads,3);
    assert.equal(p.counters.writes,1);
    assert.equal((await fence.clients[0].observe()).leases.length,0);
  });
  await harness(async({store,openFence})=>{
    const fence=await openFence([builderSpec()]);
    let reads=0;
    const p=ports(fence,{source:{async readAdmissionFacts(i){reads++;
      return driftFacts(i);}}});
    const receipt=await controller(store,p).reserveAndPublish(req);
    assert.equal(receipt.status,'BLOCKED');
    assert.equal(receipt.reason,'ADMISSION_REGISTRY_DRIFT_EXHAUSTED');
    assert.equal(reads,4);
    assert.equal(p.counters.writes,0);
    assert.equal((await store.list('tally:v2:claim-admission:')).length,0);
    assert.equal((await fence.clients[0].observe()).leases.length,0);
    // A drift read followed by a NON-drift denial stops immediately with the
    // original deny semantics, never the exhausted reason.
    let reads2=0;
    const p2=ports(await openFence([builderSpec()]),{source:{
      async readAdmissionFacts(i){reads2++;
        return reads2===1?driftFacts(i):{...goodFacts(i),
          activeIncompatibleClaimCount:1};}}});
    const other=await controller(store,p2).reserveAndPublish(req);
    assert.equal(other.reason,'SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
    assert.equal(reads2,2);
    assert.equal(p2.counters.writes,0);
  });
});
