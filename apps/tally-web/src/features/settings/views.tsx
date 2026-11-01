/** UI-G #234 — settings center views: sidebar sections with bounded
 * state-driven forms (UI-B concepts), masked secrets, honest connection
 * state. Agent providers and Tally LLM are strictly separate sections. */
import {useState} from 'react';
import type {SettingsSnapshot,SettingsSection,AgentProviderConfig} from './model.ts';
import {SETTINGS_SECTIONS,validateEndpoint,validateConcurrency,
  canToggleProvider,validateTokenBudget} from './model.ts';
import {Badge} from '../../components/ui/Badge.tsx';
import {Card,CardHeader} from '../../components/ui/Card.tsx';
import {Input} from '../../components/ui/Input.tsx';
import {Button} from '../../components/ui/Button.tsx';
import {EmptyState} from '../../components/ui/States.tsx';

const SECTION_LABEL:Record<SettingsSection,{zh:string;en:string}>={
  agentProviders:{zh:'Agent 执行配置',en:'Agent providers'},
  tallyLlm:{zh:'Tally LLM(与 Agent 严格分离)',en:'Tally LLM (separate)'},
  ci:{zh:'Woodpecker CI',en:'Woodpecker CI'},
  gitGithub:{zh:'Git / GitHub / ADS',en:'Git / GitHub / ADS'},
  deployment:{zh:'部署环境',en:'Deployment environments'},
  host:{zh:'Host / 运行时',en:'Host / runtime'},
  storage:{zh:'存储与缓存',en:'Storage & cache'},
  security:{zh:'安全与权限',en:'Security & access'},
  integrations:{zh:'集成与通知',en:'Integrations & notifications'},
  defaults:{zh:'工作区默认值',en:'Workspace defaults'},
};
export function SettingsCenter({snapshot,connected=true,locale='zh',
  onTestConnection,selectedSection}:
  {snapshot:SettingsSnapshot|null;connected?:boolean;locale?:'zh'|'en';
   selectedSection?:SettingsSection;
   onTestConnection?:(section:SettingsSection)=>void}){
  const zh=locale==='zh';
  const [internalSection,setInternalSection]=useState<SettingsSection>('agentProviders');
  const section=selectedSection??internalSection;
  if(!connected)return <EmptyState title={zh?'未连接真实数据源':'Not connected'}
    hint="UI-E #232"/>;
  if(!snapshot)return <EmptyState title={zh?'配置快照不可用':'Snapshot unavailable'}/>;
  return <div style={{display:'grid',
    gridTemplateColumns:selectedSection?'minmax(0,1fr)':'200px minmax(0,1fr)',
    gap:'var(--s5)',alignItems:'start'}}>
    {!selectedSection?<nav aria-label={zh?'配置导航':'settings nav'}
      style={{display:'grid',gap:2}}>
      {SETTINGS_SECTIONS.map(key=><button key={key}
        aria-current={section===key?'true':undefined}
        onClick={()=>setInternalSection(key)}
        style={{textAlign:'left',padding:'9px 12px',border:0,borderRadius:'var(--r-sm)',
          background:section===key?'var(--info-dim)':'transparent',
          color:section===key?'var(--text-primary)':'var(--text-secondary)',
          cursor:'pointer',font:'inherit',fontWeight:section===key?600:400}}>
        {zh?SECTION_LABEL[key].zh:SECTION_LABEL[key].en}</button>)}
    </nav>:null}
    <div style={{display:'grid',gap:'var(--s5)'}}>
      {section==='agentProviders'
        ?<AgentProvidersSection providers={snapshot.agentProviders} zh={zh}/>
        :section==='tallyLlm'
          ?<TallyLlmSection snapshot={snapshot} zh={zh}/>
          :section==='ci'
            ?<CiSection snapshot={snapshot} zh={zh}
                onTest={()=>onTestConnection?.('ci')} available={Boolean(onTestConnection)}/>
            :section==='gitGithub'
              ?<GitSection snapshot={snapshot} zh={zh}/>
              :section==='deployment'
                ?<DeploymentSection snapshot={snapshot} zh={zh}/>
                :section==='defaults'
                  ?<DefaultsSection snapshot={snapshot} zh={zh}/>
                  :<EmptyState title={zh?SECTION_LABEL[section].zh:SECTION_LABEL[section].en}
                    hint={zh?'NOT_CONNECTED · 需要受信 Host 的配置读模型和权限校验'
                      :'NOT_CONNECTED · trusted Host settings adapter required'}/>}
    </div>
  </div>;
}
function AgentProvidersSection({providers,zh}:
  {providers:readonly AgentProviderConfig[];zh:boolean}){
  return <Card>
    <CardHeader title={zh?'外部 Agent 执行器(与 Tally LLM 分离)':'Agent executors'}
      sub={zh?'仅 BUILD 选中的 Provider 可启用;密钥仅存于 Host 凭证库'
        :'BUILD-selected only; secrets stay in the Host store'}/>
    {providers.map(provider=><ProviderRow key={provider.id} provider={provider}
      zh={zh}/>)}
  </Card>;
}
function ProviderRow({provider,zh}:{provider:AgentProviderConfig;zh:boolean}){
  const enabled=provider.enabled;
  const toggle=canToggleProvider(provider,!enabled);
  return <div style={{display:'grid',gap:'var(--s2)',padding:'var(--s4) 0',
    borderBottom:'1px solid var(--border-subtle)'}}>
    <div style={{display:'flex',gap:'var(--s3)',alignItems:'center',flexWrap:'wrap'}}>
      <strong>{provider.id}</strong>
      <Badge tone={provider.buildSelected?'ok':'warn'}>
        {provider.buildSelected?'BUILD 已选中':'BUILD 未选中'}</Badge>
      <Badge tone={enabled?'ok':'neutral'}>{enabled?'启用':'停用'}</Badge>
      <label style={{display:'flex',gap:6,alignItems:'center',fontSize:13}}>
        <input type="checkbox" checked={enabled} disabled
          title="HOST_CONFIGURATION_WRITE_NOT_CONNECTED"
          onChange={()=>{}}/>
        {zh?'启用':'Enable'}</label>
      <span style={{color:'var(--text-faint)',fontSize:12}}>
        READ_ONLY · HOST_CONFIGURATION_WRITE_NOT_CONNECTED</span>
      {!toggle.ok&&provider.buildSelected===false?
        <span style={{color:'var(--warn)',fontSize:12}}>{toggle.reason}</span>:null}
    </div>
    <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'2px 12px',
      margin:0,fontSize:12.5,color:'var(--text-secondary)'}}>
      <dt>CLI</dt><dd style={{margin:0}}>{provider.cliPath??'—'}</dd>
      <dt>{zh?'主机主体':'Host principal'}</dt>
      <dd style={{margin:0}}>{provider.hostPrincipal??'—'}</dd>
      <dt>{zh?'并发':'Concurrency'}</dt>
      <dd style={{margin:0}}>{provider.concurrency}
        {validateConcurrency(provider.concurrency)!=='ok'
          ?<Badge tone="danger">out_of_range</Badge>:''}</dd>
      <dt>{zh?'凭证':'Credential'}</dt>
      <dd style={{margin:0}}>{provider.credential
        ?<span><code>{provider.credential.targetId}</code> ·
          <span style={{color:'var(--text-faint)'}}>
            {provider.credential.maskedPreview??'MASKED'}</span> ·
          {provider.credential.lastVerifiedAt??zh?'未验证':'unverified'}</span>
        :zh?'未配置':'none'}</dd>
    </dl>
  </div>;
}
function TallyLlmSection({snapshot,zh}:{snapshot:SettingsSnapshot;zh:boolean}){
  const llm=snapshot.tallyLlm;
  const [endpoint,setEndpoint]=useState(llm.endpoint??'');
  const endpointCheck=endpoint===''?null:validateEndpoint(endpoint);
  const [budget,setBudget]=useState(llm.tokenBudget===null?''
    :String(llm.tokenBudget));
  const budgetCheck=budget===''?null
    :validateTokenBudget(Number(budget));
  return <Card>
    <CardHeader title={zh?'Tally 自身 LLM(用于澄清/规划)':'Tally-owned LLM'}
      sub={zh?'与 Agent 执行配置严格分离;停用不影响任务管理'
        :'strictly separate from agent providers'}/>
    <div style={{display:'grid',gap:'var(--s4)',maxWidth:480}}>
      <label style={{display:'flex',gap:6,alignItems:'center'}}>
        <input type="checkbox" checked={llm.enabled} readOnly/>
        {zh?'启用(需 BUILD 选中 Provider)':'Enabled (BUILD-selected provider only)'}</label>
      <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'2px 12px',
        margin:0,fontSize:13}}>
        <dt>Provider</dt><dd style={{margin:0}}>{llm.provider??'—'}</dd>
        <dt>Model</dt><dd style={{margin:0}}>{llm.model??'—'}</dd>
        <dt>{zh?'提示词模板版本':'Prompt template'}</dt>
        <dd style={{margin:0}}>{llm.promptTemplateVersion??'—'}</dd>
        <dt>{zh?'隐私/同意预览':'Privacy consent preview'}</dt>
        <dd style={{margin:0}}><Badge tone={llm.privacyConsentPreviewShown?'ok':'warn'}>
          {llm.privacyConsentPreviewShown?'已展示':'未展示'}</Badge></dd>
      </dl>
      <Input label={zh?'端点':'Endpoint'} value={endpoint} disabled
        onChange={event=>setEndpoint(event.target.value)}
        error={endpointCheck&&endpointCheck!=='ok'
          ?(endpointCheck==='embedded_credentials'
            ?'URL 中不得内嵌凭证':'URL 无效'):null}
        hint={endpointCheck==='ok'?'ok':undefined}/>
      <Input label={zh?'Token 预算':'Token budget'} value={budget} disabled
        onChange={event=>setBudget(event.target.value)}
        error={budgetCheck&&budgetCheck!=='ok'?'超出范围':null}
        hint={zh?'留空 = 不限制(显示 UNKNOWN)':'empty = unlimited (UNKNOWN)'}/>
      <div style={{color:'var(--text-faint)',fontSize:12}}>
        {zh?'密钥仅由 Host 持有(当前引用':'Secrets are Host-only (ref '}
        <code>{llm.credential?.targetId??'—'}</code>
        {zh?',界面永不明文/存储)':' — never shown or stored)'}
      </div>
    </div>
  </Card>;
}
function CiSection({snapshot,zh,onTest,available}:
  {snapshot:SettingsSnapshot;zh:boolean;onTest:()=>void;available:boolean}){
  const ci=snapshot.ci;
  return <Card>
    <CardHeader title={zh?'自建 Woodpecker(禁用 GitHub Actions)'
        :'Self-hosted Woodpecker (no GitHub Actions)'}
      sub={ci.lastReadbackAt?`${zh?'最近读回':'last readback'}: ${ci.lastReadbackAt}`
        :zh?'尚无读回':'no readback yet'}/>
    <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'2px 12px',
      margin:0,fontSize:13,marginBottom:'var(--s4)'}}>
      <dt>URL</dt><dd style={{margin:0}}>{ci.woodpeckerUrl??'—'}</dd>
      <dt>Repo ID</dt><dd style={{margin:0}}>{ci.repoId??'—'}</dd>
      <dt>{zh?'Profiles':'Profiles'}</dt>
      <dd style={{margin:0}}>{ci.profiles.join(', ')||'—'}</dd>
      <dt>Webhook</dt><dd style={{margin:0}}><Badge tone={ci.webhookState==='OK'
        ?'ok':'warn'}>{ci.webhookState}</Badge></dd>
    </dl>
    <Button onClick={onTest} disabled={!available}>
      {zh?'通过 Host 适配器测试连接':'Test via Host adapter'}</Button>
    {!available?<p style={{color:'var(--text-muted)',fontSize:12}}>
      HOST_TEST_CONNECTION_NOT_CONNECTED</p>:null}
  </Card>;
}
function GitSection({snapshot,zh}:{snapshot:SettingsSnapshot;zh:boolean}){
  const git=snapshot.git;
  return <Card>
    <CardHeader title={zh?'Git / GitHub / ADS':'Git / GitHub / ADS'}
      sub={zh?'OAuth scope 与 ADS pin 以真实读回为准':'scopes & pin per real readback'}/>
    <dl style={{display:'grid',gridTemplateColumns:'auto 1fr',gap:'2px 12px',
      margin:0,fontSize:13}}>
      <dt>{zh?'仓库':'Repositories'}</dt>
      <dd style={{margin:0}}>{git.repositories.join(', ')||'—'}</dd>
      <dt>{zh?'默认基线':'Default base'}</dt>
      <dd style={{margin:0}}>{git.defaultBase}</dd>
      <dt>ADS pin</dt><dd style={{margin:0}}><code>{git.adsPin??'—'}</code></dd>
      <dt>OAuth scopes</dt><dd style={{margin:0}}>{git.oauthScopes.join(', ')||'—'}</dd>
      <dt>{zh?'缺失 scope':'Missing scopes'}</dt>
      <dd style={{margin:0}}>{git.requiredScopeGaps.length===0
        ?<Badge tone="ok">无</Badge>
        :git.requiredScopeGaps.map(gap=><Badge key={gap} tone="warn">{gap}</Badge>)}</dd>
    </dl>
  </Card>;
}
function DeploymentSection({snapshot,zh}:{snapshot:SettingsSnapshot;zh:boolean}){
  return <Card>
    <CardHeader title={zh?'部署环境(与发布分离)':'Deployment envs (separate from release)'}/>
    {snapshot.deploymentEnvs.length===0
      ?<EmptyState title={zh?'暂无环境':'No environments'}/>:
      <table style={{width:'100%',borderCollapse:'collapse',fontSize:13}}>
        <tbody>{snapshot.deploymentEnvs.map(env=><tr key={env.id}>
          <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            <strong>{env.name}</strong> <Badge>{env.target}</Badge></td>
          <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            {env.kind??'—'}</td>
          <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            {env.healthCheckUrl??'—'}</td>
          <td style={{padding:'8px 10px',borderBottom:'1px solid var(--border-subtle)'}}>
            <Badge tone={env.rollbackSeparatedFromRelease?'ok':'warn'}>
              {zh?'回滚独立':'rollback separate'}</Badge></td>
        </tr>)}</tbody>
      </table>}
  </Card>;
}
function DefaultsSection({snapshot,zh}:{snapshot:SettingsSnapshot;zh:boolean}){
  const defaults=snapshot.defaults;
  const cascade=[['workspace',defaults.cascade.workspace],
    ['project',defaults.cascade.project],['version',defaults.cascade.version],
    ['task',defaults.cascade.task]] as const;
  return <Card>
    <CardHeader title={zh?'工作区默认值与作用域级联'
        :'Workspace defaults & scope cascade'}
      sub={zh?'工作区 → 项目覆盖 → 版本冻结 → 任务最小权限;变更显示来源与授权方'
        :'workspace → project → version → task with source/authorizer'}/>
    <ol style={{margin:0,paddingLeft:20,fontSize:13}}>
      {cascade.map(([level,value])=><li key={level}>
        <code>{level}</code>: {value??'—'}</li>)}
    </ol>
    <div style={{color:'var(--text-faint)',fontSize:12,marginTop:'var(--s3)'}}>
      {zh?'界面语言':'Locale'}: {defaults.locale} · {zh?'默认视图':'default view'}: {defaults.defaultView}
    </div>
  </Card>;
}
