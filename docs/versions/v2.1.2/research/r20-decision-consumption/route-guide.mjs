import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

// Routing metadata only: not another selector, knowledge excerpt, or authority.
const GUIDE = [
  ['boundary', 'Independent variation, invariant ownership or dependency boundary is undecided.', 'No unresolved boundary or variation decision; mere file count.'],
  ['alias', 'Mutable object references cross a boundary; observable ownership or snapshot behavior is at issue.', 'Names, IDs or string aliases without shared mutable references.'],
  ['abstraction', 'A real choice of inline/helper/module/interface or semantic reuse remains.', 'No unresolved abstraction choice; code length alone.'],
  ['precision', 'Required exact numeric representation or intermediate arithmetic may lose precision.', 'Small exact integer counts with no plausible rounding; byte versus character indexing alone.'],
  ['oracle', 'A QA judgment needs authorized expected behavior, input domain or a discriminating falsifier.', 'No unresolved QA/Oracle question; routine invocation of an already qualified test.'],
  ['maintenance', 'An authorized subsequent change or compatibility decision needs evidence of change locality or old/new behavior.', 'Merely reading legacy code or implementing an unchanged contract with no unresolved maintenance decision.'],
  ['effects', 'Persistent state, concurrent writers, external effects or uncertain acknowledgement/commit are at issue.', 'Pure in-memory computation, buffer reads or byte framing without these effects.'],
  ['cost', 'A concrete tradeoff between quality, implementation, validation or maintenance costs needs a decision.', 'No cost tradeoff to decide; reporting tokens alone.']
];
const sha = value => createHash('sha256').update(value).digest('hex');
const nonempty = value => typeof value === 'string' && value.trim().length > 0;
const require = (condition, message) => { if (!condition) throw new Error(message); };

export function describeRisks() {
  return GUIDE.map(([id, when, notWhen]) => ({ id, when, notWhen }));
}

// Records a caller's per-piece account. Receipt hashing is integrity checking,
// not authentication, proof of chronology, or proof of Agent reading.
export function recordPieceApplications(receipt, report) {
  require(receipt?.status === 'DELIVERED' && Array.isArray(receipt.selected) &&
    Array.isArray(receipt.delivered) && receipt.selected.length > 0 &&
    receipt.selected.length === receipt.delivered.length, 'INVALID_DELIVERY');
  const { receiptSha256, ...payload } = receipt;
  require(sha(JSON.stringify(payload)) === receiptSha256 && report?.receiptSha256 === receiptSha256, 'DELIVERY_MISMATCH');
  require(Array.isArray(report.pieces) && report.pieces.length === receipt.delivered.length, 'INCOMPLETE_PIECE_REPORT');
  const selected = new Map(receipt.selected.map(p => [p.id, p]));
  const known = new Set(GUIDE.map(([id]) => id));
  const delivered = new Map();
  for (const p of receipt.delivered) {
    const { text, ...metadata } = p;
    require(known.has(p.id) && !delivered.has(p.id) && selected.has(p.id) &&
      nonempty(text) && sha(text) === p.sha256 && Buffer.byteLength(text, 'utf8') === p.bytes &&
      JSON.stringify(metadata) === JSON.stringify(selected.get(p.id)), 'INVALID_DELIVERED_PIECE');
    delivered.set(p.id, p);
  }
  const seen = new Set();
  const pieces = report.pieces.map(p => {
    require(delivered.has(p?.id) && !seen.has(p.id) &&
      ['YES', 'NO', 'UNKNOWN'].includes(p.read) &&
      ['APPLIED', 'ALREADY_ALIGNED', 'REJECTED', 'UNKNOWN'].includes(p.outcome) &&
      nonempty(p.reason), 'INVALID_PIECE_REPORT');
    seen.add(p.id);
    if (['APPLIED', 'ALREADY_ALIGNED'].includes(p.outcome)) {
      require(p.read === 'YES' && nonempty(p.before) && nonempty(p.after), 'MISSING_DECISION_EVIDENCE');
      require((p.before === p.after) === (p.outcome === 'ALREADY_ALIGNED'), 'DECISION_OUTCOME_MISMATCH');
    }
    return { id: p.id, excerptSha256: delivered.get(p.id).sha256, read: p.read,
      outcome: p.outcome, reason: p.reason, before: p.before ?? null, after: p.after ?? null };
  });
  return { decisionId: receipt.decisionId, receiptSha256, pieces,
    qualification: 'SELF_REPORTED_NOT_VERIFIED', qualityBenefit: 'NOT_ESTABLISHED' };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [command, receiptPath, reportPath] = process.argv.slice(2);
  const read = path => JSON.parse(readFileSync(path, 'utf8'));
  if (command === '--describe') console.log(JSON.stringify(describeRisks(), null, 2));
  else if (command === '--record') console.log(JSON.stringify(recordPieceApplications(read(receiptPath), read(reportPath)), null, 2));
  else throw new Error('USE --describe OR --record RECEIPT.json REPORT.json');
}
