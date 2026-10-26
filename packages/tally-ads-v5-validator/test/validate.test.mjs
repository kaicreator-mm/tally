import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { validateAdsV5Report, REPORT_KINDS, ADS_REPORT_SCHEMA_GIT_BLOB } from '../src/validate.mjs';
const fixture=(name)=>JSON.parse(readFileSync(new URL('../test/fixtures/'+name+'.json',import.meta.url),'utf8'));
const clone=(x)=>structuredClone(x);
test('canonical ADS schema Git blob pinned exactly',()=>{
  assert.equal(ADS_REPORT_SCHEMA_GIT_BLOB,'7478313cc006162432649d734e59c5cd77db5106');
});
for(const [name,kind] of [['summary',REPORT_KINDS.SUMMARY],['feedback',REPORT_KINDS.FEEDBACK],['correction',REPORT_KINDS.CORRECTION]]) {
  test('strict valid v5 '+name+' remains observational only',()=>{
    const x=validateAdsV5Report(fixture(name));
    assert.equal(x.valid,true,JSON.stringify(x.violations));
    assert.equal(x.reportKind,kind);
    assert.equal(x.authorized,false);
    assert.equal(x.taskPass,false);
    assert.equal(x.releasePass,false);
    assert.equal(x.verifiedGitHub,false);
  });
}
test('legacy v4 report cannot silently upgrade',()=>{
  const a=fixture('summary');a.schema='ads/github/v4';assert.equal(validateAdsV5Report(a).valid,false);
});
test('lowercase invented v5 report-kind is not canonical',()=>{
  const a=fixture('summary');a.data.report_kind='summary';assert.equal(validateAdsV5Report(a).valid,false);
});
test('unknown top-level and nested properties fail closed',()=>{
  const a=fixture('feedback');a.unexpected='123';assert.equal(validateAdsV5Report(a).valid,false);
  const b=fixture('feedback');b.data.subject.unexpected=true;assert.equal(validateAdsV5Report(b).valid,false);
});
test('v5 explicit publication clearance is required',()=>{
  const a=fixture('summary');delete a.data.publication;assert.equal(validateAdsV5Report(a).valid,false);
});
test('correction requires notice publisher source sha',()=>{
  const a=fixture('correction');delete a.data.notice_publisher_source_sha;
  assert.equal(validateAdsV5Report(a).valid,false);
});
test('unqualified source cannot claim canonical dispatch',()=>{
  const a=fixture('summary');a.data.subject.source_status='UNPINNED';
  assert.equal(validateAdsV5Report(a).valid,false);
});
test('UNKNOWN metric cannot be fabricated zero',()=>{
  const a=fixture('summary');a.data.metrics[0].value=0;assert.equal(validateAdsV5Report(a).valid,false);
});
test('verifier UNKNOWN must not report successful checks',()=>{
  const a=fixture('summary');a.data.verifier.checked=['hidden'];
  assert.equal(validateAdsV5Report(a).valid,false);
});
test('unknown or missing actor fails strict nested shape',()=>{
  const a=fixture('summary');delete a.data.actor.role;assert.equal(validateAdsV5Report(a).valid,false);
});
test('validator never changes a caller-owned report on failure or success',()=>{
  const a=fixture('summary');const old=JSON.stringify(a);validateAdsV5Report(a);assert.equal(JSON.stringify(a),old);
});
test('JSON string / YAML string are not trusted as parsed authoritative data',()=>{
  assert.equal(validateAdsV5Report(JSON.stringify(fixture('summary'))).valid,false);
  assert.equal(validateAdsV5Report('schema: ads/github/v5').valid,false);
  assert.equal(validateAdsV5Report(null).valid,false);
});
