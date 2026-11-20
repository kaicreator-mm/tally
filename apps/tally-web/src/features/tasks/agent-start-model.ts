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
