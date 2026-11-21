/** R157 #513 regression — the TasksView filter controls (search text, status,
 * severity, blocked-only) were bound to state but the row projections never
 * consumed filterTasks, so every control was dead UI. RED first (2026-11-21):
 * rendered against the pre-fix TasksView these assertions fail on rows that
 * ignore the active filters; after the wiring fix they are the GREEN set.
 * Same view-render convention as features/settings/input-aria.test.ts:
 * bundle once with the pinned esbuild toolchain, render through
 * react-dom/server, skip visibly without a toolchain checkout. */
import test from 'node:test';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdirSync,rmSync} from 'node:fs';
import {join,dirname,resolve} from 'node:path';
import {fileURLToPath,pathToFileURL} from 'node:url';

const viewToolchainAvailable=(()=>{
  const require=createRequire(import.meta.url);
  for(const pkg of ['esbuild','react','react-dom/server']){
    try{require.resolve(pkg);}catch{return false;}
  }
  return true;
})();
const viewSkipReason=viewToolchainAvailable?false:
  'needs esbuild+react toolchain (npm --prefix apps/tally-web install); dep-free source gate skips view render';

const pkgRoot=resolve(dirname(fileURLToPath(import.meta.url)),'../../..');
const bundleDir=join(pkgRoot,'node_modules','.cache','r157-tasks-filter-wiring');
const outfile=join(bundleDir,'tasks-views.bundle.mjs');

type Comp=React.FC<Record<string,unknown>>;
let viewsModule:Record<string,unknown>|null=null;
async function renderTasksView(props:Record<string,unknown>):Promise<string>{
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
    viewsModule=await import(pathToFileURL(outfile).href) as
      Record<string,unknown>;
  }
  const Component=viewsModule['TasksView'] as Comp;
  return reactDomServer.renderToStaticMarkup(
    react.createElement(Component,props));
}
const record={id:'196',versionId:'main',projectId:'kaicreator-mm/tally',
  title:'Mock Codex acceptance task',status:'open',severity:'P2',
  blockedBy:[] as string[],agentId:null,claimState:'ACCEPTED',attemptId:null,
  attemptState:null,prNumber:null,reviewState:'NOT_RUN',ciState:'NOT_RUN',
  writeSet:['MOCK_CODEX_CANDIDATE.md'],requirementRef:null,
  updatedAt:'2026-11-21T00:00:00.000Z'};
const baseProps={filters:{text:'',status:null,severity:null,agentId:null,
  onlyBlocked:false},setFilters:()=>{},onSelectTask:()=>{},
  selectedTaskId:null,versionId:'main',connected:true};

test('unfiltered list renders the task row',{skip:viewSkipReason},async()=>{
  const html=await renderTasksView(
    {...baseProps,tasks:[record]});
  assert.ok(html.includes('Mock Codex acceptance task'),'row renders');
});

test('search text with no match renders the honest empty state',
  {skip:viewSkipReason},async()=>{
    const html=await renderTasksView({...baseProps,tasks:[record],
      filters:{...baseProps.filters,text:'zzz-no-match'}});
    assert.ok(html.includes('没有匹配的任务'),
      'dead filter wiring would keep the row visible');
    assert.ok(!html.includes('Mock Codex acceptance task'));
  });

test('search matches title and write-set paths',{skip:viewSkipReason},async()=>{
  const hit=await renderTasksView({...baseProps,tasks:[record],
    filters:{...baseProps.filters,text:'MOCK_CODEX'}});
  assert.ok(hit.includes('Mock Codex acceptance task'),
    'write-set path search hits');
  const miss=await renderTasksView({...baseProps,tasks:[record],
    filters:{...baseProps.filters,text:'196'}});
  assert.ok(miss.includes('Mock Codex acceptance task'),'task id search hits');
});

test('severity and status filters apply',{skip:viewSkipReason},async()=>{
  const wrongSeverity=await renderTasksView({...baseProps,tasks:[record],
    filters:{...baseProps.filters,severity:'P0'}});
  assert.ok(wrongSeverity.includes('没有匹配的任务'));
  const wrongStatus=await renderTasksView({...baseProps,tasks:[record],
    filters:{...baseProps.filters,status:'done'}});
  assert.ok(wrongStatus.includes('没有匹配的任务'));
});

test('blocked-only hides a non-blocked task',{skip:viewSkipReason},async()=>{
  const html=await renderTasksView({...baseProps,tasks:[record],
    filters:{...baseProps.filters,onlyBlocked:true}});
  assert.ok(html.includes('没有匹配的任务'),
    'dead wiring kept the row visible with 只看阻塞 checked');
});

test('version isolation: rows of another version never surface',
  {skip:viewSkipReason},async()=>{
    const foreign={...record,id:'197',versionId:'version/other'};
    const html=await renderTasksView({...baseProps,tasks:[foreign]});
    assert.ok(html.includes('没有匹配的任务'),
      'activeVersionId must gate the visible rows');
  });
