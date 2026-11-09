/**
 * T06k: real Tally C + Host controller *composition* test, against an
 * in-process GitHub HTTP surrogate and deterministic CAS-store surrogate.
 *
 * This intentionally DOES NOT attest a physical Host deployment, actual
 * SQLite CAS, trusted dispatch source, accepted ADS Claim, GitHub workflow
 * transition, Kernel Task permit, or external Agent execution. Prior T06d
 * separately tests the real DomainHarness SQLite CAS primitive.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash,generateKeyPairSync,randomBytes} from 'node:crypto';
import {readNativeHostClaimPreflightFromClient} from
  '../src/task-claim-host-preflight.mjs';
import {createHostBoundNativeGitHubClaimPublisher} from
  '../src/task-claim-github-publisher.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from
  '../src/task-claim-host-readback-port.mjs';
import {createHostSingleWriterClaimController} from
  '../../../experiments/v2/t06d-host-claim-publication/single-writer.mjs';
import {HOST_RESOURCE_FENCE_SCHEMA,buildFrozenResourceSet,computeDispatchId,
  createHostResourceFenceAuthority,createHostResourceFenceClient,
  createHostResourceFencePort} from
  '../../../experiments/v2/t06d-host-claim-publication/host-resource-fence.mjs';

const owner='kaicreator-mm',repo='tally',issueNumber=125,sourceSha='a'.repeat(40);
const writerAuthority=Object.freeze({
  mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
  writerId:'tally-host:exclusive',repository:owner+'/'+repo,
  githubWriterLogin:'tally-host[bot]'
});
const request=Object.freeze({owner,repo,issueNumber,sourceSha,
  actorRole:'builder',operatorKind:'codex',operatorId:'codex:builder1',
  dispatchId:'dispatch:125:composition1',expectedGeneration:0});
const issueRef='https://github.com/'+owner+'/'+repo+'/issues/'+issueNumber;
const apiIssue='https://api.github.com/repos/'+owner+'/'+repo+'/issues/'+issueNumber;
const immutableDate='2026-10-28T12:30:00Z';

function github({blocked=false,ready=true,lostAck=false,edited=false}={}){
  const state={posts:0,comments:[],detailGets:0,issueReads:0};
  const issue=()=>({
    id:525,number:issueNumber,html_url:issueRef,
    state:'open',updated_at:immutableDate,
    labels:[{name:ready?'state:ready':'state:blocked'},{name:'type:task'}]
  });
  const octokit={
    rest:{
      issues:{
        async get(){state.issueReads++;return {data:issue()};},
        async listComments(){throw new Error('MUST_USE_PAGINATION');},
        async createComment({issue_number,body}){
          assert.equal(issue_number,issueNumber);
          state.posts++;
          const c={id:900+state.posts,html_url:issueRef+
            '#issuecomment-'+(900+state.posts),issue_url:apiIssue,
            body,created_at:immutableDate,updated_at:immutableDate,
            user:{login:'tally-host[bot]'}};
          state.comments.push(c);
          if(lostAck)throw Error('SECRET_TRANSPORT_ACK_LOST');
          return {data:c};
        },
        async getComment({comment_id}){
          state.detailGets++;
          const comment=state.comments.find(x=>x.id===comment_id);
          if(!comment)throw Object.assign(Error('not found'),{status:404});
          return {data:edited?{...comment,body:comment.body+'-modified'}:comment};
        }
      },
      repos:{async getCommit(){return {data:{sha:sourceSha}};}}
    },
    paginate:{
      async *iterator(route){
        if(typeof route==='string'&&route.includes('blocked_by')){
          yield {data:blocked?[{id:12,number:12,
            html_url:'https://github.com/kaicreator-mm/tally/issues/12',
            state:'open'}]:[]};
          return;
        }
        if(route===octokit.rest.issues.listComments){
          yield {data:[...state.comments]};
          return;
        }
        throw Error('UNRECOGNIZED_NATIVE_ROUTE');
      }
    }
  };
  return {octokit,state};
}
function store(){
  const rows=new Map();
  return {
    rows,
    async list(prefix=''){return [...rows.keys()].filter(key=>key.startsWith(prefix)).sort();},
    async get(key){return rows.get(key)??null;},
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      const previous=rows.get(key);
      if(mustBeAbsent&&previous)throw Error('CAS_COLLISION');
      if(mustMatchRevision!==undefined&&
         previous?.revision!==mustMatchRevision)
        throw Error('CAS_STALE_REVISION');
      const revision=previous?previous.revision+1:0;
      rows.set(key,{value:structuredClone(value),revision});
      return {revision};
    },
    async remove(key,{mustBeAbsent,mustMatchRevision}={}){
      const previous=rows.get(key);
      if(mustBeAbsent&&!previous)throw Error('CAS_MISSING');
      if(mustMatchRevision!==undefined&&
         previous?.revision!==mustMatchRevision)
        throw Error('CAS_STALE_REVISION');
      rows.delete(key);
    }
  };
}
// Minimal REAL shared-Host fence fixture (mirrors the t06d library tests):
// disposable store surrogate, disposable RSA keys, authority, per-Task
// client and port bound to this composition's #125 Task identity.
const FENCE_WRITER_SID='S-1-5-21-2839165489-1832998833-1574237796-1004';
const FENCE_AUTHORITY_KEY=generateKeyPairSync('rsa',{modulusLength:2048});
const FENCE_CLIENT_KEY=generateKeyPairSync('rsa',{modulusLength:2048});
let fenceSeq=0;
async function openFence(){
  const dir=await mkdtemp(join(tmpdir(),'tally-t06k-fence-'));
  const seq=++fenceSeq;
  const fenceStore=store();
  const anchorPath=join(dir,`anchor-${seq}.ndjson`);
  const repository=owner+'/'+repo;
  const protectedClaimKey=`${repository}#${issueNumber}:builder:__default__`;
  const issueRef='https://github.com/'+repository+'/issues/'+issueNumber;
  const reviewRef=`https://github.com/${repository}/pull/376#pullrequestreview-1000000001`;
  const ownerApprovalRef=`https://github.com/${repository}/issues/${issueNumber}#issuecomment-1000000002`;
  const files=['packages/tally-github-native-c/test/task-claim-composition.test.mjs'];
  const writeSetSha256=createHash('sha256')
    .update(JSON.stringify([...files].sort())).digest('hex');
  const identity={schema:HOST_RESOURCE_FENCE_SCHEMA,repository,issueRef,
    protectedClaimKey,generation:1,sourceSha,
    dispatchId:computeDispatchId({repository,issueNumber,sourceSha,
      taskPackBlobSha:'b'.repeat(40),reviewRef,ownerApprovalRef,writeSetSha256}),
    taskPackRef:`https://github.com/${repository}/blob/${sourceSha}/`+
      'docs/implementation/v2/HOST_SHARED_WRITER_FENCE_TASK_PACK_R1.json',
    taskPackBlobSha:'b'.repeat(40),reviewRef,ownerApprovalRef,writeSetSha256,
    exclusiveResources:[
      `claim-key:${protectedClaimKey}`,
      `cas:tally:v2:claim-admission:${createHash('sha256').update(protectedClaimKey).digest('hex')}`,
      `cas:tally:v2:task-writeset:${issueNumber}`,
      `github-issue-comment:${repository}#${issueNumber}`,
      ...files.map(file=>`file:${file}`)
    ]};
  const built=buildFrozenResourceSet(identity);
  const authority=createHostResourceFenceAuthority({store:fenceStore,
    registryId:'fence-registry-t06k-wiring-1',anchorPath,writerSid:FENCE_WRITER_SID,
    authorityPrivateKey:FENCE_AUTHORITY_KEY.privateKey,
    authorizedClients:[{repository,protectedClaimKey,
      writerId:writerAuthority.writerId,sourceSha,dispatchId:identity.dispatchId,
      frozenResourceSetSha256:built.frozenResourceSetSha256,
      publicKeyPem:FENCE_CLIENT_KEY.publicKey.export({type:'spki',format:'pem'})}]});
  await authority.initialize();
  const client=createHostResourceFenceClient({
    privateKey:FENCE_CLIENT_KEY.privateKey,authorityPublicKeyPem:authority.publicKeyPem,
    registryId:authority.registryId,writerId:writerAuthority.writerId,identity,
    nonce:randomBytes(32).toString('hex'),send:handle=>authority.handle(handle)});
  const port=createHostResourceFencePort({store:fenceStore,
    registryId:authority.registryId,anchorPath,
    authorityPublicKeyPem:authority.publicKeyPem,writerSid:FENCE_WRITER_SID,
    repository,writerId:writerAuthority.writerId,protectedClaimKey,sourceSha,
    frozenResourceSetSha256:built.frozenResourceSetSha256,
    expectedHead:()=>authority.anchorHead,withReadLock:task=>authority.withReadLock(task)});
  return {dir,client,port};
}
async function controller(opts={}){
  const {octokit,state}=github(opts),db=store(),fence=await openFence();
  const sourceReadback=createHostNativeGitHubClaimReadbackPortFromClient({
    octokit,owner,repo,writerLogin:'tally-host[bot]'
  });
  const source={
    async readAdmissionFacts(i){
      const observed=await readNativeHostClaimPreflightFromClient({
        octokit,owner,repo,issueNumber:i.issueNumber,
        expectedSourceSha:i.sourceSha
      });
      // TEST-ONLY owner simulation. A deployed Host MUST independently
      // validate current dispatch, physical exclusive writer and permissions.
      // It MUST NOT convert an advisory C preflight into admission merely by
      // changing DTO fields as this test harness does.
      if(observed.status!=='NATIVE_SOURCE_PREFLIGHT_READY')return observed;
      return {
        ...observed,sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',
        complete:true,activeIncompatibleClaimCount:0,
        dispatchSourceBound:true,publisherSourceBound:true
      };
    },
    readPublishedClaim:sourceReadback.readPublishedClaim
  };
  const publisher=createHostBoundNativeGitHubClaimPublisher({
    octokit,owner,repo,writerAuthority,fencePort:fence.port
  });
  const runner=createHostSingleWriterClaimController({
    store:db,source,publisher,writerAuthority,fenceClient:fence.client
  });
  return {state,db,source,publisher,fence,runner,
    async cleanup(){await rm(fence.dir,{recursive:true,force:true});}};
}
test('real C modules compose through existing Host CAS contract; source-only receipt',async()=>{
  const h=await controller();
  try{
    const res=await h.runner.reserveAndPublish(request);
    assert.equal(res.status,'RECORDED');
    assert.equal(res.reason,'SERIALIZED_GITHUB_PUBLISH_AND_READBACK_CONFIRMED');
    assert.equal(res.claimPublication,'VERIFIED_PENDING_KERNEL_ADMISSION');
    for(const field of ['authorized','admitted','taskPass','releasePass'])
      assert.equal(res[field],false);
    assert.equal(h.state.posts,1);
    assert.ok(h.state.detailGets>=2);
    assert.equal(h.db.rows.size,1);
    const row=[...h.db.rows.values()][0].value;
    assert.equal(row.phase,'PUBLISHED_VERIFIED');
    assert.equal(row.commentId,901);
    assert.equal(row.sourceSha,sourceSha);
    assert.equal(row.protectedClaimKey,
      'kaicreator-mm/tally#125:builder:__default__');
    // The POST went through the real port readback and the lease was
    // durably released after the final CAS.
    assert.equal((await h.fence.client.observe()).leases.length,0);
    assert.equal((await h.runner.reserveAndPublish(request)).reason,
      'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
    assert.equal(h.state.posts,1);
  }finally{await h.cleanup();}
});
test('simultaneous incompatible builders cannot post two canonical claims',async()=>{
  const h=await controller();
  try{
    const [a,b]=await Promise.all([
      h.runner.reserveAndPublish(request),
      h.runner.reserveAndPublish({...request,
        operatorId:'codex:builder2',dispatchId:'dispatch:125:other'})
    ]);
    assert.equal([a,b].filter(x=>x.status==='RECORDED').length,1);
    assert.ok([a.reason,b.reason].some(r=>
      ['CONCURRENT_CLAIM_RESERVATION_LOST','INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS',
        'FENCE_OPERATION_FAILED'].includes(r)));
    assert.equal(h.state.posts,1);
    assert.equal(h.db.rows.size,1);
  }finally{await h.cleanup();}
});
test('unready Issue and live native blocked_by deny before CAS or POST',async()=>{
  for(const opts of [{blocked:true},{ready:false}]){
    const h=await controller(opts);
    try{
      const receipt=await h.runner.reserveAndPublish(request);
      assert.equal(receipt.status,'BLOCKED');
      assert.equal(receipt.reason,'SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
      assert.equal(h.state.posts,0);
      assert.equal(h.db.rows.size,0);
      assert.equal((await h.fence.client.observe()).leases.length,0);
    }finally{await h.cleanup();}
  }
});
test('lost ACK preserves PENDING reservation, blocks same-dispatch retry',async()=>{
  const h=await controller({lostAck:true});
  try{
    const receipt=await h.runner.reserveAndPublish(request);
    assert.equal(receipt.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
    assert.equal(h.state.posts,1);
    assert.equal(h.db.rows.size,1);
    assert.equal([...h.db.rows.values()][0].value.phase,
      'PENDING_GITHUB_PUBLISH');
    const retry=await h.runner.reserveAndPublish(request);
    assert.equal(retry.reason,'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
    assert.equal(h.state.posts,1);
    assert.equal(JSON.stringify(receipt).includes('SECRET_TRANSPORT_ACK_LOST'),false);
  }finally{await h.cleanup();}
});
test('tampered exact numeric readback cannot produce accepted Host receipt',async()=>{
  const h=await controller({edited:true});
  try{
    const x=await h.runner.reserveAndPublish(request);
    assert.equal(x.status,'BLOCKED');
    assert.equal(x.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
    assert.equal(h.db.rows.size,1);
    assert.equal([...h.db.rows.values()][0].value.phase,
      'PENDING_GITHUB_PUBLISH');
    assert.equal(h.state.posts,1);
    assert.equal(x.admitted,false);
  }finally{await h.cleanup();}
});
