import { createAgentProviderRegistry, defineAgentProvider,
  createProviderAttemptPin, type AgentProvider, type ProviderExecutionRequest,
} from '../src/index.mjs';
const adapter: AgentProvider = defineAgentProvider({
  id: 'codex', contractVersion: 1, capabilities: [],
  probe: async () => ({state:'AVAILABLE'}),
  execute: async (request:ProviderExecutionRequest) => ({state:'UNKNOWN',reason:'NOT_WIRED',evidence:{taskId:request.taskId}}),
  cancel: async () => ({state:'NOT_RUNNING'}),
  reconcile: async () => ({state:'UNKNOWN',reason:'NOT_WIRED'}),
});
const registry = createAgentProviderRegistry([adapter]);
const pin = createProviderAttemptPin({taskId:'264',attemptId:'attempt:264:r1',sourceSha:'a'.repeat(40),providerId:'codex'});
const selected = registry.resolve({pin});
void selected;
