# SDK preflight — real 0.11.2 artifact available

Executed in current assistant Linux environment; no local Agent, no network npm installation, no CI. This proves the ability to consume the preserved SDK on this environment, not a fresh clean install or the v2.1.3 domain implementation.

Source: DomainHarness #1612 R65 accepted internal merge 18384259e36ba56d937416ed5262bac1de6689ae; R64 reviewed head 566cd3129a6ef51bfc40e7d4a018530cabd965a8. R63 evidence [folder](https://drive.google.com/drive/folders/1Z7sdh4fNC2kxgActEtaeT4e00tTNvUIX), actual preserved [SDK pack](https://drive.google.com/file/d/1TQVksD9CJbapqEMQ3gUymkGvVomv1jCN/view) 27,220 bytes, SHA256 a3cd22d2a514fb2f6321218b1dba27b9bd5b17b6fac7bbe2daf00b02cc25a5ff independently computed; matches formal R65 receipt. [Linux consumer archive](https://drive.google.com/file/d/1LHedFkUgAgOzZ97SXWG0W-LtiXEVlFFi/view) 13,667,161 bytes SHA256 e63c173b1a2577d322eec4c6a09bbd039d890508faf777747a7864546220cb2e computed from bytes, matches preserved [index](https://drive.google.com/file/d/1-dVNEgxHwVwzYim49a7wY4mamHlzqB2r/view). Index actual 268 entries; only selected archive/pack identity checked, not all 268. Archive's consumer/node_modules subtree (533 members) extracted with safe data filter; SDK package directory compared recursively to separately unpacked exact pack, diff exit 0. Used that node_modules in a new temporary execution directory; did not run archived consumer code. No repository binary copy. Node v22.16.0 npm 10.9.2. No network-based clean install, no new Windows coverage, no registry release claim.

Actual declarations read from verified pack: standard-query.d.ts, standard-workflow.d.ts, native-workflow-composition.d.ts. Actual root import of BoundedQueryElement and StandardWorkflowSession succeeded. Reproducible smoke from temporary directory with the preserved dependencies:

```js
import assert from 'node:assert/strict';
import { BoundedQueryElement, StandardWorkflowSession } from '@kaicreator/domain-harness';
const q = new BoundedQueryElement();
const records = [{id:'K02.F2.1',kind:'concept'},{id:'K03.F2.1',kind:'concept'},{id:'DP-K03-R19-01',kind:'procedure'}];
const select = (limits) => {
  const r=q.select(records,x=>x.kind==='concept',limits);
  return r.complete ? {status:'COMPLETE',value:r.items.map(x=>x.id)}
    : {status:'UNKNOWN',reason:{code:'QUERY_INCOMPLETE',message:r.truncationReason}};
};
const run = (limits,budget={maxCalls:1,maxSteps:2,maxDepth:0}) =>
  new StandardWorkflowSession(budget).run(root=>root.invokeOutcome(()=>select(limits)));
const a=await run({maxVisited:3,maxReturned:4});
assert.deepEqual(a,{status:'COMPLETE',value:['K02.F2.1','K03.F2.1']});
const b=await run({maxVisited:2,maxReturned:4});
assert.equal(b.status,'UNKNOWN');assert.equal(b.reason.code,'QUERY_INCOMPLETE');
const c=await run({maxVisited:3,maxReturned:4},{maxCalls:0,maxSteps:2,maxDepth:0});
assert.equal(c.status,'FAILED');assert.equal(c.reason.code,'WORKFLOW_CALL_BUDGET');
console.log(JSON.stringify({node:process.version,a,b,c}));
```

Observed exit 0, stdout:
```json
{"node":"v22.16.0","a":{"status":"COMPLETE","value":["K02.F2.1","K03.F2.1"]},"b":{"status":"UNKNOWN","reason":{"code":"QUERY_INCOMPLETE","message":"VISIT_LIMIT"}},"c":{"status":"FAILED","reason":{"code":"WORKFLOW_CALL_BUDGET","message":"Workflow could not complete"}}}
```

The records are three illustrative IDs, not an executed abstraction procedure. Actual SDK availability removes the previous acquisition obstacle in this environment. T2 still needs reviewed design, real TypeScript implementation, static generation checks where applicable, actual package and compiler/dependency evidence, and tests. Neither archive nor its embedded native deps automatically qualify another target or product release.
