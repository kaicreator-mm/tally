import test from 'node:test';
import assert from 'node:assert/strict';
import {readV2HostModel,validateV2ReadModel} from './ui-readmodel.mjs';
const SHA='a'.repeat(40),source={repository:'org/tally',sourceSha:SHA,
  currentness:'CURRENT',readback:'VERIFIED'};
const scope={projectId:'org/tally',versionId:'version/v2'};
const tasks={connected:true,versionId:'version/v2',source,tasks:[{
  id:'228',versionId:'version/v2',title:'Work item',blockedBy:[]}]};
const host={tasks:async()=>tasks};
test('missing trusted host cannot report empty success',async()=>{
  assert.deepEqual(await readV2HostModel(null,'tasks',scope),{
    connected:false,model:'tasks',reason:'HOST_READMODEL_UNAVAILABLE'});
});
test('qualified current tasks retain original projection',async()=>{
  assert.deepEqual(await readV2HostModel(host,'tasks',scope),tasks);
});
test('cross-version record fails closed',async()=>{
  const other={...tasks,tasks:[{...tasks.tasks[0],versionId:'version/v3'}]};
  assert.equal(validateV2ReadModel('tasks',other,scope),false);
  assert.equal((await readV2HostModel({tasks:async()=>other},'tasks',scope)).reason,'SOURCE_UNQUALIFIED');
});
test('foreign repository fails closed',()=>{
  assert.equal(validateV2ReadModel('tasks',
    {...tasks,source:{...source,repository:'other/tally'}},scope),false);
});
test('unverified transport/currentness and missing SHA refuse',()=>{
  assert.equal(validateV2ReadModel('tasks',
    {...tasks,source:{...source,currentness:'UNKNOWN'}},scope),false);
  assert.equal(validateV2ReadModel('tasks',
    {...tasks,source:{...source,sourceSha:''}},scope),false);
});
test('native DAG edges require explicit proof and same-version nodes',()=>{
  const node={taskId:'1',versionId:'version/v2'},node2={taskId:'2',versionId:'version/v2'};
  const dag={connected:true,versionId:'version/v2',source,
    asOf:{sourceSha:SHA,frozenAt:null},nodes:[node,node2],
    edges:[{from:'1',to:'2',kind:'NATIVE',evidenceRef:null}]};
  assert.equal(validateV2ReadModel('dag',dag,scope),false);
  assert.equal(validateV2ReadModel('dag',{...dag,
    edges:[{...dag.edges[0],kind:'PLANNED'}]},scope),true);
  assert.equal(validateV2ReadModel('dag',{...dag,
    edges:[{...dag.edges[0],evidenceRef:'https://github.com/org/tally/issues/1'}]},scope),true);
});
test('private credentials never reach browser even from selected host',async()=>{
  const snapshot={connected:true,source,snapshot:{
    agentProviders:[],apiKey:'sk-secretsecretsecret1234567'}};
  assert.equal((await readV2HostModel({settings:async()=>snapshot},'settings',{})).reason,
    'SOURCE_UNQUALIFIED');
});
test('metadata missing arrays cannot be interpreted as zero observations',async()=>{
  const incomplete={connected:true,versionId:'version/v2',source,
    attempts:[],reports:[],lineage:[]};
  assert.equal((await readV2HostModel({metadata:async()=>incomplete},'metadata',scope)).reason,
    'SOURCE_UNQUALIFIED');
});
test('provider failure never leaks exception details',async()=>{
  const bad=await readV2HostModel({tasks:async()=>{throw Error('sk-TOPSECRET');}},'tasks',scope);
  assert.deepEqual(bad,{connected:false,model:'tasks',reason:'HOST_READ_FAILED'});
});
test('catalog refuses cross-project version identity',()=>{
  const catalog={connected:true,source,projects:[{id:'org/tally',title:'Tally'}],
    versionsByProject:{'org/tally':[{id:'version/v2',projectId:'other/repo'}]}};
  assert.equal(validateV2ReadModel('catalog',catalog,{}),false);
});

test('workbench queue rejects cross-repo actions and self-asserted states',()=>{
  const base={connected:true,
    source:{repository:'org/tally',sourceSha:SHA,currentness:'CURRENT',readback:'VERIFIED'},
    actions:[{id:'a1',taskId:'227',projectId:'org/tally',versionId:'version/v2',
      title:'Review',kind:'APPROVAL',state:'PENDING',ownerRole:'owner',
      sourceRef:'https://github.com/org/tally/issues/227',reason:null,updatedAt:null}]};
  assert.equal(validateV2ReadModel('workbench',base,{}),true);
  assert.equal(validateV2ReadModel('workbench',{...base,actions:[
    {...base.actions[0],projectId:'other/repo'}]},{}),false);
  assert.equal(validateV2ReadModel('workbench',{...base,actions:[
    {...base.actions[0],state:'ACCEPTED'}]},{}),false);
  assert.equal(validateV2ReadModel('workbench',{...base,actions:[
    {...base.actions[0],sourceRef:'https://attacker.example/issues/1'}]},{}),false);
});

test('versioned lifecycle read models refuse missing source arrays',async()=>{
  const base={connected:true,versionId:'version/v2',source};
  for(const [kind,valid] of [
    ['agents',{attempts:[]}],['requirements',{nodes:[]}],['qa',{records:[]}],
    ['maintenance',{defects:[],learnings:[]}],['deployment',{deployments:[]}]
  ]){
    assert.equal(validateV2ReadModel(kind,base,scope),false,kind);
    assert.equal(validateV2ReadModel(kind,{...base,...valid},scope),true,kind);
  }
});
test('governance read model rejects source-crossing records',()=>{
  const good={connected:true,source,records:[{
    id:'1',projectId:'org/tally',versionId:'version/v2',
    title:'Package',kind:'domain-package',authority:'BUILD_SELECTED',
    sourceRef:'https://github.com/org/tally/issues/1',state:'SELECTED'}]};
  assert.equal(validateV2ReadModel('domainPackages',good,{}),true);
  assert.equal(validateV2ReadModel('domainPackages',{...good,
    records:[{...good.records[0],projectId:'other/repo'}]},{}),false);
});
