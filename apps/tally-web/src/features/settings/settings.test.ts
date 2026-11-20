import test from 'node:test';
import assert from 'node:assert/strict';
import {validateEndpoint,validateConcurrency,canToggleProvider,
  validateTokenBudget,assertNoSecrets,persistSelections,resolveScoped}
  from './model.ts';
import type {AgentProviderConfig,WorkspaceDefaults} from './model.ts';

const provider=(over:Partial<AgentProviderConfig>):AgentProviderConfig=>({
  id:'zcode',enabled:false,cliPath:'C:/bin/zcode',hostPrincipal:'operator',
  concurrency:2,sandbox:'none',buildSelected:false,credential:null,...over});
test('endpoint validation rejects embedded credentials and junk',()=>{
  assert.equal(validateEndpoint('https://ci.kaicreator.com'),'ok');
  assert.equal(validateEndpoint('http://127.0.0.1:8000'),'ok');
  assert.equal(validateEndpoint('not a url'),'invalid');
  assert.equal(validateEndpoint('https://user:pass@x.example'),'embedded_credentials');
  assert.equal(validateEndpoint('ftp://x'),'invalid');
});
test('concurrency bounds 1..8 integers',()=>{
  assert.equal(validateConcurrency(1),'ok');
  assert.equal(validateConcurrency(8),'ok');
  assert.equal(validateConcurrency(0),'out_of_range');
  assert.equal(validateConcurrency(9),'out_of_range');
  assert.equal(validateConcurrency(2.5),'out_of_range');
});
test('UI cannot enable a provider the BUILD did not select',()=>{
  const unselected=provider({});
  assert.equal(canToggleProvider(unselected,true).ok,false);
  assert.match(canToggleProvider(unselected,true).reason!,/BUILD_NOT_SELECTED/);
  assert.equal(canToggleProvider(unselected,false).ok,true);
  const selected=provider({buildSelected:true});
  assert.equal(canToggleProvider(selected,true).ok,true);
});
test('token budget bounds and unlimited-UNKNOWN display',()=>{
  assert.equal(validateTokenBudget(null),'ok');
  assert.equal(validateTokenBudget(0),'ok');
  assert.equal(validateTokenBudget(10_000_000),'ok');
  assert.equal(validateTokenBudget(10_000_001),'out_of_range');
  assert.equal(validateTokenBudget(-1),'out_of_range');
});
test('secret-leak guard: settings payloads never carry secret bytes',()=>{
  const cleanDefaults:WorkspaceDefaults={locale:'zh-CN',defaultView:'list',
    cascade:{workspace:'w',project:null,version:null,task:null}};
  const withSecret={...cleanDefaults,
    cascade:{workspace:'ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567',
      project:null,version:null,task:null}};
  const base:any={agentProviders:[],tallyLlm:{credential:null},
    ci:{},git:{},deploymentEnvs:[],defaults:cleanDefaults};
  assertNoSecrets(base); // must not throw
  assert.throws(()=>assertNoSecrets({...base,defaults:withSecret}),
    /SETTINGS_CONTAIN_SECRET_BYTES/);
});
test('persistSelections stores only non-secret selections labeled unconnected',()=>{
  const defaults:WorkspaceDefaults={locale:'zh-CN',defaultView:'board',
    cascade:{workspace:'w',project:'p',version:'v',task:null}};
  const raw=persistSelections(defaults);
  const parsed=JSON.parse(raw);
  assert.equal(parsed.kind,'tally.settings.selections.v1');
  assert.equal(parsed.unconnected,true);
  assert.equal(parsed.defaults.defaultView,'board');
});
test('scope cascade resolves task > version > project > workspace',()=>{
  assert.deepEqual(resolveScoped({workspace:'w'}),
    {value:'w',source:'workspace',authorizer:null});
  assert.deepEqual(resolveScoped({workspace:'w',
    project:{value:'p',source:'project',authorizer:'owner'}}),
    {value:'p',source:'project',authorizer:'owner'});
  assert.deepEqual(resolveScoped({workspace:'w',
    project:{value:'p',source:'project',authorizer:null},
    version:{value:'v',source:'version',authorizer:'freeze'}}),
    {value:'v',source:'version',authorizer:'freeze'});
  assert.deepEqual(resolveScoped({workspace:'w',
    project:{value:'p',source:'project',authorizer:null},
    version:{value:'v',source:'version',authorizer:'freeze'},
    task:{value:'t',source:'task',authorizer:'claim'}}),
    {value:'t',source:'task',authorizer:'claim'});
});
