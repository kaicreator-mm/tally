import {useEffect,useRef,type ReactNode} from 'react';

export function Modal({open,title,onClose,children,footer}:
  {open:boolean;title:ReactNode;onClose:()=>void;children:ReactNode;footer?:ReactNode}){
  const ref=useRef<HTMLDivElement>(null);
  useEffect(()=>{
    if(!open)return;
    ref.current?.focus();
    const onKey=(event:KeyboardEvent)=>{if(event.key==='Escape')onClose();};
    document.addEventListener('keydown',onKey,true);
    return ()=>document.removeEventListener('keydown',onKey,true);
  },[open,onClose]);
  if(!open)return null;
  return <div style={{position:'fixed',inset:0,zIndex:'var(--z-modal)',
    display:'grid',placeItems:'center',padding:'var(--s4)'}}>
    <div onClick={onClose} aria-hidden="true"
      style={{position:'absolute',inset:0,background:'#030611b0'}}/>
    <div ref={ref} role="dialog" aria-modal="true" tabIndex={-1}
      style={{position:'relative',width:'min(560px,94vw)',maxHeight:'86vh',overflow:'auto',
        background:'var(--bg-panel)',border:'1px solid var(--border-strong)',
        borderRadius:'var(--r-lg)',boxShadow:'var(--shadow-2)',padding:'var(--s5)',outline:'none'}}>
      <div style={{display:'flex',justifyContent:'space-between',alignItems:'baseline',
        marginBottom:'var(--s4)',gap:'var(--s3)'}}>
        <div style={{fontWeight:700,fontSize:16}}>{title}</div>
        <button onClick={onClose} aria-label="close"
          style={{border:0,background:'transparent',color:'var(--text-muted)',cursor:'pointer',
            fontSize:18,lineHeight:1}}>×</button>
      </div>
      {children}
      {footer?<div style={{display:'flex',justifyContent:'flex-end',gap:'var(--s2)',
        marginTop:'var(--s5)'}}>{footer}</div>:null}
    </div>
  </div>;
}
