/**
 * Scoped negative test for the BUILD-sealed T04b C output boundary.
 * Fake provider bytes are test-only; the actual deploy path separately uses
 * Host pre-load SHA256 verification and only the reviewed packaged provider.
 */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync,mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {pathToFileURL} from 'node:url';
const raw=readFileSync(new URL('./native-readonly-shim.mjs',import.meta.url),'utf8');
const temp=mkdtempSync(join(tmpdir(),'tally-native-shim-output-'));
const token='UNIT_TEST_SECRET_DO_NOT_LOG_8f5bca';
const sourceSha='a'.repeat(40);
const input={sourceSha};
try{
  for(const [name,provider,expectedReason] of [
    ['exception',`export async function observeReadiness(){throw new Error('private:'+${JSON.stringify(token)});}`,'NATIVE_PROVIDER_THREW'],
    ['getter',`export async function observeReadiness(){return {get status(){throw new Error('private:'+${JSON.stringify(token)});}};}`,'NATIVE_PROVIDER_THREW'],
    ['malformed',`export async function observeReadiness(){return {status:'UNKNOWN',reason:'BOGUS',sourceRef:'wrong',sourceSha:'a'.repeat(40),observationCount:0,authorized:true,taskPass:true,token:${JSON.stringify(token)}};}`,'UNVERIFIED_NATIVE_PROVIDER_OUTPUT'],
    ['valid-shape-with-extra',`export async function observeReadiness(){return {status:'UNKNOWN',reason:'NO_ADS_V5_REPORTS',sourceRef:'https://github.com/kaicreator-mm/tally/issues/111',sourceSha:'a'.repeat(40),observationCount:4,authorized:true,admitted:true,taskPass:true,releasePass:true,rawToken:${JSON.stringify(token)}};}`,'NO_ADS_V5_REPORTS']
  ]) {
    const p=join(temp,name+'.mjs');
    writeFileSync(p,provider);
    const source=raw.replace("'@kaicreator/tally-native-reader-bundle'",JSON.stringify(pathToFileURL(p).href));
    assert.notEqual(source,raw,'fixture module must be substituted');
    const entry='data:text/javascript;base64,'+Buffer.from(source).toString('base64');
    const shim=await import(entry);
    const value=await shim.observeReadiness(input);
    assert.equal(value.reason,expectedReason,name);
    assert.equal(value.authorized,false);
    assert.equal(value.admitted,false);
    assert.equal(value.taskPass,false);
    assert.equal(value.releasePass,false);
    assert.equal(JSON.stringify(value).includes(token),false,'no raw Provider secret/exception may cross C boundary');
    assert.deepEqual(Object.keys(value).sort(),[
      'admitted','authorized','observationCount','reason','releasePass',
      'sourceRef','sourceSha','state','status','taskPass'].sort());
  }
  console.log('T04B_SEALED_C_OUTPUT_NEGATIVES=PASS');
}finally{rmSync(temp,{recursive:true,force:true});}
