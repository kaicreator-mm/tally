import test from 'node:test';
import assert from 'node:assert/strict';
import {readWorkItemSourceFromClient} from '../src/task-workitem-source.mjs';
const sha='a'.repeat(40), owner='kaicreator-mm',repo='tally',issueNumber=125;
const base={owner,repo,issueNumber,expectedSourceSha:sha};
const url=`https://github.com/${owner}/${repo}/issues/${issueNumber}`;
const issue=(labels=['type:task','state:ready'])=>({
  id:123456,number:issueNumber,html_url:url,state:'open',updated_at:'2026-10-27T00:00:00Z',
  labels:labels.map(name=>({name}))
});
const blocker=(n,st='closed')=>({
  id:100000+n,number:n,
  html_url:`https://github.com/kaicreator-mm/tally/issues/${n}`,
  state:st
});
function client({labels,blocked=[],shaResult=sha,drift=false,apiError=null}={}){
  let reads=0;
  const calls={issues:0,commit:0,dependencies:0};
  const octokit={
    rest:{
      issues:{async get(){calls.issues++;const data=issue(labels);if(drift&&++reads===2)data.updated_at='2026-10-27T00:01:00Z';return {data};}},
      repos:{async getCommit(){calls.commit++;return {data:{sha:shaResult}};}}
    },
    paginate:{async *iterator(route,params){
      calls.dependencies++;
      assert.equal(route,'GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by');
      assert.equal(params.issue_number,issueNumber);
      if(apiError)throw Object.assign(new Error('SECRET_SHOULD_NOT_ESCAPE'),{status:apiError});
      yield {data:blocked};
    }}
  };
  return {octokit,calls};
}
test('no claim ever accepted from native ready+all closed blockers',async()=>{
  const {octokit,calls}=client({blocked:[blocker(100)]});
  const x=await readWorkItemSourceFromClient({...base,octokit});
  assert.equal(x.reason,'CLAIM_NOT_VERIFIED');
  assert.equal(x.dependencyState,'CLEAR');
  assert.equal(x.blockerCount,1);
  assert.equal(x.claimStatus,'NOT_VERIFIED');
  assert.equal(x.admitted,false);
  assert.equal(x.taskPass,false);
  assert.deepEqual(calls,{issues:2,commit:1,dependencies:1});
});
test('a native open blocked-by dependency forbids readiness',async()=>{
  const {octokit}=client({blocked:[blocker(100,'open')]});
  const x=await readWorkItemSourceFromClient({...base,octokit});
  assert.equal(x.reason,'NATIVE_ISSUE_DEPENDENCIES_BLOCKED');
  assert.equal(x.dependencyState,'BLOCKED');
  assert.equal(x.openBlockerCount,1);
  assert.equal(x.taskPass,false);
});
test('label-only state:claimed and arbitrary claimed role never prove CAS',async()=>{
  for(const labels of [
    ['type:task','state:claimed','role:reviewer','DISPATCH_CLAIMED'],
    ['type:task','state:ready','claim-accepted'],
    ['type:task','state:implementing']
  ]){
    const {octokit}=client({labels});
    const x=await readWorkItemSourceFromClient({...base,octokit,claimAccepted:true,dependenciesComplete:true});
    assert.equal(x.claimStatus,'NOT_VERIFIED');
    assert.equal(x.admitted,false);assert.equal(x.taskPass,false);
  }
});
test('missing or duplicated canonical state/type is unverified',async()=>{
  for(const labels of [[],['type:task'],['state:ready'],
    ['type:task','type:planning','state:ready'],
    ['type:task','state:ready','state:claimed']]){
    const x=await readWorkItemSourceFromClient({...base,octokit:client({labels}).octokit});
    assert.equal(x.reason,'CANONICAL_WORK_ITEM_STATE_UNVERIFIED');
    assert.equal(x.taskPass,false);
  }
});
test('wrong commit SHA and Issue drift fail closed',async()=>{
  assert.equal((await readWorkItemSourceFromClient({...base,octokit:client({shaResult:'b'.repeat(40)}).octokit})).reason,'ISSUE_OR_SOURCE_SHA_UNVERIFIED');
  assert.equal((await readWorkItemSourceFromClient({...base,octokit:client({drift:true}).octokit})).reason,'ISSUE_DRIFT_DURING_OBSERVATION');
});
test('truncated dependency list, duplicate identity and foreign malformed blocker refuse',async()=>{
  const cases=[
    {blocked:[blocker(100),blocker(101)],limit:1,reason:'DEPENDENCY_PAGINATION_LIMIT'},
    {blocked:[blocker(100),blocker(100)],reason:'DEPENDENCY_IDENTITY_UNVERIFIED'},
    {blocked:[{...blocker(100),html_url:'https://evil.test/issues/100'}],reason:'DEPENDENCY_IDENTITY_UNVERIFIED'},
  ];
  for(const {blocked,limit,reason} of cases){
    const x=await readWorkItemSourceFromClient({...base,octokit:client({blocked}).octokit,limit});
    assert.equal(x.reason,reason);
    assert.equal(x.dependencyState,'UNKNOWN');
    assert.equal(x.taskPass,false);
  }
});
test('403, 404, incomplete iterator, invalid query never false ready and never leaks provider message',async()=>{
  for(const [err,why] of [[403,'GITHUB_ACCESS_DENIED'],[404,'GITHUB_RESOURCE_NOT_FOUND'],[429,'GITHUB_RATE_LIMITED']]){
    const x=await readWorkItemSourceFromClient({...base,octokit:client({apiError:err}).octokit});
    assert.equal(x.reason,why);assert.equal(x.dependencyState,'UNKNOWN');
    assert.equal(JSON.stringify(x).includes('SECRET_SHOULD_NOT_ESCAPE'),false);
  }
  for(const invalid of [{issueNumber:0},{expectedSourceSha:'bad'},{owner:'../evil'},{limit:1001}]){
    const x=await readWorkItemSourceFromClient({...base,octokit:client().octokit,...invalid});
    assert.equal(x.reason,'INVALID_NATIVE_QUERY');
  }
});
