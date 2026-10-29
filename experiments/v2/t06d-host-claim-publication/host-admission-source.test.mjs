import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {stringify} from 'yaml';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createHostAdmissionSourceFromClient,createGitHubHostAdmissionSource} from './host-admission-source.mjs';
import {createHostBoundNativeGitHubClaimPublisher} from '../../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';
import {createHostSingleWriterClaimController} from './single-writer.mjs';

const owner='kaicreator-mm',repo='tally',repository=owner+'/'+repo;
const issueNumber=166,sourceSha='a'.repeat(40);
const issueRef=`https://github.com/${repository}/issues/${issueNumber}`;
const apiIssue=`https://api.github.com/repos/${repository}/issues/${issueNumber}`;
const date='2026-10-30T00:00:00Z',commentId=901;
const writerAuthority=Object.freeze({mode:'SINGLE_WRITER_ADMISSION',
  exclusiveHostControl:true,writerId:'tally-host:one',repository,
  githubWriterLogin:'tally-host[bot]'});
const taskAuthority=Object.freeze({issueNumber,sourceSha,actorRole:'builder',
  dispatchId:'dispatch:166:t06j:1',operatorKind:'codex',operatorId:'codex:builder1'});
const request=Object.freeze({repository,issueNumber,issueRef,sourceSha,
  actorRole:'builder',dispatchId:taskAuthority.dispatchId,
  operatorKind:'codex',operatorId:taskAuthority.operatorId,generation:1,
  protectedClaimKey:`${repository}#${issueNumber}:builder:__default__`});
const fence=String.fromCharCode(96).repeat(3);
const eventData=(changes={})=>({schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',
  actor_role:'builder',operator_kind:'codex',operator_id:taskAuthority.operatorId,
  task:'#166',issue:'#166',sha:sourceSha,dispatch_id:taskAuthority.dispatchId,
  dispatch_state:'CLAIMED',protected_claim_key:request.protectedClaimKey,
  admission_generation:1,...changes});
const bodyFor=data=>'<!-- ads:github -->\n'+fence+'yaml\n'+
  stringify({schema:'ads/github/v3',kind:'event',data}).trimEnd()+'\n'+fence+'\n';
const claim=(changes={})=>({id:commentId,body:bodyFor(eventData()),
  html_url:issueRef+'#issuecomment-'+commentId,issue_url:apiIssue,
  created_at:date,updated_at:date,user:{login:writerAuthority.githubWriterLogin},
  ...changes});

function github({blocked=false,ready=true,type='type:task',issueState='open',
  commitSha=sourceSha,comments=[],detail=null,drift=false,failComments=false,
  invalidCommentPage=false,createEdited=false}={}){
  const state={posts:0,issueReads:0,comments:[...comments]};
  const octokit={rest:{issues:{
    async get(){state.issueReads++;return {data:{id:1660,number:issueNumber,
      html_url:issueRef,state:issueState,
      updated_at:drift&&state.issueReads>=6?date.replace('00:00','00:01'):date,
      labels:[{name:type},{name:ready?'state:ready':'state:blocked'}]}};},
    async listComments(){throw Error('PAGINATION_REQUIRED');},
    async createComment({body}){
      state.posts++;
      const created=claim({id:commentId,body});
      state.comments.push(created);
      return {data:createEdited?{...created,body:created.body+'edited'}:created};
    },
    async getComment({comment_id:id}){
      if(detail)return {data:detail};
      const found=state.comments.find(row=>row.id===id);
      if(!found)throw Object.assign(Error('NOT_FOUND'),{status:404});
      return {data:found};
    }
  },repos:{async getCommit(){return {data:{sha:commitSha}};}}}};
  octokit.paginate={async *iterator(route){
    if(typeof route==='string'){
      yield {data:blocked?[{id:99,number:99,
        html_url:'https://github.com/kaicreator-mm/tally/issues/99',state:'open'}]:[]};
      return;
    }
    if(failComments)throw Object.assign(Error('PRIVATE_TOKEN_CANARY'),{status:403});
    if(invalidCommentPage){yield {data:null};return;}
    yield {data:[...state.comments]};
  }};
  return {octokit,state};
}
function memoryStore({initial=[],getError=false,listError=false,drift=false}={}){
  const rows=new Map();let revision=0,writes=0,listCalls=0;
  for(const value of initial){
    const key='tally:v2:claim-admission:test-'+rows.size;
    rows.set(key,{value:structuredClone(value),revision:++revision});
  }
  return {rows,get writes(){return writes;},
    async list(prefix){
      listCalls++;
      if(listError)throw Error('REGISTRY_PRIVATE_ERROR');
      if(drift&&listCalls===2)return [...rows.keys()].filter(key=>key.startsWith(prefix)).concat(prefix+'new');
      return [...rows.keys()].filter(key=>key.startsWith(prefix)).sort();
    },
    async get(key){if(getError)throw Error('REGISTRY_PRIVATE_ERROR');return rows.get(key)??null;},
    async put(key,value,{mustBeAbsent,mustMatchRevision}={}){
      writes++;
      const previous=rows.get(key);
      if(mustBeAbsent&&previous)throw Error('CAS_COLLISION');
      if(mustMatchRevision!==undefined&&previous?.revision!==mustMatchRevision)
        throw Error('CAS_STALE');
      const next=++revision;rows.set(key,{value:structuredClone(value),revision:next});
      return {revision:next};
    }
  };
}
function adapter({githubOptions={},store=memoryStore()}={}){
  const gh=github(githubOptions);
  const source=createHostAdmissionSourceFromClient({octokit:gh.octokit,store,
    owner,repo,writerAuthority,taskAuthority});
  return {source,gh,store};
}
function registryValue(changes={}){
  return {phase:'PENDING_GITHUB_PUBLISH',generation:1,
    protectedClaimKey:'kaicreator-mm/tally#167:builder:__default__',
    repository,issueRef:'https://github.com/kaicreator-mm/tally/issues/167',
    sourceSha:'b'.repeat(40),dispatchId:'dispatch:167:other',actorRole:'builder',
    operatorKind:'codex',operatorId:'codex:other',writerId:'tally-host:one',...changes};
}
function noPermit(x){
  assert.equal(x.complete,false,JSON.stringify(x));assert.equal(x.sourceQualified,false);
  assert.equal(x.dispatchSourceBound,false);assert.equal(x.publisherSourceBound,false);
  for(const key of ['authorized','admitted','taskPass','releasePass'])assert.equal(x[key],false);
}

test('valid current native facts + empty authoritative registry yields complete Host facts, zero writes',async()=>{
  const h=adapter(),x=await h.source.readAdmissionFacts(request);
  assert.equal(x.sourceKind,'NATIVE_GITHUB_CURRENT_ADMISSION');
  assert.equal(x.complete,true);assert.equal(x.sourceQualified,true);
  assert.equal(x.issueRef,issueRef);assert.equal(x.sourceSha,sourceSha);
  assert.equal(x.dependencyState,'CLEAR');assert.equal(x.activeIncompatibleClaimCount,0);
  assert.equal(x.dispatchSourceBound,true);assert.equal(x.publisherSourceBound,true);
  for(const key of ['authorized','admitted','taskPass','releasePass'])assert.equal(x[key],false);
  assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
});

test('wrong repository, Issue, SHA, role and forged dispatcher/writer fields deny before writes',async()=>{
  const variants=[{repository:'evil/repo'},{issueNumber:167},{sourceSha:'b'.repeat(40)},
    {actorRole:'reviewer'},{dispatchId:'dispatch:forged'},{operatorId:'codex:forged'},
    {operatorKind:'human'}];
  for(const change of variants){
    const h=adapter(),x=await h.source.readAdmissionFacts({...request,...change});
    noPermit(x);assert.equal(x.reason,'HOST_TASK_OR_WRITER_REQUEST_NOT_BOUND');
    assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
});

test('caller-forged authority fields are ignored; only Host configuration binds dispatch and writer',async()=>{
  const h=adapter(),x=await h.source.readAdmissionFacts({...request,
    writerId:'caller-controlled',githubWriterLogin:'attacker',dispatchSourceBound:false,
    publisherSourceBound:false,activeIncompatibleClaimCount:999,authorized:true,
    admitted:true,dependenciesComplete:true});
  assert.equal(x.complete,true);assert.equal(x.activeIncompatibleClaimCount,0);
  assert.equal(x.dispatchSourceBound,true);assert.equal(x.publisherSourceBound,true);
  assert.equal(x.authorized,false);assert.equal(x.admitted,false);
  assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
});

test('blocked dependency, unready/non-task/closed Issue and stale source SHA deny with zero writes',async()=>{
  for(const githubOptions of [{blocked:true},{ready:false},{type:'type:planning'},
    {issueState:'closed'},{commitSha:'b'.repeat(40)}]){
    const h=adapter({githubOptions}),x=await h.source.readAdmissionFacts(request);
    noPermit(x);assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
});

test('malformed, edited, duplicate, cross-task and unqualified Claim events fail closed',async()=>{
  const variants=[
    [claim({body:'<!-- ads:github -->\nnot-yaml'})],
    [claim({updated_at:'2026-10-30T00:00:01Z'})],
    [claim(),claim({id:902,html_url:issueRef+'#issuecomment-902'})],
    [claim({body:bodyFor(eventData({task:'#167',issue:'#167'}))})],
    [claim({body:bodyFor(eventData({dispatch_id:'dispatch:other'}))})],
    [claim({body:bodyFor(eventData({sha:'b'.repeat(40)}))})]
  ];
  for(const comments of variants){
    const h=adapter({githubOptions:{comments}}),x=await h.source.readAdmissionFacts(request);
    noPermit(x);assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
});

test('Issue drift, incomplete pagination and API errors never preserve partial facts or secrets',async()=>{
  for(const githubOptions of [{drift:true},{invalidCommentPage:true},{failComments:true}]){
    const h=adapter({githubOptions}),x=await h.source.readAdmissionFacts(request);
    noPermit(x);assert.equal(JSON.stringify(x).includes('PRIVATE_TOKEN_CANARY'),false);
    assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
});

test('active incompatible registry entry, read failure, malformed/conflicting rows and drift deny',async()=>{
  const stores=[memoryStore({initial:[registryValue()]}),memoryStore({initial:[registryValue()],getError:true}),
    memoryStore({listError:true}),memoryStore({initial:[registryValue({phase:'UNKNOWN'})]}),
    memoryStore({drift:true})];
  for(const store of stores){
    const h=adapter({store}),x=await h.source.readAdmissionFacts(request);
    noPermit(x);assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
  const conflict=memoryStore({initial:[registryValue(),registryValue()]});
  const h=adapter({store:conflict}),x=await h.source.readAdmissionFacts(request);
  noPermit(x);assert.equal(x.reason,'HOST_RESERVATION_REGISTRY_CONFLICT');
  assert.equal(h.store.writes,0);
});

test('exact native published Claim readback validates Host reservation and returns no permit',async()=>{
  const h=adapter({githubOptions:{comments:[claim()]}});
  const x=await h.source.readPublishedClaim(request,commentId);
  assert.equal(x.sourceKind,'NATIVE_GITHUB_EXACT_READBACK');assert.equal(x.complete,true);
  assert.equal(x.edited,false);assert.equal(x.commentId,commentId);
  assert.equal(x.dispatchId,request.dispatchId);assert.equal(x.generation,1);
  for(const key of ['claimAccepted','authorized','admitted','taskPass','releasePass'])
    assert.equal(x[key],false);
  assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
});

test('edited/substituted/foreign comment, duplicate event and generation drift reject readback',async()=>{
  const variants=[{detail:claim({body:claim().body+'changed'})},
    {detail:claim({updated_at:'2026-10-30T00:00:01Z'})},
    {detail:claim({user:{login:'attacker'}})},
    {detail:claim({issue_url:apiIssue+'-foreign'})},
    {detail:claim({id:902})},{comments:[]},
    {comments:[claim(),claim({id:902,html_url:issueRef+'#issuecomment-902'})]},
    {comments:[claim({body:bodyFor(eventData({admission_generation:2}))})]}];
  for(const options of variants){
    const h=adapter({githubOptions:options});
    await assert.rejects(()=>h.source.readPublishedClaim(request,commentId),
      /NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED/);
    assert.equal(h.store.writes,0);assert.equal(h.gh.state.posts,0);
  }
  const h=adapter({githubOptions:{comments:[claim()]}});
  await assert.rejects(()=>h.source.readPublishedClaim({...request,generation:2},commentId));
  assert.equal(h.store.writes,0);
});

test('existing controller composes with real adapter/publisher; denials create no reservation or POST',async()=>{
  const h=adapter();
  const publisher=createHostBoundNativeGitHubClaimPublisher({octokit:h.gh.octokit,
    owner,repo,writerAuthority});
  const controller=createHostSingleWriterClaimController({store:h.store,source:h.source,
    publisher,writerAuthority});
  const receipt=await controller.reserveAndPublish({owner,repo,issueNumber,sourceSha,
    actorRole:'builder',dispatchId:taskAuthority.dispatchId,operatorKind:'codex',
    operatorId:taskAuthority.operatorId,expectedGeneration:0});
  assert.equal(receipt.status,'RECORDED');assert.equal(h.gh.state.posts,1);
  assert.equal(h.store.rows.size,1);
  assert.equal([...h.store.rows.values()][0].value.phase,'PUBLISHED_VERIFIED');
  assert.equal(receipt.admitted,false);assert.equal(receipt.authorized,false);
  const blocked=adapter({githubOptions:{blocked:true}});
  const deniedController=createHostSingleWriterClaimController({store:blocked.store,
    source:blocked.source,publisher:createHostBoundNativeGitHubClaimPublisher({
      octokit:blocked.gh.octokit,owner,repo,writerAuthority}),writerAuthority});
  const denial=await deniedController.reserveAndPublish({owner,repo,issueNumber,sourceSha,
    actorRole:'builder',dispatchId:taskAuthority.dispatchId,operatorKind:'codex',
    operatorId:taskAuthority.operatorId,expectedGeneration:0});
  assert.equal(denial.status,'BLOCKED');assert.equal(blocked.store.writes,0);
  assert.equal(blocked.gh.state.posts,0);
});

test('real public DH SQLite CAS registry: positive and incompatible reads add no writes',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-t06j-adapter-'));
  const path=join(dir,'registry.sqlite');
  let store=await openSqliteCasStore({path,busyTimeoutMs:1000});
  try{
    const h=adapter({store});
    const good=await h.source.readAdmissionFacts(request);
    assert.equal(good.complete,true);assert.equal((await store.list('tally:v2:claim-admission:')).length,0);
    await store.put('tally:v2:claim-admission:test-other',registryValue(),{mustBeAbsent:true});
    const before=(await store.list('tally:v2:claim-admission:')).length;
    const denied=await h.source.readAdmissionFacts(request);
    noPermit(denied);assert.equal(denied.reason,'INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS');
    assert.equal((await store.list('tally:v2:claim-admission:')).length,before);
  }finally{await store.close();await rm(dir,{recursive:true,force:true});}
});

test('Host factory rejects missing credential, untrusted writer and incomplete ports',()=>{
  assert.throws(()=>createGitHubHostAdmissionSource({token:'',owner,repo,
    writerAuthority,taskAuthority}),/TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED/);
  assert.throws(()=>createHostAdmissionSourceFromClient({octokit:github().octokit,
    store:memoryStore(),owner,repo,taskAuthority,writerAuthority:{
      ...writerAuthority,exclusiveHostControl:false}}),/TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED/);
  assert.throws(()=>createHostAdmissionSourceFromClient({octokit:github().octokit,
    store:{get:async()=>null},owner,repo,writerAuthority,taskAuthority}),
    /TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED/);
});
