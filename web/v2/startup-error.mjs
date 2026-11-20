/** Map bootstrap failures to credential-free operator diagnostics. */
export function formatStartupError(error) {
  const raw = typeof error?.message === 'string' ? error.message : '';
  const code = error?.code;
  const reason = /^[A-Z][A-Z0-9_]{2,96}$/.test(raw) ? raw :
    code === 'ERR_MODULE_NOT_FOUND' || code === 'MODULE_NOT_FOUND' ?
      'MODULE_LOAD_FAILED' : 'STARTUP_INTERNAL_ERROR';
  const stage = reason === 'SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED' ?
    'source-configuration' :
    reason === 'PINNED_ROSTER_SCOPE_REQUIRED' ? 'roster-host' :
    reason === 'MODULE_LOAD_FAILED' ? 'module-loading' :
    reason.startsWith('HOST_') || reason.startsWith('DUAL_EPOCH_') ?
      'host-mount' : 'startup';
  return `TALLY_UI_START_DENIED: reason=${reason} stage=${stage}\n`;
}
