import test from 'node:test';
import assert from 'node:assert/strict';
import {createGitHubTaskDraft} from './draft-github-issue.mjs';
const repository='org/test',sha='a'.repeat(40),requestId='issue-test-123';
const args={repository,ref:'main',expectedSha:sha,title:'Add one API endpoint',
  description:'Create a /health endpoint',writeToken:'github_pat_TEST_WRITE',
  requestId};
const issued={number:18,html_url:'https://github.com/org/test/issues/18'};
const taskLabels=[{name:'type:task'},{name:'state:planned'}];
const successful=(overrides={})=>{
  let posted,issuePosts=0,labelPosts=0;
  const fetchImpl=async (url,options)=>{
    assert.match(options.headers.authorization,/^Bearer github_pat_TEST_WRITE$/);
    const segment=url.match(/\/labels\/(.*)$/)?.[1];
    if(segment){
      const name=decodeURIComponent(segment);
      if(overrides.missingLabels&&!labelPosts)
        return {ok:false,status:404};
      return {ok:true,status:200,json:async()=>({name})};
    }
    if(url.endsWith('/labels')&&options.method==='POST'){
      labelPosts++;return {ok:true,status:201};
    }
    if(url.endsWith('/issues')&&options.method==='POST'){
      issuePosts++;posted=JSON.parse(options.body);
      assert.deepEqual(posted.labels,['type:task','state:planned']);
      return {ok:true,status:201,json:async()=>issued};
    }
    assert.equal(url,'https://api.github.com/repos/org/test/issues/18');
    return {ok:true,status:200,json:async()=>({
      ...issued,title:overrides.wrongTitle?'foreign':args.title,
      body:posted.body,labels:overrides.droppedLabels?[]:taskLabels
    })};
  };
  return {fetchImpl,observed:()=>({issuePosts,labelPosts,posted})};
};
test('creates canonical planned Task Issue without accepting a Claim',async()=>{
  const f=successful();
  const result=await createGitHubTaskDraft({...args,fetchImpl:f.fetchImpl});
  assert.equal(result.status,'DRAFT_ISSUE_CREATED');
  assert.equal(result.nativeWorkflow,'PLANNED');
  assert.equal(result.taskQualified,false);
  assert.equal(result.agentAuthorized,false);
  assert.equal(result.issueUrl,issued.html_url);
  assert.equal(f.observed().issuePosts,1);
  assert.doesNotMatch(JSON.stringify(result),/github_pat/);
});
test('on empty repo creates missing canonical labels before Issue',async()=>{
  const f=successful({missingLabels:true});
  const result=await createGitHubTaskDraft({...args,fetchImpl:f.fetchImpl});
  assert.equal(result.nativeWorkflow,'PLANNED');
  assert.equal(f.observed().labelPosts,1);
});
test('silent label drop returns issue URL and explicit repair state',async()=>{
  const f=successful({droppedLabels:true});
  const result=await createGitHubTaskDraft({...args,fetchImpl:f.fetchImpl});
  assert.equal(result.status,'DRAFT_ISSUE_REQUIRES_LABEL_REPAIR');
  assert.equal(result.issueUrl,issued.html_url);
  assert.equal(result.nativeWorkflow,'NONCANONICAL');
  assert.equal(result.agentAuthorized,false);
});
test('write permission denied before any Issue effect',async()=>{
  const fetchImpl=async()=>({ok:false,status:403});
  await assert.rejects(createGitHubTaskDraft({...args,fetchImpl}),
    /GITHUB_ISSUES_WRITE_PERMISSION_REQUIRED/);
});
test('mismatched Issue readback cannot report created',async()=>{
  const f=successful({wrongTitle:true});
  await assert.rejects(createGitHubTaskDraft({...args,fetchImpl:f.fetchImpl}),
    /DRAFT_ISSUE_READBACK_MISMATCH/);
});
test('invalid request or missing one-shot credential rejects without effect',async()=>{
  let invoked=false;
  await assert.rejects(createGitHubTaskDraft({...args,writeToken:'',
    fetchImpl:async()=>{invoked=true;}}),/INVALID_DRAFT_TASK_INPUT/);
  assert.equal(invoked,false);
});
