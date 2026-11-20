/** Public surface of @kaicreator/tally-provider-zcode: the AgentProvider/1
 * factory plus the frozen CLI contract constants (#470-verified). Default-off
 * — nothing here registers the provider; mount it explicitly into
 * createAgentProviderRegistry with an enabledProviderIds entry. */
export {createZcodeAgentProvider,createDefaultOperations,
  ZCODE_PROVIDER_CONSTANTS,zcodeBuilderEnvironment,zcodeTaskPrompt,
  parseZcodeAgentResult} from './zcode-agent-provider.mjs';
