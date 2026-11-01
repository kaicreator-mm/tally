/** UI-D #231 — Requirement→Story/Epic→PRD/Acceptance→L2→Freeze view model.
 * Freeze authority stays with the Host/K; the UI only projects review state
 * and derives next-step hints for the dynamic forms (UI-B renderer).
 */
export type RequirementKind='requirement'|'epic'|'story';
export interface RequirementNode{
  id:string;kind:RequirementKind;title:string;
  parent:string|null;
  /** Frozen Product/PRD revision bound to this node (null = not frozen). */
  productRevision:number|null;
  acceptanceRefs:readonly string[];
  l2Decision:string|null;
  freezeState:'NOT_FROZEN'|'FROZEN'|'STALE'|'UNKNOWN';
  updatedAt:string;
}
export interface RequirementTree{
  roots:RequirementNode[];
  childrenOf:(id:string)=>RequirementNode[];
}
export function buildTree(nodes:readonly RequirementNode[]):RequirementTree{
  const byParent=new Map<string|null,RequirementNode[]>();
  for(const node of nodes)
    byParent.set(node.parent,[...(byParent.get(node.parent)??[]),node]);
  const sortById=(list:RequirementNode[])=>list.slice()
    .sort((a,b)=>a.id.localeCompare(b.id,undefined,{numeric:true}));
  return {roots:sortById(byParent.get(null)??[]),
    childrenOf:(id:string)=>sortById(byParent.get(id)??[])};
}
/** A story is product-complete only when its own acceptance refs exist and a
 * non-stale freeze is bound; UNKNOWN never counts as frozen. */
export function storyCompletion(node:RequirementNode):
  {frozen:boolean;acceptanceCount:number;reasons:string[]}{
  const reasons:string[]=[];
  if(node.freezeState!=='FROZEN')
    reasons.push(node.freezeState==='STALE'?'freeze_stale'
      :node.freezeState==='UNKNOWN'?'freeze_unknown':'not_frozen');
  if(node.acceptanceRefs.length===0)reasons.push('acceptance_missing');
  return {frozen:reasons.length===0,
    acceptanceCount:node.acceptanceRefs.length,reasons};
}
/** Next-step hint for the guided form: first actionable gap in order. */
export function nextStepHint(node:RequirementNode):
  'draft_product'|'add_acceptance'|'request_freeze'|'renew_freeze'|'none'{
  const completion=storyCompletion(node);
  if(completion.reasons.includes('acceptance_missing'))return 'add_acceptance';
  if(completion.reasons.includes('freeze_stale'))return 'renew_freeze';
  if(completion.reasons.includes('not_frozen'))return 'request_freeze';
  if(node.productRevision===null)return 'draft_product';
  return 'none';
}
/** Freeze-diff summary: which children changed since the frozen revision. */
export function freezeDiff(nodes:readonly RequirementNode[],
  frozenRevision:number):{changed:RequirementNode[];unchangedCount:number}{
  const changed=nodes.filter(node=>node.productRevision!==null
    &&node.productRevision>frozenRevision);
  return {changed,
    unchangedCount:nodes.length-changed.length};
}
