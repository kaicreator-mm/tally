import test from 'node:test';
import assert from 'node:assert/strict';
import {createMaintenanceReadHost,parseMaintenanceDefectIssue}
  from './maintenance-read-host.mjs';
import {maintenanceCandidateDigest} from
  '../../experiments/v2/maintenance/host-maintenance.mjs';

const SHA='a'.repeat(40);
const TAG_SHA='b'.repeat(40);
const SUBJECT='62f18fd87d5a709f42900e612dcebc02c5c4f365';
const SUMMARY='Genuine #219 maintenance lost-ACK, no second POST.';
// The marker must be the EXACT business digest over the fixture's own
// fields + the resolved tag commit (the read host re-derives it — review P1-1).
const MARKER=maintenanceCandidateDigest({
  sourceIssueRef:'https://github.com/kaicreator-mm/tally/issues/219',
  requirementRef:'https://github.com/kaicreator-mm/tally/issues/392',
  subjectSha:SUBJECT,summary:SUMMARY,
  priorRelease:{version:'v1.5.5',commitSha:TAG_SHA,
    evidenceRef:'https://github.com/kaicreator-mm/tally/releases/tag/v1.5.5',
    immutable:true}});
const body=`Defect candidate against an immutable historical release; the next
version may absorb it only through governed intake.

- source: https://github.com/kaicreator-mm/tally/issues/219
- requirement: https://github.com/kaicreator-mm/tally/issues/392
- subject: ${SUBJECT}
- reported release: v1.5.5 (immutable, never rewritten)
- summary: ${SUMMARY}

maintenance-marker: tally:maintenance:219:${MARKER}
`;
const makeIssue=(text=body)=>({number:437,
  html_url:'https://github.com/kaicreator-mm/tally/issues/437',
  state:'open',created_at:'2026-11-16T18:53:55Z',body:text});
function fixture({issue=makeIssue(),branchSha=SHA,tagType='commit',
  reject=false}={}){
  const calls=[];
  const event=(name,result)=>async args=>{calls.push({name,args});
    if(reject)throw new Error('network unavailable');
    return {data:result};
  };
  const octokit={rest:{
    repos:{getBranch:event('branch',{commit:{sha:branchSha}})},
    issues:{get:event('issue',issue)},
    git:{getRef:event('tagref',{object:{type:tagType,
      sha:tagType==='tag'?'d'.repeat(40):TAG_SHA}}),
      getTag:event('tag',{object:{type:'commit',sha:TAG_SHA}})}
  }};
  const host=createMaintenanceReadHost({octokit,owner:'kaicreator-mm',repo:'tally',
    versionId:'version/v2',expectedSha:SHA,
    now:()=>'2026-11-20T00:00:00Z'});
  return {host,calls};
}

test('canonical #437 + exact HEAD + old commit tag produces advisory only',async()=>{
  const {host,calls}=fixture();
  const actual=await host.readMaintenanceDefects({sourceSha:SHA});
  assert.equal(actual.connected,true);
  assert.equal(actual.sourceSha,SHA);
  assert.equal(actual.priorRelease.commitSha,TAG_SHA);
  assert.equal(actual.priorReleaseEvidence,'REPORTED_RELEASE_UNVERIFIED');
  assert.equal(actual.defects.length,1);
  assert.equal(actual.defects[0].issueRef,
    'https://github.com/kaicreator-mm/tally/issues/437');
  assert.equal(actual.defects[0].subjectSha,SUBJECT,
    'defect subject is NOT the old release tag commit');
  assert.equal(actual.taskPass,false);
  assert.equal(actual.releasePass,false);
  assert.equal(actual.authorized,false);
  assert.deepEqual(calls.map(x=>x.name),['branch','issue','tagref']);
  assert.deepEqual(calls.filter(x=>/^(post|patch|put|delete)$/i.test(x.name)),[]);
});
test('source mismatch and branch drift deny before a defect observation',async()=>{
  const {host,calls}=fixture();
  assert.equal((await host.readMaintenanceDefects({sourceSha:'f'.repeat(40)})).reason,
    'SOURCE_SHA_MISMATCH');
  assert.equal(calls.length,0);
  assert.equal((await fixture({branchSha:'f'.repeat(40)}).host
    .readMaintenanceDefects({sourceSha:SHA})).reason,'VERSION_BRANCH_SOURCE_STALE');
});
test('reject forged issue marker, source, issue number or missing identity',()=>{
  for(const altered of [
    {...makeIssue(),number:438},
    makeIssue(body.replace('maintenance-marker:', 'maintenance-missing:')),
    makeIssue(body.replace('/issues/392','/issues/999')),
    makeIssue(body.replace('v1.5.5 (immutable, never rewritten)','v2.9')),
    {...makeIssue(),pull_request:{}},
  ])assert.equal(parseMaintenanceDefectIssue({
    issue:altered,owner:'kaicreator-mm',repo:'tally'}),null);
});
test('edited body with drifted marker digest is refused, never canonical (P1-1)',async()=>{
  // Same 64-hex marker SHAPE but wrong VALUE: parse succeeds, the read host
  // must re-derive the digest over the issue's own fields + tag commit and
  // refuse DEFECT_MARKER_DIGEST_MISMATCH instead of surfacing the tampered
  // body as canonical.
  const drifted=makeIssue(body.replace(MARKER,'e'.repeat(64)));
  assert.notEqual(parseMaintenanceDefectIssue({
    issue:drifted,owner:'kaicreator-mm',repo:'tally'}),null,
    'parse-level still accepts the well-shaped body; the digest gate is the read host');
  const {host}=fixture({issue:drifted});
  assert.deepEqual(await host.readMaintenanceDefects({sourceSha:SHA}),
    {connected:false,reason:'DEFECT_MARKER_DIGEST_MISMATCH'});
  // Any edited FIELD (summary drift) with the ORIGINAL marker equally fails.
  const editedSummary=makeIssue(body.replace(SUMMARY,
    'Tampered summary that no longer matches the marker binding.'));
  const {host:editedHost}=fixture({issue:editedSummary});
  assert.equal((await editedHost.readMaintenanceDefects({sourceSha:SHA})).reason,
    'DEFECT_MARKER_DIGEST_MISMATCH');
});
test('annotated tags resolve commit but a missing/invalid tag never qualifies',async()=>{
  const annotated=fixture({tagType:'tag'});
  const observed=await annotated.host.readMaintenanceDefects({sourceSha:SHA});
  assert.equal(observed.connected,true);
  assert.ok(annotated.calls.some(x=>x.name==='tag'));
  const invalid=fixture({tagType:'tree'});
  assert.equal((await invalid.host.readMaintenanceDefects({sourceSha:SHA})).reason,
    'PRIOR_RELEASE_TAG_UNVERIFIED');
});
test('GitHub read failures are NOT_RUN-style refusals, not a fabricated candidate',async()=>{
  const {host}=fixture({reject:true});
  assert.deepEqual(await host.readMaintenanceDefects({sourceSha:SHA}),
    {connected:false,reason:'MAINTENANCE_GITHUB_READ_FAILED'});
});
