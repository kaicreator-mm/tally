import test from 'node:test';
import assert from 'node:assert/strict';
import {reconstructVersionTimeline,formatLineageTable} from '../src/lineage.mjs';

const repo='kaicreator-mm/domain-harness';
const observedAsOf='2026-10-31T04:00:00.000Z';
const adsSourceRef='https://github.com/kaicreator-mm/ai-development-standard/commit/2c395c55569fe2edf3f4135f12d3d069e526d0b7';
const subject=(version,sourceSha,product=530,l2=526)=>({version,sourceSha,observedAsOf,
  sourceRef:`https://github.com/${repo}/commit/${sourceSha}`,
  productFreezeRef:`https://github.com/${repo}/issues/${product}`,
  l2FreezeRef:`https://github.com/${repo}/issues/${l2}`,adsSourceRef});
const coverage={paginationComplete:true,schema:'ads/github/v3',dialect:'ai-dev/event-v2',currentnessComplete:true};
const issue=(number,state='closed',disposition='actionable',extra={})=>({number,state,disposition,
  ref:`https://github.com/${repo}/issues/${number}`,revision:`rev-${number}`,...extra});
const pr=(issueNumber,number,headSha,mergedSha,extra={})=>({issueNumber,number,headSha,mergedSha,
  merged:true,containedInSubject:true,ref:`https://github.com/${repo}/pull/${number}`,...extra});
const review=(prNumber,sourceSha,ref)=>({prNumber,sourceSha,verdict:'PASS',ref});
const check=(prNumber,sourceSha,ref,kind='CI')=>({prNumber,sourceSha,kind,status:'PASS',ref});
const evidence1013={
  coverage,
  issues:[issue(1013,'closed','historical',{acceptanceRequired:true,requiredChecks:['CI'],
    historicalFacts:[{status:'DECLARED_PASS',sourceSha:'0f233b4cbd2b8c99dfa1b11e877f29777069d44f'}]})],
  pullRequests:[pr(1013,1016,'13a7745120e4a9206ea6cf2173d0ae8f6ccf59b5','93f4cc911afc373815b948dc6c38a1ed72bee524')],
  reviews:[review(1016,'13a7745120e4a9206ea6cf2173d0ae8f6ccf59b5','https://github.com/kaicreator-mm/domain-harness/pull/1016')],
  checks:[check(1016,'13a7745120e4a9206ea6cf2173d0ae8f6ccf59b5','https://github.com/kaicreator-mm/domain-harness/pull/1016')]
};
const evidence1354={
  coverage,
  issues:[issue(1264,'open','historical'),issue(1354,'closed','actionable',{requiredChecks:['CI']})],
  pullRequests:[pr(1264,1269,'UNKNOWN','UNKNOWN',{merged:false,containedInSubject:false}),
    pr(1354,1355,'5079210cf872a12e07d2ffd8bc2c339d0d954b46','d2e6d9eb8028cd1be2359fd3cf2b7678d6f6646f')],
  reviews:[review(1355,'5079210cf872a12e07d2ffd8bc2c339d0d954b46','https://github.com/kaicreator-mm/domain-harness/pull/1355')],
  checks:[check(1355,'5079210cf872a12e07d2ffd8bc2c339d0d954b46','https://github.com/kaicreator-mm/domain-harness/pull/1355')]
};

test('three real version families preserve source, acceptance and historical boundaries',()=>{
  const v07=reconstructVersionTimeline(repo,subject('v0.7','edd6ac94e56ba61355f1e38397022c8109e17669'),{
    coverage,issues:[issue(530,'open','superseded'),issue(586,'open','historical')]});
  assert.equal(v07.items.length,2);assert.equal(v07.coverage.uniqueIssueCount,2);
  assert.equal(v07.items[0].disposition,'superseded');assert.equal(v07.items[0].completion,'UNKNOWN');
  assert.equal(v07.items[1].disposition,'historical');assert.equal(v07.coverage.realCompletedCount,0);

  const v08=reconstructVersionTimeline(repo,subject('v0.8','93f4cc911afc373815b948dc6c38a1ed72bee524',1005,1005),evidence1013);
  assert.equal(v08.items[0].completion,'SOURCE_MERGED');
  assert.equal(v08.items[0].publicAcceptance,'PUBLIC_ACCEPTANCE_NOT_PROVEN');
  assert.equal(v08.items[0].reviewStatus,'REVIEWED_EXACT_SHA');
  assert.equal(v08.coverage.realCompletedCount,0);assert.equal(v08.coverage.taskPass,false);

  const v081=reconstructVersionTimeline(repo,subject('v0.8.1','a1882f5df3decb5e49736a20eea91c95b18b5997',1005,1005),{
    coverage,issues:[issue(1100),issue(1101)],regressions:[
      {issueNumber:1100,previousIssueNumber:1013,previousVersion:'v0.8',status:'OBSERVED',
        evidenceRef:'https://github.com/kaicreator-mm/domain-harness/issues/1100',sourceSha:'0f233b4cbd2b8c99dfa1b11e877f29777069d44f'},
      {issueNumber:1101,previousIssueNumber:1013,previousVersion:'v0.8',status:'OBSERVED',
        evidenceRef:'https://github.com/kaicreator-mm/domain-harness/issues/1101',sourceSha:'0f233b4cbd2b8c99dfa1b11e877f29777069d44f'}]});
  assert.equal(v081.edges.filter(x=>x.kind==='SUBSEQUENT_REGRESSION'&&x.status==='OBSERVED').length,2);
  assert.equal(v081.items.some(x=>x.completion==='REAL_COMPLETED'),false);
  assert.equal(v08.items[0].historicalFacts[0].status,'DECLARED_PASS');

  const v0102=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),evidence1354);
  const fixed=v0102.items.find(x=>x.issue===`${repo}#1354`);
  assert.equal(fixed.completion,'REAL_COMPLETED');assert.equal(fixed.reviewStatus,'REVIEWED_EXACT_SHA');
  assert.equal(v0102.items.find(x=>x.issue===`${repo}#1264`).completion,'INCOMPLETE');
  assert.equal(v0102.coverage.realCompletedCount,1);assert.equal(v0102.coverage.releasePass,false);
  assert.match(formatLineageTable(v0102),/pull\/1355/);
});

test('merged source with stale review or missing required CI is not real completion',()=>{
  const stale={...evidence1013,reviews:[review(1016,'a'.repeat(40),'https://github.com/kaicreator-mm/domain-harness/pull/1016')]};
  assert.equal(reconstructVersionTimeline(repo,subject('v0.8','93f4cc911afc373815b948dc6c38a1ed72bee524',1005,1005),stale).items[0].completion,'SOURCE_MERGED');
  const noCi={...evidence1013,checks:[]};
  assert.equal(reconstructVersionTimeline(repo,subject('v0.8','93f4cc911afc373815b948dc6c38a1ed72bee524',1005,1005),noCi).items[0].completion,'SOURCE_MERGED');
  const physical={...evidence1354,issues:evidence1354.issues.map(row=>row.number===1354?{...row,physicalRequired:true}:row)};
  const physicalResult=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),physical);
  assert.equal(physicalResult.items.find(x=>x.issue===`${repo}#1354`).completion,'SOURCE_MERGED');
});

test('correction and rebuild are idempotent and never rewrite historical facts',()=>{
  const input=structuredClone(evidence1354),before=structuredClone(input);
  const first=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),input);
  const second=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),input);
  assert.deepEqual(first,second);assert.deepEqual(input,before);
  const corrected={...input,corrections:[{sourceRef:'https://github.com/kaicreator-mm/domain-harness/pull/1355',action:'REVOCATION'}]};
  const result=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),corrected);
  assert.equal(result.items.find(x=>x.issue===`${repo}#1354`).completion,'UNKNOWN');
  assert.equal(result.edges.find(x=>x.kind==='ISSUE_PULL_REQUEST'&&x.sourceRef.endsWith('/1355')).status,'STALE');
});

test('pagination schema currentness and duplicate evidence fail closed without zero-cost claims',()=>{
  for(const bad of [{paginationComplete:false},{schema:'old'},{currentnessComplete:false}]){
    const result=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),
      {...evidence1354,coverage:{...coverage,...bad}});
    assert.equal(result.subjectStatus,'UNKNOWN');assert.equal(result.coverage.unknownCount,2);
    assert.equal(result.coverage.realCompletedCount,0);assert.equal(result.coverage.modelCoverage,'UNKNOWN');
    assert.equal(result.coverage.activeTime,'UNKNOWN');
  }
  const incomplete=reconstructVersionTimeline(repo,subject('v0.8','93f4cc911afc373815b948dc6c38a1ed72bee524',1005,1005),
    {...evidence1013,coverage:{...coverage,paginationComplete:false}});
  assert.equal(incomplete.items[0].publicAcceptance,'UNKNOWN');
  const duplicate={...evidence1354,issues:[evidence1354.issues[1],evidence1354.issues[1]]};  const result=reconstructVersionTimeline(repo,subject('v0.10.2','cee14399d9d0af4e20e34b9111861b16a465b690',1351,1351),duplicate);
  assert.equal(result.items.length,1);assert.equal(result.items[0].completion,'UNKNOWN');
  assert.equal(result.coverage.uniqueIssueCount,1);
});

test('unqualified subject and unproved regression stay UNKNOWN',()=>{
  const invalid=reconstructVersionTimeline(repo,null,null);
  assert.equal(invalid.subjectStatus,'UNKNOWN');assert.equal(invalid.coverage.type,'ProvenanceCoverage');
  assert.equal(invalid.coverage.pagination,'UNKNOWN');assert.equal(invalid.coverage.releasePass,false);
  const result=reconstructVersionTimeline(repo,subject('v0.8.1','a1882f5df3decb5e49736a20eea91c95b18b5997',1005,1005),{
    coverage,issues:[issue(1100)],regressions:[{issueNumber:1100,previousIssueNumber:'not-a-number',status:'OBSERVED',
      evidenceRef:'https://github.com/kaicreator-mm/domain-harness/issues/1100'}]});
  const regression=result.edges.find(x=>x.kind==='SUBSEQUENT_REGRESSION');
  assert.equal(regression.status,'UNKNOWN');assert.equal(regression.from,'UNKNOWN');
  assert.equal(result.coverage.releasePass,false);
});

const unitRepo='example/unit',unitSha='1'.repeat(40),unitHead='2'.repeat(40),unitMerge='3'.repeat(40);
const unitSubject={version:'v1',sourceSha:unitSha,observedAsOf,
  sourceRef:`https://github.com/${unitRepo}/commit/${unitSha}`,
  productFreezeRef:`https://github.com/${unitRepo}/issues/3`,
  l2FreezeRef:`https://github.com/${unitRepo}/issues/4`,
  adsSourceRef:`https://github.com/${unitRepo}/commit/${unitSha}`};
const unitEvidence=()=>({coverage,
  issues:[{number:1,state:'closed',disposition:'actionable',ref:`https://github.com/${unitRepo}/issues/1`,
    physicalRequired:true,acceptanceRequired:true,requiredChecks:['CI']}],
  pullRequests:[{issueNumber:1,number:2,headSha:unitHead,mergedSha:unitMerge,merged:true,containedInSubject:true,
    ref:`https://github.com/${unitRepo}/pull/2`}],
  reviews:[{prNumber:2,sourceSha:unitHead,verdict:'PASS',ref:`https://github.com/${unitRepo}/pull/2`}],
  checks:[{prNumber:2,sourceSha:unitHead,kind:'CI',status:'PASS',ref:`https://github.com/${unitRepo}/pull/2`}],
  attempts:[{issueNumber:1,id:'attempt:unit:1',sourceSha:unitHead,physical:true,state:'VALIDATED',
    ref:`https://github.com/${unitRepo}/issues/1`,claimRef:`https://github.com/${unitRepo}/issues/1#issuecomment-1`}],
  acceptances:[{issueNumber:1,sourceSha:unitHead,status:'PROVEN',ref:`https://github.com/${unitRepo}/pull/2`}]});

test('attempt claim acceptance and unknown corrections revoke the whole current qualification',()=>{
  const positive=reconstructVersionTimeline(unitRepo,unitSubject,unitEvidence());
  assert.equal(positive.items[0].completion,'REAL_COMPLETED');
  assert.equal(positive.edges.filter(x=>['CLAIMED_PHYSICAL_ATTEMPT','ACCEPTED_CLAIM_ATTEMPT'].includes(x.kind)).length,2);
  const base=unitEvidence();
  for(const [sourceRef,action] of [[base.attempts[0].ref,'REVOCATION'],[base.attempts[0].claimRef,'REDACT'],
    [base.checks[0].ref,'FUTURE_CORRECTION_KIND']]){
    const result=reconstructVersionTimeline(unitRepo,unitSubject,{...base,corrections:[{sourceRef,action}]});
    assert.equal(result.items[0].completion,'UNKNOWN');assert.equal(result.items[0].physicalAttemptId,null);
    assert.equal(result.items[0].reviewStatus,'UNKNOWN');assert.equal(result.items[0].checksStatus,'UNKNOWN');
    assert.equal(result.items[0].publicAcceptance,'UNKNOWN');
  }
});

test('malformed rows and relationship ref mismatches fail closed instead of crashing',()=>{
  for(const key of ['issues','pullRequests','reviews','checks','attempts','acceptances','regressions','corrections']){
    const input={...unitEvidence(),[key]:[null]};
    const result=reconstructVersionTimeline(unitRepo,unitSubject,input);
    assert.equal(result.subjectStatus,'UNKNOWN');
    if(key==='issues')assert.equal(result.items.length,0);
    else assert.equal(result.items[0].completion,'UNKNOWN');
  }
  assert.equal(reconstructVersionTimeline(unitRepo,unitSubject,null).subjectStatus,'UNKNOWN');
  const wrongPr={...unitEvidence(),pullRequests:[{...unitEvidence().pullRequests[0],ref:`https://github.com/${unitRepo}/issues/999`}]};
  assert.equal(reconstructVersionTimeline(unitRepo,unitSubject,wrongPr).items[0].completion,'INCOMPLETE');
  const wrongAcceptance={...unitEvidence(),acceptances:[{...unitEvidence().acceptances[0],ref:`https://github.com/${unitRepo}/commit/${unitHead}`}]};
  const result=reconstructVersionTimeline(unitRepo,unitSubject,wrongAcceptance);
  assert.equal(result.items[0].completion,'SOURCE_MERGED');assert.equal(result.items[0].publicAcceptance,'PUBLIC_ACCEPTANCE_NOT_PROVEN');
  const badNumber={...unitEvidence(),pullRequests:[{...unitEvidence().pullRequests[0],number:'2|x',ref:`https://github.com/${unitRepo}/pull/2|x`}]};
  assert.equal(reconstructVersionTimeline(unitRepo,unitSubject,badNumber).items[0].completion,'INCOMPLETE');
  const missingCorrectionTarget={...unitEvidence(),corrections:[{action:'REVOCATION'}]};
  const corrected=reconstructVersionTimeline(unitRepo,unitSubject,missingCorrectionTarget);
  assert.equal(corrected.subjectStatus,'UNKNOWN');assert.equal(corrected.items[0].completion,'UNKNOWN');
});
