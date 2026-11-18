/** Invalidate old-version requests before their success or failure may update UI. */
export interface RequestLease {readonly generation:number; isCurrent():boolean;}
export interface LatestRequestGate {begin():RequestLease;invalidate():void;readonly generation:number;}
export function createLatestRequestGate():LatestRequestGate {
  let generation=0;
  return {
    get generation(){return generation;},
    begin(){const current=++generation;
      return {generation:current,isCurrent:()=>current===generation};},
    invalidate(){++generation;},
  };
}
