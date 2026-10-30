// CI-only preload and assertion classifier; product consumer remains unchanged.
import './claim-replay-fixture.mjs';
function classify(error) {
  const text = String(error?.message ?? error);
  let code = 105;
  if (text.includes('PROVIDER_INTEGRITY') || text.includes('SEALED_CAPABILITY')) code = 101;
  else if (text.includes('Cannot find package') || text.includes('MODULE_NOT_FOUND')) code = 102;
  else if (text.includes('PUBLISHED_VERIFIED')) code = 103;
  else if (text.includes('CLAIM_')) code = 104;
  else if (text.includes('ADMITTED_PENDING_AGENT_EFFECT')) code = 106;
  else if (text.includes('NOT_ADMITTED')) code = 107;
  else if (text.includes('replay') || text.includes('idempotent')) code = 108;
  else if (text.includes('V10_')) code = 109;
  else if (text.includes('fetch') || text.includes('Response')) code = 110;
  return code;
}
process.on('uncaughtException', error => {
  process.exitCode = classify(error);
});
process.on('unhandledRejection', error => {
  process.exitCode = classify(error);
});
