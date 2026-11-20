import {useId,useState,type ReactNode} from 'react';

export function Collapse({title,defaultOpen=true,children}:
  {title:ReactNode;defaultOpen?:boolean;children:ReactNode}){
  const [open,setOpen]=useState(defaultOpen);
  const id=useId();
  return <div>
    <button onClick={()=>setOpen(v=>!v)} aria-expanded={open} aria-controls={id}
      style={{display:'flex',alignItems:'center',gap:'var(--s2)',width:'100%',
        padding:'var(--s2) 0',border:0,background:'transparent',color:'var(--text-secondary)',
        font:'inherit',fontWeight:650,cursor:'pointer',textAlign:'left'}}>
      <span aria-hidden="true" style={{transform:open?'rotate(90deg)':'none',
        transition:'transform .12s ease',color:'var(--text-faint)'}}>▸</span>
      {title}
    </button>
    {open?<div id={id}>{children}</div>:null}
  </div>;
}
