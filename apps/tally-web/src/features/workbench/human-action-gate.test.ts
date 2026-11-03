import test from 'node:test';
import assert from 'node:assert/strict';
import {workbenchActionGate,workbenchActionControl,localizedActionStatus,
  ACTION_STATUS_MESSAGES,DISPATCH_NOT_CONNECTED_MESSAGE} from './human-action-gate.ts';
import type {WorkbenchSnapshot,WorkAction} from './model.ts';
import {buildSync} from 'esbuild';
import {createElement} from 'react';
import type {FunctionComponent} from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {pathToFileURL,fileURLToPath} from 'node:url';
const sha='a'.repeat(40),repository='org/tally';
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const work:WorkAction={id:'review-260',taskId:'260',projectId:repository,
  versionId:'version/v2',title:'Review candidate',kind:'REVIEW',state:'PENDING',
  ownerRole:'independent-reviewer',sourceRef:'https://github.com/org/tally/issues/260',
  reason:null,updatedAt:null,hostActionId:'review-task-260',
  hostActionRevision:1,hostActionKind:'REVIEW_REPAIR'};
const inbox:WorkbenchSnapshot={connected:true,source,actions:[work]};
const catalog={connected:true,source,versionId:'version/v2',actions:[{
  id:'review-task-260',kind:'REVIEW_REPAIR',revision:1,taskIssue:260,
  state:'ELIGIBLE' as const,label:'审核候选'}]};
test('only matching Host-qualified reviewer action can render enabled',()=>{
  const r=workbenchActionGate(inbox,catalog,work);
  assert.equal(r.enabled,true);
  assert.equal(r.actionId,'review-task-260');
  assert.equal(r.revision,1);
});
test('missing Host, unknown currentness and SHA mismatch stay disabled',()=>{
  for(const c of [null,{...catalog,connected:false},
    {...catalog,source:{...source,sourceSha:'b'.repeat(40)}},
    {...catalog,source:{...source,currentness:'STALE'}}]){
    assert.equal(workbenchActionGate(inbox,c,work).enabled,false);
  }
});
test('other project/version and blocked action cannot be clicked',()=>{
  assert.equal(workbenchActionGate(inbox,catalog,{...work,projectId:'evil/repo'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,versionId:'version/v3'},work).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,state:'BLOCKED'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,actions:[{
    ...catalog.actions[0],state:'BLOCKED' as const}]},work).enabled,false);
});
test('empty, missing or malformed project/version scope identifiers fail closed',()=>{
  assert.equal(workbenchActionGate({...inbox,
    source:{...source,repository:''}},catalog,work).enabled,false);
  assert.equal(workbenchActionGate({...inbox,
    source:{...source,repository:'org/'}},
    {...catalog,source:{...source,repository:'org/'}},
    {...work,projectId:'org/'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,
    {...catalog,source:{...source,repository:''}},
    {...work,projectId:''}).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,
    {...work,projectId:'org/tally/extra'}).enabled,false);

  const missingRepositorySnapshot={...inbox,
    source:{...source,repository:undefined}} as unknown as WorkbenchSnapshot;
  const missingRepositoryCatalog={...catalog,
    source:{...source,repository:undefined}} as unknown as typeof catalog;
  const missingProject={...work,projectId:undefined} as unknown as WorkAction;
  assert.equal(workbenchActionGate(missingRepositorySnapshot,
    missingRepositoryCatalog,missingProject).enabled,false);

  assert.equal(workbenchActionGate(inbox,
    {...catalog,versionId:''},{...work,versionId:''}).enabled,false);
  assert.equal(workbenchActionGate(inbox,
    {...catalog,versionId:'version/'},{...work,versionId:'version/'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,
    {...catalog,versionId:'version/v2/extra'},
    {...work,versionId:'version/v2/extra'}).enabled,false);
  const missingVersion={...work,versionId:undefined} as unknown as WorkAction;
  const missingVersionCatalog={...catalog,versionId:undefined}
    as unknown as typeof catalog;
  assert.equal(workbenchActionGate(inbox,
    missingVersionCatalog,missingVersion).enabled,false);
});

test('native blocker and unassigned approval never become fake permitted action',()=>{
  assert.equal(workbenchActionGate(inbox,catalog,{...work,kind:'BLOCKER'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,kind:'APPROVAL'}).enabled,false);
});
test('ambiguous Host action or invalid revision refuses action',()=>{
  assert.equal(workbenchActionGate(inbox,{...catalog,
    actions:[catalog.actions[0],catalog.actions[0]]},work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,
    actions:[{...catalog.actions[0],revision:0}]},work).enabled,false);
});

test('only exact Host-issued identity matching the work-item kind can enable',()=>{
  const exact={...work,hostActionId:'host-issued-1',
    hostActionRevision:7,hostActionKind:'REVIEW_REPAIR'};
  const c={...catalog,actions:[{id:'host-issued-1',kind:'REVIEW_REPAIR',
    revision:7,taskIssue:260,state:'ELIGIBLE' as const,label:'Review candidate'}]};
  const r=workbenchActionGate(inbox,c,exact);
  assert.equal(r.enabled,true);
  assert.equal(r.actionId,'host-issued-1');
  assert.equal(r.revision,7);
  assert.equal(workbenchActionGate(inbox,{...c,actions:[{
    ...c.actions[0],revision:8}]},exact).enabled,false);
  assert.equal(workbenchActionGate(inbox,c,{...exact,
    hostActionKind:'START_AGENT'}).enabled,false);
});
test('identity-free Task/category match is never a dispatch-ready operation',()=>{
  const identityFree={...work,hostActionId:null,hostActionRevision:null,
    hostActionKind:null};
  const result=workbenchActionGate(inbox,catalog,identityFree);
  assert.equal(result.enabled,false);
  assert.equal(result.reason,'HOST_ACTION_IDENTITY_NOT_BOUND');
  assert.equal(workbenchActionGate(inbox,catalog,{...work,
    hostActionRevision:undefined}).enabled,false);
  assert.equal(workbenchActionGate(inbox,catalog,{...work,
    hostActionId:'other-operation'}).enabled,false);
});
test('Recovery can request only its exact Host-issued START_AGENT action',()=>{
  const recovery={...work,kind:'RECOVERY' as const,
    hostActionId:'restart-260',hostActionRevision:3,
    hostActionKind:'START_AGENT'};
  assert.equal(workbenchActionGate(inbox,catalog,recovery).enabled,false);
  const c={...catalog,actions:[{id:'restart-260',kind:'START_AGENT',
    revision:3,taskIssue:260,state:'ELIGIBLE' as const,label:'Resume Builder'}]};
  assert.equal(workbenchActionGate(inbox,c,recovery).enabled,true);
  assert.equal(workbenchActionGate(inbox,c,{...recovery,
    hostActionKind:'REVIEW_REPAIR'}).enabled,false);
  assert.equal(workbenchActionGate(inbox,c,{...work,
    hostActionId:'restart-260',hostActionRevision:3,
    hostActionKind:'START_AGENT'}).enabled,false);
});

test('malformed source SHA and absent action catalog fail closed',()=>{
  assert.equal(workbenchActionGate({...inbox,source:{...source,sourceSha:'latest'}},
    catalog,work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,source:{...source,sourceSha:'latest'}},
    work).enabled,false);
  assert.equal(workbenchActionGate(inbox,{...catalog,actions:undefined},work).enabled,false);
});


test('unsafe numeric Task identifiers cannot alias a different Host issue number',()=>{
  const huge={...work,taskId:'9007199254740993',hostActionId:'aliased-action',
    hostActionRevision:1,hostActionKind:'REVIEW_REPAIR'};
  const c={...catalog,actions:[{id:'aliased-action',kind:'REVIEW_REPAIR',
    revision:1,taskIssue:9007199254740992,state:'ELIGIBLE' as const,label:'Review'}]};
  assert.equal(workbenchActionGate(inbox,c,huge).enabled,false);
});

test('fresh Host action without dispatch callback shows explicit not-connected state, never the positive label',()=>{
  const c=workbenchActionControl(inbox,catalog,work,false);
  assert.equal(c.disabled,true);
  assert.equal(c.dispatchReady,false);
  assert.equal(c.request,null);
  assert.notEqual(c.label,'审核候选');
  assert.ok(!c.label.includes('审核候选'));
  assert.equal(c.status,DISPATCH_NOT_CONNECTED_MESSAGE);
  assert.ok(!c.status.includes('FRESH_HOST_ACTION'));
});
test('fresh Host action with connected dispatch callback stays the positive dispatch-ready control',()=>{
  const c=workbenchActionControl(inbox,catalog,work,true);
  assert.equal(c.disabled,false);
  assert.equal(c.dispatchReady,true);
  assert.equal(c.label,'审核候选');
  assert.equal(c.status,null);
  assert.deepEqual(c.request,{repository:'org/tally',versionId:'version/v2',
    sourceSha:sha,actionId:'review-task-260',revision:1});
});
test('denial reasons are localized and never show raw internal codes',()=>{
  const disconnected=workbenchActionControl(inbox,{...catalog,connected:false},
    work,true);
  assert.equal(disconnected.disabled,true);
  assert.equal(disconnected.request,null);
  assert.equal(disconnected.status,
    ACTION_STATUS_MESSAGES.HOST_ACTION_SOURCE_NOT_CONNECTED);
  assert.ok(!disconnected.status.includes('HOST_ACTION_SOURCE_NOT_CONNECTED'));
  const unbound=workbenchActionControl(inbox,catalog,{...work,
    hostActionId:null,hostActionRevision:null,hostActionKind:null},true);
  assert.equal(unbound.disabled,true);
  assert.equal(unbound.status,ACTION_STATUS_MESSAGES.HOST_ACTION_IDENTITY_NOT_BOUND);
  assert.ok(!unbound.status.includes('HOST_ACTION_IDENTITY_NOT_BOUND'));
  for(const [code,message] of Object.entries(ACTION_STATUS_MESSAGES)){
    assert.ok(message.trim().length>0);
    assert.ok(!message.includes(code));
  }
  const fallback=localizedActionStatus('FUTURE_UNKNOWN_CODE');
  assert.ok(fallback.trim().length>0);
  assert.ok(!fallback.includes('FUTURE_UNKNOWN_CODE'));
});

// View-level render tests: Node cannot load .tsx directly, so the real view is
// bundled once with the repo's pinned esbuild toolchain and rendered through
// react-dom/server; assertions cover the exact disabled/positive/localized
// display contract for the Workbench action button.
let viewsModule:Record<string,unknown>|null=null;
async function loadViews():Promise<Record<string,unknown>>{
  if(!viewsModule){
    const dir=mkdtempSync(join(tmpdir(),'tally-workbench-view-'));
    process.on('exit',()=>{try{rmSync(dir,{recursive:true,force:true})}
      catch{/* best effort cleanup */}});
    const outfile=join(dir,'views.bundle.mjs');
    buildSync({entryPoints:[join(dirname(fileURLToPath(import.meta.url)),
      'views.tsx')],outfile,bundle:true,format:'esm',platform:'node',
      jsx:'automatic',logLevel:'silent'});
    viewsModule=await import(pathToFileURL(outfile).href) as Record<string,unknown>;
  }
  return viewsModule;
}
test('rendered action button shows dispatch-not-connected state, disabled, without positive label',async()=>{
  const {WorkbenchActionButton}=await loadViews() as
    {WorkbenchActionButton:FunctionComponent<Record<string,unknown>>};
  const html=renderToStaticMarkup(createElement(WorkbenchActionButton,{
    snapshot:inbox,catalog,action:work}));
  assert.match(html,/disabled/);
  assert.ok(html.includes('派发尚未接入'));
  assert.ok(!html.includes('审核候选'));
  assert.ok(!html.includes('FRESH_HOST_ACTION'));
});
test('rendered action button keeps positive Host label when dispatch callback is connected',async()=>{
  const {WorkbenchActionButton}=await loadViews() as
    {WorkbenchActionButton:FunctionComponent<Record<string,unknown>>};
  const html=renderToStaticMarkup(createElement(WorkbenchActionButton,{
    snapshot:inbox,catalog,action:work,onAction:()=>{}}));
  assert.ok(!html.includes('disabled'));
  assert.ok(html.includes('审核候选'));
  assert.ok(!html.includes('FRESH_HOST_ACTION'));
  assert.ok(!html.includes('派发尚未接入'));
});
test('rendered action button maps denial codes to localized messages',async()=>{
  const {WorkbenchActionButton}=await loadViews() as
    {WorkbenchActionButton:FunctionComponent<Record<string,unknown>>};
  const disconnected=renderToStaticMarkup(createElement(WorkbenchActionButton,{
    snapshot:inbox,catalog:{...catalog,connected:false},action:work,
    onAction:()=>{}}));
  assert.match(disconnected,/disabled/);
  assert.ok(disconnected.includes(
    ACTION_STATUS_MESSAGES.HOST_ACTION_SOURCE_NOT_CONNECTED));
  assert.ok(!disconnected.includes('HOST_ACTION_SOURCE_NOT_CONNECTED'));
  const unbound=renderToStaticMarkup(createElement(WorkbenchActionButton,{
    snapshot:inbox,catalog,action:{...work,hostActionId:null,
      hostActionRevision:null,hostActionKind:null},onAction:()=>{}}));
  assert.match(unbound,/disabled/);
  assert.ok(unbound.includes(ACTION_STATUS_MESSAGES.HOST_ACTION_IDENTITY_NOT_BOUND));
  assert.ok(!unbound.includes('HOST_ACTION_IDENTITY_NOT_BOUND'));
});
