import test from 'node:test';
import assert from 'node:assert/strict';
import {createVersionTaskDagHost,VERSION_TASK_DAG_HOST_IDS}
  from './version-task-dag-host.mjs';

const repository='kaicreator-mm/tally';
const owner='kaicreator-mm';
const repo='tally';
const versionId='version/v2';
const versionSha='7ebb1cdd0a79fc82cc938e35f852a80dd8e15332';
const packBlobSha='d8b1c8dcfe22fdc27a2ffe151341ac3d3728329d';
const packCommitSha='b3ad8ae7833028999a9c27cac7042037bca6d961';
const packPath='apps/tally-web/src/features/dag/version-v2-task-pack.candidate.json';
const taskPackRef='https://github.com/'+repository+'/blob/'+packCommitSha+'/'+packPath;
const issueUrl=n=>'https://github.com/'+repository+'/issues/'+n;
const digest='a'.repeat(64);
const ROSTER=[284,312,219];

function candidatePack(edges){
  return {
    schema:'tally/version-task-pack/v1',status:'CANDIDATE_NOT_FROZEN',
    repository,versionId,versionBaseSha:versionSha,
    observedAt:'2026-11-09T11:09:59.214Z',
    source:{kind:'GITHUB_NATIVE_ISSUES',query:'state=all&labels=type:task',
      taskIssueCount:3,claimCommentPages:1},
    taskIssues:ROSTER.map(n=>({issueNumber:n,title:'Task '+n,
      issueState:'open',labels:['type:task'],updatedAt:'2026-11-09T11:00:00Z'})),
    edges,
    claimSubjects:ROSTER.map(n=>({issueNumber:n,sourceSha:versionSha,
      actorRole:'builder',candidateStatus:'NO_CLAIM_EVENT_EXPECTED',commentId:null,
      commentRef:null,dispatchId:null,protectedClaimKey:
        repository+'#'+n+':builder:__default__'}))
  };
}
const plannedEdge={from:284,to:312,kind:'PLANNED',evidenceRef:issueUrl(312),
  dependencyRef:issueUrl(284)};
const nativeEdge={...plannedEdge,kind:'NATIVE'};
const nativeBlockers={312:[{id:284000,number:284,state:'open',
  html_url:issueUrl(284)}]};
const reviewWitness={status:'INDEPENDENTLY_VERIFIED',taskPackBlobSha:packBlobSha,
  sourceSha:versionSha,reviewedHeadSha:versionSha,reviewer:'independent-reviewer',
  author:'web-builder',reviewRef:'https://github.com/'+repository+'/pull/384',
  verifiedAt:'2026-11-09T12:00:00.000Z'};
const approvalWitness={status:'OWNER_APPROVED',taskPackBlobSha:packBlobSha,
  sourceSha:versionSha,owner,approvalRef:'https://github.com/'+repository+
  '/issues/284#issuecomment-6739635285',approvedAt:'2026-11-09T12:01:00.000Z'};

function memoryStore(){
  const rows=new Map();
  const store={
    async get(key){const row=rows.get(key);return row?structuredClone(row):null;},
    async put(key,value,{mustBeAbsent}={}){
      if(mustBeAbsent&&rows.has(key))throw new Error('CAS_EXISTS');
      rows.set(key,{revision:1,value:structuredClone(value)});
      return {revision:1};
    },
    async close(){store.closed=true;},
    closed:false,rows};
  return store;
}

const env0={
  TALLY_V2_VERSION_SHA:versionSha,
  TALLY_VERSION_DAG_REGISTRY_PATH:'C:/fixtures/registry-284.sqlite',
  TALLY_VERSION_DAG_WRITER_ID:'tally-host:win32:i284:1',
  TALLY_VERSION_DAG_WRITER_LOGIN:'kaicreator-mm',
  TALLY_VERSION_DAG_REVIEW_ISSUE:'384',
  TALLY_VERSION_DAG_REVIEW_COMMENT_ID:'7001',
  TALLY_VERSION_DAG_REVIEW_LOGIN:'independent-reviewer',
  TALLY_VERSION_DAG_APPROVAL_ISSUE:'284',
  TALLY_VERSION_DAG_APPROVAL_COMMENT_ID:'6739635285',
  TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON:JSON.stringify(
    Object.fromEntries(ROSTER.map(n=>[String(n),{writeSetSha256:digest,
      document:'frozen exact write set for #'+n}]))),
};

/** Full trusted-transport fake: pinned version branch, canonical live issues,
 * exact witness comments, the immutable pack Contents blob and the official
 * blocked_by dependency endpoints. */
function fakeTransports({pack=candidatePack([plannedEdge]),blockedBy={},
  reviewWitness:review=reviewWitness,approvalWitness:approval=approvalWitness,
  editWitnessBetweenReads=false,packBlobOverride}={}){
  const calls={posts:0,getContent:0};
  const blocked={...blockedBy};
  const packContent=packBlobOverride??
    Buffer.from(JSON.stringify(pack)).toString('base64');
  const commentBody=(commentId)=>{
    if(commentId===Number(env0.TALLY_VERSION_DAG_REVIEW_COMMENT_ID))
      return JSON.stringify(review);
    if(commentId===Number(env0.TALLY_VERSION_DAG_APPROVAL_COMMENT_ID))
      return JSON.stringify(approval);
    return null;
  };
  const commentRef=commentId=>'https://github.com/'+repository+'/issues/'+
    (commentId===Number(env0.TALLY_VERSION_DAG_REVIEW_COMMENT_ID)?
      env0.TALLY_VERSION_DAG_REVIEW_ISSUE:env0.TALLY_VERSION_DAG_APPROVAL_ISSUE)+
    '#issuecomment-'+commentId;
  const commentIssueUrl=commentId=>'https://api.github.com/repos/'+owner+'/'+
    repo+'/issues/'+(commentId===Number(env0.TALLY_VERSION_DAG_REVIEW_COMMENT_ID)?
      env0.TALLY_VERSION_DAG_REVIEW_ISSUE:env0.TALLY_VERSION_DAG_APPROVAL_ISSUE);
  const octokit={
    rest:{
      repos:{
        getBranch:async()=>({data:{commit:{sha:versionSha}}}),
        getCommit:async()=>({data:{sha:versionSha}}),
        getContent:async({path,ref})=>{
          calls.getContent++;
          assert.equal(ref,packCommitSha);
          if(path!==packPath)return {data:{sha:'0'.repeat(40)}};
          return {data:{sha:packBlobSha,encoding:'base64',content:packContent}};
        }
      },
      issues:{
        get:async({issue_number})=>({data:{number:issue_number,
          id:issue_number*1000,html_url:issueUrl(issue_number),
          title:'Task '+issue_number,state:'open',
          updated_at:'2026-11-09T11:00:00Z',
          repository_url:'https://api.github.com/repos/'+owner+'/'+repo,
          labels:[{name:'type:task'},{name:'state:planned'}]}}),
        getComment:async({comment_id})=>{
          const body=commentBody(comment_id);
          if(body===null)throw Object.assign(new Error('not found'),{status:404});
          if(comment_id===Number(env0.TALLY_VERSION_DAG_REVIEW_COMMENT_ID)&&
             editWitnessBetweenReads){
            editWitnessBetweenReads=false;
            return {data:{id:comment_id,body:'{"status":"EDITED"}',
              html_url:commentRef(comment_id),
              issue_url:commentIssueUrl(comment_id),
              updated_at:'2026-11-09T12:05:00Z',
              user:{login:env0.TALLY_VERSION_DAG_REVIEW_LOGIN}}};
          }
          return {data:{id:comment_id,body,html_url:commentRef(comment_id),
            issue_url:commentIssueUrl(comment_id),
            created_at:'2026-11-09T12:00:00Z',updated_at:'2026-11-09T12:00:00Z',
            user:{login:comment_id===Number(env0.TALLY_VERSION_DAG_REVIEW_COMMENT_ID)?
              env0.TALLY_VERSION_DAG_REVIEW_LOGIN:owner}}};
        }
      }
    },
    paginate:{iterator:async function*(route,params){
      assert.equal(route,
        'GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      const list=blocked[params.issue_number];
      if(list instanceof Error)throw list;
      yield {data:list??[]};
    }},
    request:async(route,params)=>{
      assert.equal(route,
        'POST /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      calls.posts++;
      const number=params.issue_id/1000;
      blocked[params.issue_number]=[...(blocked[params.issue_number]??[]),
        {id:params.issue_id,number,state:'open',html_url:issueUrl(number)}];
      return {status:201,data:{}};
    }
  };
  return {octokit,calls,blocked};
}

function hostFixture(options){
  const transports=fakeTransports(options);
  // One registry file per host: every operation reopens the SAME durable CAS
  // store, exactly like the sqlite-backed Host deployment.
  const shared=memoryStore();
  const stores=[shared];
  const openStore=async()=>shared;
  const host=createVersionTaskDagHost({env:env0,octokit:transports.octokit,
    openStore});
  return {host,transports,stores};
}

test('host composition drives the full pre-freeze chain through one selected C port',async()=>{
  assert.deepEqual(VERSION_TASK_DAG_HOST_IDS.repository.split('/'),[owner,repo]);
  const {host,transports,stores}=hostFixture({});
  // 1. The proposed serial edge writes once through the official endpoint.
  const edge=await host.addNativeTaskDependency({taskPackRef,
    taskPackBlobSha:packBlobSha,blockedIssueNumber:312,blockedByIssueNumber:284});
  assert.equal(edge.status,'NATIVE_EDGE_VERIFIED');
  assert.equal(transports.calls.posts,1);
  assert.equal(edge.authorized,false);assert.equal(edge.taskPass,false);
  // 2. Freeze refuses while the candidate still carries the PLANNED edge.
  const premature=await host.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(premature.status,'BLOCKED');
  assert.equal(premature.reason,'PLANNED_EDGES_REMAIN');
  // 3. A-refreshed candidate (native edge read back, fresh witnesses) freezes
  //    and materializes the per-Task write-set records in one CAS registry.
  const refreshed=hostFixture({pack:candidatePack([nativeEdge]),
    blockedBy:nativeBlockers});
  const frozen=await refreshed.host.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(frozen.status,'FROZEN');
  assert.equal(frozen.taskCount,3);
  assert.equal(frozen.writeSetRecordCount,3);
  const store=refreshed.stores[0];
  const dagRows=[...store.rows.keys()].filter(k=>k.startsWith('tally:v2:version-task-dag:'));
  assert.equal(dagRows.length,1);
  assert.equal(store.rows.get('tally:v2:task-writeset:284').value.writeSetSha256,digest);
  // 4. No dependency write after final freeze (same registry as the freeze).
  const after=await refreshed.host.addNativeTaskDependency({taskPackRef,
    taskPackBlobSha:packBlobSha,blockedIssueNumber:219,blockedByIssueNumber:312});
  assert.equal(after.status,'BLOCKED');
  assert.equal(after.reason,'VERSION_DAG_ALREADY_FROZEN');
  assert.equal(refreshed.transports.calls.posts,0,'no further official writes');
  assert.equal(transports.calls.posts,1,'exactly one official write in phase 1');
});

test('host refuses edited witness comments and pack identity drift',async()=>{
  const edited=hostFixture({editWitnessBetweenReads:true,
    pack:candidatePack([nativeEdge]),blockedBy:nativeBlockers});
  const editedResult=await edited.host.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(editedResult.status,'BLOCKED');
  assert.equal(editedResult.reason,'INDEPENDENT_REVIEW_REQUIRED');
  const foreignBlob=hostFixture({pack:candidatePack([nativeEdge]),
    blockedBy:nativeBlockers,
    packBlobOverride:Buffer.from(JSON.stringify(
      {...candidatePack([nativeEdge]),versionBaseSha:'0'.repeat(40)})).toString('base64')});
  const foreign=await foreignBlob.host.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(foreign.status,'BLOCKED');
  assert.equal(foreign.reason,'TASK_PACK_CONTENT_INVALID');
});

test('host configuration is trusted-env-only and closes the store per call',async()=>{
  const transports=fakeTransports();
  assert.throws(()=>createVersionTaskDagHost({env:{},octokit:transports.octokit,
    openStore:async()=>memoryStore()}),
    /TRUSTED_VERSION_TASK_DAG_HOST_CONFIGURATION_REQUIRED/);
  assert.throws(()=>createVersionTaskDagHost({env:env0,octokit:{},
    openStore:async()=>memoryStore()}),
    /TRUSTED_VERSION_TASK_DAG_HOST_TRANSPORTS_REQUIRED/);
  assert.throws(()=>createVersionTaskDagHost({env:{
    ...env0,TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON:'not json'},
    octokit:transports.octokit,openStore:async()=>memoryStore()}),
    /INVALID_VERSION_TASK_DAG_WRITESET_DOCUMENTS/);
  const host=createVersionTaskDagHost({env:env0,
    octokit:fakeTransports().octokit,
    openStore:async()=>{throw new Error('SQLITE_UNAVAILABLE');}});
  const unavailable=await host.freezeVersionTaskDag({taskPackRef,
    taskPackBlobSha:packBlobSha});
  assert.equal(unavailable.status,'BLOCKED');
  assert.equal(unavailable.reason,'VERSION_TASK_DAG_HOST_UNAVAILABLE');
  const ok=hostFixture({});
  await ok.host.addNativeTaskDependency({taskPackRef,taskPackBlobSha:packBlobSha,
    blockedIssueNumber:312,blockedByIssueNumber:284});
  assert.equal(ok.stores[0].closed,true,'registry closed after each operation');
});
