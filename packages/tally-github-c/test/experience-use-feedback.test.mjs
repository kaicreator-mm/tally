import test from 'node:test';
import assert from 'node:assert/strict';
import {feedbackReportId,projectExperienceUseFeedback} from '../src/experience-use-feedback.mjs';
const sha='a'.repeat(40),publisherSha='b'.repeat(40);
const repo='kaicreator-mm/tally',issue=274,issueRef='https://github.com/'+repo+'/issues/'+issue;
const review=issueRef+'#issuecomment-123',verify=issueRef+'#issuecomment-456';
function make(phase='PRESENTED',overrides={}) {
  const witness={kind:phase,ref:review,
    decision:phase==='ACCEPTED'?'APPROVE':phase==='SKIPPED'?'REJECT':null,
    independent:['INDEPENDENTLY_CHECKED','OBSERVED_EFFECT'].includes(phase)};
  const base={repository:repo,issueNumber:issue,taskRef:'#274',attemptRef:'attempt:1',
    sourceSha:sha,publisherSourceSha:publisherSha,adsRevision:'2c395c55569fe2edf3f4135f12d3d069e526d0b7',
    publisherNamespace:'tally-experience-feedback',revision:1,
    experienceRef:issueRef,procedureRevision:'procedure-v1',phase,
    epistemic:['INDEPENDENTLY_CHECKED','OBSERVED_EFFECT'].includes(phase)?
      'OBSERVED':'UNKNOWN',observedAt:'2026-11-10T00:00:00Z',
    actor:{role:'builder',operator_kind:'other',operator_id:'web:275',
      session_ref:review,transport_actor:'kaicreator-mm'},
    sourceRefs:[review],phaseEvidence:witness,
    verifierRefs:phase==='INDEPENDENTLY_CHECKED'?[verify]:[],
    clearanceRef:review,supersedes:null,
    sourceWitness:{repository:repo,issueNumber:issue,sourceSha:sha,
      currentness:'CURRENT',readback:'VERIFIED'}};
  const dto={...base,...overrides};dto.reportId=overrides.reportId??feedbackReportId(dto);
  return dto;
}
test('every one of six permitted ADS phase envelopes is a candidate only',()=>{
  for(const phase of ['PRESENTED','ACCEPTED','APPLIED','SKIPPED',
    'INDEPENDENTLY_CHECKED','OBSERVED_EFFECT']){
    const r=projectExperienceUseFeedback(make(phase));
    assert.equal(r.status,'CANDIDATE_ONLY',phase+': '+r.reason);
    assert.equal(r.envelope.schema,'ads/github/v5');
    assert.equal(r.envelope.kind,'report');
    assert.equal(r.envelope.data.report_kind,'EXPERIENCE_USE_FEEDBACK');
    assert.equal(r.envelope.data.phase,phase);
    assert.match(r.projectionDigest,/^[0-9a-f]{64}$/);
    assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
  }
});
test('exact same source and identity replays deterministically',()=>{
  const a=projectExperienceUseFeedback(make()),b=projectExperienceUseFeedback(make());
  assert.equal(a.dataDigest,b.dataDigest);
  assert.equal(a.projectionDigest,b.projectionDigest);
  assert.equal(a.envelope.data.report_id,b.envelope.data.report_id);
});
test('forged or missing human approval refuses adoption',()=>{
  assert.equal(projectExperienceUseFeedback(make('ACCEPTED',{
    phaseEvidence:{kind:'ACCEPTED',ref:review,decision:null,independent:false}
  })).reason,'FEEDBACK_PHASE_QUALIFICATION_MISSING');
  assert.equal(projectExperienceUseFeedback(make('SKIPPED',{
    phaseEvidence:{kind:'SKIPPED',ref:review,decision:'APPROVE',independent:false}
  })).reason,'FEEDBACK_PHASE_QUALIFICATION_MISSING');
});
test('outcomes need independent current evidence',()=>{
  assert.equal(projectExperienceUseFeedback(make('OBSERVED_EFFECT',{
    phaseEvidence:{kind:'OBSERVED_EFFECT',ref:review,decision:null,independent:false}
  })).reason,'FEEDBACK_PHASE_QUALIFICATION_MISSING');
  assert.equal(projectExperienceUseFeedback(make('INDEPENDENTLY_CHECKED',{
    verifierRefs:[]
  })).reason,'FEEDBACK_PHASE_QUALIFICATION_MISSING');
});
test('unqualified source and stale SHA cannot be reported as current',()=>{
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{
    sourceWitness:{repository:repo,issueNumber:issue,sourceSha:sha,
      currentness:'STALE',readback:'VERIFIED'}
  })).reason,'FEEDBACK_SOURCE_NOT_CURRENT');
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{sourceSha:'wrong'})).reason,
    'FEEDBACK_SOURCE_IDENTITY_INVALID');
});
test('unknown/private fields are rejected before projection',()=>{
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{
    secretToken:'should-not-appear'
  })).reason,'DTO_UNKNOWN_OR_OVERSIZED');
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{
    actor:{role:'builder',operator_kind:'other',operator_id:'me',
      session_ref:review,transport_actor:'kaicreator-mm',rawTrace:'private'}
  })).reason,'FEEDBACK_ACTOR_INVALID');
});
test('identity-changing report ID cannot be borrowed from another source',()=>{
  const v=make();v.reportId='experience-use:'+('f').repeat(32);
  assert.equal(projectExperienceUseFeedback(v).reason,'FEEDBACK_REPORT_ID_MISMATCH');
});
test('revision 1 cannot supersede and revision 2 must cite previous native revision',()=>{
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{supersedes:{
    revision:1,data_digest:'a'.repeat(64),comment_ref:review
  }})).reason,'FEEDBACK_REVISION_CHAIN_INVALID');
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{revision:2,
    supersedes:null})).reason,'FEEDBACK_REVISION_CHAIN_INVALID');
  const v=make('PRESENTED',{revision:2,supersedes:{revision:1,
    data_digest:'a'.repeat(64),comment_ref:review}});
  assert.equal(projectExperienceUseFeedback(v).status,'CANDIDATE_ONLY');
});
test('foreign repo in prior revision is refused',()=>{
  const v=make('PRESENTED',{revision:2,supersedes:{revision:1,
    data_digest:'a'.repeat(64),
    comment_ref:'https://github.com/foreign/repo/issues/274#issuecomment-123'}});
  assert.equal(projectExperienceUseFeedback(v).reason,'FEEDBACK_SUPERSESSION_CROSS_ISSUE');
});
test('phase evidence must be included in bounded source references',()=>{
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{sourceRefs:[verify]})).reason,
    'FEEDBACK_PHASE_WITNESS_INVALID');
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{
    sourceRefs:[review,review]})).reason,'FEEDBACK_SOURCE_IDENTITY_INVALID');
});
test('schema candidate does not grant K or Task authority',()=>{
  const v=projectExperienceUseFeedback(make('APPLIED'));
  assert.equal(v.status,'CANDIDATE_ONLY');
  assert.equal(v.authorized,false);assert.equal(v.admitted,false);
  assert.equal(v.taskPass,false);
});

test('wrong ADS revision is not source-qualified',()=>{
  assert.equal(projectExperienceUseFeedback(make('PRESENTED',{
    adsRevision:'c'.repeat(40)
  })).reason,'FEEDBACK_SOURCE_IDENTITY_INVALID');
});
