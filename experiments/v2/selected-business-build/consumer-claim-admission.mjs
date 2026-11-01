/** T06l public selected claim-admission C + real GitHub + durable public K/Host SQLite. */
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
// exact claim provider file. Verify before Microkernel.load imports/evaluates
// it, and falsify substitution on the actual installed npm package bytes.
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-claim-admission@v1.0.0');
assert.ok(selectedC?.moduleSource,'sealed selected claim C required');
const sha=x=>createHash('sha256').update(x).digest('hex');
assert.equal(sha(selectedC.moduleSource),selectedC.moduleSha256,
  'SEALED_CAPABILITY_SOURCE_INTEGRITY_MISMATCH');
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'claim provider byte digest must be present in BUILD-sealed C');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-native-claim-reader-bundle'));
const originalProvider=readFileSync(providerPath);
const assertProvider=(bytes)=>assert.equal(sha(bytes),digestMatch[1],
  'PROVIDER_INTEGRITY_MISMATCH: unreviewed npm implementation must refuse before K.load');
assertProvider(originalProvider);
try {
  writeFileSync(providerPath,Buffer.concat([originalProvider,Buffer.from('\n// UNREVIEWED_PROVIDER_SUBSTITUTION')]));
  assert.throws(()=>assertProvider(readFileSync(providerPath)),/PROVIDER_INTEGRITY_MISMATCH/);
}finally{writeFileSync(providerPath,originalProvider);}
assertProvider(readFileSync(providerPath));
console.log('T06L_PROVIDER_INTEGRITY_BEFORE_K_LOAD=PASS');

// Host-supplied deployment configuration. None of these values is authority:
// the sealed C re-verifies the registry row and the live GitHub readback.
const token=process.env.TALLY_GITHUB_READ_TOKEN;
const sourceSha=process.env.TALLY_CLAIM_SOURCE_SHA;
const dispatchId=process.env.TALLY_CLAIM_DISPATCH_ID;
const operatorId=process.env.TALLY_CLAIM_OPERATOR_ID;
const operatorKind=process.env.TALLY_CLAIM_OPERATOR_KIND;
const writerId=process.env.TALLY_CLAIM_WRITER_ID;
const writerLogin=process.env.TALLY_CLAIM_WRITER_LOGIN;
const commentId=Number(process.env.TALLY_CLAIM_COMMENT_ID);
const realRegistryPath=process.env.TALLY_CLAIM_REGISTRY_PATH||'';
assert.ok(typeof token==='string'&&token.length>=8,'host read-only token required');
assert.match(sourceSha??'',/^[0-9a-f]{40}$/);
assert.ok(dispatchId&&operatorId&&operatorKind&&writerId&&writerLogin,
  'TALLY_CLAIM_* host pins required');
assert.ok(Number.isSafeInteger(commentId)&&commentId>0,'TALLY_CLAIM_COMMENT_ID required');
const ISSUE_REF='https://github.com/kaicreator-mm/tally/issues/169';
const CLAIM_KEY='kaicreator-mm/tally#169:builder:__default__';
const REGISTRY_KEY='tally:v2:claim-admission:'+createHash('sha256').update(CLAIM_KEY).digest('hex');
const rowValue=()=>({phase:'PUBLISHED_VERIFIED',generation:1,protectedClaimKey:CLAIM_KEY,
  repository:'kaicreator-mm/tally',issueRef:ISSUE_REF,sourceSha,dispatchId,
  actorRole:'builder',operatorKind,operatorId,writerId,commentId,
  commentRef:`${ISSUE_REF}#issuecomment-${commentId}`});
const deployment={deployedAppId:'deploy:tally-v2-t06l',tenantId:'tenant:tally-v2',appId:sealed.appId};
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
  deploymentRoot:process.cwd(),
  tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,30)});
const messageFor=(instanceKey,messageId,input)=>({intentType:'admitClaimedTask',
  instanceKey,messageId,input});
const positiveInput={issueRef:ISSUE_REF,issueNumber:169,sourceSha};
assert.ok(!JSON.stringify(positiveInput).includes(token),'token must NOT enter journaled Business input');
const tmp=await mkdtemp(join(tmpdir(),'tally-t06l-claim-'));
const db=join(tmp,'kernel.sqlite');
const registryDb=realRegistryPath||join(tmp,'claim-registry.sqlite');
let sqlite=null,registry=null,facade=null;
try {
  // Fixture bootstrap writes the accepted-Claim registry row through the real
  // public DH CAS API. The sealed C under test independently re-reads this
  // row and re-verifies the live GitHub publication witness; every falsifier
  // below mutates one fact and must flip the admission to NOT_ADMITTED.
  // With TALLY_CLAIM_REGISTRY_PATH the Host points at a real previously
  // accepted registry instead; seeding and registry mutations are skipped.
  const realMode=realRegistryPath.length>0;
  registry=await openSqliteCasStore({path:registryDb,busyTimeoutMs:1000});
  if(!realMode){
    // Mimic the real T06d single-writer exactly: reserve PENDING (revision
    // 0), then confirm PUBLISHED_VERIFIED (revision 1). A one-shot write
    // would leave revision 0, which no genuine publication row ever has.
    const reservation={phase:'PENDING_GITHUB_PUBLISH',generation:1,
      protectedClaimKey:CLAIM_KEY,repository:'kaicreator-mm/tally',
      issueRef:ISSUE_REF,sourceSha,dispatchId,actorRole:'builder',
      operatorKind,operatorId,writerId};
    await registry.put(REGISTRY_KEY,reservation,{mustBeAbsent:true});
    await registry.put(REGISTRY_KEY,rowValue(),{mustMatchRevision:0});
  }else{
    const existing=await registry.get(REGISTRY_KEY);
    assert.ok(existing?.value?.phase==='PUBLISHED_VERIFIED'&&
      existing.value.generation===1,
      'real accepted registry must already carry the PUBLISHED_VERIFIED row');
  }
  process.env.TALLY_CLAIM_REGISTRY_PATH=registryDb;
  const restoreRow=async()=>{
    const current=await registry.get(REGISTRY_KEY);
    if(current===null){
      await registry.put(REGISTRY_KEY,rowValue(),{mustBeAbsent:true});
      return;
    }
    await registry.put(REGISTRY_KEY,rowValue(),{mustMatchRevision:current.revision});
  };
  const mutateRow=async(mutate)=>{
    const current=await registry.get(REGISTRY_KEY);
    assert.ok(current!==null,'fixture registry row required');
    await registry.put(REGISTRY_KEY,mutate(current.value),{mustMatchRevision:current.revision});
  };

  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  const instanceKey='task-claim-169';
  const selector={workflowId:'tally-readiness',instanceKey,
    businessPackageId:'business-tally-task-readiness@v2.0.0'};
  const message=messageFor(instanceKey,'message-t06l-claim-01',positiveInput);
  const receipt=await facade.send(message);
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1300));
  assert.equal(receipt.admitted?.effects?.length,1);
  assert.equal(receipt.admitted.effects[0].disposition,'executed');
  const effect=receipt.admitted.effects[0].output;
  assert.equal(effect.state,'ADMITTED_PENDING_AGENT_EFFECT');
  assert.equal(effect.admitted,true);
  assert.equal(effect.authorized,false);
  assert.equal(effect.taskPass,false);
  assert.equal(effect.agentLaunch,false);
  assert.equal(effect.releasePass,false);
  assert.equal(effect.claimObservation?.claimAccepted,true);
  assert.equal(effect.claimObservation?.generation,1);
  assert.equal(effect.claimObservation?.actorRole,'builder');
  assert.equal(effect.claimObservation?.issueRef,ISSUE_REF);
  assert.equal(effect.claimObservation?.sourceSha,sourceSha);
  assert.equal(effect.claimObservation?.commentRef,
    `${ISSUE_REF}#issuecomment-${commentId}`);
  const before=(await facade.query(selector)).snapshot;
  assert.ok(before);
  const recordsBefore=await sqlite.list('');
  assert.ok(recordsBefore.length>0);
  for(const key of recordsBefore){
    const row=await sqlite.get(key);
    assert.ok(!JSON.stringify(row).includes(token),'Host secret may not appear in ANY durable K document: '+key);
    assert.ok(!JSON.stringify(row).includes('DISPATCH_CLAIMED'),
      'raw GitHub comment body may not appear in ANY durable K document: '+key);
  }
  facade.stop();facade=null;
  await sqlite.close();sqlite=null;

  // Reopen the exact SQLite database from a fresh public K activation.
  // Remove the token BEFORE replay; a correct immutable replay needs no API.
  delete process.env.TALLY_GITHUB_READ_TOKEN;
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  assert.deepEqual((await facade.query(selector)).snapshot,before);
  // Replay is the SAME turn (same messageId): the durable turn receipt must
  // short-circuit without re-executing the native C read.
  const replay=await facade.send(messageFor(instanceKey,'message-t06l-claim-01',positiveInput));
  assert.equal(replay.status,'admitted');
  assert.equal(replay.idempotentReplay,true);
  assert.deepEqual(await sqlite.list(''),recordsBefore,
    'replay must not create new journal/effect docs');
  // Changed input on the same turn must be refused, never re-admitted.
  let blocked=false;
  try{
    const x=await facade.send(messageFor(instanceKey,'message-t06l-claim-01',
      {...positiveInput,sourceSha:'b'.repeat(40)}));
    blocked=x.status!=='admitted';
  }catch(e){blocked=typeof e.code==='string';}
  assert.equal(blocked,true);

  // Falsifiers: every registry/witness mutation must fail closed with
  // NOT_ADMITTED and no admitted effect document. In real-registry mode the
  // durable registry is never mutated.
  const falsifiers=[];
  const falsify=async(name,mutateRegistry,expectedReason,inputOverride)=>{
    if(mutateRegistry){
      if(realMode)return falsifiers.push({name,skipped:'REAL_REGISTRY'});
      await mutateRegistry();
    }
    try{
      const r=await facade.send(messageFor(instanceKey+'-'+name,
        'message-t06l-'+name,Object.assign({},positiveInput,inputOverride)));
      assert.equal(r.status,'admitted',name);
      const out=r.admitted?.effects?.[0]?.output??{};
      assert.equal(out.state,'NOT_ADMITTED',name+':'+JSON.stringify(out).slice(0,400));
      assert.equal(out.admitted,false,name);
      assert.equal(out.taskPass,false,name);
      assert.equal(out.agentLaunch,false,name);
      assert.equal(out.releasePass,false,name);
      assert.equal(out.authorized,false,name);
      if(expectedReason)assert.equal(out.reason,expectedReason,name);
      falsifiers.push({name,reason:out.reason});
    }finally{
      if(mutateRegistry&&!realMode)await restoreRow();
    }
  };
  await falsify('f1-wrong-issue',null,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
    {issueRef:'https://github.com/kaicreator-mm/tally/issues/170',issueNumber:170});
  await falsify('f2-wrong-sha-input',null,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
    {sourceSha:'c'.repeat(40)});
  await falsify('f3-missing-row',async()=>{
    const current=await registry.get(REGISTRY_KEY);
    await registry.remove(REGISTRY_KEY,{mustMatchRevision:current.revision});
  },'CLAIM_REGISTRY_RECORD_NOT_FOUND');
  await falsify('f4-pending-phase',async()=>mutateRow(v=>({...v,phase:'PENDING_GITHUB_PUBLISH'})),
    'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED');
  await falsify('f5-forged-dispatch',async()=>mutateRow(v=>({...v,dispatchId:'dispatch:169:forged:1'})),
    'CLAIM_REGISTRY_RESERVATION_MISMATCH');
  await falsify('f6-wrong-row-sha',async()=>mutateRow(v=>({...v,sourceSha:'d'.repeat(40)})),
    'CLAIM_REGISTRY_RESERVATION_MISMATCH');
  await falsify('f7-malformed-row',async()=>mutateRow(v=>({...v,
    commentRef:`${ISSUE_REF}#issuecomment-1`})),
    'CLAIM_REGISTRY_RESERVATION_MISMATCH');
  await falsify('f8-witness-reject',async()=>mutateRow(v=>({...v,
    commentId:commentId-1,commentRef:`${ISSUE_REF}#issuecomment-${commentId-1}`})),
    null);
  if(!realMode){
    const f8=falsifiers.find(x=>x.name==='f8-witness-reject');
    assert.ok(f8&&!f8.skipped,'f8 required in fixture mode');
  }
  // The admitted positive observation is unchanged after every falsifier.
  assert.deepEqual((await facade.query(selector)).snapshot,before);
  console.log(JSON.stringify({provider:'github-native-claim-admission',via:'real-sealed-k-b-c',
    persistedHost:'sqlite',reopened:true,realRegistryMode:realMode,sourceSha,
    admitted:effect.state,taskPass:false,agentLaunch:false,releasePass:false,
    replayWithoutToken:replay.idempotentReplay,docCount:recordsBefore.length,
    changedInputDenied:true,falsifiers:falsifiers.map(x=>x.skipped?`${x.name}:SKIPPED_REAL_REGISTRY`:x.name)}));
  console.log('T06L_REAL_GITHUB_CLAIM_ADMISSION_K_B_C_SQLITE=PASS');
} finally {
  if(facade)facade.stop();
  if(sqlite)await sqlite.close();
  if(registry)await registry.close();
  await rm(tmp,{recursive:true,force:true});
}
