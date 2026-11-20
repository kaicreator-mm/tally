/** Read-only projection of an already trusted mounted single-Agent Host. */
const SHA=/^[a-f0-9]{40}$/;
export function projectAgentDispatchStatus({
  mounted=false,taskIssueNumber,sourceSha,provider='codex'
}={}){
  if(!mounted)return Object.freeze({connected:false,
    reason:'AGENT_HOST_NOT_MOUNTED'});
  if(!Number.isSafeInteger(taskIssueNumber)||taskIssueNumber<1||
     !SHA.test(sourceSha??'')||
     typeof provider!=='string'||!/^[a-z][a-z0-9-]{0,63}$/.test(provider))
    return Object.freeze({connected:false,reason:'AGENT_HOST_SCOPE_INVALID'});
  return Object.freeze({connected:true,selectedTaskIssue:taskIssueNumber,
    sourceSha,provider,authority:'HOST_PINNED_ONLY'});
}
