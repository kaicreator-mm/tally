import test from 'node:test';
import assert from 'node:assert/strict';
import {agentStartDecision,draftAgentStartRequest} from './agent-start-model.ts';
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

const draft={
  issueNumber:27,repository:'org/test',versionId:'main',sourceSha:sha,
  githubIssueState:'open' as const,nativeWorkflow:'READY' as const,
  sourceCurrentness:'CURRENT' as const,
  agentHost:{connected:true,selectedTaskIssue:27,sourceSha:sha},
  claimHost:{connected:true,repository:'org/test',versionId:'main',
    issueNumber:27,sourceSha:sha},
};
test('new repo ready Issue can request exact mounted Host, not claim authority',()=>{
  assert.deepEqual(draftAgentStartRequest(draft),
    {requestable:true,reason:'HOST_K_REVALIDATION_REQUIRED'});
});
test('new repo cannot start if Draft source, identity or Claim Host differs',()=>{
  const variants=[
    {...draft,nativeWorkflow:'PLANNED' as const},
    {...draft,sourceCurrentness:'STALE' as const},
    {...draft,githubIssueState:'closed' as const},
    {...draft,agentHost:null},
    {...draft,agentHost:{connected:true,selectedTaskIssue:28,sourceSha:sha}},
    {...draft,claimHost:{...draft.claimHost,repository:'org/foreign'}},
    {...draft,claimHost:{...draft.claimHost,sourceSha:'b'.repeat(40)}},
  ];
  for(const item of variants)assert.equal(draftAgentStartRequest(item).requestable,false);
});
