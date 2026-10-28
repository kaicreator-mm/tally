import test from 'node:test';
import assert from 'node:assert/strict';
import {classifyChangedFiles,classifyCommitDiff} from './v2-change-scope.mjs';
const safe=x=>{
  assert.equal(x.existingProviderFailureWaived,false);
  assert.equal(x.mergeAuthorized,false);
  assert.equal(x.releaseAuthorized,false);
  assert.equal(x.requiredCheckApplicability,'NOT_DETERMINED');
};
test('Product-only docs PR yields a source scope hint, NOT CI/Freeze authorization',()=>{
  const x=classifyChangedFiles([
    'docs/product/v2/TALLY_V2_PRD_DOMAIN_PACKAGE_FIRST_R0.md',
    'docs/research/v2/OUTPUT_LEDGER.md'
  ]);
  assert.equal(x.classification,'DOCS_ONLY');
  assert.equal(x.goSourceAffected,false);
  assert.equal(x.docsOnly,true);safe(x);
});
test('isolated JS neutral LLM source yields Node source hint, not Go green',()=>{
  const x=classifyChangedFiles([
    'packages/tally-llm-contract/src/index.mjs',
    '.github/workflows/v2-dh-v0103-neutral-llm-contract.yml'
  ]);
  assert.equal(x.classification,'V2_NODE_OR_SDK_SOURCE');
  assert.equal(x.v2NodeAffected,true);
  assert.equal(x.goSourceAffected,false);safe(x);
});
test('selected public K/B/C/Host integration source is Node v2, not plain docs',()=>{
  const x=classifyChangedFiles([
    'experiments/v2/selected-business-build/producer.mjs',
    'packages/tally-engineering-business/src/index.mjs'
  ]);
  assert.equal(x.classification,'V2_NODE_OR_SDK_SOURCE');
  assert.equal(x.docsOnly,false);safe(x);
});
test('Go code, integration tests, or quality workflow keep Go impact visible',()=>{
  for(const p of [
    'internal/controlplane/service.go','test/e2e/v13_holdout_fail_test.go',
    'go.mod','scripts/export/main.go','.github/workflows/quality.yml'
  ]){
    const x=classifyChangedFiles([p]);assert.equal(x.goSourceAffected,true,p);
    assert.equal(x.classification,'GO_OR_MIXED_CHANGE',p);safe(x);
  }
});
test('mixed Go+TS cannot be mislabeled pure Node by file order',()=>{
  for(const files of [
    ['packages/tally-github-native-c/src/read-only.mjs','internal/controlplane/service.go'],
    ['internal/controlplane/service.go','docs/product/v2/PRD.md','packages/tally-github-c/src/report-projection.mjs']
  ]){
    const x=classifyChangedFiles(files);
    assert.equal(x.classification,'GO_OR_MIXED_CHANGE');
    assert.equal(x.goSourceAffected,true);
    assert.equal(x.v2NodeAffected,true);safe(x);
  }
});
test('ambiguous files, empty diff, duplicates and path traversal fail closed',()=>{
  const unk=classifyChangedFiles(['.github/scripts/v2-change-scope.mjs']);
  assert.equal(unk.classification,'OTHER_OR_UNKNOWN');
  assert.equal(unk.unknownOrOtherPaths,true);safe(unk);
  for(const files of [[],['README.md','README.md'],['../secret.go'],
    ['/absolute.go'],['foo\\bar.go'],['docs/file.md\0evil']]){
    const x=classifyChangedFiles(files);
    assert.equal(x.classification,'UNKNOWN_SCOPE');safe(x);
  }
  for(const [a,b] of [['notsha','b'.repeat(40)],['a'.repeat(40),'bad']]){
    const x=classifyCommitDiff(a,b);
    assert.equal(x.classification,'UNKNOWN_SCOPE');safe(x);
  }
});
