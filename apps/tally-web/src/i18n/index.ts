/** UI-A #228 — minimal i18n (zh-CN default, en available, locale remembered).
 * Messages are flat dotted keys; missing en falls back to zh-CN text; unknown
 * keys render the key itself so missing translations are visible, never silent.
 */
export type Locale='zh-CN'|'en';
export const LOCALES:readonly Locale[]=Object.freeze(['zh-CN','en']);
const STORAGE_KEY='tally.v2.locale';
export const messages={
  'zh-CN':{
    'app.title':'Tally v2 · ALM 工作台',
    'nav.group.workbench':'工作台',
    'nav.group.development':'项目研发',
    'nav.group.governance':'研发治理',
    'nav.group.settings':'配置中心',
    'nav.inbox':'人工待办',
    'nav.approvals':'审批',
    'nav.blockers':'阻塞',
    'nav.overview':'项目概览',
    'nav.requirements':'需求',
    'nav.product':'产品 / PRD',
    'nav.l2-dag':'L2 与任务规划',
    'nav.tasks':'任务',
    'nav.agents':'Agent',
    'nav.qa':'评审与 QA',
    'nav.release':'发布',
    'nav.deployment':'部署',
    'nav.maintenance':'维护与经验',
    'nav.metadata':'工程元数据',
    'nav.workflow':'Workflow',
    'nav.domain-packages':'Domain Packages',
    'nav.experience':'经验与改进',
    'nav.audit':'审计与溯源',
    'nav.agent-providers':'Agent 执行配置',
    'nav.tally-llm':'Tally LLM',
    'nav.git-github':'Git / GitHub / ADS',
    'nav.ci':'Woodpecker CI',
    'nav.host':'Host',
    'nav.storage':'SQLite / 缓存',
    'nav.security':'安全',
    'nav.deployment-env':'部署环境',
    'nav.integrations':'集成',
    'top.search':'搜索任务、需求、报告…',
    'top.notifications':'通知(与工作台待办同队列)',
    'top.project':'项目',
    'top.version':'版本',
    'top.select-project':'选择项目',
    'top.select-version':'选择版本',
    'top.version-follows-project':'先选择项目',
    'state.not-integrated':'该模块由 #{lane} 交付,尚未集成(UI-E #232)',
    'state.need-project':'请先在右上角选择项目',
    'state.need-version':'请先在右上角选择版本',
    'state.empty':'暂无数据',
    'state.loading':'加载中…',
    'state.error':'加载失败',
    'state.not-connected':'未连接真实数据源',
    'common.retry':'重试',
    'common.cancel':'取消',
    'common.confirm':'确认',
    'common.back':'返回',
    'common.more':'更多',
    'common.save':'保存',
    'inbox.empty':'没有等待你处理的事项',
  },
  'en':{
    'app.title':'Tally v2 · ALM Workspace',
    'nav.group.workbench':'Workbench',
    'nav.group.development':'Development',
    'nav.group.governance':'Governance',
    'nav.group.settings':'Settings',
    'nav.inbox':'Human inbox',
    'nav.approvals':'Approvals',
    'nav.blockers':'Blockers',
    'nav.overview':'Project overview',
    'nav.requirements':'Requirements',
    'nav.product':'Product / PRD',
    'nav.l2-dag':'L2 & task planning',
    'nav.tasks':'Tasks',
    'nav.agents':'Agents',
    'nav.qa':'Review & QA',
    'nav.release':'Release',
    'nav.deployment':'Deployment',
    'nav.maintenance':'Maintenance & learning',
    'nav.metadata':'Engineering metadata',
    'nav.workflow':'Workflow',
    'nav.domain-packages':'Domain Packages',
    'nav.experience':'Experience & improvement',
    'nav.audit':'Audit & provenance',
    'nav.agent-providers':'Agent providers',
    'nav.tally-llm':'Tally LLM',
    'nav.git-github':'Git / GitHub / ADS',
    'nav.ci':'Woodpecker CI',
    'nav.host':'Host',
    'nav.storage':'SQLite / cache',
    'nav.security':'Security',
    'nav.deployment-env':'Deployment environments',
    'nav.integrations':'Integrations',
    'top.search':'Search tasks, requirements, reports…',
    'top.notifications':'Notifications (same queue as workbench inbox)',
    'top.project':'Project',
    'top.version':'Version',
    'top.select-project':'Select project',
    'top.select-version':'Select version',
    'top.version-follows-project':'Select a project first',
    'state.not-integrated':'Delivered by lane #{lane}; not integrated yet (UI-E #232)',
    'state.need-project':'Select a project in the top-right first',
    'state.need-version':'Select a version in the top-right first',
    'state.empty':'No data',
    'state.loading':'Loading…',
    'state.error':'Failed to load',
    'state.not-connected':'Not connected to a real data source',
    'common.retry':'Retry',
    'common.cancel':'Cancel',
    'common.confirm':'Confirm',
    'common.back':'Back',
    'common.more':'More',
    'common.save':'Save',
    'inbox.empty':'Nothing needs your attention',
  },
} as const;
export type MessageKey=keyof typeof messages['zh-CN'];
export function normalizeLocale(value:string|null|undefined):Locale{
  return value==='en'?'en':'zh-CN';
}
export function detectLocale(from:{
  storage?:StorageLike;}={}):Locale{
  try{
    const stored=from.storage?.getItem?.(STORAGE_KEY);
    if(stored===null||stored===undefined)return 'zh-CN';
    return normalizeLocale(stored);
  }catch{return 'zh-CN';}
}
export interface StorageLike{getItem(key:string):string|null;setItem(key:string,value:string):void;}
export function rememberLocale(storage:StorageLike|undefined,locale:Locale):void{
  try{storage?.setItem?.(STORAGE_KEY,locale);}catch{/* persistence is best-effort */}
}
export function formatMessage(locale:Locale,key:MessageKey,
  params?:Record<string,string|number>):string{
  const table=messages[locale]??messages['zh-CN'];
  let text=(table as Record<string,string>)[key]
    ??(messages['zh-CN'] as Record<string,string>)[key]??String(key);
  if(params)for(const [name,value] of Object.entries(params))
    text=text.replaceAll(`{${name}}`,String(value));
  return text;
}
export function createT(locale:Locale){
  return (key:MessageKey,params?:Record<string,string|number>)=>
    formatMessage(locale,key,params);
}
/** Dynamic-key variant for keys composed at runtime (nav item keys etc.);
 * unknown keys still render the key itself, never silently empty. */
export function formatDyn(locale:Locale,key:string,
  params?:Record<string,string|number>):string{
  return formatMessage(locale,key as MessageKey,params);
}
