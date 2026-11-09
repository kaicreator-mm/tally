/** Per-environment deployment OBSERVATION. Release qualification is separate. */
export interface DeploymentRecord{
  id:string;versionId:string;environment:string;releaseSha:string|null;
  state:'PENDING'|'RUNNING'|'SUCCEEDED'|'FAILED'|'UNKNOWN';
  health:'HEALTHY'|'DEGRADED'|'UNKNOWN';
  deployedAt:string|null;
  receiptRef:string|null;rollbackReceiptRef:string|null;
}
export function versionDeployments(records:readonly DeploymentRecord[],
  versionId:string):DeploymentRecord[]{
  return records.filter(record=>record.versionId===versionId);
}
export function environmentOverview(records:readonly DeploymentRecord[]){
  const latest=new Map<string,DeploymentRecord>();
  for(const record of records){
    const previous=latest.get(record.environment);
    if(!previous||(record.deployedAt??'')>(previous.deployedAt??''))
      latest.set(record.environment,record);
  }
  return [...latest.values()].sort((a,b)=>a.environment.localeCompare(b.environment));
}
export function deploymentBadge(record:DeploymentRecord){
  return record.state==='FAILED'?'danger':
    record.state==='SUCCEEDED'&&record.health==='HEALTHY'?'ok':'warn';
}
