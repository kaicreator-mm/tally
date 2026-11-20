import {useEffect,useRef,type ReactNode} from 'react';

/** Right-side details drawer. Focus trap + Esc close + focus restore. */
export function Drawer({open,title,onClose,width=460,children}:
  {open:boolean;title:ReactNode;onClose:()=>void;width?:number;children:ReactNode}){
  const panelRef=useRef<HTMLDivElement>(null);
  const restoreRef=useRef<HTMLElement|null>(null);
  useEffect(()=>{
    if(!open)return;
    restoreRef.current=document.activeElement as HTMLElement|null;
    panelRef.current?.focus();
    const onKey=(event:KeyboardEvent)=>{
      if(event.key==='Escape'){event.stopPropagation();onClose();return;}
      if(event.key!=='Tab')return;
      const focusables=panelRef.current?.querySelectorAll<HTMLElement>(
        'button,[href],input,select,textarea,[tabindex]:not([tabindex="-1"])');
      if(!focusables||focusables.length===0)return;
      const first=focusables[0],last=focusables[focusables.length-1];
      if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus();}
      else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus();}
    };
    document.addEventListener('keydown',onKey,true);
    return ()=>{document.removeEventListener('keydown',onKey,true);
      restoreRef.current?.focus?.();};
  },[open,onClose]);
  if(!open)return null;
  return <div style={{position:'fixed',inset:0,zIndex:'var(--z-drawer)'}}>
    <div onClick={onClose} aria-hidden="true"
      style={{position:'absolute',inset:0,background:'#03061199'}}/>
    <div ref={panelRef} role="dialog" aria-modal="true" aria-label={typeof title==='string'?title:'drawer'}
      tabIndex={-1} style={{position:'absolute',top:0,right:0,bottom:0,width:`min(${width}px,94vw)`,
        background:'var(--bg-panel)',borderLeft:'1px solid var(--border-strong)',
        boxShadow:'var(--shadow-2)',overflow:'auto',padding:'var(--s5)',outline:'none'}}>
      <div style={{display:'flex',justifyContent:'space-between',alignItems:'baseline',
        gap:'var(--s3)',marginBottom:'var(--s4)'}}>
        <div style={{fontWeight:700,fontSize:16}}>{title}</div>
        <button onClick={onClose} aria-label="close"
          style={{border:0,background:'transparent',color:'var(--text-muted)',
            cursor:'pointer',fontSize:18,lineHeight:1}}>×</button>
      </div>
      {children}
    </div>
  </div>;
}
