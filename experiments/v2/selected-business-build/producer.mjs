/**
 * T02b — REAL installed public DomainHarness BUILD selection of Tally B.
 *
 * Producer-only: no private DomainHarness imports, no caller authority, no
 * simulated Kernel acceptance. Consumer runtime proof is separate T02b work.
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { V10_TARGET_PROFILE, sha256Hex } from '@kaicreator/domain-harness-contracts';
import { buildV10App } from '@kaicreator/domain-harness-build';

const businessFile = process.env.TALLY_BUSINESS_SOURCE;
if (!businessFile) throw new Error('TALLY_BUSINESS_SOURCE_REQUIRED');
const business = await import(pathToFileURL(businessFile).href);
assert.equal(business.PACKAGE_KIND, 'business');
assert.equal(business.ABI_VERSION, 'v08-business-abi/1');
assert.deepEqual([...business.INTENT_TYPES], ['inspectTaskReadiness']);
assert.equal(business.getPolicy().packageId, business.MODULE_ID);
const bBytes = readFileSync(businessFile, 'utf8');
const capabilityFile = new URL('./readiness-capability.mjs', import.meta.url);
const cBytes = readFileSync(capabilityFile, 'utf8');

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
    bytes:bBytes,provides:['business:policy/1','intent:inspectTaskReadiness']}),
  {...selected({packageId:'capability-tally-source-observation@v2.0.0',
    role:'capability',abiVersion:'v08-capability-abi/1',endpoint:'capability-endpoint',
    bytes:cBytes,provides:['capability:tally-readiness/1']}),
    capabilityId:'tally-source-observation',
    handlers:[{operationId:'tally.observeReadiness',handler:'observeReadiness'}],
    externalEffect:false},
];
const options={appId:'app:tally-v2-readiness',audience:'tally-engineering',
  buildId:'tally-v2-t02b-source-proof',targetProfile:V10_TARGET_PROFILE,
  modules,routes:[{intentId:'inspectTaskReadiness', operationId:'tally.observeReadiness',
    capabilityId:'tally-source-observation',handlerId:'observeReadiness'}],
  hardBudget:{maxStepsPerTurn:8,maxWorkUnitsPerTurn:64}};
const built=buildV10App(options);
assert.equal(built.modules.length,4);
assert.equal(built.modules.filter(x=>x.role==='kernel-primary').length,1);
assert.equal(built.modules.filter(x=>x.role==='business').length,1);
assert.equal(built.modules.filter(x=>x.role==='capability').length,1);
assert.equal(built.routes[0].intentId,'inspectTaskReadiness');
assert.equal(built.modules.find(x=>x.role==='business').moduleSha256,sha256Hex(bBytes));
assert.ok(built.subjectDigest);
assert.equal(built.targetProfile,V10_TARGET_PROFILE);
// BUILD must not admit a duplicate K or forged Business role/ABI.
const negatives=[];
for(const [caseName,mutate] of [
 ['duplicate-k',opts=>({...opts,modules:[...modules,modules[0]]})],
 ['invalid-b-abi',opts=>({...opts,modules:modules.map(m=>m.role==='business'?{...m,abiVersion:'invalid'}:m)})],
 ['forged-b-digest',opts=>({...opts,modules:modules.map(m=>m.role==='business'?{...m,moduleSha256:'0'.repeat(64)}:m)})],
 ['wrong-route-handler',opts=>({...opts,routes:[{...opts.routes[0],handlerId:'unselectedPrivateHandler'}]})],
]){
  assert.throws(()=>buildV10App(mutate(options)),undefined,caseName);
  negatives.push(caseName);
}
console.log(JSON.stringify({
  selectedBusiness:business.MODULE_ID,policyIntent:business.INTENT_TYPES[0],
  profile:built.targetProfile,subjectDigest:built.subjectDigest,
  loadOrder:built.loadOrder,selectedModules:built.modules.map(m=>({
    packageId:m.packageId,role:m.role,moduleSha256:m.moduleSha256})),
  negativeRefusals:negatives,producerOnly:true,realKernelSend:'NOT_RUN',
  githubEffect:'NOT_RUN',taskPass:false,releasePass:false
},null,2));
console.log('T02B_PUBLIC_BUILD_SELECTED_B=PASS');
