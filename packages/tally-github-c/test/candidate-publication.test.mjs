import test from 'node:test';
import assert from 'node:assert/strict';
import {publishCandidatePullRequestFromClient} from '../src/candidate-publication.mjs';
const candidateCommit='a'.repeat(40),branch='tally/reviewed/attempt-alm-j05-a-1';
const input={owner:'kaicreator-mm',repo:'tally',branch,baseRef:'version/v2',candidateCommit,
  title:'Reviewed candidate',body:'Refs: #200'};
const pr={number:203,state:'open',html_url:'https://github.com/kaicreator-mm/tally/pull/203',
  head:{sha:candidateCommit,ref:branch,repo:{full_name:'kaicreator-mm/tally'}},
  base:{ref:'version/v2',repo:{full_name:'kaicreator-mm/tally'}}};
function client({existing=[],created=pr,readback=pr,refSha=candidateCommit,error=null}={}){
  const calls=[];
  return {calls,git:{getRef:async()=>{if(error)throw error;return {data:{object:{sha:refSha}}};}},
    pulls:{list:async()=>{if(error)throw error;return {data:existing};},
      create:async arg=>{calls.push(['create',arg]);if(error)throw error;return {data:created};},
      get:async()=>{if(error)throw error;return {data:readback};}}};
}
test('creates one PR only after exact remote candidate ref readback',async()=>{
  const octokit=client({existing:[]});const result=await publishCandidatePullRequestFromClient({octokit,...input});
  assert.equal(result.state,'PUBLISHED_VERIFIED');assert.equal(result.pullNumber,203);
  assert.equal(octokit.calls.length,1);assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
});
test('lost ACK retry reuses the existing exact PR without a duplicate create',async()=>{
  const octokit=client({existing:[pr]});const result=await publishCandidatePullRequestFromClient({octokit,...input});
  assert.equal(result.state,'PUBLISHED_VERIFIED');assert.equal(octokit.calls.length,0);
});
test('wrong ref, base, head or closed PR fails closed',async()=>{
  for(const octokit of [client({refSha:'b'.repeat(40)}),client({existing:[{...pr,state:'merged'}]}),
    client({existing:[{...pr,base:{...pr.base,ref:'main'}}]}),client({readback:{...pr,head:{...pr.head,sha:'b'.repeat(40)}}})]){
    const result=await publishCandidatePullRequestFromClient({octokit,...input});
    assert.ok(['NOT_ADMITTED','PUBLICATION_UNKNOWN'].includes(result.state));
    assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  }
});
test('transport errors have bounded reasons and no token',async()=>{
  for(const [status,reason] of [[403,'GITHUB_ACCESS_DENIED'],[429,'GITHUB_RATE_LIMITED'],[500,'GITHUB_PUBLICATION_UNKNOWN']]){
    const result=await publishCandidatePullRequestFromClient({octokit:client({error:{status}}),...input});
    assert.equal(result.state,'PUBLICATION_UNKNOWN');assert.equal(result.reason,reason);
  }
});
test('cross-repo branch and unapproved base are refused before transport',async()=>{
  for(const change of [{owner:'other'},{repo:'other'},{branch:'main'},{baseRef:'refs/heads/main'},{
    candidateCommit:'bad'},{title:''}]){
    const octokit=client();const result=await publishCandidatePullRequestFromClient({octokit,...input,...change});
    assert.equal(result.state,'NOT_ADMITTED');assert.equal(result.reason,'PUBLICATION_INPUT_INVALID');
    assert.equal(octokit.calls.length,0);
  }
});
