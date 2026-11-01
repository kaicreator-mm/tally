/** Derived IssueContainer/WorkUnit identity and collision projection. No Claim/CAS authority. */
const SHA=/^[0-9a-f]{40}$/;
const KEY=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const KINDS=new Set(['FEATURE','SOURCE_TASK','REPAIR_CONCERN','REVIEW','VERIFICATION_CHECKPOINT','CAMPAIGN','PLANNING','RELEASE_GATE','UNKNOWN']);
const array=value=>Array.isArray(value)?value:[];
const validRef=value=>typeof value==='string'&&
  /^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull|commit|releases)\/[A-Za-z0-9_.~/#-]+$/.test(value);
const validClaimRef=(repo,number,value)=>validRef(value)&&/#issuecomment-[1-9][0-9]*$/.test(value)&&
  value.startsWith(issueRef(repo,number)+'#issuecomment-');
const intersection=(a,b)=>a.filter(value=>b.includes(value));
const unique=value=>[...new Set(value)];
const issueRef=(repo,number)=>`https://github.com/${repo}/issues/${number}`;
function correctedRefs(corrections){
  return new Set(array(corrections).filter(row=>typeof row.sourceRef==='string').map(row=>row.sourceRef));
}
function claimFor(unit,claims,corrections){
  const history=claims.filter(row=>row.issueNumber===unit.issueNumber&&row.workUnitKey===unit.key).map(row=>{
    const corrected=corrections.has(row.claimRef),native=unit.identityStatus==='OBSERVED'&&
      validClaimRef(unit.repo,unit.issueNumber,row.claimRef)&&
      SHA.test(row.sourceSha??'')&&row.sourceSha===unit.sourceSha&&row.authoritative===true&&!corrected;
    const accepted=native&&row.status==='ACCEPTED',terminal=native&&row.status==='TERMINAL';
    return {claimRef:row.claimRef,attemptId:row.attemptId??null,actor:row.actor??'UNKNOWN',
      status:corrected?'UNKNOWN':accepted?'ACCEPTED':terminal?'TERMINAL':'UNVERIFIED',active:accepted&&!terminal,
      sourceSha:row.sourceSha??null,observedAt:row.observedAt??null};
  });
  const active=history.filter(row=>row.active);
  return {history,active:active.length===1?active[0]:null,
    status:active.length>1?'CONFLICT':active.length===1?'ACCEPTED':history.some(row=>row.status==='UNKNOWN')?'UNKNOWN':
      history.some(row=>row.status==='TERMINAL')?'TERMINAL':'UNVERIFIED'};
}
function collision(repo,left,right){
  if(left.id===right.id||left.identityStatus==='CONFLICT'||right.identityStatus==='CONFLICT')return null;
  const shared=intersection(left.writeSet,right.writeSet),active=left.claim.active&&right.claim.active;
  const exact=left.sourceSha===right.sourceSha&&left.claim.active?.sourceSha===left.sourceSha&&
    right.claim.active?.sourceSha===right.sourceSha;
  const status=shared.length>0&&active&&exact?'CONFIRMED_COLLISION':
    shared.length>0||active&&(left.writeSet.length===0||right.writeSet.length===0)?'POTENTIAL_OVERLAP':null;
  if(!status)return null;
  return {type:'WorkUnitCollision',status,units:[left.id,right.id],sharedWriteSet:shared,
    exactSourceSha:exact?left.sourceSha:null,authorizationNeeded:true,
    mustNot:['DO_NOT_CANCEL_CLAIM','DO_NOT_REORDER_NATIVE_DAG','DO_NOT_MERGE_PULL_REQUEST']};
}
/** Native/public observations only; no free-text identity promotion. */
export function projectWorkUnits(input={}){
  const {repo,observedAsOf,containers=[],claims=[],pullRequests=[],reviews=[],corrections=[],publishedSpans=[],wallOverlapMs=null}=input??{};
  const malformed=typeof repo!=='string'||!/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repo)||
    typeof observedAsOf!=='string'||!Number.isFinite(Date.parse(observedAsOf))||
    [containers,claims,pullRequests,reviews,corrections,publishedSpans].some(value=>!Array.isArray(value)||
      value.some(row=>!row||typeof row!=='object'||Array.isArray(row)))||
    containers.some(container=>!Array.isArray(container.workUnits)||container.workUnits.some(row=>
      !row||typeof row!=='object'||Array.isArray(row)))||
    reviews.some(row=>!Number.isSafeInteger(row.issueNumber)||row.issueNumber<1||!KEY.test(row.concernKey??'')||
      typeof row.revision!=='string'||row.revision.length===0||typeof row.verdict!=='string'||
      !validRef(row.sourceRef)||!row.sourceRef.startsWith(`https://github.com/${repo}/`)||
      !SHA.test(row.sourceSha??'')||!Number.isFinite(Date.parse(row.observedAt)));
  if(malformed)return Object.freeze({status:'UNKNOWN',containers:[],collisions:[],coverage:{comparison:'COUNTS_NOT_PERCENTAGES',
    measuredActive:'UNKNOWN',taskPass:false,releasePass:false},advisoryOnly:true});
  const correctionsSet=correctedRefs(corrections),projected=[],collisions=[];
  for(const container of containers){
    if(!Number.isSafeInteger(container.issueNumber)||container.issueNumber<1||!Array.isArray(container.workUnits))
      return Object.freeze({status:'UNKNOWN',containers:[],collisions:[],coverage:{comparison:'COUNTS_NOT_PERCENTAGES',
        measuredActive:'UNKNOWN',taskPass:false,releasePass:false},advisoryOnly:true});
    const units=[],keyCounts=new Map();
    for(const unit of container.workUnits)keyCounts.set(unit.key,(keyCounts.get(unit.key)??0)+1);
    for(const unit of container.workUnits){
      const keyValid=KEY.test(unit.key??''),kind=KINDS.has(unit.kind)?unit.kind:'UNKNOWN';
      const duplicate=keyCounts.get(unit.key)>1;
      const sourceSha=SHA.test(unit.sourceSha??'')?unit.sourceSha:null;
      const base={repo,issueNumber:container.issueNumber,key:unit.key,sourceSha,
        id:`${repo}#${container.issueNumber}/${unit.key}`,kind,
        featureKey:typeof unit.featureKey==='string'?unit.featureKey:null,
        writeSet:array(unit.writeSet).filter(value=>typeof value==='string'),owner:unit.owner??'UNKNOWN',
        identityStatus:duplicate?'CONFLICT':keyValid&&validRef(unit.sourceRef)&&sourceSha?'OBSERVED':'UNKNOWN',
        sourceRef:validRef(unit.sourceRef)?unit.sourceRef:null,observedAsOf};
      const claim=claimFor(base,claims,correctionsSet),value={...base,claim};units.push(value);
    }
    for(let left=0;left<units.length;left++)for(let right=left+1;right<units.length;right++){
      const found=collision(repo,units[left],units[right]);if(found)collisions.push(found);
    }
    const qualifiedSourceRefs=unique(pullRequests.filter(row=>row.issueNumber===container.issueNumber&&
      row.qualified===true&&Number.isSafeInteger(row.number)&&row.number>0&&
      row.ref===`https://github.com/${repo}/pull/${row.number}`&&SHA.test(row.headSha??'')).map(row=>row.ref));
    projected.push({type:'IssueContainer',issue:`${repo}#${container.issueNumber}`,issueRef:issueRef(repo,container.issueNumber),
      kind:KINDS.has(container.kind)?container.kind:'UNKNOWN',workUnits:units,qualifiedSourceRefs,observedAsOf});
  }
  const allUnits=projected.flatMap(container=>container.workUnits),reviewHistory=reviews.map(row=>({
    type:'ReviewRevision',identityStatus:'OBSERVED',issue:`${repo}#${row.issueNumber}`,concernKey:row.concernKey,revision:row.revision??'UNKNOWN',
    verdict:row.verdict??'UNKNOWN',sourceSha:SHA.test(row.sourceSha??'')?row.sourceSha:null,
    sourceRef:validRef(row.sourceRef)?row.sourceRef:null,observedAt:row.observedAt??null}));
  const activeSpan=publishedSpans.length===1&&publishedSpans[0].visibility==='PUBLIC'&&
    Number.isSafeInteger(publishedSpans[0].activeMs)&&publishedSpans[0].activeMs>=0&&validRef(publishedSpans[0].sourceRef)?
    publishedSpans[0].activeMs:null;
  const coverage={capabilityCount:unique(allUnits.filter(unit=>unit.kind==='FEATURE'&&unit.featureKey).map(unit=>unit.featureKey)).length,
    nativeExecutableTaskCount:unique(allUnits.filter(unit=>unit.kind==='SOURCE_TASK').flatMap(unit=>
      unit.claim.history.filter(row=>row.status==='ACCEPTED').map(row=>row.claimRef))).length,
    campaignCount:projected.filter(container=>container.kind==='CAMPAIGN').length,
    checkpointCount:allUnits.filter(unit=>unit.kind==='VERIFICATION_CHECKPOINT').length,
    reviewAttemptCount:reviewHistory.length,
    repairConcernCount:unique(allUnits.filter(unit=>unit.kind==='REPAIR_CONCERN').map(unit=>unit.key)).length,
    activeClaimCount:unique(allUnits.flatMap(unit=>unit.claim.active?[unit.claim.active.claimRef]:[])).length,
    wallOverlapMs:Number.isSafeInteger(wallOverlapMs)&&wallOverlapMs>=0?wallOverlapMs:null,
    measuredActive:activeSpan===null?'UNKNOWN':'MEASURED',activeMs:activeSpan,
    comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false};
  return Object.freeze({status:'OBSERVED',observedAsOf,containers:projected,collisions,reviewHistory,coverage,advisoryOnly:true});
}
