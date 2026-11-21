/** Linux execution boundary for existing sealed K -> Host -> Agent.
 * No second scheduler or authority: only POSIX process/worktree operations.
 * Unavailable user namespace or bubblewrap means DENIED, never fallback.
 */
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {realpath,stat,mkdir} from 'node:fs/promises';
import {dirname,join,isAbsolute,resolve,sep} from 'node:path';

const execFileAsync=promisify(execFile);
const SHA=/^[a-f0-9]{40}$/;
const PATH=/^\/[^:\0\n\r]{1,300}$/;
export const LINUX_BUILDER_UID=65534;
const BWRAP_UID=String(LINUX_BUILDER_UID);
const safePath=value=>typeof value==='string'&&PATH.test(value)&&
  isAbsolute(value)&&resolve(value)===value;
const inside=(p,parent)=>p===parent||p.startsWith(parent+sep);
const nativeExec=async(command,args,options={})=>{
  const result=await execFileAsync(command,args,{timeout:15000,maxBuffer:65536,...options});
  return {code:0,stdout:result.stdout??'',stderr:result.stderr??''};
};
export function linuxBuilderEnvironment(config={}){
  return {
    PATH:'/usr/local/bin:/usr/bin:/bin',
    HOME:'/home/tally-builder',
    CODEX_HOME:'/home/tally-builder/.codex',
    TMPDIR:'/tmp',
    GIT_CONFIG_NOSYSTEM:'1',
    GIT_CONFIG_GLOBAL:'/tmp/tally-empty.gitconfig',
    GIT_TERMINAL_PROMPT:'0',
    GCM_INTERACTIVE:'never',
    ...(config.modelProxy?{HTTPS_PROXY:config.modelProxy,HTTP_PROXY:config.modelProxy}:{}),
  };
}
export function linuxSandboxArgs(config){
  const paths=['worktreePath','builderProfile','operatorHome','hiddenFixture',
    'emptyGitConfig','cliPath','bwrapPath'];
  if(paths.some(name=>!safePath(config[name])))
    throw new Error('LINUX_SANDBOX_PATH_INVALID');
  if(config.platform!=='linux')throw new Error('LINUX_HOST_PLATFORM_MISMATCH');
  if(!inside(config.cliPath,'/usr')||!inside(config.bwrapPath,'/usr'))
    throw new Error('LINUX_EXECUTABLE_NOT_IN_READONLY_RUNTIME');
  if(config.builderSid!=='linux-uid:'+BWRAP_UID||
     config.builderPrincipal!=='linux-bwrap:'+BWRAP_UID)
    throw new Error('LINUX_BUILDER_PRINCIPAL_INVALID');
  const work=resolve(config.worktreePath),home=resolve(config.builderProfile),
    owner=resolve(config.operatorHome),hidden=resolve(config.hiddenFixture);
  if(inside(work,home)||inside(home,work)||inside(owner,home)||
     inside(owner,work)||inside(hidden,home)||inside(hidden,work)||home===owner)
    throw new Error('LINUX_SANDBOX_SCOPE_OVERLAP');
  if(config.codexHome!==join(home,'.codex'))
    throw new Error('LINUX_CODEX_HOME_NOT_PRIVATE');
  // Arguments/CAS records must never carry proxy usernames or passwords.
  if(config.modelProxy){
    try{
      const u=new URL(config.modelProxy);
      if(!['http:','https:'].includes(u.protocol)||u.username||u.password||
         u.search||u.hash||u.pathname!=='/')
        throw Error('PRIVATE_PROXY');
    }catch{throw new Error('LINUX_MODEL_PROXY_INVALID');}
  }
  const e=linuxBuilderEnvironment(config);
  return [
    '--unshare-user','--uid',BWRAP_UID,'--gid',BWRAP_UID,
    '--unshare-pid','--unshare-ipc','--unshare-uts','--unshare-cgroup-try',
    '--die-with-parent','--new-session','--clearenv',
    '--ro-bind','/usr','/usr','--ro-bind-try','/bin','/bin',
    '--ro-bind-try','/lib','/lib','--ro-bind-try','/lib64','/lib64',
    '--ro-bind-try','/etc/ssl','/etc/ssl',
    '--ro-bind-try','/etc/resolv.conf','/etc/resolv.conf',
    '--ro-bind-try','/etc/hosts','/etc/hosts',
    '--ro-bind-try','/etc/nsswitch.conf','/etc/nsswitch.conf',
    '--tmpfs','/tmp','--proc','/proc','--dev','/dev',
    '--dir','/home','--dir','/home/tally-builder',
    '--bind',home,'/home/tally-builder',
    '--dir','/workspace','--bind',work,'/workspace',
    '--ro-bind',config.emptyGitConfig,'/tmp/tally-empty.gitconfig',
    '--chdir','/workspace',
    ...Object.entries(e).flatMap(([key,value])=>['--setenv',key,String(value)]),
    '--',
  ];
}
const outputCode=async(exec,command,args,options={})=>{
  try{
    const result=await exec(command,args,options);
    return {code:result?.code??0,stdout:String(result?.stdout??''),stderr:String(result?.stderr??'')};
  }catch{return {code:1,stdout:'',stderr:''};}
};
/** Takes the Windows Host's OS-independent candidate git/CAS functions. */
export function createLinuxHostOperations({defaults,exec=nativeExec}={}){
  if(!defaults||typeof defaults.finalizeCandidate!=='function')
    throw new TypeError('LINUX_HOST_DEFAULTS_REQUIRED');
  const run=async(command,args,options)=>outputCode(exec,command,args,options);
  const git=async(config,args)=>run(config.gitPath??'/usr/bin/git',args,{
    env:{PATH:'/usr/local/bin:/usr/bin:/bin',HOME:config.builderProfile,
      GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:config.emptyGitConfig,
      GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never'}});
  return {
    ...defaults,
    async probeProvider(config){
      // A root Host would bypass the intended unprivileged trust boundary.
      if(typeof process.getuid==='function'&&process.getuid()===0)
        return {available:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
      try{linuxSandboxArgs(config);}catch{return {available:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};}
      const bwrap=await run(config.bwrapPath,['--version']);
      // Check real unprivileged user namespace *before* adding a git worktree.
      // Version existence alone cannot prove bwrap is usable on this kernel.
      const namespace=bwrap.code===0?await run(config.bwrapPath,[
        '--unshare-user','--uid',BWRAP_UID,'--gid',BWRAP_UID,
        '--unshare-pid','--ro-bind','/usr','/usr',
        '--proc','/proc','--dev','/dev','/usr/bin/id','-u']):
        {code:1,stdout:''};
      const codex=namespace.code===0?await run(config.cliPath,['--version'],{
        env:{PATH:'/usr/local/bin:/usr/bin:/bin',HOME:config.builderProfile,
          CODEX_HOME:config.codexHome}}):{code:1,stdout:''};
      return {available:bwrap.code===0&&namespace.code===0&&
        namespace.stdout.trim()===BWRAP_UID&&codex.code===0&&
        /^codex(?:-cli)?\s+\d+\./.test(codex.stdout.trim()),
        ...(namespace.code!==0?{reason:'BUILDER_CONFINEMENT_UNAVAILABLE'}:{})};
    },
    async prepareWorktree(config){
      if(config.platform!=='linux')throw Error('LINUX_HOST_PLATFORM_MISMATCH');
      const path=config.worktreePath;
      if(!safePath(path)||!safePath(config.repositoryPath)||!SHA.test(config.sourceSha??''))
        throw Error('CANDIDATE_WORKTREE_UNAVAILABLE');
      await mkdir(dirname(path),{recursive:true});
      const added=await git(config,['-C',config.repositoryPath,'worktree','add','--detach',
        path,config.sourceSha]);
      if(added.code!==0)throw Error('CANDIDATE_WORKTREE_UNAVAILABLE');
      const head=await git(config,['-C',path,'rev-parse','HEAD']);
      if(head.code!==0||head.stdout.trim()!==config.sourceSha)
        throw Error('CANDIDATE_BASE_MISMATCH');
      return {worktreePath:path,baseSha:head.stdout.trim()};
    },
    async proveConfinement(config,worktreePath){
      let args;
      try{
        if(config.worktreePath!==worktreePath)throw Error('WORKTREE_MISMATCH');
        args=linuxSandboxArgs(config);
        const [work,home,operator,hidden,cli,bwrap,codex,emptyGit]=await Promise.all([
          realpath(config.worktreePath),realpath(config.builderProfile),
          realpath(config.operatorHome),realpath(config.hiddenFixture),
          realpath(config.cliPath),realpath(config.bwrapPath),
          realpath(config.codexHome),realpath(config.emptyGitConfig)]);
        if(inside(operator,work)||inside(operator,home)||inside(work,home)||
           inside(hidden,home)||inside(hidden,work)||
           codex!==join(home,'.codex')||inside(emptyGit,work)||
           !inside(cli,'/usr')||!inside(bwrap,'/usr'))
          throw Error('PATH_OVERLAP');
        const [worktree,profile,modelHome,gitConfig]=await Promise.all([
          stat(config.worktreePath),stat(config.builderProfile),
          stat(config.codexHome),stat(config.emptyGitConfig)]);
        // No inherited Git credentials, and no public model home. Require the
        // checked-out candidate and its native credential directory to exist.
        if(!worktree.isDirectory()||!profile.isDirectory()||!modelHome.isDirectory()||
           (profile.mode&0o077)!==0||(modelHome.mode&0o077)!==0||
           !gitConfig.isFile()||gitConfig.size!==0)
          throw Error('LINUX_SANDBOX_MOUNT_INVALID');
      }catch{return {verified:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};}
      const id=await run(config.bwrapPath,[...args,'/usr/bin/id','-u']);
      if(id.code!==0||id.stdout.trim()!==BWRAP_UID)
        return {verified:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
      const writable=await run(config.bwrapPath,[...args,'/usr/bin/test','-w','/workspace']);
      if(writable.code!==0)return {verified:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
      const deniedPaths=[config.hiddenFixture,
        join(config.operatorHome,'.config','gh','hosts.yml'),
        join(config.operatorHome,'.git-credentials'),
        join(config.operatorHome,'.ssh'),
        join(config.operatorHome,'.codex','auth.json')];
      for(const file of deniedPaths){
        const absent=await run(config.bwrapPath,[...args,'/usr/bin/test','!','-e',file]);
        if(absent.code!==0)
          return {verified:false,reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
      }
      const login=await run(config.bwrapPath,[...args,config.cliPath,'login','status']);
      if(login.code!==0)return {verified:false,reason:'LOCAL_AGENT_UNAVAILABLE'};
      return {verified:true,builderPrincipal:config.builderPrincipal,
        builderSid:config.builderSid,hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'};
    },
  };
}
