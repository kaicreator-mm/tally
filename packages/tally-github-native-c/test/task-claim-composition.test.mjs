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
import {readNativeHostClaimPreflightFromClient} from
  '../src/task-claim-host-preflight.mjs';
import {createHostBoundNativeGitHubClaimPublisher} from
  '../src/task-claim-github-publisher.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from
  '../src/task-claim-host-readback-port.mjs';
import {createHostSingleWriterClaimController} from
  '../../../experiments/v2/t06d-host-claim-publication/single-writer.mjs';

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
  const rows=new Map();let rev=0;
  return {
    rows,
    async get(key){return rows.get(key)??null;},
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      const previous=rows.get(key);
      if(mustBeAbsent&&previous)throw Error('CAS_COLLISION');
      if(mustMatchRevision!==undefined&&
         previous?.revision!==mustMatchRevision)
        throw Error('CAS_STALE_REVISION');
      const revision=++rev;
      rows.set(key,{value:structuredClone(value),revision});
      return {revision};
    }
  };
}
function controller(opts={}){
  const {octokit,state}=github(opts),db=store();
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
    octokit,owner,repo,writerAuthority
  });
  return {
    state,db,source,publisher,
    runner:createHostSingleWriterClaimController({
      store:db,source,publisher,writerAuthority
    })
  };
}
test('real C modules compose through existing Host CAS contract; source-only receipt',async()=>{
  const h=controller();
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
  assert.equal((await h.runner.reserveAndPublish(request)).reason,
    'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION');
  assert.equal(h.state.posts,1);
});
test('simultaneous incompatible builders cannot post two canonical claims',async()=>{
  const h=controller();
  const [a,b]=await Promise.all([
    h.runner.reserveAndPublish(request),
    h.runner.reserveAndPublish({...request,
      operatorId:'codex:builder2',dispatchId:'dispatch:125:other'})
  ]);
  assert.equal([a,b].filter(x=>x.status==='RECORDED').length,1);
  assert.equal(h.state.posts,1);
  assert.equal(h.db.rows.size,1);
});
test('unready Issue and live native blocked_by deny before CAS or POST',async()=>{
  for(const opts of [{blocked:true},{ready:false}]){
    const h=controller(opts);
    const receipt=await h.runner.reserveAndPublish(request);
    assert.equal(receipt.status,'BLOCKED');
    assert.equal(receipt.reason,'SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
    assert.equal(h.state.posts,0);
    assert.equal(h.db.rows.size,0);
  }
});
test('lost ACK preserves PENDING reservation, blocks same-dispatch retry',async()=>{
  const h=controller({lostAck:true});
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
});
test('tampered exact numeric readback cannot produce accepted Host receipt',async()=>{
  const h=controller({edited:true});
  const x=await h.runner.reserveAndPublish(request);
  assert.equal(x.status,'BLOCKED');
  assert.equal(x.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
  assert.equal(h.db.rows.size,1);
  assert.equal([...h.db.rows.values()][0].value.phase,
    'PENDING_GITHUB_PUBLISH');
  assert.equal(h.state.posts,1);
  assert.equal(x.admitted,false);
});
