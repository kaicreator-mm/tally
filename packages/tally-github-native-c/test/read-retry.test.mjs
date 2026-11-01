import test from 'node:test';
import assert from 'node:assert/strict';
import {createNativeReadRetryFetch} from '../src/read-retry.mjs';
const response=status=>new Response(JSON.stringify({status}),{status,headers:{'content-type':'application/json'}});
const config=(answers,extra={})=>{
  let calls=0;const delays=[];
  const fetch=createNativeReadRetryFetch({fetchImpl:async()=>{const answer=answers[Math.min(calls++,answers.length-1)];
    if(answer instanceof Error)throw answer;return answer;},sleep:async ms=>{delays.push(ms);},baseDelayMs:1,maxDelayMs:3,...extra});
  return {fetch,calls:()=>calls,delays};
};
test('500 and 429 reads retry with bounded delay then return the exact response',async()=>{
  const f=config([response(500),response(429),response(200)]);
  const result=await f.fetch('https://api.github.com/read');assert.equal(result.status,200);
  assert.equal(f.calls(),3);assert.deepEqual(f.delays,[1,2]);
});
test('403 is returned immediately and non-idempotent methods are never retried',async()=>{
  const denied=config([response(403),response(200)]);assert.equal((await denied.fetch('https://x')).status,403);
  assert.equal(denied.calls(),1);
  const write=config([response(500),response(200)]);assert.equal((await write.fetch('https://x',{method:'POST'})).status,500);
  assert.equal(write.calls(),1);
});
test('transport exhaustion and HTTP exhaustion remain explicit failures',async()=>{
  const network=config([new Error('offline')]);await assert.rejects(network.fetch('https://x'),/offline/);
  assert.equal(network.calls(),3);
  const http=config([response(500)],{maxAttempts:2});
  assert.equal((await http.fetch('https://x')).status,500);assert.equal(http.calls(),2);
});
test('retry count and sleep are bounded and no request payload is logged',async()=>{
  assert.throws(()=>createNativeReadRetryFetch({maxAttempts:5}),/CONFIG_INVALID/);
  const f=config([response(500)],{maxAttempts:4});await f.fetch('https://x');
  assert.equal(f.calls(),4);assert.ok(f.delays.every(ms=>ms>=0&&ms<=3));
});
test('concurrent native reads are serialized across Octokit client instances',async()=>{
  let active=0,maxActive=0,calls=0;
  const make=()=>createNativeReadRetryFetch({fetchImpl:async()=>{active++;maxActive=Math.max(maxActive,active);calls++;
    await new Promise(resolve=>setTimeout(resolve,2));active--;return response(200);},sleep:async()=>{}});
  const clients=[make(),make(),make()];
  await Promise.all(Array.from({length:12},(_,i)=>clients[i%3]('https://x/'+i)));
  assert.equal(calls,12);assert.equal(maxActive,1);
});
