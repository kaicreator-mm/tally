import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiAdsReportHost} from './ui-report-host.mjs';
const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const catalog={connected:true,source:{repository,sourceSha:sha,
  currentness:'CURRENT',readback:'VERIFIED'}};
const record={status:'CURRENT_OBSERVED',source:{
  repository,sourceSha:sha,issueNumber:11,commentId:91,reportId:'r1',
  revision:1,kind:'EXECUTION_SUMMARY'}};
const observed={status:'CURRENT_OBSERVED',records:[record],
  authorized:false,taskPass:false,releasePass:false};
const setup=extra=>({
  repository,versionId,expectedSha:sha,taskIssueNumbers:[11,12],
  nativeC:{catalog:async()=>catalog},publisherBindings:{'ads/task-meta':['agent-bot']},
  acceptedCommentIdsByIssue:{11:[91],12:[]},
  readReports:async()=>observed,...extra
});
const scope={projectId:repository,versionId};
test('qualified ADS publication stays advisory and all telemetry stays UNKNOWN',async()=>{
  const host=createUiAdsReportHost(setup());
  const snapshot=await host.metadata(scope);
  assert.equal(snapshot.connected,true);
  assert.deepEqual(snapshot.reports.map(r=>r.taskId),['11']);
  assert.equal(snapshot.reports[0].kind,'ads-v5-summary');
  assert.equal(snapshot.reports[0].independentlyReviewed,false);
  assert.equal(snapshot.reports[0].observedAt,null);
  assert.equal(snapshot.rosterCoverage,'UNVERIFIED');
  assert.equal(snapshot.reportCoverage,'PARTIAL');
  assert.equal(snapshot.telemetryCoverage,'UNKNOWN');
  assert.equal(snapshot.lineageCoverage,'UNKNOWN');
});
test('unconfigured publisher identity and comment-grant list are refused',()=>{
  assert.throws(()=>createUiAdsReportHost(setup({publisherBindings:{}})),
    /TRUSTED_ADS_REPORT_SOURCE_CONFIGURATION_REQUIRED/);
  assert.throws(()=>createUiAdsReportHost(setup({
    acceptedCommentIdsByIssue:{11:[],12:[]}})),/ADS_COMMENT_WITNESSES_MISSING/);
  assert.throws(()=>createUiAdsReportHost(setup({
    acceptedCommentIdsByIssue:{99:[91]}})),/CROSS_TASK_ADS_GRANT/);
});
test('cross-project or cross-version query refused without native report read',async()=>{
  let calls=0;
  const host=createUiAdsReportHost(setup({readReports:async()=>{calls++;return observed}}));
  assert.equal((await host.metadata({projectId:'evil/repo',versionId})).connected,false);
  assert.equal((await host.metadata({projectId:repository,versionId:'version/v3'})).connected,false);
  assert.equal(calls,0);
});
test('publication mismatch and provider self-claimed PASS never accepted',async()=>{
  const forged={...observed,authorized:true};
  const host=createUiAdsReportHost(setup({readReports:async()=>forged}));
  assert.equal((await host.metadata(scope)).connected,false);
  const cross=createUiAdsReportHost(setup({readReports:async()=>({...observed,
    records:[{...record,source:{...record.source,issueNumber:12}}]})}));
  assert.equal((await cross.metadata(scope)).reason,'ADS_NATIVE_REPORT_SCOPE_DRIFT');
});
test('correct source is not enough without verified CURRENT ADS projection',async()=>{
  for(const status of ['STALE','UNKNOWN','PARTIAL','INVALID_OBSERVATION']){
    const host=createUiAdsReportHost(setup({readReports:async()=>({
      ...observed,status})}));
    assert.equal((await host.metadata(scope)).connected,false);
  }
});
test('source SHA changes before complete read make report set NOT_CONNECTED',async()=>{
  let times=0;
  const host=createUiAdsReportHost(setup({nativeC:{catalog:async()=>({
    ...catalog,source:{...catalog.source,sourceSha:++times===1?sha:'b'.repeat(40)}})}}));
  assert.equal((await host.metadata(scope)).reason,'PINNED_VERSION_CHANGED_DURING_READ');
});
