import test from 'node:test';
import assert from 'node:assert/strict';
import {readUiNativeCatalogFromClient,readUiNativeVersionFromClient}
  from '../src/ui-version-readonly.mjs';

const sha='a'.repeat(40),owner='org',repo='tally',
  versionId='version/v2';
const issue=(n,changes={})=>({
  id:n+100,number:n,title:`Task #${n}`,state:'open',
  updated_at:'2026-11-01T00:00:00Z',
  html_url:`https://github.com/org/tally/issues/${n}`,
  labels:[{name:'type:task'},{name:'state:ready'},{name:'priority:P1'}],
  ...changes,
});
const native=(deps={},branchSha=sha,issueMap={})=>({
  rest:{
    repos:{
      getBranch:async()=>({data:{commit:{sha:branchSha}}}),
      getCommit:async()=>({data:{sha}}),
    },
    issues:{get:async({issue_number})=>({data:issueMap[issue_number]??issue(issue_number)})},
  },
  paginate:{iterator:async function*(_route,{issue_number}){
    yield {data:deps[issue_number]??[]};
  }},
});
const sourceReader=({issueNumber,expectedSourceSha})=>({
  sourceSha:expectedSourceSha,workItemType:'type:task',
  claimStatus:'NOT_VERIFIED',authorized:false,taskPass:false,
  dependencyState:'CLEAR',blockerCount:0,openBlockerCount:0,
  workflowState:'state:ready',issueState:'open',
});
const args=octokit=>({octokit,owner,repo,versionId,expectedSha:sha});
test('pinned version produces exactly one real Host-selected project/version',async()=>{
  const response=await readUiNativeCatalogFromClient(args(native()));
  assert.equal(response.connected,true);
  assert.equal(response.source.sourceSha,sha);
  assert.deepEqual(response.projects.map(p=>p.id),['org/tally']);
  assert.deepEqual(response.versionsByProject['org/tally'].map(v=>v.id),['version/v2']);
});
test('moved GitHub version branch cannot be labelled CURRENT',async()=>{
  const response=await readUiNativeCatalogFromClient(args(native({},'b'.repeat(40))));
  assert.deepEqual(response,{connected:false,reason:'VERSION_BRANCH_NOT_PINNED'});
});
test('unconfigured task roster is UNKNOWN rather than zero tasks',async()=>{
  const r=await readUiNativeVersionFromClient({...args(native()),taskIssueNumbers:[]});
  assert.equal(r.connected,false);
  assert.equal(r.reason,'VERSION_TASK_ROSTER_NOT_CONFIGURED');
});
test('source-qualified native C WorkItem yields unknown Agent/Claim and real Task identity',async()=>{
  const r=await readUiNativeVersionFromClient({...args(native()),
    taskIssueNumbers:[1,2],workItemReader:sourceReader});
  assert.equal(r.connected,true);
  assert.deepEqual(r.tasks.map(x=>x.id),['1','2']);
  assert.equal(r.tasks[0].severity,'P1');
  assert.equal(r.tasks[0].claimState,'NOT_VERIFIED');
  assert.equal(r.tasks[0].reviewState,'UNKNOWN');
  assert.deepEqual(r.dag.edges,[]);
  assert.equal(r.rosterCoverage,'UNVERIFIED');
  assert.equal(r.dag.rosterCoverage,'UNVERIFIED');
  assert.equal(r.dag.nodes[0].executable,false);
});
test('native dependencies are C readback edges, not assistant planned edges',async()=>{
  const dep={number:1,html_url:'https://github.com/org/tally/issues/1',
    state:'closed'};
  const reader=({issueNumber,expectedSourceSha})=>({
    ...sourceReader({issueNumber,expectedSourceSha}),
    // A closed blocker is still a native edge, but source state is CLEAR.
    dependencyState:'CLEAR',
    blockerCount:issueNumber===2?1:0,openBlockerCount:0,
  });
  const r=await readUiNativeVersionFromClient({
    ...args(native({2:[dep]})),taskIssueNumbers:[1,2],workItemReader:reader});
  assert.equal(r.connected,true);
  assert.deepEqual(r.tasks[1].blockedBy,['1']);
  assert.deepEqual(r.dag.edges.map(e=>[e.from,e.to,e.kind]),[['1','2','NATIVE']]);
});
test('out-of-roster dependency cannot be silently rendered as complete graph',async()=>{
  const r=await readUiNativeVersionFromClient({
    ...args(native({2:[{number:99,state:'open',
      html_url:'https://github.com/org/tally/issues/99'}]})),
    taskIssueNumbers:[1,2],workItemReader:sourceReader});
  assert.equal(r.connected,false);
  assert.equal(r.reason,'NATIVE_DEPENDENCY_OUTSIDE_PINNED_ROSTER');
});
test('non-Task, duplicated issue roster or missing native C claim facts are denied',async()=>{
  const notTask=issue(1,{labels:[{name:'type:planning'},{name:'state:ready'}]});
  const r=await readUiNativeVersionFromClient({
    ...args(native({},sha,{1:notTask})),
    taskIssueNumbers:[1],workItemReader:sourceReader});
  assert.equal(r.connected,false);
  assert.equal((await readUiNativeVersionFromClient({
    ...args(native()),taskIssueNumbers:[1,1],workItemReader:sourceReader})
    ).connected,false);
  const fake=await readUiNativeVersionFromClient({
    ...args(native()),taskIssueNumbers:[1],
    workItemReader:()=>({...sourceReader({issueNumber:1,expectedSourceSha:sha}),
      authorized:true})});
  assert.equal(fake.connected,false);
});

test('closed GitHub Issue with state:ready does not render task DONE',async()=>{
  const observedIssue=issue(166,{state:'closed'});
  const result=await readUiNativeVersionFromClient({
    ...args(native({},sha,{166:observedIssue})),taskIssueNumbers:[166],
    workItemReader:({issueNumber,expectedSourceSha})=>({
      ...sourceReader({issueNumber,expectedSourceSha}),issueState:'closed',
    })});
  assert.equal(result.connected,true);
  assert.equal(result.tasks[0].status,'unknown');
  assert.equal(result.tasks[0].claimState,'NOT_VERIFIED');
});
test('state:done label alone never certifies independent closure',async()=>{
  const observedIssue=issue(201,{state:'closed',
    labels:[{name:'type:task'},{name:'state:done'}]});
  const result=await readUiNativeVersionFromClient({
    ...args(native({},sha,{201:observedIssue})),taskIssueNumbers:[201],
    workItemReader:({issueNumber,expectedSourceSha})=>({
      ...sourceReader({issueNumber,expectedSourceSha}),
      workflowState:'state:done',issueState:'closed',
    })});
  assert.equal(result.connected,true);
  assert.equal(result.tasks[0].status,'unknown');
});

test('dependency open/closed state drift with unchanged count is rejected',async()=>{
  const dependency={number:1,id:10001,
    html_url:'https://github.com/org/tally/issues/1',state:'open'};
  const fake=({issueNumber,expectedSourceSha})=>({
    ...sourceReader({issueNumber,expectedSourceSha}),
    dependencyState:issueNumber===2?'CLEAR':'CLEAR',
    blockerCount:issueNumber===2?1:0,openBlockerCount:0,
  });
  const r=await readUiNativeVersionFromClient({
    ...args(native({2:[dependency]})),taskIssueNumbers:[1,2],
    workItemReader:fake});
  assert.equal(r.connected,false);
  assert.equal(r.reason,'NATIVE_DEPENDENCY_COUNT_DRIFT');
});
