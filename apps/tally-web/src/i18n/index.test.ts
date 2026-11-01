import test from 'node:test';
import assert from 'node:assert/strict';
import {formatMessage,createT,detectLocale,normalizeLocale,LOCALES,
  rememberLocale,messages} from './index.ts';

test('zh-CN is default locale and en falls back for missing keys',()=>{
  assert.equal(normalizeLocale(null),'zh-CN');
  assert.equal(normalizeLocale('en'),'en');
  assert.deepEqual([...LOCALES],['zh-CN','en']);
  assert.equal(formatMessage('en','app.title'),'Tally v2 · ALM Workspace');
});
test('missing key renders the key itself (never silent)',()=>{
  const t=createT('zh-CN');
  assert.equal(t('not.a.real.key' as never),'not.a.real.key');
});
test('parameter interpolation',()=>{
  const t=createT('zh-CN');
  assert.equal(t('state.not-integrated',{lane:230}),
    '该模块由 #230 交付,尚未集成(UI-E #232)');
  const en=createT('en');
  assert.equal(en('state.not-integrated',{lane:230}),
    'Delivered by lane #230; not integrated yet (UI-E #232)');
});
test('zh and en message tables stay key-aligned',()=>{
  const zhKeys=Object.keys(messages['zh-CN']).sort();
  const enKeys=Object.keys(messages['en']).sort();
  assert.deepEqual(zhKeys,enKeys);
});
test('locale persistence is remembered and best-effort',()=>{
  const store=new Map<string,string>();
  const storage={getItem:(k:string)=>store.get(k)??null,
    setItem:(k:string,v:string)=>store.set(k,v)};
  rememberLocale(storage,'en');
  assert.equal(detectLocale({storage}),'en');
  rememberLocale(undefined,'en'); // no storage: must not throw
  assert.equal(detectLocale({storage:undefined}),'zh-CN');
});
