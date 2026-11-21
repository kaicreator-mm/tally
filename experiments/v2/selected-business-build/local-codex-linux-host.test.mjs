/** Source-only Linux Host contract probes. A fake exec is NOT a native
 * Linux isolation or Agent proof. A physical positive is recorded separately.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,mkdir,writeFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {
  linuxSandboxArgs,linuxBuilderEnvironment,createLinuxHostOperations,
} from './local-codex-linux-host.mjs';
const config=(root='/tmp/tally-r152')=>({
  platform:'linux',worktreePath:join(root,'worktree'),
  builderProfile:join(root,'builder'),codexHome:join(root,'builder','.codex'),
  operatorHome:join(root,'operator'),hiddenFixture:join(root,'operator','hidden'),
  emptyGitConfig:join(root,'empty-gitconfig'),cliPath:'/usr/bin/node',
  bwrapPath:'/usr/bin/bwrap',gitPath:'/usr/bin/git',
  builderPrincipal:'linux-bwrap:65534',builderSid:'linux-uid:65534',
  sourceSha:'a'.repeat(40),repositoryPath:join(root,'repository'),
});
const defaults={
  finalizeCandidate:async()=>({writeSetValid:true}),
  candidatePathOccupied:async()=>({occupied:false}),
};
test('Linux namespace has one writable worktree and dedicated model home',()=>{
  const x=config();
  const args=linuxSandboxArgs(x);
  assert.deepEqual(args.slice(0,6),['--unshare-user','--uid','65534','--gid','65534','--unshare-pid']);
  assert.ok(args.includes('--clearenv'));
  assert.ok(args.includes('--die-with-parent'));
  assert.ok(args.includes('--new-session'));
  assert.ok(args.includes('--chdir'));
  assert.ok(args.includes('--'));
  assert.deepEqual(args.slice(args.indexOf('--bind')+1,args.indexOf('--bind')+3),
    [x.builderProfile,'/home/tally-builder']);
  assert.ok(args.some((value,i)=>value==='--bind'&&args[i+1]===x.worktreePath&&
    args[i+2]==='/workspace'));
  assert.ok(!args.includes(x.operatorHome));
  assert.ok(!args.includes(x.hiddenFixture));
  assert.ok(!args.some(v=>v==='--ro-bind'&&args.includes('/')));
  assert.deepEqual(Object.keys(linuxBuilderEnvironment(x)).sort(),
    ['CODEX_HOME','GCM_INTERACTIVE','GIT_CONFIG_GLOBAL','GIT_CONFIG_NOSYSTEM',
      'GIT_TERMINAL_PROMPT','HOME','PATH','TMPDIR'].sort());
});
test('Linux Host does not accept an OS mismatch, operator root or invalid identity',()=>{
  for(const override of [
    {platform:'win32'}, {builderSid:'S-1-5-21-12345'},
    {builderPrincipal:'root'}, {builderProfile:'/tmp/tally-r152/operator'},
    {worktreePath:'/tmp/tally-r152/builder/worktree'},
    {cliPath:'/tmp/untrusted-codex'},
    {modelProxy:'http://user:password@127.0.0.1:10808'},
    {codexHome:'/tmp/operator/.codex'},
  ])assert.throws(()=>linuxSandboxArgs({...config(),...override}));
});
test('Linux Host probe refuses missing bubblewrap and does not issue Agent commands',async()=>{
  const calls=[];
  const exec=async(command,args)=>{
    calls.push({command,args});
    if(command.endsWith('bwrap'))throw Error('ENOENT');
    return {code:0,stdout:'codex-cli 0.1.0\n'};
  };
  const ops=createLinuxHostOperations({defaults,exec});
  const probe=await ops.probeProvider(config());
  assert.equal(probe.available,false);
  assert.equal(calls.filter(x=>x.args.some(v=>v==='exec')).length,0);
});
test('Linux worktree setup uses exact SHA and native git, not Windows icacls',async()=>{
  const calls=[];
  const exec=async(command,args)=>{
    calls.push({command,args});
    return {code:0,stdout:args.includes('rev-parse')?'a'.repeat(40)+'\n':''};
  };
  const root=await mkdtemp(join(tmpdir(),'tally-r152-native-git-'));
  try{
    const x=config(root);
    const ops=createLinuxHostOperations({defaults,exec});
    const prepared=await ops.prepareWorktree(x);
    assert.equal(prepared.baseSha,x.sourceSha);
    assert.deepEqual(calls[0].args,
      ['-C',x.repositoryPath,'worktree','add','--detach',x.worktreePath,x.sourceSha]);
    assert.ok(calls.every(c=>c.command==='/usr/bin/git'));
  }finally{await rm(root,{recursive:true,force:true});}
});
test('confinement readback checks namespace UID, workspace and hidden secrets',async()=>{
  const root=await mkdtemp(join(tmpdir(),'tally-r152-confinement-'));
  try{
    const x=config(root);
    // Mocked exec uses stable system directory paths, not a CI-specific binary.
    x.cliPath='/usr/bin';
    x.bwrapPath='/usr/bin';
    for(const dir of [x.worktreePath,x.builderProfile,x.operatorHome])
      await mkdir(dir,{recursive:true});
    await writeFile(x.hiddenFixture,'private',{mode:0o600});
    await writeFile(x.emptyGitConfig,'',{mode:0o600});
    const calls=[];
    const exec=async(command,args)=>{
      calls.push(args);
      if(args.at(-2)==='id'&&args.at(-1)==='-u')throw Error('unlikely');
      if(args.includes('/usr/bin/id'))return {code:0,stdout:'65534\n'};
      return {code:0,stdout:''};
    };
    const ops=createLinuxHostOperations({defaults,exec});
    const proof=await ops.proveConfinement(x,x.worktreePath);
    assert.equal(proof.verified,true);
    assert.equal(proof.builderSid,'linux-uid:65534');
    assert.ok(calls.some(a=>a.includes(x.hiddenFixture)));
    assert.ok(calls.some(a=>a.includes('login')&&a.includes('status')));
    const failed=createLinuxHostOperations({defaults,
      exec:async()=>{throw Error('user namespace unavailable');}});
    assert.equal((await failed.proveConfinement(x,x.worktreePath)).verified,false);
  }finally{await rm(root,{recursive:true,force:true});}
});
