/** Real GitHub-only observation of public task-meta ADS v5 reports. */
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {pathToFileURL} from 'node:url';
import {readIssueReportsFromClient} from '../../../packages/tally-github-native-c/src/read-only.mjs';

const require=createRequire(new URL('../../../packages/tally-github-native-c/package.json',import.meta.url));
const {Octokit}=await import(pathToFileURL(require.resolve('@octokit/rest')).href);

const namespace='task-meta:kaicreator-mm/task-meta-e2e';
const octokit=new Octokit({request:{timeout:15000}});
const publisherSha='5b7072d78016052b1ea816b2b36322b946ef5c78';
const unavailableSha='1c8ec325c2d4df61246e02d3f01d28f31ea9d91a';
const base={octokit,owner:'kaicreator-mm',repo:'task-meta-e2e',issueNumber:9,
  publisherBindings:{[namespace]:['kaicreator-mm']},limit:1000};
const live=await readIssueReportsFromClient({...base,expectedSourceSha:publisherSha,
  publisherRepository:'kaicreator-mm/task-meta',acceptedCommentIds:[6381704604,6381719186,6381725137]});
assert.equal(live.reportCount,3,JSON.stringify({status:live.status,reason:live.reason}));
const unavailable=await readIssueReportsFromClient({...base,expectedSourceSha:unavailableSha,acceptedCommentIds:[6382627467]});
assert.equal(unavailable.status,'UNKNOWN');
assert.ok(['GITHUB_NATIVE_READ_FAILED','GITHUB_RESOURCE_NOT_FOUND'].includes(unavailable.reason));
const expected=new Map([[6381704604,'zcode'],[6381719186,'claude'],[6381725137,'codex']]);
const actors=[];
for(const row of live.records){
  const id=row.source.commentId,envelope=row.advisory;
  assert.equal(row.source.sourceSha,publisherSha);
  assert.equal(row.source.namespace,namespace);
  assert.equal(row.source.dataDigest.length,64);
  assert.equal(envelope.actorOperatorId,expected.get(id));
  assert.equal(envelope.publisherSourceSha,publisherSha);
  assert.equal(envelope.sourceSha,null);
  assert.equal(envelope.sourceStatus,'UNPINNED');
  assert.equal(envelope.sourceCurrentness,'UNKNOWN');
  assert.equal(envelope.verifierStatus,'NOT_RUN');
  assert.equal(envelope.costBasis,'UNKNOWN');
  assert.equal(row.status,'STALE');
  assert.equal(row.reason,'NATIVE_SOURCE_STALE');
  assert.ok(Array.isArray(envelope.metricSummaries)&&envelope.metricSummaries.length>0);
  for(const metric of envelope.metricSummaries){
    if(metric.epistemic==='OBSERVED')assert.notEqual(metric.value,null);
    if(metric.value===null)assert.equal(metric.epistemic,'UNKNOWN');
  }
  actors.push({commentId:id,operatorId:envelope.actorOperatorId,reportId:row.source.reportId,
    captureCoverage:envelope.captureCoverage,metricCount:envelope.metricSummaries.length,
    sourceStatus:envelope.sourceStatus,verifier:envelope.verifierStatus,cost:envelope.costBasis,
    dataDigest:row.source.dataDigest});
}
assert.deepEqual(actors.map(row=>row.operatorId).sort(),['claude','codex','zcode']);
assert.equal(actors.find(row=>row.operatorId==='zcode').captureCoverage,'PARTIAL');
assert.equal(actors.find(row=>row.operatorId==='claude').captureCoverage,'COMPLETE');
assert.equal(actors.find(row=>row.operatorId==='codex').captureCoverage,'COMPLETE');
for(const result of [live,unavailable]){
  assert.equal(result.authorized,false);assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
}
console.log(JSON.stringify({source:'actual-public-github-reports',issue:'task-meta-e2e#9',
  publisherRepository:'kaicreator-mm/task-meta',publisherSha,observationCount:live.observationCount,
  reportCount:live.reportCount,actors,
  sourceUnavailable:{commentId:6382627467,sourceSha:unavailableSha,status:'UNKNOWN',reason:unavailable.reason},
  limits:['UNPINNED_AND_UNKNOWN_CURRENTNESS','VERIFIER_NOT_RUN','COST_UNKNOWN',
    'SOURCE_UNAVAILABLE_FOR_FOURTH_REPORT','ISOLATED_NON_PRODUCTION_ADOPTION'],
  taskPass:false,releasePass:false}));
console.log('TALLY_REAL_GITHUB_ONLY_OBSERVATION=PASS');
