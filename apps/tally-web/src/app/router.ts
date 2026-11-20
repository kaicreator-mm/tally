/** UI-A #228 — zero-dependency hash router.
 * Route shape: #/<group>/<view>?projectId=&versionId=
 * Pure logic (no DOM writes beyond the injected listener) so lane tests run
 * under plain `node --test` on Node >= 24.
 */
export interface RouteQuery {[key: string]: string;}
export interface Route{
  group:string;view:string;query:RouteQuery;
}
const SEG=/^[a-z0-9-]{1,40}$/;
export function parseHash(hash:string):Route{
  const raw=(hash??'').replace(/^#\/?/,'');
  const [pathPart,queryPart]=raw.split('?');
  const segments=(pathPart??'').split('/').filter(Boolean);
  const group=segments[0]??'workbench';
  const view=segments[1]??'index';
  if(!SEG.test(group)||!SEG.test(view))
    return {group:'workbench',view:'index',query:{}};
  const query:RouteQuery={};
  for(const pair of (queryPart??'').split('&').filter(Boolean)){
    const eq=pair.indexOf('=');
    if(eq<1)continue;
    const key=pair.slice(0,eq),value=pair.slice(eq+1);
    if(!/^[a-zA-Z][a-zA-Z0-9_]{0,30}$/.test(key))continue;
    try{query[key]=decodeURIComponent(value);}catch{query[key]=value;}
  }
  return {group,view,query};
}
export function buildHash(route:Partial<Route>&{group:string;view:string}):string{
  if(!SEG.test(route.group)||!SEG.test(route.view))
    throw new Error('ROUTE_SEGMENT_INVALID');
  const path=`/${route.group}/${route.view}`;
  const query=route.query??{};
  const pairs=Object.keys(query)
    .filter(key=>/^[a-zA-Z][a-zA-Z0-9_]{0,30}$/.test(key))
    .map(key=>`${key}=${encodeURIComponent(query[key])}`);
  return `#${path}${pairs.length?'?'+pairs.join('&'):''}`;
}
export type RouteListener=(route:Route)=>void;
export interface RouterHandle{get():Route;navigate(hash:string):void;subscribe(fn:RouteListener):()=>void;}
/** Bind to a Location-like source; `location.hash` and hashchange events. */
export function createRouter(source:{
  getHash():string;setHash(hash:string):void;
  addEventListener(type:'hashchange',fn:()=>void):void;
  removeEventListener(type:'hashchange',fn:()=>void):void;
}):RouterHandle{
  const listeners=new Set<RouteListener>();
  const emit=()=>{const route=parseHash(source.getHash());
    for(const fn of listeners)fn(route);};
  source.addEventListener('hashchange',emit);
  return Object.freeze({
    get:()=>parseHash(source.getHash()),
    navigate(hash:string){
      if(!hash.startsWith('#'))throw new Error('HASH_REQUIRED');
      if(hash===source.getHash()){emit();return;}
      source.setHash(hash);
    },
    subscribe(fn:RouteListener){listeners.add(fn);return()=>listeners.delete(fn);},
  });
}
