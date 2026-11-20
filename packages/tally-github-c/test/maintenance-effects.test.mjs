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
        getBranch:async()=>({data:{commit:{sha:expected.sourceSha}}}),
        compareCommits:async()=>({data:{ahead_by:0,behind_by:0,
          merge_base_commit:{sha:expected.sourceSha}}})}}},
      commentId:mReviewCommentId,expected,builderClaim,reviewerClaim});
test('#219 source-qualified official IssueComment ReviewWitness is read-only and not Task approval',async()=>{
  const got=await mReview();
  assert.equal(got.state,'REVIEW_VERIFIED');
  assert.equal(got.reviewRef,mReviewRef);assert.equal(got.p2,1);
  assert.equal(got.reviewedAt,mComment.created_at,'trusted original GitHub first-post time');
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

// #417 — merged two-parent source qualification. The reviewed base stays the
// pre-merge base; the live source is the official merge commit itself or a
// GitHub-confirmed ahead-only descendant of it. Fixtures mirror the official
// REST shape (pulls.get + repos.getCommit + repos.getBranch +
// repos.compareCommits) exactly as the reader consumes them.
const wBase='2'.repeat(40),wHead='3'.repeat(40),wTree='4'.repeat(40);
const wMerge='5'.repeat(40),wDesc='7'.repeat(40);
const wCommentId=6840000010;
const wRef='https://github.com/kaicreator-mm/tally/issues/219#issuecomment-'+wCommentId;
const wExpected=live=>({issueNumber:219,sourceSha:live,
  candidateDigest:mReviewDigest,reviewedPullNumber:410,
  reviewedHead:wHead,reviewedTree:wTree,reviewedBase:wBase});
const wClaims=live=>({
  builder:{accepted:true,role:'builder',issueNumber:219,sourceSha:live,
    candidateDigest:mReviewDigest,claimRef:'claim:410:builder',
    operatorId:'operator:410:builder',writerId:'writer:410:builder',
    transportActor:'kaicreator-mm'},
  reviewer:{accepted:true,role:'reviewer',issueNumber:219,sourceSha:live,
    candidateDigest:mReviewDigest,claimRef:'claim:410:reviewer',
    operatorId:'operator:410:reviewer',writerId:'writer:410:reviewer',
    transportActor:'kaicreator-mm'}});
const wWitnessBody=live=>({schema:'tally.v2.maintenance-review-witness/1',
  status:'INDEPENDENTLY_VERIFIED',repository:'kaicreator-mm/tally',
  issueNumber:219,...wExpected(live),
  builderClaimRef:'claim:410:builder',reviewerClaimRef:'claim:410:reviewer',
  reviewerOperatorId:'operator:410:reviewer',
  reviewerWriterId:'writer:410:reviewer',p0:0,p1:0,p2:0});
const wCommentFor=live=>({id:wCommentId,node_id:'IC_merged410',
  html_url:wRef,
  issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/219',
  user:{login:'kaicreator-mm'},created_at:'2026-11-14T09:00:00Z',
  updated_at:'2026-11-14T09:00:00Z',body:JSON.stringify(wWitnessBody(live))});
const wGraphFor=live=>({__typename:'IssueComment',id:'IC_merged410',
  fullDatabaseId:String(wCommentId),url:wRef,
  body:JSON.stringify(wWitnessBody(live)),lastEditedAt:null,editor:null,
  issue:{number:219,repository:{nameWithOwner:'kaicreator-mm/tally'}}});
/** Merged-source fixture. Defaults describe the fully valid case where the
 * live source is a strict descendant of the official two-parent merge. */
const wReview=({live=wDesc,prState='closed',merged=true,mergedAt='2026-11-14T10:00:00Z',
  mergeCommitSha=wMerge,parents=[{sha:wBase},{sha:wHead}],headTree=wTree,
  baseSha=wBase,branchRef='version/v2',headRepo='kaicreator-mm/tally',
  liveSha=live,compare={ahead_by:3,behind_by:0,
    merge_base_commit:{sha:wMerge}},comment,graph,noCompare=false,
  compareThrows=false}={})=>
  readMaintenanceReviewWitnessFromClient({
    octokit:{graphql:async()=>({node:graph??wGraphFor(live)}),rest:{
      issues:{getComment:async()=>({data:comment??wCommentFor(live)})},
      pulls:{get:async()=>({data:{number:410,state:prState,merged,
        merged_at:mergedAt,merge_commit_sha:mergeCommitSha,
        head:{sha:wHead,repo:{full_name:headRepo}},
        base:{sha:baseSha,ref:branchRef}}})},
      repos:{getCommit:async({ref})=>{
          if(ref===wHead)return{data:{sha:ref,commit:{tree:{sha:headTree}}}};
          if(ref===wBase)return{data:{sha:ref,commit:{tree:{sha:'0'.repeat(40)}}}};
          if(ref===mergeCommitSha)
            return{data:{sha:ref,commit:{tree:{sha:'8'.repeat(40)}},parents}};
          return{data:{sha:ref,commit:{tree:{sha:'9'.repeat(40)}}}};
        },
        getBranch:async()=>({data:{commit:{sha:liveSha}}}),
        compareCommits:noCompare?undefined:compareThrows?
          async()=>{throw {status:500};}:async()=>({data:compare})}}},
    commentId:wCommentId,expected:wExpected(live),
    builderClaim:wClaims(live).builder,reviewerClaim:wClaims(live).reviewer});
test('#417 officially merged two-parent source qualifies: merge-is-live and ahead-only descendant',async()=>{
  const atMerge=await wReview({live:wMerge,
    compare:{ahead_by:0,behind_by:0,merge_base_commit:{sha:wMerge}}});
  assert.equal(atMerge.state,'REVIEW_VERIFIED');
  assert.equal(atMerge.reason,'MAINTENANCE_SOURCE_REVIEW_WITNESS_VERIFIED');
  assert.equal(atMerge.sourceSha,wMerge);
  assert.equal(atMerge.reviewedHead,wHead);assert.equal(atMerge.p0,0);
  const descendant=await wReview();
  assert.equal(descendant.state,'REVIEW_VERIFIED');
  assert.equal(descendant.sourceSha,wDesc);
  for(const k of ['authorized','admitted','claimAccepted','taskPass','releasePass'])
    assert.equal(descendant[k],false,k);
  assert.equal(Object.isFrozen(descendant),true);
});
test('#417 merged source re-read under a Host fence stays verified while the witness is untouched',async()=>{
  const first=await wReview();
  assert.equal(first.state,'REVIEW_VERIFIED');
  const second=await wReview();
  assert.equal(second.state,'REVIEW_VERIFIED');
  assert.equal(second.reason,first.reason);
});
test('#417 witness edited between Host fence reads fails closed on re-read',async()=>{
  const first=await wReview();
  assert.equal(first.state,'REVIEW_VERIFIED');
  const editedRest=await wReview({comment:{
    ...wCommentFor(wDesc),updated_at:'2026-11-14T09:04:00Z'}});
  assert.equal(editedRest.state,'REVIEW_UNKNOWN');
  assert.equal(editedRest.reason,'MAINTENANCE_REVIEW_COMMENT_UNVERIFIED');
  const editedGraph=await wReview({graph:{
    ...wGraphFor(wDesc),lastEditedAt:'2026-11-14T09:04:00.500Z',
    editor:{login:'someone-else'}}});
  assert.equal(editedGraph.state,'REVIEW_UNKNOWN');
  assert.equal(editedGraph.reason,'MAINTENANCE_REVIEW_EDIT_PROVENANCE_UNVERIFIED');
  const rewrittenBody=JSON.stringify(wWitnessBody(wDesc)).replace('{','{ ');
  const rewritten=await wReview({comment:{
    ...wCommentFor(wDesc),body:rewrittenBody},
    graph:{...wGraphFor(wDesc),body:rewrittenBody}});
  assert.equal(rewritten.state,'REVIEW_UNKNOWN');
  assert.equal(rewritten.reason,'MAINTENANCE_REVIEW_NONCANONICAL');
  const outOfRangeBody=JSON.stringify({...wWitnessBody(wDesc),p2:99});
  const outOfRange=await wReview({comment:{
    ...wCommentFor(wDesc),body:outOfRangeBody},
    graph:{...wGraphFor(wDesc),body:outOfRangeBody}});
  assert.equal(outOfRange.state,'REVIEW_UNKNOWN');
  assert.equal(outOfRange.reason,'MAINTENANCE_REVIEW_WITNESS_MISMATCH');
  const mismatchedBody=JSON.stringify({
    ...wWitnessBody(wDesc),sourceSha:'0'.repeat(40)});
  const mismatched=await wReview({comment:{
    ...wCommentFor(wDesc),body:mismatchedBody},
    graph:{...wGraphFor(wDesc),body:mismatchedBody}});
  assert.equal(mismatched.state,'REVIEW_UNKNOWN');
  assert.equal(mismatched.reason,'MAINTENANCE_REVIEW_WITNESS_MISMATCH');
});
test('#417 merged source rejects squash/rebase and any non-two-parent merge',async()=>{
  for(const [opts,reason] of [
    [{parents:[{sha:wBase}]},'MAINTENANCE_REVIEW_SOURCE_NOT_TWO_PARENT_MERGE'],
    [{parents:[{sha:wDesc}]},'MAINTENANCE_REVIEW_SOURCE_NOT_TWO_PARENT_MERGE'],
    [{parents:[{sha:wBase},{sha:wHead},{sha:wMerge}]},
      'MAINTENANCE_REVIEW_SOURCE_NOT_TWO_PARENT_MERGE'],
    [{parents:[]},'MAINTENANCE_REVIEW_SOURCE_NOT_TWO_PARENT_MERGE'],
    [{parents:[null,null]},'MAINTENANCE_REVIEW_MERGE_PARENT_ORDER_INVALID'],
  ]){
    const got=await wReview(opts);
    assert.equal(got.state,'REVIEW_UNKNOWN',JSON.stringify(opts));
    assert.equal(got.reason,reason,JSON.stringify(opts));
    assert.equal(got.reviewed,false);assert.equal(got.taskPass,false);
  }
});
test('#417 merged source rejects wrong parent order and identity',async()=>{
  for(const [opts,reason] of [
    [{parents:[{sha:wHead},{sha:wBase}]},
      'MAINTENANCE_REVIEW_MERGE_PARENT_ORDER_INVALID'],
    [{parents:[{sha:wBase},{sha:wMerge}]},
      'MAINTENANCE_REVIEW_MERGE_PARENT_ORDER_INVALID'],
    [{parents:[{sha:'1'.repeat(40)},{sha:wHead}]},
      'MAINTENANCE_REVIEW_MERGE_PARENT_ORDER_INVALID'],
    [{parents:[{sha:wBase},'not-an-object']},
      'MAINTENANCE_REVIEW_MERGE_PARENT_ORDER_INVALID'],
  ]){
    const got=await wReview(opts);
    assert.equal(got.state,'REVIEW_UNKNOWN',JSON.stringify(opts));
    assert.equal(got.reason,reason,JSON.stringify(opts));
  }
});
test('#417 merged source rejects unavailable or malformed merge commit',async()=>{
  const absent=await wReview({mergeCommitSha:null});
  assert.equal(absent.state,'REVIEW_UNKNOWN');
  assert.equal(absent.reason,'MAINTENANCE_REVIEW_MERGE_COMMIT_UNAVAILABLE');
  const malformed=await wReview({mergeCommitSha:'release-tag-object'});
  assert.equal(malformed.reason,'MAINTENANCE_REVIEW_MERGE_COMMIT_UNAVAILABLE');
});
test('#417 merged source rejects closed-unmerged and inconsistent merged flags',async()=>{
  const closed=await wReview({merged:false,mergedAt:null});
  assert.equal(closed.state,'REVIEW_UNKNOWN');
  assert.equal(closed.reason,'MAINTENANCE_REVIEW_SOURCE_NOT_MERGED');
  const openMerged=await wReview({prState:'open'});
  assert.equal(openMerged.state,'REVIEW_UNKNOWN');
  assert.equal(openMerged.reason,'MAINTENANCE_REVIEW_SOURCE_NOT_MERGED');
});
test('#417 merged source rejects wrong repo, branch, base, head tree and stale live',async()=>{
  for(const [opts,reason] of [
    [{headRepo:'other/repo'},'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{branchRef:'main'},'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{baseSha:'1'.repeat(40)},'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{headTree:'a'.repeat(40)},'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{liveSha:wMerge},'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
  ]){
    const got=await wReview(opts);
    assert.equal(got.state,'REVIEW_UNKNOWN',JSON.stringify(opts));
    assert.equal(got.reason,reason,JSON.stringify(opts));
  }
});
test('#417 merged source rejects stale, divergent or unverifiable ancestry',async()=>{
  for(const [opts,reason] of [
    [{compare:{ahead_by:0,behind_by:2,merge_base_commit:{sha:wMerge}}},
      'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{compare:{ahead_by:4,behind_by:0,
      merge_base_commit:{sha:'a'.repeat(40)}}},
      'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{compare:{ahead_by:0,behind_by:0,
      merge_base_commit:{sha:'a'.repeat(40)}}},
      'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{compare:{ahead_by:'many',behind_by:0,
      merge_base_commit:{sha:wMerge}}},
      'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT'],
    [{compare:{ahead_by:1,behind_by:0,merge_base_commit:{sha:wMerge}},
      noCompare:true},'MAINTENANCE_REVIEW_INPUT_INVALID'],
  ]){
    const got=await wReview(opts);
    assert.equal(got.state,'REVIEW_UNKNOWN',JSON.stringify(opts));
    assert.equal(got.reason,reason,JSON.stringify(opts));
  }
  const thrown=await wReview({compareThrows:true});
  assert.equal(thrown.state,'REVIEW_UNKNOWN');
  assert.equal(thrown.reason,'MAINTENANCE_REVIEW_GITHUB_READ_UNKNOWN');
});
test('#417 OPEN current-base path stays green alongside merged qualification',async()=>{
  const open=await mReview();
  assert.equal(open.state,'REVIEW_VERIFIED');
  // OPEN with a moved live source (claims/witness consistently re-bound to
  // the new source) must fail the current-base requirement, not identity.
  const staleSha='1'.repeat(40);
  const staleBody=JSON.stringify({...mBody,sourceSha:staleSha});
  const openStaleBase=await mReview({
    expected:{...mReviewExpected,sourceSha:staleSha},
    builderClaim:{...mBuilder,sourceSha:staleSha},
    reviewerClaim:{...mReviewer,sourceSha:staleSha},
    comment:{...mComment,body:staleBody},
    graph:{...mGraph,body:staleBody}});
  assert.equal(openStaleBase.state,'REVIEW_UNKNOWN');
  assert.equal(openStaleBase.reason,'MAINTENANCE_REVIEW_SOURCE_NOT_CURRENT');
  // OPEN fixture keeps refusing a closed PR (not merged).
  const closed=await mReview({prState:'closed'});
  assert.equal(closed.state,'REVIEW_UNKNOWN');
  assert.equal(closed.reason,'MAINTENANCE_REVIEW_SOURCE_NOT_MERGED');
});
