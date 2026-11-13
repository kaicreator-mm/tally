import test from 'node:test';
import assert from 'node:assert/strict';
import {publishMaintenanceIssueFromClient,findMaintenanceIssueFromClient,
  readMaintenanceIssueFromClient,readMaintenanceReviewWitnessFromClient,
  maintenanceMarkerFor,__test}
  from '../src/maintenance-effects.mjs';

const digest='c'.repeat(64);
const marker=`tally:maintenance:219:${digest}`;
const title='[maintenance] Defect against immutable release v1.5.5';
const body=`Defect candidate for the next version.\n\nmarker: ${marker}`;
const issueRaw=(n,bodyText=body)=>({number:n,state:'open',title,
  html_url:`https://github.com/kaicreator-mm/tally/issues/${n}`,body:bodyText});
const client=(create,issueGet,listForRepo)=>({rest:{issues:{create,issuesGet:
  undefined,get:issueGet,listForRepo}}});
const publishClient=(created,readback)=>client(
  async()=>({data:created}),async()=>({data:readback}),undefined);

test('canonical publish verifies exact create plus canonical readback',async()=>{
  const calls=[];
  const octokit={rest:{issues:{
    create:async arg=>{calls.push(['create',arg]);return {data:issueRaw(302)};},
    get:async arg=>{calls.push(['get',arg]);return {data:issueRaw(302)};}}}};
  const result=await publishMaintenanceIssueFromClient({octokit,title,body,marker});
  assert.equal(result.state,'ISSUE_PUBLISHED');
  assert.equal(result.reason,'GITHUB_ISSUE_CREATE_AND_READBACK_VERIFIED');
  assert.equal(result.issue.number,302);
  assert.equal(result.issue.url,'https://github.com/kaicreator-mm/tally/issues/302');
  assert.equal(result.externalRef,result.issue.url);
  assert.deepEqual(calls[0][1],{owner:'kaicreator-mm',repo:'tally',title,body});
  assert.equal(result.authorized,false);assert.equal(result.admitted,false);
  assert.equal(result.claimAccepted,false);assert.equal(result.taskPass,false);
  assert.equal(result.releasePass,false);assert.equal(result.duplicate,false);
  assert.equal(result.historicalReleaseMutation,false);
  assert.equal(Object.isFrozen(result),true);
});

test('input and transport refusals never fabricate a published issue',async()=>{
  let called=false;
  const octokit={rest:{issues:{create:async()=>{called=true;return {data:issueRaw(1)};},
    get:async()=>({data:issueRaw(1)})}}};
  for(const args of [{},{octokit},{octokit,title:'',body,marker},
    {octokit,title,body:body.slice(0,5),marker},{octokit,title,body,marker:'evil'},
    {octokit,title:' '.repeat(10),body,marker}]){
    const result=await publishMaintenanceIssueFromClient(args);
    assert.equal(result.state,'NOT_PUBLISHED',JSON.stringify(args));
    assert.equal(result.reason,'MAINTENANCE_ISSUE_INPUT_INVALID');
    assert.equal(result.issue,null);
  }
  assert.equal(called,false,'invalid requests refuse without any client call');
  for(const [broken,reason] of [
    [{rest:{issues:{create:async()=>{throw {status:422};},
      get:async()=>({data:issueRaw(1)})}}},'GITHUB_PUBLICATION_CONFLICT'],
    [{rest:{issues:{create:async()=>{throw {status:429};},
      get:async()=>({data:issueRaw(1)})}}},'GITHUB_RATE_LIMITED'],
    [{rest:{issues:{create:async()=>{throw new Error('x');},
      get:async()=>({data:issueRaw(1)})}}},'GITHUB_PUBLICATION_UNKNOWN'],
    [publishClient({...issueRaw(302),title:'changed'},issueRaw(302)),
      'MAINTENANCE_ISSUE_CREATE_READBACK_MISSING'],
    [publishClient({...issueRaw(302),state:'closed'},issueRaw(302)),
      'MAINTENANCE_ISSUE_CREATE_READBACK_MISSING'],
    [publishClient(issueRaw(302),{...issueRaw(302),body:'no marker inside'}),
      'MAINTENANCE_ISSUE_READBACK_MISMATCH'],
    [publishClient(issueRaw(302),{...issueRaw(303),title}),
      'MAINTENANCE_ISSUE_READBACK_MISMATCH'],
  ]){
    const result=await publishMaintenanceIssueFromClient({...{octokit:broken},title,body,marker});
    assert.equal(result.state,'PUBLICATION_UNKNOWN');
    assert.equal(result.reason,reason);
    assert.equal(result.issue,null);assert.equal(result.githubEffect,undefined);
  }
});

test('reconcile search distinguishes absent, single match and identity conflict',async()=>{
  const listForRepo=async()=>({data:[issueRaw(300,'unrelated report'),
    issueRaw(301,'other discussion')]});
  const absent=await findMaintenanceIssueFromClient({
    octokit:client(undefined,undefined,listForRepo),marker,title});
  assert.equal(absent.state,'RECONCILE_COMPLETE');
  assert.equal(absent.reason,'MAINTENANCE_ISSUE_ABSENT');
  assert.equal(absent.matches.length,0);
  const single=await findMaintenanceIssueFromClient({octokit:client(undefined,
    undefined,async()=>({data:[{...issueRaw(300),body:'unrelated'},issueRaw(301)]})),
    marker,title});
  assert.equal(single.reason,'MAINTENANCE_ISSUE_FOUND');
  assert.deepEqual(single.matches.map(x=>x.number),[301]);
  const conflict=await findMaintenanceIssueFromClient({octokit:client(undefined,
    undefined,async()=>({data:[issueRaw(300),issueRaw(301)]})),marker,title});
  assert.equal(conflict.state,'RECONCILE_CONFLICT');
  assert.equal(conflict.reason,'MAINTENANCE_ISSUE_IDENTITY_CONFLICT');
  assert.equal(conflict.matches.length,2);
  for(const [broken,reason] of [
    [client(undefined,undefined,async()=>{throw {status:403};}),'GITHUB_ACCESS_DENIED'],
    [client(undefined,undefined,async()=>({data:'garbage'})),
      'MAINTENANCE_RECONCILE_READ_FAILED'],
  ]){
    const result=await findMaintenanceIssueFromClient({octokit:broken,marker,title});
    assert.equal(result.state,'RECONCILE_UNKNOWN');assert.equal(result.reason,reason);
  }
  const bad=await findMaintenanceIssueFromClient({octokit:client(),marker:'evil',title});
  assert.equal(bad.reason,'MAINTENANCE_RECONCILE_INPUT_INVALID');
});

test('single-issue readback verifies title, open state and marker body',async()=>{
  const good=await readMaintenanceIssueFromClient({octokit:client(undefined,
    async()=>({data:issueRaw(302)})),issueNumber:302,title,marker});
  assert.equal(good.state,'READBACK_VERIFIED');
  assert.equal(good.issue.number,302);
  for(const [raw,state] of [
    [{...issueRaw(302),state:'closed'},'READBACK_MISMATCH'],
    [{...issueRaw(302),title:'other'},'READBACK_MISMATCH'],
    [{...issueRaw(302),body:'marker missing'},'READBACK_MISMATCH'],
  ]){
    const result=await readMaintenanceIssueFromClient({octokit:client(undefined,
      async()=>({data:raw})),issueNumber:302,title,marker});
    assert.equal(result.state,state);assert.equal(result.issue,null);
  }
  const thrown=await readMaintenanceIssueFromClient({octokit:client(undefined,
    async()=>{throw {status:404};}),issueNumber:302,title,marker});
  assert.equal(thrown.state,'READBACK_UNKNOWN');
  assert.equal(thrown.reason,'GITHUB_RESOURCE_NOT_FOUND');
  const invalid=await readMaintenanceIssueFromClient({octokit:client(),
    issueNumber:0,title,marker});
  assert.equal(invalid.reason,'MAINTENANCE_READBACK_INPUT_INVALID');
});

test('marker derivation requires an exact candidate digest',()=>{
  assert.equal(maintenanceMarkerFor(digest),marker);
  assert.equal(maintenanceMarkerFor('short'),null);
  assert.equal(maintenanceMarkerFor(undefined),null);
});

test('module holds no credential surface and stays repository-pinned',()=>{
  assert.equal(__test.OWNER,'kaicreator-mm');
  assert.equal(__test.REPO,'tally');
  assert.equal(__test.REPOSITORY,'kaicreator-mm/tally');
  assert.equal(__test.validRequest({title,body,marker:marker+'x'}),false);
  assert.equal(__test.readbackIssue({number:1,state:'closed',title,
    html_url:'https://github.com/kaicreator-mm/tally/issues/1'}),null);
  assert.equal(__test.errorReason({status:404}),'GITHUB_RESOURCE_NOT_FOUND');
});


const mReviewSha='a'.repeat(40),mReviewTree='b'.repeat(40);
const mReviewDigest='c'.repeat(64),mReviewCommentId=6840000001;
const mReviewRef='https://github.com/kaicreator-mm/tally/issues/219#issuecomment-'+mReviewCommentId;
const mReviewExpected={issueNumber:219,sourceSha:mReviewSha,
  candidateDigest:mReviewDigest,reviewedPullNumber:278,
  reviewedHead:'d'.repeat(40),reviewedTree:mReviewTree,reviewedBase:mReviewSha};
const mBuilder={accepted:true,role:'builder',issueNumber:219,
  sourceSha:mReviewSha,candidateDigest:mReviewDigest,
  claimRef:'claim:219:builder',operatorId:'operator:builder',
  writerId:'writer:builder',transportActor:'kaicreator-mm'};
const mReviewer={...mBuilder,role:'reviewer',
  claimRef:'claim:219:reviewer',operatorId:'operator:reviewer',
  writerId:'writer:reviewer'};
const mBody={schema:'tally.v2.maintenance-review-witness/1',
  status:'INDEPENDENTLY_VERIFIED',repository:'kaicreator-mm/tally',
  issueNumber:219,...mReviewExpected,
  builderClaimRef:mBuilder.claimRef,reviewerClaimRef:mReviewer.claimRef,
  reviewerOperatorId:mReviewer.operatorId,reviewerWriterId:mReviewer.writerId,
  p0:0,p1:0,p2:1};
const mComment={id:mReviewCommentId,node_id:'IC_maintenance219',
  html_url:mReviewRef,
  issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/219',
  user:{login:'kaicreator-mm'},created_at:'2026-11-13T01:00:00Z',
  updated_at:'2026-11-13T01:00:00Z',body:JSON.stringify(mBody)};
const mGraph={__typename:'IssueComment',id:mComment.node_id,
  fullDatabaseId:String(mComment.id),url:mReviewRef,
  body:mComment.body,lastEditedAt:null,editor:null,
  issue:{number:219,repository:{nameWithOwner:'kaicreator-mm/tally'}}};
const mReview=({comment=mComment,graph=mGraph,expected=mReviewExpected,
  builderClaim=mBuilder,reviewerClaim=mReviewer,noGraphql=false,prState='open'}={})=>
  readMaintenanceReviewWitnessFromClient({
    octokit:{graphql:noGraphql?undefined:async()=>({node:graph}),rest:{
      issues:{getComment:async()=>({data:comment})},
      pulls:{get:async()=>({data:{number:278,state:prState,merged:false,
        head:{sha:expected.reviewedHead,repo:{full_name:'kaicreator-mm/tally'}},
        base:{sha:expected.reviewedBase,ref:'version/v2'}}})},
      repos:{getCommit:async({ref})=>({data:{sha:ref,commit:{
        tree:{sha:ref===expected.reviewedHead?expected.reviewedTree:'f'.repeat(40)}}}}),
        getBranch:async()=>({data:{commit:{sha:expected.sourceSha}}})}}}},
    commentId:mReviewCommentId,expected,builderClaim,reviewerClaim});
test('#219 source-qualified official IssueComment ReviewWitness is read-only and not Task approval',async()=>{
  const got=await mReview();
  assert.equal(got.state,'REVIEW_VERIFIED');
  assert.equal(got.reviewRef,mReviewRef);assert.equal(got.p2,1);
  assert.equal(got.authorized,false);assert.equal(got.taskPass,false);
});
test('#219 review witness refuses missing Claims, same process, same-second edits and forged identity',async()=>{
  for(const opts of [
    {reviewerClaim:{...mReviewer,accepted:false}},
    {reviewerClaim:{...mReviewer,operatorId:mBuilder.operatorId}},
    {reviewerClaim:{...mReviewer,writerId:mBuilder.writerId}},
    {comment:{...mComment,updated_at:'2026-11-13T01:00:01Z'}},
    {graph:{...mGraph,lastEditedAt:'2026-11-13T01:00:00.5Z'}},
    {graph:{...mGraph,fullDatabaseId:'6840000002'}},
    {graph:{...mGraph,id:'IC_wrong'}},
    {graph:{...mGraph,body:'{}'}},
    {graph:{...mGraph,issue:{number:219,repository:{nameWithOwner:'other/repo'}}}},
    {noGraphql:true},{prState:'closed'}
  ]){
    const got=await mReview(opts);
    assert.equal(got.state,'REVIEW_UNKNOWN',JSON.stringify(opts));
    assert.equal(got.reviewed,false);assert.equal(got.taskPass,false);
  }
});
