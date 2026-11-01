import {useId,type InputHTMLAttributes,type TextareaHTMLAttributes} from 'react';

const fieldStyle:React.CSSProperties={background:'var(--bg-app)',
  border:'1px solid var(--border-strong)',borderRadius:'var(--r-sm)',
  color:'var(--text-primary)',padding:'9px 12px',font:'inherit',width:'100%'};
export function Input({label,error,hint,...rest}:
  {label?:string;error?:string|null;hint?:string}&InputHTMLAttributes<HTMLInputElement>){
  const id=useId();
  return <div style={{display:'grid',gap:'var(--s1)'}}>
    {label?<label htmlFor={id} style={{color:'var(--text-secondary)',fontSize:12.5,
      fontWeight:600}}>{label}</label>:null}
    <input id={id} aria-invalid={error?true:undefined} {...rest}
      style={{...fieldStyle,borderColor:error?'#a34b5b':'var(--border-strong)'}}/>
    {error?<div role="alert" style={{color:'var(--danger)',fontSize:12}}>{error}</div>
      :hint?<div style={{color:'var(--text-faint)',fontSize:12}}>{hint}</div>:null}
  </div>;
}
export function TextArea({label,error,hint,rows=4,...rest}:
  {label?:string;error?:string|null;hint?:string;rows?:number}&
  TextareaHTMLAttributes<HTMLTextAreaElement>){
  const id=useId();
  return <div style={{display:'grid',gap:'var(--s1)'}}>
    {label?<label htmlFor={id} style={{color:'var(--text-secondary)',fontSize:12.5,
      fontWeight:600}}>{label}</label>:null}
    <textarea id={id} rows={rows} aria-invalid={error?true:undefined} {...rest}
      style={{...fieldStyle,resize:'vertical',borderColor:error?'#a34b5b':'var(--border-strong)'}}/>
    {error?<div role="alert" style={{color:'var(--danger)',fontSize:12}}>{error}</div>
      :hint?<div style={{color:'var(--text-faint)',fontSize:12}}>{hint}</div>:null}
  </div>;
}
