import type {ReactNode} from 'react';

export interface Column<T>{key:string;header:ReactNode;
  cell:(row:T)=>ReactNode;width?:string;}
/** Minimal accessible table: sticky header, row keyboard focus, mobile
 * stacked fallback via CSS in app.css is handled by consumers when needed. */
export function DataTable<T>({rows,columns,rowKey,onRowClick,empty}:
  {rows:readonly T[];columns:readonly Column<T>[];rowKey:(row:T)=>string;
   onRowClick?:(row:T)=>void;empty?:ReactNode}){
  if(rows.length===0&&empty)return <>{empty}</>;
  return <div style={{overflowX:'auto',border:'1px solid var(--border-subtle)',
    borderRadius:'var(--r-lg)',background:'var(--bg-panel)'}}>
    <table style={{width:'100%',borderCollapse:'collapse',minWidth:640}}>
      <thead><tr>{columns.map(column=><th key={column.key} scope="col"
        style={{textAlign:'left',padding:'11px 14px',color:'var(--text-muted)',
          fontSize:11,fontWeight:650,letterSpacing:'.05em',textTransform:'uppercase',
          background:'var(--bg-raised)',borderBottom:'1px solid var(--border-subtle)',
          width:column.width}}>{column.header}</th>)}</tr></thead>
      <tbody>{rows.map(row=><tr key={rowKey(row)} tabIndex={onRowClick?0:undefined}
        onClick={onRowClick?()=>onRowClick(row):undefined}
        onKeyDown={onRowClick?event=>{
          if(event.key==='Enter'||event.key===' '){event.preventDefault();onRowClick(row);}
        }:undefined}
        style={{cursor:onRowClick?'pointer':'default',
          borderBottom:'1px solid var(--border-subtle)'}}>
        {columns.map(column=><td key={column.key} style={{padding:'12px 14px',
          fontSize:13,color:'var(--text-primary)',verticalAlign:'top'}}>
          {column.cell(row)}</td>)}
      </tr>)}</tbody>
    </table>
  </div>;
}
