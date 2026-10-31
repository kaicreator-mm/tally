/** J07c physical evidence *capture*, not independent acceptance / Release RQ.
 * The selected Windows Host must run original checks, protect Hidden verifier
 * identity, then produce an independently signed/reviewed RQ later.
 */
const SHA=/^[a-f0-9]{40}$/;
export const INSTALLED_GATES=Object.freeze([
  'cleanInstall','selectedKernelHost','githubReadback','realBrowserClick',
  'fullRegression','hiddenValidation','criticalJourneys','userAcceptance'
]);
const deny=reason=>Object.freeze({status:'NOT_RUN',reason,
  physicalProof:false,independentVerification:false,releasePass:false,
  qualification:'NOT_QUALIFIED',gates:Object.freeze([])});
const record=(gate,result,sourceSha)=>{
  const idValid=result?.sourceSha===sourceSha&&typeof result.evidencePath==='string'&&
    result.evidencePath.length>0&&typeof result.observedAt==='string'&&
    Number.isFinite(Date.parse(result.observedAt));
  return Object.freeze({gate,state:idValid&&result.state==='OBSERVED'?
    'REPORTED_OBSERVED_UNVERIFIED':'UNKNOWN',
    reason:idValid?'REQUIRES_INDEPENDENT_PHYSICAL_VERIFICATION':'RECEIPT_INCOMPLETE',
    evidencePath:idValid?result.evidencePath:null});
};
/** Executes requested Host probes sequentially; read-only output of the
 * function is always unqualified. Do not run Hidden checks with Builder's
 * principal; a missing separate verifier is a mandatory hard stop. */
export async function captureInstalledJourney({sourceSha,platform=process.platform,
  nodeMajor=Number(process.versions.node.split('.')[0]),verifyVerifierPrincipal,
  runProbe}={}){
  if(!SHA.test(sourceSha??''))return deny('EXACT_SOURCE_REQUIRED');
  if(platform!=='win32'||nodeMajor<24)return deny('WINDOWS_NODE24_HOST_REQUIRED');
  if(typeof verifyVerifierPrincipal!=='function'||typeof runProbe!=='function')
    return deny('INDEPENDENT_RUNNERS_NOT_WIRED');
  let verifier;
  try{verifier=await verifyVerifierPrincipal();}catch{return deny('VERIFIER_PRINCIPAL_UNKNOWN');}
  if(verifier?.state!=='VERIFIED_DISTINCT'||
    typeof verifier?.builderSid!=='string'||typeof verifier?.verifierSid!=='string'||
    verifier.builderSid===verifier.verifierSid)
    return deny('HIDDEN_VERIFIER_ISOLATION_NOT_PROVEN');
  const gates=[];
  for(const gate of INSTALLED_GATES){
    let result;
    try{result=await runProbe({gate,sourceSha});}catch{}
    gates.push(record(gate,result,sourceSha));
    if(gates.at(-1).state==='UNKNOWN')break;
  }
  return Object.freeze({status:gates.length===INSTALLED_GATES.length&&
      gates.every(x=>x.state==='REPORTED_OBSERVED_UNVERIFIED')?
      'CAPTURE_REQUIRES_INDEPENDENT_VERIFICATION':'INCOMPLETE',
    sourceSha,reason:'PHYSICAL_EVIDENCE_NOT_INDEPENDENTLY_ATTESTED',
    gates:Object.freeze(gates),physicalProof:false,
    independentVerification:false,releasePass:false,qualification:'NOT_QUALIFIED'});
}
