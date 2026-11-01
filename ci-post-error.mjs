import {readFileSync} from 'node:fs';
const path = process.argv[2];
if (!path) throw new Error('CI_ERROR_LOG_PATH_REQUIRED');
const body = readFileSync(path, 'utf8');
console.error(body);
try {
  const response = await fetch('http://172.18.0.1:52421/error', {
    method: 'POST',
    headers: {'content-type': 'text/plain; charset=utf-8'},
    body
  });
  if (!response.ok) throw new Error('CI_ERROR_UPLOAD_FAILED');
} catch {
  // The original test exit code is preserved by the caller; no fallback claim.
}
