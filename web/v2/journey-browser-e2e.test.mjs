import test from 'node:test';
import assert from 'node:assert/strict';
import {runJourneyScenarios,probeBrowser,scenarioList} from './journey-browser-e2e.mjs';

const REQUIRED_SCENARIOS=['initial-claim-not-accepted','stale-sha-rejected','csrf-and-origin-exact',
  'cross-repo-claim-rejected','provider-failure-stays-failed','native-read-403','native-read-429',
  'native-read-500','lost-ack-unknown','tests-pass-without-installed-acceptance',
  'review-finding-preserves-stale-check','fresh-canonical-receipt-accepted'];

test('J06 journey harness passes every negative scenario through the real same-origin UI/API',async()=>{
  const witness=await runJourneyScenarios({browser:'none'});
  assert.equal(witness.ok,true,JSON.stringify(witness.failures??[]));
  assert.equal(witness.repository,'kaicreator-mm/tally');
  assert.equal(witness.issue,207);
  assert.match(witness.sourceSha,/^[0-9a-f]{40}$/);
  assert.equal(witness.hostExecution,'INJECTED_RUN_SEAM');
  assert.equal(witness.realWindowsCodex,'NOT_RUN');
  for(const vector of ['V74','V75','V76','V82'])assert.ok(witness.vectorsCovered.includes(vector),vector);
  const names=witness.scenarios.map(entry=>entry.name);
  for(const required of REQUIRED_SCENARIOS)assert.ok(names.includes(required),`missing scenario ${required}`);
  for(const entry of witness.scenarios)assert.equal(entry.ok,true,`${entry.name}: ${entry.detail??''}`);
});

test('GUI witness stays NOT_PROVEN when no real browser is available, never claims success',async()=>{
  const witness=await runJourneyScenarios({browser:'none'});
  assert.equal(witness.gui.browser,'GUI_NOT_PROVEN');
  assert.equal(witness.gui.outcome,'NOT_PROVEN');
  assert.ok(witness.gui.reason==='BROWSER_DISABLED_BY_FLAG'||witness.gui.reason==='BROWSER_NOT_FOUND');
  const text=JSON.stringify(witness);
  assert.equal(/"taskPass":true|"releasePass":true|GUI_PROVEN/.test(text),false);
  assert.equal(text.includes('harness-secret-token'),false);
});

test('browser probe reports absence instead of faking a witness',()=>{
  const missing=probeBrowser('C:/definitely-not-here/journey-browser.exe');
  assert.deepEqual(missing,{found:false,path:null});
  const explicit=probeBrowser(process.execPath);
  assert.equal(explicit.found,true);assert.equal(explicit.path,process.execPath);
});

test('scenario catalogue covers the frozen vector set',async()=>{
  const entries=await Promise.all(scenarioList());
  const names=entries.map(entry=>entry.name);
  assert.deepEqual([...names].sort(),[...REQUIRED_SCENARIOS].sort());
  const vectors=new Set(entries.flatMap(entry=>entry.vectors));
  for(const vector of ['V74','V75','V76','V82'])assert.ok(vectors.has(vector),`vector ${vector} untagged`);
});
