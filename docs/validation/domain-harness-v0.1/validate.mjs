import assert from 'node:assert/strict';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const entry = process.env.DOMAIN_HARNESS_ENTRY;
if (!entry) throw new Error('DOMAIN_HARNESS_ENTRY must point to the exact-SHA built DomainHarness dist/index.js');

const sdk = await import(pathToFileURL(entry).href);
const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, 'tally');

let aiCalls = 0;
let toolCalls = 0;
const ai = {
  async execute(request) {
    aiCalls += 1;
    assert.equal(request.skillId, 'assess_task');
    return { assessment: 'candidate requires deterministic CompletionContract verification' };
  },
};

const tools = {
  tally_verify_completion_contract: {
    effect: 'none',
    async execute(input) {
      toolCalls += 1;
      assert.equal(typeof input?.assessment, 'string');
      return { contractSatisfied: true, evidenceCount: 2 };
    },
  },
};

const runtime = await sdk.createDomainHarness({ root, sqlitePath: ':memory:', ai, tools });
const started = await runtime.start({
  workflowId: 'main',
  input: { taskId: 'TALLY-VALIDATION-001', evidence: ['tests', 'review'] },
});
const waiting = await runtime.wait(started.runId, { timeoutMs: 5000 });
assert.equal(waiting.status, 'waiting');
assert.equal(aiCalls, 1);
assert.equal(toolCalls, 1);

const completed = await runtime.send(started.runId, {
  type: 'approve',
  payload: { actor: 'human-authority', decision: 'approve' },
});
assert.equal(completed.status, 'completed');
assert.equal(aiCalls, 1);
assert.equal(toolCalls, 1);

console.log(JSON.stringify({
  domain: 'tally',
  runId: completed.runId,
  status: completed.status,
  definitionHash: completed.definitionHash,
  aiCalls,
  toolCalls,
  output: completed.output,
}, null, 2));
