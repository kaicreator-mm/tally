import test from 'node:test';
import assert from 'node:assert/strict';
import {createCredentialStore} from './credential-store.mjs';

const ID='11111111-2222-4333-8444-555555555555';
const token='github_pat_test_token123';

test('Linux Secret Service stores only opaque reference, no argv secret',async()=>{
  const calls=[];const secrets=new Map();
  const run=async(command,args,input)=>{
    calls.push({command,args,input});
    assert.equal(command,'secret-tool');
    if(args[0]==='store'){secrets.set(args.at(-1),input);return '';}
    if(args[0]==='lookup')return secrets.get(args.at(-1));
    throw new Error('unexpected command');
  };
  const vault=createCredentialStore({platform:'linux',run,idFactory:()=>ID});
  const ref=await vault.seal(token);
  assert.equal(ref,'keyring-v1:'+ID);
  assert.equal(await vault.open(ref),token);
  assert.ok(calls.every(call=>!call.args.join(' ').includes(token)));
  assert.equal(calls[0].input,token);
  assert.equal(calls[1].input,undefined);
});

test('macOS Keychain input is stdin batch only, never process arguments',async()=>{
  const calls=[];
  const run=async(command,args,input)=>{
    calls.push({command,args,input});
    assert.equal(command,'security');
    if(args[0]==='-i')return 'security> ';
    if(args[0]==='find-generic-password')return token;
    throw new Error('unexpected command');
  };
  const vault=createCredentialStore({platform:'darwin',run,idFactory:()=>ID});
  const ref=await vault.seal(token);
  assert.equal(await vault.open(ref),token);
  assert.ok(calls.every(call=>!call.args.join(' ').includes(token)));
  assert.equal(calls[0].command,'security');
  assert.deepEqual(calls[0].args,['-i']);
  assert.match(calls[0].input,/^add-generic-password -U -s com\.kaicreator\.tally\.v2\.github /);
  assert.match(calls[0].input,/\nquit\n$/);
});

test('invalid references, injected token text and missing OS keyring fail closed',async()=>{
  const run=async()=>{throw new Error('secret contains redacted token');};
  for(const platform of ['linux','darwin']){
    const vault=createCredentialStore({platform,run,idFactory:()=>ID});
    await assert.rejects(vault.open('not-an-opaque-ref'),/CREDENTIAL_REFERENCE_INVALID/);
    await assert.rejects(vault.seal('abcd\nmalicious'),/CREDENTIAL_FORMAT_UNSUPPORTED/);
    await assert.rejects(vault.seal(token),error=>{
      assert.equal(error.message,'CREDENTIAL_STORE_UNAVAILABLE');
      assert.doesNotMatch(error.message,/token123/);
      return true;
    });
  }
  const unsupported=createCredentialStore({platform:'freebsd'});
  await assert.rejects(unsupported.seal(token),/CREDENTIAL_PLATFORM_UNSUPPORTED/);
});

test('empty native response is unavailable, not success',async()=>{
  const vault=createCredentialStore({platform:'linux',run:async()=>'',idFactory:()=>ID});
  await assert.rejects(vault.open('keyring-v1:'+ID),/CREDENTIAL_UNAVAILABLE/);
});
