/**
 * Tally v2 app-side neutral dh.llm.generate/1 request/response adapter.
 * CONTRACT-FIRST against reviewed DomainHarness v0.10.3 L2; no v0.10.3
 * SDK import, no auto-router, provider HTTP, token or Kernel execution.
 */
const CONTRACT='dh.llm.generate/1';
const PROVIDERS=Object.freeze({
  'openai-compatible':Object.freeze({
    intentId:'llmViaOpenAI',capabilityId:'llm.openai',
    operationId:'llm.openai.generate',handlerId:'generate'
  }),
  'anthropic-messages':Object.freeze({
    intentId:'llmViaAnthropic',capabilityId:'llm.anthropic',
    operationId:'llm.anthropic.generate',handlerId:'generate'
  })
});
const FINISH=new Set(['stop','length','tool_proposal','content_filtered','other']);
const ERRORS=new Set(['UNAVAILABLE','UNSUPPORTED_FEATURE','RATE_LIMITED',
  'AUTH_DENIED','MALFORMED_RESPONSE','CONTEXT_OVERFLOW','CANCELLED',
  'DEADLINE','OUTCOME_UNKNOWN']);
const allowedText=(x,max)=>typeof x==='string'&&x.length>0&&x.length<=max;
const safeCount=(v,max)=>Number.isSafeInteger(v)&&v>=0&&v<=max;
const safeOutput=(reason,extra={})=>Object.freeze({
  status:'BLOCKED',reason,authorized:false,admitted:false,
  taskPass:false,releasePass:false,...extra
});
const invalid=(req)=>!req||typeof req!=='object'||Array.isArray(req)||
  req.contractVersion!==CONTRACT||!allowedText(req.requestId,128)||
  !allowedText(req.modelId,128)||!Number.isSafeInteger(req.deadlineUnixMs)||
  !safeCount(req.maxOutputTokens,8192)||req.maxOutputTokens===0||
  !['text','json'].includes(req.outputFormat)||
  !Array.isArray(req.messages)||req.messages.length===0||req.messages.length>32||
  req.messages.some(m=>!m||!['system','user','assistant'].includes(m.role)||
    !allowedText(m.content,32768))||
  req.messages.reduce((n,m)=>n+(m?.content?.length||0),0)>65536||
  (req.temperature!==undefined&&(!Number.isFinite(req.temperature)||
    req.temperature<0||req.temperature>1));
/**
 * Host/BUILD trusted binding is supplied by integration owner, never by
 * user text or LLM. This validates the intent only; K Permit remains required.
 */
export function planNeutralGeneration(req,{
  selectedProviders=[],chosenProvider=null,approvedModels={},
  nowUnixMs,maxWindowMs=3600000
}={}){
  try{
    if(invalid(req))return safeOutput('INVALID_NEUTRAL_LLM_REQUEST');
    if(!Number.isSafeInteger(nowUnixMs)||
      !safeCount(maxWindowMs,3600000)||maxWindowMs===0)
      return safeOutput('HOST_TIME_POLICY_UNVERIFIED');
    if(req.deadlineUnixMs<=nowUnixMs||
      req.deadlineUnixMs>nowUnixMs+maxWindowMs)
      return safeOutput('HOST_DEADLINE_REFUSED');
    if(req.outputSchemaRef!==undefined||req.toolCatalogRef!==undefined)
      return safeOutput('OPTIONAL_FEATURE_NOT_HOST_APPROVED');
    if(!Array.isArray(selectedProviders)||
      new Set(selectedProviders).size!==selectedProviders.length||
      selectedProviders.some(p=>!Object.hasOwn(PROVIDERS,p)))
      return safeOutput('SELECTED_PROVIDER_SET_UNVERIFIED');
    if(selectedProviders.length===0)
      return safeOutput('NO_SELECTED_MODEL_CAPABILITY');
    const provider=chosenProvider===null&&selectedProviders.length===1?
      selectedProviders[0]:chosenProvider;
    if(!Object.hasOwn(PROVIDERS,provider)||
      !selectedProviders.includes(provider))
      return safeOutput(selectedProviders.length>1&&chosenProvider===null?
        'EXPLICIT_MODEL_ROUTE_REQUIRED':'UNSELECTED_MODEL_CAPABILITY');
    const models=approvedModels[provider];
    if(!Array.isArray(models)||!models.includes(req.modelId)||
      models.some(x=>!allowedText(x,128)))
      return safeOutput('HOST_MODEL_ALLOWLIST_REFUSED');
    const copy={
      contractVersion:CONTRACT,requestId:req.requestId,modelId:req.modelId,
      deadlineUnixMs:req.deadlineUnixMs,messages:req.messages.map(m=>({
        role:m.role,content:m.content})),maxOutputTokens:req.maxOutputTokens,
      outputFormat:req.outputFormat
    };
    if(req.temperature!==undefined)copy.temperature=req.temperature;
    return safeOutput('EXPLICIT_SELECTED_MODEL_INTENT',{
      status:'PLANNED',providerKind:provider,
      route:Object.freeze({...PROVIDERS[provider]}),
      request:Object.freeze(copy),
      requiresKernelPermit:true,
      providerInvocationPerformed:false
    });
  }catch{return safeOutput('UNTRUSTED_MODEL_PLAN_INPUT');}
}
const validUsage=u=>u&&typeof u==='object'&&
  (u.evidenceMode==='unavailable'?
    !Object.hasOwn(u,'inputTokens')&&!Object.hasOwn(u,'outputTokens'):
    ['provider-reported','measured'].includes(u.evidenceMode)&&
    safeCount(u.inputTokens,Number.MAX_SAFE_INTEGER)&&
    safeCount(u.outputTokens,Number.MAX_SAFE_INTEGER));
const validError=r=>ERRORS.has(r.errorCategory)&&
  allowedText(r.reasonCode,128)&&/^[A-Z0-9_:-]+$/.test(r.reasonCode);
const validProposal=p=>p&&allowedText(p.name,128)&&
  typeof p.argumentsJson==='string'&&p.argumentsJson.length<=16384;
/** Normalize only externally observed, independently chosen route output. */
export function projectNeutralGenerationResult(plan,result){
  try{
    if(plan?.status!=='PLANNED'||plan?.reason!=='EXPLICIT_SELECTED_MODEL_INTENT'||
      plan?.admitted!==false||plan?.authorized!==false||
      !Object.hasOwn(PROVIDERS,plan.providerKind)||
      !result||typeof result!=='object'||
      result.contractVersion!==CONTRACT||
      result.requestId!==plan.request.requestId||
      result.modelId!==plan.request.modelId||
      result.providerKind!==plan.providerKind||
      !validUsage(result.usage))
      return safeOutput('PROVIDER_PROVENANCE_OR_USAGE_UNVERIFIED');
    const usage=Object.freeze({...result.usage});
    if(result.status==='completed'){
      if(typeof result.text!=='string'||result.text.length>65536||
        !FINISH.has(result.finishReason)||
        (result.toolProposals!==undefined&&
          (!Array.isArray(result.toolProposals)||result.toolProposals.length>8||
            result.toolProposals.some(p=>!validProposal(p)))))
        return safeOutput('MALFORMED_NEUTRAL_COMPLETION');
      // Proposals are not Tools. The caller must obtain a distinct K Permit.
      return safeOutput('OBSERVED_MODEL_COMPLETION',{
        status:'OBSERVED',providerKind:plan.providerKind,
        modelId:plan.request.modelId,text:result.text,
        finishReason:result.finishReason,usage,
        toolProposalCount:result.toolProposals?.length??0,
        toolExecutionAuthorized:false
      });
    }
    if(!['refused','failed','unknown'].includes(result.status)||!validError(result))
      return safeOutput('MALFORMED_NEUTRAL_FAILURE');
    return safeOutput(result.status==='unknown'?'REMOTE_OUTCOME_UNKNOWN':
      'OBSERVED_MODEL_NONCOMPLETION',{
      status:result.status.toUpperCase(),
      providerKind:plan.providerKind,modelId:plan.request.modelId,
      errorCategory:result.errorCategory,reasonCode:result.reasonCode,
      usage,toolExecutionAuthorized:false
    });
  }catch{return safeOutput('HOST_PROVIDER_RESULT_UNTRUSTED');}
}
export const frozenContractVersion=CONTRACT;
