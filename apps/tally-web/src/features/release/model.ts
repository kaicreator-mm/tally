/** UI-D #231 — user-facing release qualification view model.
 * No decorative controls: an action is displayable ONLY when a fresh Host
 * permit exists; J07a advisory PASS never unlocks anything here.
 */
export interface ReleaseGate{
  name:string;
  status:'PASS'|'FAIL'|'NOT_RUN'|'UNKNOWN'|'STALE';
  owner:string|null;
}
export interface ReleasePermit{
  operation:'merge'|'tag'|'publish';
  state:'GRANTED'|'DENIED'|'UNKNOWN';
  sourceSha:string|null;
  grantedAt:string|null;
  /** Issued by selected Host only after current source readback. */
  currentness?:'CURRENT'|'STALE'|'UNKNOWN';
}
export interface ReleaseReadiness{
  versionId:string;
  /** Exact version source pinned by the trusted Host. Absent => no actions. */
  currentSourceSha?:string|null;
  gates:readonly ReleaseGate[];
  /** Host-issued fresh permits (source-qualified); null = not connected. */
  permits:readonly ReleasePermit[]|null;
  rqState:'QUALIFIED'|'NOT_QUALIFIED'|'UNKNOWN';
  ownerGrantState:'GRANTED'|'NOT_GRANTED'|'UNKNOWN';
}
/** 1–3 user-facing blockers (Owner UX rule); raw gate matrix is secondary. */
export interface ReleaseBlocker{code:string;detail:string;owner:string|null;}
export function releaseBlockers(readiness:ReleaseReadiness):ReleaseBlocker[]{
  const blockers:ReleaseBlocker[]=[];
  if(readiness.rqState!=='QUALIFIED')
    blockers.push({code:readiness.rqState==='UNKNOWN'?'RQ_UNKNOWN':'RQ_MISSING',
      detail:readiness.rqState==='UNKNOWN'
        ?'独立 RQ 状态未知,需 Host 对账'
        :'独立 RQ 尚未取得,不能执行发布操作',
      owner:'独立验证者'});
  if(readiness.ownerGrantState!=='GRANTED')
    blockers.push({code:readiness.ownerGrantState==='UNKNOWN'?'GRANT_UNKNOWN'
        :'GRANT_MISSING',
      detail:readiness.ownerGrantState==='UNKNOWN'
        ?'Owner 授权状态未知'
        :'需要 Owner 对精确 SHA/操作的新鲜授权',
      owner:'Owner'});
  const failing=readiness.gates.filter(gate=>gate.status!=='PASS')
    .sort((a,b)=>(a.status==='FAIL'?0:1)-(b.status==='FAIL'?0:1));
  if(failing.length>0&&blockers.length<3)
    blockers.push({code:'GATES_INCOMPLETE',
      detail:`仍有 ${failing.length} 项 gate 未满足(首个:${failing[0].name})`,
      owner:failing[0].owner});
  return blockers.slice(0,3);
}
/** Displayable actions: only fresh Host GRANTED permits render enabled;
 * everything else explains what is missing instead of a dead button. */
export interface ActionDisplay{
  operation:ReleasePermit['operation'];
  enabled:boolean;
  reason:string|null;
}
export function actionDisplay(readiness:ReleaseReadiness,
  operation:ReleasePermit['operation']):ActionDisplay{
  const blockers=releaseBlockers(readiness);
  const permit=(readiness.permits??[])
    .find(entry=>entry.operation===operation);
  // A GRANTED-looking flag alone is not a source-qualified release permit.
  // Never enable if RQ, Owner grant or any independent gate is incomplete.
  if(blockers.length>0||readiness.gates.length===0)
    return {operation,enabled:false,reason:blockers[0]
      ?`${blockers[0].code}: ${blockers[0].detail}`:'RELEASE_GATES_UNVERIFIED'};
  if(!permit||permit.state!=='GRANTED')
    return {operation,enabled:false,reason:permit
      ?`HOST_PERMIT_${permit.state}: 许可未生效`:'HOST_PERMIT_MISSING'};
  const source=readiness.currentSourceSha;
  if(typeof source!=='string'||!/^[0-9a-f]{40}$/.test(source)||
     permit.sourceSha!==source||permit.currentness!=='CURRENT'||
     typeof permit.grantedAt!=='string'||Number.isNaN(Date.parse(permit.grantedAt)))
    return {operation,enabled:false,reason:'HOST_PERMIT_STALE_OR_UNVERIFIED'};
  return {operation,enabled:true,reason:null};
}
