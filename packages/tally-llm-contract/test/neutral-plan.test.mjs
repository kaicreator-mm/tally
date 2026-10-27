import test from 'node:test';
import assert from 'node:assert/strict';
import {planNeutralGeneration,projectNeutralGenerationResult,frozenContractVersion}
  from '../src/index.mjs';

const now=1793136000000;
const request=()=>({contractVersion:'dh.llm.generate/1',requestId:'tally-148:ask:1',
  modelId:'approved-model',deadlineUnixMs:now+30000,
  messages:[{role:'system',content:'Summarize evidence only.'},
    {role:'user',content:'What is missing for Product freeze?'}],
  maxOutputTokens:500,outputFormat:'text'});
const config={nowUnixMs:now,selectedProviders:['openai-compatible'],
  approvedModels:{'openai-compatible':['approved-model']}};
const planning=()=>planNeutralGeneration(request(),config);
const assertDeny=r=>{
  assert.equal(r.authorized,false);assert.equal(r.admitted,false);
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
};
test('exact v0.10.3 logical contract requests an explicit selected C route, but no Kernel effect',()=>{
  assert.equal(frozenContractVersion,'dh.llm.generate/1');
  const p=planning();
  assert.equal(p.status,'PLANNED');
  assert.equal(p.providerKind,'openai-compatible');
  assert.deepEqual(p.route,{intentId:'llmViaOpenAI',capabilityId:'llm.openai',
    operationId:'llm.openai.generate',handlerId:'generate'});
  assert.equal(p.request.contractVersion,'dh.llm.generate/1');
  assert.equal(p.request.messages.length,2);
  assert.equal(p.providerInvocationPerformed,false);
  assert.equal(p.requiresKernelPermit,true);
  assertDeny(p);
});
test('zero selected, both ambiguous, explicit one-of-two, and unselected routes',()=>{
  assert.equal(planNeutralGeneration(request(),{...config,selectedProviders:[]}).reason,
    'NO_SELECTED_MODEL_CAPABILITY');
  const both={...config,selectedProviders:['openai-compatible','anthropic-messages'],
    approvedModels:{...config.approvedModels,'anthropic-messages':['approved-model']}};
  assert.equal(planNeutralGeneration(request(),both).reason,
    'EXPLICIT_MODEL_ROUTE_REQUIRED');
  const chosen=planNeutralGeneration(request(),{...both,chosenProvider:'anthropic-messages'});
  assert.equal(chosen.route.intentId,'llmViaAnthropic');
  assert.equal(chosen.route.operationId,'llm.anthropic.generate');
  assertDeny(chosen);
  assert.equal(planNeutralGeneration(request(),{...config,chosenProvider:'anthropic-messages'}).reason,
    'UNSELECTED_MODEL_CAPABILITY');
  assert.equal(planNeutralGeneration(request(),{...config,
    selectedProviders:['openai-compatible','openai-compatible']}).reason,
    'SELECTED_PROVIDER_SET_UNVERIFIED');
});
test('Host model binding, deadlines and optional unsupported Tool or schema features reject',()=>{
  const invalids=[
    [request(),{...config,approvedModels:{'openai-compatible':['other']}},'HOST_MODEL_ALLOWLIST_REFUSED'],
    [{...request(),deadlineUnixMs:now-1},config,'HOST_DEADLINE_REFUSED'],
    [{...request(),deadlineUnixMs:now+3600001},config,'HOST_DEADLINE_REFUSED'],
    [{...request(),toolCatalogRef:'all-tools'},config,'OPTIONAL_FEATURE_NOT_HOST_APPROVED'],
    [{...request(),outputSchemaRef:'unapproved-schema'},config,'OPTIONAL_FEATURE_NOT_HOST_APPROVED'],
    [{...request(),maxOutputTokens:9000},config,'INVALID_NEUTRAL_LLM_REQUEST'],
    [{...request(),temperature:1.1},config,'INVALID_NEUTRAL_LLM_REQUEST'],
    [{...request(),messages:[{role:'tool',content:'untrusted'}]},config,
      'INVALID_NEUTRAL_LLM_REQUEST']
  ];
  for(const [r,c,reason] of invalids){
    const out=planNeutralGeneration(r,c);
    assert.equal(out.reason,reason);
    assertDeny(out);
  }
});
test('provider-reported tokens and tool proposals remain data, never tool execution',()=>{
  const plan=planning();
  const x=projectNeutralGenerationResult(plan,{
    contractVersion:'dh.llm.generate/1',requestId:plan.request.requestId,
    providerKind:plan.providerKind,modelId:plan.request.modelId,
    status:'completed',text:'Need source evidence.',finishReason:'tool_proposal',
    usage:{evidenceMode:'provider-reported',inputTokens:10,outputTokens:8},
    toolProposals:[{name:'git.write',argumentsJson:'{"effect":"forbidden"}'}]
  });
  assert.equal(x.status,'OBSERVED');
  assert.equal(x.text,'Need source evidence.');
  assert.equal(x.toolProposalCount,1);
  assert.equal(x.toolExecutionAuthorized,false);
  assert.deepEqual(x.usage,{evidenceMode:'provider-reported',inputTokens:10,outputTokens:8});
  assertDeny(x);
});
test('missing usage is not fabricated; unavailable usage never carries invented zeros',()=>{
  const plan=planning();
  const common={contractVersion:'dh.llm.generate/1',requestId:plan.request.requestId,
    providerKind:plan.providerKind,modelId:plan.request.modelId,
    status:'completed',text:'ok',finishReason:'stop'};
  assert.equal(projectNeutralGenerationResult(plan,{...common}).reason,
    'PROVIDER_PROVENANCE_OR_USAGE_UNVERIFIED');
  assert.equal(projectNeutralGenerationResult(plan,{
    ...common,usage:{evidenceMode:'unavailable',inputTokens:0,outputTokens:0}
  }).reason,'PROVIDER_PROVENANCE_OR_USAGE_UNVERIFIED');
  const x=projectNeutralGenerationResult(plan,{
    ...common,usage:{evidenceMode:'unavailable'}});
  assert.equal(x.status,'OBSERVED');
  assert.deepEqual(x.usage,{evidenceMode:'unavailable'});
  assertDeny(x);
});
test('different provider/model/requestId cannot be projected as successful',()=>{
  const p=planning();
  const result={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'completed',
    text:'ok',finishReason:'stop',usage:{evidenceMode:'unavailable'}};
  for(const m of [
    {requestId:'different'}, {providerKind:'anthropic-messages'},
    {modelId:'unselected-model'}, {status:'completed',finishReason:'invented'}
  ]){
    const x=projectNeutralGenerationResult(p,{...result,...m});
    assert.equal(x.status,'BLOCKED');
    assertDeny(x);
  }
});
test('unknown timeout not converted into refused no-effect or a retry; errors sanitized',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',requestId:p.request.requestId,
    providerKind:p.providerKind,modelId:p.request.modelId,
    status:'unknown',errorCategory:'OUTCOME_UNKNOWN',reasonCode:'REMOTE_TIMEOUT',
    usage:{evidenceMode:'unavailable'}};
  const x=projectNeutralGenerationResult(p,common);
  assert.equal(x.status,'UNKNOWN');
  assert.equal(x.reason,'REMOTE_OUTCOME_UNKNOWN');
  assertDeny(x);
  const leak=projectNeutralGenerationResult(p,{...common,reasonCode:'raw: Bearer abc123'});
  assert.equal(leak.status,'BLOCKED');
  assert.equal(JSON.stringify(leak).includes('abc123'),false);
  assertDeny(leak);
});
test('hostile provider getters are normalized to typed denial without raw error',()=>{
  const p=planning();
  const secret='TALLY_PROVIDER_SECRET_NEVER_JOURNAL';
  const r={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'completed',
    get usage(){throw Error(secret);}};
  const x=projectNeutralGenerationResult(p,r);
  assert.equal(x.reason,'HOST_PROVIDER_RESULT_UNTRUSTED');
  assert.equal(JSON.stringify(x).includes(secret),false);
  assertDeny(x);
});

test('provider usage with unapproved fields is denied without forwarding secrets',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'completed',text:'ok',
    finishReason:'stop'};
  const secret='PROVIDER_PRIVATE_TOKEN_NEVER_FORWARD';
  for(const usage of [
    {evidenceMode:'unavailable',internalToken:secret},
    {evidenceMode:'provider-reported',inputTokens:2,outputTokens:3,debug:secret},
    {evidenceMode:'measured',inputTokens:2,outputTokens:3,credentials:secret}
  ]){
    const out=projectNeutralGenerationResult(p,{...common,usage});
    assert.equal(out.status,'BLOCKED');
    assert.equal(out.reason,'PROVIDER_PROVENANCE_OR_USAGE_UNVERIFIED');
    assert.equal(JSON.stringify(out).includes(secret),false);
    assertDeny(out);
  }
});
test('planned request messages are an immutable snapshot of validated input',()=>{
  const original=request();
  const p=planNeutralGeneration(original,config);
  assert.equal(p.status,'PLANNED');
  assert.equal(Object.isFrozen(p.request),true);
  assert.equal(Object.isFrozen(p.request.messages),true);
  assert.equal(Object.isFrozen(p.request.messages[0]),true);
  const observed=p.request.messages[0].content;
  original.messages[0].content='tamper after planning';
  original.messages.push({role:'user',content:'new unreviewed message'});
  assert.equal(p.request.messages[0].content,observed);
  assert.equal(p.request.messages.length,2);
  assert.throws(()=>{p.request.messages[0].content='post-plan rewrite';},TypeError);
  assert.throws(()=>{p.request.messages.push({role:'user',content:'injected'});},TypeError);
  assertDeny(p);
});
