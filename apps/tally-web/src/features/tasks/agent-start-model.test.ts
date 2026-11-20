import test from 'node:test';
import assert from 'node:assert/strict';
import {agentStartDecision} from './agent-start-model.ts';
const sha='a'.repeat(40);
const ready={taskId:'9',claimState:'ACCEPTED',sourceSha:sha,
  host:{connected:true,selectedTaskIssue:9,sourceSha:sha}};
test('only exact accepted Host/Task/Claim/SHA allows Agent start',()=>{
  assert.deepEqual(agentStartDecision(ready),
    {enabled:true,reason:'HOST_READY',issueNumber:9});
});
test('Host absent denies even accepted task',()=>{
  assert.equal(agentStartDecision({...ready,host:null}).reason,'AGENT_HOST_NOT_MOUNTED');
});
test('wrong Task or changed SHA fail',()=>{
  assert.equal(agentStartDecision({...ready,taskId:'10'}).enabled,false);
  assert.equal(agentStartDecision({...ready,sourceSha:'b'.repeat(40)}).enabled,false);
});
test('an Issue draft or unknown Claim cannot start Agent',()=>{
  assert.equal(agentStartDecision({...ready,claimState:'NOT_VERIFIED'}).enabled,false);
  assert.equal(agentStartDecision({...ready,taskId:'draft:9'}).enabled,false);
});
