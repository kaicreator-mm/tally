// #449 U3 — programmatic label/description contract for Input/TextArea.
// RED first (2026-11-19): rendered against the pre-fix component this file
// fails on the missing aria-describedby wiring and the rest-spread overrides;
// after the fix every assertion below is the GREEN criterion set recorded in
// the #449 R95 admission (comment 6988445631).
// View-level render tests: Node cannot load .tsx directly, so the component is
// bundled once with the repo's pinned esbuild toolchain and rendered through
// react-dom/server; without a toolchain checkout the render tests skip visibly
// (same convention as the workbench view gate — the dep-free source gate runs
// before any npm install by design).
import test from 'node:test';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdirSync, rmSync} from 'node:fs';
import {join, dirname, resolve} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';

const viewToolchainAvailable=(()=>{
  const require=createRequire(import.meta.url);
  for(const pkg of ['esbuild','react','react-dom/server']){
    try{require.resolve(pkg);}catch{return false;}
  }
  return true;
})();
const viewSkipReason=viewToolchainAvailable?false:
  'needs esbuild+react toolchain (npm --prefix apps/tally-web install); dep-free source gate skips view render';

// The bundle must NOT inline react: useId runs inside the bundled copy while
// react-dom/server binds the outer one, and the dual-instance dispatcher is
// null. Keeping react external and writing the bundle under the package's
// node_modules/.cache lets the bare imports resolve to the SAME react copies
// the test (and react-dom) use — one dispatcher, hooks work.
const pkgRoot=resolve(dirname(fileURLToPath(import.meta.url)),'../../..');
const bundleDir=join(pkgRoot,'node_modules','.cache','i449-u3-input-aria');
const outfile=join(bundleDir,'input.bundle.mjs');

type Comp=React.FC<Record<string,unknown>>;
let inputModule:Record<string,unknown>|null=null;
async function renderInput(comp:'Input'|'TextArea',
  props:Record<string,unknown>):Promise<string>{
  const {buildSync}=await import('esbuild') as {buildSync:(options:{
    entryPoints:string[];outfile:string;bundle:boolean;format:'esm';
    platform:'node';jsx:'automatic';external:string[];logLevel:'silent'})=>unknown};
  const react=await import('react') as {createElement:(
    component:unknown,props:Record<string,unknown>)=>unknown};
  const reactDomServer=await import('react-dom/server') as
    {renderToStaticMarkup:(element:unknown)=>string};
  if(!inputModule){
    mkdirSync(bundleDir,{recursive:true});
    process.on('exit',()=>{try{rmSync(bundleDir,{recursive:true,force:true})}
      catch{/* best effort cleanup */}});
    buildSync({entryPoints:[join(pkgRoot,'src','components','ui','Input.tsx')],
      outfile,bundle:true,format:'esm',platform:'node',jsx:'automatic',
      external:['react','react/jsx-runtime','react-dom'],logLevel:'silent'});
    inputModule=await import(pathToFileURL(outfile).href) as Record<string,unknown>;
  }
  const Component=inputModule[comp] as Comp;
  return reactDomServer.renderToStaticMarkup(react.createElement(Component,props));
}
const attr=(html:string,name:string)=>{
  const m=new RegExp(`${name}="([^"]*)"`).exec(html);
  return m?m[1]:null;
};
const firstId=(html:string,tag:'input'|'textarea')=>
  new RegExp(`<${tag} id="([^"]+)"`).exec(html)?.[1]??null;

test('label htmlFor binds the generated id and hint is described',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{label:'端点',hint:'ok 状态提示',value:''});
    const id=firstId(html,'input');
    assert.ok(id,'input renders an id');
    assert.equal(attr(html,'<label for'),id,'label htmlFor equals input id');
    assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
    assert.ok(html.includes(`id="${id}-hint"`),'hint element carries the description id');
    assert.ok(html.includes('ok 状态提示'),'hint text renders');
    assert.equal(attr(html,'aria-invalid'),null,'no aria-invalid without error');
  });

test('error replaces hint in the description chain and sets aria-invalid',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{label:'端点',error:'超出范围',hint:'不该渲染',value:''});
    const id=firstId(html,'input');
    assert.ok(id);
    assert.equal(attr(html,'aria-describedby'),`${id}-error`);
    assert.ok(html.includes(`id="${id}-error"`),'error element carries the description id');
    assert.ok(html.includes('role="alert"'),'error keeps role=alert');
    assert.ok(html.includes('超出范围')&&!html.includes('不该渲染'),'error wins over hint');
    assert.equal(attr(html,'aria-invalid'),'true','error forces aria-invalid=true');
  });

test('caller-supplied id is honored for label/description binding',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{id:'my-field',label:'Token 预算',error:'必填'});
    assert.equal(attr(html,'<input id'),'my-field');
    assert.equal(attr(html,'<label for'),'my-field');
    assert.equal(attr(html,'aria-describedby'),'my-field-error');
    assert.ok(html.includes('id="my-field-error"'));
  });

test('caller-supplied aria-describedby is preserved and merged',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{'aria-describedby':'caller-x',error:'错',value:''});
    const id=firstId(html,'input');
    assert.ok(id);
    const described=(attr(html,'aria-describedby')??'').split(' ');
    assert.ok(described.includes('caller-x'),'caller describedby kept');
    assert.ok(described.includes(`${id}-error`),'error description appended');
    assert.equal(described.length,2,'no duplicate description entries');
  });

test('clearing the error leaves no stale description reference',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{label:'端点',error:null,hint:'留空 = 不限制',value:''});
    const id=firstId(html,'input');
    assert.ok(id);
    assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
    assert.ok(!html.includes(`${id}-error`),'no stale error id in markup');
    assert.equal(attr(html,'aria-invalid'),null,'cleared error clears aria-invalid');
  });

test('no description attribute when neither hint nor error given',
  {skip:viewSkipReason},async()=>{
    const html=await renderInput('Input',{label:'端点',value:''});
    assert.equal(attr(html,'aria-describedby'),null);
    assert.equal(attr(html,'aria-invalid'),null);
  });

test('TextArea mirrors the same contract',{skip:viewSkipReason},async()=>{
  const labeled=await renderInput('TextArea',{label:'说明',hint:'可选',rows:4});
  const id=firstId(labeled,'textarea');
  assert.ok(id,'textarea renders an id');
  assert.equal(attr(labeled,'<label for'),id);
  assert.equal(attr(labeled,'aria-describedby'),`${id}-hint`);
  assert.ok(labeled.includes(`id="${id}-hint"`));
  const errored=await renderInput('TextArea',{label:'说明',error:'过长'});
  assert.equal(attr(errored,'aria-invalid'),'true');
  assert.ok(errored.includes('role="alert"'));
  const withCaller=await renderInput('TextArea',{id:'ta-1','aria-describedby':'keep-me',hint:'h'});
  assert.equal(attr(withCaller,'aria-describedby'),'keep-me ta-1-hint');
});

test('English labels render through the same wiring',{skip:viewSkipReason},async()=>{
  const html=await renderInput('Input',{label:'Endpoint',hint:'unlimited when empty',value:''});
  const id=firstId(html,'input');
  assert.ok(id);
  assert.equal(attr(html,'aria-describedby'),`${id}-hint`);
  assert.ok(html.includes('unlimited when empty'));
});
