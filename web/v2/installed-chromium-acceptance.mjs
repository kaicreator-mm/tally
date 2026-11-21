/** R156: actual Chromium -> installed /app DOM/click/network/screenshots.
 * ALL model/GitHub/credential/Host authority here is explicitly MOCKED.
 * The React bundle, setup and ALM HTTP servers and local Git are real.
 */
import assert from 'node:assert/strict';
import {spawn,spawnSync} from 'node:child_process';
import {mkdtemp,mkdir,readFile,writeFile,rm,stat} from 'node:fs/promises';
import {join,dirname} from 'node:path';
import {tmpdir} from 'node:os';
import {createHash} from 'node:crypto';
import {startFirstRunServer} from './first-run-server.mjs';
import {LocalProjectCatalog} from './local-project-catalog.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';
const tmp=await mkdtemp(join(tmpdir(),'tally-r156-ui-'));
const evidence=process.env.TALLY_R156_EVIDENCE_DIR??join(process.cwd(),'_tmp','r156');
await mkdir(evidence,{recursive:true});
let app=null,chromium=null,client=null;
const clicks=[],net=[],errors=[],screens=[],mockRuns=[],checks=[];
const sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const poll=async(f,label,maxMs=16000)=>{
  const until=Date.now()+maxMs;
  while(Date.now()<until){const value=await f();if(value)return value;await sleep(130);}
  throw Error('WAIT_'+label);
};
const git=(dir,...args)=>{
  const out=spawnSync('git',['-C',dir,...args],{encoding:'utf8',timeout:12000,
    env:{...process.env,GIT_CONFIG_GLOBAL:'/dev/null',GIT_CONFIG_NOSYSTEM:'1'}});
  assert.equal(out.status,0,'GIT_'+(out.stderr??'').slice(0,150));
  return out.stdout.trim();
};
function mockRunner(repository,sourceSha){
  return async kind=>{
    if(kind!=='start')return {status:'FAIL',exitCode:1};
    const cwd=join(tmp,'repos',repository.replace('/','-'));
    const dir=join(tmp,'candidate',repository.replace('/','-'));
    await mkdir(dirname(dir),{recursive:true});
    git(cwd,'worktree','add','--detach',dir,sourceSha);
    const file=join(dir,'MOCK_CODEX_CANDIDATE.md');
    const outcome=spawnSync(process.execPath,['-e',
      "require('fs').writeFileSync(process.argv[1],'mock codex\\n')",file],
      {encoding:'utf8',timeout:12000});
    assert.equal(outcome.status,0);
    git(dir,'add','MOCK_CODEX_CANDIDATE.md');
    git(dir,'-c','user.name=Mock','-c','user.email=mock@localhost',
      'commit','-qm','mock agent');
    const commit=git(dir,'rev-parse','HEAD');
    const tree=git(dir,'rev-parse','HEAD^{tree}');
    assert.equal(git(dir,'rev-parse','HEAD^'),sourceSha);
    assert.equal(git(dir,'diff','--name-only',sourceSha,commit),
      'MOCK_CODEX_CANDIDATE.md');
    mockRuns.push({commit,tree,sourceSha,repository,realGit:true,realCodex:false});
    return {status:'PASS',exitCode:0,observedAt:new Date().toISOString(),
      evidenceDir:dir,result:{status:'PASS',realAgentEffect:true,
        claimReceipts:[],hiddenVerifier:'HIDDEN_NOT_QUALIFIED',
        installedAcceptance:{state:'NOT_RUN'},
        result:{state:'LOCAL_CANDIDATE_READY',candidateCommit:commit,candidateTree:tree}}};
  };
}
function mockModels(repository,sourceSha,setupUrl){
  const source={repository,sourceSha,currentness:'CURRENT',readback:'VERIFIED'};
  const data=extra=>({connected:true,source,versionId:'main',...extra});
  return {
    catalog:async()=>data({
      projects:[{id:repository,title:repository.split('/')[1],repository}],
      versionsByProject:{[repository]:[{id:'main',projectId:repository,label:'main'}]},
      defaultVersionId:'main'}),
    tasks:async()=>data({tasks:[{
      id:'196',versionId:'main',projectId:repository,title:'Mock Codex acceptance task',
      status:'open',severity:'P2',blockedBy:[],agentId:null,
      claimState:'ACCEPTED',attemptId:null,attemptState:null,prNumber:null,
      reviewState:'NOT_RUN',ciState:'NOT_RUN',
      writeSet:['MOCK_CODEX_CANDIDATE.md'],requirementRef:null,
      updatedAt:new Date().toISOString()}]}),
    dag:async()=>data({asOf:{sourceSha,frozenAt:null},nodes:[],edges:[]}),
    requirements:async()=>data({nodes:[]}),
    qa:async()=>data({records:[]}),
    release:async()=>data({readiness:{
      versionId:'main',rqState:'UNKNOWN',ownerGrantState:'UNKNOWN',permits:null,
      gates:[{name:'PHYSICAL_HOST',status:'NOT_RUN',owner:'Mock Host'}]}}),
    settings:async()=>data({configurationUrl:setupUrl,
      selectedRepository:repository,selectedRef:'main',currentSha:sourceSha,
      agentHostMounted:true}),
    taskDrafts:async()=>data({drafts:[],repository,coverage:'COMPLETE_SINGLE_PAGE'})
  };
}
function connectCDP(url){
  return new Promise((resolve,reject)=>{
    const socket=new WebSocket(url),pending=new Map();let id=0;
    socket.addEventListener('error',reject,{once:true});
    socket.addEventListener('open',()=>{
      socket.addEventListener('message',event=>{
        const packet=JSON.parse(typeof event.data==='string'?event.data:
          Buffer.from(event.data).toString());
        if(packet.id){
          const item=pending.get(packet.id);if(!item)return;
          pending.delete(packet.id);
          packet.error?item.reject(Error(packet.error.message)):item.resolve(packet.result??{});
        }else if(packet.method==='Network.responseReceived'){
          const response=packet.params?.response;
          if(response?.url?.startsWith('http://127.0.0.1:'))
            net.push({url:response.url.replace(/actionKey=[^&#]+/g,'actionKey=REDACTED'),
              status:response.status,type:packet.params.type});
        }else if(packet.method==='Runtime.exceptionThrown')
          errors.push('JS_EXCEPTION:'+JSON.stringify(packet.params.exceptionDetails).slice(0,320));
      });
      const call=(method,params={})=>new Promise((yes,no)=>{
        const n=++id;pending.set(n,{resolve:yes,reject:no});
        socket.send(JSON.stringify({id:n,method,params}));
      });
      const evaluate=async code=>{
        const result=await call('Runtime.evaluate',{expression:code,
          returnByValue:true,awaitPromise:true});
        if(result.exceptionDetails)throw Error('CHROMIUM_JS_ERROR:'+
          result.exceptionDetails.text);
        return result.result?.value;
      };
      resolve({call,evaluate,close:()=>socket.close()});
    },{once:true});
  });
}
async function browser(){
  const binary=process.env.TALLY_BROWSER_PATH??'/usr/bin/chromium';
  const profile=join(tmp,'chrome');await mkdir(profile);
  chromium=spawn(binary,['--headless=new','--no-sandbox','--disable-gpu',
    '--disable-dev-shm-usage','--remote-allow-origins=*',
    '--remote-debugging-port=0','--user-data-dir='+profile,'about:blank'],
    {stdio:'ignore'});
  const port=await poll(async()=>{
    try{return Number((await readFile(join(profile,'DevToolsActivePort'),'utf8'))
      .split('\n')[0]);}catch{return null;}
  },'CDP_PORT',20000);
  const targets=await (await fetch('http://127.0.0.1:'+port+'/json')).json();
  client=await connectCDP(targets.find(t=>t.type==='page').webSocketDebuggerUrl);
  for(const method of ['Page.enable','Runtime.enable','Network.enable'])
    await client.call(method);
}
async function evaluate(s){return client.evaluate(s);}
async function navigate(url,label){
  await client.call('Page.navigate',{url});
  await poll(()=>evaluate("document.readyState==='complete'"),'GOTO_'+label);
  clicks.push({kind:'navigate',label,url:url.replace(/actionKey=[^&#]+/g,'actionKey=REDACTED')});
}
const located=selector=>evaluate('Boolean(document.querySelector('+
  JSON.stringify(selector)+'))');
async function click(selector,label){
  const position=await poll(()=>evaluate('(()=>{const e=document.querySelector('+
    JSON.stringify(selector)+');if(!e||e.disabled||!e.getClientRects().length)return null;'+
    'const r=e.getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()'),
  'CLICK_'+label);
  await client.call('Input.dispatchMouseEvent',{type:'mousePressed',
    x:position.x,y:position.y,button:'left',clickCount:1});
  await client.call('Input.dispatchMouseEvent',{type:'mouseReleased',
    x:position.x,y:position.y,button:'left',clickCount:1});
  clicks.push({kind:'click',label,selector});
}
async function fill(selector,text){
  const p=await poll(()=>evaluate('(()=>{const e=document.querySelector('+
    JSON.stringify(selector)+');if(!e||!e.getClientRects().length)return null;'+
    'const r=e.getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()'),
  'FIELD_'+selector);
  await client.call('Input.dispatchMouseEvent',{type:'mousePressed',
    x:p.x,y:p.y,button:'left',clickCount:1});
  await client.call('Input.dispatchMouseEvent',{type:'mouseReleased',
    x:p.x,y:p.y,button:'left',clickCount:1});
  await client.call('Input.dispatchKeyEvent',{type:'keyDown',key:'a',
    code:'KeyA',windowsVirtualKeyCode:65,modifiers:2});
  await client.call('Input.dispatchKeyEvent',{type:'keyUp',key:'a',
    code:'KeyA',windowsVirtualKeyCode:65,modifiers:2});
  await client.call('Input.insertText',{text});
  clicks.push({kind:'fill',selector,length:text.length});
}
async function shot(name){
  const result=await client.call('Page.captureScreenshot',
    {format:'jpeg',quality:40});
  const bytes=Buffer.from(result.data,'base64');
  const path=join(evidence,name+'.jpg');
  await writeFile(path,bytes);
  const entry={name,bytes:bytes.length,sha256:createHash('sha256').update(bytes)
    .digest('hex')};
  screens.push(entry);
  console.log('R156_SCREENSHOT='+JSON.stringify({...entry,base64:result.data}));
}
const bodyHas=text=>evaluate('document.body.innerText.includes('+
  JSON.stringify(text)+')');
const note=(name,value)=>checks.push({name,pass:Boolean(value)});
try{
  await stat(join(process.cwd(),'apps/tally-web/dist/index.html'));
  const heads=new Map(),vaultMap=new Map();
  const projects=['kaicreator-mm/tally','mock-owner/project2'];
  for(const name of projects){
    const dir=join(tmp,'repos',name.replace('/','-'));
    await mkdir(dir,{recursive:true});
    git(dir,'init','-q');
    await writeFile(join(dir,'README.md'),name+'\n');
    git(dir,'add','README.md');
    git(dir,'-c','user.name=Mock','-c','user.email=mock@localhost',
      'commit','-qm','mock source');
    heads.set(name,git(dir,'rev-parse','HEAD'));
  }
  const catalog=new LocalProjectCatalog({dir:join(tmp,'catalog'),
    vault:{seal:async text=>{const id='mock:'+vaultMap.size;vaultMap.set(id,text);return id;},
      open:async id=>vaultMap.get(id)??''},
    readHead:async({repository,ref,token})=>{
      assert.equal(ref,'main');assert.equal(token,'fake-token-r156');
      return heads.get(repository);
    }});
  app=await startFirstRunServer({env:{TALLY_V2_CONFIG_DIR:join(tmp,'catalog')},
    catalog,
    startConfigured:async({env})=>{
      const repo=env.TALLY_V2_REPOSITORY,sha=env.TALLY_V2_VERSION_SHA;
      assert.equal(sha,heads.get(repo));
      const server=createAlmServer(defaultAlmConfig({dhPath:tmp,
        tallyPath:tmp,tallyBPath:tmp,githubToken:env.TALLY_GITHUB_READ_TOKEN,
        currentSourceSha:sha,workRoot:join(tmp,'work',repo.replace('/','-'))}),
        {run:mockRunner(repo,sha),
          v2ReadModelHost:mockModels(repo,sha,env.TALLY_SETUP_URL),
          agentHostMounted:true,selectedAgentTaskIssue:196});
      const address=await server.listen();
      return {address,close:server.close};
    }});
  await browser();
  const setup=app.address.origin+'/setup';
  await navigate(setup,'setup');
  await poll(()=>located('#add input[name=repository]'),'setup form');
  await shot('01-setup');
  async function addProject(name){
    await fill('#add input[name=repository]',name);
    await fill('#add input[name=ref]','main');
    await fill('#add input[name=token]','fake-token-r156');
    await click('#add button[type=submit]','add '+name);
    return poll(async()=>{
      const u=await evaluate('location.href');
      return u.includes('/app?')&&u.includes('actionKey=')?u:null;
    },'app-with-key',19000);
  }
  let url=await addProject(projects[0]);
  await poll(()=>located('#root'),'React root');
  await poll(()=>bodyHas('Tally v2'),'React shell');
  note('setup-to-installed-app-actionKey',url.includes('actionKey='));
  await shot('02-app-start');
  await navigate(url+'#/development/product?projectId=kaicreator-mm%2Ftally',
    'product');
  await poll(()=>located('input[aria-label="产品目标"]'),'Product form');
  await fill('input[aria-label="产品目标"]','Chromium product goal');
  const btn=async label=>evaluate('(()=>{const all=[...document.querySelectorAll("button")];'+
    'for(const old of all)delete old.dataset.r156Action;'+
    'const b=all.find(x=>x.textContent.trim()==='+JSON.stringify(label)+');'+
    'if(b){b.dataset.r156Action="1";return !b.disabled}return false;})()');
  assert.equal(await poll(()=>btn('保存需求目标'),'goal button'),true);
  await click('button[data-r156-action]','submit goal');
  await poll(()=>bodyHas('submit-goal → OK'),'goal response');
  await shot('03-product-goal');
  assert.equal(await poll(()=>btn('生成 Product / L2 草稿'),'draft button'),true);
  await click('button[data-r156-action]','draft Product');
  await poll(()=>bodyHas('当前 Product 草稿'),'Product draft');
  await shot('04-product-draft');
  await navigate(url+'#/development/tasks?projectId=kaicreator-mm%2Ftally&versionId=main','tasks');
  await poll(()=>located('table tbody tr'),'Task list');
  await click('table tbody tr','select Task');
  await shot('05-task-drawer');
  // Mock Agent is only valid in this explicitly named fixture.
  const action=await poll(()=>evaluate('(()=>{const b=[...document.querySelectorAll("button")]'+
    '.find(x=>/启动|Start/.test(x.textContent)&&!x.disabled);'+
    'if(b){b.dataset.r156Start="1";return true}return false;})()'),
  'mock Agent button');
  assert.equal(action,true);
  await click('button[data-r156-start]','Mock Codex task start');
  await poll(()=>mockRuns.length===1,'mock Codex + actual Git',20000);
  await shot('06-agent-result');
  await navigate(url+'#/development/agents?projectId=kaicreator-mm%2Ftally&versionId=main',
    'agent history');
  await poll(()=>bodyHas('Attempt'),'Agent history screen');
  await shot('07-agent-history');
  await navigate(url+'#/development/l2-dag?projectId=kaicreator-mm%2Ftally','DAG');
  await poll(()=>bodyHas('Task'),'DAG screen');
  await navigate(url+'#/development/qa?projectId=kaicreator-mm%2Ftally&versionId=main','QA');
  await poll(()=>bodyHas('QA')||bodyHas('验证'),'QA screen');
  await navigate(url+'#/development/release?projectId=kaicreator-mm%2Ftally&versionId=main',
    'Release');
  await poll(()=>bodyHas('发布资格'),'Release');
  assert.equal(await evaluate('([...document.querySelectorAll("button")]'+
    '.filter(x=>/发布 Release|合并到 main/.test(x.textContent)).every(x=>x.disabled))'),true,
    'Release must remain blocked under mocks');
  await shot('08-release-blocked');
  await navigate(url+'#/development/maintenance?projectId=kaicreator-mm%2Ftally',
    'Maintenance');
  await poll(()=>bodyHas('真实缺陷维护工作台'),'Maintenance snapshot');
  const link=await evaluate('document.querySelector("a[href*=diagnostics]")?.href');
  assert.ok(link?.includes('actionKey='),'maintenance key not passed');
  await shot('09-maintenance');
  await click('a[href*=diagnostics]','Maintenance workbench');
  await poll(()=>located('#maintenance-app'),'Legacy maintenance controller');
  await navigate(url+'#/settings/git-github','Settings');
  await poll(()=>located('a[href*="/setup"]'),'Real Setup link');
  await click('a[href*="/setup"]','switch via Setup Host');
  await poll(async()=>(await evaluate('location.href')).startsWith(setup),'setup round trip');
  url=await addProject(projects[1]);
  await poll(()=>bodyHas(projects[1]),'new Host source identity');
  await shot('10-project-switched');
  await navigate(url+'#/development/product?projectId=mock-owner%2Fproject2',
    'Product other repo');
  await poll(()=>bodyHas('其他项目的通用需求建模 Host 尚未接入'),'no fake Product');
  assert.equal(await evaluate('document.querySelector("input[aria-label=产品目标]")?.disabled'),
    true,'other repo must stay disabled');
  await shot('11-product-scope');
  note('native Chromium clickflow with Mock Codex and real Git',true);
}catch(e){
  errors.push(String(e?.stack??e).slice(0,1800));
  if(client)try{await shot('99-failure');}catch{}
}finally{
  try{client?.close();}catch{}
  if(chromium)chromium.kill('SIGTERM');
  if(app)await app.close().catch(()=>{});
  await rm(tmp,{recursive:true,force:true});
}
const result={label:'TALLY_V2_R156_INSTALLED_CHROMIUM_UI_E2E',
  sourceSha:process.env.CI_COMMIT_SHA??'NOT_PINNED',
  result:errors.length?'FAIL':'PASS_MOCK_SCOPE_ONLY',screens,clicks,network:net,
  browserErrors:errors,checks,mockRuns,
  claims:['MODEL_MOCK','HOST_READMODELS_MOCK','GITHUB_HEAD_MOCK','VAULT_MOCK'],
  real:['CHROMIUM','REACT_BUILD','SETUP_AND_ALM_HTTP','LOCAL_GIT_COMMIT'],
  realCodex:false,releaseEligible:false};
await writeFile(join(evidence,'r156-browser-result.json'),JSON.stringify(result,null,2)+'\n');
console.log('R156_BROWSER_RESULT='+JSON.stringify(result));
if(errors.length)process.exitCode=1;
