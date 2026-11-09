/** UI-A #228 — navigation IA model: four collapsible top groups with the
 * Owner-mandated structure (2026-11-01). Project/version selection lives in
 * the TOP-RIGHT header only (Owner directive: no duplicate sidebar entries).
 * Feature views are owned by lanes UI-B/C/D/F/G/H and integrated by UI-E;
 * this model maps every item to a route slot without importing them.
 */
export interface NavItem{
  key:string;route:string;
  /** Owning lane issue; used for honest "not integrated" placeholders. */
  lane:number;
  /** Requires a selected project context (version-level items also require a
   * selected version); keeps strict Project→Version→Task three-level rule. */
  scope:'global'|'project'|'version';
}
export interface NavGroup{key:string;items:NavItem[];}
export const NAV_GROUPS:readonly NavGroup[]=Object.freeze([
  {key:'workbench',items:[
    {key:'inbox',route:'/workbench/inbox',lane:228,scope:'global'},
    {key:'approvals',route:'/workbench/approvals',lane:228,scope:'global'},
    {key:'blockers',route:'/workbench/blockers',lane:228,scope:'global'},
    {key:'overview',route:'/workbench/overview',lane:228,scope:'project'},
  ]},
  {key:'development',items:[
    {key:'requirements',route:'/development/requirements',lane:231,scope:'version'},
    {key:'product',route:'/development/product',lane:231,scope:'version'},
    {key:'l2-dag',route:'/development/l2-dag',lane:231,scope:'version'},
    {key:'tasks',route:'/development/tasks',lane:230,scope:'version'},
    {key:'agents',route:'/development/agents',lane:230,scope:'version'},
    {key:'qa',route:'/development/qa',lane:231,scope:'version'},
    {key:'release',route:'/development/release',lane:231,scope:'version'},
    {key:'deployment',route:'/development/deployment',lane:231,scope:'version'},
    {key:'maintenance',route:'/development/maintenance',lane:231,scope:'version'},
  ]},
  {key:'governance',items:[
    {key:'metadata',route:'/governance/metadata',lane:233,scope:'version'},
    {key:'workflow',route:'/governance/workflow',lane:233,scope:'global'},
    {key:'domain-packages',route:'/governance/domain-packages',lane:233,scope:'global'},
    {key:'experience',route:'/governance/experience',lane:233,scope:'global'},
    {key:'audit',route:'/governance/audit',lane:233,scope:'global'},
  ]},
  {key:'settings',items:[
    {key:'agent-providers',route:'/settings/agent-providers',lane:234,scope:'global'},
    {key:'tally-llm',route:'/settings/tally-llm',lane:234,scope:'global'},
    {key:'git-github',route:'/settings/git-github',lane:234,scope:'global'},
    {key:'ci',route:'/settings/ci',lane:234,scope:'global'},
    {key:'host',route:'/settings/host',lane:234,scope:'global'},
    {key:'storage',route:'/settings/storage',lane:234,scope:'global'},
    {key:'security',route:'/settings/security',lane:234,scope:'global'},
    {key:'deployment-env',route:'/settings/deployment-env',lane:234,scope:'global'},
    {key:'integrations',route:'/settings/integrations',lane:234,scope:'global'},
  ]},
]);
/** Strict three-level rule: an item unavailable for the current context is
 * returned with the missing scope so the sidebar can explain (not hide
 * silently) why it cannot open yet. */
export interface NavAvailability{item:NavItem;available:boolean;missing:'project'|'version'|null;}
export function navAvailability(item:NavItem,
  context:{projectId?:string|null;versionId?:string|null}):NavAvailability{
  if(item.scope==='global')return {item,available:true,missing:null};
  if(item.scope==='project')
    return context.projectId?{item,available:true,missing:null}
      :{item,available:false,missing:'project'};
  return context.projectId&&context.versionId?
      {item,available:true,missing:null}
    :{item,available:false,missing:context.projectId?'version':'project'};
}
export function findItemByRoute(route:{group:string;view:string;query?:unknown}):NavItem|null{
  const path=`/${route.group}/${route.view}`;
  for(const group of NAV_GROUPS)
    for(const item of group.items)
      if(item.route===path)return item;
  return null;
}
export function groupKeyOf(item:NavItem):string|null{
  for(const group of NAV_GROUPS)
    if(group.items.includes(item))return group.key;
  return null;
}
