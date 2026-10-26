/**
 * Tally v2 T02b real installed PUBLIC Microkernel consumer.
 * Receives only BUILD-produced sealed App plain JSON, not the BUILD package.
 * InMemory CAS is an isolated test-only V10DocStore; production uses the
 * previously qualified PUBLIC Host SQLite CAS adapter (#100/#101).
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { load } from '@kaicreator/domain-harness-microkernel';

const appBytes=readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8');
const sealed=JSON.parse(appBytes);
assert.ok(sealed.subjectDigest);
assert.equal(sealed.modules.filter(m=>m.role==='business').length,1);
assert.equal(sealed.modules.filter(m=>m.role==='kernel-primary').length,1);

class InMemoryCas {
  #docs=new Map();
  storeErrors=0;
  async get(key){
    const d=this.#docs.get(key);
    return d===undefined?null:{value:d.value,revision:d.revision};
  }
  async put(key,value,condition){
    const d=this.#docs.get(key);
    if(condition.kind==='create'&&d===undefined){
      this.#docs.set(key,{value,revision:0});return {revision:0};
    }
    if(condition.kind==='cas'&&d!==undefined&&d.revision===condition.revision){
      this.#docs.set(key,{value,revision:d.revision+1});return {revision:d.revision+1};
    }
    this.storeErrors++;
    throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
  }
  async remove(key,condition){
    const d=this.#docs.get(key);
    if((condition.kind==='cas'&&d!==undefined&&d.revision===condition.revision)||
       (condition.kind==='create'&&d===undefined)){
      this.#docs.delete(key);return;
    }
    this.storeErrors++;
    throw Object.assign(new Error('CAS removal refused'),{code:'HOST_CAS_CONFLICT'});
  }
  keys(){return [...this.#docs.keys()];}
}
const store=new InMemoryCas();
const deployment={deployedAppId:'deploy:tally-v2-t02b',tenantId:'tenant:tally-v2',
  appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:async(identity,target)=>identity?.tenantId===target.tenantId};
const facade=await load(sealed,{
  store,tenantId:deployment.tenantId,
  deploymentIdentity:deployment,queryAuthority:authority,
  now:()=>Date.UTC(2026,9,26)
});
assert.deepEqual(Object.keys(facade).sort(),['query','send','stop','subscribe']);
const msg={intentType:'inspectTaskReadiness',instanceKey:'task-108',messageId:'message-t02b-01',
 input:{issueRef:'https://github.com/kaicreator-mm/tally/issues/108',issueNumber:108,
   sourceSha:'a'.repeat(40),claimRef:'github:claim:108',dependenciesComplete:true}};
try {
  const receipt=await facade.send(msg);
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1000));
  assert.equal(receipt.admitted?.effects?.length,1,JSON.stringify(receipt).slice(0,1000));
  assert.equal(receipt.admitted.effects[0].disposition,'executed');
  // A source-observation Business intent can be admitted while Task/Review PASS
  // remains false. The Claim ref is input metadata, not a K approval.
  assert.equal(receipt.admitted.effects[0].output?.state,'OBSERVED_INPUT_ONLY');
  assert.equal(receipt.admitted.effects[0].output?.admitted,false);
  assert.equal(receipt.admitted.effects[0].output?.taskPass,false);
  const answer=await facade.query({workflowId:'tally-readiness',instanceKey:'task-108',
    businessPackageId:'business-tally-task-readiness@v2.0.0'});
  assert.equal(answer.target.tenantId,deployment.tenantId);
  assert.ok(answer.snapshot);
  const snapshotBefore=JSON.stringify(answer.snapshot);
  const replay=await facade.send({...msg,input:{...msg.input}});
  assert.equal(replay.status,'admitted');
  assert.equal(replay.idempotentReplay,true);
  // Legitimate K CAS retries may have touched the test store during admission.
  // Record a baseline immediately before negatives, rather than assuming zero
  // historical conflicts for the whole positive journey.
  const storeErrorsBeforeNegative=store.storeErrors;
  let conflict=false;
  try {const changed=await facade.send({...msg,input:{...msg.input,claimRef:'changed'}});
    conflict=changed.status!=='admitted';
  } catch(e){conflict=typeof e.code==='string';}
  assert.ok(conflict,'changed-input same-message replay must refuse');
  assert.equal(store.storeErrors,storeErrorsBeforeNegative,'negative must be Kernel denial, not store failure');
  let wrongIntent=false;
  try {const invalid=await facade.send({...msg,messageId:'message-t02b-02',
      intentType:'unknownTaskApproval'});
    wrongIntent=invalid.status!=='admitted';}catch(e){wrongIntent=typeof e.code==='string';}
  assert.ok(wrongIntent,'unselected intent cannot be admitted');
  assert.equal(store.storeErrors,storeErrorsBeforeNegative,'negative must not depend on throwing store');
  const after=await facade.query({workflowId:'tally-readiness',instanceKey:'task-108',businessPackageId:'business-tally-task-readiness@v2.0.0'});
  assert.equal(JSON.stringify(after.snapshot),snapshotBefore,'denied send must not change current snapshot');
  console.log(JSON.stringify({realKLoad:true,realKSend:receipt.status,
    realEffectDisposition:receipt.admitted.effects[0].disposition,
    key:'inspect:108:'+msg.input.sourceSha,realScopedQuery:true,
    immutableReplay:replay.idempotentReplay,changedInputRefused:conflict,
    unselectedIntentRefused:wrongIntent,authorizedTask:false,
    nativeGithubClaim:'NOT_VERIFIED',releasePass:false}));
  console.log('T02B_REAL_K_CONSUMER=PASS');
} finally {facade.stop();}
