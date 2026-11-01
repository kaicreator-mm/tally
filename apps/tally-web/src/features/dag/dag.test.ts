import test from 'node:test';
import assert from 'node:assert/strict';
import {classifyEdge,detectCycles,danglingEdges,fanSummary,layerNodes,
  filterDag,versionIsolated,toMermaid,toJson,NOT_CONNECTED_DAG}
  from './model.ts';
import type {VersionDag,DagNode,DagEdge} from './model.ts';

const node=(over:Partial<DagNode>):DagNode=>({taskId:'228',versionId:'v2',
  title:'UI-A',status:'open',severity:'P1',requirementGroup:'shell',
  phase:'build',lane:'a',agentId:null,executable:false,...over});
const edge=(over:Partial<DagEdge>):DagEdge=>({from:'228',to:'232',
  kind:'NATIVE',evidenceRef:'https://github.com/kaicreator-mm/tally/issues/228',
  amendedAt:null,...over});
const dagOf=(nodes:DagNode[],edges:DagEdge[],versionId='v2',
  connected=true):VersionDag=>({versionId,
  asOf:{sourceSha:'a'.repeat(40),frozenAt:'2026-11-01T00:00:00Z'},
  nodes,edges,connected});
test('edge classification: text mentions never become native',()=>{
  assert.equal(classifyEdge({nativeAccepted:true,planned:false}),'NATIVE');
  assert.equal(classifyEdge({nativeAccepted:false,planned:true}),'PLANNED');
  assert.equal(classifyEdge({nativeAccepted:false,planned:false}),'UNRESOLVED');
});
test('four-way parallel lanes topologize; cycle members land in a final layer',()=>{
  const dag=dagOf([
    node({taskId:'228'}),node({taskId:'229'}),node({taskId:'230'}),
    node({taskId:'231'}),node({taskId:'232'}),
    node({taskId:'c1'}),node({taskId:'c2'}),
  ],[
    edge({from:'228',to:'232'}),edge({from:'229',to:'232'}),
    edge({from:'230',to:'232'}),edge({from:'231',to:'232'}),
    edge({from:'c1',to:'c2',kind:'PLANNED'}),
    edge({from:'c2',to:'c1',kind:'PLANNED'}),
  ]);
  const layers=layerNodes(dag);
  assert.equal(layers[0].length,4); // the four parallel lanes first
  assert.ok(layers[0].includes('228')&&layers[0].includes('231'));
  assert.deepEqual(layers.at(-1),['c1','c2']); // cycle members flagged last
  const cycles=detectCycles(dag);
  assert.equal(cycles.length,1);
  // Closed-walk representation: starts and ends on the same node.
  assert.equal(cycles[0][0],cycles[0].at(-1));
  assert.deepEqual([...new Set(cycles[0])].sort(),['c1','c2']);
});
test('late legal amendment keeps the edge native with a stamp',()=>{
  const dag=dagOf([node({taskId:'a'}),node({taskId:'b'})],
    [edge({from:'a',to:'b',amendedAt:'2026-11-02T00:00:00Z'})]);
  assert.equal(dag.edges[0].kind,'NATIVE');
  assert.equal(dag.edges[0].amendedAt,'2026-11-02T00:00:00Z');
  assert.deepEqual(detectCycles(dag),[]);
  assert.deepEqual(danglingEdges(dag),[]);
});
test('dangling edges are exposed, never silently cross-filled',()=>{
  const dag=dagOf([node({taskId:'a'})],[edge({from:'a',to:'MISSING'})]);
  const dangling=danglingEdges(dag);
  assert.equal(dangling.length,1);
  assert.equal(dangling[0].to,'MISSING');
});
test('filters preserve identity and drop edges of filtered-out nodes',()=>{
  const dag=dagOf([node({taskId:'a',status:'open'}),
    node({taskId:'b',status:'blocked'})],
    [edge({from:'a',to:'b'})]);
  const blockedOnly=filterDag(dag,{status:null,severity:null,agentId:null,
    onlyBlocked:true,group:null});
  assert.deepEqual(blockedOnly.nodes.map(n=>n.taskId),['b']);
  assert.deepEqual(blockedOnly.edges,[]);
  const all=filterDag(dag,{status:null,severity:null,agentId:null,
    onlyBlocked:false,group:null});
  assert.equal(all.edges.length,1);
});
test('cross-version contamination is detected and refused',()=>{
  assert.equal(versionIsolated(dagOf([node({taskId:'a',versionId:'v2'}),
    node({taskId:'b',versionId:'v1'})],[])),false);
  assert.equal(versionIsolated(dagOf([node({taskId:'a'})],[])),true);
});
test('exports carry as-of identity and edge authority',()=>{
  const dag=dagOf([node({taskId:'a'}),node({taskId:'b'})],
    [edge({from:'a',to:'b',kind:'PLANNED',evidenceRef:null})]);
  const mmd=toMermaid(dag);
  assert.match(mmd,/flowchart LR/);
  assert.match(mmd,/version=v2 asOf=aaaaaaaaaaaa/);
  assert.match(mmd,/task_0 -\.-> task_1/); // generated safe node aliases, planned dashed
  const parsed=JSON.parse(toJson(dag));
  assert.equal(parsed.asOf.sourceSha,'a'.repeat(40));
  assert.equal(parsed.edges[0].authority,'planned_candidate');
});
test('NOT_CONNECTED is explicit and distinct from an empty qualified DAG',()=>{
  assert.equal(NOT_CONNECTED_DAG.connected,false);
  assert.equal(NOT_CONNECTED_DAG.nodes.length,0);
  const empty=dagOf([],[]);
  assert.equal(empty.connected,true);
  assert.notEqual(empty.connected,NOT_CONNECTED_DAG.connected);
});
test('fan summary counts in/out degrees',()=>{
  const dag=dagOf([node({taskId:'a'}),node({taskId:'b'}),node({taskId:'c'})],
    [edge({from:'a',to:'c'}),edge({from:'b',to:'c'})]);
  const {inDegree,outDegree}=fanSummary(dag);
  assert.equal(inDegree.get('c'),2);
  assert.equal(outDegree.get('a'),1);
  assert.equal(outDegree.get('c'),undefined);
});

test('Mermaid export rejects injected issue-title commands and unsafe IDs',()=>{
  const graph=dagOf([
    node({taskId:'a/b',title:'Normal"]\\nflowchart RL\\n%% evil'}),
    node({taskId:'b:c',title:'<img src=x onerror=alert(1)>'}),
  ],[edge({from:'a/b',to:'b:c',kind:'NATIVE'})]);
  const text=toMermaid(graph);
  assert.match(text,/task_0 --> task_1/);
  assert.match(text,/task_0\\["a b Normal/);
  assert.ok(!text.includes('\\nflowchart RL'));
  assert.ok(!text.includes('<img'));
  assert.equal(text.split('flowchart LR').length,2);
});
test('UNKNOWN native Task workflow and priority survive source-qualified DAG',()=>{
  const graph=dagOf([node({status:'unknown',severity:'UNKNOWN'})],[]);
  assert.equal(filterDag(graph,{status:'unknown',severity:'UNKNOWN',agentId:null,
    onlyBlocked:false,group:null}).nodes.length,1);
  assert.match(toMermaid(graph),/task_0/);
});
