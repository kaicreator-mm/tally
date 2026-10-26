/**
 * T04b BUILD-sealed read-only Capability ABI.
 * Actual existing T03b Octokit/Ajv/YAML implementation is a separately
 * selected and installed npm provider. Never inline its CommonJS dependency
 * closure into Microkernel's import-policy-governed moduleSource.
 */
export {MODULE_ID,PACKAGE_KIND,ABI_VERSION,CAPABILITY_ID,OPERATIONS,observeReadiness}
  from '@kaicreator/tally-native-reader-bundle';
