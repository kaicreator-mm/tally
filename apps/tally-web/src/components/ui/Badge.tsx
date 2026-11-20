import type {ReactNode} from 'react';

export type BadgeTone='neutral'|'ok'|'warn'|'danger'|'info'|'brand';
const toneStyle:Record<BadgeTone,{bg:string;fg:string;border:string}>={
  neutral:{bg:'var(--bg-raised)',fg:'var(--text-secondary)',border:'var(--border-strong)'},
  ok:{bg:'var(--ok-dim)',fg:'var(--ok)',border:'#2d7664'},
  warn:{bg:'var(--warn-dim)',fg:'var(--warn)',border:'#a2763c'},
  danger:{bg:'var(--danger-dim)',fg:'var(--danger)',border:'#a34b5b'},
  info:{bg:'var(--info-dim)',fg:'var(--info)',border:'#3777b9'},
  brand:{bg:'#16233d',fg:'var(--brand-strong)',border:'var(--border-strong)'},
};
export function Badge({tone='neutral',title,children}:
  {tone?:BadgeTone;title?:string;children:ReactNode}){
  const t=toneStyle[tone];
  return <span title={title} style={{display:'inline-flex',alignItems:'center',
    padding:'2px 9px',borderRadius:'var(--r-pill)',border:`1px solid ${t.border}`,
    background:t.bg,color:t.fg,fontSize:10.5,fontWeight:700,letterSpacing:'.04em',
    whiteSpace:'nowrap'}}>{children}</span>;
}
