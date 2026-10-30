// CI-only native response replay. No credential or live-GitHub claim is made.
if (process.env.T06L_WITNESS_REPLAY === '1') {
  const sourceSha = process.env.TALLY_CLAIM_SOURCE_SHA;
  const commentId = Number(process.env.TALLY_CLAIM_COMMENT_ID);
  const issueRef = 'https://github.com/kaicreator-mm/tally/issues/169';
  const apiIssue = 'https://api.github.com/repos/kaicreator-mm/tally/issues/169';
  const date = '2026-10-29T23:35:02Z';
  const body = `<!-- ads:github -->
\`\`\`yaml
schema: ads/github/v3
kind: event
data:
  schema: ai-dev/event-v2
  event: DISPATCH_CLAIMED
  actor_role: builder
  operator_kind: other
  operator_id: zcode:t125:kernel:1
  task: '#169'
  issue: '#169'
  sha: ${sourceSha}
  dispatch_id: dispatch:169:t06l:kernel:20261030:1
  dispatch_state: CLAIMED
  protected_claim_key: kaicreator-mm/tally#169:builder:__default__
  admission_generation: 1
\`\`\`
`;
  const comment = {
    id: commentId,
    body,
    html_url: `${issueRef}#issuecomment-${commentId}`,
    issue_url: apiIssue,
    created_at: date,
    updated_at: date,
    user: {login: 'kaicreator-mm'}
  };
  const issue = {
    id: 1690,
    number: 169,
    html_url: issueRef,
    state: 'open',
    updated_at: date,
    labels: [{name: 'type:task'}, {name: 'state:ready'}]
  };
  globalThis.fetch = async input => {
    const url = new URL(input);
    let value, status = 200;
    if (url.pathname === '/repos/kaicreator-mm/tally/issues/169') value = issue;
    else if (url.pathname === `/repos/kaicreator-mm/tally/commits/${sourceSha}`) value = {sha: sourceSha};
    else if (url.pathname === '/repos/kaicreator-mm/tally/issues/169/dependencies/blocked_by') value = [];
    else if (url.pathname === '/repos/kaicreator-mm/tally/issues/169/comments') value = [comment];
    else if (url.pathname === `/repos/kaicreator-mm/tally/issues/comments/${commentId}`) value = comment;
    else if (url.pathname === '/user') value = {login: 'kaicreator-mm'};
    else {
      status = 404;
      value = {message: 'fixture-not-found'};
    }
    return new Response(JSON.stringify(value), {
      status,
      headers: {'content-type': 'application/json', 'x-github-request-id': 't06l-replay-fixture'}
    });
  };
}
