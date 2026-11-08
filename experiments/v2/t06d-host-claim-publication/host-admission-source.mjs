/**
 * T06j — privileged Host admission source for the existing T06d controller.
 *
 * Native C proves current GitHub facts; Host configuration proves dispatch and
 * writer identity; the public DH CAS registry proves incompatible reservations.
 * None of these facts alone is a CAS-accepted Claim, K permit or Agent effect.
 * This adapter is read-only: callers must never supply authority booleans or
 * writer identity through a request, comment, B port, Agent or UI endpoint.
 */
import {Octokit} from '@octokit/rest';
import {readNativeHostClaimPreflightFromClient} from '../../../packages/tally-github-native-c/src/task-claim-host-preflight.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from '../../../packages/tally-github-native-c/src/task-claim-host-readback-port.mjs';
import {createHash} from 'node:crypto';
import {TARGET,readDispatchStateEventsFromClient} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';
import {terminalEventForReservation} from '../../../packages/tally-github-native-c/src/task-claim-github-publisher.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human',
  'github-actions','woodpecker','other']);
const PREFIX='tally:v2:claim-admission:';
const HEX64=/^[a-f0-9]{64}$/;
const RESOURCE_FENCE_AGE_MS=15000;
const PHASES=new Set(['PENDING_GITHUB_PUBLISH','PUBLISHED_VERIFIED',
  'PENDING_GITHUB_TERMINALIZATION','TERMINAL_VERIFIED']);
const CLAIM_KEY=new RegExp('^([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#([1-9][0-9]*):'+
  '(builder|reviewer|validator):__default__$');
const CONTEXT=/^[A-Za-z0-9][A-Za-z0-9 .:_/-]{0,127}$/;
const REF_KEYS=['writer','attempt','subprocess','effect','resource','writeSet'];
const readbackFailure=()=>new Error('NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED');

function denied(reason,request){
  return Object.freeze({
    status:'BLOCKED',reason,sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',
    sourceQualified:false,complete:false,
    issueRef:request?.issueRef??null,sourceSha:request?.sourceSha??null,
    activeIncompatibleClaimCount:null,dispatchSourceBound:false,
    publisherSourceBound:false,claimStatus:'NOT_VERIFIED',
    authorized:false,admitted:false,taskPass:false,releasePass:false
  });
}
function validTask(task,repository){
  return task&&Number.isSafeInteger(task.issueNumber)&&task.issueNumber>0&&
    SHA.test(task.sourceSha??'')&&ROLES.has(task.actorRole)&&
    ID.test(task.dispatchId??'')&&KINDS.has(task.operatorKind)&&
    ID.test(task.operatorId??'')&&
    task.protectedClaimKey===undefined;
}
function boundRequest(request,task,repository){
  return request&&typeof request==='object'&&!Array.isArray(request)&&
    request.repository===repository&&request.issueNumber===task.issueNumber&&
    request.issueRef===`https://github.com/${repository}/issues/${task.issueNumber}`&&
    request.sourceSha===task.sourceSha&&request.actorRole===task.actorRole&&
    request.dispatchId===task.dispatchId&&request.operatorKind===task.operatorKind&&
    request.operatorId===task.operatorId&&request.generation===1&&
    request.protectedClaimKey===`${repository}#${task.issueNumber}:${task.actorRole}:__default__`;
}
function validWriter(authority,repository){
  return authority?.mode==='SINGLE_WRITER_ADMISSION'&&
    authority?.exclusiveHostControl===true&&authority?.repository===repository&&
    ID.test(authority?.writerId??'')&&LOGIN.test(authority?.githubWriterLogin??'');
}
function validRegistryRow(row){
  const value=row?.value;
  if(!((Number.isSafeInteger(row?.revision)&&row.revision>=0)||
    (typeof row?.revision==='string'&&row.revision.length>0))||
    !value||typeof value!=='object'||!PHASES.has(value.phase)||
    value.generation!==1||!SHA.test(value.sourceSha??'')||
    !ID.test(value.dispatchId??'')||!ID.test(value.operatorId??'')||
    !KINDS.has(value.operatorKind)||!ID.test(value.writerId??''))return false;
  const match=CLAIM_KEY.exec(value.protectedClaimKey??'');
  if(!match||value.repository!==match[1]||value.actorRole!==match[3]||
    value.issueRef!==`https://github.com/${match[1]}/issues/${match[2]}`)return false;
  if(['PUBLISHED_VERIFIED','PENDING_GITHUB_TERMINALIZATION','TERMINAL_VERIFIED'].includes(value.phase)&&
    (!Number.isSafeInteger(value.commentId)||value.commentId<1||
     value.commentRef!==`${value.issueRef}#issuecomment-${value.commentId}`))return false;
  if(value.phase==='TERMINAL_VERIFIED'&&
    (!Number.isSafeInteger(value.terminalCommentId)||value.terminalCommentId<1||
     value.terminalCommentRef!==`${value.issueRef}#issuecomment-${value.terminalCommentId}`||
     value.protectedClaimKey!==TARGET.protectedClaimKey||value.sourceSha!==TARGET.sourceSha||
     value.dispatchId!==TARGET.dispatchId||value.commentId!==TARGET.claimCommentId||
     value.commentRef!==TARGET.claimCommentRef||typeof value.terminalEvidence!=='string'||!value.terminalEvidence||
     value.actorRole!=='builder'||value.operatorKind!=='other'||value.operatorId!=='zcode:candidate-publication:1'||
     value.writerId!=='tally-host:win32:candidate-publication:1'||value.terminalReservationRevision!==2||
     !ID.test(value.terminalWriterId??'')||!Number.isFinite(Date.parse(value.terminalOccurredAt))))return false;
  return true;
}

/**
 * The default shared-registry policy refuses ALL live foreign Claim keys.
 * An opt-in Host-controlled resource fence can make an exact subset compatible.
 * This is a read-only preflight witness, not a lease acquisition or a K permit:
 * the selected Host MUST hold the same durable, cross-writer resource lease
 * across reserveAndPublish (CAS reservation, GitHub POST/readback and final CAS).
 * Merely passing a boolean from a Task, UI, GitHub comment or Agent is invalid.
 * Legacy writers without the common resource-fence protocol remain incompatible.
 */
function foreignClaimDigest(rows){
  const fields=rows.map(row=>({
    protectedClaimKey:row.protectedClaimKey,phase:row.phase,
    sourceSha:row.sourceSha,dispatchId:row.dispatchId,writerId:row.writerId
  })).sort((a,b)=>a.protectedClaimKey.localeCompare(b.protectedClaimKey));
  return createHash('sha256').update(JSON.stringify(fields)).digest('hex');
}
function validResourceFenceProof(proof,expected){
  if(!proof||typeof proof!=='object'||Array.isArray(proof))return false;
  const observed=Date.parse(proof.observedAt);
  return proof.sourceKind==='TRUSTED_HOST_RESOURCE_FENCE_READBACK'&&
    proof.complete===true&&proof.fenceHeld===true&&
    proof.fenceScope==='EXACT_FROZEN_RESOURCE_SET'&&
    proof.heldThrough==='CAS_PUBLISH_READBACK'&&
    proof.repository===expected.repository&&
    proof.writerId===expected.writerId&&
    proof.protectedClaimKey===expected.protectedClaimKey&&
    proof.sourceSha===expected.sourceSha&&
    proof.foreignClaimsSha256===expected.foreignClaimsSha256&&
    proof.foreignClaimCount===expected.foreignClaimCount&&
    proof.incompatibleResourceCount===0&&
    HEX64.test(proof.frozenResourceSetSha256??'')&&
    ID.test(proof.fenceId??'')&&
    Number.isSafeInteger(proof.fenceRevision)&&proof.fenceRevision>=0&&
    Number.isFinite(observed)&&observed<=Date.now()&&
    Date.now()-observed<=RESOURCE_FENCE_AGE_MS;
}

/** Test/selected-Host seam. All authority comes from Host-owned configuration. */
export function createHostAdmissionSourceFromClient({
  octokit,store,owner,repo,writerAuthority,taskAuthority,resourceConflictPort
}={}){
  const repository=NAME.test(owner??'')&&NAME.test(repo??'')?`${owner}/${repo}`:'';
  if(!repository||!validWriter(writerAuthority,repository)||
    !validTask(taskAuthority,repository)||
    typeof octokit?.rest?.issues?.get!=='function'||
    typeof octokit?.rest?.issues?.listComments!=='function'||
    typeof octokit?.rest?.repos?.getCommit!=='function'||
    typeof octokit?.paginate?.iterator!=='function'||
    typeof store?.get!=='function'||typeof store?.list!=='function'||
    (resourceConflictPort!==undefined&&(
      resourceConflictPort?.sourceKind!=='TRUSTED_HOST_RESOURCE_FENCE_PORT'||
      resourceConflictPort?.repository!==repository||
      resourceConflictPort?.writerId!==writerAuthority.writerId||
      typeof resourceConflictPort?.readHeldFence!=='function')))
    throw new TypeError('TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED');
  const readback=createHostNativeGitHubClaimReadbackPortFromClient({
    octokit,owner,repo,writerLogin:writerAuthority.githubWriterLogin
  });
  async function registrySnapshot(){
    let rows;
    try{
      const read=async()=>{
        const keys=await store.list(PREFIX);
        if(!Array.isArray(keys)||keys.some(key=>typeof key!=='string'||!key.startsWith(PREFIX)))
          throw new Error('INCOMPLETE');
        const result=[];
        for(const key of keys)result.push({key,...await store.get(key)});
        return result;
      };
      const first=await read(),second=await read();
      if(first.length!==second.length||first.some((row,index)=>
        row.key!==second[index].key||row.revision!==second[index].revision))
        return {error:'HOST_RESERVATION_REGISTRY_DRIFT'};
      rows=first;
    }catch{return {error:'HOST_RESERVATION_REGISTRY_UNAVAILABLE'};}
    const seen=new Set();
    for(const row of rows){
      if(!validRegistryRow(row))return {error:'HOST_RESERVATION_REGISTRY_INCOMPLETE'};
      const key=row.value.protectedClaimKey;
      if(seen.has(key))return {error:'HOST_RESERVATION_REGISTRY_CONFLICT'};
      seen.add(key);
    }
    return {rows:rows.map(row=>row.value)};
  }
  async function readAdmissionFacts(request){
    if(!boundRequest(request,taskAuthority,repository))
      return denied('HOST_TASK_OR_WRITER_REQUEST_NOT_BOUND',request);
    const observed=await readNativeHostClaimPreflightFromClient({
      octokit,owner,repo,issueNumber:taskAuthority.issueNumber,
      expectedSourceSha:taskAuthority.sourceSha,limit:1000
    });
    if(observed.status!=='NATIVE_SOURCE_PREFLIGHT_READY')
      return denied(observed.reason??'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE',request);
    const registry=await registrySnapshot();
    if(registry.error)return denied(registry.error,request);
    // An existing active reservation for the SAME protected key can never be
    // excused by a disjoint-resource witness; the CAS controller must reconcile.
    const active=registry.rows.filter(row=>row.phase!=='TERMINAL_VERIFIED');
    const own=active.filter(row=>row.protectedClaimKey===request.protectedClaimKey);
    const foreign=active.filter(row=>row.protectedClaimKey!==request.protectedClaimKey);
    let activeIncompatibleClaimCount=own.length+foreign.length;
    if(own.length>0)return Object.freeze({
      ...denied('EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION',request),
      activeIncompatibleClaimCount
    });
    if(foreign.length>0&&resourceConflictPort){
      const foreignClaimsSha256=foreignClaimDigest(foreign);
      // The witness must bind the exact before/after Host registry snapshot,
      // not a stale list captured before another writer published a Claim.
      const registrySnapshotSha256=createHash('sha256')
        .update(JSON.stringify(registry.rows)).digest('hex');
      let proof=null;
      try{
        proof=await resourceConflictPort.readHeldFence(Object.freeze({
          repository,writerId:writerAuthority.writerId,
          protectedClaimKey:request.protectedClaimKey,sourceSha:request.sourceSha,
          foreignClaimCount:foreign.length,foreignClaimsSha256,
          // A copy: the port must obtain real resource/FSW and lease authority
          // from the trusted Host, NOT infer non-overlap from these projections.
          foreignClaims:Object.freeze(foreign.map(row=>Object.freeze({
            protectedClaimKey:row.protectedClaimKey,phase:row.phase,
            sourceSha:row.sourceSha,dispatchId:row.dispatchId,writerId:row.writerId
          })))
        }));
      }catch{proof=null;}
      if(validResourceFenceProof(proof,{repository,writerId:writerAuthority.writerId,
        protectedClaimKey:request.protectedClaimKey,sourceSha:request.sourceSha,
        foreignClaimCount:foreign.length,foreignClaimsSha256})){
        const after=await registrySnapshot();
        const afterSha=after.error?null:createHash('sha256')
          .update(JSON.stringify(after.rows)).digest('hex');
        if(afterSha!==registrySnapshotSha256)
          return denied('HOST_RESERVATION_REGISTRY_DRIFT',request);
        activeIncompatibleClaimCount=0;
      }
    }
    if(activeIncompatibleClaimCount>0)return Object.freeze({
      ...denied('INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS',request),
      activeIncompatibleClaimCount
    });
    return Object.freeze({
      sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',sourceQualified:true,
      complete:true,issueRef:request.issueRef,sourceSha:request.sourceSha,
      issueState:observed.issueState,workflowState:observed.workflowState,
      workItemType:observed.workItemType,dependencyState:observed.dependencyState,
      activeIncompatibleClaimCount,dispatchSourceBound:true,
      publisherSourceBound:true,claimStatus:'NOT_VERIFIED',
      authorized:false,admitted:false,taskPass:false,releasePass:false
    });
  }
  async function readPublishedClaim(request,commentId){
    if(!boundRequest(request,taskAuthority,repository))throw readbackFailure();
    return readback.readPublishedClaim(request,commentId);
  }
  return Object.freeze({readAdmissionFacts,readPublishedClaim});
}

/** Production Host wiring; token is never returned, compared or journaled. */
export function createGitHubHostAdmissionSource({token,...input}={}){
  if(typeof token!=='string'||token.length<8)
    throw new TypeError('TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return createHostAdmissionSourceFromClient({octokit,...input});
}

/** Selected Host only: the proof port must hold its writer fence and measure
 * every listed resource, including effects outside the local process tree.
 * It is not a request parameter or a public B/Agent endpoint. */
export function createHostTerminalizationSourceFromClient({octokit,admissionStore,writerAuthority,quiescence}={}){
  if(!validWriter(writerAuthority,TARGET.repository)||
    typeof admissionStore?.get!=='function'||typeof quiescence?.readFreshProof!=='function'||
    typeof octokit?.rest?.pulls?.get!=='function'||typeof octokit?.rest?.repos?.getCombinedStatusForRef!=='function')
    throw new TypeError('TRUSTED_TERMINAL_HOST_SOURCE_REQUIRED');
  const authority=Object.freeze({...writerAuthority});
  const claimReader=createHostNativeGitHubClaimReadbackPortFromClient({
    octokit,owner:'kaicreator-mm',repo:'tally',writerLogin:'kaicreator-mm'});
  const claim={...TARGET,actorRole:'builder',operatorKind:'other',operatorId:'zcode:candidate-publication:1'};
  const hash=value=>createHash('sha256').update(JSON.stringify(value)).digest('hex');
  // Durable public evidence reference: an exact https URL with no userinfo,
  // no query or fragment material and no characters the URL parser would
  // normalize away. Credential-bearing or secret-carrying URLs fail closed
  // and can never be published in the public #203 terminal evidence.
  const reference=x=>{
    if(typeof x!=='string'||x.length===0||x.length>2048)return false;
    let url;
    try{url=new URL(x);}catch{return false;}
    return url.protocol==='https:'&&url.href===x&&
      url.username===''&&url.password===''&&url.search===''&&url.hash==='';
  };
  async function readTerminalizationPreflight(row){
    const deny=()=>Object.freeze({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',complete:false});
    try{
      if(row?.revision!==1||row.value?.phase!=='PUBLISHED_VERIFIED')return deny();
      const [admission,frozen,original,terminal,pr204,pr205]=await Promise.all([
        admissionStore.get('tally:v2:claim-admission:471dc65c8c0ef32ac912306d962581d6db92f63af2e0bf88000d5a5370daa50d'),
        admissionStore.get('tally:v2:task-writeset:319'),
        claimReader.readPublishedClaim(claim,TARGET.claimCommentId),
        readDispatchStateEventsFromClient({octokit,writerLogin:authority.githubWriterLogin}),
        octokit.rest.pulls.get({owner:'kaicreator-mm',repo:'tally',pull_number:204}),
        octokit.rest.pulls.get({owner:'kaicreator-mm',repo:'tally',pull_number:205})
      ]);
      if(admission?.revision!==1||admission.value?.phase!=='PUBLISHED_VERIFIED'||
        admission.value?.commentId!==6668875091||
        admission.value?.protectedClaimKey!=='kaicreator-mm/tally#319:builder:__default__'||
        admission.value?.repository!==TARGET.repository||
        admission.value?.issueRef!=='https://github.com/kaicreator-mm/tally/issues/319'||
        admission.value?.sourceSha!=='7ebb1cdd0a79fc82cc938e35f852a80dd8e15332'||
        admission.value?.actorRole!=='builder'||admission.value?.operatorKind!=='other'||
        admission.value?.operatorId!=='zcode:i319:t06d-terminalization:1'||
        admission.value?.writerId!=='tally-host:win32:t06d-terminalization:1'||
        admission.value?.commentRef!=='https://github.com/kaicreator-mm/tally/issues/319#issuecomment-6668875091'||
        admission.value?.generation!==1||admission.value?.dispatchId!=='dispatch:319:t06d-terminalization:20261106:1'||
        frozen?.revision!==0||hash(frozen.value)!=='9f6a949a889d1a705116e2f4d8cc91f1577539ef8d1606b41bd65f9592cf0b5b'||
        original?.complete!==true||original.edited!==false||terminal.status!=='ABSENT'||
        [pr204,pr205].some((pr,i)=>pr?.data?.number!==204+i||pr.data.state!=='open'||pr.data.merged_at!==null||
          !SHA.test(pr.data.head?.sha??'')))return deny();
      const historicalPulls=await Promise.all([pr204,pr205].map(async pr=>{
        const state=(await octokit.rest.repos.getCombinedStatusForRef({owner:'kaicreator-mm',repo:'tally',ref:pr.data.head.sha}))?.data;
        if(state?.state!=='failure'||!Array.isArray(state.statuses)||state.statuses.length>100)
          throw Error('PR_CI_STATUS_UNKNOWN');
        const failures=state.statuses.filter(x=>x.state==='failure'&&
          CONTEXT.test(x.context??'')&&reference(x.target_url))
          .map(x=>({context:x.context,ref:x.target_url}));
        if(!failures.length)throw Error('PR_CI_FAILURE_NOT_PROVEN');
        return {number:pr.data.number,state:'open',merged:false,headSha:pr.data.head.sha,failures};
      }));
      const proof=await quiescence.readFreshProof(Object.freeze({...TARGET}));
      const timestamp=Date.parse(proof?.observedAt),now=Date.now();
      if(!Number.isFinite(timestamp)||timestamp>now||now-timestamp>30000||proof?.fenceHeld!==true||
        !ID.test(proof.fenceId??'')||proof.dispatchId!==TARGET.dispatchId||
        proof.sourceSha!==TARGET.sourceSha||proof.protectedClaimKey!==TARGET.protectedClaimKey||proof.generation!==1||
        proof.activeWriterCount!==0||proof.activeAttemptCount!==0||proof.activeSubprocessCount!==0||
        proof.inflightEffectCount!==0||proof.unresolvedResourceCount!==0||proof.writeSetReconciled!==true||
        !proof.evidenceRefs||Object.keys(proof.evidenceRefs).length!==REF_KEYS.length||
        !REF_KEYS.every(k=>reference(proof.evidenceRefs[k])))return deny();
      const occurredAt=new Date().toISOString();
      // Explicit allowlist: the arbitrary Host proof object is NEVER
      // serialized verbatim into the public #203 evidence. Only the exact
      // fields validated above — and no unknown proof field or extra
      // evidence reference — may appear in the published event.
      const evidence=JSON.stringify({
        taskPackRef:frozen.value.taskPackRef,taskPackBlobSha:frozen.value.taskPackBlobSha,
        claim319:{key:'kaicreator-mm/tally#319:builder:__default__',generation:1,
          dispatchId:admission.value.dispatchId,commentId:admission.value.commentId,commentRef:admission.value.commentRef},
        frozenWriteSet:{key:'tally:v2:task-writeset:319',revision:frozen.revision,digest:hash(frozen.value)},
        targetClaim:{casKey:TARGET.casKey,revision:row.revision,commentId:TARGET.claimCommentId,commentRef:TARGET.claimCommentRef},
        quiescence:{observedAt:proof.observedAt,fenceHeld:proof.fenceHeld,fenceId:proof.fenceId,
          dispatchId:proof.dispatchId,sourceSha:proof.sourceSha,
          protectedClaimKey:proof.protectedClaimKey,generation:proof.generation,
          activeWriterCount:proof.activeWriterCount,activeAttemptCount:proof.activeAttemptCount,
          activeSubprocessCount:proof.activeSubprocessCount,inflightEffectCount:proof.inflightEffectCount,
          unresolvedResourceCount:proof.unresolvedResourceCount,writeSetReconciled:proof.writeSetReconciled,
          evidenceRefs:proof.evidenceRefs},
        openHistoricalPulls:historicalPulls
      });
      return Object.freeze({sourceKind:'HOST_TERMINALIZATION_PREFLIGHT',complete:true,targetRevision:row.revision,
        fenceId:proof.fenceId,evidence,occurredAt});
    }catch{return deny();}
  }
  async function readTerminalEvent(reservation){
    try{
      const [terminal,original]=await Promise.all([
        readDispatchStateEventsFromClient({octokit,writerLogin:authority.githubWriterLogin,
          expectedEvent:terminalEventForReservation(reservation,authority)}),
        claimReader.readPublishedClaim(claim,TARGET.claimCommentId)
      ]);
      return original?.complete===true&&original.edited===false?terminal:{status:'UNKNOWN'};
    }
    catch{return {status:'UNKNOWN'};}
  }
  return Object.freeze({readTerminalizationPreflight,readTerminalEvent});
}
