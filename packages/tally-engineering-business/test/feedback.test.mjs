import test from 'node:test';
import assert from 'node:assert/strict';
import {projectFeedbackCases} from '../src/feedback.mjs';

const observedAsOf='2026-10-31T07:30:00.000Z',tallySha='2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9',dhSha='061376f4a023070f215f9feffad5c24d96d503cb';
const tallyIssue=number=>`https://github.com/kaicreator-mm/tally/issues/${number}`;
const dhIssue=number=>`https://github.com/kaicreator-mm/domain-harness/issues/${number}`;
const base1393={id:'tally-86-to-dh-1393',sourceProject:'tally',sourceVersion:'v2',sourceIssueRef:tallyIssue(86),sourceSha:tallySha,
  classification:'SDK_CONTRACT',ownerRepo:'kaicreator-mm/domain-harness',ownerVersion:'v0.10.3',ownerIssueRef:dhIssue(1393),
  observedEvidence:[tallyIssue(86)],fingerprint:'provider-integrity',stages:{
    triaged:{ref:dhIssue(1393)},ownerAccepted:{ref:dhIssue(1393)},fixReview:{ref:dhIssue(1393),sourceSha:dhSha},
    downstreamRetest:{ref:tallyIssue(172),sourceSha:dhSha,selectedPath:'PASS'}}};
const base1394={...base1393,id:'dh-1394-doc-guidance',classification:'DOC_GUIDANCE',ownerIssueRef:dhIssue(1394),
  fingerprint:'docstore-doc-guidance',stages:{triaged:{ref:dhIssue(1394)}}};

test('1393 keeps Release missing NOT_PROVEN even with real downstream retest evidence',()=>{
  const result=projectFeedbackCases({observedAsOf,cases:[base1393]});
  const row=result.cases[0];
  assert.equal(row.stages.OWNER_ACCEPTED.status,'OBSERVED');assert.equal(row.stages.FIX_PR_REVIEWED.status,'OBSERVED');
  assert.equal(row.stages.RELEASED.status,'NOT_RUN');assert.equal(row.stages.DOWNSTREAM_RETESTED.status,'OBSERVED');
  assert.equal(row.outcome,'NOT_PROVEN');assert.equal(row.targetVersionChanged,true);
  assert.match(row.mustNot.join(' '),/DO_NOT_INFER_DOWNSTREAM_PASS_FROM_UPSTREAM_CLOSED/);
  assert.equal(result.coverage.consumerConfirmedCount,0);assert.equal(result.coverage.notProvenCount,1);
});

test('1394 documentation concern is not merged into provider integrity',()=>{
  const result=projectFeedbackCases({observedAsOf,cases:[base1393,base1394]});
  assert.equal(result.cases[1].classification,'DOC_GUIDANCE');assert.equal(result.duplicates.length,0);
  assert.equal(result.cases[1].outcome,'NOT_PROVEN');
});

test('cross-owner source-of-truth links produce candidates not automatic closure',()=>{
  const owners=[
    {id:'ads-1348',sourceProject:'tally',ownerRepo:'kaicreator-mm/ai-development-standard',ownerIssueRef:'https://github.com/kaicreator-mm/ai-development-standard/issues/1348'},
    {id:'task-meta-80',sourceProject:'tally',ownerRepo:'kaicreator-mm/task-meta',ownerIssueRef:'https://github.com/kaicreator-mm/task-meta/issues/80'},
    {id:'tally-136',sourceProject:'tally',ownerRepo:'kaicreator-mm/tally',ownerIssueRef:tallyIssue(136)}
  ].map(row=>({...base1393,...row,ownerVersion:'next',fingerprint:'source-of-truth',stages:{triaged:{ref:row.ownerIssueRef}}}));
  const result=projectFeedbackCases({observedAsOf,cases:owners});
  assert.equal(result.duplicates.length,3);assert.equal(result.duplicates[0].status,'POTENTIAL_DUPLICATE');
  assert.equal(result.duplicates[0].autoMerge,false);assert.equal(result.coverage.consumerConfirmedCount,0);
  assert.equal(result.coverage.targetVersionChangeCount,3);
});

test('local incorrect pin and valid deferral are legitimate non-upstream outcomes',()=>{
  const local={...base1393,id:'local-incorrect-pin',classification:'LOCAL_MISUSE',localMisuseEvidence:tallyIssue(136),stages:{}};
  const deferred={...base1393,id:'validly-deferred',stages:{deferred:{ref:dhIssue(1393)}}};
  const result=projectFeedbackCases({observedAsOf,cases:[local,deferred]});
  assert.equal(result.cases[0].outcome,'NO_UPSTREAM_ISSUE_REQUIRED');
  assert.equal(result.cases[1].outcome,'DEFERRED_VALIDLY');
  assert.equal(result.coverage.localMisuseCount,1);assert.equal(result.coverage.deferredValidlyCount,1);
});

test('synthetic complete chain proves predicate only and unavailable source stays UNKNOWN',()=>{
  const complete={...base1393,id:'synthetic-complete-chain',stages:{...base1393.stages,
    release:{ref:'https://github.com/kaicreator-mm/domain-harness/releases/tag/v0.10.3',sourceSha:dhSha}}};
  const unavailable={...base1393,id:'source-unavailable',sourceAvailable:false};
  const result=projectFeedbackCases({observedAsOf,cases:[complete,unavailable]});
  assert.equal(result.cases[0].outcome,'CONSUMER_CONFIRMED');
  assert.equal(result.cases[1].outcome,'UNKNOWN');assert.equal(result.cases[1].stages.REPORTED.status,'UNKNOWN');
  assert.equal(result.coverage.consumerConfirmedCount,1);assert.equal(result.coverage.unknownCount,1);
});

test('malformed or duplicate case identity cannot manufacture a closed loop',()=>{
  assert.equal(projectFeedbackCases(null).status,'UNKNOWN');
  const result=projectFeedbackCases({observedAsOf,cases:[base1393,base1393]});
  assert.equal(result.status,'UNKNOWN');assert.equal(result.coverage.taskPass,false);assert.equal(result.coverage.releasePass,false);
  const badSha=projectFeedbackCases({observedAsOf,cases:[{...base1393,stages:{...base1393.stages,
    triaged:{ref:dhIssue(1393),sourceSha:'not-a-sha'}}}]});
  assert.equal(badSha.cases[0].stages.TRIAGED.status,'UNKNOWN');assert.equal(badSha.cases[0].outcome,'NOT_PROVEN');
  const dotRef={...base1393,sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/../../pull/9'};
  assert.equal(projectFeedbackCases({observedAsOf,cases:[dotRef]}).status,'UNKNOWN');
  const objectRef={...base1393,sourceIssueRef:{toString:()=>tallyIssue(86)}};
  assert.equal(projectFeedbackCases({observedAsOf,cases:[objectRef]}).status,'UNKNOWN');
});
