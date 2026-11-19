// #449 U5 — TaskDrawer action-section honesty contract (audit finding X11/G4).
// RED first (2026-11-19): rendered against the pre-fix component this file
// fails on the missing catalog/projectId fallback (the "当前可用操作" section
// silently disappears — dead end) and on the bare UNKNOWN attempt code; after
// the fix every assertion below is the GREEN criterion set recorded in the
// #449 R104 admission (comment 7008437447).
// View-level render tests: Node cannot load .tsx directly, so the component is
// bundled once with the repo's pinned esbuild toolchain and rendered through
// react-dom/server; without a toolchain checkout the render tests skip visibly
// (same convention as the workbench view gate — the dep-free source gate runs
// before any npm install by design).
import test from 'node:test';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdirSync, rmSync} from 'node:fs';
import {join, dirname, resolve} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';

const viewToolchainAvailable=(()=>{
  const require=createRequire(import.meta.url);
  for(const pkg of ['esbuild','react','react-dom/server']){
    try{require.resolve(pkg);}catch{return false;}
  }
  return true;
})();
const viewSkipReason=viewToolchainAvailable?false:
  'needs esbuild+react toolchain (npm --prefix apps/tally-web install); dep-free source gate skips view render';

// The bundle must NOT inline react (same dual-dispatcher trap as the U3
// input-aria suite); writing it under the package's node_modules/.cache keeps
// bare imports bound to the same react copies the test itself uses.
const pkgRoot=resolve(dirname(fileURLToPath(import.meta.url)),'../../..');
const bundleDir=join(pkgRoot,'node_modules','.cache','i449-u5-task-drawer');
const outfile=join(bundleDir,'views.bundle.mjs');

type Comp=React.FC<Record<string,unknown>>;
let viewsModule:Record<string,unknown>|null=null;
async function renderDrawer(props:Record<string,unknown>):Promise<string>{
  const {buildSync}=await import('esbuild') as {buildSync:(options:{
    entryPoints:string[];outfile:string;bundle:boolean;format:'esm';
    platform:'node';jsx:'automatic';external:string[];logLevel:'silent'})=>unknown};
  const react=await import('react') as {createElement:(
    component:unknown,props:Record<string,unknown>)=>unknown};
  const reactDomServer=await import('react-dom/server') as
    {renderToStaticMarkup:(element:unknown)=>string};
  if(!viewsModule){
    mkdirSync(bundleDir,{recursive:true});
    process.on('exit',()=>{try{rmSync(bundleDir,{recursive:true,force:true})}
      catch{/* best effort cleanup */}});
    buildSync({entryPoints:[join(pkgRoot,'src','features','tasks','views.tsx')],
      outfile,bundle:true,format:'esm',platform:'node',jsx:'automatic',
      external:['react','react/jsx-runtime','react-dom'],logLevel:'silent'});
    viewsModule=await import(pathToFileURL(outfile).href) as Record<string,unknown>;
  }
  const Component=viewsModule['TaskDrawer'] as Comp;
  return reactDomServer.renderToStaticMarkup(react.createElement(Component,props));
}

/** Minimal honest TaskRecord fixture; attemptState 'UNKNOWN' mirrors an
 * un-reconciled read model (blockerSummary surfaces ATTEMPT_UNKNOWN). */
const makeTask=(overrides:Record<string,unknown>={})=>({
  id:'449',versionId:'version/v2',title:'UI 修复任务',status:'blocked' as const,
  severity:'P1' as const,blockedBy:[],agentId:null,
  claimState:'NOT_VERIFIED' as const,attemptId:null,attemptState:'UNKNOWN' as const,
  prNumber:null,reviewState:'NOT_RUN' as const,ciState:'NOT_RUN' as const,
  writeSet:[],requirementRef:null,updatedAt:'2026-11-19T00:00:00Z',
  ...overrides});
const validCatalog={connected:true,
  source:{repository:'kaicreator-mm/tally',sourceSha:'a'.repeat(40),
    currentness:'CURRENT',readback:'VERIFIED'},
  versionId:'version/v2',
  actions:[{id:'act-1',kind:'DISPATCH',revision:1,label:'派发构建',
    taskIssue:449,state:'ELIGIBLE'}]};

test('catalog/projectId absent renders an honest actions section, not silence',
  {skip:viewSkipReason},async()=>{
    // Current App wiring passes neither projectId nor actionCatalog — the
    // pre-fix drawer dropped the whole section (X11 dead end).
    const html=await renderDrawer({task:makeTask(),onClose(){}});
    assert.ok(html.includes('aria-label="任务可执行操作"'),
      'actions section keeps its accessible name');
    assert.ok(html.includes('操作清单不可用'),
      'fallback states the catalog is unavailable');
    assert.ok(html.includes('当前可用操作'),'section keeps its heading');
  });

test('fallback reason distinguishes missing catalog from missing project',
  {skip:viewSkipReason},async()=>{
    const noProject=await renderDrawer(
      {task:makeTask(),onClose(){},actionCatalog:validCatalog});
    assert.ok(noProject.includes('未指定项目'),
      'catalog present but projectId missing says so');
    assert.ok(!noProject.includes('操作清单不可用'),
      'does not claim catalog unavailable when it was provided');
  });

test('fallback exposes zero actionable buttons (no fabricated affordance)',
  {skip:viewSkipReason},async()=>{
    const html=await renderDrawer({task:makeTask(),onClose(){}});
    assert.ok(!html.includes('>继续<'),'no dispatch Continue button renders');
    assert.ok(!html.includes('可操作'),'no Eligible badge renders');
    assert.ok(!html.includes('未准入'),'no Not-admitted row renders');
  });

test('blocker summary stays model-authoritative in the drawer',
  {skip:viewSkipReason},async()=>{
    const html=await renderDrawer({task:makeTask(),onClose(){}});
    assert.ok(html.includes('CLAIM_NOT_ACCEPTED'),
      'claim blocker code preserved');
    assert.ok(html.includes('ATTEMPT_UNKNOWN'),
      'attempt blocker code preserved');
    assert.ok(html.includes('需要处理'),'blocker section heading preserved');
  });

test('qualified catalog renders the real action path without regression',
  {skip:viewSkipReason},async()=>{
    const html=await renderDrawer({task:makeTask(),onClose(){},
      projectId:'kaicreator-mm/tally',actionCatalog:validCatalog,
      onAction(){}});
    assert.ok(html.includes('当前可用操作'),'real path heading renders');
    assert.ok(html.includes('派发构建'),'admitted action label renders');
    assert.ok(html.includes('>继续<'),'Continue button renders for ELIGIBLE');
    assert.ok(!html.includes('操作清单不可用'),'fallback absent on real path');
  });

test('bare UNKNOWN attempt state gains a factual annotation, keeps the code',
  {skip:viewSkipReason},async()=>{
    const reported=await renderDrawer({task:makeTask(),onClose(){}});
    assert.ok(reported.includes('UNKNOWN'),'UNKNOWN literal preserved');
    assert.ok(reported.includes('执行状态未知,需 Host 对账'),
      'reported UNKNOWN keeps its factual annotation');
    const absent=await renderDrawer(
      {task:makeTask({attemptState:null}),onClose(){}});
    assert.ok(absent.includes('UNKNOWN'),'absent state still renders UNKNOWN');
    assert.ok(absent.includes('读模型未上报执行状态'),
      'absent state annotation names the missing report');
  });

test('English locale renders the same honesty contract',
  {skip:viewSkipReason},async()=>{
    const html=await renderDrawer(
      {task:makeTask(),onClose(){},locale:'en'});
    assert.ok(html.includes('aria-label="Task actions"'),
      'actions section accessible name in English');
    assert.ok(html.includes('Action catalog unavailable'),
      'fallback title in English');
  });
