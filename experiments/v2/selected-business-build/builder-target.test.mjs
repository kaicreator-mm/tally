import test from 'node:test';
import assert from 'node:assert/strict';
import {resolveBuilderBuildTarget,sealBuilderShimTarget,
  exactSelectedWriteSet} from './builder-target.mjs';
const selected={T06M_TARGET_REPOSITORY:'kaicreator-mm/tally-reps-test',
  T06M_TARGET_ISSUE_NUMBER:'27',T06M_TARGET_VERSION_REF:'main',
  T06M_TARGET_CANDIDATE_PATH:'src/health.mjs'};
test('legacy Builder build keeps historical sealed source',()=>{
  const target=resolveBuilderBuildTarget({});
  assert.equal(target.mode,'HISTORICAL_DEFAULT');
  assert.equal(target.repository,'kaicreator-mm/tally');
  assert.equal(target.versionId,'version/v2');
  const text="const ISSUES=new Set([172]);\nconst REVIEWER_ISSUE=179;\nconst OWNER='kaicreator-mm',REPO='tally';";
  assert.equal(sealBuilderShimTarget(text,target),text);
});
test('selected new repository, task, ref and candidate are one BUILD identity',()=>{
  const target=resolveBuilderBuildTarget(selected);
  assert.equal(target.mode,'SELECTED_BUILD');
  assert.equal(target.issueNumber,27);
  assert.equal(target.repository,'kaicreator-mm/tally-reps-test');
  assert.equal(target.versionId,'main');
  const text="const ISSUES=new Set([172]);\nconst REVIEWER_ISSUE=179;\nconst OWNER='kaicreator-mm',REPO='tally';";
  const result=sealBuilderShimTarget(text,target);
  assert.match(result,/const ISSUES=new Set\(\[27\]\)/);
  assert.match(result,/const OWNER="kaicreator-mm",REPO="tally-reps-test"/);
  assert.doesNotMatch(result,/const ISSUES=new Set\(\[172\]\)/);
});
test('target-specific Claim cannot silently use historical Builder BUILD',()=>{
  assert.throws(()=>resolveBuilderBuildTarget({T06L_CLAIM:'1',
    T06L_TARGET_REPOSITORY:'kaicreator-mm/tally-reps-test',
    T06L_TARGET_ISSUE_NUMBER:'27'}),/T06M_CLAIM_BUILD_TARGET_MISMATCH/);
});
test('selected Builder and accepted Claim target must bind same Issue',()=>{
  assert.throws(()=>resolveBuilderBuildTarget({...selected,
    T06L_CLAIM:'1',T06L_TARGET_REPOSITORY:selected.T06M_TARGET_REPOSITORY,
    T06L_TARGET_ISSUE_NUMBER:'28'}),/T06M_CLAIM_BUILD_TARGET_MISMATCH/);
  const target=resolveBuilderBuildTarget({...selected,T06L_CLAIM:'1',
    T06L_TARGET_REPOSITORY:selected.T06M_TARGET_REPOSITORY,
    T06L_TARGET_ISSUE_NUMBER:selected.T06M_TARGET_ISSUE_NUMBER});
  assert.equal(target.issueNumber,27);
});
test('partial/foreign/candidate traversal selected target denied',()=>{
  for(const bad of [
    {T06M_TARGET_REPOSITORY:'owner/repo'},
    {...selected,T06M_TARGET_REPOSITORY:'owner/repo/extra'},
    {...selected,T06M_TARGET_ISSUE_NUMBER:'0'},
    {...selected,T06M_TARGET_ISSUE_NUMBER:'27.0'},
    {...selected,T06M_TARGET_VERSION_REF:'feature/unsafe'},
    {...selected,T06M_TARGET_CANDIDATE_PATH:'../.git/config'},
    {...selected,T06M_TARGET_CANDIDATE_PATH:'src//health.mjs'},
    {...selected,T06M_TARGET_CANDIDATE_PATH:'C:\\secret.txt'},
    {...selected,T06M_TARGET_CANDIDATE_PATH:'/etc/passwd'},
  ])assert.throws(()=>resolveBuilderBuildTarget(bad),TypeError);
});
test('new selected Builder needs exact frozen CAS row and candidate',()=>{
  const target=resolveBuilderBuildTarget(selected);
  const sha='a'.repeat(40),digest='b'.repeat(64);
  const row={phase:'FROZEN_DAG_WRITESET',repository:target.repository,
    issueNumber:27,sourceSha:sha,writeSetSha256:digest,
    document:JSON.stringify(['src/health.mjs','src/health.test.mjs'])};
  const pins={...target,sourceSha:sha,digest};
  assert.equal(exactSelectedWriteSet(row,pins),true);
  assert.equal(exactSelectedWriteSet({...row,repository:'kaicreator-mm/tally'},pins),false);
  assert.equal(exactSelectedWriteSet({...row,sourceSha:'c'.repeat(40)},pins),false);
  assert.equal(exactSelectedWriteSet({...row,document:'src/health.mjs.evil'},pins),false);
  assert.equal(exactSelectedWriteSet({...row,document:JSON.stringify(['src/health.mjs','src/health.mjs'])},pins),false);
});
