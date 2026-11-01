/** UI-G #234 — settings center view model.
 * Agent providers and the Tally-owned LLM stay STRICTLY separate sections.
 * Secrets exist only in the trusted Host/credential store: the UI holds
 * masked references (never values), never persists secrets to localStorage,
 * and test-connection runs only through the permitted Host adapter.
 */
export type SettingsSection='agentProviders'|'tallyLlm'|'ci'|'gitGithub'
  |'deployment'|'defaults';
export const SETTINGS_SECTIONS:readonly SettingsSection[]=Object.freeze(
  ['agentProviders','tallyLlm','ci','gitGithub','deployment','defaults']);
export interface SecretRef{
  /** Host-side credential target id (e.g. Windows Credential Manager key). */
  targetId:string;
  /** Masked preview issued by the Host; never the secret itself. */
  maskedPreview:string|null;
  lastVerifiedAt:string|null;
}
export interface AgentProviderConfig{
  id:'codex'|'claude-code'|'zcode';
  enabled:boolean;
  cliPath:string|null;
  hostPrincipal:string|null;
  concurrency:number;
  sandbox:string|null;
  /** Whether this provider is selected by the BUILD (UI cannot enable an
   * unselected C route). */
  buildSelected:boolean;
  credential:SecretRef|null;
}
export interface TallyLlmConfig{
  enabled:boolean;
  /** BUILD-selected public DH-compatible provider only. */
  provider:'openai'|'anthropic'|null;
  model:string|null;
  endpoint:string|null;
  promptTemplateVersion:string|null;
  privacyConsentPreviewShown:boolean;
  tokenBudget:number|null;
  credential:SecretRef|null;
}
export interface CiConfig{
  woodpeckerUrl:string|null;
  repoId:number|null;
  profiles:readonly string[];
  webhookState:'OK'|'MISSING'|'UNKNOWN';
  lastReadbackAt:string|null;
}
export interface GitConfig{
  repositories:readonly string[];
  defaultBase:'main'|'version/v2';
  adsPin:string|null;
  oauthScopes:readonly string[];
  requiredScopeGaps:readonly string[];
}
export interface DeploymentEnvConfig{
  id:string;name:string;target:'dev'|'staging'|'prod';
  kind:'host-local'|'ssh'|'docker'|'runx'|null;
  healthCheckUrl:string|null;
  rollbackSeparatedFromRelease:boolean;
}
/** Non-secret UI selections (demo/unconnected state preserved per Owner UX). */
export interface WorkspaceDefaults{
  locale:'zh-CN'|'en';
  defaultView:'list'|'board';
  /** Scope cascade snapshot for display: workspace→project→version→task. */
  cascade:{workspace:string;project:string|null;version:string|null;
    task:string|null};
}
export interface SettingsSnapshot{
  agentProviders:readonly AgentProviderConfig[];
  tallyLlm:TallyLlmConfig;
  ci:CiConfig;
  git:GitConfig;
  deploymentEnvs:readonly DeploymentEnvConfig[];
  defaults:WorkspaceDefaults;
}

/** URL validation for endpoints (no credentials embedded in URLs). */
export function validateEndpoint(url:string):'ok'|'invalid'|'embedded_credentials'{
  let parsed:URL;
  try{parsed=new URL(url);}catch{return 'invalid';}
  if(parsed.username||parsed.password)return 'embedded_credentials';
  return parsed.protocol==='https:'||parsed.protocol==='http:'
    ||parsed.protocol==='ssh:'?'ok':'invalid';
}
/** Concurrency bounds (Host contract: 1..8). */
export function validateConcurrency(value:number):'ok'|'out_of_range'{
  return Number.isSafeInteger(value)&&value>=1&&value<=8
    ?'ok':'out_of_range';
}
/** Provider enable rule: the UI may not enable a provider that the BUILD has
 * not selected (fail-closed against unselected C routes). */
export function canToggleProvider(provider:AgentProviderConfig,
  nextEnabled:boolean):{ok:boolean;reason:string|null}{
  if(nextEnabled&&!provider.buildSelected)
    return {ok:false,reason:'BUILD_NOT_SELECTED: 未在 BUILD 中选中的 Provider 不能由 UI 启用'};
  return {ok:true,reason:null};
}
/** Token budget bound (null allowed = unlimited display UNKNOWN). */
export function validateTokenBudget(value:number|null){
  if(value===null)return 'ok';
  return Number.isSafeInteger(value)&&value>=0&&value<=10_000_000
    ?'ok':'out_of_range';
}
/** Secret-leak guard: settings payloads must never contain secret values. */
export function assertNoSecrets(snapshot:SettingsSnapshot):void{
  const serialized=JSON.stringify(snapshot);
  // SecretRef carries only target ids/masked previews; raw token shapes must
  // never appear anywhere in a settings payload.
  if(/gh[pousr]_[A-Za-z0-9]{16,}|sk-[A-Za-z0-9]{16,}|github_pat_/.test(serialized))
    throw new Error('SETTINGS_CONTAIN_SECRET_BYTES');
}
/** Persistable UI state = non-secret selections only, labeled unconnected. */
export function persistSelections(defaults:WorkspaceDefaults):string{
  assertNoSecrets({agentProviders:[],tallyLlm:{enabled:false,provider:null,
    model:null,endpoint:null,promptTemplateVersion:null,
    privacyConsentPreviewShown:false,tokenBudget:null,credential:null},
    ci:{woodpeckerUrl:null,repoId:null,profiles:[],webhookState:'UNKNOWN',
      lastReadbackAt:null},git:{repositories:[],defaultBase:'version/v2',
      adsPin:null,oauthScopes:[],requiredScopeGaps:[]},deploymentEnvs:[],
    defaults});
  return JSON.stringify({kind:'tally.settings.selections.v1',
    unconnected:true,defaults});
}
/** Effective value resolution: task-specific > version frozen > project
 * override > workspace default; every level records its authorizer. */
export interface ScopedValue<T>{value:T;source:'task'|'version'|'project'|'workspace';
  authorizer:string|null;}
export function resolveScoped<T>(chain:{
  workspace:T;project?:ScopedValue<T>;version?:ScopedValue<T>;task?:ScopedValue<T>}):
  ScopedValue<T>{
  return chain.task??chain.version??chain.project??
    {value:chain.workspace,source:'workspace',authorizer:null};
}
