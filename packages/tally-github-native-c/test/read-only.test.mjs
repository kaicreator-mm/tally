import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import canonicalize from 'canonicalize';
import { stringify } from 'yaml';
import { createHash } from 'node:crypto';
import { readIssueReportsFromClient, readGitHubIssueReports } from '../src/read-only.mjs';
const SHA='a'.repeat(40);
const [owner,repo,issueNumber]=['example-org','example-repo',42];
const publisherBindings={'task-meta:example-org/example-repo':['trusted-bot'],'tally:example-org/example-repo':['trusted-bot']};
const scope={owner,repo,issueNumber,expectedSourceSha:SHA,publisherBindings};
const fixture=(name)=>JSON.parse(readFileSync(new URL(
  '../../tally-ads-v5-validator/test/fixtures/'+name+'.json',import.meta.url),'utf8'));
let commentId=120;
function observation(envelope,opts={}) {
  const id=opts.id??++commentId;
  return {
    id,body:opts.body??`<!-- ads:github -->\n\`\`\`yaml\n${stringify(envelope).trimEnd()}\n\`\`\`\n\nHuman explanation (not machine authority).`,
    user:{login:opts.login??'trusted-bot'},author_association:'NONE',
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
  const expected=createHash('sha256').update(canonicalize(approved().data)).digest('hex');
  assert.equal(result.records[0].source.dataDigest,expected);
  assert.match(result.records[0].source.dataDigest,/^[0-9a-f]{64}$/);
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
test('bad/unmarked YAML wire and duplicate keys, anchors, tags fail closed',async()=>{
  const valid=approved();
  const samples=[
    JSON.stringify(valid,null,2),
    "<!-- ads:github -->\n```yaml\nschema: ads/github/v5\nschema: ads/github/v5\nkind: report\ndata: {}\n```",
    "<!-- ads:github -->\n```yaml\nschema: &danger ads/github/v5\nkind: report\ndata: {}\n```",
    "<!-- ads:github -->\n```yaml\nschema: !!js/function hacked\nkind: report\ndata: {}\n```",
  ];
  for(const [i,bad] of samples.entries()){
    const result=await read([[observation(valid,{body:bad})]]);
    assert.equal(result.status,'UNKNOWN');
    // ADS ignores unmarked human discussion/JSON rather than treating it as
    // an invalid normative operation. Marked malformed YAML is invalid.
    assert.equal(result.reason,i===0?'NO_ADS_V5_REPORTS':'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
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

test('untrusted GitHub account cannot forge a publication or poison the report set',async()=>{
  const result=await read([[
    observation(approved(),{login:'stranger'}),
    observation(approved(),{login:'trusted-bot'})
  ]]);
  assert.equal(result.status,'CURRENT_OBSERVED');
  assert.equal(result.reportCount,1);
  const fakeCorrection=observation(approved('correction'),{login:'stranger'});
  const protectedSet=await read([[observation(approved()),fakeCorrection]]);
  assert.equal(protectedSet.status,'CURRENT_OBSERVED');
});

test('trusted login cannot impersonate a different publisher namespace',async()=>{
  const x=approved('feedback');
  const result=await read([[observation(x)]],{},{
    publisherBindings:{'task-meta:example-org/example-repo':['trusted-bot']}
  });
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
});
test('missing trusted namespace binding never grants transport provenance',async()=>{
  const missing=await readIssueReportsFromClient({octokit:client([[observation(approved())]]),
    owner,repo,issueNumber,expectedSourceSha:SHA});
  assert.equal(missing.reason,'PUBLISHER_BINDINGS_UNAVAILABLE');
});
test('canonical report publication has optional Markdown without losing the data digest',async()=>{
  const f=approved();
  const one=observation(f);
  const two=observation(f,{id:one.id,body:one.body+'\nAdditional text.'});
  assert.equal((await read([[one]])).records[0].source.dataDigest,
    (await read([[two]])).records[0].source.dataDigest);
});


test('R2 P1: CRLF correction after LF summary must invalidate CURRENT',async()=>{
  const base=observation(approved());
  const corrected=observation(approved('correction'));
  corrected.body=corrected.body.replace(/\n/g,'\r\n');
  const result=await read([[base,corrected]]);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.records[1].reason,'CORRECTION_NOT_RECONCILED');
});
test('R2 P1: displaced ADS marker in trusted comment must fail closed',async()=>{
  const displaced=observation(approved('correction'));
  displaced.body='\n'+displaced.body;
  const result=await read([[observation(approved()),displaced]]);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
});
test('R2: CRLF single summary and blank lines after marker are valid ADS wire',async()=>{
  const only=observation(approved());
  only.body=only.body.replace(/\n/g,'\r\n');
  assert.equal((await read([[only]])).status,'CURRENT_OBSERVED');
  const padded=observation(approved());
  padded.body=padded.body.replace('<!-- ads:github -->\n','<!-- ads:github -->\n\n');
  assert.equal((await read([[padded]])).status,'CURRENT_OBSERVED');
  const mixed=observation(approved('correction'));
  mixed.body=mixed.body.replace('<!-- ads:github -->\n','<!-- ads:github -->\r\n\r\n');
  assert.equal((await read([[observation(approved()),mixed]])).status,'UNKNOWN');
});
test('R2: misplaced duplicate marker and body with no YAML fence are invalid',async()=>{
  const d=observation(approved('correction'));
  const invalids=[
    '\n'+d.body, d.body+'\n<!-- ads:github -->',
    '<!-- ads:github -->\n\nNot a fence',
  ];
  for(const body of invalids){
    const r=await read([[observation(approved()),observation(approved('correction'),{body})]]);
    assert.equal(r.status,'UNKNOWN');
    assert.equal(r.reason,'INVALID_OR_UNVERIFIED_REPORT_COMMENT');
  }
});
