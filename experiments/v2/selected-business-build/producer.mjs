/**
 * T02b — REAL installed public DomainHarness BUILD selection of Tally B.
 *
 * Producer-only: no private DomainHarness imports, no caller authority, no
 * simulated Kernel acceptance. Consumer runtime proof is separate T02b work.
 */
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import {createHash} from 'node:crypto';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { V10_TARGET_PROFILE, sha256Hex } from '@kaicreator/domain-harness-contracts';
import { buildV10App } from '@kaicreator/domain-harness-build';
import { V10BuildError } from '@kaicreator/domain-harness-contracts';

const businessFile = process.env.TALLY_BUSINESS_SOURCE;
if (!businessFile) throw new Error('TALLY_BUSINESS_SOURCE_REQUIRED');
const business = await import(pathToFileURL(businessFile).href);
assert.equal(business.PACKAGE_KIND, 'business');
assert.equal(business.ABI_VERSION, 'v08-business-abi/1');
assert.deepEqual([...business.INTENT_TYPES], ['inspectTaskReadiness','admitClaimedTask','runLocalCodexAttempt','joinReviewedCandidates']);
assert.equal(business.getPolicy().packageId, business.MODULE_ID);
const bBytes = readFileSync(businessFile, 'utf8');
const capabilityFile = process.env.T06F_NATIVE_PAIR === '1'
  ? fileURLToPath(new URL('./native-pair-shim.mjs',import.meta.url))
  : process.env.T04B_NATIVE === '1'
    ? fileURLToPath(new URL('./native-readonly-shim.mjs',import.meta.url))
    : fileURLToPath(new URL('./readiness-capability.mjs',import.meta.url));
let cBytes = readFileSync(capabilityFile, 'utf8');
if(process.env.T04B_NATIVE==='1'){
  const providerFile=process.env.T04B_PROVIDER_FILE;
  if(!providerFile)throw Error('T04B_PROVIDER_FILE_REQUIRED');
  const providerSha=createHash('sha256').update(readFileSync(providerFile)).digest('hex');
  if(!cBytes.includes('__PROVIDER_SHA256__'))throw Error('SEALED_PROVIDER_DIGEST_MARKER_MISSING');
  cBytes=cBytes.replace('__PROVIDER_SHA256__',providerSha);
  assert.match(cBytes,/NATIVE_PROVIDER_SHA256='[a-f0-9]{64}'/);
  // Published with the exact B/K/C BUILD digest; never merely an unpinned
  // npm package name. The Host checks installed bytes before any K load.
  console.log('T04B_PROVIDER_SHA256='+providerSha);
}
// T06l: second sealed capability for accepted-Claim execution admission.
// It composes with the existing readiness capability selection above; both
// capabilities are sealed into the same App when T06L_CLAIM=1.
let claimBytes=null;
if(process.env.T06L_CLAIM==='1'){
  const claimFile=fileURLToPath(new URL('./claim-admission-shim.mjs',import.meta.url));
  claimBytes=readFileSync(claimFile,'utf8');
  const claimProviderFile=process.env.T06L_PROVIDER_FILE;
  if(!claimProviderFile)throw Error('T06L_PROVIDER_FILE_REQUIRED');
  const claimProviderSha=createHash('sha256').update(readFileSync(claimProviderFile)).digest('hex');
  if(!claimBytes.includes('__PROVIDER_SHA256__'))throw Error('T06L_SEALED_PROVIDER_DIGEST_MARKER_MISSING');
  claimBytes=claimBytes.replace('__PROVIDER_SHA256__',claimProviderSha);
  assert.match(claimBytes,/NATIVE_PROVIDER_SHA256='[a-f0-9]{64}'/);
  console.log('T06L_PROVIDER_SHA256='+claimProviderSha);
}
let localCodexBytes=null;
if(process.env.T06M_LOCAL==='1'){
  const localFile=fileURLToPath(new URL('./local-codex-shim.mjs',import.meta.url));
  localCodexBytes=readFileSync(localFile,'utf8');
  const providerFile=process.env.T06M_PROVIDER_FILE;
  if(!providerFile)throw Error('T06M_PROVIDER_FILE_REQUIRED');
  const providerSha=createHash('sha256').update(readFileSync(providerFile)).digest('hex');
  if(!localCodexBytes.includes('__PROVIDER_SHA256__'))throw Error('T06M_SEALED_PROVIDER_DIGEST_MARKER_MISSING');
  localCodexBytes=localCodexBytes.replace('__PROVIDER_SHA256__',providerSha);
  assert.match(localCodexBytes,/NATIVE_PROVIDER_SHA256='[a-f0-9]{64}'/);
  console.log('T06M_PROVIDER_SHA256='+providerSha);
}

const payload = (specifier) => readFileSync(fileURLToPath(import.meta.resolve(specifier)), 'utf8');
const kBytes = payload('@kaicreator/domain-kernel-basic');
const sdkBytes = payload('@kaicreator/domain-standard-basic');
const routerBytes = payload('@kaicreator/domain-standard-basic/workflow-router');
// Upstream public v0.10.2 external-producer implementation strategy: compose
// installed S SDK and its public pure router, never a second Tally SDK.
const routerBody = routerBytes.replace(/^export /gm, '');
const sBytes = `${sdkBytes}
// Public pure Standard WorkflowRouter derived from installed selected S package.
const __router = (() => {
${routerBody}
return { routeWorkflowIntent };
})();
export const routeWorkflowIntent = __router.routeWorkflowIntent;
`;
function selected({packageId,role,abiVersion,endpoint,bytes,provides}) {
  return {packageId,role,abiVersion,endpoint,moduleSource:bytes,
    moduleSha256:sha256Hex(bytes),provides};
}
const modules = [
  selected({packageId:'v08-kernel-mechanism@v0.10.0',role:'kernel-primary',
    abiVersion:'v08-kernel-abi/1',endpoint:'occurrence-runtime',
    bytes:kBytes,provides:['kernel:occurrence-runtime/1']}),
  selected({packageId:'standard-sdk@v08.1.0.0',role:'standard-primary',
    abiVersion:'v08-standard-sdk-abi/1',endpoint:'rule-interpreter',
    bytes:sBytes,provides:['standard:interpret/1','standard:route/1']}),
  selected({packageId:business.MODULE_ID,role:'business',
    abiVersion:business.ABI_VERSION,endpoint:'business-policy',
    bytes:bBytes,provides:['business:policy/1','intent:inspectTaskReadiness',
      'intent:admitClaimedTask','intent:runLocalCodexAttempt','intent:joinReviewedCandidates']}),
  {...selected({packageId:'capability-tally-source-observation@v2.0.0',
    role:'capability',abiVersion:'v08-capability-abi/1',endpoint:'capability-endpoint',
    bytes:cBytes,provides:['capability:tally-readiness/1']}),
    capabilityId:'tally-source-observation',
    handlers:[{operationId:'tally.observeReadiness',handler:'observeReadiness'}],
    externalEffect:false,
    ...(process.env.T04B_NATIVE==='1'?{externalImports:{
      node:[],npm:['@kaicreator/tally-native-reader-bundle'],native:[]}}:{})},
  ...(claimBytes?[{...selected({packageId:'capability-tally-claim-admission@v1.0.0',
    role:'capability',abiVersion:'v08-capability-abi/1',endpoint:'capability-endpoint',
    bytes:claimBytes,provides:['capability:tally-claim-admission/1',
      'intent:admitClaimedTask']}),
    capabilityId:'tally-claim-admission',
    handlers:[{operationId:'tally.admitClaimedTask',handler:'admitClaimedTask'}],
    externalEffect:false,
    externalImports:{node:[],npm:['@kaicreator/tally-native-claim-reader-bundle'],
      native:['better-sqlite3']}}]:[]),
  ...(localCodexBytes?[{...selected({packageId:'capability-tally-local-codex@v1.0.0',
    role:'capability',abiVersion:'v08-capability-abi/1',endpoint:'capability-endpoint',
    bytes:localCodexBytes,provides:['capability:tally-local-codex/1',
      'intent:runLocalCodexAttempt',
      ...(process.env.T06N_CONCURRENT==='1'?['intent:joinReviewedCandidates']:[])]}),
    capabilityId:'tally-local-codex',
    handlers:[{operationId:'tally.runLocalCodexAttempt',handler:'runLocalCodexAttempt'},
      ...(process.env.T06N_CONCURRENT==='1'?
        [{operationId:'tally.joinReviewedCandidates',handler:'joinReviewedCandidates'}]:[])],
    externalEffect:true,
    externalImports:{node:[],npm:['@kaicreator/tally-local-codex-host-bundle'],
      native:['better-sqlite3']}}]:[]),
];
const routes=[
  {intentId:'inspectTaskReadiness', operationId:'tally.observeReadiness',
    capabilityId:'tally-source-observation',handlerId:'observeReadiness'},
  ...(claimBytes?[{intentId:'admitClaimedTask', operationId:'tally.admitClaimedTask',
    capabilityId:'tally-claim-admission',handlerId:'admitClaimedTask'}]:[]),
  ...(localCodexBytes?[{intentId:'runLocalCodexAttempt',operationId:'tally.runLocalCodexAttempt',
    capabilityId:'tally-local-codex',handlerId:'runLocalCodexAttempt'},
    ...(process.env.T06N_CONCURRENT==='1'?
      [{intentId:'joinReviewedCandidates',operationId:'tally.joinReviewedCandidates',
        capabilityId:'tally-local-codex',handlerId:'joinReviewedCandidates'}]:[])]:[]),
];
const options={appId:'app:tally-v2-readiness',audience:'tally-engineering',
  buildId:'tally-v2-t02b-source-proof',targetProfile:V10_TARGET_PROFILE,
  modules,routes,
  hardBudget:{maxStepsPerTurn:8,maxWorkUnitsPerTurn:64}};
const built=buildV10App(options);
const selectedCapabilityCount=1+(claimBytes?1:0)+(localCodexBytes?1:0);
assert.equal(built.modules.length,3+selectedCapabilityCount);
assert.equal(built.modules.filter(x=>x.role==='kernel-primary').length,1);
assert.equal(built.modules.filter(x=>x.role==='business').length,1);
assert.equal(built.modules.filter(x=>x.role==='capability').length,selectedCapabilityCount);
assert.equal(built.routes[0].intentId,'inspectTaskReadiness');
const selectedRouteCount=selectedCapabilityCount+
  (localCodexBytes&&process.env.T06N_CONCURRENT==='1'?1:0);
assert.equal(built.routes.length,selectedRouteCount);
if(claimBytes){
  assert.equal(built.routes[1].intentId,'admitClaimedTask');
  assert.equal(built.routes[1].capabilityId,'tally-claim-admission');
}
if(localCodexBytes){
  const route=built.routes.find(x=>x.intentId==='runLocalCodexAttempt');
  assert.equal(route.capabilityId,'tally-local-codex');
  assert.equal(route.handlerId,'runLocalCodexAttempt');
}
if(process.env.T06N_CONCURRENT==='1'){
  const joinRoute=built.routes.find(x=>x.intentId==='joinReviewedCandidates');
  assert.equal(joinRoute.capabilityId,'tally-local-codex');
  assert.equal(joinRoute.handlerId,'joinReviewedCandidates');
}else{
  assert.equal(built.routes.some(x=>x.intentId==='joinReviewedCandidates'),false,
    'T06n join route is selected only when T06N_CONCURRENT=1');
}
assert.equal(built.modules.find(x=>x.role==='business').moduleSha256,sha256Hex(bBytes));
assert.ok(built.subjectDigest);
if(process.env.T02B_SEALED_APP_OUTPUT) {
  // Sealed BUILD plain data, transferred as bytes to a separate consumer.
  writeFileSync(process.env.T02B_SEALED_APP_OUTPUT, JSON.stringify(built));
}
assert.equal(built.targetProfile,V10_TARGET_PROFILE);
// BUILD must not admit a duplicate K or forged Business role/ABI.
const negatives=[];
for(const [caseName,mutate,code] of [
 ['duplicate-k',opts=>({...opts,modules:[...modules,{...modules[0],packageId:'kernel-second@v2'}]}),'V10_ROLE_CARDINALITY'],
 ['invalid-b-abi',opts=>({...opts,modules:modules.map(m=>m.role==='business'?{...m,abiVersion:'invalid'}:m)}), 'V10_ABI_UNSUPPORTED'],
 ['forged-b-digest',opts=>({...opts,modules:modules.map(m=>m.role==='business'?{...m,moduleSha256:'0'.repeat(64)}:m)}), 'V10_DIGEST_MISMATCH'],
 ['wrong-route-handler',opts=>({...opts,routes:[{...opts.routes[0],handlerId:'unselectedPrivateHandler'}]}), 'V10_ROUTE_INVALID'],
  ...(claimBytes?[
   ['claim-wrong-route-handler',opts=>({...opts,routes:opts.routes.map(r=>r.intentId==='admitClaimedTask'?{...r,handlerId:'unselectedClaimHandler'}:r)}), 'V10_ROUTE_INVALID'],
   ['forged-claim-c-digest',opts=>({...opts,modules:modules.map(m=>m.capabilityId==='tally-claim-admission'?{...m,moduleSha256:'0'.repeat(64)}:m)}), 'V10_DIGEST_MISMATCH'],
   ['duplicate-claim-capability',opts=>({...opts,modules:[...modules,modules.find(m=>m.capabilityId==='tally-claim-admission')]}), 'BUILD_GRAPH_INVALID'],
  ]:[]),
  ...(localCodexBytes?[
   ['local-codex-wrong-route-handler',opts=>({...opts,routes:opts.routes.map(r=>r.intentId==='runLocalCodexAttempt'?{...r,handlerId:'unselectedLocalHandler'}:r)}), 'V10_ROUTE_INVALID'],
   ['forged-local-codex-c-digest',opts=>({...opts,modules:modules.map(m=>m.capabilityId==='tally-local-codex'?{...m,moduleSha256:'0'.repeat(64)}:m)}), 'V10_DIGEST_MISMATCH'],
   ['duplicate-local-codex-capability',opts=>({...opts,modules:[...modules,modules.find(m=>m.capabilityId==='tally-local-codex')]}), 'BUILD_GRAPH_INVALID'],
   ...(process.env.T06N_CONCURRENT==='1'?[
    ['join-wrong-route-handler',opts=>({...opts,routes:opts.routes.map(r=>r.intentId==='joinReviewedCandidates'?{...r,handlerId:'unselectedJoinHandler'}:r)}), 'V10_ROUTE_INVALID'],
   ]:[]),
  ]:[]),
]){
  assert.throws(()=>buildV10App(mutate(options)),e=>e instanceof V10BuildError && e.code===code,caseName);
  negatives.push(caseName);
}
console.log(JSON.stringify({
  selectedBusiness:business.MODULE_ID,policyIntents:[...business.INTENT_TYPES],
  profile:built.targetProfile,subjectDigest:built.subjectDigest,
  loadOrder:built.loadOrder,selectedModules:built.modules.map(m=>({
    packageId:m.packageId,role:m.role,moduleSha256:m.moduleSha256})),
  negativeRefusals:negatives,producerOnly:true,realKernelSend:'NOT_RUN',
  claimAdmissionCapability:claimBytes?'SELECTED':'NOT_SELECTED',
  localCodexCapability:localCodexBytes?'SELECTED_EXTERNAL_EFFECT':'NOT_SELECTED',
  concurrentJoinRoute:process.env.T06N_CONCURRENT==='1'?'SELECTED':'NOT_SELECTED',
  githubEffect:'NOT_RUN',taskPass:false,releasePass:false
},null,2));
console.log('T02B_PUBLIC_BUILD_SELECTED_B=PASS');
