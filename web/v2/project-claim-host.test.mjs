import test from 'node:test';
import assert from 'node:assert/strict';
import {createProjectClaimHost} from './project-claim-host.mjs';
const sha='a'.repeat(40),repository='org/new';
const issueNumber=27,issueRef='https://github.com/org/new/issues/27';
const settings={repository,owner:'org',repo:'new',versionId:'main',
  expectedSha:sha,token:'github_pat_test_read_only'};
const env={TALLY_V2_CLAIM_DISPATCH:'host-mounted',
  TALLY_V2_CLAIM_REPOSITORY:repository,TALLY_V2_CLAIM_VERSION_REF:'main',
  TALLY_V2_CLAIM_SOURCE_SHA:sha,TALLY_V2_CLAIM_ISSUE_NUMBER:'27',
  TALLY_V2_CLAIM_WRITER_ID:'host-single-writer',
  TALLY_V2_CLAIM_WRITER_LOGIN:'operator',
  TALLY_V2_CLAIM_DISPATCH_ID:'dispatch:27:1',
  TALLY_V2_CLAIM_OPERATOR_ID:'operator:27:1',
  TALLY_V2_CLAIM_OPERATOR_KIND:'human',
  TALLY_V2_CLAIM_REGISTRY_PATH:'/tmp/scoped-claim.sqlite'};
const nativePreflight=async()=>({
  connected:true,stage:'NATIVE_PREFLIGHT_READY',
  issueUrl:issueRef,sourceSha:sha});
const witness=async()=>({
  status:'PUBLICATION_WITNESSED',sourceQualified:true,
  issueRef,sourceSha:sha,commentId:41,
  commentRef:issueRef+'#issuecomment-41',
  dispatchId:'dispatch:27:1',actorRole:'builder',
  operatorKind:'human',operatorId:'operator:27:1',
  protectedClaimKey:'org/new#27:builder:__default__',
  generation:1,transportActor:'operator',
  claimPublication:'NATIVE_COMMENT_READBACK_ONLY',
  claimAccepted:false,admitted:false,authorized:false,
  taskPass:false,releasePass:false});
const octokit={rest:{issues:{createComment:()=>{}}}};
function store(){
  const rows=new Map();
  return {async get(key){return rows.get(key)??null},
    async put(key,value,opt={}){
      const prior=rows.get(key);
      if(opt.mustBeAbsent&&prior)throw new Error('CAS_EXISTS');
      if(opt.mustMatchRevision!==undefined&&
         prior?.revision!==opt.mustMatchRevision)throw new Error('CAS_CONFLICT');
      const row={revision:(prior?.revision??0)+1,value};
      rows.set(key,row);return row;
    },async close(){},rows};
}
test('no Host mount means no Claim publisher',()=>{
  assert.equal(createProjectClaimHost({settings,env:{}}),null);
});
test('wrong project/ref/source or missing writer stops before any effect',()=>{
  for(const bad of [{...env,TALLY_V2_CLAIM_REPOSITORY:'org/other'},
    {...env,TALLY_V2_CLAIM_VERSION_REF:'version/v2'},
    {...env,TALLY_V2_CLAIM_SOURCE_SHA:'b'.repeat(40)},
    {...env,TALLY_V2_CLAIM_WRITER_LOGIN:'bad login'}]){
    assert.throws(()=>createProjectClaimHost({settings,env:bad,
      octokit,openStore:async()=>store()}),
    /TRUSTED_SELECTED_CLAIM_HOST_REQUIRED/);
  }
});
test('Task scope mismatch cannot even open CAS store',async()=>{
  let calls=0;
  const host=createProjectClaimHost({settings,env,octokit,
    openStore:async()=>{calls++;return store();},
    createPublisher:()=>({publishClaim:async()=>{calls++}}),
    readAdmission:nativePreflight,readWitness:witness});
  const result=await host.publish({issueNumber:28,sourceSha:sha});
  assert.equal(result.status,'BLOCKED');
  assert.equal(result.authorized,false);
  assert.equal(calls,0);
});
test('exact selected Issue → CAS reservation → verified native witness',async()=>{
  const db=store();let posts=0,reads=0;
  const host=createProjectClaimHost({settings,env,octokit,
    openStore:async()=>db,
    createPublisher:()=>({publishClaim:async i=>{
      posts++;assert.equal(i.issueNumber,27);
      assert.equal(i.repository,repository);assert.equal(i.writerId,
        'host-single-writer');
      return {commentId:41};
    }}),
    readAdmission:async req=>{reads++;return nativePreflight(req);},
    readWitness:async input=>{
      assert.equal(input.commentId,41);
      assert.equal(input.protectedClaimKey,'org/new#27:builder:__default__');
      return witness(input);
    }});
  const result=await host.publish({issueNumber,sourceSha:sha});
  assert.equal(result.status,'RECORDED');
  assert.equal(result.claimPublication,'VERIFIED_PENDING_KERNEL_ADMISSION');
  assert.equal(result.authorized,false);
  assert.equal(result.admitted,false);
  assert.equal(posts,1);assert.equal(reads,2);
  assert.equal(db.rows.size,1);
  const claim=[...db.rows.values()][0].value;
  assert.equal(claim.phase,'PUBLISHED_VERIFIED');
  assert.equal(claim.issueRef,issueRef);
  const again=await host.publish({issueNumber,sourceSha:sha});
  assert.equal(again.status,'BLOCKED');
  assert.equal(posts,1,'never duplicate the GitHub POST');
});
test('ambiguous GitHub POST leaves CAS pending; never retry blindly',async()=>{
  const db=store();let posts=0;
  const host=createProjectClaimHost({settings,env,octokit,
    openStore:async()=>db,
    createPublisher:()=>({publishClaim:async()=>{
      posts++;throw new Error('ACK_LOST');
    }}),
    readAdmission:nativePreflight,readWitness:witness});
  const first=await host.publish({issueNumber,sourceSha:sha});
  assert.equal(first.status,'BLOCKED');
  assert.equal(first.reason,'GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
  const second=await host.publish({issueNumber,sourceSha:sha});
  assert.equal(second.status,'BLOCKED');
  assert.equal(posts,1);
  assert.equal([...db.rows.values()][0].value.phase,'PENDING_GITHUB_PUBLISH');
});
test('native preflight refusal cannot reserve CAS or POST',async()=>{
  const db=store();
  const host=createProjectClaimHost({settings,env,octokit,
    openStore:async()=>db,readAdmission:async()=>({
      connected:true,stage:'STALE_SOURCE',issueUrl:issueRef,sourceSha:sha}),
    createPublisher:()=>({publishClaim:async()=>{throw new Error('must not run')}}),
    readWitness:witness});
  const out=await host.publish({issueNumber,sourceSha:sha});
  assert.equal(out.status,'BLOCKED');
  assert.equal(out.reason,'SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
  assert.equal(db.rows.size,0);
});

test('post-publication source or witness drift never records accepted CAS row',async()=>{
  for(const scenario of ['stale-source','foreign-comment']){
    const db=store();let admissionReads=0;
    const host=createProjectClaimHost({settings,env,octokit,
      openStore:async()=>db,
      createPublisher:()=>({publishClaim:async()=>({commentId:41})}),
      readAdmission:async()=>{
        admissionReads++;
        return admissionReads===2&&scenario==='stale-source'?
          {connected:true,stage:'BLOCKED',issueUrl:issueRef,sourceSha:sha,
            reason:'TASK_DRAFT_SOURCE_SHA_STALE'}:nativePreflight();
      },
      readWitness:async()=>scenario==='foreign-comment'?
        {...await witness(),protectedClaimKey:'org/other#27:builder:__default__'}:
        witness()});
    const result=await host.publish({issueNumber:27,sourceSha:sha});
    assert.equal(result.status,'BLOCKED');
    assert.equal(result.reason,'GITHUB_CLAIM_READBACK_NOT_VERIFIED');
    const persisted=[...db.rows.values()][0].value;
    assert.equal(persisted.phase,'PENDING_GITHUB_PUBLISH');
  }
});
