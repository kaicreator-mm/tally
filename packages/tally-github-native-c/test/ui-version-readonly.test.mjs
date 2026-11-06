import test from 'node:test';
import assert from 'node:assert/strict';
import {readUiNativeCatalogFromClient,readUiNativeVersionFromClient,
  readUiNativeRequirementsFromClient,readUiNativeQaFromClient}
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
    issues:{
      get:async({issue_number})=>({
        data:issueMap[issue_number]??issue(issue_number)}),
    },
  },
  paginate:{iterator:async function*(route,params={}){
    if(route==='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by'){
      yield {data:deps[params.issue_number]??[]};
      return;
    }
    throw new Error('UNEXPECTED_PAGINATION_ROUTE');
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
  assert.equal(r.tasks[1].dependencyState,'CLEAR');
  assert.equal(r.tasks[1].openBlockerCount,0);
  assert.deepEqual(r.dag.edges.map(e=>[e.from,e.to,e.kind]),[['1','2','NATIVE']]);
});
test('open GitHub dependency is source-visible as BLOCKED, not a CAS claim',async()=>{
  const dep={number:1,html_url:'https://github.com/org/tally/issues/1',
    state:'open'};
  const reader=({issueNumber,expectedSourceSha})=>({
    ...sourceReader({issueNumber,expectedSourceSha}),
    dependencyState:issueNumber===2?'BLOCKED':'CLEAR',
    blockerCount:issueNumber===2?1:0,
    openBlockerCount:issueNumber===2?1:0
  });
  const r=await readUiNativeVersionFromClient({
    ...args(native({2:[dep]})),taskIssueNumbers:[1,2],workItemReader:reader});
  assert.equal(r.connected,true);
  assert.equal(r.tasks[1].dependencyState,'BLOCKED');
  assert.equal(r.tasks[1].openBlockerCount,1);
  assert.equal(r.tasks[1].claimState,'NOT_VERIFIED');
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
    ...args(native({},sha,{})),taskIssueNumbers:[1,1],workItemReader:sourceReader})
    ).connected,false);
  const fake=await readUiNativeVersionFromClient({
    ...args(native({},sha,{})),taskIssueNumbers:[1],
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

// Scripted client: gets(number,call) is the call-th issues.get for that
// selected frozen Task (call 0 = initial snapshot, call 1 = coherence recheck).
const scriptedNative=(gets,deps={})=>{
  const getCalls=new Map();
  return {getCalls,
    rest:{
      repos:{
        getBranch:async()=>({data:{commit:{sha}}}),
        getCommit:async()=>({data:{sha}}),
      },
      issues:{
        get:async({issue_number})=>{
          const call=getCalls.get(issue_number)??0;
          getCalls.set(issue_number,call+1);
          return {data:gets(issue_number,call)};
        },
      },
    },
    paginate:{iterator:async function*(route,params={}){
      if(route==='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by'){
        yield {data:deps[params.issue_number]??[]};
        return;
      }
      throw new Error('UNEXPECTED_PAGINATION_ROUTE');
    }},
  };
};

test('Task Issue metadata mutated between the first GET and the nested WorkItem observation is denied',async()=>{
  for(const changes of [
    {title:'Task #1 renamed mid-observation'},
    {updated_at:'2026-11-03T12:00:00Z'},
    {labels:[{name:'type:task'},{name:'state:claimed'},{name:'priority:P1'}]},
  ]){
    let armed=false;
    const mutatingReader=observed=>{
      armed=true;
      return sourceReader(observed);
    };
    const r=await readUiNativeVersionFromClient({
      ...args(scriptedNative(
        (n,call)=>armed&&call>0?issue(n,changes):issue(n))),
      taskIssueNumbers:[1,2],workItemReader:mutatingReader});
    assert.deepEqual(r,{connected:false,
      reason:'NATIVE_TASK_ROSTER_DRIFT_DURING_OBSERVATION'});
  }
});

test('unrelated repository Task Issues from another version do not contaminate the selected frozen roster',async()=>{
  const c=native();
  const otherVersionTask=issue(300,{labels:[
    {name:'type:task'},{name:'state:ready'},{name:'priority:P1'},
    {name:'version:v2.1.1'}
  ]});
  c.rest.issues.listForRepo=async()=>({data:[issue(1),issue(2),otherVersionTask]});
  c.paginate.iterator=async function*(route,params={}){
    if(route===c.rest.issues.listForRepo){
      yield {data:[issue(1),issue(2),otherVersionTask]};
      return;
    }
    if(route==='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by'){
      yield {data:[]};
      return;
    }
    throw new Error('UNEXPECTED_PAGINATION_ROUTE');
  };
  const r=await readUiNativeVersionFromClient({...args(c),
    taskIssueNumbers:[1,2],workItemReader:sourceReader});
  assert.equal(r.connected,true);
  assert.deepEqual(r.tasks.map(task=>task.id),['1','2']);
});

test('a missing selected frozen Task Issue fails closed',async()=>{
  const c=native();
  const get=c.rest.issues.get;
  c.rest.issues.get=async args=>args.issue_number===2
    ?{data:undefined}:get(args);
  const r=await readUiNativeVersionFromClient({...args(c),
    taskIssueNumbers:[1,2],workItemReader:sourceReader});
  assert.deepEqual(r,{connected:false,reason:'NATIVE_ISSUE_IDENTITY_UNVERIFIED'});
});

test('stable unchanged roster still returns connected with exact metadata',async()=>{
  const {getCalls,...octo}=scriptedNative(n=>issue(n));
  const r=await readUiNativeVersionFromClient({
    ...args(octo),taskIssueNumbers:[1,2],workItemReader:sourceReader});
  assert.equal(r.connected,true);
  assert.equal(r.source.sourceSha,sha);
  assert.deepEqual(r.tasks.map(t=>[t.id,t.title,t.updatedAt,t.issueState,
    t.status,t.severity,t.claimState,t.dependencyState]),
    [['1','Task #1','2026-11-01T00:00:00Z','open','open','P1','NOT_VERIFIED','CLEAR'],
     ['2','Task #2','2026-11-01T00:00:00Z','open','open','P1','NOT_VERIFIED','CLEAR']]);
  assert.equal(r.rosterCoverage,'UNVERIFIED');
  // The coherence recheck re-read every selected frozen Task Issue.
  assert.deepEqual([...getCalls.entries()].sort((a,b)=>a[0]-b[0]),[[1,2],[2,2]]);
});


const requirements=[{number:77,kind:'requirement',parent:null},
  {number:78,kind:'story',parent:77}];
const intakeObservation={status:'PROJECT_INTAKE_OBSERVED',sourceQualified:true,
  authorized:false,taskPass:false,releasePass:false,
  repository:{fullName:'org/tally'},observedAt:'2026-11-02T00:00:00Z',
  sources:[{number:77,title:'Product goal',state:'open',
    isPullRequest:false,url:'https://github.com/org/tally/issues/77'},
    {number:78,title:'Feature story',state:'open',
      isPullRequest:false,url:'https://github.com/org/tally/issues/78'}]};
test('native Product source validates real GitHub Issue titles; freeze stays UNKNOWN',async()=>{
  const r=await readUiNativeRequirementsFromClient({...args(native({},sha,{
    77:issue(77,{title:'Product goal'}),78:issue(78,{title:'Feature story'})})),
    requirementIssueSpecs:requirements,
    intakeReader:async()=>intakeObservation});
  assert.equal(r.connected,true);
  assert.equal(r.structureCoverage,'HOST_DECLARED_UNVERIFIED');
  assert.equal(r.freezeCoverage,'UNKNOWN');
  assert.deepEqual(r.nodes.map(x=>[x.id,x.parent,x.freezeState,x.productRevision]),
    [['77',null,'UNKNOWN',null],['78','77','UNKNOWN',null]]);
  assert.equal(r.nodes[0].title,'Product goal');
  assert.deepEqual(r.nodes[0].acceptanceRefs,[]);
});
test('invalid requirement hierarchy, cycle and untrusted issue roster fail closed',async()=>{
  const input={...args(native()),intakeReader:async()=>intakeObservation};
  assert.equal((await readUiNativeRequirementsFromClient({
    ...input,requirementIssueSpecs:[{number:77,kind:'story',parent:99}]}))
    .reason,'REQUIREMENT_PARENT_OUTSIDE_ROSTER');
  assert.equal((await readUiNativeRequirementsFromClient({
    ...input,requirementIssueSpecs:[{number:77,kind:'story',parent:78},
      {number:78,kind:'story',parent:77}]})).reason,'REQUIREMENT_CYCLE');
  assert.equal((await readUiNativeRequirementsFromClient({
    ...input,requirementIssueSpecs:[{number:77,kind:'product-owner',parent:null}]}))
    .reason,'INVALID_HOST_REQUIREMENT_ROSTER');
});
test('native Requirement read rejects issue title drift and PR source',async()=>{
  const o=await readUiNativeRequirementsFromClient({
    ...args(native({},sha,{
      77:issue(77,{title:'Product goal modified'}),
      78:issue(78,{title:'Feature story'})})),
    requirementIssueSpecs:requirements,intakeReader:async()=>intakeObservation});
  assert.equal(o.reason,'REQUIREMENT_ISSUE_CHANGED_DURING_READ');
  const c={...intakeObservation,sources:[{
    ...intakeObservation.sources[0],isPullRequest:true},
    intakeObservation.sources[1]]};
  const f=await readUiNativeRequirementsFromClient({
    ...args(native()),requirementIssueSpecs:requirements,
    intakeReader:async()=>c});
  assert.equal(f.reason,'REQUIREMENT_ISSUE_IDENTITY_UNVERIFIED');
});
test('moved pinned version branch blocks otherwise valid requirements',async()=>{
  const r=await readUiNativeRequirementsFromClient({
    ...args(native({},'b'.repeat(40))),requirementIssueSpecs:requirements,
    intakeReader:async()=>intakeObservation});
  assert.equal(r.reason,'VERSION_BRANCH_NOT_PINNED');
});

const qaContext='ci/woodpecker/push/v2-alm-workspace';
const qaClient=(list=[{context:qaContext,state:'success',
  target_url:'https://ci.kaicreator.com/repos/12/pipeline/369/2',
  updated_at:'2026-11-02T00:00:00Z'}],branchSha=sha)=>{
  const base=native({},branchSha);
  return {...base,rest:{...base.rest,repos:{...base.rest.repos,
    getCombinedStatusForRef:async()=>({data:{
      sha,total_count:list.length,statuses:list}})}}};
};
test('GitHub commit status yields REPORTED only, never independent QA',async()=>{
  const r=await readUiNativeQaFromClient({...args(qaClient()),
    qaStatusContext:qaContext});
  assert.equal(r.connected,true);
  assert.equal(r.independentCoverage,'UNKNOWN');
  assert.equal(r.ciCoverage,'REPORTED_ONLY');
  assert.equal(r.records[0].layer,'fullRegression');
  assert.equal(r.records[0].provenance,'REPORTED');
  assert.equal(r.records[0].outcome,'PASS');
});
test('no matching scoped CI context is observed absence, not QA PASS',async()=>{
  const r=await readUiNativeQaFromClient({...args(qaClient([])),
    qaStatusContext:qaContext});
  assert.equal(r.connected,true);
  assert.equal(r.ciCoverage,'NO_MATCHING_STATUS');
  assert.deepEqual(r.records,[]);
});
test('forged CI target URL or incomplete status pages deny source',async()=>{
  const bad=await readUiNativeQaFromClient({...args(qaClient([{
    context:qaContext,state:'success',
    target_url:'https://attacker.example/ci/pass',
    updated_at:'2026-11-02T00:00:00Z'}])),qaStatusContext:qaContext});
  assert.equal(bad.reason,'GITHUB_CI_STATUS_IDENTITY_UNQUALIFIED');
  const incomplete=qaClient();
  incomplete.rest.repos.getCombinedStatusForRef=async()=>({
    data:{sha,total_count:120,statuses:[]}});
  const r=await readUiNativeQaFromClient({...args(incomplete),qaStatusContext:qaContext});
  assert.equal(r.reason,'GITHUB_CI_STATUS_PAGE_UNQUALIFIED');
});
test('moved branch or invalid CI context cannot yield a connected QA state',async()=>{
  assert.equal((await readUiNativeQaFromClient({
    ...args(qaClient([], 'b'.repeat(40))),qaStatusContext:qaContext}))
    .reason,'VERSION_BRANCH_NOT_PINNED');
  assert.equal((await readUiNativeQaFromClient({
    ...args(qaClient()),qaStatusContext:'hiddenValidation/PASS'}))
    .reason,'QA_STATUS_SOURCE_NOT_CONFIGURED');
});
