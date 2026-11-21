import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiFeedbackPublisherHost} from './ui-feedback-publisher-host.mjs';
const sha='a'.repeat(40),repo='kaicreator-mm/tally';
const proposalRef='https://github.com/'+repo+'/issues/274#issuecomment-2001';
const receiptRef='https://github.com/'+repo+'/issues/274#issuecomment-2002';
const q={repository:repo,versionId:'version/v2',sourceSha:sha,
  feedbackId:'experience-1',revision:1,issueNumber:274};
const proposal={...q,status:'VERIFIED',currentness:'CURRENT',state:'REVIEWED',
  proposalRef,digest:'b'.repeat(64)};
const publication={...q,status:'VERIFIED',proposalDigest:proposal.digest,
  publicationRef:receiptRef};
function mock(opts={}) {
  const calls={read:0,dispatch:0,readback:0,last:null};
  const selectedHost={
    readFeedbackProposal:async()=>{
      calls.read++;return calls.read===2&&opts.moved?
        {...proposal,digest:'c'.repeat(64)}:{...proposal};
    },
    dispatchFeedbackPublication:async(input)=>{
      calls.dispatch++;calls.last=input;
      if(opts.ackLost)throw Error('NO_ACK');
      return {status:'PUBLISHED_VERIFIED'};
    },
    readFeedbackPublication:async()=>{
      calls.readback++;return opts.foreign?{...publication,
        publicationRef:'https://github.com/foreign/repo/issues/274#issuecomment-2002'}:
        opts.noReadback?{status:'UNKNOWN'}:{...publication};
    }};
  const instance=createUiFeedbackPublisherHost({repository:repo,
    versionId:'version/v2',expectedSha:sha,selectedHost});
  return {instance,calls};
}
test('reviewed proposal and exact native readback yield publication only',async()=>{
  const {instance,calls}=mock(),r=await instance.publish({...q});
  assert.equal(r.status,'PUBLISHED_VERIFIED');
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
  assert.equal(r.measuredBenefit,'UNKNOWN');assert.equal(r.autoActivate,false);
  assert.equal(calls.read,2);assert.equal(calls.dispatch,1);
  assert.deepEqual(Object.keys(calls.last).sort(),
    ['feedbackId','issueNumber','proposalDigest','proposalRef','repository',
      'revision','sourceSha','versionId'].sort());
});
test('lost ACK recovered only from the selected Host readback',async()=>{
  const {instance,calls}=mock({ackLost:true});
  const r=await instance.publish({...q});
  assert.equal(r.status,'PUBLISHED_VERIFIED');
  assert.equal(r.reason,'ACK_RECONCILED_FROM_HOST_READBACK');
  assert.equal(calls.dispatch,1);
});
test('no readback after dispatch remains UNKNOWN, never PASS',async()=>{
  const {instance,calls}=mock({noReadback:true});
  const r=await instance.publish({...q});
  assert.equal(r.status,'UNKNOWN');
  assert.equal(calls.dispatch,1);assert.equal(r.taskPass,false);
});
test('stale or changed source refuses before any Host effect',async()=>{
  const {instance,calls}=mock({moved:true});
  const r=await instance.publish({...q});
  assert.equal(r.reason,'FEEDBACK_PROPOSAL_MOVED');assert.equal(calls.dispatch,0);
});
test('forged request fields and native GitHub writer flags are rejected',async()=>{
  const {instance,calls}=mock();
  const r=await instance.publish({...q,writerToken:'secret',authorized:true});
  assert.equal(r.reason,'FEEDBACK_UI_REQUEST_INVALID');
  assert.equal(calls.read,0);assert.equal(calls.dispatch,0);
});
test('wrong version, issue or SHA cannot change selected Host source',async()=>{
  const {instance,calls}=mock();
  for(const input of [{...q,sourceSha:'c'.repeat(40)},
    {...q,versionId:'version/v2.1.1'},{...q,issueNumber:0}]){
    const r=await instance.publish(input);assert.equal(r.status,'BLOCKED');
  }
  assert.equal(calls.dispatch,0);
});
test('foreign native publication ref never makes current outcome',async()=>{
  const {instance}=mock({foreign:true});
  const r=await instance.publish({...q});
  assert.equal(r.status,'UNKNOWN');
});
test('unmounted selected Host cannot dispatch UI writes',async()=>{
  const api=createUiFeedbackPublisherHost({repository:repo,
    versionId:'version/v2',expectedSha:sha,selectedHost:{}});
  const r=await api.publish({...q});
  assert.equal(r.reason,'FEEDBACK_SELECTED_HOST_PORT_UNAVAILABLE');
});
test('missing exact selected source cannot instantiate publisher UI bridge',()=>{
  assert.throws(()=>createUiFeedbackPublisherHost({repository:repo,
    versionId:'version/v2',expectedSha:'BAD',selectedHost:{}}),
    /FEEDBACK_SELECTED_SOURCE_REQUIRED/);
});
