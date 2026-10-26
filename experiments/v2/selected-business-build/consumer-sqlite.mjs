/**
 * T04c: actual public DH v0.10.2 Kernel + Host SQLite CAS with restart.
 * No Task authority, no private imports, no synthetic sqlite mock.
 */
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {mkdtemp, rm} from 'node:fs/promises';
import {join} from 'node:path';
import {tmpdir} from 'node:os';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore,probeTrueCasStore} from '@kaicreator/domain-harness-host-node';

const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
const deployment={deployedAppId:'deploy:tally-v2-t04c',tenantId:'tenant:tally-v2',
  appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>identity?.tenantId===target.tenantId &&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const options=(store)=>({store,tenantId:deployment.tenantId,
  deploymentIdentity:deployment,queryAuthority:authority,now:()=>Date.UTC(2026,9,27)});
const selector={workflowId:'tally-readiness',instanceKey:'task-108',
  businessPackageId:'business-tally-task-readiness@v2.0.0'};
const message={intentType:'inspectTaskReadiness',instanceKey:'task-108',
  messageId:'message-t04c-sqlite-01',
  input:{issueRef:'https://github.com/kaicreator-mm/tally/issues/108',
    issueNumber:108,sourceSha:'a'.repeat(40),
    claimRef:'github:unverified-input:108',dependenciesComplete:true}};

// The only adapter is the condition-spelling bridge between two public
// contract types. The Host-node SQLite driver owns CAS, transactions and WAL.
const condition=(c)=>c?.kind==='create'?{mustBeAbsent:true}:
  c?.kind==='cas'&&Number.isSafeInteger(c.revision)?
    {mustMatchRevision:c.revision}:
    (()=>{throw Object.assign(new Error('CONDITION_INVALID'),{code:'HOST_PUT_CONDITION_REQUIRED'});})();
function kernelStore(host){
  return Object.freeze({
    get:(key)=>host.get(key),
    put:(key,value,c)=>host.put(key,value,condition(c)),
    remove:(key,c)=>host.remove(key,condition(c)),
  });
}
const dir=await mkdtemp(join(tmpdir(),'tally-v2-t04c-'));
const db=join(dir,'kernel.sqlite');
let sqlite=null,facade=null;
try {
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  await probeTrueCasStore(sqlite);
  facade=await load(sealed,options(kernelStore(sqlite)));
  const first=await facade.send(message);
  assert.equal(first.status,'admitted',JSON.stringify(first));
  assert.equal(first.admitted?.effects?.length,1);
  assert.equal(first.admitted.effects[0].disposition,'executed');
  assert.equal(first.admitted.effects[0].output?.taskPass,false);
  assert.equal(first.admitted.effects[0].output?.sourceObservation?.state,'OBSERVATION_NOT_NATIVE_VERIFIED');
  assert.equal(first.admitted.effects[0].output?.sourceObservation?.authorized,false);
  const before=await facade.query(selector);
  assert.ok(before.snapshot);
  facade.stop(); facade=null;
  await sqlite.close();sqlite=null;

  // Physically reopen the SAME file, not a reused memory Map.
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(kernelStore(sqlite)));
  const after=await facade.query(selector);
  assert.deepEqual(after.snapshot,before.snapshot,'Kernel snapshot must survive actual Host reopen');
  const replay=await facade.send({...message,input:{...message.input}});
  assert.equal(replay.status,'admitted');
  assert.equal(replay.idempotentReplay,true,'same-message restart replay must not rerun a C effect');
  let rejected=false;
  try {
    const changed=await facade.send({...message,input:{...message.input,claimRef:'forged-changed'}});
    rejected=changed.status!=='admitted';
  } catch(error) {rejected=typeof error?.code==='string';}
  assert.equal(rejected,true,'changed payload with same messageId must reject');
  assert.deepEqual((await facade.query(selector)).snapshot,before.snapshot,
    'rejected changed input must not replace durable K snapshot');
  let unselected=false;
  try{
    const wrong=await facade.send({...message,messageId:'message-t04c-unknown',
      intentType:'approveTaskUnsealed'});
    unselected=wrong.status!=='admitted';
  }catch(error){unselected=typeof error?.code==='string';}
  assert.equal(unselected,true,'unselected Task approval must reject');
  assert.deepEqual((await facade.query(selector)).snapshot,before.snapshot);
  console.log(JSON.stringify({source:'public-host-node-sqlite',dbReopened:true,
    publicKernelLoadedTwice:true,realSelectedC:first.admitted.effects[0].output.sourceObservation.state,
    kernelReplayed:replay.idempotentReplay,changedMessageDenied:rejected,
    unsealedApprovalDenied:unselected,authorizedTask:false,releasePass:false}));
  console.log('T04C_REAL_SQLITE_KERNEL_RESTART=PASS');
}finally{
  if(facade)facade.stop();
  if(sqlite)await sqlite.close();
  await rm(dir,{recursive:true,force:true});
}
