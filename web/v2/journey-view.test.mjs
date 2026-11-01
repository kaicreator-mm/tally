import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {renderJourneyWorkspace,__test} from './journey-view.mjs';
const sourceSha='ec9820c3a67e387d5ca51f736be0d6f316f36c12';
const observedAt='2026-10-30T19:00:00.000Z';
const task=(sourceRef,changes={})=>({sourceRef,sourceSha,observedAsOf:observedAt,
  currentness:'OBSERVED_AT_QUERY_COMPLETION',status:'AWAITING_HOST_ADMISSION',
  reason:'NATIVE_READY_NO_ACCEPTED_CLAIM',dependencyState:'CLEAR',claimState:'NOT_VERIFIED',
  kernelTaskState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
  ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,...changes});
const one=task('https://github.com/kaicreator-mm/tally/issues/172');
const snapshot={status:'OBSERVED',project:'tally',version:'v2',sourceSha,observedAsOf:observedAt,
  currentness:'OBSERVED_AT_QUERY_COMPLETION',tasks:[one],authorized:false,admitted:false,taskPass:false,releasePass:false};
const eligible={authorizationType:'START_ELIGIBLE_CLAIM',reason:'NATIVE_READY_NO_ACCEPTED_CLAIM',
  actorRole:'builder',requiredCapability:'HOST_CLAIM_WRITER',approvalNeeded:true,costRisk:'LOW',
  authorization:'HOST_AND_ADS_REQUIRED',currentness:snapshot.currentness,
  evidence:{sourceRef:one.sourceRef,sourceSha,observedAsOf:observedAt}};
test('Agent/CI eligible work has no Human card and preserves independent layers',()=>{
  const html=renderJourneyWorkspace(snapshot,{actions:[eligible],humanActionInbox:[]});
  assert.match(html,/No human action required/);assert.match(html,/START_ELIGIBLE_CLAIM/);
  assert.match(html,/HOST_AND_ADS_REQUIRED/);assert.match(html,/NOT_VERIFIED/);
  assert.match(html,/Dependency/);assert.match(html,/Attempt/);assert.match(html,/Review/);
  assert.match(html,/CI/);assert.match(html,/Release/);
  assert.doesNotMatch(html,/<button/i);assert.doesNotMatch(html,/onclick/i);
});
test('genuine Host duty includes exact source and expected evidence readback',()=>{
  const hostTask=task('https://github.com/kaicreator-mm/tally/issues/999',{
    requiredHostEvidence:{kind:'WINDOWS_CODEX_HOST',available:false,reason:'CODEX_HOST_UNREACHABLE'}});
  const html=renderJourneyWorkspace({...snapshot,tasks:[hostTask]},{
    actions:[{...eligible,authorizationType:'NEEDS_HUMAN_HOST',actorRole:'human',
      requiredCapability:'WINDOWS_CODEX_HOST',approvalNeeded:true,reason:'CODEX_HOST_UNREACHABLE'}],
    humanActionInbox:[{kind:'NEEDS_HUMAN_HOST',reason:'CODEX_HOST_UNREACHABLE',
      expectedEvidence:'WINDOWS_CODEX_HOST',readback:hostTask.sourceRef,sourceSha}]});
  assert.match(html,/Human-action inbox/);assert.match(html,/WINDOWS_CODEX_HOST/);
  assert.match(html,/Read back https:\/\/github.com\/kaicreator-mm\/tally\/issues\/999/);
  assert.match(html,/CODEX_HOST_UNREACHABLE/);
});
test('HTML and URLs are escaped and no script execution surface is introduced',()=>{
  const hostile=task('https://evil.test/x',{reason:'<img src=x onerror=alert(1)>'});
  const html=renderJourneyWorkspace({...snapshot,tasks:[hostile]},{
    actions:[{...eligible,reason:hostile.reason}],humanActionInbox:[]});
  assert.doesNotMatch(html,/<img src=x onerror/);assert.match(html,/&lt;img src=x onerror/);
  assert.doesNotMatch(html,/evil.test/);assert.doesNotMatch(html,/javascript:/i);
  assert.equal(__test.url(hostile.sourceRef),'#');
});
test('unverified snapshot stays UNKNOWN with no available action',()=>{
  const html=renderJourneyWorkspace(null,{actions:[eligible],humanActionInbox:[]});
  assert.match(html,/UNKNOWN/);assert.match(html,/No action is available/);
});
test('Task rows match actions by sourceRef, not array position',()=>{
  const two=task('https://github.com/kaicreator-mm/tally/issues/177',{status:'AWAITING_REVIEW',
    reason:'LOCAL_CANDIDATE_READY'});
  const review={...eligible,authorizationType:'START_ELIGIBLE_INDEPENDENT_REVIEW',
    evidence:{sourceRef:two.sourceRef,sourceSha,observedAsOf:observedAt}};
  const html=renderJourneyWorkspace({...snapshot,tasks:[one,two]},
    {actions:[review,eligible],humanActionInbox:[]});
  const first=html.indexOf('issues/172'),second=html.indexOf('issues/177');
  assert.ok(first>0&&second>first);
  assert.ok(html.indexOf('START_ELIGIBLE_INDEPENDENT_REVIEW')>html.indexOf('START_ELIGIBLE_CLAIM'));
  assert.match(html,/START_ELIGIBLE_CLAIM/);assert.match(html,/START_ELIGIBLE_INDEPENDENT_REVIEW/);
});
test('write mode exposes only real action buttons and native readback',()=>{
  const html=renderJourneyWorkspace({...snapshot,result:{status:'PASS',exitCode:0,evidenceDir:'C:/evidence',
    result:{status:'PASS',realAgentEffect:true}}},{
    actions:[{...eligible,kind:'start'}],humanActionInbox:[],mode:'write'});
  assert.match(html,/data-action="start"/);assert.match(html,/Real action readback/);
  assert.match(html,/realAgentEffect/);assert.doesNotMatch(html,/onclick|onerror/i);
});
test('static page contains the real module and no write controls',()=>{
  const html=readFileSync(new URL('./journey-view.html',import.meta.url),'utf8');
  assert.match(html,/<script defer src="\.\/journey-bootstrap\.mjs"><\/script>/);
  assert.doesNotMatch(html,/<script(?:\s+type="module")?\s*>/);
  assert.match(html,/defer/);assert.doesNotMatch(html,/<button/i);
  assert.doesNotMatch(html,/onerror|onclick/i);
});
