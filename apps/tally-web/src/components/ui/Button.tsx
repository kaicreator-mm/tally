import type {ButtonHTMLAttributes,ReactNode,CSSProperties} from 'react';
import {Spinner} from './States.tsx';

export type ButtonVariant='primary'|'secondary'|'ghost'|'danger';
const variantStyle:Record<ButtonVariant,CSSProperties>={
  primary:{background:'var(--brand)',border:'1px solid var(--brand)',
    color:'var(--brand-ink)',fontWeight:650},
  secondary:{background:'var(--bg-raised)',border:'1px solid var(--border-strong)',
    color:'var(--text-primary)'},
  ghost:{background:'transparent',border:'1px solid transparent',
    color:'var(--brand-strong)'},
  danger:{background:'var(--danger-dim)',border:'1px solid #7c3a4d',
    color:'var(--danger)'},
};
export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>{
  variant?:ButtonVariant;loading?:boolean;children:ReactNode;
}
export function Button({variant='secondary',loading=false,children,
  style,disabled,...rest}:ButtonProps){
  return <button {...rest} disabled={disabled||loading}
    style={{padding:'7px 14px',borderRadius:'var(--r-md)',font:'inherit',
      cursor:disabled||loading?'not-allowed':'pointer',opacity:disabled?0.55:1,
      ...variantStyle[variant],...(style??{})}}>
    {loading?<Spinner inline/>:null}{children}
  </button>;
}
export function IconButton({label,children,...rest}:
  ButtonHTMLAttributes<HTMLButtonElement>&{label:string;children:ReactNode}){
  return <button {...rest} aria-label={label} title={label}
    style={{display:'inline-flex',alignItems:'center',justifyContent:'center',
      width:34,height:34,borderRadius:'var(--r-md)',border:'1px solid transparent',
      background:'transparent',color:'var(--text-secondary)',cursor:'pointer',
      ...(rest.style??{})}}>{children}</button>;
}
