/** UI-H #235 — per-version canonical Task DAG read model.
 * Edge kinds stay source-qualified: NATIVE (accepted native blocked_by),
 * PLANNED (candidate only, dashed), UNRESOLVED (no executable authority).
 * Execution availability comes from Business/Host, never inferred here.
 * No durations are invented; versions never share nodes or edges.
 */
export type EdgeKind='NATIVE'|'PLANNED'|'UNRESOLVED';
export interface DagNode{
  taskId:string;versionId:string;title:string;
  status:'open'|'in_progress'|'review'|'blocked'|'done'|'failed'|'unknown';
  severity:'P0'|'P1'|'P2'|'UNKNOWN';
  requirementGroup:string|null;
  phase:string|null;
  lane:string|null;
  agentId:string|null;
  /** Host-derived execution availability (not graph-inferred). */
  executable:boolean;
}
export interface DagEdge{
  from:string;to:string;kind:EdgeKind;
  /** Native acceptance evidence (issue comment ref) or null for planned. */
  evidenceRef:string|null;
  /** Legal amendment record if the edge was amended after freeze. */
  amendedAt:string|null;
}
export interface VersionDag{
  versionId:string;
  /** Configured Task IDs do not prove exhaustive frozen version membership. */
  rosterCoverage?:'UNVERIFIED';
  /** Frozen as-of identity of this snapshot. */
  asOf:{sourceSha:string;frozenAt:string|null};
  nodes:readonly DagNode[];
  edges:readonly DagEdge[];
  /** False = read model not connected; absence is never an empty DAG. */
  connected:boolean;
}
export const NOT_CONNECTED_DAG:VersionDag=Object.freeze({
  versionId:'',asOf:{sourceSha:'',frozenAt:null},nodes:[],edges:[],
  connected:false});

/** Classify an incoming dependency claim into an edge kind; GitHub text
 * mentions never become native edges. */
export function classifyEdge(claim:{
  nativeAccepted:boolean;planned:boolean;}):EdgeKind|null{
  if(claim.nativeAccepted)return 'NATIVE';
  if(claim.planned)return 'PLANNED';
  return 'UNRESOLVED';
}
/** Cycle detection (iterative DFS over native+planned edges). */
export function detectCycles(dag:VersionDag):string[][]{
  const adjacency=new Map<string,string[]>();
  for(const edge of dag.edges)
    adjacency.set(edge.to,[...(adjacency.get(edge.to)??[]),edge.from]);
  const color=new Map<string,0|1|2>();
  const cycles:string[][]=[];
  const stack:string[]=[];
  const visit=(nodeId:string)=>{
    color.set(nodeId,1);stack.push(nodeId);
    for(const prev of adjacency.get(nodeId)??[]){
      const state=color.get(prev)??0;
      if(state===1){
        const at=stack.indexOf(prev);
        if(at>=0)cycles.push([...stack.slice(at),prev]);
      }else if(state===0)visit(prev);
    }
    stack.pop();color.set(nodeId,2);
  };
  for(const node of dag.nodes)
    if((color.get(node.taskId)??0)===0)visit(node.taskId);
  // dedupe rotated cycles
  const seen=new Set<string>();
  return cycles.filter(cycle=>{
    const key=[...cycle].sort().join('>');
    if(seen.has(key))return false;
    seen.add(key);return true;
  });
}
/** Missing dependencies: edges pointing at tasks absent from this version
 * (never silently cross versions). */
export function danglingEdges(dag:VersionDag):DagEdge[]{
  const ids=new Set(dag.nodes.map(node=>node.taskId));
  return dag.edges.filter(edge=>!ids.has(edge.from)||!ids.has(edge.to));
}
/** Fan-in/fan-out summary for layout heuristics and table view. */
export function fanSummary(dag:VersionDag){
  const inDegree=new Map<string,number>(),outDegree=new Map<string,number>();
  for(const edge of dag.edges){
    inDegree.set(edge.to,(inDegree.get(edge.to)??0)+1);
    outDegree.set(edge.from,(outDegree.get(edge.from)??0)+1);
  }
  return {inDegree,outDegree};
}
/** Topological layering (native+planned only) for readable layout; nodes on
 * cycles keep their DFS layer. 200+ nodes remain readable via layers. */
export function layerNodes(dag:VersionDag):string[][]{
  const edges=dag.edges.filter(edge=>edge.kind!=='UNRESOLVED');
  const indegree=new Map<string,number>();
  const ids=dag.nodes.map(node=>node.taskId);
  for(const id of ids)indegree.set(id,0);
  for(const edge of edges)indegree.set(edge.to,(indegree.get(edge.to)??0)+1);
  const layers:string[][]=[];
  let current=ids.filter(id=>(indegree.get(id)??0)===0);
  const placed=new Set(current);
  while(current.length>0){
    layers.push(current);
    const next:string[]=[];
    for(const id of current)
      for(const edge of edges)
        if(edge.from===id){
          const remain=(indegree.get(edge.to)??0)-1;
          indegree.set(edge.to,remain);
          if(remain===0&&!placed.has(edge.to)){placed.add(edge.to);next.push(edge.to);}
        }
    current=next;
  }
  // cycle members never reached: append as a final layer (visually flagged)
  const rest=ids.filter(id=>!placed.has(id));
  if(rest.length>0)layers.push(rest);
  return layers;
}
export interface DagFilters{status:string|null;severity:string|null;
  agentId:string|null;onlyBlocked:boolean;group:'requirement'|'phase'|'lane'|null;}
export function filterDag(dag:VersionDag,filters:DagFilters):VersionDag{
  if(!dag.connected)return dag;
  const keep=new Set(dag.nodes.filter(node=>
    (!filters.status||node.status===filters.status)&&
    (!filters.severity||node.severity===filters.severity)&&
    (!filters.agentId||node.agentId===filters.agentId)&&
    (!filters.onlyBlocked||node.status==='blocked')).map(node=>node.taskId));
  return {...dag,nodes:dag.nodes.filter(node=>keep.has(node.taskId)),
    edges:dag.edges.filter(edge=>keep.has(edge.from)&&keep.has(edge.to))};
}
/** Cross-version contamination guard: every node must carry this version's
 * id; any foreign node makes the snapshot invalid. */
export function versionIsolated(dag:VersionDag):boolean{
  return dag.nodes.every(node=>node.versionId===dag.versionId);
}
/** Mermaid export with source-qualified styling (dashed for PLANNED,
 * dotted+red for UNRESOLVED, %% comments carry as-of identity). */
export function toMermaid(dag:VersionDag):string{
  // GitHub-controlled titles and IDs are never Mermaid source.
  const safe=(value:string)=>String(value).replace(/[^\p{L}\p{N} ._#-]/gu,' ')
    .replace(/\s+/g,' ').trim().slice(0,96);
  const aliases=new Map(dag.nodes.map((node,i)=>[node.taskId,'task_'+i]));
  const lines=['flowchart LR',
    `%% version=${safe(dag.versionId)} asOf=${safe(dag.asOf.sourceSha.slice(0,12))}`,
    `%% frozen=${safe(dag.asOf.frozenAt??'NOT_FROZEN')} connected=${dag.connected}`];
  for(const node of dag.nodes)
    lines.push(`  ${aliases.get(node.taskId)}["${safe(node.taskId+" "+node.title)}"]`);
  for(const edge of dag.edges){
    const style=edge.kind==='NATIVE'?'-->'
      :edge.kind==='PLANNED'?'-.->':'-.->';
    const from=aliases.get(edge.from),to=aliases.get(edge.to);
    if(!from||!to)continue;
    lines.push(`  ${from} ${style} ${to}`);
    if(edge.kind==='UNRESOLVED')
      lines.push('  %% UNRESOLVED: not an executable dependency');
  }
  return lines.join('\n');
}
/** JSON export carries per-edge evidence and as-of identity. */
export function toJson(dag:VersionDag):string{
  return JSON.stringify({versionId:dag.versionId,asOf:dag.asOf,
    connected:dag.connected,
    nodes:dag.nodes,edges:dag.edges.map(edge=>({...edge,
      authority:edge.kind==='NATIVE'?'native_blocked_by'
        :edge.kind==='PLANNED'?'planned_candidate':'unresolved'}))},null,2);
}
