/** UI-D #231 — requirement hierarchy, product freeze and next-step surfaces. */
import {buildTree,storyCompletion,nextStepHint,freezeDiff} from './model.ts';
import type {RequirementNode} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Collapse} from '../../components/ui/Collapse.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const HINT_ZH:Record<string,string>={draft_product:'起草产品 PRD',
  add_acceptance:'补充验收标准',request_freeze:'发起独立评审冻结',
  renew_freeze:'重新冻结(内容已变更)',none:'已就绪'};
export function RequirementsView({nodes,connected=true,locale='zh'}:
  {nodes:readonly RequirementNode[];connected?:boolean;locale?:'zh'|'en'}){
  const zh=locale==='zh';
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  const tree=buildTree(nodes);
  const render=(node:RequirementNode,depth:number):React.ReactNode=>
    <li key={node.id} style={{listStyle:'none',marginBottom:'var(--s2)'}}>
      <div style={{display:'flex',gap:'var(--s2)',alignItems:'center',
        paddingLeft:depth*18}}>
        <Badge tone={node.kind==='requirement'?'brand'
          :node.kind==='epic'?'info':'neutral'}>{node.kind}</Badge>
        <code style={{fontSize:12}}>{node.id}</code>
        <span>{node.title}</span>
        <Badge tone={node.freezeState==='FROZEN'?'ok'
          :node.freezeState==='STALE'?'warn':'neutral'}>
          {node.freezeState}</Badge>
        {nextStepHint(node)!=='none'?
          <span style={{color:'var(--brand-strong)',fontSize:12}}>
            → {zh?HINT_ZH[nextStepHint(node)]:nextStepHint(node)}</span>:null}
      </div>
      {tree.childrenOf(node.id).length>0?
        <ul style={{margin:'6px 0 0',padding:0}}>
          {tree.childrenOf(node.id).map(child=>render(child,depth+1))}</ul>:null}
    </li>;
  const frozen=nodes.filter(node=>node.freezeState==='FROZEN');
  const diff=freezeDiff(nodes,frozen.length?Math.min(
    ...frozen.map(node=>node.productRevision??0)):0);
  return <div style={{display:'grid',gap:'var(--s5)'}}>
    <Card>
      <CardHeader title={zh?'需求层级':'Requirement hierarchy'}
        sub={zh?'Requirement → Epic/Story → PRD/验收 → L2 → 冻结':'hierarchy'}/>
      {tree.roots.length===0?<EmptyState title={zh?'暂无需求':'No requirements'}/>:
        <ul style={{margin:0,padding:0}}>{tree.roots.map(node=>render(node,0))}</ul>}
    </Card>
    <Collapse title={zh?'冻结差异与技术细节':'Freeze diff & technical detail'}
      defaultOpen={false}>
      <div style={{fontSize:13}}>
        {zh?'冻结后变更节点':'Changed since freeze'}: {diff.changed.length}
        {' · '}{zh?'未变更':'unchanged'}: {diff.unchangedCount}
        {diff.changed.slice(0,5).map(node=>
          <div key={node.id}><code>{node.id}</code> {node.title}
            {' → rev '}{node.productRevision}</div>)}
      </div>
    </Collapse>
  </div>;
}
export {storyCompletion};
