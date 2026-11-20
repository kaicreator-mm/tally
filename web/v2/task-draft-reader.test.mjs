import test from 'node:test';
import assert from 'node:assert/strict';
import {readGitHubTaskDrafts} from './task-draft-reader.mjs';
const sha='a'.repeat(40),repo='org/test';
const taskBody=[
  'Tally development task draft — not yet admitted to Task DAG.',
  'Repository: org/test','Selected branch: main',
  'Observed source SHA: '+sha,'Request: tally:draft:request-123','',
  'Build one API endpoint','',
  'Status: UNQUALIFIED_DRAFT. No Agent, Claim or Release authority.',
].join('\n');
const issue={
  number:11,title:'Create API',state:'open',
  html_url:'https://github.com/org/test/issues/11',body:taskBody
};
const args={repository:repo,versionId:'main',sourceSha:sha,
  token:'github_pat_TEST_READ'};
test('native draft list is read-only and never promotes to Task',async()=>{
  let count=0;
  const data=await readGitHubTaskDrafts({...args,fetchImpl:async (url,options)=>{
    count++;
    assert.equal(options.method,undefined);
    assert.match(options.headers.authorization,/^Bearer github_pat_TEST_READ$/);
    assert.match(url,/issues\?state=all/);
    return {ok:true,json:async()=>[issue,{...issue,number:12,body:'different'}]};
  }});
  assert.equal(count,1);
  assert.equal(data.connected,true);
  assert.equal(data.drafts.length,1);
  assert.equal(data.drafts[0].taskQualified,false);
  assert.equal(data.drafts[0].agentAuthorized,false);
  assert.equal(data.drafts[0].issueNumber,11);
  assert.equal(data.coverage,'COMPLETE_SINGLE_PAGE');
});
test('GitHub token cannot read Issues -> explicit disconnected reason',async()=>{
  const data=await readGitHubTaskDrafts({...args,fetchImpl:async()=>({status:403,ok:false})});
  assert.deepEqual(data,{connected:false,reason:'GITHUB_ISSUES_READ_PERMISSION_REQUIRED'});
});
test('empty real issue page is connected empty draft list, not qualified DAG',async()=>{
  const data=await readGitHubTaskDrafts({...args,fetchImpl:async()=>({
    status:200,ok:true,json:async()=>[]})});
  assert.equal(data.connected,true);
  assert.deepEqual(data.drafts,[]);
  assert.equal(data.coverage,'COMPLETE_SINGLE_PAGE');
});
test('foreign and malformed Issue rows never appear',async()=>{
  const data=await readGitHubTaskDrafts({...args,fetchImpl:async()=>({
    status:200,ok:true,json:async()=>[
      {...issue,html_url:'https://github.com/org/foreign/issues/11'},
      {...issue,number:12,pull_request:{}},
      {...issue,number:13,title:'Foreign claim',body:'Fake'},
    ]})});
  assert.deepEqual(data.drafts,[]);
});
