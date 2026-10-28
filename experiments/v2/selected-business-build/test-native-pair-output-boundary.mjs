/**
 * T06f selected C pair output boundary: fake provider is test-only.
 * Tests bounded projection before any K/SQLite journal is allowed to consume it.
 * The separate actual Host CI still checks genuine GitHub/SQLite/replay.
 */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync,mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {pathToFileURL} from 'node:url';

const source=readFileSync(new URL('./native-pair-shim.mjs',import.meta.url),'utf8');
const root=mkdtempSync(join(tmpdir(),'tally-native-pair-boundary-'));
const sha='a'.repeat(40);
const issue='https://github.com/kaicreator-mm/tally/issues/125';
const canary='CANDIDATE_SECRET_MUST_NEVER_ENTER_JOURNAL';
const denies={authorized:false,admitted:false,taskPass:false,releasePass:false};
const positive=()=>({
  state:'NATIVE_TASK_SOURCE_ADVISORY',status:'SOURCE_PAIR_OBSERVED',
  reason:'ACCEPTED_CAS_CLAIM_NOT_VERIFIED',sourceRef:issue,sourceSha:sha,
  observationCount:1,claimStatus:'NOT_VERIFIED',...denies,
  workItem:{issueRef:issue,sourceSha:sha,reason:'CLAIM_NOT_VERIFIED',
    issueState:'open',workflowState:'state:ready',workItemType:'type:task',
    dependencyState:'CLEAR',blockerCount:0,openBlockerCount:0,
    claimStatus:'NOT_VERIFIED',...denies},
  claimEvents:{reason:'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED',
    candidateCount:1,claimStatus:'NOT_VERIFIED',...denies,
    candidates:[{sourceSha:sha,
      sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',
      dispatchId:'observed-comment-123',claimAccepted:false}]}
});
const cases=[
  ['bounded-observation',()=>{const p=positive();
    p.rawComment=canary;p.token=canary;p.workItem.original=canary;
    p.claimEvents.candidates[0].rawBody=canary;
    p.claimEvents.candidates[0].actorCredential=canary;
    return p;},'ACCEPTED_CAS_CLAIM_NOT_VERIFIED',true],
  ['candidate-forged-accepted',()=>{const p=positive();
    p.claimEvents.candidates[0].claimAccepted=true;
    p.claimEvents.candidates[0].secret=canary;
    return p;},'UNVERIFIED_NATIVE_PAIR_PROVIDER_OUTPUT',false],
  ['candidate-unbounded-dispatch',()=>{const p=positive();
    p.claimEvents.candidates[0].dispatchId=canary.repeat(50);
    return p;},'UNVERIFIED_NATIVE_PAIR_PROVIDER_OUTPUT',false],
  ['candidate-stale-sha',()=>{const p=positive();
    p.claimEvents.candidates[0].sourceSha='b'.repeat(40);
    return p;},'UNVERIFIED_NATIVE_PAIR_PROVIDER_OUTPUT',false],
  ['provider-error',()=>{throw Error(canary);},'NATIVE_PAIR_PROVIDER_THREW',false]
];
try {
  for(const [name,build,expected,success] of cases){
    const payload=name==='provider-error'
      ? `export async function observeReadiness(){throw Error(${JSON.stringify(canary)});}`
      : `export async function observeReadiness(){return ${JSON.stringify(build())};}`;
    const file=join(root,name+'.mjs');
    writeFileSync(file,payload);
    const replaced=source.replace(
      "'@kaicreator/tally-native-reader-bundle'",
      JSON.stringify(pathToFileURL(file).href));
    assert.notEqual(replaced,source,'provider import must be substituted');
    const shim=await import('data:text/javascript;base64,'+
      Buffer.from(replaced).toString('base64'));
    const out=await shim.observeReadiness({sourceSha:sha});
    assert.equal(out.reason,expected,name);
    for(const [key,value] of Object.entries(denies))
      assert.equal(out[key],value,name+'/'+key);
    assert.equal(JSON.stringify(out).includes(canary),false,name+'/secret');
    if(success){
      assert.equal(out.claimEvents.candidates.length,1);
      assert.deepEqual(Object.keys(out.claimEvents.candidates[0]).sort(),
        ['sourceSha','sourceClass','dispatchId','claimAccepted'].sort());
      assert.equal(out.claimEvents.candidates[0].claimAccepted,false);
      assert.equal(out.claimStatus,'NOT_VERIFIED');
      assert.equal(out.observationCount,1);
      assert.equal(Object.hasOwn(out.workItem,'original'),false);
      assert.equal(out.pairDiagnosis,undefined);
    }else{
      assert.equal(out.status,'UNKNOWN');
      assert.equal(out.claimStatus,'NOT_VERIFIED');
      assert.equal(out.observationCount,0);
      assert.equal(out.claimEvents,undefined);
    }
  }
  console.log('T06F_SEALED_PAIR_UNTRUSTED_CANDIDATE_NEGATIVES=PASS');
}finally{rmSync(root,{recursive:true,force:true});}
