import test from 'node:test';
import assert from 'node:assert/strict';
import {createGitHubTaskDraft} from './draft-github-issue.mjs';
const repository='org/test',sha='a'.repeat(40),requestId='issue-test-123';
const args={repository,ref:'main',expectedSha:sha,title:'Add one API endpoint',
  description:'Create a /health endpoint',writeToken:'github_pat_TEST_WRITE',
  requestId};
const issued={number:18,html_url:'https://github.com/org/test/issues/18'};
test('creates a draft by explicit POST and independent GitHub issue GET readback',async()=>{
  let posted;
  const fetchImpl=async(url,options)=>{
    assert.match(options.headers.authorization,/^Bearer github_pat_TEST_WRITE$/);
    if(options.method==='POST'){
      posted=JSON.parse(options.body);
      assert.equal(url,'https://api.github.com/repos/org/test/issues');
      assert.match(posted.body,/UNQUALIFIED_DRAFT/);
      return {ok:true,status:201,json:async()=>issued};
    }
    assert.equal(url,'https://api.github.com/repos/org/test/issues/18');
    return {ok:true,json:async()=>({...issued,title:args.title,body:posted.body})};
  };
  const result=await createGitHubTaskDraft({...args,fetchImpl});
  assert.equal(result.status,'DRAFT_ISSUE_CREATED');
  assert.equal(result.taskQualified,false);
  assert.equal(result.agentAuthorized,false);
  assert.equal(result.issueUrl,issued.html_url);
  assert.doesNotMatch(JSON.stringify(result),/github_pat/);
});
test('write permission denied has bounded error code, never logs token',async()=>{
  const fetchImpl=async()=>({ok:false,status:403});
  await assert.rejects(createGitHubTaskDraft({...args,fetchImpl}),
    /GITHUB_ISSUES_WRITE_PERMISSION_REQUIRED/);
});
test('mismatched readback remains UNKNOWN rather than reporting created',async()=>{
  const fetchImpl=async(url)=>{
    if(url.endsWith('/issues'))return {ok:true,status:201,json:async()=>issued};
    return {ok:true,json:async()=>({...issued,title:'foreign',body:'wrong'})};
  };
  await assert.rejects(createGitHubTaskDraft({...args,fetchImpl}),
    /DRAFT_ISSUE_READBACK_MISMATCH/);
});
test('invalid request or missing one-shot credential rejects without effect',async()=>{
  let invoked=false;
  await assert.rejects(createGitHubTaskDraft({...args,writeToken:'',
    fetchImpl:async()=>{invoked=true;}}),/INVALID_DRAFT_TASK_INPUT/);
  assert.equal(invoked,false);
});
