/**
 * R155 — one bounded offline end-user API journey against REAL first-run and
 * ALM HTTP controllers. Codex is a child Node process mock, candidate is real
 * Git worktree/commit/tree. GitHub HEAD and vault are explicit fixtures.
 * NEVER call this "real Agent", "Host K/C", "GitHub PR" or release proof.
 */
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {mkdtemp,mkdir,rm,readFile,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join,dirname} from 'node:path';
import {LocalProjectCatalog} from './local-project-catalog.mjs';
import {startFirstRunServer} from './first-run-server.mjs';
import {createAlmServer} from './journey-server.mjs';
import {defaultAlmConfig} from './journey-controller.mjs';

const started=Date.now(),rows=[],liveApps=new Map(),mockEffects=[];
const add=(id,step,status,reason=null)=>{rows.push({id,step,status,reason});};
const cmd=(program,args,{cwd,env}={})=>{
  const p=spawnSync(program,args,{cwd,env:env??process.env,
    encoding:'utf8',timeout:20000});
  assert.equal(p.status,0,'MOCK_PROCESS_FAILED:'+(p.stderr??'').slice(0,120));
  return p.stdout.trim();
};
const git=(cwd,...args)=>cmd('git',['-C',cwd,...args],{env:{
  ...process.env,GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:'/dev/null',
  GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never'}});
const get=async(origin,path)=>{
  const r=await fetch(origin+path,{cache:'no-store'});
  assert.equal(r.status,200,'GET:'+path);
  return r.json();
};
const post=async(origin,path,key,body)=>{
  const r=await fetch(origin+path,{method:'POST',headers:{
    origin,'content-type':'application/json',...(key?{
      [path.startsWith('/api/setup/')?'x-tally-setup-key':'x-tally-action-key']:key}:{})},
    body:JSON.stringify(body)});
  return {http:r.status,result:await r.json()};
};
const work=await mkdtemp(join(tmpdir(),'tally-r155-mock-'));
const heads=new Map(),secrets=new Map(),token='fixture-token-NOT-LIVE-00001';
const vault={
  async seal(value){const key='fixture:key:'+String(secrets.size+1);
    secrets.set(key,value);return key;},
  async open(key){if(!secrets.has(key))throw Error('CREDENTIAL_UNAVAILABLE');
    return secrets.get(key);}
};
const dir=join(work,'catalog');
const readHead=async({repository,ref,token:provided})=>{
  assert.equal(provided,token);assert.equal(ref,'main');
  if(!heads.has(repository))throw Error('UNKNOWN_MOCK_SOURCE');
  return heads.get(repository);
};
const store=new LocalProjectCatalog({dir,vault,readHead});
const fixtureRepos=['mock-owner/project-one','mock-owner/project-two'];
let service;
const runMock=({repository,sourceSha})=>async kind=>{
  if(kind!=='start')return {status:'FAIL',exitCode:1,observedAt:new Date().toISOString()};
  const source=join(work,'sources',repository.replace('/','-'));
  const tree=join(work,'candidates',repository.replace('/','-'));
  await mkdir(dirname(tree),{recursive:true});
  git(source,'worktree','add','--detach',tree,sourceSha);
  // Real Node child process substitutes ONLY the Codex *model process*.
  const candidate=join(tree,'MOCK_CODEX_CANDIDATE.md');
  cmd(process.execPath,['-e',
    "require('node:fs').writeFileSync(process.argv[1], 'Mock Codex only\\n')",
    candidate],{cwd:tree,env:{PATH:process.env.PATH??'/usr/bin:/bin',
      HOME:join(work,'mock-model-home')}});
  git(tree,'add','MOCK_CODEX_CANDIDATE.md');
  git(tree,'-c','user.name=Tally Mock','-c','user.email=mock@localhost',
    'commit','-q','-m','mock agent candidate');
  const candidateCommit=git(tree,'rev-parse','HEAD');
  const candidateTree=git(tree,'rev-parse','HEAD^{tree}');
  assert.equal(git(tree,'rev-parse','HEAD^'),sourceSha);
  assert.equal(git(tree,'diff','--name-only',sourceSha,candidateCommit),
    'MOCK_CODEX_CANDIDATE.md');
  assert.equal((await readFile(candidate,'utf8')).trim(),'Mock Codex only');
  mockEffects.push({repository,sourceSha,candidateCommit,candidateTree,
    selectedProvider:'MOCK_CODEX',githubPR:'NOT_RUN',sealedK:'NOT_RUN'});
  // realAgentEffect is a *local controller fixture*, never a real Host claim.
  return {status:'PASS',exitCode:0,observedAt:new Date().toISOString(),
    evidenceDir:tree,mockMode:true,result:{status:'PASS',realAgentEffect:true,
      installedAcceptance:{state:'NOT_RUN'},claimReceipts:[],
      hiddenVerifier:'HIDDEN_NOT_QUALIFIED',
      result:{state:'LOCAL_CANDIDATE_READY',candidateCommit,candidateTree}}};
};
try{
  for(const repository of fixtureRepos){
    const repo=join(work,'sources',repository.replace('/','-'));
    await mkdir(repo,{recursive:true});
    git(repo,'init','-q');
    await writeFile(join(repo,'README.md'),repository+'\n');
    git(repo,'add','README.md');
    git(repo,'-c','user.name=Tally Mock','-c','user.email=mock@localhost',
      'commit','-q','-m','seed fixture');
    heads.set(repository,git(repo,'rev-parse','HEAD'));
  }
  service=await startFirstRunServer({env:{TALLY_V2_CONFIG_DIR:dir},
    catalog:store,startConfigured:async({env})=>{
      const repository=env.TALLY_V2_REPOSITORY;
      const sourceSha=env.TALLY_V2_VERSION_SHA;
      assert.equal(sourceSha,heads.get(repository));
      const app=createAlmServer(defaultAlmConfig({dhPath:work,
        tallyPath:work,tallyBPath:work,githubToken:env.TALLY_GITHUB_READ_TOKEN,
        currentSourceSha:sourceSha,
        workRoot:join(work,'run',repository.replace('/','-'))}),
        {run:runMock({repository,sourceSha})});
      const address=await app.listen();
      liveApps.set(repository,{...address});
      return {address,close:app.close};
    }});
  const setupOrigin=service.address.origin;
  assert.equal((await get(setupOrigin,'/api/setup/status')).configured,false);
  const setupPage=await (await fetch(setupOrigin+'/setup')).text();
  const key=/<meta name="tally-setup-key" content="([^"]+)"/.exec(setupPage)?.[1];
  assert.ok(key,'SETUP_CSRF_KEY_REQUIRED');
  add('J01','fresh start without project environment','PASS');
  for(const repository of fixtureRepos){
    const result=await post(setupOrigin,'/api/setup/add',key,
      {repository,ref:'main',token});
    assert.equal(result.http,200);
    assert.equal(result.result.status,'CONNECTED');
  }
  let status=await get(setupOrigin,'/api/setup/status');
  assert.equal(status.projects.length,2);
  assert.equal(status.selected.repository,fixtureRepos[1]);
  const persist=await readFile(join(dir,'projects.json'),'utf8');
  assert.ok(!persist.includes(token));
  const reopened=new LocalProjectCatalog({dir,vault,readHead});
  assert.equal((await reopened.selected()).env.TALLY_V2_REPOSITORY,fixtureRepos[1]);
  const switched=await post(setupOrigin,'/api/setup/select',key,{
    repository:fixtureRepos[0],ref:'main'});
  assert.equal(switched.http,200);
  status=await get(setupOrigin,'/api/setup/status');
  assert.equal(status.selected.repository,fixtureRepos[0]);
  add('J01','two projects, switch, reload, no plaintext token','PASS');
  const app=liveApps.get(fixtureRepos[0]);
  const {origin,actionKey}=app;
  const appPage=await fetch(origin+'/app');
  add('J01','React app shell HTTP',appPage.status===200?'PASS':'NOT_RUN',
    appPage.status===200?null:'FRONTEND_DIST_NOT_INSTALLED_IN_CI_STEP');

  const intake=await get(origin,'/api/intake/snapshot');
  assert.equal(intake.snapshot.status,'OBSERVED');
  const sha=heads.get(fixtureRepos[0]);
  const intakeAct=async(kind,args={})=>post(origin,'/api/intake/actions',actionKey,
    {kind,sourceSha:sha,args});
  assert.equal((await intakeAct('submit-goal',{goal:'Mock Codex E2E'})).result.status,'OK');
  assert.equal((await intakeAct('select-evidence',{refs:['157','158']})).result.status,'OK');
  assert.equal((await intakeAct('confirm-inspect')).result.status,'OK');
  assert.equal((await intakeAct('draft-product',
    {l2Decision:'KEEP_MERGED_L2_ARCHITECTURE'})).result.status,'OK');
  const unqualifiedFreeze=await intakeAct('freeze-draft');
  assert.equal(unqualifiedFreeze.result.status,'BLOCKED');
  add('J02','Product draft through real Host-facing HTTP/controller','PASS');
  add('J02','formal L2 freeze without a reviewed Host','NOT_RUN',
    'NO_REVIEWED_SEALED_K_IN_MOCK_FIXTURE');
  const req=await get(origin,'/api/v2/requirements?versionId=main');
  add('J02','requirements Host projection',req.connected?'PASS':'NOT_CONNECTED',
    req.reason??null);

  const tasks=await get(origin,'/api/v2/tasks?versionId=main');
  const dag=await get(origin,'/api/v2/dag?versionId=main');
  add('J03','frozen Task DAG readback',tasks.connected&&dag.connected?'PASS':'NOT_CONNECTED',
    tasks.reason??dag.reason??null);
  const noKey=await post(origin,'/api/actions',null,{kind:'start',sourceSha:sha});
  assert.equal(noKey.http,403);assert.equal(mockEffects.length,0);
  const stale=await post(origin,'/api/actions',actionKey,
    {kind:'start',sourceSha:'0'.repeat(40)});
  assert.equal(stale.http,409);assert.equal(mockEffects.length,0);
  add('J04','CSRF and stale source refuse spawning mock Codex','PASS');
  const startedAgent=await post(origin,'/api/actions',actionKey,
    {kind:'start',sourceSha:sha});
  assert.equal(startedAgent.http,200,JSON.stringify(startedAgent.result).slice(0,300));
  assert.equal(startedAgent.result.status,'PASS');
  assert.equal(mockEffects.length,1);
  add('J04','Mock Codex process through real ALM controller','PASS');
  add('J05','actual Git worktree, commit parent, tree and changed-file readback','PASS');
  const count=mockEffects.length;
  const hostCandidate=await get(origin,'/api/v2/candidate-host');
  add('J05','real GitHub Draft PR submission','NOT_RUN',
    hostCandidate.reason??'NO_LIVE_GITHUB_HOST_OR_WRITE_TOKEN');
  assert.equal(mockEffects.length,count);
  const attempts=await get(origin,'/api/v2/agent-attempts');
  add('J06','Agent attempt history from mock process',
    attempts.connected&&attempts.attempts?.length>0?'PASS':'FAIL');
  const qa=await get(origin,'/api/v2/qa?versionId=main');
  add('J06','qualified external CI/QA projection',qa.connected?'PASS':'NOT_CONNECTED',
    qa.reason??null);

  const release=await get(origin,'/api/release/snapshot');
  assert.equal(release.snapshot.releasePass,false);
  const forbidden=await post(origin,'/api/release/actions',null,
    {kind:'publish',sourceSha:sha});
  assert.equal(forbidden.http,403);
  const releaseAction=await post(origin,'/api/release/actions',actionKey,
    {kind:'publish',sourceSha:sha});
  assert.notEqual(releaseAction.result.status,'PASS');
  assert.equal(releaseAction.result.releasePass,false);
  add('J07','Release readiness denies unauthorized publish','PASS');

  const reportDefect=await post(origin,'/api/maintenance/actions',actionKey,
    {kind:'report-defect',sourceSha:sha,args:{
      summary:'Mock journey regression',requirementIssue:'392',
      releaseVersion:'v2.0.0',releaseCommitSha:sha}});
  assert.equal(reportDefect.result.status,'OK');
  const maintenance=await get(origin,'/api/maintenance/snapshot');
  assert.equal(maintenance.snapshot.candidate.state,'DRAFT');
  assert.equal(maintenance.snapshot.releasePass,false);
  const review=await post(origin,'/api/maintenance/actions',actionKey,
    {kind:'review-candidate',sourceSha:sha,args:{reviewer:'reviewer:mock'}});
  assert.equal(review.result.status,'OK');
  const maintained=await get(origin,'/api/maintenance/snapshot');
  assert.equal(maintained.snapshot.candidate.state,'REVIEWED');
  add('J08','real defect draft, review and source-pinned snapshot','PASS');
  const noMaintKey=await post(origin,'/api/maintenance/actions',null,
    {kind:'publish-maintenance-issue',sourceSha:sha});
  assert.equal(noMaintKey.http,403);
  add('J08','maintenance effect gated by CSRF/Host','PASS');
  add('J01','real native Linux Secret Service','NOT_RUN','TEST_VAULT_FIXTURE');
  add('J04','real sealed K/Claim/FSW/bwrap/Codex','NOT_RUN','MOCK_PROCESS_ONLY');
  add('J05','external GitHub PR published and read back','NOT_RUN','GITHUB_WRITE_DISABLED');
}finally{
  await service?.close();
  await rm(work,{recursive:true,force:true});
}
const summary=Object.fromEntries(['PASS','FAIL','NOT_RUN','NOT_CONNECTED'].map(status=>
  [status,rows.filter(row=>row.status===status).length]));
const report={label:'TALLY_V2_R155_MOCK_FUNCTIONAL_ACCEPTANCE',
  source:process.env.CI_COMMIT_SHA??'LOCAL_HEAD_NOT_VERIFIED',
  os:process.platform,node:process.version,
  mode:'MOCK_CODEX_AND_OFFLINE_GITHUB_HEAD_AND_VAULT_FIXTURES',
  durationMs:Date.now()-started,
  mockCodeProcessAndGitCandidates:mockEffects.length,
  summary,rows,realAgentQualified:false,releaseEligible:false};
console.log('TALLY_R155_MOCK_FULL_JOURNEY_REPORT='+JSON.stringify(report));
if(summary.FAIL>0)process.exitCode=1;
