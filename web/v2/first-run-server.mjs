/** Unprivileged loopback Setup service; always separate from the Host authority. */
import http from 'node:http';
import {randomBytes} from 'node:crypto';
import {homedir} from 'node:os';
import {join} from 'node:path';
import {readFile} from 'node:fs/promises';
import {LocalProjectCatalog} from './local-project-catalog.mjs';
import {createGitHubTaskDraft} from './draft-github-issue.mjs';

const html=String.raw`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Tally — 配置项目</title></head><body>
<main style="max-width:700px;margin:2rem auto;padding:1rem;font:16px system-ui">
<h1>Tally · 项目配置</h1>
<p>连接 GitHub 仓库和开发分支。SHA 自动从 GitHub 读取；Token 只保存在当前 Windows 用户的加密凭据文件中。</p>
<form id="add"><label>GitHub 仓库 (owner/repo)<br><input name="repository" required
placeholder="kaicreator-mm/tally-reps-test" style="width:100%;padding:8px"></label>
<p><label>开发分支<br><input name="ref" value="main" required></label></p>
<p><label>GitHub Token（仅需要仓库读取权限）<br>
<input name="token" type="password" autocomplete="new-password" style="width:100%;padding:8px"></label></p>
<button type="submit">验证连接并保存项目</button></form>
<hr><h2>已有项目</h2><div id="projects" aria-live="polite"></div>
<section id="draft-panel" hidden>
<h2>创建 GitHub 开发任务草稿</h2>
<p>仅创建当前项目的 GitHub Issue。它尚不是已冻结/已授权的 Tally Task，
不会自动执行 Agent。</p>
<form id="draft">
<label>任务标题<br><input name="title" required minlength="3" maxlength="180"
style="width:100%;padding:8px"></label>
<p><label>任务说明<br><textarea name="description" rows="5"
style="width:100%;padding:8px"></textarea></label></p>
<p><label>Issues:write Token（一次性使用，不保存）<br>
<input name="writeToken" type="password" required autocomplete="new-password"
style="width:100%;padding:8px"></label></p>
<button type="submit">在当前项目创建 Task 草稿 Issue</button></form>
<div id="draft-result" aria-live="polite"></div></section>
<p id="status" role="status" aria-live="polite"></p>
</main><script type="module" src="/setup-client.mjs"></script></body></html>`;
const body=async req=>{
  let data='';
  for await(const chunk of req){
    data+=chunk;
    if(data.length>65536)throw new Error('SETUP_BODY_TOO_LARGE');
  }
  try{return JSON.parse(data);}catch{throw new Error('SETUP_INVALID_JSON');}
};
const denial=reason=>({status:'DENIED',reason});
const codeOf=error=>{
  const message=error?.message??'';
  return /^[A-Z][A-Z0-9_]{2,90}$/.test(message)?message:'SETUP_OPERATION_FAILED';
};
export async function startFirstRunServer({
  env=process.env,startConfigured,catalog,port=0,
  createDraft=createGitHubTaskDraft
}={}){
  if(typeof startConfigured!=='function')throw new Error('START_CONFIGURED_REQUIRED');
  const dataDir=env.TALLY_V2_CONFIG_DIR??
    join(env.LOCALAPPDATA??join(homedir(),'.tally'),'Tally','v2');
  const store=catalog??new LocalProjectCatalog({dir:dataDir});
  const key=randomBytes(24).toString('hex');
  let origin='',active=null,activeSource=null,busy=false,lastError=null;
  const submittedDrafts=new Set();
  const activate=async ({settings,identity,legacy=false})=>{
    // The selected UI project is NOT itself a Host grant. Preserve existing
    // operator-pinned effects only for the exact same authorized source;
    // every other browser selection receives source credentials only.
    const matchesHostSource=
      env.TALLY_V2_REPOSITORY===settings.TALLY_V2_REPOSITORY&&
      env.TALLY_V2_VERSION_REF===settings.TALLY_V2_VERSION_REF&&
      env.TALLY_V2_VERSION_SHA===settings.TALLY_V2_VERSION_SHA;
    const next=await startConfigured({env:legacy||matchesHostSource?
      {...env,...settings,TALLY_SETUP_URL:origin+'/setup'}:
      {...settings,TALLY_SETUP_URL:origin+'/setup'}});
    const old=active;active=next;activeSource=identity;lastError=null;
    if(old)await old.close().catch(()=>{});
    return next.address.origin+'/app';
  };
  const server=http.createServer(async(req,res)=>{
    const response=(status,type,data)=>{
      res.writeHead(status,{'content-type':type,'cache-control':'no-store',
        'x-content-type-options':'nosniff','content-security-policy':
        "default-src 'none'; style-src 'unsafe-inline'; script-src 'self'; connect-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"});
      res.end(data);
    };
    const json=(code,value)=>response(code,'application/json',JSON.stringify(value));
    // DNS-rebinding and local CSRF guard. Setup never accepts remote origins.
    if(req.headers.host!==new URL(origin).host)
      return json(403,denial('SETUP_HOST_MISMATCH'));
    const url=new URL(req.url,origin);
    if(req.method==='GET'&&url.pathname==='/setup-client.mjs')
      return response(200,'text/javascript',await readFile(
        new URL('./setup-client.mjs',import.meta.url)));
    if(req.method==='GET'&&(url.pathname==='/setup'||url.pathname==='/setup/'))
      return response(200,'text/html; charset=utf-8',
        html.replace('<script type="module"',
          '<meta name="tally-setup-key" content="'+key+'"><script type="module"'));
    if(req.method==='GET'&&url.pathname==='/api/setup/status'){
      return json(200,{...(await store.status()),activeUrl:active?.address.origin+'/app'??null,
        activeSource,lastError});
    }
    if(req.method==='GET'&&(url.pathname==='/'||url.pathname==='/app')){
      res.writeHead(302,{location:active?active.address.origin+'/app':'/setup'});
      return res.end();
    }
    if(req.method==='POST'&&
      (url.pathname==='/api/setup/add'||url.pathname==='/api/setup/select'||
       url.pathname==='/api/setup/task-draft')){
      if(req.headers.origin!==origin||req.headers['x-tally-setup-key']!==key||
        req.headers['content-type']?.split(';')[0]!=='application/json')
        return json(403,denial('SETUP_CSRF_OR_ORIGIN'));
      if(busy)return json(409,denial('SETUP_BUSY'));
      busy=true;
      try{
        const input=await body(req);
        if(url.pathname==='/api/setup/task-draft'){
          const current=(await store.status()).selected;
          if(!current||current.repository!==input.repository||
              current.ref!==input.ref)
            throw new Error('SELECTED_PROJECT_MISMATCH');
          if(typeof input.requestId!=='string'||
              !/^[a-zA-Z0-9-]{8,80}$/.test(input.requestId))
            throw new Error('INVALID_DRAFT_TASK_INPUT');
          if(submittedDrafts.has(input.requestId))
            throw new Error('DUPLICATE_DRAFT_REQUEST');
          submittedDrafts.add(input.requestId);
          const selected=await store.select(current);
          const result=await createDraft({
            repository:current.repository,ref:current.ref,
            expectedSha:selected.env.TALLY_V2_VERSION_SHA,
            title:input.title,description:input.description,
            writeToken:input.writeToken,requestId:input.requestId});
          return json(201,result);
        }
        let status;
        if(url.pathname==='/api/setup/add')
          status=await store.add({repository:input.repository,ref:input.ref,
            token:input.token||undefined});
        else status=await store.status();
        const selected=url.pathname==='/api/setup/add'?status.selected:
          {repository:input.repository,ref:input.ref};
        const chosen=await store.select(selected);
        const activeUrl=await activate({settings:chosen.env,
          identity:{repository:selected.repository,ref:selected.ref,
            sha:chosen.env.TALLY_V2_VERSION_SHA}});
        return json(200,{status:'CONNECTED',activeUrl,
          ...(await store.status())});
      }catch(error){
        lastError=codeOf(error);
        return json(422,denial(lastError));
      }finally{busy=false;}
    }
    return json(404,denial('NOT_FOUND'));
  });
  await new Promise(resolve=>server.listen(port,'127.0.0.1',resolve));
  origin='http://127.0.0.1:'+server.address().port;
  // Existing environment settings still work, but setup remains available
  // to add/switch projects after startup. Partial explicit pins fail closed.
  if(env.TALLY_V2_REPOSITORY!==undefined){
    try{
      await activate({settings:{},legacy:true,identity:{
        repository:env.TALLY_V2_REPOSITORY,
        ref:env.TALLY_V2_VERSION_REF,sha:env.TALLY_V2_VERSION_SHA}});
    }catch(error){lastError=codeOf(error);}
  }else{
    try{
      const chosen=await store.selected();
      if(chosen)await activate({settings:chosen.env,identity:{
        repository:chosen.env.TALLY_V2_REPOSITORY,
        ref:chosen.env.TALLY_V2_VERSION_REF,
        sha:chosen.env.TALLY_V2_VERSION_SHA}});
    }catch(error){lastError=codeOf(error);}
  }
  return Object.freeze({address:{origin,appUrl:origin+'/app'},
    close:async()=>{
      if(active)await active.close().catch(()=>{});
      await new Promise(resolve=>server.close(resolve));
    }});
}
