/** T04b public selected native C + real GitHub REST + durable public K/Host SQLite. */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
// Independent trusted Host bootstrap check: BUILD-sealed C digest binds the
// exact Provider file. Verify before Microkernel.load imports/evaluates it.
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-source-observation@v2.0.0');
assert.ok(selectedC?.moduleSource,'sealed selected C required');
// Do not trust a digest literal from a locally rewritten sealed App.
// The selected C module bytes themselves must still match the BUILD seal;
// deployment integrity of the entire App also requires a separately trusted
// root digest (tracked as a generic DomainHarness v0.10.3 Host contract).
const moduleSha=bytes=>createHash('sha256').update(bytes).digest('hex');
assert.equal(moduleSha(selectedC.moduleSource),selectedC.moduleSha256,
  'SEALED_CAPABILITY_SOURCE_INTEGRITY_MISMATCH');
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'provider byte digest must be present in BUILD-sealed C');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-native-reader-bundle'));
const originalProvider=readFileSync(providerPath);
const sha=x=>createHash('sha256').update(x).digest('hex');
const assertProvider=(bytes)=>assert.equal(sha(bytes),digestMatch[1],
  'PROVIDER_INTEGRITY_MISMATCH: unreviewed npm implementation must refuse before K.load');
assertProvider(originalProvider);
// Real substitution falsifier on the actual installed Node package bytes.
// Restore original bytes after positive assertion and before any K.load.
try {
  writeFileSync(providerPath,Buffer.concat([originalProvider,Buffer.from('\\n// UNREVIEWED_PROVIDER_SUBSTITUTION')]));
  assert.throws(()=>assertProvider(readFileSync(providerPath)),/PROVIDER_INTEGRITY_MISMATCH/);
}finally{writeFileSync(providerPath,originalProvider);}
assertProvider(readFileSync(providerPath));
console.log('T04B_PROVIDER_INTEGRITY_BEFORE_K_LOAD=PASS');
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
const pairMode=process.env.T06F_NATIVE_PAIR==='1';
const issueNumber=pairMode?125:111;
const selector={workflowId:'tally-readiness',instanceKey:'task-'+issueNumber,
  businessPackageId:'business-tally-task-readiness@v2.0.0'};
const message={intentType:'inspectTaskReadiness',instanceKey:'task-'+issueNumber,
  messageId:pairMode?'message-t06f-native-pair-01':'message-t04b-native-01',
  input:{issueRef:`https://github.com/kaicreator-mm/tally/issues/${issueNumber}`,
    issueNumber,sourceSha,claimRef:'unverified:source:'+issueNumber,
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
  if(pairMode){
    const o=effect.sourceObservation;
    assert.equal(o?.state,'NATIVE_TASK_SOURCE_ADVISORY');
    assert.equal(o?.status,'SOURCE_PAIR_OBSERVED');
    assert.equal(o?.reason,'ACCEPTED_CAS_CLAIM_NOT_VERIFIED');
    assert.equal(o?.claimStatus,'NOT_VERIFIED');
    assert.equal(o?.workItem?.issueRef,message.input.issueRef);
    assert.equal(o?.workItem?.sourceSha,sourceSha);
    assert.equal(o?.claimEvents?.claimStatus,'NOT_VERIFIED');
    assert.equal(o?.claimEvents?.candidateCount,o?.observationCount);
    assert.equal(o?.claimEvents?.authorized,false);
    assert.ok(effect.pairDiagnosis);
    assert.equal(effect.pairDiagnosis.state,'NOT_ADMITTED');
    assert.equal(effect.pairDiagnosis.admitted,false);
    assert.equal(effect.pairDiagnosis.taskPass,false);
    assert.equal(effect.pairDiagnosis.releasePass,false);
    // Native Github Task read is an observation, not an accepted execution.
    assert.equal(effect.admitted,false);
  }else{
    assert.equal(effect.sourceObservation?.state,'NATIVE_NOT_CURRENT');
    assert.equal(effect.sourceObservation?.status,'UNKNOWN');
    assert.equal(effect.sourceObservation?.reason,'NO_ADS_V5_REPORTS');
  }
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
  console.log(pairMode?'T06F_REAL_GITHUB_PAIR_K_B_C_SQLITE_READONLY=PASS':
    'T04B_REAL_GITHUB_C_KERNEL_SQLITE=PASS');
} finally {
  if(facade)facade.stop();
  if(sqlite)await sqlite.close();
  await rm(tmp,{recursive:true,force:true});
}
