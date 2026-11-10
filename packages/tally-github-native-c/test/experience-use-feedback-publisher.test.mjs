import test from 'node:test';
import assert from 'node:assert/strict';
import {createHostBoundExperienceFeedbackPublisher} from '../src/experience-use-feedback-publisher.mjs';
import {projectExperienceUseFeedback,feedbackReportId}
  from '../../tally-github-c/src/experience-use-feedback.mjs';
const sha='a'.repeat(40),pubSha='b'.repeat(40),repo='kaicreator-mm/tally';
const ref='https://github.com/'+repo+'/issues/274#issuecomment-1234';
function candidate(phase='PRESENTED',extra={}) {
  const input={repository:repo,issueNumber:274,taskRef:'#274',attemptRef:'attempt:1',
    sourceSha:sha,publisherSourceSha:pubSha,adsRevision:'2c395c55569fe2edf3f4135f12d3d069e526d0b7',
    publisherNamespace:'tally-feedback',revision:1,
    experienceRef:'https://github.com/'+repo+'/issues/274',
    procedureRevision:'procedure-v1',phase,epistemic:'OBSERVED',
    observedAt:'2026-11-10T00:00:00Z',
    actor:{role:'builder',operator_kind:'other',operator_id:'host:1',
      session_ref:ref,transport_actor:'kaicreator-mm'},
    sourceRefs:[ref],phaseEvidence:{kind:phase,ref,decision:null,
      independent:false},verifierRefs:[],clearanceRef:ref,supersedes:null,
    sourceWitness:{repository:repo,issueNumber:274,sourceSha:sha,
      currentness:'CURRENT',readback:'VERIFIED'},...extra};
  input.reportId=feedbackReportId(input);
  return projectExperienceUseFeedback(input);
}
function mock(options={}) {
  const comments=[],state={posts:0,leaseReads:0,permits:0,evidenceReads:0};
  const clock='2026-11-10T00:00:00Z';
  const octokit={rest:{issues:{
    get:async()=>({data:{number:274,repository_url:'https://api.github.com/repos/'+repo}}),
    listComments:()=>{},
    createComment:async({body})=>{
      state.posts++;
      if(options.write===false)throw Error('HTTP 500');
      const id=100+state.posts;
      const c={id,body,html_url:'https://github.com/'+repo+
        '/issues/274#issuecomment-'+id,issue_url:'https://api.github.com/repos/'+repo+
        '/issues/274',user:{login:options.login??'kaicreator-mm'},
        created_at:clock,updated_at:clock};
      comments.push(c);
      if(options.lostAck)throw Error('LOST_ACK');
      return {data:c};
    },
    getComment:async({comment_id})=>({data:comments.find(c=>c.id===comment_id)})
  }},paginate:{iterator:async function*() {
    if(options.paginationFails)throw Error('PAGE_FAILED');
    yield {data:comments.map(c=>({...c}))};
  }}};
  const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',
    exclusiveHostControl:true,repository:repo,writerId:'host:1',
    githubWriterLogin:options.login??'kaicreator-mm',publisherNamespace:'tally-feedback',
    sourceSha:sha,publisherSourceSha:pubSha,
    readHeldFence:async()=>{
      state.leaseReads++;
      return {sourceKind:'TRUSTED_HOST_RESOURCE_FENCE_READBACK',
        complete:true,fenceHeld:options.lease!==false&&
          !(Number.isSafeInteger(options.failLeaseAfter)&&
            state.leaseReads>options.failLeaseAfter),
        heldThrough:'CAS_PUBLISH_READBACK',repository:repo,writerId:'host:1'};
    },
    readKernelPermit:async()=>{
      state.permits++;
      return {status:options.permit===false?'BLOCKED':'AUTHORIZED',
        repository:repo,issueNumber:274,sourceSha:sha};
    },
    readFeedbackEvidence:async(input)=>{
      state.evidenceReads++;
      return {status:options.evidence===false?'UNKNOWN':'VERIFIED',
        sourceKind:'HOST_VERIFIED_FEEDBACK_EVIDENCE',repository:repo,
        issueNumber:274,sourceSha:sha,reportId:input.reportId,
        revision:input.revision,phase:input.phase,
        experienceRef:input.experienceRef,procedureRevision:input.procedureRevision,
        evidenceRef:input.evidenceRef,dataDigest:input.expectedDataDigest,
        humanDecision:input.phase==='ACCEPTED'?'APPROVE':
          input.phase==='SKIPPED'?'REJECT':null,
        independentVerified:true};
    }};
  return {octokit,writerAuthority,comments,state,publisher:()=> 
    createHostBoundExperienceFeedbackPublisher({octokit,
      owner:'kaicreator-mm',repo:'tally',writerAuthority})};
}
test('new native publish uses one POST and real exact GitHub comment readback',async()=>{
  const m=mock(),candidateValue=candidate(),r=await m.publisher().publishFeedback({
    issueNumber:274,candidate:candidateValue});
  assert.equal(candidateValue.status,'CANDIDATE_ONLY');
  assert.equal(r.status,'PUBLISHED_VERIFIED');assert.equal(r.newPost,true);
  assert.equal(m.state.posts,1);assert.equal(r.taskPass,false);
  assert.equal(m.state.leaseReads,3);assert.equal(m.state.permits,3);
});
test('exact rerun is idempotent: zero additional POST',async()=>{
  const m=mock(),p=m.publisher(),v=candidate();
  await p.publishFeedback({issueNumber:274,candidate:v});
  const r=await p.publishFeedback({issueNumber:274,candidate:v});
  assert.equal(r.reason,'EXACT_IDEMPOTENT_NATIVE_REPLAY');
  assert.equal(m.state.posts,1);
});
test('ACK lost but native comment present is reconciled, never retried',async()=>{
  const m=mock({lostAck:true}),r=await m.publisher().publishFeedback({
    issueNumber:274,candidate:candidate()});
  assert.equal(r.status,'PUBLISHED_VERIFIED');
  assert.equal(r.reason,'LOST_ACK_RECOVERED_BY_READBACK');
  assert.equal(m.state.posts,1);
});
test('no POST ACK and no native comment stays UNKNOWN',async()=>{
  const m=mock({write:false}),r=await m.publisher().publishFeedback({
    issueNumber:274,candidate:candidate()});
  assert.equal(r.status,'UNKNOWN');
  assert.equal(m.state.posts,1);
});
test('pagination failure and missing fence/permit refuse without POST',async()=>{
  for(const options of [{paginationFails:true},{lease:false},{permit:false}]){
    const m=mock(options),r=await m.publisher().publishFeedback({
      issueNumber:274,candidate:candidate()});
    assert.notEqual(r.status,'PUBLISHED_VERIFIED');
    assert.equal(m.state.posts,0);
  }
});
test('same report identity with divergent bytes conflicts, never last-write-wins',async()=>{
  const m=mock(),p=m.publisher();await p.publishFeedback({
    issueNumber:274,candidate:candidate()});
  const divergent=candidate('PRESENTED',{observedAt:'2026-11-10T01:00:00Z'});
  const r=await p.publishFeedback({issueNumber:274,candidate:divergent});
  assert.equal(r.reason,'DIVERGENT_FEEDBACK_IDENTITY');
  assert.equal(m.state.posts,1);
});
test('foreign target or unbound publisher identity denies source mutation',async()=>{
  const m=mock(),p=m.publisher();
  assert.equal((await p.publishFeedback({issueNumber:275,candidate:candidate()})).status,
    'BLOCKED');
  const bad=candidate('PRESENTED',{publisherSourceSha:'d'.repeat(40)});
  assert.equal((await p.publishFeedback({issueNumber:274,candidate:bad})).status,
    'BLOCKED');
  assert.equal(m.state.posts,0);
});
test('unqualified Host construction cannot be borrowed by UI',()=>{
  const m=mock();delete m.writerAuthority.readHeldFence;
  assert.throws(()=>m.publisher(),/HOST_BOUND_FEEDBACK_PUBLISHER_REQUIRED/);
});
test('revision two requires actual native preceding revision and digest',async()=>{
  const m=mock(),p=m.publisher(),first=candidate();
  assert.equal((await p.publishFeedback({issueNumber:274,candidate:first})).status,
    'PUBLISHED_VERIFIED');
  const second=candidate('PRESENTED',{revision:2,
    supersedes:{revision:1,data_digest:first.dataDigest,
      comment_ref:m.comments[0].html_url}});
  const r=await p.publishFeedback({issueNumber:274,candidate:second});
  assert.equal(r.status,'PUBLISHED_VERIFIED');
  assert.equal(m.state.posts,2);
});

test('unverified Host feedback review/phase witness blocks before GitHub POST',async()=>{
  const m=mock({evidence:false});
  const r=await m.publisher().publishFeedback({issueNumber:274,candidate:candidate()});
  assert.equal(r.status,'BLOCKED');
  assert.equal(m.state.posts,0);
  assert.equal(m.state.evidenceReads,1);
});
test('publisher construction refuses absent trusted feedback evidence reader',()=>{
  const m=mock();delete m.writerAuthority.readFeedbackEvidence;
  assert.throws(()=>m.publisher(),/HOST_BOUND_FEEDBACK_PUBLISHER_REQUIRED/);
});

test('valid GitHub App bot writer login is accepted with exact matching transport actor',async()=>{
  const m=mock({login:'myapp[bot]'});
  const original=candidate('PRESENTED',{
    actor:{role:'builder',operator_kind:'other',operator_id:'host:1',
      session_ref:ref,transport_actor:'myapp[bot]'}
  });
  const r=await m.publisher().publishFeedback({issueNumber:274,candidate:original});
  assert.equal(r.status,'PUBLISHED_VERIFIED');assert.equal(m.state.posts,1);
});
test('mismatched transport actor is rejected before any GitHub POST',async()=>{
  const m=mock();
  const forged=candidate('PRESENTED',{
    actor:{role:'builder',operator_kind:'other',operator_id:'host:1',
      session_ref:ref,transport_actor:'wrong-native-actor'}
  });
  const r=await m.publisher().publishFeedback({issueNumber:274,candidate:forged});
  assert.equal(r.reason,'FEEDBACK_NATIVE_ACTOR_MISMATCH');
  assert.equal(m.state.posts,0);
});
test('losing the held fence after successful native readback never reports publication PASS',async()=>{
  const m=mock({failLeaseAfter:2});
  const r=await m.publisher().publishFeedback({issueNumber:274,candidate:candidate()});
  assert.equal(m.state.posts,1);
  assert.equal(r.status,'UNKNOWN');
  assert.equal(r.reason,'FEEDBACK_POST_LEASE_OR_PERMIT_LOST');
  assert.equal(r.taskPass,false);
});
test('lost ACK with post-readback fence loss remains UNKNOWN without duplicate POST',async()=>{
  const m=mock({lostAck:true,failLeaseAfter:2});
  const r=await m.publisher().publishFeedback({issueNumber:274,candidate:candidate()});
  assert.equal(m.state.posts,1);
  assert.equal(r.status,'UNKNOWN');
  assert.equal(r.reason,'FEEDBACK_POST_LEASE_OR_PERMIT_LOST');
});
