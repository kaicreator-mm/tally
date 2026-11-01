import {useShell} from './AppContext.tsx';

/** The bell is a shortcut to the SAME human-action queue as the workbench
 * inbox (Owner addendum) — one queue, two entrances, never two sources. */
export function InboxPanel({open,onToggle,onNavigate}:
  {open:boolean;onToggle:()=>void;onNavigate:(hash:string)=>void}){
  const {t}=useShell();
  return <>
    <button onClick={onToggle} aria-label={t('top.notifications')}
      aria-expanded={open} title={t('top.notifications')}
      style={{background:'transparent',border:'1px solid transparent',borderRadius:'var(--r-md)',
        color:'var(--text-secondary)',cursor:'pointer',fontSize:16,padding:'6px 8px'}}>🔔</button>
    {open?<div role="menu" style={{position:'absolute',top:'calc(var(--topbar-h) - 6px)',
      right:'var(--s4)',width:'min(360px,92vw)',background:'var(--bg-panel)',
      border:'1px solid var(--border-strong)',borderRadius:'var(--r-lg)',
      boxShadow:'var(--shadow-2)',padding:'var(--s4)',zIndex:'var(--z-modal)'}}>
      <div style={{fontWeight:650,marginBottom:'var(--s2)'}}>{t('nav.inbox')}</div>
      <button role="menuitem" onClick={()=>onNavigate('#/workbench/inbox')}
        style={{width:'100%',textAlign:'left',padding:'8px 10px',border:0,borderRadius:'var(--r-sm)',
          background:'transparent',color:'var(--brand-strong)',cursor:'pointer',font:'inherit'}}>
        {t('nav.inbox')} · 查看受信待办队列
      </button>
    </div>:null}
  </>;
}
