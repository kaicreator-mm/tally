import test from 'node:test';
import assert from 'node:assert/strict';
import {NAV_GROUPS,navAvailability,findItemByRoute,groupKeyOf} from './nav-model.ts';

test('exactly four top groups in Owner-mandated order',()=>{
  assert.deepEqual(NAV_GROUPS.map(group=>group.key),
    ['workbench','development','governance','settings']);
});
test('no duplicate routes or sidebar project/version entries',()=>{
  const routes=new Set<string>(),keys=new Set<string>();
  for(const group of NAV_GROUPS)for(const item of group.items){
    assert.equal(routes.has(item.route),false,item.route);
    assert.equal(keys.has(item.key),false,item.key);
    routes.add(item.route);keys.add(item.key);
    assert.match(item.route,new RegExp(`^/${group.key}/`));
  }
});
test('strict three-level scope gating (project before version)',()=>{
  const tasks=NAV_GROUPS.flatMap(g=>g.items).find(item=>item.key==='tasks')!;
  const none=navAvailability(tasks,{projectId:null,versionId:null});
  assert.equal(none.available,false);
  assert.equal(none.missing,'project');
  const noVersion=navAvailability(tasks,{projectId:'p1',versionId:null});
  assert.equal(noVersion.available,false);
  assert.equal(noVersion.missing,'version');
  const ok=navAvailability(tasks,{projectId:'p1',versionId:'v1'});
  assert.equal(ok.available,true);
  const settings=NAV_GROUPS.flatMap(g=>g.items).find(item=>item.key==='ci')!;
  assert.equal(navAvailability(settings,{}).available,true);
});
test('route lookup and group membership',()=>{
  const item=findItemByRoute({group:'development',view:'tasks',query:{}});
  assert.equal(item?.key,'tasks');
  assert.equal(groupKeyOf(item!),'development');
  assert.equal(findItemByRoute({group:'nope',view:'nope',query:{}}),null);
});

test('version-bound lifecycle pages never open without a selected version',()=>{
  const keys=['requirements','product','l2-dag','maintenance','metadata'];
  for(const key of keys){
    const item=NAV_GROUPS.flatMap(g=>g.items).find(i=>i.key===key)!;
    assert.equal(navAvailability(item,{projectId:'owner/repo',versionId:null}).available,
      false,key);
    assert.equal(navAvailability(item,{projectId:'owner/repo',versionId:'version/v2'})
      .available,true,key);
  }
});
