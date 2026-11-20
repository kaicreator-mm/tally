import test from 'node:test';
import assert from 'node:assert/strict';
import {isValidVersionRef} from '../src/version-ref.mjs';

test('accepts plain main and existing version refs',()=>{
  for(const input of ['main','version/v2','version/test-01'])
    assert.equal(isValidVersionRef(input),true,input);
});
test('rejects unsafe, ambiguous and unrelated refs',()=>{
  for(const input of [null,undefined,'','Main','main/evil','feature/a','version/',
    'refs/heads/main','version/x/y','version/hello world','../main','version/.bad?'])
    assert.equal(isValidVersionRef(input),false,String(input));
});
