/** UI-H #235 — per-version DAG view: layered SVG (pan/zoom/fit), filters,
 * node drawer, keyboard-accessible table alternative, exports. UI-C links
 * here instead of duplicating rendering. */
import {useMemo,useState} from 'react';
import type {VersionDag,DagNode,DagFilters} from './model.ts';
import {NOT_CONNECTED_DAG,detectCycles,danglingEdges,filterDag,
  layerNodes,toMermaid,toJson,versionIsolated} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Drawer} from '../../components/ui/Drawer.tsx';
import {Tabs} from '../../components/ui/Tabs.tsx';
import {DataTable} from '../../components/ui/DataTable.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const NODE_W=150,NODE_H=44,GAP_X=70,GAP_Y=26;
export function DagView({dag,locale='zh',onOpenTask}:
  {dag:VersionDag|null;versionId?:string|null;locale?:'zh'|'en';
   onOpenTask?:(taskId:string)=>void}){
  const zh=locale==='zh';
  const [filters,setFilters]=useState<DagFilters>({status:null,severity:null,
    agentId:null,onlyBlocked:false,group:null});
  const [selected,setSelected]=useState<DagNode|null>(null);
  const [zoom,setZoom]=useState(1);
  if(!dag||!dag.connected){
    return <EmptyState title={zh?'NOT_CONNECTED:该版本暂无可用 DAG 读模型'
        :'NOT_CONNECTED: no DAG read model for this version'}
      hint={zh?'缺失就是未连接,不显示空图冒充已合格 DAG'
        :'absence renders NOT_CONNECTED, never an empty qualified DAG'}/>;
  }
  if(!versionIsolated(dag))
    return <EmptyState title="VERSION_CONTAMINATION_DETECTED"
      hint={zh?'快照包含其它版本节点,已拒绝渲染':'foreign nodes detected; refused'}/>;
  const filtered=filterDag(dag,filters);
  const cycles=detectCycles(filtered);
  const dangling=danglingEdges(filtered);
  const layers=layerNodes(filtered);
  const width=Math.max(...layers.map(layer=>layer.length))*(NODE_W+GAP_X);
  const height=layers.length*(NODE_H+GAP_Y);
  const position=new Map<string,{x:number;y:number}>();
  layers.forEach((layer,row)=>layer.forEach((taskId,column)=>
    position.set(taskId,{x:column*(NODE_W+GAP_X),y:row*(NODE_H+GAP_Y)})));
  const agents=[...new Set(dag.nodes.map(node=>node.agentId)
    .filter((id):id is string=>id!==null))];
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?`版本任务 DAG · ${dag.versionId}`:`Version DAG · ${dag.versionId}`}
        sub={zh?`as-of ${dag.asOf.sourceSha.slice(0,12)} · ${
          dag.asOf.frozenAt?'已冻结':'未冻结'} · ${dag.nodes.length} 节点 / ${dag.edges.length} 边`
          :`as-of ${dag.asOf.sourceSha.slice(0,12)}`}/>
      <div style={{display:'flex',gap:'var(--s2)',flexWrap:'wrap',marginBottom:'var(--s3)'}}>
        <select aria-label="status" value={filters.status??''}
          onChange={event=>setFilters({...filters,status:event.target.value||null})}>
          <option value="">{zh?'全部状态':'all statuses'}</option>
          {['open','in_progress','review','blocked','done','failed']
            .map(status=><option key={status} value={status}>{status}</option>)}
        </select>
        <select aria-label="severity" value={filters.severity??''}
          onChange={event=>setFilters({...filters,severity:event.target.value||null})}>
          <option value="">{zh?'全部级别':'all'}</option>
          {['P0','P1','P2'].map(severity=>
            <option key={severity} value={severity}>{severity}</option>)}
        </select>
        <select aria-label="agent" value={filters.agentId??''}
          onChange={event=>setFilters({...filters,agentId:event.target.value||null})}>
          <option value="">{zh?'全部 Agent':'all agents'}</option>
          {agents.map(agent=><option key={agent} value={agent}>{agent}</option>)}
        </select>
        <label style={{display:'flex',gap:6,alignItems:'center'}}>
          <input type="checkbox" checked={filters.onlyBlocked}
            onChange={event=>setFilters({...filters,onlyBlocked:event.target.checked})}/>
          {zh?'只看阻塞':'blocked only'}</label>
        <label style={{display:'flex',gap:6,alignItems:'center',marginLeft:'auto'}}>
          {zh?'缩放':'zoom'}
          <input type="range" min={0.4} max={2} step={0.1} value={zoom}
            aria-label="zoom"
            onChange={event=>setZoom(Number(event.target.value))}/>
          <button onClick={()=>setZoom(1)}>{zh?'复位':'fit'}</button></label>
      </div>
      {cycles.length>0?<div role="alert" style={{color:'var(--danger)',
        marginBottom:'var(--s2)',fontSize:12.5}}>
        {zh?`检测到 ${cycles.length} 个环:`:`${cycles.length} cycles:`}
        {cycles.slice(0,2).map(cycle=>cycle.join('→')).join('; ')}</div>:null}
      {dangling.length>0?<div role="alert" style={{color:'var(--warn)',
        marginBottom:'var(--s2)',fontSize:12.5}}>
        {zh?`${dangling.length} 条边指向缺失任务(不跨版本补齐)`
          :`${dangling.length} dangling edges`}</div>:null}
      <div style={{overflow:'auto',border:'1px solid var(--border-subtle)',
        borderRadius:'var(--r-lg)',background:'var(--bg-app)',maxHeight:520}}>
        <svg role="img" aria-label={zh?'任务依赖图':'task dependency graph'}
          width={width*zoom+40} height={height*zoom+40}
          viewBox={`0 0 ${width+40} ${height+40}`}
          style={{display:'block'}}>
          <g transform={`translate(20,20) scale(${zoom})`}>
            {filtered.edges.map((edge,index)=>{
              const from=position.get(edge.from),to=position.get(edge.to);
              if(!from||!to)return null;
              const x1=from.x+NODE_W,y1=from.y+NODE_H/2,x2=to.x,y2=to.y+NODE_H/2;
              const style=edge.kind==='NATIVE'
                ?{stroke:'var(--brand)',strokeDasharray:'',strokeWidth:2}
                :edge.kind==='PLANNED'
                  ?{stroke:'var(--info)',strokeDasharray:'6 4',strokeWidth:1.6}
                  :{stroke:'var(--danger)',strokeDasharray:'2 4',strokeWidth:1.4};
              return <g key={`${edge.from}-${edge.to}-${index}`}>
                <line x1={x1} y1={y1} x2={x2} y2={y2} {...style}
                  markerEnd={edge.kind==='NATIVE'?'url(#arrow)':''}/>
              </g>;})}
            <defs><marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5"
              markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M 0 0 L 10 5 L 0 10 z" fill="var(--brand)"/></marker></defs>
            {filtered.nodes.map(node=>{
              const pos=position.get(node.taskId);
              if(!pos)return null;
              const fill=node.status==='done'?'var(--ok-dim)'
                :node.status==='blocked'||node.status==='failed'?'var(--danger-dim)'
                :node.status==='review'?'var(--info-dim)':'var(--bg-raised)';
              return <g key={node.taskId} transform={`translate(${pos.x},${pos.y})`}
                onClick={()=>setSelected(node)}
                onKeyDown={event=>{if(event.key==='Enter')setSelected(node);}}
                tabIndex={0} role="button"
                aria-label={`${node.taskId} ${node.title} ${node.status}`}
                style={{cursor:'pointer'}}>
                <rect width={NODE_W} height={NODE_H} rx={10} fill={fill}
                  stroke={node.severity==='P0'?'var(--danger)'
                    :node.severity==='P1'?'var(--warn)':'var(--border-strong)'}
                  strokeWidth={node.severity==='P0'?2:1}/>
                <text x={10} y={18} fontSize={11} fill="var(--text-secondary)">
                  {node.taskId} · {node.severity}{node.executable?'':' 🔒'}</text>
                <text x={10} y={33} fontSize={11.5} fill="var(--text-primary)">
                  {node.title.slice(0,16)}</text>
              </g>;})}
          </g>
        </svg>
      </div>
    </Card>
    <Tabs ariaLabel="dag-alt" items={[
      {key:'table',label:zh?'表格(键盘可达)':'Table (keyboard)',
        content:<DataTable rows={filtered.nodes} rowKey={node=>node.taskId}
          onRowClick={node=>onOpenTask?.(node.taskId)}
          columns={[
            {key:'id',header:'Task',cell:node=><code>{node.taskId}</code>},
            {key:'title',header:zh?'标题':'Title',cell:node=>node.title},
            {key:'status',header:zh?'状态':'Status',cell:node=>
              <Badge tone={node.status==='done'?'ok':'neutral'}>{node.status}</Badge>},
            {key:'exec',header:zh?'可执行':'Executable',
              cell:node=><Badge tone={node.executable?'ok':'warn'}>
                {node.executable?'YES':'HOST_GATE'}</Badge>},
          ]}/>},
      {key:'mermaid',label:'Mermaid',
        content:<textarea readOnly rows={12} value={toMermaid(filtered)}
          style={{width:'100%',fontFamily:'var(--font-mono)',fontSize:12}}/>},
      {key:'json',label:'JSON',
        content:<textarea readOnly rows={12} value={toJson(filtered)}
          style={{width:'100%',fontFamily:'var(--font-mono)',fontSize:12}}/>},
    ]}/>
    <Drawer open={selected!==null} onClose={()=>setSelected(null)}
      title={selected?`${selected.taskId} · ${selected.title}`:''}>
      {selected?<dl style={{display:'grid',gridTemplateColumns:'auto 1fr',
        gap:'6px 12px',margin:0,fontSize:13}}>
        <dt>{zh?'状态':'Status'}</dt><dd style={{margin:0}}>{selected.status}</dd>
        <dt>{zh?'级别':'Severity'}</dt><dd style={{margin:0}}>{selected.severity}</dd>
        <dt>{zh?'需求组':'Requirement'}</dt>
        <dd style={{margin:0}}>{selected.requirementGroup??'—'}</dd>
        <dt>{zh?'阶段/泳道':'Phase/Lane'}</dt>
        <dd style={{margin:0}}>{selected.phase??'—'} / {selected.lane??'—'}</dd>
        <dt>Agent</dt><dd style={{margin:0}}>{selected.agentId??'—'}</dd>
        <dt>{zh?'可执行性(Host 判定)':'Executable (Host-derived)'}</dt>
        <dd style={{margin:0}}><Badge tone={selected.executable?'ok':'warn'}>
          {selected.executable?'YES':'HOST_GATE'}</Badge></dd>
        <dt>{zh?'进入任务':'Open task'}</dt>
        <dd style={{margin:0}}><button onClick={()=>onOpenTask?.(selected.taskId)}
          style={{border:0,background:'transparent',color:'var(--brand-strong)',
            cursor:'pointer',padding:0,font:'inherit'}}>
          {zh?'打开任务详情':'open task'}</button></dd>
      </dl>:null}
    </Drawer>
  </div>;
}
export {NOT_CONNECTED_DAG};
export function useVersionDag(load:(versionId:string)=>VersionDag|null,
  versionId:string|null){
  // Hook kept trivial: per-version identity means switching versions always
  // reloads; previous nodes are never reused across versions.
  return useMemo(()=>versionId?load(versionId):null,[versionId,load]);
}
