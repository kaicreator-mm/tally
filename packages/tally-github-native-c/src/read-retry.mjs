/** Bounded retry for idempotent native GitHub reads only. Never use for writes. */
let nativeReadChain=Promise.resolve();
export function createNativeReadRetryFetch({fetchImpl=globalThis.fetch,maxAttempts=3,
  baseDelayMs=50,maxDelayMs=250,sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms))}={}){
  if(typeof fetchImpl!=='function'||!Number.isSafeInteger(maxAttempts)||maxAttempts<1||maxAttempts>4)
    throw new TypeError('NATIVE_READ_RETRY_CONFIG_INVALID');
  async function perform(input,options){
    let lastError;
    for(let attempt=1;attempt<=maxAttempts;attempt++){
      try{
        const response=await fetchImpl(input,options);
        const retryable=response.status===429||response.status>=500;
        if(!retryable||attempt===maxAttempts)return response;
        void response.body?.cancel().catch(()=>{});
        const retryAfterHeader=response.headers?.get?.('retry-after');
        const retryAfter=retryAfterHeader===null||retryAfterHeader===undefined||retryAfterHeader===''?NaN:Number(retryAfterHeader);
        const delay=Number.isFinite(retryAfter)&&retryAfter>=0?
          Math.min(maxDelayMs,retryAfter*1000):Math.min(maxDelayMs,baseDelayMs*attempt);
        await sleep(delay);
      }catch(error){
        lastError=error;
        if(attempt===maxAttempts)throw error;
        await sleep(Math.min(maxDelayMs,baseDelayMs*attempt));
      }
    }
    throw lastError;
  }
  return async function nativeReadRetryFetch(input,options={}){
    const method=String(options?.method??'GET').toUpperCase();
    if(method!=='GET'&&method!=='HEAD')return fetchImpl(input,options);
    const result=nativeReadChain.then(()=>perform(input,options),()=>perform(input,options));
    nativeReadChain=result.catch(()=>{});
    return result;
  };
}
