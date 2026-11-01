import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {existsSync,mkdirSync,writeFileSync,readFileSync,mkdtempSync,rmSync,readdirSync,copyFileSync}
  from 'node:fs';
import {createHash} from 'node:crypto';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createProjectIntakeHost,freezeProductIntake,PRODUCT_INTAKE_KEY}
  from './host-project.mjs';

const REPO_ROOT=fileURLToPath(new URL('../../../',import.meta.url));
const run=(cmd,args,opts={})=>{
  // Node refuses to spawn .cmd shims without a shell; route npm via cmd.exe.
  const spawnCmd=process.platform==='win32'&&cmd==='npm'?
    process.env.comspec??'cmd.exe':cmd;
  const spawnArgs=process.platform==='win32'&&cmd==='npm'?['/c','npm',...args]:args;
  const r=spawnSync(spawnCmd,spawnArgs,{encoding:'utf8',windowsHide:true,...opts});
  return {status:r.status,stdout:r.stdout??'',stderr:r.stderr??''};};
const sha256=value=>createHash('sha256').update(value).digest('hex');

const sha='a'.repeat(40),digest='b'.repeat(64);
const env=()=>({TALLY_CLAIM_SOURCE_SHA:sha,TALLY_GITHUB_READ_TOKEN:'token-1234567890',
  TALLY_TASK_CLAIM_210_REGISTRY_PATH:'C:/registry/registry-210.sqlite'});
const claimOk=async()=>({ok:true});
const writeSetOk=async()=>({ok:true});
const frozenValue=Object.freeze({phase:'FROZEN_PRODUCT_INTAKE',
  issueRef:'https://github.com/kaicreator-mm/tally/issues/210',sourceSha:sha,
  productDigest:digest,review:{author:'author:1',reviewer:'reviewer:1'},
  frozenAt:'2026-10-31T09:00:00.000Z'});
function memoryStore(existing){const docs=new Map(existing??[]);
  return {async get(key){const d=docs.get(key);
      return d===undefined?null:{value:d.value,revision:d.revision};},
    async put(key,value,condition){const create=condition?.kind==='create'||condition?.mustBeAbsent===true;
      if(create&&docs.has(key))throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
      docs.set(key,{value,revision:(docs.get(key)?.revision??0)+1});},
    async close(){}};}
const openStoreOk=async options=>memoryStore([[PRODUCT_INTAKE_KEY,
  {value:frozenValue,revision:1}]]);
const host=(overrides={})=>createProjectIntakeHost({openStore:openStoreOk,
  verifyAcceptedClaim:claimOk,verifyFrozenWriteSet:writeSetOk,
  env:env(),...overrides});
const input=()=>({issueNumber:210,sourceSha:sha,productDigest:digest,
  projectTitle:'Tally v2 Intake'});

test('full governed effect: claim, write-set, frozen review, publish and canonical readback',async()=>{
  const publishCalls=[],readCalls=[];
  const h=host({publish:async arg=>{publishCalls.push(arg);
      return {state:'PROJECT_CREATED',project:{number:7,title:'Tally v2 Intake'}};},
    readProjectIntake:async arg=>{readCalls.push(arg);
      return {status:'PROJECT_INTAKE_OBSERVED',projects:[{number:7,
        title:'Tally v2 Intake',url:'https://github.com/kaicreator-mm/tally/projects/7',
        itemCount:1}]};}});
  const result=await h.hostProjectIntake(input());
  assert.equal(result.state,'PROJECT_INTAKE_ADMITTED');
  assert.equal(result.reason,'PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED');
  assert.equal(result.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(result.claimAccepted,true);assert.equal(result.admitted,true);
  assert.equal(result.authorized,false);assert.equal(result.agentLaunch,false);
  assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  assert.equal(result.project.number,7);
  assert.deepEqual(publishCalls.map(x=>x.projectTitle),['Tally v2 Intake']);
  assert.equal(JSON.stringify(publishCalls).includes('token-1234567890'),false,
    'Host module must not hand credentials to transports');
});

test('missing claim, write-set, freeze record or transport fails closed with bounded reasons',async()=>{
  const cases=[
    [host({verifyAcceptedClaim:async()=>({ok:false,reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND'})}),
      'CLAIM_REGISTRY_RECORD_NOT_FOUND'],
    [host({verifyFrozenWriteSet:async()=>({ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'})}),
      'DAG_WRITESET_NOT_VERIFIED'],
    [host({openStore:async()=>memoryStore()}),'PRODUCT_FREEZE_NOT_VERIFIED'],
    [host({openStore:async()=>{throw new Error('disk');}}),'PRODUCT_FREEZE_STORE_UNAVAILABLE'],
    [createProjectIntakeHost({env:env()}),'PROJECT_INTAKE_HOST_CONFIG_INVALID'],
    [host({env:{...env(),TALLY_GITHUB_READ_TOKEN:undefined}}),
      'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE'],
  ];
  for(const [h,reason] of cases){
    const result=await h.hostProjectIntake(input());
    assert.equal(result.state,'NOT_ADMITTED');assert.equal(result.reason,reason);
    assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.taskPass,false);
    assert.equal(result.claimAccepted,false);
  }
});

test('digest or reviewer mismatch against the frozen record refuses',async()=>{
  for(const [change,reason] of [
    [{productDigest:'c'.repeat(64)},'PRODUCT_FREEZE_DIGEST_MISMATCH'],
    [{sourceSha:'d'.repeat(40)},'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH'],
  ]){
    const result=await host().hostProjectIntake({...input(),...change});
    assert.equal(result.reason,reason);assert.equal(result.admitted,false);
  }
  const notIndependent=memoryStore([[PRODUCT_INTAKE_KEY,{value:{...frozenValue,
    review:{author:'same:1',reviewer:'same:1'}},revision:1}]]);
  const result=await host({openStore:async()=>notIndependent}).hostProjectIntake(input());
  assert.equal(result.reason,'PRODUCT_REVIEW_NOT_INDEPENDENT');
});

test('host pin refuses cross-issue, stale source and malformed input before any verification',async()=>{
  const h=host();
  let verifierCalls=0;
  const counting=createProjectIntakeHost({openStore:openStoreOk,
    verifyAcceptedClaim:async()=>{verifierCalls++;return {ok:true};},
    verifyFrozenWriteSet:writeSetOk,env:env()});
  for(const change of [
    {issueNumber:197},{sourceSha:'short'},{productDigest:'nothex'},
    {projectTitle:''},{sourceSha:'e'.repeat(40)},
  ]){
    const result=await counting.hostProjectIntake({...input(),...change});
    assert.equal(result.reason,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
    assert.equal(result.admitted,false);
  }
  assert.equal(verifierCalls,0,'no verifier may run on pinned-input refusal');
  assert.equal(await h.hostProjectIntake(null).then(r=>r.reason),
    'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
});

test('safe replay admits the intent but keeps the GitHub effect explicitly NOT_RUN',async()=>{
  const h=host({env:{...env(),TALLY_LOCAL_REPLAY:'1'}});
  const result=await h.hostProjectIntake(input());
  assert.equal(result.state,'REPLAY_COMPLETE');assert.equal(result.reason,'REPLAY_FIXTURE_ONLY');
  assert.equal(result.githubEffect,'NOT_RUN');assert.equal(result.replayFixture,true);
  assert.equal(result.claimAccepted,true);assert.equal(result.taskPass,false);
});

test('publish failure stays UNKNOWN and never fabricates a verified project',async()=>{
  for(const publish of [
    async()=>{throw new Error('network down');},
    async()=>({state:'BOGUS'}),
    async()=>({state:'PROJECT_CREATED',project:{number:0,title:'x'}}),
  ]){
    const result=await host({publish,readProjectIntake:async()=>null})
      .hostProjectIntake(input());
    assert.equal(result.state,'PROJECT_INTAKE_UNKNOWN');
    assert.equal(result.reason,'GITHUB_PUBLISH_UNKNOWN');
    assert.equal(result.githubEffect,'UNKNOWN');
  }
  const mismatched=host({publish:async()=>({state:'PROJECT_CREATED',
      project:{number:9,title:'Tally v2 Intake'}}),
    readProjectIntake:async()=>({status:'PROJECT_INTAKE_OBSERVED',projects:[]})});
  const result=await mismatched.hostProjectIntake(input());
  assert.equal(result.reason,'PROJECT_INTAKE_READBACK_MISMATCH');
  assert.equal(result.githubEffect,'UNKNOWN');assert.equal(result.project,null);
});

test('freeze writes one CAS record only after an independent review',async()=>{
  const store=memoryStore();
  const args={openStore:async()=>store,registryPath:'C:/registry/r210.sqlite',
    sourceSha:sha,productDigest:digest,author:'author:1',reviewer:'reviewer:1',
    frozenAt:'2026-10-31T09:00:00.000Z'};
  const selfReview=await freezeProductIntake({...args,reviewer:'author:1'});
  assert.equal(selfReview.state,'NOT_FROZEN');
  assert.equal(selfReview.reason,'PRODUCT_REVIEW_NOT_INDEPENDENT');
  const frozen=await freezeProductIntake(args);
  assert.equal(frozen.state,'FROZEN_COMPLETE');assert.equal(frozen.frozen,true);
  assert.equal(frozen.receiptRef,`receipt:sha256:${digest}`);
  const record=(await store.get(PRODUCT_INTAKE_KEY)).value;
  assert.equal(record.phase,'FROZEN_PRODUCT_INTAKE');assert.equal(record.sourceSha,sha);
  assert.equal(record.review.reviewer,'reviewer:1');
  const replay=await freezeProductIntake(args);
  assert.equal(replay.state,'FROZEN_COMPLETE');
  assert.equal(replay.reason,'PRODUCT_FREEZE_ALREADY_RECORDED');
  const changed=await freezeProductIntake({...args,productDigest:'c'.repeat(64)});
  assert.equal(changed.state,'NOT_FROZEN');
  assert.equal(changed.reason,'PRODUCT_FREEZE_DIGEST_MISMATCH');
});


// --- T210 physical producer + real K consumer replay (env-gated, no new paths) ---
// Runs ONLY with T210_REPLAY=1 and DH_V0103=<DH v0.10.3 checkout>. It packs the
// real DH v0.10.3 packages, runs the REAL producer.mjs with T210_PROJECT_INTAKE=1
// (and without it), then loads the sealed app with the REAL Microkernel and
// sends freezeProductIntake/admitProjectIntake through facade.send with a
// replay-mode provider that reuses the real host-project.mjs logic. Physical
// GitHub effects and Claim/registry verification stay explicitly NOT_RUN.
const REPLAY_DH_SHA='061376f4a023070f215f9feffad5c24d96d503cb';
const REPLAY_CANARY='UNIT_TEST_SECRET_t210_replay_do_not_leak';
const REPLAY_PROVIDER=String.raw`import {createProjectIntakeHost,freezeProductIntake as hostFreezeProductIntake}
  from '__TALLY_ROOT__experiments/v2/project-intake/host-project.mjs';
const docs=new Map();
let creates=0;
const openStore=async()=>({
  async get(key){const d=docs.get(key);return d===undefined?null:{value:d.value,revision:d.revision};},
  async put(key,value,condition){
    const create=condition&&(condition.kind==='create'||condition.mustBeAbsent===true);
    if(create&&docs.has(key))
      throw Object.assign(new Error('CAS refused'),{code:'HOST_CAS_CONFLICT'});
    const previous=docs.get(key);
    const revision=(previous?previous.revision:0)+1;
    docs.set(key,{value,revision});
    if(create)creates++;
    return {revision};},
  async close(){}});
globalThis.__T210_REPLAY_STORE={
  createCount:()=>creates,
  frozenRecord:()=>docs.get('tally:v2:product-intake:210')||null};
const env=Object.assign({},process.env,{TALLY_LOCAL_REPLAY:'1',
  TALLY_TASK_CLAIM_210_REGISTRY_PATH:'T210_REPLAY_IN_MEMORY'});
const host=createProjectIntakeHost({env,openStore,
  verifyAcceptedClaim:async()=>({ok:true}),
  verifyFrozenWriteSet:async()=>({ok:true})});
export async function freezeProductIntake(input){
  try{
    const result=await hostFreezeProductIntake({openStore,registryPath:'T210_REPLAY_IN_MEMORY',
      sourceSha:input&&input.sourceSha,productDigest:input&&input.productDigest,
      author:input&&input.author,reviewer:input&&input.reviewer,
      frozenAt:new Date().toISOString(),caseRefDigest:null});
    console.error('T210_PROVIDER_FREEZE_RETURN',JSON.stringify(result));
    return result;
  }catch(error){console.error('T210_PROVIDER_FREEZE_ERROR',
    error&&error.message,error&&error.code);throw error;}
}
export async function hostProjectIntake(input){
  try{
    const result=await host.hostProjectIntake(input);
    console.error('T210_PROVIDER_ADMIT_RETURN',JSON.stringify(result));
    return result;
  }
  catch(error){console.error('T210_PROVIDER_ADMIT_ERROR',
    error&&error.message,error&&error.code);throw error;}
}
export async function runLocalCodexAttempt(input){
  return {issueRef:input&&input.sourceRef,sourceSha:input&&input.sourceSha,
    attemptId:input&&input.attemptId,claimGeneration:1,claimAccepted:true,
    admitted:true,agentLaunch:false,replayFixture:true,authorized:false,
    taskPass:false,releasePass:false,state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY'};
}
export async function joinReviewedCandidates(input){
  return {issueRef:input&&input.sourceRef,sourceSha:input&&input.sourceSha,
    joinId:input&&input.joinId,claimGeneration:1,claimAccepted:true,
    admitted:true,agentLaunch:false,replayFixture:true,authorized:false,
    taskPass:false,releasePass:false,state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY'};
}
export async function publishReviewedCandidate(){
  return {state:'NOT_ADMITTED',reason:'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
    authorized:false,taskPass:false,releasePass:false};
}
`;

const REPLAY_CONSUMER=String.raw`import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sha=x=>createHash('sha256').update(x).digest('hex');
const sealed=JSON.parse(readFileSync(process.env.T210_SEALED_PATH,'utf8'));
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-local-codex@v1.0.0');
assert.ok(selectedC&&selectedC.moduleSource,'sealed local-codex capability required');
assert.equal(sha(selectedC.moduleSource),selectedC.moduleSha256);
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'provider digest marker required');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
assert.equal(sha(readFileSync(providerPath)),digestMatch[1],'PROVIDER_INTEGRITY_MISMATCH');
const mode=process.env.T210_CONSUMER_MODE;
const sourceSha=process.env.T210_REPLAY_SOURCE_SHA;
const productDigest=process.env.T210_REPLAY_PRODUCT_DIGEST;
const issueRef='https://github.com/kaicreator-mm/tally/issues/210';
const deployment={deployedAppId:'deploy:tally-v2-t210-replay',tenantId:'tenant:tally-v2',
  appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>!!identity&&identity.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c&&c.kind==='create'?{mustBeAbsent:true}:
  c&&c.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw new Error('INVALID_K_CONDITION');})();
const dbPath=process.env.T210_CONSUMER_TMP+'/kernel.sqlite';
const openFacade=async()=>{
  const sqlite=await openSqliteCasStore({path:dbPath,busyTimeoutMs:1000});
  const facade=await load(sealed,{store:{
    get:key=>sqlite.get(key),
    put:(key,value,cond)=>sqlite.put(key,value,condition(cond)),
    remove:(key,cond)=>sqlite.remove(key,condition(cond))},
    deploymentIdentity:deployment,deploymentRoot:process.cwd(),
    tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,31)});
  return {sqlite,facade};
};
const freezeInput={issueRef,issueNumber:210,sourceSha,productDigest,
  author:'author:t210',reviewer:'reviewer:t210'};
if(mode==='noflag'){
  const {sqlite,facade}=await openFacade();
  for(const intentType of ['admitProjectIntake','freezeProductIntake']){
    let refused=false,detail='';
    try{
      const receipt=await facade.send({intentType,instanceKey:'intake-210',
        messageId:'t210-noflag-'+intentType,
        input:{issueRef,issueNumber:210,sourceSha,productDigest,
          author:'author:t210',reviewer:'reviewer:t210',projectTitle:'Tally v2 Intake'}});
      detail=JSON.stringify(receipt).slice(0,200);
      refused=receipt.status!=='admitted'||!(receipt.admitted&&
        receipt.admitted.effects&&receipt.admitted.effects.length>0);
    }catch(error){refused=true;detail=String(error&&error.code||error&&error.message||error).slice(0,120);}
    assert.ok(refused,'unselected intent must not be admitted: '+intentType+' '+detail);
  }
  console.log('T210_NOFLAG_ROUTES_ABSENT=PASS');
  facade.stop();await sqlite.close();process.exit(0);
}
const {sqlite,facade}=await openFacade();
const receipt=await facade.send({intentType:'freezeProductIntake',
  instanceKey:'intake-210',messageId:'t210-freeze-1',input:Object.assign({},freezeInput)});
assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,900));
assert.equal(receipt.admitted.effects.length,1);
const effect=receipt.admitted.effects[0];
assert.equal(effect.disposition,'executed');
assert.equal(effect.output.state,'FROZEN_COMPLETE',JSON.stringify(effect.output).slice(0,400));
assert.equal(effect.output.frozen,true);
assert.equal(effect.output.authorized,false);
assert.equal(effect.output.taskPass,false);
assert.equal(effect.output.releasePass,false);
const createsAfterFirst=globalThis.__T210_REPLAY_STORE.createCount();
const sameMessageReplay=await facade.send({intentType:'freezeProductIntake',
  instanceKey:'intake-210',messageId:'t210-freeze-1',input:Object.assign({},freezeInput)});
assert.equal(sameMessageReplay.status,'admitted',JSON.stringify(sameMessageReplay).slice(0,400));
if(sameMessageReplay.admitted&&sameMessageReplay.admitted.effects.length>0)
  assert.equal(sameMessageReplay.admitted.effects[0].output.frozen,true);
assert.equal(globalThis.__T210_REPLAY_STORE.createCount(),createsAfterFirst,
  'a same-message resend must never write another CAS create');
const secondMessage=await facade.send({intentType:'freezeProductIntake',
  instanceKey:'intake-210',messageId:'t210-freeze-2',input:Object.assign({},freezeInput)});
assert.equal(globalThis.__T210_REPLAY_STORE.createCount(),createsAfterFirst,
  'a second freeze must never write another CAS create');
if(secondMessage.status==='admitted'&&secondMessage.admitted.effects.length>0)
  assert.equal(secondMessage.admitted.effects[0].output.frozen,true);
const admitReceipt=await facade.send({intentType:'admitProjectIntake',
  instanceKey:'intake-210',messageId:'t210-admit-1',
  input:{issueRef,issueNumber:210,sourceSha,productDigest,
    projectTitle:'Tally v2 Intake Replay'}});
assert.equal(admitReceipt.status,'admitted',JSON.stringify(admitReceipt).slice(0,900));
const admitEffect=admitReceipt.admitted.effects[0].output;
assert.equal(admitEffect.state,'REPLAY_COMPLETE',JSON.stringify(admitEffect).slice(0,400));
assert.equal(admitEffect.reason,'REPLAY_FIXTURE_ONLY');
assert.equal(admitEffect.githubEffect,'NOT_RUN');
assert.equal(admitEffect.admitted,true);
assert.equal(admitEffect.taskPass,false);
assert.equal(admitEffect.releasePass,false);
const keys=await sqlite.list('');
let docCount=0;
for(const key of keys){
  const row=await sqlite.get(key);
  const text=JSON.stringify(row);
  docCount++;
  assert.ok(text.indexOf(process.env.T210_REPLAY_CANARY)===-1,'secret must not enter K documents');
  assert.ok(text.indexOf('DISPATCH_CLAIMED')===-1,'raw claim body must not enter K documents');
}
assert.ok(docCount>0,'K journal must contain documents');
const answer=await facade.query({workflowId:'tally-readiness',instanceKey:'intake-210',
  businessPackageId:'business-tally-task-readiness@v2.0.0'});
assert.ok(answer.snapshot);
facade.stop();await sqlite.close();
// Durable journal replay across a full reopen (consumer-local-codex pattern).
const reopenedStore=await openSqliteCasStore({path:dbPath,busyTimeoutMs:1000});
const reopenedFacade=await load(sealed,{store:{
  get:key=>reopenedStore.get(key),
  put:(key,value,cond)=>reopenedStore.put(key,value,condition(cond)),
  remove:(key,cond)=>reopenedStore.remove(key,condition(cond))},
  deploymentIdentity:deployment,deploymentRoot:process.cwd(),
  tenantId:deployment.tenantId,queryAuthority:authority,now:()=>Date.UTC(2026,9,31)});
const durableReplay=await reopenedFacade.send({intentType:'freezeProductIntake',
  instanceKey:'intake-210',messageId:'t210-freeze-1',input:Object.assign({},freezeInput)});
assert.equal(durableReplay.status,'admitted',JSON.stringify(durableReplay).slice(0,400));
assert.equal(durableReplay.idempotentReplay,true,'durable journal replay expected');
assert.equal(globalThis.__T210_REPLAY_STORE.createCount(),createsAfterFirst,
  'durable replay must not write another CAS create');
reopenedFacade.stop();await reopenedStore.close();
console.log(JSON.stringify({realKLoad:true,realKSend:receipt.status,
  freezeEffect:effect.output.state,freezeReceipt:effect.output.receiptRef,
  durableIdempotentReplay:durableReplay.idempotentReplay,
  secondFreezeCreates:createsAfterFirst,admitEffect:admitEffect.state,
  admitGithubEffect:admitEffect.githubEffect,docCount,
  physicalGithubEffect:'NOT_RUN',authorized:false,taskPass:false,releasePass:false}));
console.log('T210_REPLAY_CONSUMER=PASS');
`;

test('T210 sealed producer + real K consumer replay (physical DH v0.10.3 run)',{
  skip:process.env.T210_REPLAY!=='1'||!process.env.DH_V0103?
    'set T210_REPLAY=1 and DH_V0103=<DH v0.10.3 checkout> to run the physical replay':false,
},()=>{
  const dh=process.env.DH_V0103;
  assert.equal(run('git',['-C',dh,'rev-parse','HEAD']).stdout.trim(),REPLAY_DH_SHA,
    'DH_V0103 must be the exact v0.10.3 checkout pinned by the existing lanes');
  if(!existsSync(join(dh,'tooling','build','dist','index.js'))){
    const build=run('npm',['run','build','-w','@kaicreator/domain-harness-contracts',
      '-w','@kaicreator/domain-harness-build','-w','@kaicreator/domain-harness-microkernel',
      '-w','@kaicreator/domain-harness-host-node'],{cwd:dh,timeout:600000});
    assert.equal(build.status,0,build.stderr.slice(-2000));
  }
  const root=mkdtempSync(join(tmpdir(),'t210-replay-'));
  try{
    const packed=join(root,'packed');mkdirSync(packed);
    for(const name of ['@kaicreator/domain-harness-contracts','@kaicreator/domain-harness-build',
      '@kaicreator/domain-harness-microkernel','@kaicreator/domain-harness-host-node'])
      assert.equal(run('npm',['pack','--pack-destination',packed,'-w',name],{cwd:dh}).status,0,name);
    for(const dir of ['./domain-packages/kernel/basic','./domain-packages/standard/basic'])
      assert.equal(run('npm',['pack','--pack-destination',packed,dir],{cwd:dh}).status,0,dir);
    const packedTarballs=()=>readdirSync(packed);
    assert.ok(packedTarballs().length>=6,'six DH tarballs required, got '+packedTarballs().length);
    const tgz=prefix=>{const f=packedTarballs().find(x=>x.startsWith(prefix));
      assert.ok(f,'missing tarball '+prefix);return join(packed,f);};
    // Replay provider fixture: real host-project.mjs logic, in-memory Host CAS.
    const providerDir=join(root,'provider');mkdirSync(providerDir);
    writeFileSync(join(providerDir,'package.json'),JSON.stringify({name:'@kaicreator/tally-local-codex-host-bundle',
      version:'0.0.0',private:true,type:'module',exports:'./index.mjs'}));
    writeFileSync(join(providerDir,'index.mjs'),
      REPLAY_PROVIDER.replace('__TALLY_ROOT__','file:///'+REPO_ROOT.replace(/\\/g,'/')));
    assert.equal(run('npm',['pack','--pack-destination',packed,providerDir]).status,0,'provider pack');
    const producerDir=join(root,'producer');mkdirSync(producerDir);
    writeFileSync(join(producerDir,'package.json'),JSON.stringify({name:'tally-t210-producer',
      private:true,type:'module'}));
    const producerInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund',tgz('kaicreator-domain-harness-build-'),tgz('kaicreator-domain-harness-contracts-'),
      tgz('kaicreator-domain-harness-microkernel-'),tgz('kaicreator-domain-harness-host-node-'),
      tgz('kaicreator-domain-kernel-basic-'),tgz('kaicreator-domain-standard-basic-')],
      {cwd:producerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:300000});
    assert.equal(producerInstall.status,0,producerInstall.stderr.slice(-2000));
    const businessSource=join(REPO_ROOT,'packages','tally-engineering-business','src','index.mjs');
    // Node resolves bare specifiers from the importing file's location, so
    // run the producer from a copy inside the packed install (run-clean precedent).
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','producer.mjs'),
      join(producerDir,'producer.mjs'));
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','readiness-capability.mjs'),
      join(producerDir,'readiness-capability.mjs'));
    copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build','local-codex-shim.mjs'),
      join(producerDir,'local-codex-shim.mjs'));
    const producerRun=flag=>run(process.execPath,[join(producerDir,'producer.mjs')],
      {cwd:producerDir,env:{...process.env,TALLY_BUSINESS_SOURCE:businessSource,
        T06M_LOCAL:'1',T06M_PROVIDER_FILE:join(providerDir,'index.mjs'),
        T02B_SEALED_APP_OUTPUT:join(root,flag?'sealed-app.json':'sealed-app-noflag.json'),
        ...(flag?{T210_PROJECT_INTAKE:'1'}:{})},timeout:300000});
    const withFlag=producerRun(true);
    assert.equal(withFlag.status,0,withFlag.stderr.slice(-2000)+withFlag.stdout.slice(-2000));
    assert.match(withFlag.stdout,/T02B_PUBLIC_BUILD_SELECTED_B=PASS/);
    assert.match(withFlag.stdout,/"projectIntakeRoutes": "SELECTED"/);
    const sealedPath=join(root,'sealed-app.json');
    const sealed=JSON.parse(readFileSync(sealedPath,'utf8'));
    for(const intent of ['freezeProductIntake','admitProjectIntake']){
      const route=sealed.routes.find(r=>r.intentId===intent);
      assert.ok(route,'intake route missing: '+intent);
      assert.equal(route.capabilityId,'tally-local-codex');
    }
    const withoutFlag=producerRun(false);
    assert.equal(withoutFlag.status,0,withoutFlag.stderr.slice(-2000));
    assert.match(withoutFlag.stdout,/"projectIntakeRoutes": "NOT_SELECTED"/);
    const noflagPath=join(root,'sealed-app-noflag.json');
    const noflag=JSON.parse(readFileSync(noflagPath,'utf8'));
    assert.equal(noflag.routes.some(r=>['freezeProductIntake','admitProjectIntake'].includes(r.intentId)),
      false,'intake routes must be absent without T210_PROJECT_INTAKE=1');
    assert.ok(!readFileSync(sealedPath,'utf8').includes(REPLAY_CANARY),'no canary in sealed bytes');
    const consumerDir=join(root,'consumer');mkdirSync(consumerDir);
    writeFileSync(join(consumerDir,'package.json'),JSON.stringify({name:'tally-t210-consumer',
      private:true,type:'module'}));
    const consumerInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund',tgz('kaicreator-domain-harness-contracts-'),tgz('kaicreator-domain-harness-microkernel-'),
      tgz('kaicreator-domain-harness-host-node-'),tgz('kaicreator-tally-local-codex-host-bundle-')],
      {cwd:consumerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:600000});
    assert.equal(consumerInstall.status,0,consumerInstall.stderr.slice(-2000));
    // better-sqlite3 needs its install script to fetch the Node24 prebuild.
    const nativeInstall=run('npm',['install','--save-exact','--no-audit','--no-fund',
      'better-sqlite3@12.4.1'],
      {cwd:consumerDir,env:{...process.env,npm_config_registry:'https://registry.npmmirror.com'},
        timeout:600000});
    assert.equal(nativeInstall.status,0,nativeInstall.stderr.slice(-2000));
    writeFileSync(join(consumerDir,'consumer-t210.mjs'),REPLAY_CONSUMER);
    const sourceSha=run('git',['-C',REPO_ROOT,'rev-parse','HEAD']).stdout.trim();
    assert.match(sourceSha,/^[0-9a-f]{40}$/);
    const productDigest=sha256('t210 replay product fixture: goal + L2 + DAG digest');
    const dataDir=join(root,'consumer-data');mkdirSync(dataDir);
    const consumerEnv=mode=>({...process.env,
      T210_SEALED_PATH:mode==='flag'?sealedPath:noflagPath,
      T210_CONSUMER_MODE:mode,T210_REPLAY_SOURCE_SHA:sourceSha,
      TALLY_CLAIM_SOURCE_SHA:sourceSha,
      T210_REPLAY_PRODUCT_DIGEST:productDigest,T210_REPLAY_CANARY:REPLAY_CANARY,
      T210_CONSUMER_TMP:dataDir});
    const flagged=run(process.execPath,[join(consumerDir,'consumer-t210.mjs')],
      {cwd:consumerDir,env:consumerEnv('flag'),timeout:300000});
    assert.equal(flagged.status,0,flagged.stderr.slice(-3000)+flagged.stdout.slice(-3000));
    assert.match(flagged.stdout,/T210_REPLAY_CONSUMER=PASS/);
    const unflagged=run(process.execPath,[join(consumerDir,'consumer-t210.mjs')],
      {cwd:consumerDir,env:consumerEnv('noflag'),timeout:300000});
    assert.equal(unflagged.status,0,unflagged.stderr.slice(-3000));
    assert.match(unflagged.stdout,/T210_NOFLAG_ROUTES_ABSENT=PASS/);
    console.log('T210_PHYSICAL_REPLAY_EVIDENCE='+JSON.stringify({
      producerRoutes:'SELECTED',noFlagRoutes:'ABSENT',freeze:'FROZEN_COMPLETE',
      freezeIdempotentReplay:true,admit:'REPLAY_COMPLETE',admitGithubEffect:'NOT_RUN',
      physicalGithubEffect:'NOT_RUN',dh:REPLAY_DH_SHA}));
  }finally{rmSync(root,{recursive:true,force:true});}
});
