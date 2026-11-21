import test from 'node:test';
import assert from 'node:assert/strict';
import {PINNED_DSH_VERSION,DSH_PERMISSION_MODES,resolveDshSelection,dshConfigFromEnv}
  from '../src/dsh-config.mjs';

test('provider selection is default-off and requires two explicit gates',()=>{
  assert.deepEqual(resolveDshSelection({}),{selected:false,providerId:'dsh',
    reason:'NOT_SELECTED_DEFAULT_OFF'});
  assert.deepEqual(resolveDshSelection({TALLY_DSH_ENABLED:'1'}),{selected:false,
    providerId:'dsh',reason:'NOT_SELECTED_DEFAULT_OFF'});
  assert.deepEqual(resolveDshSelection({TALLY_ATTEMPT_PROVIDER:'codex',
    TALLY_DSH_ENABLED:'1'}),{selected:false,providerId:'dsh',
    reason:'NOT_SELECTED_DEFAULT_OFF'});
  assert.deepEqual(resolveDshSelection({TALLY_ATTEMPT_PROVIDER:'dsh'}),
    {selected:false,providerId:'dsh',reason:'EXPLICIT_ENABLE_GATE_MISSING'});
  assert.deepEqual(resolveDshSelection({TALLY_ATTEMPT_PROVIDER:'dsh',
    TALLY_DSH_ENABLED:'1'}),{selected:true,providerId:'dsh',reason:'SELECTED'});
});

test('pinned version default tracks the verified @deepseek-ai/dsh release',()=>{
  assert.equal(PINNED_DSH_VERSION,'0.2.2-rc.5');
  assert.deepEqual([...DSH_PERMISSION_MODES],['read-only','workspace-write','danger-full-access']);
});

test('env config: defaults, overrides and credential passthrough',()=>{
  const config=dshConfigFromEnv({DEEPSEEK_API_KEY:'sk-test'});
  assert.equal(config.pinnedVersion,PINNED_DSH_VERSION);
  assert.equal(config.permissionMode,'workspace-write');
  assert.equal(config.timeoutMs,300000);
  assert.equal(config.pipeGraceMs,5000);
  assert.equal(config.deepseekApiKey,'sk-test');
  assert.equal(config.allowDangerousFullAccess,false);
  const override=dshConfigFromEnv({TALLY_DSH_PERMISSION_MODE:'read-only',
    TALLY_DSH_PINNED_VERSION:'0.1.5-rc.1',TALLY_DSH_TIMEOUT_MS:'5000',
    TALLY_DSH_PIPE_GRACE_MS:'250',TALLY_DSH_BIN_PATH:'C:\\Public\\dsh.exe'});
  assert.equal(override.permissionMode,'read-only');
  assert.equal(override.pinnedVersion,'0.1.5-rc.1');
  assert.equal(override.timeoutMs,5000);
  assert.equal(override.pipeGraceMs,250);
  assert.equal(override.binPath,'C:\\Public\\dsh.exe');
  assert.equal(dshConfigFromEnv({}).deepseekApiKey,null);
});

test('invalid permission modes and ungated danger-full-access are refused',()=>{
  assert.throws(()=>dshConfigFromEnv({TALLY_DSH_PERMISSION_MODE:'total-control'}),
    e=>e.code==='DSH_CONFIG_INVALID_PERMISSION_MODE');
  assert.throws(()=>dshConfigFromEnv({TALLY_DSH_PERMISSION_MODE:'danger-full-access'}),
    e=>e.code==='DSH_CONFIG_DANGER_FULL_ACCESS_NOT_PERMITTED');
  assert.equal(dshConfigFromEnv({TALLY_DSH_PERMISSION_MODE:'danger-full-access',
    TALLY_DSH_ALLOW_DANGER_FULL_ACCESS:'1'}).permissionMode,'danger-full-access');
  assert.throws(()=>dshConfigFromEnv({TALLY_DSH_TIMEOUT_MS:'soon'}),
    e=>e.code==='DSH_CONFIG_INVALID_TIMEOUT');
});
