import test from 'node:test';
import assert from 'node:assert/strict';
import {projectAgentDispatchStatus} from './agent-dispatch-status.mjs';
const sourceSha='a'.repeat(40);
test('absent trusted Host denies agent even with selected task',()=>{
  assert.deepEqual(projectAgentDispatchStatus({taskIssueNumber:9,
    sourceSha}),{connected:false,reason:'AGENT_HOST_NOT_MOUNTED'});
});
test('valid trusted Host binds precisely one selected Task and SHA',()=>{
  const result=projectAgentDispatchStatus({mounted:true,
    taskIssueNumber:9,sourceSha,provider:'codex'});
  assert.deepEqual(result,{connected:true,selectedTaskIssue:9,sourceSha,
    provider:'codex',authority:'HOST_PINNED_ONLY'});
});
test('missing source or issue cannot expose a dispatch key',()=>{
  for(const broken of [{mounted:true,taskIssueNumber:0,sourceSha},
    {mounted:true,taskIssueNumber:9,sourceSha:'invalid'},
    {mounted:true,taskIssueNumber:9,sourceSha,provider:'BAD NAME'}]){
    const result=projectAgentDispatchStatus(broken);
    assert.equal(result.connected,false);
    assert.equal(result.reason,'AGENT_HOST_SCOPE_INVALID');
  }
});
