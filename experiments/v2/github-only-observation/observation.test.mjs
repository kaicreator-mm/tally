import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {readIssueReportsFromClient} from '../../../packages/tally-github-native-c/src/read-only.mjs';

const require=createRequire(new URL('../../../packages/tally-github-native-c/package.json',import.meta.url));
const {stringify}=require('yaml');

const SHA='a'.repeat(40),owner='example-org',repo='example-repo',issueNumber=42;
const scope={owner,repo,issueNumber,expectedSourceSha:SHA,
  publisherBindings:{'task-meta:example-org/example-repo':['trusted-bot']}};
const fixture=name=>JSON.parse(readFileSync(new URL(`../../../packages/tally-ads-v5-validator/test/fixtures/${name}.json`,import.meta.url),'utf8'));
function approved(){const obj=fixture('summary');obj.data.limits=[];obj.data.capture_coverage='COMPLETE';return obj;}
function collectorApproved(){
  const obj=approved();
  Object.assign(obj.data.subject,{attempt_binding:'COLLECTOR_ONLY',source_sha:null,publisher_source_sha:SHA,
    source_status:'UNPINNED',source_currentness:'UNKNOWN',source_reason:'no immutable execution bytes',
    collector_observation_refs:['collector:session:1']});
  obj.data.subject.currentness.state='UNKNOWN';obj.data.subject.currentness.limits=['HEAD not witnessed'];
  obj.data.limits=['execution bytes not witnessed'];obj.data.capture_coverage='PARTIAL';
  obj.data.verifier.status='NOT_RUN';obj.data.cost.basis='UNKNOWN';
  obj.data.manifest_ref='task-meta-private://sha256/'+'0'.repeat(64);
  return obj;
}
let nextId=100;
const observation=(envelope,{id,edited=false}={})=>({id:id??++nextId,
  body:`<!-- ads:github -->\n\`\`\`yaml\n${stringify(envelope).trimEnd()}\n\`\`\``,
  user:{login:'trusted-bot'},created_at:'2026-10-24T00:00:00Z',
  updated_at:edited?'2026-10-24T01:00:00Z':'2026-10-24T00:00:00Z',
  html_url:`https://github.com/${owner}/${repo}/issues/${issueNumber}#issuecomment-${id??nextId}`});
function client(pages=[],opts={}){return {rest:{issues:{get:async()=>({data:{number:issueNumber}}),listComments:async()=>{throw Error('iterator only');}},
  repos:{getCommit:async()=>{if(opts.errorAt==='commit')throw {status:opts.error??404};return {data:{sha:SHA}};}}},
  paginate:{iterator:async function*(){for(let index=0;index<pages.length;index++){
    if(opts.errorAt==='page'&&index===1)throw {status:opts.error??403};yield {data:pages[index]};}}}};}
const read=(pages=[],opts={},query={})=>readIssueReportsFromClient({octokit:client(pages,opts),...scope,...query});

test('duplicate identity and edited source fail closed',async()=>{
  const first=observation(approved()),second=observation(approved());
  const duplicate=await read([[first,second]]);
  assert.equal(duplicate.status,'CONFLICT');assert.equal(duplicate.taskPass,false);
  const edited=await read([[observation(approved(),{edited:true})]]);
  assert.equal(edited.records[0].reason,'NATIVE_SOURCE_STALE');
});

test('schema mismatch missing publication and 429 are not report success',async()=>{
  const wrong=approved();wrong.schema='ads/github/v4';
  assert.equal((await read([[observation(wrong)]])).reason,'NO_ADS_V5_REPORTS');
  assert.equal((await read([])).reason,'NO_ADS_V5_REPORTS');
  const limited=await read([[observation(approved())],[observation(approved())]],{errorAt:'page',error:429});
  assert.equal(limited.reason,'GITHUB_RATE_LIMITED');
  assert.equal(limited.status,'UNKNOWN');
});

test('incomplete collector evidence keeps public metrics and never dereferences private manifest',async()=>{
  const result=await readIssueReportsFromClient({octokit:client([[observation(collectorApproved())]]),
    ...scope,publisherRepository:'example-org/publisher-repo'});
  assert.equal(result.reportCount,1);assert.equal(result.records[0].status,'STALE');
  const row=result.records[0];
  assert.equal(row.advisory.captureCoverage,'PARTIAL');assert.equal(row.advisory.verifierStatus,'NOT_RUN');
  assert.equal(row.advisory.costBasis,'UNKNOWN');assert.equal(row.advisory.sourceStatus,'UNPINNED');
  assert.equal(row.advisory.actorOperatorId,'codex:example');
  assert.ok(row.advisory.metricSummaries.length>0);
});

test('Host allowlist selects only the exact source identity and malformed allowlist refuses',async()=>{
  const pinned=observation(approved()),collector=observation(collectorApproved());
  const selected=await read([[pinned,collector]],{},{acceptedCommentIds:[pinned.id]});
  assert.equal(selected.reportCount,1);assert.equal(selected.records[0].source.commentId,pinned.id);
  const invalid=await readIssueReportsFromClient({octokit:client([]),...scope,acceptedCommentIds:['bad']});
  assert.equal(invalid.reason,'INVALID_NATIVE_QUERY');
});
