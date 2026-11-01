import {useState,type ReactNode} from 'react';

export interface TabItem{key:string;label:ReactNode;content:ReactNode;}
/** Roving-tabindex tabs: arrow-key navigation, Home/End, ARIA wiring. */
export function Tabs({items,initial,ariaLabel='tabs'}:
  {items:readonly TabItem[];initial?:string;ariaLabel?:string}){
  const [active,setActive]=useState(initial??items[0]?.key??'');
  const current=items.find(item=>item.key===active)??items[0];
  const onKeyDown=(event:React.KeyboardEvent)=>{
    const index=items.findIndex(item=>item.key===current?.key);
    if(index<0)return;
    let next=-1;
    if(event.key==='ArrowRight')next=(index+1)%items.length;
    else if(event.key==='ArrowLeft')next=(index-1+items.length)%items.length;
    else if(event.key==='Home')next=0;
    else if(event.key==='End')next=items.length-1;
    if(next>=0){event.preventDefault();
      setActive(items[next].key);
      document.getElementById(`tab-${ariaLabel}-${items[next].key}`)?.focus();}
  };
  return <div>
    <div role="tablist" aria-label={ariaLabel} onKeyDown={onKeyDown}
      style={{display:'flex',gap:'var(--s2)',borderBottom:'1px solid var(--border-subtle)',
        marginBottom:'var(--s4)',overflowX:'auto'}}>
      {items.map(item=>{
        const selected=item.key===current?.key;
        return <button key={item.key} id={`tab-${ariaLabel}-${item.key}`}
          role="tab" aria-selected={selected} tabIndex={selected?0:-1}
          onClick={()=>setActive(item.key)}
          style={{padding:'9px 14px',border:0,borderBottom:selected
            ?'2px solid var(--brand)':'2px solid transparent',background:'transparent',
            color:selected?'var(--text-primary)':'var(--text-muted)',font:'inherit',
            fontWeight:selected?650:400,cursor:'pointer',whiteSpace:'nowrap'}}>
          {item.label}
        </button>;})}
    </div>
    <div role="tabpanel" aria-labelledby={`tab-${ariaLabel}-${current?.key??''}`}
      tabIndex={0}>{current?.content}</div>
  </div>;
}
