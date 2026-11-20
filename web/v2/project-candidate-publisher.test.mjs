import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createSelectedCandidatePublisher}
  from './project-candidate-publisher.mjs';
const sourceSha='a'.repeat(40),commit='b'.repeat(40),tree='c'.repeat(40);
const repository='org/new',ref='https://github.com/org/new/issues/27';
const attemptId='attempt:task:27:1',issueNumber=27,candidateRelativePath='health.txt';
const env={TALLY_V2_CANDIDATE_PUBLISH:'host-mounted',
  T06M_TARGET_REPOSITORY:repository,T06M_TARGET_VERSION_REF:'main',
  T06M_TARGET_ISSUE_NUMBER:'27',T06M_TARGET_CANDIDATE_PATH:candidateRelativePath,
  TALLY_V2_CLAIM_REPOSITORY:repository,TALLY_V2_CLAIM_ISSUE_NUMBER:'27',
  TALLY_V2_CLAIM_SOURCE_SHA:sourceSha,TALLY_V2_AGENT_CLAIM_SOURCE_SHA:sourceSha,
  TALLY_V2_AGENT_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
  TALLY_V2_CLAIM_REGISTRY_PATH:'/tmp/claim.sqlite',
  TALLY_V2_AGENT_CLAIM_ATTEMPT_ID:attemptId,
  TALLY_V2_AGENT_EVIDENCE_DIR:'/tmp/evidence',
  TALLY_V2_AGENT_WORKTREE_PATH:'/tmp/worktree',
  TALLY_GITHUB_WRITE_TOKEN:'token-test-write',
  TALLY_V2_CLAIM_DISPATCH_ID:'dispatch:27',
  TALLY_V2_CLAIM_OPERATOR_ID:'operator:27',
  TALLY_V2_CLAIM_OPERATOR_KIND:'human',
  TALLY_V2_CLAIM_WRITER_ID:'writer:27',
  TALLY_V2_CLAIM_WRITER_LOGIN:'operator'};
const settings={repository,owner:'org',repo:'new',versionId:'main',
  expectedSha:sourceSha,token:'test-read-token'};
const key=repository+'#27:builder:__default__';
const h=s=>createHash('sha256').update(s).digest('hex');
const document=JSON.stringify([candidateRelativePath]);
const claimKey='tally:v2:claim-admission:'+h(key);
const wsKey='tally:v2:task-writeset:27';
const attemptKey='tally:v2:local-agent-attempt:'+attemptId;
const candidateRef='refs/tally-candidates/'+attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
const claim={revision:1,value:{phase:'PUBLISHED_VERIFIED',repository,
  issueRef:ref,sourceSha,actorRole:'builder',protectedClaimKey:key,
  dispatchId:'dispatch:27',operatorId:'operator:27',operatorKind:'human',
  writerId:'writer:27',commentId:42}};
const fsw={revision:0,value:{phase:'FROZEN_DAG_WRITESET',repository,
  issueNumber,sourceSha,writeSetSha256:h(document),document}};
const attempt={revision:4,value:{phase:'CANDIDATE_READY',attemptId,
  sourceSha,issueRef:ref,candidateRelativePath,
  candidateCommit:commit,candidateTree:tree,candidateRef,
  worktreePath:'/tmp/worktree'}};
function fixture({claimRow=claim,fswRow=fsw,attemptRow=attempt,
  branchHead=sourceSha,postFail=false,pushFail=false}={}){
  const db=new Map([[claimKey,claimRow],[wsKey,fswRow]]);
  const local=new Map([[attemptKey,attemptRow]]);
  let pushes=0,posts=0,pr=null,publishedSha=null;
  const openStore=async({path})=>({
    get:async key=>(path.includes('claim')?db:local).get(key)??null,
    put:async(key,value,options={})=>{
      const entries=path.includes('claim')?db:local,prior=entries.get(key);
      if(options.mustBeAbsent&&prior)throw Error('CAS_ALREADY_EXISTS');
      if(options.mustMatchRevision!==undefined&&
         prior?.revision!==options.mustMatchRevision)throw Error('CAS_REVISION');
      const row={revision:(prior?.revision??-1)+1,value};entries.set(key,row);return row;
    },close:async()=>{}});
  const exec=async(git,args)=>{
    if(args.includes('push')){
      pushes++;if(pushFail)throw Error('ACK_LOST');
      publishedSha=commit;return {stdout:''};
    }
    if(args.includes('diff'))return {stdout:'health.txt\n'};
    const last=args.at(-1);
    return {stdout:last===commit+'^'?sourceSha+'\n':
      last===commit+'^{tree}'?tree+'\n':commit+'\n'};
  };
  const octokit={rest:{
    repos:{getBranch:async()=>({data:{commit:{sha:branchHead}}})},
    git:{getRef:async()=>{
      if(publishedSha)return {data:{object:{sha:publishedSha}}};
      throw Object.assign(Error('not-found'),{status:404});
    }},
    pulls:{
      list:async()=>({data:pr?[{number:pr.number,head:{ref:pr.head.ref}}]:[]}),
      create:async input=>{
        posts++;
        if(postFail)throw Error('ACK_LOST');
        assert.equal(input.draft,true);
        pr={number:5,html_url:'https://github.com/org/new/pull/5',
          state:'open',draft:true,head:{ref:input.head,sha:commit,
            repo:{full_name:repository}},base:{ref:'main',
            repo:{full_name:repository}}};
        return {data:pr};
      },get:async()=>({data:pr})
    }
  }};
  const readEvidence=async()=>JSON.stringify({status:'PASS',realAgentEffect:true,
    hiddenPass:false,result:{sourceSha,attemptId,agentLaunch:true,
      result:'LOCAL_CANDIDATE_READY',candidateCommit:commit,
      candidateTree:tree,hostStorePath:'/tmp/evidence/host.sqlite'}});
  const publisher=createSelectedCandidatePublisher({settings,env,octokit,
    openStore,exec,readEvidence,
    readStat:async()=>({isFile:()=>true,size:512}),
    resolveRealPath:async x=>x,
    readWitness:async()=>({status:'PUBLICATION_WITNESSED',commentId:42,
      issueRef:ref,sourceSha})});
  return {publisher,db,counts:()=>({pushes,posts})};
}
test('new-project exact K/CAS/FSW candidate publishes ONLY a draft PR',async()=>{
  const f=fixture();
  const result=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(result.status,'DRAFT_PR_PUBLISHED_VERIFIED');
  assert.equal(result.pullNumber,5);assert.equal(result.authorized,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.deepEqual(f.counts(),{pushes:1,posts:1});
  const replay=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(replay.status,'DRAFT_PR_PUBLISHED_VERIFIED');
  assert.deepEqual(f.counts(),{pushes:1,posts:1});
});
test('wrong request scope never accesses GitHub or CAS',async()=>{
  const f=fixture();
  const result=await f.publisher.publish({issueNumber:28,sourceSha});
  assert.equal(result.status,'BLOCKED');assert.equal(result.reason,
    'SELECTED_CANDIDATE_TASK_SCOPE_MISMATCH');
  assert.deepEqual(f.counts(),{pushes:0,posts:0});
});
test('wrong branch source or missing/altered proof prevents push',async()=>{
  for(const f of [fixture({branchHead:'f'.repeat(40)}),
    fixture({claimRow:null}),fixture({fswRow:null}),
    fixture({attemptRow:{...attempt,value:{...attempt.value,
      candidateRelativePath:'other.txt'}}})]){
    const result=await f.publisher.publish({issueNumber:27,sourceSha});
    assert.equal(result.status,'BLOCKED');
    assert.deepEqual(f.counts(),{pushes:0,posts:0});
  }
});
test('lost Git push ACK keeps CAS fence and never retries mutation',async()=>{
  const f=fixture({pushFail:true});
  const first=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(first.status,'PUBLICATION_UNKNOWN');
  const second=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(second.reason,'PR_PUBLICATION_RECONCILIATION_REQUIRED');
  assert.deepEqual(f.counts(),{pushes:1,posts:0});
});
test('lost Draft PR POST never creates second PR by blind retry',async()=>{
  const f=fixture({postFail:true});
  const first=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(first.status,'PUBLICATION_UNKNOWN');
  const second=await f.publisher.publish({issueNumber:27,sourceSha});
  assert.equal(second.reason,'PR_PUBLICATION_RECONCILIATION_REQUIRED');
  assert.deepEqual(f.counts(),{pushes:1,posts:1});
});
test('target mismatches reject Host at creation',()=>{
  assert.equal(createSelectedCandidatePublisher({settings,env:{}}),null);
  assert.throws(()=>createSelectedCandidatePublisher({settings,
    env:{...env,T06M_TARGET_REPOSITORY:'org/foreign'}}),TypeError);
});
