import test from 'node:test';
import assert from 'node:assert/strict';
import * as claimTarget from './claim-target.mjs';
const {resolveClaimBuildTarget,markClaimProviderTarget,assertClaimProviderTarget}=claimTarget;

// Builder-owned tests for the marked manifest frame. pilot-contract.test.mjs
// is the public acceptance probe; these add the exact frame-limit edges, a
// writer/reader agreement sweep and a bounded-inspection check it does not reach.
const code='export const accepted=false;';
const PREFIX='// TALLY_T06L_BUILD_TARGET_V1=';
const LIMIT=4096;
const INVALID=/^TypeError: T06L_PROVIDER_TARGET_MANIFEST_INVALID$/;
const MISMATCH=/^TypeError: T06L_PROVIDER_TARGET_MISMATCH$/;
const MISSING=/^TypeError: T06L_PROVIDER_TARGET_MANIFEST_MISSING$/;
const WRITE_INVALID=/^TypeError: T06L_PROVIDER_TARGET_INVALID$/;
const selected=(repository='org/repo',issue='123')=>resolveClaimBuildTarget({
  T06L_TARGET_REPOSITORY:repository,T06L_TARGET_ISSUE_NUMBER:issue});
const legacy=resolveClaimBuildTarget({});
const encode=(value)=>Buffer.from(JSON.stringify(value),'utf8').toString('base64');
const manifest=(t)=>({repository:t.repository,issueNumber:t.issueNumber,
  protectedClaimKey:t.protectedClaimKey});
// Frame size in bytes including the LF, derived from the published format
// rather than from the implementation under test.
const frameTotal=(t)=>PREFIX.length+encode(manifest(t)).length+1;
// Frame sizes are 31 + 4k, so 4095 and 4099 are the sizes adjacent to the
// 4096-byte limit; nothing the writer can produce falls between them.
const targetWithFrameTotal=(total)=>{
  for(let digits=1;digits<=10;digits++)for(let owner=1;owner<2100;owner++){
    const t=selected('a'.repeat(owner)+'/b','9'.repeat(digits));
    if(frameTotal(t)===total)return t;
  }
  throw new Error('no admitted target has a frame of '+total+' bytes');
};

test('writer emits the largest admitted frame and refuses the first one past the limit',()=>{
  const inside=targetWithFrameTotal(4095),outside=targetWithFrameTotal(4099);
  const marked=markClaimProviderTarget(code,inside);
  assert.equal(marked.indexOf('\n')+1,4095);
  assert.equal(marked.slice(4095),code);
  assert.throws(()=>markClaimProviderTarget(code,outside),WRITE_INVALID);
});

test('reader accepts a valid 4095-byte frame and calls a valid 4099-byte frame invalid',()=>{
  const inside=targetWithFrameTotal(4095),outside=targetWithFrameTotal(4099);
  const admitted=markClaimProviderTarget(code,inside);
  assert.doesNotThrow(()=>assertClaimProviderTarget(Buffer.from(admitted,'latin1'),inside));
  assert.throws(()=>assertClaimProviderTarget(Buffer.from(admitted,'latin1'),outside),MISMATCH);
  // The writer refuses this target, so build its well-formed frame from the format.
  const oversized=Buffer.from(PREFIX+encode(manifest(outside))+'\n'+code,'latin1');
  assert.equal(oversized.indexOf(10)+1,4099);
  // Past the limit is a frame fault, never a match or a mismatch.
  for(const t of [outside,inside,legacy])
    assert.throws(()=>assertClaimProviderTarget(oversized,t),INVALID);
});

test('writer and reader agree on every admitted target around the limit',()=>{
  let largest=0,smallestRefused=Infinity;
  for(let digits=1;digits<=10;digits++)for(let owner=1470;owner<=1500;owner++){
    const t=selected('a'.repeat(owner)+'/b','9'.repeat(digits));
    let marked;
    try{marked=markClaimProviderTarget(code,t);}
    catch(error){
      assert.equal(error.message,'T06L_PROVIDER_TARGET_INVALID');
      smallestRefused=Math.min(smallestRefused,frameTotal(t));
      continue;
    }
    const total=marked.indexOf('\n')+1;
    assert.equal(total,frameTotal(t));
    assert.doesNotThrow(()=>assertClaimProviderTarget(Buffer.from(marked,'latin1'),t));
    largest=Math.max(largest,total);
  }
  assert.equal(largest,4095);
  assert.equal(smallestRefused,4099);
});

test('a marked header with no LF inside the frame is invalid, never truncated or accepted',()=>{
  const t=selected();
  const marked=Buffer.from(markClaimProviderTarget(code,t)),end=marked.indexOf(10);
  for(const bytes of [
    marked.subarray(0,end),
    // Dropping a trailing character must not turn an unterminated frame into a valid one.
    Buffer.concat([marked.subarray(0,end),Buffer.from('A')]),
    Buffer.concat([marked.subarray(0,end),Buffer.from(code)]),
    Buffer.concat([marked.subarray(0,end),Buffer.alloc(LIMIT,0x41),Buffer.from('\n'+code)]),
    Buffer.from(PREFIX+'A'.repeat(LIMIT)+'\n'+code)])
    assert.throws(()=>assertClaimProviderTarget(bytes,t),INVALID);
});

test('bytes past the old 512-byte read are validated, not skipped',()=>{
  const t=selected('a'.repeat(139)+'/'+'b'.repeat(139),'1234567890');
  const marked=Buffer.from(markClaimProviderTarget(code,t)),end=marked.indexOf(10);
  // Past the old read, still inside the header.
  assert.ok(end>600);
  assert.doesNotThrow(()=>assertClaimProviderTarget(marked,t));
  for(const poison of [[0xc3,0xa9],[0xff],[0x0d],[0x20]]){
    const bytes=Buffer.concat([marked.subarray(0,600),Buffer.from(poison),
      marked.subarray(600)]);
    assert.throws(()=>assertClaimProviderTarget(bytes,t),INVALID);
  }
  // Non-canonical base64: extra padding, a dropped character, a stray character.
  const header=encode(manifest(t));
  for(const bad of [header+'=',header.slice(0,-1),header+'A'])
    assert.throws(()=>assertClaimProviderTarget(
      Buffer.from(PREFIX+bad+'\n'+code,'latin1'),t),INVALID);
  assert.doesNotThrow(()=>assertClaimProviderTarget(
    Buffer.concat([marked,Buffer.alloc(20000,0xff)]),t));
});

test('metadata that parses but is not this target stays a mismatch',()=>{
  const t=selected('a'.repeat(39)+'/'+'b'.repeat(100),'1234567890');
  const other=selected('a'.repeat(39)+'/'+'b'.repeat(100),'1234567891');
  const marked=Buffer.from(markClaimProviderTarget(code,t));
  assert.throws(()=>assertClaimProviderTarget(marked,other),MISMATCH);
  assert.throws(()=>assertClaimProviderTarget(marked,legacy),MISMATCH);
  for(const value of [null,[],{...manifest(t),issueNumber:t.issueNumber+1},
    {issueNumber:t.issueNumber,repository:t.repository,
      protectedClaimKey:t.protectedClaimKey}]){
    const bytes=Buffer.from(PREFIX+encode(value)+'\n'+code,'latin1');
    assert.throws(()=>assertClaimProviderTarget(bytes,t),MISMATCH);
  }
});

test('markerless input is classified by its first bytes only, as before',()=>{
  const t=selected();
  for(const bytes of [Buffer.alloc(0),Buffer.from('historical provider'),
    Buffer.from(code),
    Buffer.concat([Buffer.alloc(5000,0x41),Buffer.from('\n'+code)]),
    Buffer.from('x'+markClaimProviderTarget(code,t)),
    Buffer.from(PREFIX.slice(0,-1)+'\n'+code),
    Buffer.concat([Buffer.from([0xef,0xbb,0xbf]),
      Buffer.from(markClaimProviderTarget(code,t))])]){
    assert.doesNotThrow(()=>assertClaimProviderTarget(bytes,legacy));
    assert.throws(()=>assertClaimProviderTarget(bytes,t),MISSING);
  }
});

test('reader asks the provider buffer for no more than the first 4096 bytes',()=>{
  const t=selected();
  const marked=Buffer.from(markClaimProviderTarget(code,t));
  const provider=Buffer.concat([marked,Buffer.alloc(20000,0xff)]);
  const ranges=[],unboundedSearches=[],elementReads=[];
  // Log every access the reader makes on the provider buffer.
  const watched=new Proxy(provider,{get(buffer,property){
    const member=Reflect.get(buffer,property,buffer);
    if(typeof member!=='function'){
      if(/^\d+$/.test(String(property)))elementReads.push(Number(property));
      return member;
    }
    return (...args)=>{
      if(property==='toString')ranges.push([args[1],args[2]]);
      else if(property==='subarray'||property==='slice')ranges.push([args[0],args[1]]);
      else if(['indexOf','lastIndexOf','includes'].includes(property))unboundedSearches.push(property);
      return member.apply(buffer,args);
    };
  }});
  assert.ok(Buffer.isBuffer(watched));
  assert.doesNotThrow(()=>assertClaimProviderTarget(watched,t));
  // A search on the whole buffer has no end bound; only bounded reads are allowed.
  assert.deepEqual(unboundedSearches,[]);
  assert.ok(ranges.length>0);
  for(const [start,end] of ranges){
    assert.equal(start,0);
    assert.ok(end<=LIMIT,'read end '+end);
  }
  for(const index of elementReads)assert.ok(index<LIMIT,'byte read at '+index);
});

test('writer output for an ordinary target is byte-identical to the frozen baseline',()=>{
  // Captured from claim-target.mjs blob ee60d8b2 at 3dbbb7d6 before this change.
  assert.equal(markClaimProviderTarget(code,selected('kaicreator-mm/tally-reps-test')),
    PREFIX+'eyJyZXBvc2l0b3J5Ijoia2FpY3JlYXRvci1tbS90YWxseS1yZXBzLXRlc3QiLCJpc3N1ZU51bWJlciI6MTIzLCJwcm90ZWN0ZWRDbGFpbUtleSI6ImthaWNyZWF0b3ItbW0vdGFsbHktcmVwcy10ZXN0IzEyMzpidWlsZGVyOl9fZGVmYXVsdF9fIn0=\n'+code);
});

test('the public surface is unchanged',()=>{
  assert.deepEqual(Object.keys(claimTarget).sort(),['assertClaimProviderTarget',
    'markClaimProviderTarget','resolveClaimBuildTarget','sealClaimShimTarget']);
});
