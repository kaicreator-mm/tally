import test from 'node:test';
import assert from 'node:assert/strict';
import {resolveTallyConfigDir} from './portable-config-dir.mjs';
import {join} from 'node:path';

test('Windows existing local app data location remains unchanged',()=>{
  assert.equal(resolveTallyConfigDir({platform:'win32',env:{LOCALAPPDATA:'C:\\AppData'},home:'C:\\Users\\test'}),
    join('C:\\AppData','Tally','v2'));
});
test('Linux XDG and macOS Application Support are per user',()=>{
  assert.equal(resolveTallyConfigDir({platform:'linux',env:{XDG_CONFIG_HOME:'/tmp/config'},home:'/home/dev'}),'/tmp/config/tally/v2');
  assert.equal(resolveTallyConfigDir({platform:'linux',env:{},home:'/home/dev'}),'/home/dev/.config/tally/v2');
  assert.equal(resolveTallyConfigDir({platform:'darwin',env:{},home:'/Users/dev'}),'/Users/dev/Library/Application Support/Tally/v2');
});
test('explicit TALLY_V2_CONFIG_DIR overrides platform defaults',()=>{
  for(const platform of ['linux','darwin','win32'])
    assert.equal(resolveTallyConfigDir({platform,env:{TALLY_V2_CONFIG_DIR:'/private/state'},home:'/tmp'}),'/private/state');
});
