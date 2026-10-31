import test from 'node:test';
import assert from 'node:assert/strict';
import {renderReleasePanel} from './release-view.mjs';
const sha='a'.repeat(40),target='b'.repeat(40);
test('read-only Release table never grants release based on caller PASS',()=>{
  const view=renderReleasePanel({sourceSha:sha,targetSha:target,
    readiness:{status:'BLOCKED',reason:'HIDDEN_NOT_RUN',releasePass:true,canMerge:true,
      gates:[{name:'hiddenValidation',status:'NOT_RUN',reason:'MISSING'}]}});
  assert.match(view,/ADVISORY_AUTHORITY_IGNORED/);
  assert.match(view,/hiddenValidation/);assert.match(view,/NOT_RUN/);
  assert.match(view,new RegExp(sha));assert.match(view,new RegExp(target));
  assert.equal((view.match(/ disabled aria-disabled="true"/g)||[]).length,3);
  assert.doesNotMatch(view,/<script|onclick=|javascript:/i);
});
test('untrusted source and gate strings are escaped',()=>{
  const html=renderReleasePanel({sourceSha:'not-a-sha',targetSha:'<img>',
    readiness:{status:'FOO',reason:'<img src=x onerror=alert(1)>',
      gates:[{name:'<img src=x>',status:'<script>alert(1)</script>',reason:'" onclick="evil()'}]}});
  assert.match(html,/UNKNOWN/);assert.match(html,/&lt;img/);
  assert.match(html,/&lt;script/);assert.doesNotMatch(html,/<img|<script|onclick="evil/);
});
test('empty data is explicit NOT_RUN and no enabled controls',()=>{
  const html=renderReleasePanel(null);
  assert.match(html,/Release evidence NOT_RUN/);
  assert.match(html,/UNKNOWN/);
  assert.equal((html.match(/<button/g)||[]).length,3);
  assert.equal((html.match(/disabled aria-disabled/g)||[]).length,3);
});
