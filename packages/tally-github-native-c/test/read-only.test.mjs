import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import canonicalize from 'canonicalize';
import { readIssueReportsFromClient, readGitHubIssueReports } from '../src/read-only.mjs';
const SHA='a'.repeat(40);
const [owner,repo,issueNumber]=['example-org','example-repo',42];
const scope={owner,repo,issueNumber,expectedSourceSha:SHA};
const fixture=(name)=>JSON.parse(readFileSync(new URL(
  '../../tally-ads-v5-validator/test/fixtures/'+name+'.json',import.meta.url),'utf8'));
let commentId=120;
function observation(envelope,opts={}) {
  const id=opts.id??++commentId;
  return {
    id,body:opts.body??canonicalize(envelope),
    created_at:'2026-10-21T00:00:00Z',
    updated_at:opts.edited?'2026-10-22T00:00:00Z':'2026-10-21T00:00:00Z',
    html_url:`https://github.com/${owner}/${repo}/issues/${issueNumber}#issuecomment-${id}`
  };
}
function approved(name='summary'){
  const obj=fixture(name);
  obj.data.limits=[];
  if(name==='summary') obj.data.capture_coverage='COMPLETE';
  return obj;
}
function client(pages=[],opts={}) {
  const api={
    rest:{
      issues:{
        get:async()=>opts.errorAt==='issue'?Promise.reject({status:opts.error??403}):{
          data:{number:opts.wrongIssue?7:42}
        },
        listComments:async()=>{throw Error('iterator only')}
      },
      repos:{getCommit:async()=>opts.errorAt==='commit'?Promise.reject({status:opts.error??404}):{
        data:{sha:opts.wrongCommit?'b'.repeat(40):SHA}
      }}
    },
    paginate:{iterator:async function* (){
      for(let i=0;i<pages.length;i++){
        if(opts.errorAt==='page'&&i===1) throw {status:opts.error??403};
        yield {data:pages[i]};
      }
    }}
  };
  return api;
}
const read=(pages=[],opts={},q={})=>readIssueReportsFromClient({
  octokit:client(pages,opts),...scope,...q
});
test('real pinned ADS summary through GitHub-style native page yields advisory only',async()=>{
  const result=await read([[observation(approved())]]);
  assert.equal(result.status,'CURRENT_OBSERVED');
  assert.equal(result.records[0].source.sourceSha,SHA);
  assert.match(result.records[0].source.dataDigest,/^jcs-sha256:[0-9a-f]{64}$/);
  assert.equal(result.authorized,false);
  assert.equal(result.taskPass,false);
  assert.equal(result.releasePass,false);
  assert.equal(result.observationCount,1);
});
test('pagination consumes all provider pages including canonical feedback',async()=>{
  const result=await read([[{body:'ordinary issue discussion'}],[observation(approved()),observation(approved('feedback'))]]);
  assert.equal(result.status,'CURRENT_OBSERVED');
  assert.equal(result.reportCount,2);
  assert.equal(result.observationCount,3);
});
test('a canonical ADS v5 REVOKE correction makes entire set UNKNOWN',async()=>{
  const result=await read([[observation(approved()),observation(approved('correction'))]]);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.records[1].reason,'CORRECTION_NOT_RECONCILED');
  assert.equal(result.taskPass,false);
});
test('no reports and no native page coverage do not become success',async()=>{
  assert.equal((await read([[]])).status,'UNKNOWN');
  assert.equal((await read([[{body:'human discussion'}]])).reason,'NO_ADS_V5_REPORTS');
  assert.equal((await read([])).status,'UNKNOWN');
});
test('reject edited/stale comment; never trust a supplied currentness flag',async()=>{
  const result=await read([[observation(approved(),{edited:true})]]);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.records[0].reason,'NATIVE_SOURCE_STALE');
});
test('reject cross-Issue/cross-repo source and wrong native sha',async()=>{
  for(const change of [
    x=>{x.data.subject.repository='evil/elsewhere';},
    x=>{x.data.subject.issue_ref='https://github.com/evil/elsewhere/issues/42';},
    x=>{x.data.subject.source_sha='b'.repeat(40);},
    x=>{x.data.publication.classification='BLOCKED';}
  ]){
    const report=approved();change(report);
    assert.equal((await read([[observation(report)]])).reason,'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
  }
});
test('bad wire and noncanonical JSON fail closed, duplicate keys refused',async()=>{
  const valid=approved();
  for(const bad of [
    JSON.stringify(valid,null,2),
    canonicalize(valid).replace('"schema":','"schema":"ads/github/v5","schema":'),
    '{ "schema": "ads/github/v5", ',
  ]){
    const result=await read([[observation(valid,{body:bad})]]);
    assert.equal(result.status,'UNKNOWN');
    assert.equal(result.reason,'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
  }
});
test('failure halfway through native GitHub pagination never partial-success',async()=>{
  const result=await read([[observation(approved())],[observation(approved('feedback'))]],{errorAt:'page'});
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'GITHUB_ACCESS_DENIED');
});
test('403/404 native errors and wrong source commit fail closed',async()=>{
  assert.equal((await read([],{errorAt:'issue',error:403})).reason,'GITHUB_ACCESS_DENIED');
  assert.equal((await read([],{errorAt:'commit',error:404})).reason,'GITHUB_RESOURCE_NOT_FOUND');
  assert.equal((await read([],{wrongCommit:true})).reason,'NATIVE_ISSUE_OR_COMMIT_MISMATCH');
});
test('bounded native comment scan never treats truncated page set as complete',async()=>{
  const r=await read([[observation(approved()),observation(approved('feedback'))]],{},{limit:1});
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.reason,'GITHUB_READ_LIMIT_REACHED');
});
test('absent token does not silently run anonymous GitHub reads',async()=>{
  const r=await readGitHubIssueReports({...scope});
  assert.equal(r.status,'UNKNOWN');
  assert.equal(r.reason,'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
});
test('caller cannot inject validation, permission or precomputed JCS digest through query',async()=>{
  const r=await read([[observation(approved())]],{},{
    validation:{passed:true},permission:'GRANTED',dataDigest:'forged'
  });
  assert.equal(r.status,'CURRENT_OBSERVED');
  assert.notEqual(r.records[0].source.dataDigest,'forged');
  assert.equal(r.authorized,false);
});
