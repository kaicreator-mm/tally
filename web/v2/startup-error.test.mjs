import test from 'node:test';
import assert from 'node:assert/strict';
import {formatStartupError} from './startup-error.mjs';

test('valid source and roster errors are distinct', () => {
  assert.equal(formatStartupError(new TypeError('SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED')),
    'TALLY_UI_START_DENIED: reason=SELECTED_UI_SOURCE_CONFIGURATION_REQUIRED stage=source-configuration\n');
  assert.equal(formatStartupError(new TypeError('PINNED_ROSTER_SCOPE_REQUIRED')),
    'TALLY_UI_START_DENIED: reason=PINNED_ROSTER_SCOPE_REQUIRED stage=roster-host\n');
});
test('module load failure uses no path or secret', () => {
  const error = new Error('Cannot find module /Users/private/github_pat_SECRET');
  error.code = 'ERR_MODULE_NOT_FOUND';
  assert.equal(formatStartupError(error),
    'TALLY_UI_START_DENIED: reason=MODULE_LOAD_FAILED stage=module-loading\n');
});
test('unexpected failure never leaks credentials', () => {
  const error = new Error('github_pat_SENSITIVE_TOKEN');
  assert.equal(formatStartupError(error),
    'TALLY_UI_START_DENIED: reason=STARTUP_INTERNAL_ERROR stage=startup\n');
  assert.doesNotMatch(formatStartupError(error), /github_pat|SENSITIVE_TOKEN/);
});
test('host mount failure retains bounded reason code', () => {
  assert.equal(formatStartupError(new TypeError('HOST_FROZEN_DAG_REGISTRY_REQUIRED')),
    'TALLY_UI_START_DENIED: reason=HOST_FROZEN_DAG_REGISTRY_REQUIRED stage=host-mount\n');
});
