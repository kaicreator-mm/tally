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
test('provider-reported usage accompanies an ordinary observed text completion only',()=>{
  const plan=planning();
  const x=projectNeutralGenerationResult(plan,{
    contractVersion:'dh.llm.generate/1',requestId:plan.request.requestId,
    providerKind:plan.providerKind,modelId:plan.request.modelId,
    status:'completed',text:'Need source evidence.',finishReason:'stop',
    usage:{evidenceMode:'provider-reported',inputTokens:10,outputTokens:8}
  });
  assert.equal(x.status,'OBSERVED');
  assert.equal(x.text,'Need source evidence.');
  assert.equal(x.toolProposalCount,0);
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


test('remote outcome UNKNOWN cannot be misclassified as a terminal known failure',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,usage:{evidenceMode:'unavailable'},
    reasonCode:'REMOTE_TIMEOUT'};
  for(const bad of [
    {status:'failed',errorCategory:'OUTCOME_UNKNOWN'},
    {status:'unknown',errorCategory:'DEADLINE'}
  ]){
    const out=projectNeutralGenerationResult(p,{...common,...bad});
    assert.equal(out.reason,'MALFORMED_NEUTRAL_FAILURE');
    assertDeny(out);
  }
  const unknown=projectNeutralGenerationResult(p,{
    ...common,status:'unknown',errorCategory:'OUTCOME_UNKNOWN'});
  assert.equal(unknown.status,'UNKNOWN');
  assertDeny(unknown);
  const deadline=projectNeutralGenerationResult(p,{
    ...common,status:'refused',errorCategory:'DEADLINE'});
  assert.equal(deadline.status,'REFUSED');
  assertDeny(deadline);
});
test('tool_proposal completion is denied without an approved Tool Catalog',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'completed',text:'secret result',
    finishReason:'tool_proposal',usage:{evidenceMode:'unavailable'}};
  for(const proposal of [
    undefined,[],
    [{name:'check',argumentsJson:'{"path":"a"}'}],
    [{name:'effect',argumentsJson:'{"secret":"CANARY_DO_NOT_JOURNAL"}'}]
  ]){
    const out=projectNeutralGenerationResult(p,{...common,toolProposals:proposal});
    assert.equal(out.reason,'UNREQUESTED_TOOL_PROPOSAL');
    assert.equal(out.status,'BLOCKED');
    assert.equal(JSON.stringify(out).includes('CANARY_DO_NOT_JOURNAL'),false);
    assertDeny(out);
  }
});
test('nonempty unsolicited toolProposals are denied even with finishReason stop',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'completed',text:'ok',
    finishReason:'stop',usage:{evidenceMode:'unavailable'}};
  for(const proposal of [
    [{name:'check',argumentsJson:'{"path":"a"}'}],
    [{name:'check',argumentsJson:'[]'}],
    [{name:'check',argumentsJson:'not-json'}]
  ]){
    const out=projectNeutralGenerationResult(p,{...common,toolProposals:proposal});
    assert.equal(out.reason,'UNREQUESTED_TOOL_PROPOSAL');
    assert.equal(out.toolExecutionAuthorized,undefined);
    assertDeny(out);
  }
  const empty=projectNeutralGenerationResult(p,{...common,toolProposals:[]});
  assert.equal(empty.status,'OBSERVED');
  assert.equal(empty.toolProposalCount,0);
  assertDeny(empty);
});
test('error reasonCode matches frozen transport guard grammar',()=>{
  const p=planning();
  const common={contractVersion:'dh.llm.generate/1',
    requestId:p.request.requestId,providerKind:p.providerKind,
    modelId:p.request.modelId,status:'failed',errorCategory:'UNAVAILABLE',
    usage:{evidenceMode:'unavailable'}};
  for(const code of ['1BAD','A:B','A-B','a_lower']){
    const out=projectNeutralGenerationResult(p,{...common,reasonCode:code});
    assert.equal(out.reason,'MALFORMED_NEUTRAL_FAILURE');
    assertDeny(out);
  }
  const ok=projectNeutralGenerationResult(p,{
    ...common,reasonCode:'REMOTE_UNAVAILABLE'});
  assert.equal(ok.status,'FAILED');
  assertDeny(ok);
});
