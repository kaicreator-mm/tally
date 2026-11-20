import type {ReactNode} from 'react';

export function EmptyState({title,hint,action}:{title:string;hint?:string;action?:ReactNode}){
  return <div style={{padding:'var(--s8)',textAlign:'center',color:'var(--text-muted)',
    border:'1px dashed var(--border-strong)',borderRadius:'var(--r-lg)',
    background:'var(--bg-panel)'}}>
    <div style={{color:'var(--text-secondary)',fontWeight:600,marginBottom:'var(--s1)'}}>{title}</div>
    {hint?<div style={{fontSize:12.5}}>{hint}</div>:null}
    {action?<div style={{marginTop:'var(--s4)'}}>{action}</div>:null}
  </div>;
}
export function ErrorState({title,hint,onRetry,retryLabel='Retry'}:
  {title:string;hint?:string;onRetry?:()=>void;retryLabel?:string}){
  return <div role="alert" style={{padding:'var(--s6)',border:'1px solid #7c3a4d',
    borderRadius:'var(--r-lg)',background:'var(--danger-dim)'}}>
    <div style={{color:'var(--danger)',fontWeight:650}}>{title}</div>
    {hint?<div style={{color:'var(--text-secondary)',fontSize:12.5,marginTop:'var(--s1)'}}>{hint}</div>:null}
    {onRetry?<button onClick={onRetry} style={{marginTop:'var(--s3)',padding:'6px 12px',
      borderRadius:'var(--r-sm)',border:'1px solid var(--border-strong)',
      background:'var(--bg-raised)',color:'var(--text-primary)',cursor:'pointer'}}>{retryLabel}</button>:null}
  </div>;
}
export function Skeleton({lines=3}:{lines?:number}){
  return <div aria-busy="true" aria-hidden="true">
    {Array.from({length:lines},(_,i)=><div key={i} style={{height:14,
      margin:'10px 0',borderRadius:7,
      background:'linear-gradient(90deg,#16223a 25%,#1e2c49 50%,#16223a 75%)',
      backgroundSize:'200% 100%',animation:'tally-shimmer 1.2s infinite'}}/>)}
    <style>{'@keyframes tally-shimmer{0%{background-position:200% 0}100%{background-position:-200% 0}}'}</style>
  </div>;
}
export function Spinner({inline=false,label='loading'}:{inline?:boolean;label?:string}){
  return <>
    <span role="status" aria-label={label} style={{
      display:'inline-flex',width:inline?14:22,height:inline?14:22,
      marginRight:inline?6:0,borderRadius:'50%',
      border:inline?'2px solid #ffffff33':'3px solid #ffffff2a',
      borderTopColor:'var(--brand)',animation:'tally-spin .8s linear infinite'}}/>
    <style>{'@keyframes tally-spin{to{transform:rotate(360deg)}}'}</style>
  </>;
}
