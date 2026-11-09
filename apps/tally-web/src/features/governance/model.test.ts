import test from 'node:test';
import assert from 'node:assert/strict';
import {governanceItems,governanceAuthority} from './model.ts';
import type {GovernanceSnapshot,GovernanceItem} from './model.ts';
const item:GovernanceItem={id:'pack1',kind:'domain-package',
  title:'Engineering B',state:'SELECTED',projectId:'org/tally',
  versionId:'version/v2',sourceRef:'https://github.com/org/tally/issues/227',
  summary:null,authority:'BUILD_SELECTED'};
const snapshot:GovernanceSnapshot={connected:true,
  source:{repository:'org/tally',sourceSha:'a'.repeat(40),
    currentness:'CURRENT',readback:'VERIFIED'},records:[
  item,{...item,id:'wf',kind:'workflow',authority:'PROPOSAL'},
  {...item,id:'learning',kind:'experience',authority:'PROPOSAL'}]};
test('unconnected governance is not an empty verified ledger',()=>{
  assert.equal(governanceItems(null,'workflow'),null);
  assert.equal(governanceItems({...snapshot,connected:false},'workflow'),null);
});
test('governance sections never mix records',()=>{
  assert.deepEqual(governanceItems(snapshot,'domain-package')?.map(x=>x.id),['pack1']);
  assert.deepEqual(governanceItems(snapshot,'workflow')?.map(x=>x.id),['wf']);
});
test('Meta proposals and BUILD selected package remain distinguishable',()=>{
  assert.match(governanceAuthority(snapshot.records[1]),/候选/);
  assert.match(governanceAuthority(item),/BUILD/);
});
