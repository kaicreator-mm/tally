import test from 'node:test';
import assert from 'node:assert/strict';
import {projectIntakeReadModel,readProjectIntakeFromClient} from '../src/project-intake.mjs';

const sha='a'.repeat(40);
const repository={owner:'kaicreator-mm',repo:'tally',fullName:'kaicreator-mm/tally',
  defaultBranch:'version/v2',visibility:'private'};
const repoRest={full_name:'kaicreator-mm/tally',default_branch:'version/v2',
  visibility:'private'};
const rawSource={number:157,state:'closed',title:'docs(v2 Product): scope integration',
  url:'https://github.com/kaicreator-mm/tally/pull/157',isPullRequest:true,merged:true};
const rawProject={number:7,title:'Tally v2 Intake',closed:false,
  url:'https://github.com/kaicreator-mm/tally/projects/7',itemCount:2,
  items:[{issueNumber:210,title:'[v2][P0][J01–J03] Intake workbench',state:'open'},
    {issueNumber:157,title:'docs(v2 Product)',state:'closed'}]};
const observation=()=>({repository,observedAt:'2026-10-31T10:00:00.000Z',
  sources:[rawSource],projects:[rawProject]});

test('canonical observation projects to bounded read model without authority',()=>{
  const model=projectIntakeReadModel(observation());
  assert.equal(model.status,'PROJECT_INTAKE_OBSERVED');
  assert.equal(model.reason,'CANONICAL_GITHUB_FACTS_OBSERVED');
  assert.equal(model.sourceQualified,true);
  assert.equal(model.currentness,'OBSERVED_AT_QUERY_COMPLETION');
  assert.equal(model.sources[0].number,157);assert.equal(model.sources[0].merged,true);
  assert.equal(model.projects[0].items[0].issueNumber,210);
  assert.equal(model.authorized,false);assert.equal(model.admitted,false);
  assert.equal(model.taskPass,false);assert.equal(model.releasePass,false);
  assert.equal(Object.isFrozen(model),true);assert.equal(Object.isFrozen(model.sources),true);
});

test('malformed observations fail closed with bounded reasons',()=>{
  for(const [value,reason] of [
    [null,'INVALID_PROJECT_INTAKE_REQUEST'],
    [{...observation(),repository:{...repository,fullName:'other/tally'}},'PROJECT_INTAKE_REPOSITORY_UNVERIFIED'],
    [{...observation(),repository:{...repository,visibility:'internal'}},'PROJECT_INTAKE_REPOSITORY_UNVERIFIED'],
    [{...observation(),observedAt:'not-a-time'},'PROJECT_INTAKE_OBSERVATION_TIME_UNAVAILABLE'],
    [{...observation(),sources:[{...rawSource,number:0}]},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
    [{...observation(),sources:[{...rawSource,url:'https://evil.test/157'}]},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
    [{...observation(),projects:[{...rawProject,url:'https://github.com/kaicreator-mm/tally/x'}]},'PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED'],
    [{...observation(),projects:[{...rawProject,itemCount:99999}]},'PROJECT_INTAKE_PROJECT_FACTS_UNVERIFIED'],
    [{...observation(),sources:Array.from({length:17},(_,i)=>({...rawSource,number:i+1}))},'PROJECT_INTAKE_SOURCE_FACTS_UNVERIFIED'],
  ]){
    const model=projectIntakeReadModel(value);
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,reason);
    assert.equal(model.sourceQualified,false);assert.equal(model.taskPass,false);
  }
});

test('cross-repo readback and hostile clients refuse before any fact is trusted',async()=>{
  for(const client of [
    {rest:{repos:{get:async()=>({data:{...repoRest,full_name:'other/tally'}})}},graphql:async()=>({})},
    {rest:{repos:{get:async()=>({data:{...repoRest,visibility:'internal'}})}},graphql:async()=>({})},
  ]){
    const model=await readProjectIntakeFromClient({octokit:client,owner:'kaicreator-mm',
      repo:'tally',expectedSourceSha:sha,sources:[{number:157}]});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,'PROJECT_INTAKE_REPOSITORY_UNVERIFIED');
  }
});

test('pinned sources and exact project-title readback become observed facts',async()=>{
  const calls=[];
  const issuesGet=async arg=>{
    calls.push('issue:'+arg.issue_number);
    return {data:{number:arg.issue_number,state:'closed',
      title:'docs(v2 Product): scope integration',
      html_url:`https://github.com/kaicreator-mm/tally/pull/${arg.issue_number}`,
      pull_request:{merged_at:'2026-09-01T00:00:00Z'}}};
  };
  const projectsData={repository:{nameWithOwner:'kaicreator-mm/tally',projectsV2:{nodes:[
    {number:7,title:'Tally v2 Intake',url:'https://github.com/kaicreator-mm/tally/projects/7',
      closed:false,
      items:{totalCount:2,nodes:[
        {content:{__typename:'Issue',number:210,state:'open',title:'Intake workbench'}},
        {content:{__typename:'DraftIssue',title:'draft (no issue)'}}]}}]}}};
  const octokit={
    rest:{
      repos:{get:async()=>{calls.push('repos');return {data:repoRest};}},
      issues:{get:issuesGet}},
    graphql:async(query,vars)=>{
      calls.push('graphql');
      assert.match(query,/projectsV2/);assert.equal(vars.owner,'kaicreator-mm');
      return projectsData}};
  const model=await readProjectIntakeFromClient({octokit,owner:'kaicreator-mm',repo:'tally',
    expectedSourceSha:sha,sources:[{number:157},{number:158}],projectTitle:'Tally v2 Intake'});
  assert.equal(model.status,'PROJECT_INTAKE_OBSERVED');
  assert.deepEqual(calls,['repos','issue:157','issue:158','graphql']);
  assert.equal(model.projects.length,1);assert.equal(model.projects[0].itemCount,2);
  assert.equal(model.projects[0].items.length,1);assert.equal(model.projects[0].items[0].issueNumber,210);
});

test('transport failures map to bounded reasons and never fabricate observation time',async()=>{
  for(const [client,reason] of [
    [{rest:{repos:{get:async()=>{throw {status:403};}}}},'GITHUB_ACCESS_DENIED'],
    [{rest:{repos:{get:async()=>{throw {status:404};}}}},'GITHUB_RESOURCE_NOT_FOUND'],
    [{rest:{repos:{get:async()=>{throw {status:429};}}}},'GITHUB_RATE_LIMITED'],
    [{rest:{repos:{get:async()=>({data:repoRest})},issues:{get:async()=>{throw new Error('x');}}}},'PROJECT_INTAKE_READ_FAILED'],
    [{rest:{repos:{get:async()=>({data:repoRest})}}},'PROJECT_INTAKE_READ_FAILED'],
  ]){
    const model=await readProjectIntakeFromClient({octokit:client,owner:'kaicreator-mm',
      repo:'tally',expectedSourceSha:sha,sources:[{number:157}],projectTitle:'Tally v2 Intake'});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,reason);
    assert.equal(model.observedAt,null);assert.equal(model.taskPass,false);
  }
});

test('invalid requests refuse without any client call',async()=>{
  let called=false;
  const octokit={rest:{repos:{get:async()=>{called=true;return {data:repoRest};}}}};
  for(const args of [
    {octokit:undefined},{octokit,owner:'bad name'},{octokit,owner:'kaicreator-mm',repo:'tally',
      expectedSourceSha:'short'},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,
      sources:[{number:0}]},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,
      projectTitle:''},{octokit,owner:'kaicreator-mm',repo:'tally',expectedSourceSha:sha,limit:1000},
  ]){
    const model=await readProjectIntakeFromClient({owner:'kaicreator-mm',repo:'tally',
      expectedSourceSha:sha,...args});
    assert.equal(model.status,'UNKNOWN');assert.equal(model.reason,'INVALID_PROJECT_INTAKE_REQUEST');
  }
  assert.equal(called,false);
});


import {readProductReviewWitnessFromClient} from '../src/project-intake.mjs';
const witness312Sha='a'.repeat(40),witness312Blob='b'.repeat(40);
const witness312Digest='c'.repeat(64),witness312Set='d'.repeat(64);
const witness312Expected={
  issueNumber:312,sourceSha:witness312Sha,
  draftRef:'draft:sha256:'+witness312Digest,productDigest:witness312Digest,
  taskPackBlobSha:witness312Blob,writeSetSha256:witness312Set,
  reviewedHead:'e'.repeat(40),reviewedTree:'f'.repeat(40),
  reviewedBase:witness312Sha};
const witness312Builder={accepted:true,issueNumber:312,role:'builder',
  sourceSha:witness312Sha,taskPackBlobSha:witness312Blob,
  writeSetSha256:witness312Set,claimRef:'claim:312:builder',
  operatorId:'host:builder',writerId:'writer:builder',transportActor:'kaicreator-mm'};
const witness312Reviewer={...witness312Builder,role:'reviewer',
  claimRef:'claim:312:reviewer',operatorId:'host:reviewer',writerId:'writer:reviewer'};
const witness312Ref='https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345';
const witness312Body={
  schema:'tally.v2.product-review-witness/2',status:'INDEPENDENTLY_VERIFIED',
  repository:'kaicreator-mm/tally',issueNumber:312,reviewedPullNumber:401,
  draftRef:witness312Expected.draftRef,productDigest:witness312Digest,
  sourceSha:witness312Sha,taskPackBlobSha:witness312Blob,writeSetSha256:witness312Set,
  reviewedHead:witness312Expected.reviewedHead,
  reviewedTree:witness312Expected.reviewedTree,
  reviewedBase:witness312Expected.reviewedBase,
  builderClaimRef:witness312Builder.claimRef,
  reviewerClaimRef:witness312Reviewer.claimRef,
  reviewerOperatorId:witness312Reviewer.operatorId,
  reviewerWriterId:witness312Reviewer.writerId,p0:0,p1:0,p2:1};
const witness312Comment={id:12345,node_id:'IC_witness12345',html_url:witness312Ref,
  issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/312',
  user:{login:'kaicreator-mm'},created_at:'2026-11-12T12:00:00Z',
  updated_at:'2026-11-12T12:00:00Z',body:JSON.stringify(witness312Body)};
const witness312Pr={number:401,state:'open',merged:false,
  head:{sha:witness312Expected.reviewedHead,
    repo:{full_name:'kaicreator-mm/tally'}},
  base:{sha:witness312Sha,ref:'version/v2',
    repo:{full_name:'kaicreator-mm/tally'}}};
const witness312Head={sha:witness312Expected.reviewedHead,
  commit:{tree:{sha:witness312Expected.reviewedTree}}};
const witness312Base={sha:witness312Sha,commit:{tree:{sha:'2'.repeat(40)}}};
const witness312Read=({comment=witness312Comment,expected=witness312Expected,
  builderClaim=witness312Builder,reviewerClaim=witness312Reviewer,fail,
  call,pr=witness312Pr,head=witness312Head,base=witness312Base,
  liveSha=witness312Sha,commitError,graphOverride,graphError,noGraphql=false}={})=>readProductReviewWitnessFromClient({
  octokit:{graphql:noGraphql?undefined:async(query,{id})=>{
    if(graphError)throw {status:graphError};
    assert.match(query,/lastEditedAt/);
    assert.equal(id,comment.node_id);
    return {node:graphOverride??{
      __typename:'IssueComment',databaseId:comment.id,url:comment.html_url,
      body:comment.body,lastEditedAt:null,editor:null,
      issue:{number:312,repository:{nameWithOwner:'kaicreator-mm/tally'}}}};
  },rest:{
    issues:{getComment:async()=>{
      if(call)call();
      if(fail)throw {status:fail};
      return {data:comment};}},
    pulls:{get:async()=>({data:pr})},
    repos:{
      getCommit:async({ref})=>{
        if(commitError)throw {status:commitError};
        return {data:ref===expected.reviewedHead?head:base};},
      getBranch:async()=>({data:{commit:{sha:liveSha}}})}}},
  commentId:12345,expected,builderClaim,reviewerClaim});

test('#312 canonical unedited review with same GitHub login but independent processes qualifies only source',async()=>{
  const got=await witness312Read();
  assert.equal(got.status,'INDEPENDENTLY_VERIFIED');
  assert.equal(got.independentlyVerified,true);
  assert.equal(got.reviewRef,witness312Ref);
  assert.equal(got.p2,1);
  assert.equal(got.reviewerLogin,'kaicreator-mm');
  assert.equal(got.taskPass,false);assert.equal(got.releasePass,false);
});


test('#312 same-second edited comment is refused by official GraphQL edit provenance',async()=>{
  const baseGraph={
    __typename:'IssueComment',databaseId:12345,url:witness312Ref,
    body:witness312Comment.body,lastEditedAt:null,editor:null,
    issue:{number:312,repository:{nameWithOwner:'kaicreator-mm/tally'}}};
  for(const mutation of [
    {lastEditedAt:'2026-11-12T12:00:00.750Z',editor:{login:'kaicreator-mm'}},
    {lastEditedAt:'2026-11-12T12:00:00Z'},
    {body:'{}'},
    {databaseId:12346},
    {url:'https://github.com/other/repo/issues/312#issuecomment-12345'},
    {issue:{number:312,repository:{nameWithOwner:'other/repo'}}}
  ]){
    const got=await witness312Read({
      graphOverride:{...baseGraph,...mutation},
      comment:{...witness312Comment,updated_at:witness312Comment.created_at}});
    assert.equal(got.status,'UNKNOWN',JSON.stringify(mutation));
    assert.equal(got.reason,'PRODUCT_REVIEW_COMMENT_EDIT_PROVENANCE_UNVERIFIED');
    assert.equal(got.taskPass,false);
  }
  const noGraph=await witness312Read({noGraphql:true});
  assert.equal(noGraph.reason,'PRODUCT_REVIEW_INPUT_INVALID');
  const inaccessible=await witness312Read({graphError:403});
  assert.equal(inaccessible.reason,'PRODUCT_REVIEW_COMMENT_EDIT_PROVENANCE_UNAVAILABLE');
  const noNode=await witness312Read({comment:{...witness312Comment,node_id:undefined}});
  assert.equal(noNode.reason,'PRODUCT_REVIEW_COMMENT_EDIT_PROVENANCE_UNAVAILABLE');
});

test('#312 missing/fake Claims and shared operator or writer fail before GitHub HTTP',async()=>{
  for(const [builderClaim,reviewerClaim,reason] of [
    [witness312Builder,{...witness312Reviewer,accepted:false},'PRODUCT_REVIEW_CLAIM_UNVERIFIED'],
    [witness312Builder,{...witness312Reviewer,operatorId:witness312Builder.operatorId},'PRODUCT_REVIEW_SAME_PROCESS'],
    [witness312Builder,{...witness312Reviewer,writerId:witness312Builder.writerId},'PRODUCT_REVIEW_SAME_PROCESS'],
    [witness312Builder,{...witness312Reviewer,role:'builder'},'PRODUCT_REVIEW_CLAIM_UNVERIFIED'],
    [{...witness312Builder,taskPackBlobSha:'9'.repeat(40)},witness312Reviewer,'PRODUCT_REVIEW_CLAIM_UNVERIFIED']
  ]){
    let reads=0;
    const got=await witness312Read({builderClaim,reviewerClaim,call:()=>reads++});
    assert.equal(got.status,'UNKNOWN');assert.equal(got.reason,reason);
    assert.equal(reads,0);assert.equal(got.taskPass,false);
  }
});

test('#312 forged, edited, misbound and non-PASS witness fails closed',async()=>{
  const bad=[
    [{...witness312Comment,updated_at:'2026-11-12T12:00:01Z'},'PRODUCT_REVIEW_COMMENT_UNVERIFIED'],
    [{...witness312Comment,user:{login:'other'}},'PRODUCT_REVIEW_COMMENT_UNVERIFIED'],
    [{...witness312Comment,html_url:'https://github.com/other/repo'},'PRODUCT_REVIEW_COMMENT_UNVERIFIED'],
    [{...witness312Comment,body:JSON.stringify({...witness312Body,p1:1})},'PRODUCT_REVIEW_FINDINGS_BLOCKED'],
    [{...witness312Comment,body:JSON.stringify({...witness312Body,reviewRef:'fake'})},'PRODUCT_REVIEW_SCHEMA_INVALID'],
    [{...witness312Comment,issue_url:'https://api.github.com/repos/other/repo/issues/312'},'PRODUCT_REVIEW_COMMENT_UNVERIFIED'],
    [{...witness312Comment,body:JSON.stringify({...witness312Body,sourceSha:'8'.repeat(40)})},'PRODUCT_REVIEW_WITNESS_MISMATCH'],
    [{...witness312Comment,body:JSON.stringify({...witness312Body,unexpected:true})},'PRODUCT_REVIEW_SCHEMA_INVALID'],
    [{...witness312Comment,body:'not valid JSON'},'PRODUCT_REVIEW_JSON_NONCANONICAL'],
    [{...witness312Comment,body:'{"a":1,"a":2}'},'PRODUCT_REVIEW_JSON_NONCANONICAL']
  ];
  for(const [comment,reason] of bad){
    const got=await witness312Read({comment});
    assert.equal(got.status,'UNKNOWN');assert.equal(got.reason,reason);
    assert.equal(got.taskPass,false);
  }
});

test('#312 source/Pack/FSW/HEAD/Tree/Base drift and Github denial refuse',async()=>{
  for(const key of ['sourceSha','taskPackBlobSha','writeSetSha256',
    'reviewedHead','reviewedTree','reviewedBase']){
    const value=key==='writeSetSha256'?'9'.repeat(64):'9'.repeat(40);
    const got=await witness312Read({expected:{...witness312Expected,[key]:value}});
    assert.equal(got.status,'UNKNOWN',key);
    assert.equal(got.taskPass,false,key);
  }
  const access=await witness312Read({fail:403});
  assert.equal(access.status,'UNKNOWN');
  assert.equal(access.reason,'GITHUB_ACCESS_DENIED');
});


test('#312 immutable first POST then server-assigned GET comment ref is derivable without edits',async()=>{
  // The published payload has no comment ID or self-ref; GitHub assigns it.
  const payload=JSON.stringify(witness312Body);
  assert.equal(JSON.parse(payload).reviewRef,undefined);
  let posted=null,assigned=null,reads=0;
  const transport={graphql:async(query,{id})=>({
    node:{__typename:'IssueComment',databaseId:posted.id,url:posted.html_url,
      body:posted.body,lastEditedAt:null,editor:null,
      issue:{number:312,repository:{nameWithOwner:'kaicreator-mm/tally'}}}}),
    rest:{
    issues:{
      createComment:async ({owner,repo,issue_number,body})=>{
        assert.equal(owner,'kaicreator-mm');assert.equal(repo,'tally');
        assert.equal(issue_number,312);assert.equal(body,payload);
        posted={...witness312Comment,id:89921,node_id:'IC_witness89921',
          html_url:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-89921',
          body};
        return {data:posted};},
      getComment:async ({comment_id})=>{reads++;assert.equal(comment_id,89921);
        return {data:posted};}},
    pulls:{get:async({pull_number})=>{
      assert.equal(pull_number,401);return {data:witness312Pr};}},
    repos:{
      getCommit:async({ref})=>({data:ref===witness312Expected.reviewedHead?
        witness312Head:witness312Base}),
      getBranch:async()=>({data:{commit:{sha:witness312Sha}}})}}};
  const delivered=await transport.rest.issues.createComment({
    owner:'kaicreator-mm',repo:'tally',issue_number:312,body:payload});
  assigned=delivered.data.id;
  const verified=await readProductReviewWitnessFromClient({
    octokit:transport,commentId:assigned,expected:witness312Expected,
    builderClaim:witness312Builder,reviewerClaim:witness312Reviewer});
  assert.equal(reads,1);
  assert.equal(verified.status,'INDEPENDENTLY_VERIFIED');
  assert.equal(verified.reviewRef,delivered.data.html_url);
  assert.equal(JSON.parse(posted.body).reviewRef,undefined);
  assert.equal(verified.authorized,false);assert.equal(verified.taskPass,false);
});

test('#312 GitHub PR/commit/tree/base/branch API truth outranks matching body and caller',async()=>{
  for(const [opts,reason] of [
    [{pr:{...witness312Pr,number:402}},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{pr:{...witness312Pr,state:'closed'}},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{pr:{...witness312Pr,head:{...witness312Pr.head,sha:'7'.repeat(40)}}},
      'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{pr:{...witness312Pr,head:{...witness312Pr.head,
      repo:{full_name:'attacker/repo'}}}},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{pr:{...witness312Pr,base:{...witness312Pr.base,sha:'7'.repeat(40)}}},
      'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{pr:{...witness312Pr,base:{...witness312Pr.base,ref:'main'}}},
      'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{head:{...witness312Head,commit:{tree:{sha:'8'.repeat(40)}}}},
      'PRODUCT_REVIEW_GIT_OBJECT_MISMATCH'],
    [{base:{...witness312Base,commit:{tree:{sha:'not-a-sha'}}}},
      'PRODUCT_REVIEW_GIT_OBJECT_MISMATCH'],
    [{liveSha:'7'.repeat(40)},'PRODUCT_REVIEW_VERSION_SOURCE_MOVED'],
    [{commitError:403},'GITHUB_ACCESS_DENIED']
  ]){
    const actual=await witness312Read(opts);
    assert.equal(actual.status,'UNKNOWN',reason);
    assert.equal(actual.reason,reason);
    assert.equal(actual.taskPass,false);
  }
  const v1=await witness312Read({comment:{...witness312Comment,
    body:JSON.stringify({...witness312Body,
      schema:'tally.v2.product-review-witness/1'})}});
  assert.equal(v1.status,'UNKNOWN');
  assert.equal(v1.reason,'PRODUCT_REVIEW_WITNESS_MISMATCH');
});


// #392 source-only sequencing: a legally reviewed PR is merged BEFORE the
// new current 51/37 Pack, accepted #312 Claims and true Host Freeze. Therefore
// an original unedited Reviewer Witness must be able to attest an immutable
// merged source via official GitHub merge parents/current ancestry, not just
// a still-open PR. No caller-controlled stale SHA or synthetic Claim accepted.
const merged312Sha='6'.repeat(40),later312Sha='9'.repeat(40);
async function merged312Read({
  sourceSha=merged312Sha,mergeSha=merged312Sha,
  mergeParents=[witness312Sha,witness312Expected.reviewedHead],
  mergeTree='7'.repeat(40),mergedAt='2026-11-13T01:00:00Z',
  state='closed',merged=true,compareStatus='ahead',
  behindBy=0,aheadBy=1,mergeBaseSha=mergeSha,withCompare=true,
  branchSha=sourceSha,prOverride={},commitDenied=false
}={}){
  const expected={...witness312Expected,sourceSha};
  const builderClaim={...witness312Builder,sourceSha};
  const reviewerClaim={...witness312Reviewer,sourceSha};
  const comment={...witness312Comment,body:JSON.stringify({
    ...witness312Body,sourceSha})};
  const pr={...witness312Pr,state,merged,merged_at:mergedAt,
    merge_commit_sha:mergeSha,...prOverride};
  const mergeCommit={sha:mergeSha,commit:{tree:{sha:mergeTree}},
    parents:mergeParents.map(sha=>({sha}))};
  const compareData={status:compareStatus,behind_by:behindBy,
    ahead_by:aheadBy,merge_base_commit:{sha:mergeBaseSha}};
  const repos={
    getCommit:async({ref})=>{
      if(commitDenied)throw {status:403};
      return {data:ref===witness312Expected.reviewedHead?witness312Head:
        ref===witness312Sha?witness312Base:mergeCommit};},
    getBranch:async()=>({data:{commit:{sha:branchSha}}})};
  if(withCompare)repos.compareCommits=async({base,head})=>{
    assert.equal(base,mergeSha);assert.equal(head,sourceSha);
    return {data:compareData};};
  return readProductReviewWitnessFromClient({octokit:{rest:{
    issues:{getComment:async()=>({data:comment})},
    pulls:{get:async()=>({data:pr})},repos}},
    expected,commentId:12345,builderClaim,reviewerClaim});
}

test('#312 real-style merged source provenance: 2-parent git merge qualifies current review, never release',async()=>{
  const got=await merged312Read();
  assert.equal(got.status,'INDEPENDENTLY_VERIFIED');
  assert.equal(got.sourceProvenance,'MERGED_PR_ANCESTRY');
  assert.equal(got.mergeCommitSha,merged312Sha);
  assert.equal(got.sourceSha,merged312Sha);
  assert.equal(got.reviewedBase,witness312Sha);
  assert.equal(got.reviewedHead,witness312Expected.reviewedHead);
  assert.equal(got.taskPass,false);assert.equal(got.releasePass,false);
  const later=await merged312Read({sourceSha:later312Sha});
  assert.equal(later.status,'INDEPENDENTLY_VERIFIED');
  assert.equal(later.sourceProvenance,'MERGED_PR_ANCESTRY');
  assert.equal(later.sourceSha,later312Sha);
});

test('#312 wrong merge provenance, squash/rebase and stale branch cannot qualify current Host',async()=>{
  const cases=[
    [{mergeParents:[witness312Expected.reviewedHead,witness312Sha]},
      'PRODUCT_REVIEW_GIT_MERGE_PROVENANCE_MISMATCH'],
    [{mergeParents:[witness312Sha]},
      'PRODUCT_REVIEW_GIT_MERGE_PROVENANCE_MISMATCH'],
    [{mergeParents:[witness312Sha,'8'.repeat(40)]},
      'PRODUCT_REVIEW_GIT_MERGE_PROVENANCE_MISMATCH'],
    [{mergeTree:'bad'},'PRODUCT_REVIEW_GIT_MERGE_PROVENANCE_MISMATCH'],
    [{mergeSha:'bad'},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{mergedAt:null},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{state:'closed',merged:false},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{prOverride:{base:{...witness312Pr.base,
      repo:{full_name:'evil/fork'}}}},'PRODUCT_REVIEW_GIT_PR_NOT_VERIFIED'],
    [{sourceSha:later312Sha,compareStatus:'diverged'},
      'PRODUCT_REVIEW_GIT_ANCESTRY_UNVERIFIED'],
    [{sourceSha:later312Sha,behindBy:1},
      'PRODUCT_REVIEW_GIT_ANCESTRY_UNVERIFIED'],
    [{sourceSha:later312Sha,mergeBaseSha:'8'.repeat(40)},
      'PRODUCT_REVIEW_GIT_ANCESTRY_UNVERIFIED'],
    [{sourceSha:later312Sha,withCompare:false},
      'PRODUCT_REVIEW_GIT_ANCESTRY_UNVERIFIED'],
    [{sourceSha:later312Sha,branchSha:'8'.repeat(40)},
      'PRODUCT_REVIEW_VERSION_SOURCE_MOVED'],
    [{commitDenied:true},'GITHUB_ACCESS_DENIED'],
  ];
  for(const [options,reason] of cases){
    const got=await merged312Read(options);
    assert.equal(got.status,'UNKNOWN',JSON.stringify(options));
    assert.equal(got.reason,reason,JSON.stringify(options));
    assert.equal(got.taskPass,false);
  }
});
