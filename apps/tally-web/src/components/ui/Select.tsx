import {useId} from 'react';

export interface SelectOption{value:string;label:string;disabled?:boolean;}
export function Select({label,value,options,onChange,placeholder,disabled}:
  {label?:string;value:string|null;options:readonly SelectOption[];
   onChange:(value:string)=>void;placeholder?:string;disabled?:boolean}){
  const id=useId();
  // #449 U4: no inline min-width floor here — .topbar .select-field lets the
  // picker shrink under flex pressure (sole mount: TopBar) instead of forcing
  // the shell's implicit grid track (and the document) wider than the viewport.
  return <div className="select-field" style={{display:'grid',gap:'var(--s1)'}}>
    {label?<label htmlFor={id} className="sr-only">{label}</label>:null}
    <select id={id} value={value??''} disabled={disabled}
      onChange={event=>onChange(event.target.value)}
      style={{background:'var(--bg-app)',border:'1px solid var(--border-strong)',
        borderRadius:'var(--r-sm)',color:'var(--text-primary)',padding:'7px 10px',
        font:'inherit',cursor:disabled?'not-allowed':'pointer',maxWidth:220}}>
      {placeholder!==undefined?
        <option value="" disabled={value!==null&&value!==''}>{placeholder}</option>:null}
      {options.map(option=><option key={option.value} value={option.value}
        disabled={option.disabled}>{option.label}</option>)}
    </select>
  </div>;
}
