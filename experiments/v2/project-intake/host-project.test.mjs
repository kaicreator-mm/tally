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


import {createProductReviewHost312,PRODUCT312_DRAFT_KEY,
  PRODUCT312_FREEZE_KEY} from './host-project.mjs';
const p312Source='a'.repeat(40),p312Pack='b'.repeat(40),p312Set='c'.repeat(64);
const p312Draft={product:{goal:'Real Product ReviewWitness',nonGoals:['No fake Task PASS']},
  l2:{decision:'KEEP_MERGED_L2_ARCHITECTURE'},provenance:['157','158']};
const p312Input={sourceSha:p312Source,taskPackBlobSha:p312Pack,
  writeSetSha256:p312Set,draft:p312Draft};
const p312Claim=role=>({accepted:true,issueNumber:312,role,sourceSha:p312Source,
  taskPackBlobSha:p312Pack,writeSetSha256:p312Set,claimRef:'claim:312:'+role,
  operatorId:'operator:'+role,writerId:'writer:'+role,transportActor:'kaicreator-mm'});
const p312Make=(overrides={})=>{
  const store=memoryStore();
  const calls=[];
  const provider=createProductReviewHost312({
    openStore:async()=>store,
    verifyAcceptedClaim:async(issue,role)=>{
      calls.push('claim:'+role);
      return {ok:true,claim:p312Claim(role)};},
    verifyFrozenWriteSet:async()=>({ok:true,
      taskPackBlobSha:p312Pack,writeSetSha256:p312Set}),
    readCurrentVersion:async()=>({ok:true,sha:p312Source}),
    readReviewWitness:async({expected,reviewerClaim})=>({status:'INDEPENDENTLY_VERIFIED',
      independentlyVerified:true,sourceQualified:true,
      authorized:false,taskPass:false,releasePass:false,
      sourceSha:expected.sourceSha,productDigest:expected.productDigest,
      draftRef:expected.draftRef,taskPackBlobSha:expected.taskPackBlobSha,
      writeSetSha256:expected.writeSetSha256,
      reviewedHead:expected.reviewedHead,reviewedTree:expected.reviewedTree,
      reviewedBase:expected.reviewedBase,
      reviewerOperatorId:reviewerClaim.operatorId,
      reviewerWriterId:reviewerClaim.writerId,p0:0,p1:0,p2:0,
      reviewRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345'}),
    withVersionFence:async(meta,fn)=>{calls.push('fence');return fn();},
    env:{TALLY_CLAIM_SOURCE_SHA:p312Source,
      TALLY_TASK_CLAIM_312_REGISTRY_PATH:'fixture.sqlite'},
    ...overrides});
  return {provider,calls,store};
};
const p312FreezeArgs={sourceSha:p312Source,taskPackBlobSha:p312Pack,
  writeSetSha256:p312Set,reviewCommentId:12345,
  reviewedHead:'d'.repeat(40),reviewedTree:'e'.repeat(40),
  reviewedBase:p312Source};
test('#312 trusted Host persists immutable draft digest (not caller), then CAS freezes witnessed source',async()=>{
  const {provider,store,calls}=p312Make();
  const written=await provider.persistDraft({...p312Input,productDigest:'9'.repeat(64)});
  assert.equal(written.state,'DRAFT_PENDING_REVIEW');
  assert.match(written.draftRef,/^draft:sha256:[a-f0-9]{64}$/);
  assert.notEqual(written.productDigest,'9'.repeat(64));
  const actual=(await store.get(PRODUCT312_DRAFT_KEY+p312Source)).value;
  assert.equal(actual.productDigest,written.productDigest);
  const frozen=await provider.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(frozen.state,'FROZEN_COMPLETE');
  assert.equal(frozen.receiptRef,'receipt:sha256:'+written.productDigest);
  assert.equal(frozen.taskPass,false);assert.equal(frozen.releasePass,false);
  assert.equal((await store.get(PRODUCT312_FREEZE_KEY+p312Source)).value.reviewRef,
    'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345');
  assert.ok(calls.includes('claim:builder'));
  assert.ok(calls.includes('claim:reviewer'));
  assert.ok(calls.includes('fence'));
  const repeated=await provider.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(repeated.state,'FROZEN_COMPLETE');
  assert.equal((await provider.persistDraft(p312Input)).state,'DRAFT_PENDING_REVIEW');
});

test('#312 Host cannot freeze UI-only reviewer string without accepted draft and two roles',async()=>{
  const {provider}=p312Make();
  assert.equal((await provider.freezeReviewedDraft(p312FreezeArgs)).reason,
    'PRODUCT_DRAFT_NOT_PERSISTED_OR_STALE');
  await provider.persistDraft(p312Input);
  const refuse=p312Make({verifyAcceptedClaim:async(issue,role)=>({
    ok:true,claim:{...p312Claim(role),
      operatorId:role==='reviewer'?'operator:builder':'operator:builder'}})});
  await refuse.provider.persistDraft(p312Input);
  assert.equal((await refuse.provider.freezeReviewedDraft(p312FreezeArgs)).reason,
    'PRODUCT_REVIEW_PROCESS_OR_CLAIM_UNVERIFIED');
});

test('#312 Host no shared physical fence, edited/missing witness or moved source stays UNKNOWN',async()=>{
  const {provider,store}=p312Make({withVersionFence:undefined});
  await provider.persistDraft(p312Input);
  assert.equal((await provider.freezeReviewedDraft(p312FreezeArgs)).reason,
    'PRODUCT_REVIEW_SHARED_HOST_FENCE_REQUIRED');
  assert.equal(await store.get(PRODUCT312_FREEZE_KEY+p312Source),null);
  const lost=p312Make({readReviewWitness:async()=>({status:'UNKNOWN',reason:'COMMENT_EDITED'})});
  await lost.provider.persistDraft(p312Input);
  assert.equal((await lost.provider.freezeReviewedDraft(p312FreezeArgs)).reason,
    'PRODUCT_REVIEW_WITNESS_UNVERIFIED');
  const moved=p312Make({readCurrentVersion:async()=>({ok:true,sha:'9'.repeat(40)})});
  assert.equal((await moved.provider.persistDraft(p312Input)).reason,
    'PRODUCT_DRAFT_SOURCE_OR_CLAIM_STALE');
});

test('#312 Host ACK-loss after CAS write remains UNKNOWN, no fake receipt or Task PASS',async()=>{
  const store=memoryStore();let ambiguous=true;
  const openStore=async()=>({
    get:key=>store.get(key),
    close:()=>store.close(),
    async put(key,value,condition){
      await store.put(key,value,condition);
      if(ambiguous){ambiguous=false;throw Error('ACK lost after server committed');}
    }
  });
  const {provider}=p312Make({openStore});
  const first=await provider.persistDraft(p312Input);
  assert.equal(first.state,'UNKNOWN');
  assert.equal(first.reason,'PRODUCT_DRAFT_CAS_UNKNOWN');
  assert.equal(first.taskPass,false);
  const retry=await provider.persistDraft(p312Input);
  assert.equal(retry.state,'DRAFT_PENDING_REVIEW');
  assert.equal((await store.get(PRODUCT312_DRAFT_KEY+p312Source)).value.phase,
    'DRAFT_PENDING_REVIEW');
});


test('#312 Host rejects a falsely positive C status with missing coordinate binding',async()=>{
  const bad=p312Make({readReviewWitness:async({expected,reviewerClaim})=>({
    status:'INDEPENDENTLY_VERIFIED',independentlyVerified:true,
    sourceQualified:true,authorized:false,taskPass:false,releasePass:false,
    sourceSha:expected.sourceSha,productDigest:expected.productDigest,
    draftRef:expected.draftRef,taskPackBlobSha:expected.taskPackBlobSha,
    writeSetSha256:expected.writeSetSha256,
    reviewedHead:'9'.repeat(40),reviewedTree:expected.reviewedTree,
    reviewedBase:expected.reviewedBase,
    reviewerOperatorId:reviewerClaim.operatorId,
    reviewerWriterId:reviewerClaim.writerId,
    p0:0,p1:0,p2:0,
    reviewRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345'})});
  await bad.provider.persistDraft(p312Input);
  const denied=await bad.provider.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(denied.state,'UNKNOWN');
  assert.equal(denied.reason,'PRODUCT_REVIEW_WITNESS_UNVERIFIED');
  assert.equal(await bad.store.get(PRODUCT312_FREEZE_KEY+p312Source),null);
});


test('#312 second witness read inside shared fence catches comment edit with zero Freeze CAS',async()=>{
  let reads=0;
  const {provider,store,calls}=p312Make({
    readReviewWitness:async({expected,reviewerClaim})=>{
      reads++;
      if(reads===2)return {status:'UNKNOWN',reason:'GITHUB_COMMENT_EDITED'};
      return {status:'INDEPENDENTLY_VERIFIED',independentlyVerified:true,
        sourceQualified:true,authorized:false,taskPass:false,releasePass:false,
        sourceSha:expected.sourceSha,productDigest:expected.productDigest,
        draftRef:expected.draftRef,taskPackBlobSha:expected.taskPackBlobSha,
        writeSetSha256:expected.writeSetSha256,
        reviewedHead:expected.reviewedHead,reviewedTree:expected.reviewedTree,
        reviewedBase:expected.reviewedBase,
        reviewerOperatorId:reviewerClaim.operatorId,
        reviewerWriterId:reviewerClaim.writerId,
        p0:0,p1:0,p2:0,
        reviewRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345'};
    }
  });
  await provider.persistDraft(p312Input);
  const frozen=await provider.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(reads,2);
  assert.ok(calls.includes('fence'));
  assert.equal(frozen.state,'UNKNOWN');
  assert.equal(frozen.reason,'PRODUCT_REVIEW_WITNESS_UNVERIFIED');
  assert.equal(frozen.taskPass,false);
  assert.equal(await store.get(PRODUCT312_FREEZE_KEY+p312Source),null);
});


const p312OfficialGitTransport=commentReader=>({rest:{
  issues:{getComment:commentReader},
  pulls:{get:async()=>({data:{number:401,state:'open',merged:false,
    head:{sha:'d'.repeat(40),repo:{full_name:'kaicreator-mm/tally'}},
    base:{sha:p312Source,ref:'version/v2',
      repo:{full_name:'kaicreator-mm/tally'}}}})},
  repos:{
    getCommit:async({ref})=>({data:{sha:ref,commit:{tree:{sha:
      ref==='d'.repeat(40)?'e'.repeat(40):'b'.repeat(40)}}}}),
    getBranch:async()=>({data:{commit:{sha:p312Source}}})}}});
test('#312 Host binds the real GitHub C witness validator when only octokit is injected',async()=>{
  // No readReviewWitness injection: the C→Host source wiring must route the
  // REAL readProductReviewWitnessFromClient over the selected C client.
  const store=memoryStore();
  let comment=null;
  const provider=createProductReviewHost312({
    openStore:async()=>store,
    verifyAcceptedClaim:async(issue,role)=>({ok:true,claim:p312Claim(role)}),
    verifyFrozenWriteSet:async()=>({ok:true,
      taskPackBlobSha:p312Pack,writeSetSha256:p312Set}),
    readCurrentVersion:async()=>({ok:true,sha:p312Source}),
    octokit:p312OfficialGitTransport(async()=>{
      if(!comment)throw {status:404};
      return {data:comment};
    }),
    withVersionFence:async(meta,fn)=>fn(),
    env:{TALLY_CLAIM_SOURCE_SHA:p312Source,
      TALLY_TASK_CLAIM_312_REGISTRY_PATH:'fixture.sqlite'}});
  const persisted=await provider.persistDraft(p312Input);
  assert.equal(persisted.state,'DRAFT_PENDING_REVIEW');
  // The canonical witness comment is authored AFTER the Host persisted the
  // draft, binding the Host-derived digest (never a caller digest).
  comment={id:12345,
    html_url:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345',
    issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/312',
    user:{login:'kaicreator-mm'},created_at:'2026-11-12T12:00:00Z',
    updated_at:'2026-11-12T12:00:00Z',body:JSON.stringify({
      schema:'tally.v2.product-review-witness/2',status:'INDEPENDENTLY_VERIFIED',
      repository:'kaicreator-mm/tally',issueNumber:312,reviewedPullNumber:401,
      draftRef:persisted.draftRef,productDigest:persisted.productDigest,
      sourceSha:p312Source,taskPackBlobSha:p312Pack,writeSetSha256:p312Set,
      reviewedHead:'d'.repeat(40),reviewedTree:'e'.repeat(40),
      reviewedBase:p312Source,
      builderClaimRef:'claim:312:builder',reviewerClaimRef:'claim:312:reviewer',
      reviewerOperatorId:'operator:reviewer',reviewerWriterId:'writer:reviewer',
      p0:0,p1:0,p2:1})};
  const frozen=await provider.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(frozen.state,'FROZEN_COMPLETE');
  assert.equal(frozen.taskPackBlobSha,p312Pack);
  assert.equal(frozen.writeSetSha256,p312Set);
  assert.equal(frozen.receiptRef,'receipt:sha256:'+persisted.productDigest);
  // Same login, distinct accepted reviewer process: GitHub login equality is
  // provenance only; the edited comment still fails closed.
  const edited={...comment,updated_at:'2026-11-12T12:00:01Z'};
  const secondStore=memoryStore();
  const refused=createProductReviewHost312({
    openStore:async()=>secondStore,verifyAcceptedClaim:async(issue,role)=>(
      {ok:true,claim:p312Claim(role)}),
    verifyFrozenWriteSet:async()=>({ok:true,
      taskPackBlobSha:p312Pack,writeSetSha256:p312Set}),
    readCurrentVersion:async()=>({ok:true,sha:p312Source}),
    octokit:p312OfficialGitTransport(async()=>({data:edited})),
    withVersionFence:async(meta,fn)=>fn(),
    env:{TALLY_CLAIM_SOURCE_SHA:p312Source,
      TALLY_TASK_CLAIM_312_REGISTRY_PATH:'fixture.sqlite'}});
  await refused.persistDraft(p312Input);
  const denied=await refused.freezeReviewedDraft(p312FreezeArgs);
  assert.equal(denied.state,'UNKNOWN');
  assert.equal(denied.reason,'PRODUCT_REVIEW_WITNESS_UNVERIFIED');
  assert.equal(await secondStore.get(PRODUCT312_FREEZE_KEY+p312Source),null);
});

test('#312 Host refuses malformed draft shapes before any persistence',async()=>{
  const {provider}=p312Make();
  assert.equal((await provider.persistDraft({...p312Input,
    draft:{product:{},l2:{},provenance:[]}})).reason,
    'PRODUCT_DRAFT_INPUT_INVALID');
  assert.equal((await provider.persistDraft({...p312Input,
    draft:{...p312Draft,product:{goal:'',nonGoals:[]}}})).reason,
    'PRODUCT_DRAFT_INPUT_INVALID');
  assert.equal((await provider.persistDraft({...p312Input,
    draft:{...p312Draft,extra:'field'}})).reason,
    'PRODUCT_DRAFT_INPUT_INVALID');
});


// --- #312 physical selected-K lane: real DH v0.10.3 kernel + sealed Business
// + tally-local-codex shim + Host provider, T312_PRODUCT_REVIEW=1 routes. ---
// Runs ONLY with T312_REPLAY=1 and DH_V0103=<pinned DH v0.10.3 checkout>.
// The fixture provider substitutes the sealed transport with the REAL
// host-project.mjs Host functions, the REAL GitHub C witness validator, a
// verbatim mirror of the sealed provider's claim/Pack/FSW gate and the shared
// version-dag effect fence, over REAL SQLite CAS files shared by separate
// processes. The REAL sealed provider bytes are exercised by the no-claim
// BLOCKED case. No real Claim is forged: every positive uses clearly-marked
// fixture rows in fixture registries; production registries are untouched.
const T312_DH_SHA='061376f4a023070f215f9feffad5c24d96d503cb';
const T312_PROVIDER=String.raw`import {createProductReviewHost312,PRODUCT312_FREEZE_KEY}
  from '__TALLY_ROOT__experiments/v2/project-intake/host-project.mjs';
import {readProductReviewWitnessFromClient}
  from '__TALLY_ROOT__packages/tally-github-c/src/project-intake.mjs';
import {createHash,randomUUID} from 'node:crypto';
import {readFileSync} from 'node:fs';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const OWNER='kaicreator-mm',REPO='tally';
const SHA=/^[a-f0-9]{40}$/;
const issueRefFor=n=>'https://github.com/'+OWNER+'/'+REPO+'/issues/'+n;
const claimKeyFor=(n,role)=>OWNER+'/'+REPO+'#'+n+':'+role+':__default__';
const registryKeyFor=k=>'tally:v2:claim-admission:'+
  createHash('sha256').update(k).digest('hex');
const fixture=()=>JSON.parse(readFileSync(process.env.T312_FIXTURE,'utf8'));
const claimEnvFor=(env,role)=>{const p='TALLY_TASK_CLAIM_312_'+role.toUpperCase()+'_';
  return {sourceSha:env.TALLY_CLAIM_SOURCE_SHA,dispatchId:env[p+'DISPATCH_ID'],
    operatorId:env[p+'OPERATOR_ID'],operatorKind:env[p+'OPERATOR_KIND'],
    writerId:env[p+'WRITER_ID'],writerLogin:env[p+'WRITER_LOGIN'],
    commentId:Number(env[p+'COMMENT_ID']),registryPath:env[p+'REGISTRY_PATH']};};
// Mirror of the sealed provider's #312 claim gate: durable CAS row bound to
// the exact Task Pack/FSW plus a bounded fixture-native publication witness.
async function verifyAcceptedClaim312(issueNumber,actorRole){
  const env=process.env,claim=claimEnvFor(env,actorRole);
  if(issueNumber!==312||!SHA.test(claim.sourceSha??'')||!claim.dispatchId||
    !claim.operatorId||!claim.operatorKind||!claim.writerId||!claim.writerLogin||
    !Number.isSafeInteger(claim.commentId)||!claim.registryPath)
    return {ok:false,reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH'};
  let registry,row;
  try{registry=await openSqliteCasStore({path:claim.registryPath,busyTimeoutMs:1000});
    row=await registry.get(registryKeyFor(claimKeyFor(312,actorRole)));
  }catch{return {ok:false,reason:'CLAIM_REGISTRY_UNAVAILABLE'};}
  finally{if(registry)await registry.close();}
  if(!row)return {ok:false,reason:'CLAIM_REGISTRY_RECORD_NOT_FOUND'};
  const v=row.value;
  if(v?.phase!=='PUBLISHED_VERIFIED'||v.generation!==1||
    v.protectedClaimKey!==claimKeyFor(312,actorRole)||
    v.repository!==OWNER+'/'+REPO||v.issueRef!==issueRefFor(312)||
    v.sourceSha!==claim.sourceSha||v.actorRole!==actorRole||
    v.dispatchId!==claim.dispatchId||v.operatorId!==claim.operatorId||
    v.operatorKind!==claim.operatorKind||v.writerId!==claim.writerId||
    v.commentId!==claim.commentId||
    v.commentRef!==v.issueRef+'#issuecomment-'+v.commentId)
    return {ok:false,reason:'CLAIM_REGISTRY_RESERVATION_MISMATCH'};
  const pack=env.TALLY_TASK_CLAIM_312_TASK_PACK_BLOB_SHA;
  const writeSet=env.TALLY_TASK_CLAIM_312_WRITESET_SHA256;
  if(!SHA.test(pack??'')||!/^[0-9a-f]{64}$/.test(writeSet??'')||
    claim.registryPath!==env.TALLY_TASK_CLAIM_312_REGISTRY_PATH||
    v.taskPackBlobSha!==pack||v.writeSetSha256!==writeSet)
    return {ok:false,reason:'PRODUCT_REVIEW_ROLE_CLAIM_PACK_UNVERIFIED'};
  const witness=fixture().claimWitness[actorRole];
  if(witness?.login!==claim.writerLogin||typeof witness?.body!=='string'||
    !witness.body.includes(claimKeyFor(312,actorRole))||
    !witness.body.includes(claim.dispatchId)||
    !witness.body.includes(claim.sourceSha))
    return {ok:false,reason:'NATIVE_CLAIM_WITNESS_NOT_VERIFIED'};
  return {ok:true,claim:{accepted:true,issueNumber:312,role:actorRole,
    sourceSha:claim.sourceSha,taskPackBlobSha:pack,writeSetSha256:writeSet,
    claimRef:v.commentRef,operatorId:claim.operatorId,writerId:claim.writerId,
    transportActor:claim.writerLogin}};
}
async function verifyFrozenWriteSet312(issueNumber,sourceSha){
  const env=process.env;
  const registryPath=env.TALLY_TASK_CLAIM_312_REGISTRY_PATH;
  const pack=env.TALLY_TASK_CLAIM_312_TASK_PACK_BLOB_SHA;
  const digest=env.TALLY_TASK_CLAIM_312_WRITESET_SHA256;
  if(issueNumber!==312||!SHA.test(sourceSha??'')||!SHA.test(pack??'')||
    !/^[0-9a-f]{64}$/.test(digest??'')||typeof registryPath!=='string'||!registryPath)
    return {ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'};
  let db,row;
  try{db=await openSqliteCasStore({path:registryPath,busyTimeoutMs:1000});
    row=await db.get('tally:v2:task-writeset:312');}
  catch{return {ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'};}
  finally{if(db)await db.close();}
  const value=row?.value;
  if(value?.phase!=='FROZEN_DAG_WRITESET'||value.sourceSha!==sourceSha||
    value.taskPackBlobSha!==pack||value.writeSetSha256!==digest||
    typeof value.document!=='string'||value.document.length>65536||
    !value.document.includes('experiments/v2/project-intake/host-project.mjs')||
    !value.document.includes('web/v2/intake-controller.mjs'))
    return {ok:false,reason:'DAG_WRITESET_NOT_VERIFIED'};
  return {ok:true,taskPackBlobSha:pack,writeSetSha256:digest};
}
// Verbatim mirror of the sealed provider's shared version-dag effect fence:
// same key/schema/registry/env contract and HELD/RELEASED semantics.
const fenceKey='tally:v2:version-dag-effect-fence:'+
  createHash('sha256').update(OWNER+'/'+REPO+'#version/v2').digest('hex');
const unknown=reason=>({state:'UNKNOWN',reason,issueNumber:312,frozen:false,
  authorized:false,admitted:false,taskPass:false,releasePass:false});
const same=(a,b)=>JSON.stringify(a)===JSON.stringify(b);
const safeNoEffect=new Set(['PRODUCT_REVIEW_WITNESS_UNVERIFIED',
  'PRODUCT_REVIEW_SOURCE_MOVED','PRODUCT_REVIEW_FREEZE_CONFLICT']);
async function withFence({sourceSha,operation},fn){
  const env=process.env;
  if(env.TALLY_LOCAL_REPLAY==='1'||sourceSha!==env.TALLY_V2_VERSION_SHA||
    sourceSha!==env.TALLY_CLAIM_SOURCE_SHA||
    typeof env.TALLY_VERSION_DAG_REGISTRY_PATH!=='string'||!env.TALLY_VERSION_DAG_REGISTRY_PATH||
    typeof env.TALLY_VERSION_DAG_WRITER_ID!=='string'||!env.TALLY_VERSION_DAG_WRITER_ID)
    return unknown('PRODUCT_REVIEW_SHARED_HOST_FENCE_REQUIRED');
  let db,reservedRevision=null;
  try{
    db=await openSqliteCasStore({path:env.TALLY_VERSION_DAG_REGISTRY_PATH,busyTimeoutMs:1000});
    const before=await db.get(fenceKey);
    if(before){const v=before.value;
      if(!Number.isSafeInteger(before.revision)||
        v?.schema!=='tally/v2/version-dag-effect-fence/v1'||
        v.repository!==OWNER+'/'+REPO||v.versionId!=='version/v2'||
        v.sourceSha!==sourceSha||v.phase!=='RELEASED')
        return unknown('PRODUCT_REVIEW_SHARED_HOST_FENCE_REQUIRED');}
    const held={schema:'tally/v2/version-dag-effect-fence/v1',phase:'HELD',
      repository:OWNER+'/'+REPO,versionId:'version/v2',sourceSha,
      writerId:env.TALLY_VERSION_DAG_WRITER_ID,operation,nonce:randomUUID()};
    const reservation=await db.put(fenceKey,held,
      before?{mustMatchRevision:before.revision}:{mustBeAbsent:true});
    if(!Number.isSafeInteger(reservation?.revision))
      return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');
    reservedRevision=reservation.revision;
    const proof=await db.get(fenceKey);
    if(proof?.revision!==reservedRevision||!same(proof.value,held))
      return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');
    let result;
    try{result=await fn();}
    catch{return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');}
    if(result?.state!=='FROZEN_COMPLETE'&&
      !(result?.state==='UNKNOWN'&&safeNoEffect.has(result.reason)))
      return result??unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');
    const released={...held,phase:'RELEASED',
      closedStatus:result.state==='FROZEN_COMPLETE'?'FROZEN':result.reason};
    const ack=await db.put(fenceKey,released,{mustMatchRevision:reservedRevision});
    const final=await db.get(fenceKey);
    if(!Number.isSafeInteger(ack?.revision)||final?.revision!==ack.revision||
      !same(final.value,released))
      return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');
    return result;
  }catch{return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');}
  finally{if(db)await db.close().catch(()=>{});}
}
// Test seams for the cross-process fence case (fixture provider only).
export async function __acquireHeldFence(){
  const env=process.env;
  let db;
  try{db=await openSqliteCasStore({path:env.TALLY_VERSION_DAG_REGISTRY_PATH,
      busyTimeoutMs:1000});
    const before=await db.get(fenceKey);
    if(before)return {held:false,reason:'FENCE_NOT_VACANT'};
    const held={schema:'tally/v2/version-dag-effect-fence/v1',phase:'HELD',
      repository:OWNER+'/'+REPO,versionId:'version/v2',
      sourceSha:env.TALLY_CLAIM_SOURCE_SHA,
      writerId:env.TALLY_VERSION_DAG_WRITER_ID,operation:'T312_TEST_HOLD',
      nonce:randomUUID()};
    await db.put(fenceKey,held,{mustBeAbsent:true});
    return {held:true};
  }finally{if(db)await db.close().catch(()=>{});}
}
export async function __readFence(){
  let db;
  try{db=await openSqliteCasStore({path:process.env.TALLY_VERSION_DAG_REGISTRY_PATH,
      busyTimeoutMs:1000});
    const row=await db.get(fenceKey);
    return row?{phase:row.value?.phase??null,sourceSha:row.value?.sourceSha??null}:null;
  }finally{if(db)await db.close().catch(()=>{});}
}
const fixtureOctokit=()=>{const state=fixture();
  const body=state.witness?.comment?.body?
    JSON.parse(state.witness.comment.body):null;
  return {rest:{
    repos:{
      getBranch:async()=>({data:{commit:{sha:state.branchSha}}}),
      getCommit:async({ref})=>({data:{sha:ref,commit:{tree:{sha:
        body&&ref===body.reviewedHead?body.reviewedTree:'b'.repeat(40)}}}})},
    pulls:{get:async({pull_number})=>({data:body&&
      pull_number===body.reviewedPullNumber?
      {number:pull_number,state:'open',merged:false,
        head:{sha:body.reviewedHead,repo:{full_name:'kaicreator-mm/tally'}},
        base:{sha:body.reviewedBase,ref:'version/v2',
          repo:{full_name:'kaicreator-mm/tally'}}}:null})},
    issues:{getComment:async({comment_id})=>({data:
      comment_id===state.witness?.id?state.witness.comment:null})}}};};
function host312(){
  const openStore=async options=>{
    const store=await openSqliteCasStore(options);
    if(process.env.T312_FAULT==='ack_loss'){
      const put=store.put.bind(store);
      store.put=async (key,value,condition)=>{
        const result=await put(key,value,condition);
        if(String(key).startsWith(PRODUCT312_FREEZE_KEY))
          throw Error('T312 fixture: ACK lost after server committed');
        return result;};
    }
    return store;};
  return createProductReviewHost312({env:process.env,octokit:fixtureOctokit(),
    openStore,verifyAcceptedClaim:verifyAcceptedClaim312,
    verifyFrozenWriteSet:verifyFrozenWriteSet312,
    readCurrentVersion:async()=>({ok:true,sha:fixture().branchSha}),
    withVersionFence:withFence});}
const refused312=(input,freeze)=>
  input?.issueNumber!==312||input.sourceRef!==issueRefFor(312)?
    unknown(freeze?'PRODUCT_REVIEW_FREEZE_INPUT_INVALID':'PRODUCT_DRAFT_INPUT_INVALID'):null;
export async function persistProductDraft312(input){
  const refused=refused312(input,false);
  if(refused)return refused;
  try{return await host312().persistDraft(input);}
  catch{return unknown('PRODUCT_DRAFT_CAS_UNKNOWN');}
}
export async function freezeProductReview312(input){
  const refused=refused312(input,true);
  if(refused)return refused;
  try{return await host312().freezeReviewedDraft(input);}
  catch{return unknown('PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');}
}
// The sealed shim statically imports every provider operation; the lanes this
// fixture does not exercise get honest NOT_RUN stubs (never called here).
const notRun=reason=>({state:'NOT_RUN',reason,authorized:false,admitted:false,
  taskPass:false,releasePass:false});
export const runLocalCodexAttempt=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const joinReviewedCandidates=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const publishReviewedCandidate=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const hostProjectIntake=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const freezeProductIntake=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const publishMaintenanceIssue=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const recordTaskLearningOutcome=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const hostReleaseDelivery=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const addNativeTaskDependency=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
export const freezeVersionTaskDag=async()=>notRun('T312_FIXTURE_LANE_NOT_ENABLED');
`;

// Provider-direct driver: seeds fixture CAS rows, drives the fixture
// provider outside the K for reason-fidelity negatives, and reads back the
// physical SQLite state (registry rows, fence phase). Runs in the
// consumer-fixture install dir with per-case env.
const T312_DRIVER=String.raw`import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync} from 'node:fs';
const [,,command,arg]=process.argv;
const env=process.env;
const sha256=x=>createHash('sha256').update(x).digest('hex');
const {openSqliteCasStore}=await import('@kaicreator/domain-harness-host-node');
const input=()=>JSON.parse(readFileSync(env.T312_INPUT,'utf8'));
const readFixture=()=>JSON.parse(readFileSync(env.T312_FIXTURE,'utf8'));
const writeFixture=f=>writeFileSync(env.T312_FIXTURE,JSON.stringify(f));
const reg=async()=>openSqliteCasStore({path:env.TALLY_TASK_CLAIM_312_REGISTRY_PATH,
  busyTimeoutMs:1000});
if(command==='seed'){
  const data=input();
  const db=await reg();
  const roles={builder:data.builder,reviewer:data.reviewer};
  for(const [role,c] of Object.entries(roles)){
    if(!c)continue;
    const claimKey='kaicreator-mm/tally#312:'+role+':__default__';
    const unbound=arg==='no-pack-binding'&&role==='reviewer';
    const value={phase:'PUBLISHED_VERIFIED',generation:1,protectedClaimKey:claimKey,
      repository:'kaicreator-mm/tally',
      issueRef:'https://github.com/kaicreator-mm/tally/issues/312',
      sourceSha:data.sourceSha,actorRole:role,dispatchId:c.dispatchId,
      operatorId:c.operatorId,operatorKind:c.operatorKind,writerId:c.writerId,
      commentId:c.commentId,
      commentRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+c.commentId,
      ...(unbound?{}:{taskPackBlobSha:data.taskPackBlobSha,
        writeSetSha256:data.writeSetSha256})};
    await db.put('tally:v2:claim-admission:'+sha256(claimKey),value,
      {mustBeAbsent:true});
  }
  const fsw={phase:'FROZEN_DAG_WRITESET',
    sourceSha:arg==='fsw-stale'?'9'.repeat(40):data.sourceSha,
    taskPackBlobSha:data.taskPackBlobSha,writeSetSha256:data.writeSetSha256,
    document:data.fswDocument};
  await db.put('tally:v2:task-writeset:312',fsw,{mustBeAbsent:true});
  await db.close();
  writeFixture({branchSha:data.sourceSha,
    claimWitness:Object.fromEntries(Object.entries(roles)
      .filter(([,c])=>c).map(([role,c])=>[role,{login:c.writerLogin,
        body:JSON.stringify({claimKey:'kaicreator-mm/tally#312:'+role+':__default__',
          dispatchId:c.dispatchId,sourceSha:data.sourceSha,actorRole:role})}])),
    witness:null});
  console.log('T312_DRIVER_SEED=PASS'+(arg?' variant='+arg:''));
}
else if(command==='persist'){
  const provider=await import('@kaicreator/tally-local-codex-host-bundle');
  const data=input();
  const out=await provider.persistProductDraft312({issueNumber:312,
    sourceRef:'https://github.com/kaicreator-mm/tally/issues/312',
    sourceSha:data.sourceSha,taskPackBlobSha:data.taskPackBlobSha,
    writeSetSha256:data.writeSetSha256,draft:data.draft});
  console.log('T312_DRIVER_PERSIST='+JSON.stringify(out));
  if(arg==='bind-witness'&&out.state==='DRAFT_PENDING_REVIEW'){
    const f=readFixture();
    const ref='https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+
      data.reviewCommentId;
    const body={schema:'tally.v2.product-review-witness/2',
      status:'INDEPENDENTLY_VERIFIED',repository:'kaicreator-mm/tally',
      issueNumber:312,reviewedPullNumber:401,draftRef:out.draftRef,
      productDigest:out.productDigest,sourceSha:data.sourceSha,
      taskPackBlobSha:data.taskPackBlobSha,writeSetSha256:data.writeSetSha256,
      reviewedHead:data.reviewedHead,reviewedTree:data.reviewedTree,
      reviewedBase:data.reviewedBase,
      builderClaimRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+
        data.builder.commentId,
      reviewerClaimRef:'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+
        data.reviewer.commentId,
      reviewerOperatorId:data.reviewer.operatorId,
      reviewerWriterId:data.reviewer.writerId,p0:0,p1:0,p2:1};
    f.witness={id:data.reviewCommentId,comment:{id:data.reviewCommentId,
      html_url:ref,issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/312',
      user:{login:data.reviewer.writerLogin},
      created_at:'2026-11-13T08:00:00Z',
      updated_at:env.T312_EDIT_WITNESS==='1'?'2026-11-13T08:00:01Z':'2026-11-13T08:00:00Z',
      body:JSON.stringify(body)}};
    writeFixture(f);
    console.log('T312_DRIVER_WITNESS_BOUND='+out.productDigest);
  }
}
else if(command==='freeze'){
  const provider=await import('@kaicreator/tally-local-codex-host-bundle');
  const data=input();
  const out=await provider.freezeProductReview312({issueNumber:312,
    sourceRef:'https://github.com/kaicreator-mm/tally/issues/312',
    sourceSha:data.sourceSha,taskPackBlobSha:data.taskPackBlobSha,
    writeSetSha256:data.writeSetSha256,reviewCommentId:data.reviewCommentId,
    reviewedHead:data.reviewedHead,reviewedTree:data.reviewedTree,
    reviewedBase:data.reviewedBase});
  console.log('T312_DRIVER_FREEZE='+JSON.stringify(out));
}
else if(command==='readreg'){
  const db=await reg();
  const draft=await db.get('tally:v2:product-intake:312:draft:'+input().sourceSha);
  const frozen=await db.get('tally:v2:product-intake:312:frozen:'+input().sourceSha);
  await db.close();
  console.log('T312_DRIVER_REG='+JSON.stringify({
    draftPhase:draft?.value?.phase??null,draftDigest:draft?.value?.productDigest??null,
    frozenPhase:frozen?.value?.phase??null,frozenRevision:frozen?.revision??null,
    frozenReviewRef:frozen?.value?.reviewRef??null}));
}
else if(command==='holdfence'){
  const provider=await import('@kaicreator/tally-local-codex-host-bundle');
  const out=await provider.__acquireHeldFence();
  console.log('T312_DRIVER_HOLD='+JSON.stringify(out));
}
else if(command==='readfence'){
  const provider=await import('@kaicreator/tally-local-codex-host-bundle');
  const out=await provider.__readFence();
  console.log('T312_DRIVER_FENCE='+JSON.stringify(out));
}
else throw Error('T312_DRIVER_UNKNOWN_COMMAND '+command);
`;
// Real-K consumer: loads the sealed app through the public Microkernel and
// sends the two #312 intents. Modes: positive | noclaim | held | blocked-real.
const T312_CONSUMER=String.raw`import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sha=x=>createHash('sha256').update(x).digest('hex');
const sealed=JSON.parse(readFileSync(process.env.T312_SEALED_PATH,'utf8'));
const capability=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-local-codex@v1.0.0');
assert.ok(capability&&capability.moduleSource,'tally-local-codex capability required');
assert.equal(sha(capability.moduleSource),capability.moduleSha256);
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(capability.moduleSource);
assert.ok(digestMatch,'provider digest marker required');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
assert.equal(sha(readFileSync(providerPath)),digestMatch[1],'PROVIDER_INTEGRITY_MISMATCH');
const mode=process.env.T312_CASE;
const input=JSON.parse(readFileSync(process.env.T312_INPUT,'utf8'));
const deployment={deployedAppId:'deploy:tally-v2-t312-replay',tenantId:'tenant:tally-v2',
  appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>!!identity&&identity.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c&&c.kind==='create'?{mustBeAbsent:true}:
  c&&c.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw new Error('INVALID_K_CONDITION');})();
const dbPath=process.env.T312_CONSUMER_TMP+'/kernel-'+mode+'.sqlite';
const openFacade=async()=>{
  const sqlite=await openSqliteCasStore({path:dbPath,busyTimeoutMs:1000});
  const facade=await load(sealed,{store:{
    get:key=>sqlite.get(key),
    put:(key,value,cond)=>sqlite.put(key,value,condition(cond)),
    remove:(key,cond)=>sqlite.remove(key,condition(cond))},
    deploymentIdentity:deployment,deploymentRoot:process.cwd(),
    tenantId:deployment.tenantId,queryAuthority:authority,
    now:()=>Date.UTC(2026,10,13)});
  return {sqlite,facade};};
const send=(facade,intentType,payload,messageId)=>facade.send({intentType,
  instanceKey:'intake-312',messageId,input:payload});
const freezePayload=()=>({sourceSha:input.sourceSha,
  taskPackBlobSha:input.taskPackBlobSha,writeSetSha256:input.writeSetSha256,
  reviewCommentId:input.reviewCommentId,reviewedHead:input.reviewedHead,
  reviewedTree:input.reviewedTree,reviewedBase:input.reviewedBase});
if(mode!=='positive'){
  const {facade,sqlite}=await openFacade();
  const receipt=await send(facade,'freezeProductReview312',freezePayload(),
    't312-'+mode+'-1');
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,700));
  const effect=receipt.admitted.effects[0];
  assert.equal(effect.disposition,'executed');
  const output=effect.output;
  assert.equal(output.state,'UNKNOWN',JSON.stringify(output).slice(0,300));
  assert.equal(output.frozen,false);
  assert.equal(output.authorized,false);assert.equal(output.taskPass,false);
  console.log('T312_K_BLOCKED_'+mode.toUpperCase().replace(/-/g,'_')+'=PASS reason='+
    output.reason);
  facade.stop();await sqlite.close();process.exit(0);
}
const {facade,sqlite}=await openFacade();
const persistReceipt=await send(facade,'persistProductDraft312',
  {sourceSha:input.sourceSha,taskPackBlobSha:input.taskPackBlobSha,
    writeSetSha256:input.writeSetSha256,draft:input.draft},'t312-persist-1');
assert.equal(persistReceipt.status,'admitted',
  JSON.stringify(persistReceipt).slice(0,900));
const persistEffect=persistReceipt.admitted.effects[0];
assert.equal(persistEffect.disposition,'executed');
const persistOut=persistEffect.output;
assert.equal(persistOut.state,'DRAFT_PENDING_REVIEW',
  JSON.stringify(persistOut).slice(0,400));
assert.match(persistOut.draftRef,/^draft:sha256:[0-9a-f]{64}$/);
assert.equal(persistOut.taskPackBlobSha,input.taskPackBlobSha);
assert.equal(persistOut.writeSetSha256,input.writeSetSha256);
const reg=await openSqliteCasStore({
  path:process.env.TALLY_TASK_CLAIM_312_REGISTRY_PATH,busyTimeoutMs:1000});
const frozenKey='tally:v2:product-intake:312:frozen:'+input.sourceSha;
const freezeReceipt=await send(facade,'freezeProductReview312',freezePayload(),
  't312-freeze-1');
assert.equal(freezeReceipt.status,'admitted',
  JSON.stringify(freezeReceipt).slice(0,900));
const freezeOut=freezeReceipt.admitted.effects[0].output;
assert.equal(freezeOut.state,'FROZEN_COMPLETE',JSON.stringify(freezeOut).slice(0,400));
assert.equal(freezeOut.frozen,true);
assert.equal(freezeOut.receiptRef,'receipt:sha256:'+freezeOut.productDigest);
assert.equal(freezeOut.draftRef,persistOut.draftRef);
assert.equal(freezeOut.reviewRef,
  'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+input.reviewCommentId);
const rev1=(await reg.get(frozenKey))?.revision;
const replay=await send(facade,'freezeProductReview312',freezePayload(),'t312-freeze-1');
assert.equal(replay.idempotentReplay,true,'same-message resend replays the journal');
const second=await send(facade,'freezeProductReview312',freezePayload(),'t312-freeze-2');
if(second.admitted&&second.admitted.effects.length)
  assert.equal(second.admitted.effects[0].output.state,'FROZEN_COMPLETE');
const rev2=(await reg.get(frozenKey))?.revision;
assert.equal(rev2,rev1,'resends must never write a second freeze CAS create');
const draftRow=await reg.get('tally:v2:product-intake:312:draft:'+input.sourceSha);
assert.equal(draftRow?.value?.phase,'DRAFT_PENDING_REVIEW');
await reg.close();
console.log('T312_K_POSITIVE=PASS '+JSON.stringify({draftRef:persistOut.draftRef,
  receiptRef:freezeOut.receiptRef,reviewRef:freezeOut.reviewRef,
  frozenRevision:rev2,taskPass:freezeOut.taskPass,authorized:freezeOut.authorized}));
facade.stop();await sqlite.close();process.exit(0);
`;
test('T312 selected K physical replay: two intents, shared SQLite fence, fail-closed negatives',{
  skip:process.env.T312_REPLAY!=='1'||!process.env.DH_V0103?
    'set T312_REPLAY=1 and DH_V0103=<pinned DH v0.10.3 checkout> to run the physical replay':false,
},async()=>{
  const dh=process.env.DH_V0103;
  assert.equal(run('git',['-C',dh,'rev-parse','HEAD']).stdout.trim(),T312_DH_SHA,
    'DH_V0103 must be the exact v0.10.3 checkout pinned by the T06M/T06N lanes');
  if(!existsSync(join(dh,'tooling','build','dist','index.js'))){
    const build=run('npm',['run','build','-w','@kaicreator/domain-harness-contracts',
      '-w','@kaicreator/domain-harness-build','-w','@kaicreator/domain-harness-microkernel',
      '-w','@kaicreator/domain-harness-host-node'],{cwd:dh,timeout:600000});
    assert.equal(build.status,0,build.stderr.slice(-2000));
  }
  const root=mkdtempSync(join(tmpdir(),'t312-replay-'));
  const npmRegistry={npm_config_registry:'https://registry.npmmirror.com'};
  try{
    const packed=join(root,'packed');mkdirSync(packed);
    for(const name of ['@kaicreator/domain-harness-contracts','@kaicreator/domain-harness-build',
      '@kaicreator/domain-harness-microkernel','@kaicreator/domain-harness-host-node'])
      assert.equal(run('npm',['pack','--pack-destination',packed,'-w',name],{cwd:dh}).status,0,name);
    for(const dir of ['./domain-packages/kernel/basic','./domain-packages/standard/basic'])
      assert.equal(run('npm',['pack','--pack-destination',packed,dir],{cwd:dh}).status,0,dir);
    const tarballs=()=>readdirSync(packed);
    const tgz=prefix=>{const f=tarballs().find(x=>x.startsWith(prefix));
      assert.ok(f,'missing tarball '+prefix);return join(packed,f);};
    // Producer install: DH deps + esbuild for the real provider bundle build.
    const producerDir=join(root,'producer');mkdirSync(producerDir);
    writeFileSync(join(producerDir,'package.json'),JSON.stringify(
      {name:'tally-t312-producer',private:true,type:'module'}));
    const producerInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund',tgz('kaicreator-domain-harness-build-'),tgz('kaicreator-domain-harness-contracts-'),
      tgz('kaicreator-domain-harness-microkernel-'),tgz('kaicreator-domain-harness-host-node-'),
      tgz('kaicreator-domain-kernel-basic-'),tgz('kaicreator-domain-standard-basic-')],
      {cwd:producerDir,env:{...process.env,...npmRegistry},timeout:300000});
    assert.equal(producerInstall.status,0,producerInstall.stderr.slice(-2000));
    const esbuildInstall=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
      '--no-fund','esbuild@0.25.12'],
      {cwd:producerDir,env:{...process.env,...npmRegistry},timeout:300000});
    assert.equal(esbuildInstall.status,0,esbuildInstall.stderr.slice(-2000));
    for(const name of ['producer.mjs','readiness-capability.mjs','local-codex-shim.mjs',
      'bundle-local-codex.mjs'])
      copyFileSync(join(REPO_ROOT,'experiments','v2','selected-business-build',name),
        join(producerDir,name));
    const businessSource=join(REPO_ROOT,'packages','tally-engineering-business','src','index.mjs');
    // Real sealed provider bytes (bundle-local-codex template, no substitution).
    const realProvider=join(root,'real-provider');mkdirSync(realProvider);
    const bundleRun=run(process.execPath,[join(producerDir,'bundle-local-codex.mjs')],
      {cwd:producerDir,env:{...process.env,...npmRegistry,
        TALLY:REPO_ROOT,T06M_C_OUTPUT:join(realProvider,'index.mjs'),
        T06M_IMPORTS_OUTPUT:join(realProvider,'imports.json')},timeout:300000});
    assert.equal(bundleRun.status,0,bundleRun.stderr.slice(-2000));
    writeFileSync(join(realProvider,'package.json'),JSON.stringify(
      {name:'@kaicreator/tally-local-codex-host-bundle',version:'0.0.0',private:true,
        type:'module',exports:'./index.mjs'}));
    // Fixture provider: REAL Host + REAL GitHub C + verbatim fence/claim-gate
    // mirror over a fixture transport; __TALLY_ROOT__ becomes a file:// URL.
    const providerDir=join(root,'provider');mkdirSync(providerDir);
    writeFileSync(join(providerDir,'package.json'),JSON.stringify(
      {name:'@kaicreator/tally-local-codex-host-bundle',version:'0.0.0',private:true,
        type:'module',exports:'./index.mjs'}));
    writeFileSync(join(providerDir,'index.mjs'),
      T312_PROVIDER.replaceAll('__TALLY_ROOT__','file:///'+REPO_ROOT.replace(/\\/g,'/')));
    for(const dir of [realProvider,providerDir])
      assert.equal(run('npm',['pack','--pack-destination',packed,dir],
        {cwd:dir,env:{...process.env,...npmRegistry}}).status,0,'provider pack '+dir);
    const sealApp=async(providerFile,outName)=>{
      const out=join(root,outName);
      const produced=run(process.execPath,[join(producerDir,'producer.mjs')],
        {cwd:producerDir,env:{...process.env,...npmRegistry,
          TALLY_BUSINESS_SOURCE:businessSource,
          T02B_SEALED_APP_OUTPUT:out,T06M_LOCAL:'1',T06M_PROVIDER_FILE:providerFile,
          T312_PRODUCT_REVIEW:'1'},timeout:300000});
      assert.equal(produced.status,0,
        produced.stderr.slice(-2500)+produced.stdout.slice(-2500));
      return out;};
    const sealedFixture=await sealApp(join(providerDir,'index.mjs'),'sealed-fixture.json');
    const sealedReal=await sealApp(join(realProvider,'index.mjs'),'sealed-real.json');
    // Route manifest: both intents route to the single selected K capability.
    for(const sealedPath of [sealedFixture,sealedReal]){
      const sealed=JSON.parse(readFileSync(sealedPath,'utf8'));
      for(const intent of ['persistProductDraft312','freezeProductReview312']){
        const route=sealed.routes.find(r=>r.intentId===intent);
        assert.ok(route,'route missing: '+intent);
        assert.equal(route.capabilityId,'tally-local-codex');
        assert.equal(route.operationId,'tally.'+intent);
      }
      const capability=sealed.modules.find(m=>m.packageId==='capability-tally-local-codex@v1.0.0');
      assert.ok(capability?.provides?.includes('intent:persistProductDraft312')&&
        capability?.provides?.includes('intent:freezeProductReview312'),
        'capability provides must list both #312 intents');
    }
    // Consumer installs: fixture + real providers with the DH kernel packages.
    const consumerFixture=join(root,'consumer-fixture');
    const consumerReal=join(root,'consumer-real');
    const installConsumer=async(dir,providerTarPrefix)=>{
      mkdirSync(dir);
      writeFileSync(join(dir,'package.json'),JSON.stringify(
        {name:'tally-t312-consumer',private:true,type:'module'}));
      const install=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
        '--no-fund',tgz('kaicreator-domain-harness-contracts-'),
        tgz('kaicreator-domain-harness-microkernel-'),
        tgz('kaicreator-domain-harness-host-node-'),tgz(providerTarPrefix)],
        {cwd:dir,env:{...process.env,...npmRegistry},timeout:600000});
      assert.equal(install.status,0,install.stderr.slice(-2000));
      const native=run('npm',['install','--save-exact','--no-audit','--no-fund',
        'better-sqlite3@12.4.1'],
        {cwd:dir,env:{...process.env,...npmRegistry},timeout:600000});
      assert.equal(native.status,0,native.stderr.slice(-2000));
      writeFileSync(join(dir,'driver-312.mjs'),T312_DRIVER);
      writeFileSync(join(dir,'consumer-312.mjs'),T312_CONSUMER);};
    await installConsumer(consumerFixture,'kaicreator-tally-local-codex-host-bundle-');
    await installConsumer(consumerReal,'kaicreator-tally-local-codex-host-bundle-');
    // Both providers pack to the same tarball name; the second pack (fixture)
    // overwrote the real one in packed/. Re-pack the real provider separately
    // and pin the real consumer to its own bytes.
    {
      const realPacked=join(root,'packed-real');mkdirSync(realPacked);
      assert.equal(run('npm',['pack','--pack-destination',realPacked,realProvider],
        {cwd:realProvider,env:{...process.env,...npmRegistry}}).status,0,'real repack');
      const realTgz=join(realPacked,readdirSync(realPacked)[0]);
      const swap=run('npm',['install','--save-exact','--ignore-scripts','--no-audit',
        '--no-fund',realTgz],
        {cwd:consumerReal,env:{...process.env,...npmRegistry},timeout:300000});
      assert.equal(swap.status,0,swap.stderr.slice(-2000));
    }
    // Physical case scaffolding.
    const sourceSha=run('git',['-C',REPO_ROOT,'rev-parse','HEAD']).stdout.trim();
    assert.match(sourceSha,/^[0-9a-f]{40}$/);
    const pack='b'.repeat(40),set='c'.repeat(64),witnessId=77003;
    const builder={dispatchId:'dispatch:312:builder:phys',operatorId:'zcode:t312:builder',
      operatorKind:'other',writerId:'writer:312:builder',writerLogin:'kaicreator-mm',
      commentId:77001};
    const reviewer={dispatchId:'dispatch:312:reviewer:phys',operatorId:'zcode:t312:reviewer',
      operatorKind:'other',writerId:'writer:312:reviewer',writerLogin:'kaicreator-mm',
      commentId:77002};
    const draft={product:{goal:'T312 physical Product ReviewWitness replay',
      nonGoals:['No forged Task PASS','No fake Release']},l2:{decision:'KEEP_MERGED_L2_ARCHITECTURE'},
      provenance:['157','158']};
    const fswDocument=['.woodpecker/v2-alm-workspace.yaml',
      'experiments/v2/project-intake/host-project.mjs',
      'experiments/v2/project-intake/host-project.test.mjs',
      'packages/tally-engineering-business/src/index.mjs',
      'packages/tally-github-c/src/project-intake.mjs',
      'web/v2/intake-controller.mjs','web/v2/journey-server.mjs'].join('\n');
    const caseInput=extra=>({sourceSha,taskPackBlobSha:pack,writeSetSha256:set,
      reviewCommentId:witnessId,reviewedHead:'d'.repeat(40),reviewedTree:'e'.repeat(40),
      reviewedBase:sourceSha,draft,builder,reviewer,fswDocument,...extra});
    const mkCase=(name,input)=>{
      const dir=join(root,'data-'+name);mkdirSync(dir);
      writeFileSync(join(dir,'t312-input.json'),JSON.stringify(input));
      return dir;};
    const caseEnv=(dir,extra={})=>({...process.env,
      TALLY_TASK_CLAIM_312_REGISTRY_PATH:join(dir,'registry.sqlite'),
      TALLY_VERSION_DAG_REGISTRY_PATH:join(dir,'fence.sqlite'),
      TALLY_VERSION_DAG_WRITER_ID:'writer:t312-physical',
      TALLY_V2_VERSION_SHA:sourceSha,TALLY_CLAIM_SOURCE_SHA:sourceSha,
      TALLY_TASK_CLAIM_312_TASK_PACK_BLOB_SHA:pack,
      TALLY_TASK_CLAIM_312_WRITESET_SHA256:set,
      TALLY_TASK_CLAIM_312_BUILDER_DISPATCH_ID:builder.dispatchId,
      TALLY_TASK_CLAIM_312_BUILDER_OPERATOR_ID:builder.operatorId,
      TALLY_TASK_CLAIM_312_BUILDER_OPERATOR_KIND:builder.operatorKind,
      TALLY_TASK_CLAIM_312_BUILDER_WRITER_ID:builder.writerId,
      TALLY_TASK_CLAIM_312_BUILDER_WRITER_LOGIN:builder.writerLogin,
      TALLY_TASK_CLAIM_312_BUILDER_COMMENT_ID:String(builder.commentId),
      TALLY_TASK_CLAIM_312_BUILDER_REGISTRY_PATH:join(dir,'registry.sqlite'),
      TALLY_TASK_CLAIM_312_REVIEWER_DISPATCH_ID:reviewer.dispatchId,
      TALLY_TASK_CLAIM_312_REVIEWER_OPERATOR_ID:reviewer.operatorId,
      TALLY_TASK_CLAIM_312_REVIEWER_OPERATOR_KIND:reviewer.operatorKind,
      TALLY_TASK_CLAIM_312_REVIEWER_WRITER_ID:reviewer.writerId,
      TALLY_TASK_CLAIM_312_REVIEWER_WRITER_LOGIN:reviewer.writerLogin,
      TALLY_TASK_CLAIM_312_REVIEWER_COMMENT_ID:String(reviewer.commentId),
      TALLY_TASK_CLAIM_312_REVIEWER_REGISTRY_PATH:join(dir,'registry.sqlite'),
      T312_FIXTURE:join(dir,'fixture.json'),T312_INPUT:join(dir,'t312-input.json'),
      T312_CONSUMER_TMP:dir,TALLY_GITHUB_READ_TOKEN:'fixture-token-1234567890',
      ...extra});
    const driver=(dir,args,extra={})=>{
      const r=run(process.execPath,[join(consumerFixture,'driver-312.mjs'),...args],
        {cwd:consumerFixture,env:caseEnv(dir,extra),timeout:120000});
      if(r.status!==0)throw new Error('driver '+args.join(' ')+' exit '+r.status+
        ' STDOUT-tail:'+r.stdout.slice(-1500)+' STDERR-tail:'+r.stderr.slice(-1500));
      return r;};
    const consumer=(dir,mode,sealedPath,installDir=consumerFixture)=>{
      const r=run(process.execPath,[join(installDir,'consumer-312.mjs')],
        {cwd:installDir,env:caseEnv(dir,{T312_CASE:mode,T312_SEALED_PATH:sealedPath}),
          timeout:180000});
      if(r.status!==0)throw new Error('consumer '+mode+' exit '+r.status+
        ' STDOUT-tail:'+r.stdout.slice(-1500)+' STDERR-tail:'+r.stderr.slice(-1500));
      return r;};
    const jsonLine=(text,prefix)=>{
      const line=text.split('\n').find(l=>l.startsWith(prefix));
      assert.ok(line,'missing marker '+prefix+' in:\n'+text.slice(-1200));
      return JSON.parse(line.slice(prefix.length));};

    // K1 positive: seed -> direct persist binds the witness -> real K freeze.
    const posDir=mkCase('positive',caseInput());
    assert.equal(driver(posDir,['seed']).status,0,'seed positive');
    const persistDirect=driver(posDir,['persist','bind-witness']);
    assert.equal(persistDirect.status,0,persistDirect.stderr.slice(-800));
    const persisted=jsonLine(persistDirect.stdout,'T312_DRIVER_PERSIST=');
    assert.equal(persisted.state,'DRAFT_PENDING_REVIEW');
    const kPositive=consumer(posDir,'positive',sealedFixture);
    assert.equal(kPositive.status,0,kPositive.stderr.slice(-3000)+kPositive.stdout.slice(-800));
    assert.match(kPositive.stdout,/T312_K_POSITIVE=PASS/);
    const kPos=jsonLine(kPositive.stdout,'T312_K_POSITIVE=PASS ');
    assert.equal(kPos.draftRef,persisted.draftRef);
    assert.equal(kPos.taskPass,false);assert.equal(kPos.authorized,false);
    const posReg=jsonLine(driver(posDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(posReg.draftPhase,'DRAFT_PENDING_REVIEW');
    assert.equal(posReg.frozenPhase,'FROZEN_PRODUCT_INTAKE');
    assert.equal(posReg.frozenRevision,kPos.frozenRevision);
    assert.equal(posReg.frozenReviewRef,
      'https://github.com/kaicreator-mm/tally/issues/312#issuecomment-'+witnessId);
    const posFence=jsonLine(driver(posDir,['readfence']).stdout,'T312_DRIVER_FENCE=');
    assert.equal(posFence.phase,'RELEASED');

    // K2 noclaim: reviewer Claim row absent; builder-only seed.
    const nocDir=mkCase('noclaim',caseInput({reviewer:null}));
    assert.equal(driver(nocDir,['seed']).status,0,'seed noclaim builder-only');
    assert.equal(driver(nocDir,['persist']).status,0,'persist noclaim builder-only');
    const kNoc=consumer(nocDir,'noclaim',sealedFixture);
    assert.equal(kNoc.status,0,kNoc.stderr.slice(-3000)+kNoc.stdout.slice(-800));
    assert.match(kNoc.stdout,/T312_K_BLOCKED_NOCLAIM=PASS/);
    const nocReg=jsonLine(driver(nocDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(nocReg.frozenPhase,null,'no freeze without the reviewer Claim');
    assert.equal(nocReg.draftPhase,'DRAFT_PENDING_REVIEW');

    // K3 held fence: process A holds; the K freeze in process B is refused.
    const heldDir=mkCase('held',caseInput());
    assert.equal(driver(heldDir,['seed']).status,0,'seed held');
    assert.equal(driver(heldDir,['persist','bind-witness']).status,0,'persist held');
    const hold=jsonLine(driver(heldDir,['holdfence']).stdout,'T312_DRIVER_HOLD=');
    assert.equal(hold.held,true,'process A must acquire the shared fence');
    const kHeld=consumer(heldDir,'held',sealedFixture);
    assert.equal(kHeld.status,0,kHeld.stderr.slice(-3000)+kHeld.stdout.slice(-800));
    assert.match(kHeld.stdout,/T312_K_BLOCKED_HELD=PASS/);
    const heldFenceAfter=jsonLine(driver(heldDir,['readfence']).stdout,'T312_DRIVER_FENCE=');
    assert.equal(heldFenceAfter.phase,'HELD','cross-process HELD must survive');
    const heldReg=jsonLine(driver(heldDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(heldReg.frozenPhase,null,'no freeze under a foreign HELD fence');

    // K4 blocked-real: REAL sealed provider bytes, no lawful Claims anywhere.
    const realDir=mkCase('blocked-real',caseInput());
    const kReal=consumer(realDir,'blocked-real',sealedReal,consumerReal);
    assert.equal(kReal.status,0,kReal.stderr.slice(-3000)+kReal.stdout.slice(-800));
    assert.match(kReal.stdout,/T312_K_BLOCKED_BLOCKED_REAL=PASS/);
    assert.match(kReal.stdout,/reason=PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN/);

    // D1 edited witness (provider-direct, precise Host reason).
    const editDir=mkCase('edited',caseInput());
    assert.equal(driver(editDir,['seed']).status,0,'seed edited');
    assert.equal(driver(editDir,['persist','bind-witness'],{T312_EDIT_WITNESS:'1'}).status,0,
      'persist edited');
    const editedFreeze=driver(editDir,['freeze']);
    const edited=jsonLine(editedFreeze.stdout,'T312_DRIVER_FREEZE=');
    assert.equal(edited.state,'UNKNOWN');
    assert.equal(edited.reason,'PRODUCT_REVIEW_WITNESS_UNVERIFIED');
    const editedFence=jsonLine(driver(editDir,['readfence']).stdout,'T312_DRIVER_FENCE=');
    assert.equal(editedFence,null,'witness refusal predates the fence');
    const editedReg=jsonLine(driver(editDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(editedReg.frozenPhase,null);

    // D2 source drift: live version/v2 moved away from the claimed source.
    // The witness is bound first so its preflight passes and the refusal is
    // attributed to the moved branch, not the missing witness.
    const driftDir=mkCase('drift',caseInput());
    assert.equal(driver(driftDir,['seed']).status,0,'seed drift');
    assert.equal(driver(driftDir,['persist','bind-witness']).status,0,'persist drift');
    {
      const fixturePath=join(driftDir,'fixture.json');
      const fixtureState=JSON.parse(readFileSync(fixturePath,'utf8'));
      fixtureState.branchSha='9'.repeat(40);
      writeFileSync(fixturePath,JSON.stringify(fixtureState));
    }
    const drifted=jsonLine(driver(driftDir,['freeze']).stdout,'T312_DRIVER_FREEZE=');
    assert.equal(drifted.state,'UNKNOWN');
    assert.equal(drifted.reason,'PRODUCT_REVIEW_SOURCE_MOVED');
    const driftReg=jsonLine(driver(driftDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(driftReg.frozenPhase,null);

    // D3 ACK loss after the freeze CAS commit: UNKNOWN, fence stays HELD and
    // a retry cannot bypass it (no expiry, no reseed).
    const ackDir=mkCase('ackloss',caseInput());
    assert.equal(driver(ackDir,['seed']).status,0,'seed ackloss');
    assert.equal(driver(ackDir,['persist','bind-witness']).status,0,'persist ackloss');
    const ackLossRun=driver(ackDir,['freeze'],{T312_FAULT:'ack_loss'});
    const ackLoss=jsonLine(ackLossRun.stdout,'T312_DRIVER_FREEZE=');
    const ackReg=jsonLine(driver(ackDir,['readreg']).stdout,'T312_DRIVER_REG=');
    const ackFence=jsonLine(driver(ackDir,['readfence']).stdout,'T312_DRIVER_FENCE=');
    const ackRetry=jsonLine(driver(ackDir,['freeze']).stdout,'T312_DRIVER_FREEZE=');
    const ackFenceAfter=jsonLine(driver(ackDir,['readfence']).stdout,'T312_DRIVER_FENCE=');
    assert.equal(ackLoss.state,'UNKNOWN');
    assert.equal(ackLoss.reason,'PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN');
    assert.equal(ackReg.frozenPhase,'FROZEN_PRODUCT_INTAKE',
      'the committed-but-unacknowledged freeze record stays for reconciliation');
    assert.equal(ackFence.phase,'HELD','ambiguous effect keeps the fence HELD');
    assert.equal(ackRetry.state,'UNKNOWN');
    assert.equal(ackRetry.reason,'PRODUCT_REVIEW_SHARED_HOST_FENCE_REQUIRED');
    assert.equal(ackFenceAfter.phase,'HELD','the retry must not release the fence');

    // D4 reviewer Claim without exact Pack/FSW binding: builder-only persist
    // still qualifies (its own Claim is Pack-bound), the two-role freeze
    // refuses the unbound reviewer identity.
    const packDir=mkCase('no-pack',caseInput());
    assert.equal(driver(packDir,['seed','no-pack-binding']).status,0,'seed no-pack');
    const packPersist=jsonLine(driver(packDir,['persist']).stdout,'T312_DRIVER_PERSIST=');
    assert.equal(packPersist.state,'DRAFT_PENDING_REVIEW');
    const packFreeze=jsonLine(driver(packDir,['freeze']).stdout,'T312_DRIVER_FREEZE=');
    assert.equal(packFreeze.state,'UNKNOWN');
    assert.equal(packFreeze.reason,'PRODUCT_REVIEW_PROCESS_OR_CLAIM_UNVERIFIED');
    const packReg=jsonLine(driver(packDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(packReg.frozenPhase,null);
    assert.equal(packReg.draftPhase,'DRAFT_PENDING_REVIEW');

    // D5 frozen write-set row stale against the exact source.
    const fswDir=mkCase('fsw-stale',caseInput());
    assert.equal(driver(fswDir,['seed','fsw-stale']).status,0,'seed fsw-stale');
    const fswPersist=jsonLine(driver(fswDir,['persist']).stdout,'T312_DRIVER_PERSIST=');
    assert.equal(fswPersist.state,'UNKNOWN');
    assert.equal(fswPersist.reason,'PRODUCT_DRAFT_SOURCE_OR_CLAIM_STALE');
    const fswReg=jsonLine(driver(fswDir,['readreg']).stdout,'T312_DRIVER_REG=');
    assert.equal(fswReg.draftPhase,null);assert.equal(fswReg.frozenPhase,null);

    console.log('T312_PHYSICAL_REPLAY_EVIDENCE='+JSON.stringify({
      dh:T312_DH_SHA,sourceSha,routes:'SELECTED_T312',provider:'FIXTURE_TRANSPORT_REAL_HOST_C',
      realProviderBlocked:'FROZEN_ABSENT_UNKNOWN',positive:'FROZEN_COMPLETE',
      idempotentResend:true,crossProcessHeldFence:true,editedWitness:'UNVERIFIED',
      sourceDrift:'SOURCE_MOVED',ackLoss:'HELD_RETRY_REFUSED',noPackBinding:'UNVERIFIED',
      staleFsw:'REFUSED',physicalGithubEffect:'NOT_RUN',authorized:false,taskPass:false,
      releasePass:false}));
  }finally{rmSync(root,{recursive:true,force:true});}
});

test('#312 Host accepts K-kernel data-only (null-prototype) draft payloads',async()=>{
  // The selected K kernel projects nested effect payload values as canonical
  // data-only clones with null prototypes; the draft gate must accept them.
  const {provider}=p312Make();
  const nullProtoDraft=JSON.parse(JSON.stringify(p312Draft),function(k,v){
    return v&&typeof v==='object'&&!Array.isArray(v)?Object.assign(Object.create(null),v):v;});
  assert.notEqual(Object.getPrototypeOf(nullProtoDraft),Object.prototype);
  assert.equal(Object.getPrototypeOf(nullProtoDraft),null);
  const written=await provider.persistDraft({...p312Input,draft:nullProtoDraft});
  assert.equal(written.state,'DRAFT_PENDING_REVIEW');
  assert.match(written.draftRef,/^draft:sha256:[a-f0-9]{64}$/);
});
