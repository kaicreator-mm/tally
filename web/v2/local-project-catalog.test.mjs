import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp,readFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {LocalProjectCatalog,readGithubHead} from './local-project-catalog.mjs';

const shaA='a'.repeat(40),shaB='b'.repeat(40);
const vault={seal:async value=>'cipher:'+Buffer.from(value).toString('base64'),
  open:async value=>Buffer.from(value.slice(7),'base64').toString('utf8')};
test('first-run project A and B persist, switch, pin per-project SHA',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-catalog-'));
  const readHead=async ({repository,ref,token})=>{
    assert.equal(ref,'main');
    assert.ok(token.startsWith('github_pat_test_'));
    return repository==='org/a'?shaA:shaB;
  };
  try{
    const catalog=new LocalProjectCatalog({dir,vault,readHead});
    await catalog.add({repository:'org/a',ref:'main',token:'github_pat_test_A'});
    await catalog.add({repository:'org/b',ref:'main',token:'github_pat_test_B'});
    let status=await catalog.status();
    assert.equal(status.projects.length,2);
    assert.deepEqual(status.selected,{repository:'org/b',ref:'main'});
    const reboot=new LocalProjectCatalog({dir,vault,readHead});
    let selected=await reboot.select({repository:'org/a',ref:'main'});
    assert.equal(selected.env.TALLY_V2_VERSION_SHA,shaA);
    assert.equal(selected.env.TALLY_GITHUB_READ_TOKEN,'github_pat_test_A');
    selected=await reboot.select({repository:'org/b',ref:'main'});
    assert.equal(selected.env.TALLY_V2_VERSION_SHA,shaB);
    assert.equal(selected.env.TALLY_GITHUB_READ_TOKEN,'github_pat_test_B');
    const nonsecret=await readFile(join(dir,'projects.json'),'utf8');
    assert.doesNotMatch(nonsecret,/github_pat/);
    status=await reboot.status();
    assert.equal(status.selected.repository,'org/b');
    assert.equal(status.projects[0].versions[0].sha,shaA);
  }finally{await rm(dir,{recursive:true,force:true});}
});
test('reject invalid repo/ref and refuse unprovided new credential',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-catalog-'));
  const readHead=async()=>shaA;
  try{
    const catalog=new LocalProjectCatalog({dir,vault,readHead});
    await assert.rejects(catalog.add({repository:'org/c',ref:'feature/x',
      token:'github_pat_test_C'}),/INVALID_SELECTED_GITHUB_SOURCE/);
    await assert.rejects(catalog.add({repository:'org/c',ref:'main'}),
      /GITHUB_TOKEN_REQUIRED/);
    await assert.rejects(catalog.select({repository:'org/missing',ref:'main'}),
      /UNKNOWN_PROJECT_VERSION/);
  }finally{await rm(dir,{recursive:true,force:true});}
});
test('read Github HEAD sends private bearer only to exact repo branch URL',async()=>{
  let seen;
  const sha=await readGithubHead({repository:'org/a',ref:'main',
    token:'github_pat_test_A',fetchImpl:async (url,options)=>{
      seen={url,auth:options.headers.authorization};
      return {ok:true,json:async()=>({commit:{sha:shaA}})};
    }});
  assert.equal(sha,shaA);
  assert.equal(seen.url,'https://api.github.com/repos/org/a/branches/main');
  assert.equal(seen.auth,'Bearer github_pat_test_A');
  await assert.rejects(readGithubHead({repository:'org/a',ref:'main',
    token:'github_pat_test_A',fetchImpl:async()=>({ok:false})}),
    /GITHUB_SOURCE_UNAVAILABLE/);
});


test('native vault keeps only opaque refs in persisted project state',async()=>{
  const {createCredentialStore}=await import('./credential-store.mjs');
  const id='11111111-2222-4333-8444-555555555555';
  const token='github_pat_test_credential';
  const secrets=new Map();
  const run=async(_exe,args,input)=>{
    if(args[0]==='store'){secrets.set(args.at(-1),input);return '';}
    if(args[0]==='lookup')return secrets.get(args.at(-1));
    throw Error('UNEXPECTED_VAULT_COMMAND');
  };
  const native=createCredentialStore({platform:'linux',run,idFactory:()=>id});
  const dir=await mkdtemp(join(tmpdir(),'tally-keyring-'));
  try{
    const catalog=new LocalProjectCatalog({dir,vault:native,readHead:async()=>shaA});
    await catalog.add({repository:'org/native',ref:'main',token});
    const project=await readFile(join(dir,'projects.json'),'utf8');
    const stored=await readFile(catalog.credentialPath('org/native'),'utf8');
    assert.doesNotMatch(project,/github_pat_test_credential/);
    assert.equal(stored,'keyring-v1:'+id);
    assert.doesNotMatch(stored,/github_pat_test_credential/);
    const reboot=new LocalProjectCatalog({dir,vault:native,readHead:async()=>shaA});
    const selected=await reboot.select({repository:'org/native',ref:'main'});
    assert.equal(selected.status.selected.repository,'org/native');
    assert.equal(selected.env.TALLY_GITHUB_READ_TOKEN,token);
  }finally{await rm(dir,{recursive:true,force:true});}
});

test('missing native keyring cannot persist a configured project',async()=>{
  const dir=await mkdtemp(join(tmpdir(),'tally-denied-keyring-'));
  try{
    const vault={seal:async()=>{throw Error('CREDENTIAL_STORE_UNAVAILABLE');},
      open:async()=>{throw Error('CREDENTIAL_STORE_UNAVAILABLE');}};
    const catalog=new LocalProjectCatalog({dir,vault,readHead:async()=>shaA});
    await assert.rejects(catalog.add({repository:'org/native',ref:'main',
      token:'github_pat_test_credential'}),/CREDENTIAL_STORE_UNAVAILABLE/);
    assert.deepEqual((await catalog.status()).projects,[]);
  }finally{await rm(dir,{recursive:true,force:true});}
});
