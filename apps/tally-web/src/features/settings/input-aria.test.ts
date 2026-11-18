// #449 U3 — programmatic label/description contract for Input/TextArea.
// RED first (2026-11-19): rendered against the pre-fix component this file
// must fail on the missing aria-describedby wiring and the rest-spread
// overrides; after the fix every assertion below is the GREEN criterion set
// recorded in the #449 R95 admission (comment 6988445631).
// Renders via vite ssrLoadModule (TSX) + react-dom/server so the markup
// assertions run under the plain `node --test` glob without new deps.
import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'vite';
import {renderToStaticMarkup} from 'react-dom/server';
import React from 'react';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const here=path.dirname(fileURLToPath(import.meta.url));
const root=path.resolve(here,'../../..');

type Comp=React.FC<Record<string,unknown>>;

test('Input/TextArea programmatic label and description contract',async t=>{
  const server=await createServer({root,logLevel:'error',
    server:{middlewareMode:true},optimizeDeps:{noDiscovery:true}});
  t.after(()=>{void server.close();});
  const mod=await server.ssrLoadModule('/src/components/ui/Input.tsx') as
    {Input:Comp;TextArea:Comp};
  const render=(comp:Comp,props:Record<string,unknown>)=>
    renderToStaticMarkup(React.createElement(comp,props));
  const attr=(html:string,name:string)=>{
    const m=new RegExp(`${name}="([^"]*)"`).exec(html);
    return m?m[1]:null;
  };

  await t.test('label htmlFor binds the generated id and hint is described',()=>{
    const html=render(mod.Input,{label:'端点',hint:'ok 状态提示',value:''});
    const id=/<input id="([^"]+)"/.exec(html)?.[1]??null;
    assert.ok(id,'input renders an id');
    assert.equal(attr(html,'<label for'),id,'label htmlFor equals input id');
    assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
    assert.ok(html.includes(`id="${id}-hint"`),'hint element carries the description id');
    assert.ok(html.includes('ok 状态提示'),'hint text renders');
    assert.equal(attr(html,'aria-invalid'),null,'no aria-invalid without error');
  });

  await t.test('error replaces hint in the description chain and sets aria-invalid',()=>{
    const html=render(mod.Input,{label:'端点',error:'超出范围',hint:'不该渲染',value:''});
    const id=/<input id="([^"]+)"/.exec(html)?.[1]??null;
    assert.ok(id);
    assert.equal(attr(html,'aria-describedby'),`${id}-error`);
    assert.ok(html.includes(`id="${id}-error"`),'error element carries the description id');
    assert.ok(html.includes('role="alert"'),'error keeps role=alert');
    assert.ok(html.includes('超出范围')&&!html.includes('不该渲染'),'error wins over hint');
    assert.equal(attr(html,'aria-invalid'),'true','error forces aria-invalid=true');
  });

  await t.test('caller-supplied id is honored for label/description binding',()=>{
    const html=render(mod.Input,{id:'my-field',label:'Token 预算',error:'必填'});
    assert.equal(attr(html,'<input id'),'my-field');
    assert.equal(attr(html,'<label for'),'my-field');
    assert.equal(attr(html,'aria-describedby'),'my-field-error');
    assert.ok(html.includes('id="my-field-error"'));
  });

  await t.test('caller-supplied aria-describedby is preserved and merged',()=>{
    const html=render(mod.Input,{'aria-describedby':'caller-x',error:'错',value:''});
    const id=/<input id="([^"]+)"/.exec(html)?.[1]??null;
    assert.ok(id);
    const described=attr(html,'aria-describedby')??'';
    const parts=described.split(' ');
    assert.ok(parts.includes('caller-x'),'caller describedby kept');
    assert.ok(parts.includes(`${id}-error`),'error description appended');
    assert.equal(parts.length,2,'no duplicate description entries');
  });

  await t.test('clearing the error leaves no stale description reference',()=>{
    const html=render(mod.Input,{label:'端点',error:null,hint:'留空 = 不限制',value:''});
    const id=/<input id="([^"]+)"/.exec(html)?.[1]??null;
    assert.ok(id);
    assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
    assert.ok(!html.includes(`${id}-error`),'no stale error id in markup');
    assert.equal(attr(html,'aria-invalid'),null,'cleared error clears aria-invalid');
  });

  await t.test('no description attribute when neither hint nor error given',()=>{
    const html=render(mod.Input,{label:'端点',value:''});
    assert.equal(attr(html,'aria-describedby'),null);
    assert.equal(attr(html,'aria-invalid'),null);
  });

  await t.test('TextArea mirrors the same contract',()=>{
    const labeled=render(mod.TextArea,{label:'说明',hint:'可选',rows:4});
    const id=/<textarea id="([^"]+)"/.exec(labeled)?.[1]??null;
    assert.ok(id,'textarea renders an id');
    assert.equal(attr(labeled,'<label for'),id);
    assert.equal(attr(labeled,'aria-describedby'),`${id}-hint`);
    assert.ok(labeled.includes(`id="${id}-hint"`));
    const errored=render(mod.TextArea,{label:'说明',error:'过长'});
    assert.equal(attr(errored,'aria-invalid'),'true');
    assert.ok(errored.includes('role="alert"'));
    const withCaller=render(mod.TextArea,{id:'ta-1','aria-describedby':'keep-me',hint:'h'});
    assert.equal(attr(withCaller,'aria-describedby'),'keep-me ta-1-hint');
  });

  await t.test('English labels render through the same wiring',()=>{
    const html=render(mod.Input,{label:'Endpoint',hint:'unlimited when empty',value:''});
    const id=/<input id="([^"]+)"/.exec(html)?.[1]??null;
    assert.ok(id);
    assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
    assert.ok(html.includes('unlimited when empty'));
  });
});
