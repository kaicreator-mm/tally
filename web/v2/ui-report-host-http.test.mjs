import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiAdsReportHost} from './ui-report-host.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createAlmServer} from './journey-server.mjs';

const sha='a'.repeat(40),repository='org/tally',versionId='version/v2';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const report={status:'CURRENT_OBSERVED',source:{repository,sourceSha:sha,
  issueNumber:11,commentId:91,reportId:'observed1',
  revision:1,kind:'EXECUTION_SUMMARY'}};
const metadata=createUiAdsReportHost({repository,versionId,expectedSha:sha,
  nativeC:{catalog:async()=>({connected:true,source})},
  taskIssueNumbers:[11],acceptedCommentIdsByIssue:{11:[91]},
  publisherBindings:{'ads/task-meta':['agent-bot']},
  readReports:async()=>({status:'CURRENT_OBSERVED',records:[report],
    authorized:false,taskPass:false,releasePass:false})});
const config=defaultAlmConfig({dhPath:'/fake/dh',tallyPath:'/fake/tally',
  tallyBPath:'/fake/b',githubToken:'secretonlyinternal',currentSourceSha:sha,
  workRoot:'/tmp/tally-v2-report-host-http'});
test('source-qualified ADS report is visible over real HTTP without execution permission',async()=>{
  const app=createAlmServer(config,{v2ReadModelHost:{metadata:metadata.metadata}});
  const {origin}=await app.listen();
  try{
    const q='?projectId=org%2Ftally&versionId=version%2Fv2';
    const result=await fetch(origin+'/api/v2/metadata'+q).then(r=>r.json());
    assert.equal(result.connected,true);
    assert.equal(result.reports.length,1);
    assert.equal(result.reports[0].independentlyReviewed,false);
    assert.equal(result.telemetryCoverage,'UNKNOWN');
    assert.equal(result.rosterCoverage,'UNVERIFIED');
    assert.equal(result.attempts.length,0);
    assert.equal(JSON.stringify(result).includes('secretonlyinternal'),false);
  }finally{await app.close();}
});
test('unselected project or version returns NOT_CONNECTED, never a fake report',async()=>{
  const app=createAlmServer(config,{v2ReadModelHost:{metadata:metadata.metadata}});
  const {origin}=await app.listen();
  try{
    for(const q of ['?projectId=evil%2Frepo&versionId=version%2Fv2',
      '?projectId=org%2Ftally&versionId=version%2Fother']){
      const result=await fetch(origin+'/api/v2/metadata'+q).then(r=>r.json());
      assert.equal(result.connected,false);
    }
  }finally{await app.close();}
});
