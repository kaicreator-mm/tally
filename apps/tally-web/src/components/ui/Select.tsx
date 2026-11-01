import {useId} from 'react';

export interface SelectOption{value:string;label:string;disabled?:boolean;}
export function Select({label,value,options,onChange,placeholder,disabled}:
  {label?:string;value:string|null;options:readonly SelectOption[];
   onChange:(value:string)=>void;placeholder?:string;disabled?:boolean}){
  const id=useId();
  return <div style={{display:'grid',gap:'var(--s1)',minWidth:120}}>
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
