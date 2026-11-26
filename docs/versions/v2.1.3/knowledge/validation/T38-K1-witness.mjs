import assert from 'node:assert/strict';

// A: common calculation, different caller contracts.
const sum = xs => xs.reduce((a,b) => a+b,0);
const preview = x => sum(x.lines);
const publish = x => {
  if (!x.signed || !x.allowed) throw new Error('DENIED');
  return sum(x.lines);
};
const wrongMerge = x => sum(x.lines);
const unsigned = {lines:[120,80],signed:false,allowed:true};
assert.equal(preview(unsigned),200);
assert.throws(() => publish(unsigned),/DENIED/);
assert.throws(() => publish({...unsigned,signed:true,allowed:false}),/DENIED/);
assert.equal(publish({...unsigned,signed:true}),200);
assert.equal(wrongMerge(unsigned),200); // counterexample to the publish contract
console.log('A preview=200; unsigned/unauthorized publish denied; authorized=200; wrongMerge=200');

// B: input alias versus independently specified snapshot / live view.
const x = [{lo:1,hi:2}];
const shallow = [...x];
const snapshot = x.map(v => ({...v}));
const live = x;
x[0].hi = 9;
assert.equal(shallow[0].hi,9);
assert.equal(snapshot[0].hi,2);
assert.equal(live[0].hi,9);
console.log('B shallow=9; snapshot=2; live=9 (legal for live contract only)');

// C: same abstract value, different representation invariant.
const expand = rs => [...new Set(rs.flatMap(([l,h]) => Array.from({length:h-l+1},(_,i)=>l+i)))].sort((a,b)=>a-b);
const ri = rs => rs.every(([l,h],i) => l<=h && (i===0 || l>rs[i-1][1]+1));
const canonical = [[1,3],[7,8]], split = [[1,2],[3,3],[7,8]];
assert.deepEqual(expand(canonical),expand(split));
assert.equal(ri(canonical),true);
assert.equal(ri(split),false);
assert.notDeepEqual(canonical,split); // if original intervals matter, set equality is insufficient
console.log('C AF equal; RI canonical=true split=false; original intervals differ');

// D: deterministic interleaving; one available slot in one event loop.
async function attempt(guard) {
  let used=0;
  const take = async () => {
    if (used>=1) return false;
    await Promise.resolve();
    if (guard && used>=1) return false;
    used++;
    return true;
  };
  return {results:await Promise.all([take(),take()]),used};
}
assert.deepEqual(await attempt(false),{results:[true,true],used:2});
assert.deepEqual(await attempt(true),{results:[true,false],used:1});
console.log('D check-before-await: 2 accepted; recheck+increment without await: 1 accepted');
console.log('4 fixture groups observed as expected; not product or knowledge-wide PASS');
