/** T04b public selected native C + real GitHub REST + durable public K/Host SQLite. */
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
const sourceSha=process.env.TALLY_NATIVE_EXPECTED_SHA;
const token=process.env.TALLY_GITHUB_READ_TOKEN;
assert.match(sourceSha??'',/^[0-9a-f]{40}$/);
assert.ok(typeof token==='string'&&token.length>=8,'host read-only token required');
const deployment={deployedAppId:'deploy:tally-v2-t04b',tenantId:'tenant:tally-v2',appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>identity?.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c?.kind==='create'?{mustBeAbsent:true}:
  c?.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw Object.assign(new Error('INVALID_K_CONDITION'),{code:'HOST_PUT_CONDITION_REQUIRED'});})();
const kDocs=sqlite=>({get:key=>sqlite.get(key),
  put:(key,v,c)=>sqlite.put(key,v,condition(c)),
  remove:(key,c)=>sqlite.remove(key,condition(c))});
const options=sqlite=>({store:kDocs(sqlite),deploymentIdentity:deployment,
  // Resolve exactly the installed, BUILD-declared native C provider.
  deploymentRoot:process.cwd(),
  tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,27)});
const selector={workflowId:'tally-readiness',instanceKey:'task-111',
  businessPackageId:'business-tally-task-readiness@v2.0.0'};
const message={intentType:'inspectTaskReadiness',instanceKey:'task-111',
  messageId:'message-t04b-native-01',
  input:{issueRef:'https://github.com/kaicreator-mm/tally/issues/111',
    issueNumber:111,sourceSha,claimRef:'unverified:source:111',
    dependenciesComplete:true}};
assert.ok(!JSON.stringify(message).includes(token),'token must NOT enter journaled Business input');
const tmp=await mkdtemp(join(tmpdir(),'tally-t04b-native-'));
const db=join(tmp,'kernel.sqlite');
let sqlite=null,facade=null;
try {
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  const receipt=await facade.send(message);
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1300));
  assert.equal(receipt.admitted?.effects?.length,1);
  assert.equal(receipt.admitted.effects[0].disposition,'executed');
  const effect=receipt.admitted.effects[0].output;
  assert.equal(effect.state,'OBSERVED_INPUT_ONLY');
  assert.equal(effect.admitted,false);
  assert.equal(effect.taskPass,false);
  assert.equal(effect.sourceObservation?.state,'NATIVE_NOT_CURRENT');
  assert.equal(effect.sourceObservation?.status,'UNKNOWN');
  assert.equal(effect.sourceObservation?.reason,'NO_ADS_V5_REPORTS');
  assert.equal(effect.sourceObservation?.sourceSha,sourceSha);
  assert.equal(effect.sourceObservation?.authorized,false);
  assert.equal(effect.sourceObservation?.taskPass,false);
  assert.equal(effect.sourceObservation?.releasePass,false);
  assert.ok(effect.sourceObservation?.observationCount>=0);
  const before=(await facade.query(selector)).snapshot;
  assert.ok(before);
  const recordsBefore=await sqlite.list('');
  assert.ok(recordsBefore.length>0);
  assert.ok(!JSON.stringify(before).includes(token),'token may not enter SQLite Kernel snapshot');
  for(const key of recordsBefore){
    const row=await sqlite.get(key);
    assert.ok(!JSON.stringify(row).includes(token),'Host secret may not appear in ANY durable K document: '+key);
  }
  facade.stop();facade=null;
  await sqlite.close();sqlite=null;

  // Reopen the exact SQLite database from a fresh public K activation.
  // Remove the token BEFORE replay; a correct immutable replay needs no API.
  delete process.env.TALLY_GITHUB_READ_TOKEN;
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  assert.deepEqual((await facade.query(selector)).snapshot,before);
  const replay=await facade.send({...message,input:{...message.input}});
  assert.equal(replay.status,'admitted');
  assert.equal(replay.idempotentReplay,true);
  assert.deepEqual(await sqlite.list(''),recordsBefore,'replay must not create new journal/effect docs');
  let blocked=false;
  try{
    const x=await facade.send({...message,input:{...message.input,claimRef:'untrusted-change'}});
    blocked=x.status!=='admitted';
  }catch(e){blocked=typeof e.code==='string';}
  assert.equal(blocked,true);
  assert.deepEqual((await facade.query(selector)).snapshot,before);
  console.log(JSON.stringify({provider:'github-native-readonly',via:'real-sealed-k-b-c',
    persistedHost:'sqlite',reopened:true,sourceSha,observedComments:effect.sourceObservation.observationCount,
    reportReason:effect.sourceObservation.reason,
    replayWithoutToken:replay.idempotentReplay,docCount:recordsBefore.length,
    changedInputDenied:blocked,taskPass:false,releasePass:false}));
  console.log('T04B_REAL_GITHUB_C_KERNEL_SQLITE=PASS');
} finally {
  if(facade)facade.stop();
  if(sqlite)await sqlite.close();
  await rm(tmp,{recursive:true,force:true});
}
