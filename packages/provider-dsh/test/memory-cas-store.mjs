/**
 * Faithful in-memory test double for the public DomainHarness CAS store
 * surface used by the DSH host: get/put/list/remove with mustBeAbsent and
 * mustMatchRevision revision semantics. Production wiring uses the real
 * openSqliteCasStore (see createLocalDshHostWithDhStore); this double exists
 * only so the standalone package has zero non-npm dependencies.
 *
 * Stores are shared per path within one registry, mirroring one SQLite file.
 */
export function createMemoryCasStoreRegistry(){
  const databases=new Map();
  const conflict=code=>Object.assign(new Error(code),{code:'CAS_CONFLICT'});
  function openDatabase(path){
    if(databases.has(path))return databases.get(path);
    const rows=new Map();
    const store={
      async get(key){
        const row=rows.get(key);
        return row?{value:structuredClone(row.value),revision:row.revision}:null;
      },
      async put(key,value,options={}){
        const row=rows.get(key);
        if(options.mustBeAbsent&&row)throw conflict('CAS_KEY_EXISTS');
        if(options.mustMatchRevision!==undefined){
          if(!row||row.revision!==options.mustMatchRevision)throw conflict('CAS_REVISION_MISMATCH');
        }
        const revision=row?row.revision+1:0;
        rows.set(key,{value:structuredClone(value),revision});
        return {revision};
      },
      async list(prefix){
        return [...rows.keys()].filter(key=>key.startsWith(prefix)).sort();
      },
      async remove(key,options={}){
        const row=rows.get(key);
        if(!row)return;
        if(options.mustMatchRevision!==undefined&&row.revision!==options.mustMatchRevision)
          throw conflict('CAS_REVISION_MISMATCH');
        rows.delete(key);
      },
      async close(){}
    };
    databases.set(path,store);
    return store;
  }
  return {open:async({path})=>openDatabase(path),databases};
}
