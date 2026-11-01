/** Immutable governance source ledger. Workflows/packages/feedback are NOT
 * a second Agent scheduler, package activator or Meta self-approval route. */
export type GovernanceKind='workflow'|'domain-package'|'experience'|'audit';
export interface GovernanceItem{
  id:string;kind:GovernanceKind;title:string;state:string;
  projectId:string;versionId:string|null;
  sourceRef:string;summary:string|null;
  authority:'OBSERVED'|'PROPOSAL'|'REVIEWED'|'BUILD_SELECTED';
}
export interface GovernanceSnapshot{
  connected:boolean;source:{repository:string;sourceSha:string;
    currentness:string;readback:string};
  records:readonly GovernanceItem[];
}
export function governanceItems(snapshot:GovernanceSnapshot|null,
  kind:GovernanceKind):GovernanceItem[]|null{
  if(!snapshot?.connected)return null;
  return snapshot.records.filter(item=>item.kind===kind);
}
export function governanceAuthority(item:GovernanceItem){
  if(item.authority==='PROPOSAL')return '仅候选 · 不自动生效';
  if(item.kind==='domain-package'&&item.authority==='BUILD_SELECTED')
    return '当前 BUILD 选择（只读）';
  return '来源观察；不授予执行权限';
}
