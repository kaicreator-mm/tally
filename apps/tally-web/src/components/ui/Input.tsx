import {useId,type InputHTMLAttributes,type TextareaHTMLAttributes} from 'react';

const fieldStyle:React.CSSProperties={background:'var(--bg-app)',
  border:'1px solid var(--border-strong)',borderRadius:'var(--r-sm)',
  color:'var(--text-primary)',padding:'9px 12px',font:'inherit',width:'100%'};
// #449 U3: hint/error are programmatic descriptions of the field. The
// effective id (caller id wins, otherwise useId) drives label htmlFor and the
// generated description ids; a caller-provided aria-describedby is preserved
// and merged ahead of the generated ones; aria-invalid is forced true while
// an error is shown and never silently overridden by the rest spread.
function describedId(id:string,hint?:string|null,error?:string|null){
  return error?`${id}-error`:hint!=null?`${id}-hint`:undefined;
}
export function Input({label,error,hint,id:idProp,
  'aria-describedby':extraDescribedBy,'aria-invalid':ariaInvalid,...rest}:
  {label?:string;error?:string|null;hint?:string}&InputHTMLAttributes<HTMLInputElement>){
  const generatedId=useId();
  const id=idProp??generatedId;
  const describedBy=[extraDescribedBy,describedId(id,hint,error)]
    .filter(Boolean).join(' ')||undefined;
  return <div style={{display:'grid',gap:'var(--s1)'}}>
    {label?<label htmlFor={id} style={{color:'var(--text-secondary)',fontSize:12.5,
      fontWeight:600}}>{label}</label>:null}
    <input id={id} aria-invalid={error?true:ariaInvalid} aria-describedby={describedBy}
      {...rest} style={{...fieldStyle,borderColor:error?'#a34b5b':'var(--border-strong)'}}/>
    {error?<div id={`${id}-error`} role="alert" style={{color:'var(--danger)',fontSize:12}}>{error}</div>
      :hint?<div id={`${id}-hint`} style={{color:'var(--text-faint)',fontSize:12}}>{hint}</div>:null}
  </div>;
}
export function TextArea({label,error,hint,rows=4,id:idProp,
  'aria-describedby':extraDescribedBy,'aria-invalid':ariaInvalid,...rest}:
  {label?:string;error?:string|null;hint?:string;rows?:number}&
  TextareaHTMLAttributes<HTMLTextAreaElement>){
  const generatedId=useId();
  const id=idProp??generatedId;
  const describedBy=[extraDescribedBy,describedId(id,hint,error)]
    .filter(Boolean).join(' ')||undefined;
  return <div style={{display:'grid',gap:'var(--s1)'}}>
    {label?<label htmlFor={id} style={{color:'var(--text-secondary)',fontSize:12.5,
      fontWeight:600}}>{label}</label>:null}
    <textarea id={id} rows={rows} aria-invalid={error?true:ariaInvalid}
      aria-describedby={describedBy} {...rest}
      style={{...fieldStyle,resize:'vertical',borderColor:error?'#a34b5b':'var(--border-strong)'}}/>
    {error?<div id={`${id}-error`} role="alert" style={{color:'var(--danger)',fontSize:12}}>{error}</div>
      :hint?<div id={`${id}-hint`} style={{color:'var(--text-faint)',fontSize:12}}>{hint}</div>:null}
  </div>;
}
