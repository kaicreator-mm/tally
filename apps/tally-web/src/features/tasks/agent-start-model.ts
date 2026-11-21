/** One selected Builder task may start only under exact Host Task/Claim/SHA. */
import {canonicalTaskIssueNumber} from '../task-actions/model.ts';
export interface AgentCandidate{
  connected:boolean;selectedTaskIssue?:number;sourceSha?:string;
}
export function agentStartDecision({
  taskId,claimState,sourceSha,host
}:{
  taskId:string|null;claimState:string|null;sourceSha:string|null;
  host:AgentCandidate|null
}):{enabled:boolean;reason:string;issueNumber:number|null}{
  const issueNumber=taskId?canonicalTaskIssueNumber(taskId):null;
  if(issueNumber===null||!sourceSha)
    return {enabled:false,reason:'TASK_SOURCE_NOT_SELECTED',issueNumber};
  if(claimState!=='ACCEPTED')
    return {enabled:false,reason:'CLAIM_NOT_ACCEPTED',issueNumber};
  if(host?.connected!==true)
    return {enabled:false,reason:'AGENT_HOST_NOT_MOUNTED',issueNumber};
  if(host.selectedTaskIssue!==issueNumber||host.sourceSha!==sourceSha)
    return {enabled:false,reason:'HOST_TASK_SOURCE_MISMATCH',issueNumber};
  return {enabled:true,reason:'HOST_READY',issueNumber};
}

/** New-project issue is NOT a qualified Frozen DAG Task. This function only
 * gates visibility of a user-requested Host action. K/C still re-checks the
 * accepted CAS Claim, native GitHub witness and allowed write-set at runtime.
 */
export function draftAgentStartRequest({
  issueNumber,repository,versionId,sourceSha,githubIssueState,nativeWorkflow,
  sourceCurrentness,agentHost,claimHost
}:{
  issueNumber:number;repository:string;versionId:string;sourceSha:string;
  githubIssueState:'open'|'closed';
  nativeWorkflow:'PLANNED'|'READY'|'NONCANONICAL';
  sourceCurrentness:'CURRENT'|'STALE';
  agentHost:AgentCandidate|null;
  claimHost:{connected:boolean;repository?:string;versionId?:string;
    issueNumber?:number;sourceSha?:string}|null;
}):{requestable:boolean;reason:string}{
  if(githubIssueState!=='open'||nativeWorkflow!=='READY'||
     sourceCurrentness!=='CURRENT')
    return {requestable:false,reason:'DRAFT_NOT_CURRENT_READY'};
  if(claimHost?.connected!==true||claimHost.repository!==repository||
     claimHost.versionId!==versionId||
     claimHost.issueNumber!==issueNumber||claimHost.sourceSha!==sourceSha)
    return {requestable:false,reason:'CLAIM_HOST_NOT_PINNED'};
  if(agentHost?.connected!==true||
     agentHost.selectedTaskIssue!==issueNumber||
     agentHost.sourceSha!==sourceSha)
    return {requestable:false,reason:'AGENT_HOST_NOT_PINNED'};
  return {requestable:true,reason:'HOST_K_REVALIDATION_REQUIRED'};
}
