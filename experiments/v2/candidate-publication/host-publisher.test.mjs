import test from 'node:test';
import assert from 'node:assert/strict';
import {createCandidatePublisher} from './host-publisher.mjs';
const sourceSha='d'.repeat(40),candidateCommit='a'.repeat(40),candidateTree='b'.repeat(40);
const issueRef='https://github.com/kaicreator-mm/tally/issues/197';
const attemptId='attempt:alm:j05:a:1';
const candidate={attemptId,issueRef,candidateCommit,candidateTree,candidateRelativePath:'A.md'};
const joined={phase:'JOINED',sourceSha,joinedAt:'2026-10-31T06:07:22.223Z',candidates:[candidate]};
function fixture({rows=[{value:joined,revision:1}],publishResult,parent=sourceSha,tree=candidateTree,pushError=null}={}){
  const calls=[];
  const openStore=async()=>({list:async()=>rows.map((_,i)=>'join'+i),get:async key=>rows[Number(key.slice(4))],close:async()=>{}});
  const exec=async(command,args)=>{calls.push(args);
    if(args.includes('push')&&pushError)throw pushError;
    return {stdout:(args.at(-1)===candidateCommit+'^'?parent:tree)+'\n'};};
  const publish=async input=>{calls.push(['publish',input]);return publishResult??{state:'PUBLISHED_VERIFIED',pullNumber:203,pullRef:'pr',externalRef:'pr'};};
  const publisher=createCandidatePublisher({openStore,exec,publish});
  return {calls,publish:()=>publisher.publishReviewedCandidate({issueRef,sourceSha,attemptId,
    hostStorePath:'host.sqlite',repositoryPath:'repo',githubToken:'secret-token'})};
}
test('latest reviewed exact candidate is verified before one Host/C publication',async()=>{
  const f=fixture();const result=await f.publish();
  assert.equal(result.state,'PUBLISHED_VERIFIED');assert.equal(result.pullNumber,203);
  assert.equal(f.calls.filter(x=>x.includes('push')).length,1);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
});
test('the same attempt remains valid in the later repair re-Join',async()=>{
  const f=fixture({rows:[{value:joined,revision:1},{revision:2,value:{...joined,
    joinedAt:'2026-10-31T06:08:22.223Z'}}]});
  const result=await f.publish();assert.equal(result.state,'PUBLISHED_VERIFIED');
});
test('unreviewed and repaired-superseded candidates never reach git or GitHub',async()=>{
  for(const f of [fixture({rows:[]}),fixture({rows:[{value:joined,revision:1},{revision:2,value:{...joined,
    joinedAt:'2026-10-31T06:08:22.223Z',candidates:[{...candidate,attemptId:'attempt:repair'}]}}]})]){
    const result=await f.publish();assert.equal(result.state,'NOT_ADMITTED');assert.equal(f.calls.length,0);
  }
});
test('local commit parent/tree must match the reviewed record',async()=>{
  for(const f of [fixture({parent:'0'.repeat(40)}),fixture({tree:'0'.repeat(40)})]){
    const result=await f.publish();assert.equal(result.reason,'CANDIDATE_READBACK_MISMATCH');
    assert.equal(f.calls.filter(x=>x.includes('push')).length,0);
  }
});
test('lost push ACK is UNKNOWN and cannot become a publication receipt',async()=>{
  const f=fixture({pushError:new Error('lost ack')});const result=await f.publish();
  assert.equal(result.state,'PUBLICATION_UNKNOWN');assert.equal(result.reason,'GITHUB_REF_PUBLICATION_UNKNOWN');
  assert.equal(f.calls.filter(x=>Array.isArray(x)&&x[0]==='publish').length,0);
});
test('cross-task input and missing C credential are refused before reads or writes',async()=>{
  const publisher=createCandidatePublisher({openStore:async()=>{throw new Error('must not read');}});
  assert.equal((await publisher.publishReviewedCandidate({issueRef:'https://github.com/kaicreator-mm/tally/issues/196'})).reason,'PUBLICATION_TASK_NOT_ALLOWED');
  assert.equal((await publisher.publishReviewedCandidate({issueRef,sourceSha,attemptId,hostStorePath:'x',repositoryPath:'y'})).reason,'GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
});
