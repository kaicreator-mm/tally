import test from 'node:test';
import assert from 'node:assert/strict';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
import {createUiNativeAuditHost} from './ui-audit-host.mjs';

const sha='a'.repeat(40),repository='org/tally';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const audit=createUiNativeAuditHost({repository,versionId:'version/v2',
  expectedSha:sha,taskIssueNumbers:[210],
  nativeC:{catalog:async()=>({connected:true,source})},
  readPair:async()=>({
    sourceQualified:true,status:'SOURCE_PAIR_OBSERVED',sourceSha:sha,
    issueRef:'https://github.com/org/tally/issues/210',
    claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
    taskPass:false,releasePass:false,workItem:{claimStatus:'NOT_VERIFIED'},
    claimEvents:{candidateCount:1,candidates:[{sourceSha:sha,
      dispatchId:'observed-comment-123',
      sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',claimAccepted:false}]}
  })
});
const config=defaultAlmConfig({dhPath:'/fake/dh',tallyPath:'/fake/tally',
  tallyBPath:'/fake/b',githubToken:'secretonlyinternal',
  currentSourceSha:sha,workRoot:'/tmp/tally-v2-audit-host-http'});
test('real HTTP Governance audit is observed-only and redacts claim operator',async()=>{
  const app=createAlmServer(config,{v2ReadModelHost:{audit:audit.audit}});
  const {origin}=await app.listen();
  try{
    const result=await fetch(origin+'/api/v2/audit?projectId=org%2Ftally')
      .then(r=>r.json());
    assert.equal(result.connected,true);
    assert.equal(result.records.length,1);
    assert.equal(result.records[0].kind,'audit');
    assert.equal(result.records[0].authority,'OBSERVED');
    assert.equal(result.claimAcceptance,'NOT_VERIFIED');
    assert.equal(JSON.stringify(result).includes('observed-comment-123'),false);
    assert.equal(JSON.stringify(result).includes('secretonlyinternal'),false);
    const foreign=await fetch(origin+'/api/v2/audit?projectId=evil%2Frepo')
      .then(r=>r.json());
    assert.equal(foreign.connected,false);
  }finally{await app.close();}
});
