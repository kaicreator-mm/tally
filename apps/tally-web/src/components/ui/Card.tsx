import type {ReactNode} from 'react';

export function Card({children,style,interactive=false,...rest}:
  {children:ReactNode;style?:React.CSSProperties;interactive?:boolean}&
  React.HTMLAttributes<HTMLDivElement>){
  return <div {...rest} style={{background:'var(--bg-panel)',
    border:'1px solid var(--border-subtle)',borderRadius:'var(--r-lg)',
    boxShadow:'var(--shadow-1)',padding:'var(--s5)',
    cursor:interactive?'pointer':undefined,
    transition:'border-color .12s ease',...(style??{})}}>{children}</div>;
}
export function CardHeader({title,sub,right}:
  {title:ReactNode;sub?:ReactNode;right?:ReactNode}){
  return <div style={{display:'flex',alignItems:'baseline',justifyContent:'space-between',
    gap:'var(--s4)',marginBottom:'var(--s4)'}}>
    <div>
      <div style={{fontWeight:650,fontSize:15}}>{title}</div>
      {sub?<div style={{color:'var(--text-muted)',fontSize:12,marginTop:2}}>{sub}</div>:null}
    </div>
    {right}
  </div>;
}
