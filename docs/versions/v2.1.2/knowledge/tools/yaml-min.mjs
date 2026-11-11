// yaml-min.mjs — strict, dependency-free YAML subset parser (native Node).
//
// Scope: exactly the constructs used by the frozen v2.1.1 canonical corpus
// (ten lanes/*/records.yaml, case-intake sidecar, SOURCE_REGISTRY.yaml,
// RELATIONSHIP_CATALOG.yaml): block maps, block sequences, folded block
// scalars (">", ">-"), empty flow sequences ("[]"), plain and quoted scalars,
// comments. Everything else fails closed.
//
// Semantics mirror PyYAML (YAML 1.1) for this corpus: duplicate keys are a
// hard error (UniqueLoader discipline), plain-scalar typing follows the YAML
// 1.1 resolver (null/bool/int/float/timestamp); timestamps are normalized to
// ISO date strings so the parsed tree is JSON-representable. Parity with
// PyYAML is asserted by build-c1-canonical.test.mjs against the committed
// parity hashes (tools/make-parity-snapshot.py regenerates them).

import { readFileSync } from 'node:fs';

const INT_RE = /^[-+]?[0-9][0-9_]*$/;
const HEX_RE = /^[-+]?0x[0-9a-fA-F_]+$/;
const FLOAT_RE = /^[-+]?(?:\.[0-9]+|[0-9][0-9_]*(?:\.[0-9_]*)?)(?:[eE][-+]?[0-9]+)?$/;
const BOOL_TRUE = new Set(['true', 'True', 'TRUE', 'yes', 'Yes', 'YES', 'on', 'On', 'ON']);
const BOOL_FALSE = new Set(['false', 'False', 'FALSE', 'no', 'No', 'NO', 'off', 'Off', 'OFF']);
const NULL_WORDS = new Set(['~', 'null', 'Null', 'NULL', '']);
const DATE_RE = /^([0-9]{4})-([0-9]{1,2})-([0-9]{1,2})$/;

class YamlError extends Error {}

// defensive loop bound: the frozen corpus never needs more steps than this;
// exceeding it means a structural bug, not a big file.
const STEP_CAP = Number(process.env.YAML_MIN_STEP_CAP ?? 2_000_000);
let steps = 0;
function tick(lineNo) {
  steps += 1;
  if (steps > STEP_CAP) throw new YamlError(`yaml-min: step cap exceeded near line ${lineNo ?? '?'} (parser loop bug)`);
}

function fail(msg, lineNo) {
  throw new YamlError(`yaml-min: line ${lineNo ?? '?'}: ${msg}`);
}

// ---------------------------------------------------------------------------
// line helpers

function indentOf(line) {
  let n = 0;
  while (n < line.length && line[n] === ' ') n += 1;
  return n;
}

function isBlank(line) {
  const t = line.trim();
  return t.length === 0 || t.startsWith('#');
}

// strip an inline comment from a value (quote-aware); returns trimmed text.
function stripComment(text) {
  let inSingle = false;
  let inDouble = false;
  for (let i = 0; i < text.length; i += 1) {
    const c = text[i];
    if (inSingle) {
      if (c === "'") {
        if (text[i + 1] === "'") i += 1; // escaped ''
        else inSingle = false;
      }
    } else if (inDouble) {
      if (c === '\\') i += 1;
      else if (c === '"') inDouble = false;
    } else if (c === "'") inSingle = true;
    else if (c === '"') inDouble = true;
    else if (c === '#' && i > 0 && (text[i - 1] === ' ' || text[i - 1] === '\t')) {
      return stripComment(text.slice(0, i)).trimEnd();
    }
  }
  return text.trim();
}

// ---------------------------------------------------------------------------
// scalars

function resolvePlain(raw, lineNo) {
  if (BOOL_TRUE.has(raw)) return true;
  if (BOOL_FALSE.has(raw)) return false;
  if (NULL_WORDS.has(raw)) return null;
  if (INT_RE.test(raw)) {
    const v = Number(raw.replace(/_/g, ''));
    if (!Number.isSafeInteger(v)) fail(`integer out of range: ${raw}`, lineNo);
    return v;
  }
  if (HEX_RE.test(raw)) {
    const v = Number(raw.replace(/_/g, ''));
    if (!Number.isSafeInteger(v)) fail(`integer out of range: ${raw}`, lineNo);
    return v;
  }
  if (FLOAT_RE.test(raw) && /[.eE]/.test(raw.replace(/^[-+]/, ''))) return Number(raw.replace(/_/g, ''));
  if (raw === '.inf' || raw === '.Inf' || raw === '.INF') return (raw.startsWith('-') ? -Infinity : Infinity);
  if (raw === '.nan' || raw === '.NaN' || raw === '.NAN') return NaN;
  const d = DATE_RE.exec(raw);
  if (d) return `${d[1]}-${String(d[2]).padStart(2, '0')}-${String(d[3]).padStart(2, '0')}`;
  return raw;
}

function unquoteDouble(text, lineNo) {
  if (text.length < 2 || text[0] !== '"' || text[text.length - 1] !== '"') {
    fail('unterminated double-quoted scalar', lineNo);
  }
  let out = '';
  for (let i = 1; i < text.length - 1; i += 1) {
    const c = text[i];
    if (c !== '\\') { out += c; continue; }
    const e = text[++i];
    const simple = { '0': '\0', a: '\x07', b: '\b', t: '\t', n: '\n', v: '\v', f: '\f', r: '\r', e: '\x1b', ' ': ' ', '"': '"', '/': '/', '\\': '\\' };
    if (simple[e] !== undefined) out += simple[e];
    else if (e === 'x') { out += String.fromCharCode(parseInt(text.slice(i + 1, i + 3), 16)); i += 2; }
    else if (e === 'u') { out += String.fromCharCode(parseInt(text.slice(i + 1, i + 5), 16)); i += 4; }
    else if (e === 'U') { out += String.fromCodePoint(parseInt(text.slice(i + 1, i + 9), 16)); i += 8; }
    else fail(`unsupported double-quote escape \\${e}`, lineNo);
  }
  return out;
}

function unquoteSingle(text, lineNo) {
  if (text.length < 2 || text[0] !== "'" || text[text.length - 1] !== "'") {
    fail('unterminated single-quoted scalar', lineNo);
  }
  return text.slice(1, -1).replace(/''/g, "'");
}

// one-line flow sequence: "[a, b, 'c']" — flat, no nested collections
function parseFlowSeq(text, lineNo) {
  if (text[0] !== '[' || text[text.length - 1] !== ']') fail('malformed flow sequence', lineNo);
  const inner = text.slice(1, -1);
  if (inner.trim().length === 0) return [];
  const items = [];
  let cur = '';
  let inSingle = false;
  let inDouble = false;
  for (let i = 0; i < inner.length; i += 1) {
    const c = inner[i];
    if (inSingle) {
      if (c === "'") { if (inner[i + 1] === "'") { cur += "''"; i += 1; } else inSingle = false; }
      else cur += c;
    } else if (inDouble) {
      if (c === '\\') { cur += c + (inner[i + 1] ?? ''); i += 1; }
      else if (c === '"') { inDouble = false; cur += c; }
      else cur += c;
    } else if (c === "'") { inSingle = true; cur += c; }
    else if (c === '"') { inDouble = true; cur += c; }
    else if (c === ',') { items.push(cur.trim()); cur = ''; }
    else if (c === '[' || c === ']' || c === '{' || c === '}') fail('nested flow collections are outside the frozen corpus dialect', lineNo);
    else cur += c;
  }
  items.push(cur.trim());
  if (inSingle || inDouble) fail('unterminated quote inside flow sequence', lineNo);
  if (items.some((x) => x.length === 0)) fail('empty entry inside flow sequence', lineNo);
  return items.map((x) => {
    const v = parseInlineScalar(x, lineNo);
    if (v.kind === 'block' || v.kind === 'empty') fail('unsupported entry inside flow sequence', lineNo);
    return v.value;
  });
}

function parseInlineScalar(text, lineNo) {
  const t = text.trim();
  if (t.length === 0) return { kind: 'empty' };
  if (t === '[]') return { kind: 'value', value: [] };
  if (t.startsWith('[')) return { kind: 'value', value: parseFlowSeq(stripComment(t), lineNo) };
  if (t.startsWith('{')) fail('flow mappings are outside the frozen corpus dialect', lineNo);
  if (t === '|' || t === '>' || t === '|-' || t === '>-' || t === '|+' || t === '>+') return { kind: 'block', header: t };
  if (t[0] === '|' || t[0] === '>') fail(`unsupported block-scalar header "${t}"`, lineNo);
  if (t.startsWith('&') || t.startsWith('*') || t.startsWith('!')) {
    fail('anchors, aliases and tags are outside the frozen corpus dialect', lineNo);
  }
  if (t[0] === '"') return { kind: 'value', value: unquoteDouble(stripComment(t), lineNo) };
  if (t[0] === "'") return { kind: 'value', value: unquoteSingle(stripComment(t), lineNo) };
  return { kind: 'value', value: resolvePlain(stripComment(t), lineNo) };
}

// ---------------------------------------------------------------------------
// block scalars (folded only; literal "|" fails closed above)

function parseBlockScalar(lines, i, keyIndent, header, lineNo) {
  const chomp = header.endsWith('-') ? 'strip' : (header.endsWith('+') ? 'keep' : 'clip');
  const body = [];
  let j = i;
  let contentIndent = -1;
  while (j < lines.length) {
    const line = lines[j];
    if (line.trim().length === 0) { body.push({ blank: true, text: '' }); j += 1; continue; }
    const ind = indentOf(line);
    if (ind <= keyIndent) break;
    if (contentIndent < 0) contentIndent = ind;
    body.push({ blank: false, text: line.slice(contentIndent) });
    j += 1;
  }
  // YAML 1.1 folded-scalar assembly
  let out = '';
  let pendingBreaks = 0;
  let started = false;
  let lastWasLiteral = false;
  for (const b of body) {
    if (b.blank) { pendingBreaks += 1; continue; }
    const moreIndented = /^[ \t]/.test(b.text);
    if (!started) {
      out = b.text;
      started = true;
    } else if (moreIndented || lastWasLiteral) {
      // literal-style join: keep line breaks (one per pending blank + line break)
      out += '\n'.repeat(pendingBreaks > 0 ? pendingBreaks : 1) + b.text;
    } else if (pendingBreaks > 0) {
      out += '\n'.repeat(pendingBreaks) + b.text;
    } else {
      out += ' ' + b.text;
    }
    pendingBreaks = 0;
    lastWasLiteral = moreIndented;
  }
  if (!started) return { value: chomp === 'keep' ? '\n'.repeat(pendingBreaks) : '', next: j };
  if (chomp === 'strip') out = out.replace(/\n+$/, '');
  else if (chomp === 'clip') out = out.replace(/\n+$/, '') + '\n';
  else out = out + '\n'.repeat(pendingBreaks > 0 ? pendingBreaks : 1);
  return { value: out, next: j };
}

const MAP_KEY_RE = /^([^\s:#][^:]*?):(?:[ \t]+(.*))?$/;
const SEQ_RE = /^-(?:[ \t]+(.*))?$/;

function parseNode(lines, i, minIndent) {
  while (i < lines.length && isBlank(lines[i])) i += 1;
  if (i >= lines.length) return { value: null, next: i };
  const ind = indentOf(lines[i]);
  if (ind < minIndent) return { value: null, next: i };
  const text = lines[i].slice(ind);
  const lineNo = i + 1;
  if (SEQ_RE.test(text)) return parseSeq(lines, i, ind);
  if (MAP_KEY_RE.test(text)) return parseMap(lines, i, ind);
  const v = parseInlineScalar(text, lineNo);
  if (v.kind === 'block') return parseBlockScalar(lines, i + 1, ind - 1, v.header, lineNo);
  if (v.kind === 'empty') return { value: null, next: i + 1 };
  const cont = typeof v.value === 'string' && v.value.length > 0
    ? foldPlainContinuation(lines, i + 1, ind)
    : { value: null, next: i + 1 };
  if (cont.value !== null) return { value: v.value + ' ' + cont.value, next: cont.next };
  return { value: v.value, next: i + 1 };
}

// Parse a block mapping at indent `ind`. `firstLine` (optional) replaces
// lines[i] — used for sequence entries that open a mapping inline ("- id: x").
function parseMap(lines, i, ind, firstLine) {
  const out = {};
  let first = true;
  let j = i;
  while (j < lines.length) {
    tick(j + 1);
    if (isBlank(lines[j])) { j += 1; continue; }
    let cur = indentOf(lines[j]);
    let text;
    if (first && firstLine !== undefined) {
      // mapping opened inline on a sequence dash line ("- id: x"): the dash
      // line's own indent belongs to the sequence, not to this mapping block.
      cur = ind;
      text = firstLine;
    } else {
      if (cur < ind) break;
      if (cur > ind) fail('unexpected deeper indentation inside mapping', j + 1);
      text = lines[j].slice(cur);
      if (SEQ_RE.test(text)) break;
    }
    const lineNo = j + 1;
    const m = MAP_KEY_RE.exec(text);
    if (!m) fail(`not a mapping entry: "${text.slice(0, 60)}"`, lineNo);
    const key = m[1].trim();
    if (Object.prototype.hasOwnProperty.call(out, key)) {
      fail(`duplicate key "${key}" (UniqueLoader discipline)`, lineNo);
    }
    const rest = m[2] === undefined ? '' : m[2];
    if (rest.trim().length === 0) {
      j += 1;
      let k = j;
      while (k < lines.length && isBlank(lines[k])) k += 1;
      if (k < lines.length && indentOf(lines[k]) > ind) {
        const child = parseNode(lines, k, ind + 1);
        out[key] = child.value;
        j = child.next;
      } else if (k < lines.length && indentOf(lines[k]) === ind
        && SEQ_RE.test(lines[k].slice(ind))) {
        // YAML allows block sequences at the parent key's own indent
        const child = parseSeq(lines, k, ind);
        out[key] = child.value;
        j = child.next;
      } else {
        out[key] = null;
      }
    } else {
      const v = parseInlineScalar(rest, lineNo);
      if (v.kind === 'block') {
        const r = parseBlockScalar(lines, j + 1, ind, v.header, lineNo);
        out[key] = r.value;
        j = r.next;
      } else if (v.kind === 'empty') {
        out[key] = null;
        j += 1;
      } else {
        j += 1;
        let value = v.value;
        if (typeof value === 'string' && value.length > 0) {
          const cont = foldPlainContinuation(lines, j, ind);
          if (cont.value !== null) { value = value + ' ' + cont.value; j = cont.next; }
        }
        out[key] = value;
      }
    }
    first = false;
  }
  return { value: out, next: j };
}

function parseSeq(lines, i, ind) {
  const out = [];
  let j = i;
  while (j < lines.length) {
    tick(j + 1);
    if (isBlank(lines[j])) { j += 1; continue; }
    const cur = indentOf(lines[j]);
    if (cur < ind) break;
    if (cur > ind) fail('unexpected deeper indentation inside sequence', j + 1);
    const text = lines[j].slice(cur);
    const lineNo = j + 1;
    const m = SEQ_RE.exec(text);
    if (!m) break;
    const rest = m[1] === undefined ? '' : m[1];
    if (rest.trim().length === 0) {
      j += 1;
      let k = j;
      while (k < lines.length && isBlank(lines[k])) k += 1;
      if (k < lines.length && indentOf(lines[k]) > ind) {
        const child = parseNode(lines, k, ind + 1);
        out.push(child.value);
        j = child.next;
      } else {
        out.push(null);
      }
    } else {
      const trimmedRest = rest.trim();
      const looksMap = MAP_KEY_RE.test(rest)
        && !trimmedRest.startsWith("'") && !trimmedRest.startsWith('"')
        && !trimmedRest.startsWith('[') && !trimmedRest.startsWith('{')
        && trimmedRest !== '[]';
      if (looksMap) {
        // mapping opened inline on the dash line; its indent is dash indent + 2
        const r = parseMap(lines, j, ind + 2, rest);
        out.push(r.value);
        j = r.next;
      } else {
        const v = parseInlineScalar(rest, lineNo);
        if (v.kind === 'block') {
          const r = parseBlockScalar(lines, j + 1, ind, v.header, lineNo);
          out.push(r.value);
          j = r.next;
        } else if (v.kind === 'empty') {
          out.push(null);
          j += 1;
        } else {
          j += 1;
          let value = v.value;
          if (typeof value === 'string' && value.length > 0) {
            const cont = foldPlainContinuation(lines, j, ind);
            if (cont.value !== null) { value = value + ' ' + cont.value; j = cont.next; }
          }
          out.push(value);
        }
      }
    }
  }
  return { value: out, next: j };
}

// plain multi-line continuation: deeper-indented non-structural lines fold in
function foldPlainContinuation(lines, i, keyIndent) {
  const parts = [];
  let j = i;
  while (j < lines.length) {
    if (isBlank(lines[j])) { j += 1; continue; }
    const ind = indentOf(lines[j]);
    if (ind <= keyIndent) break;
    const text = lines[j].slice(ind);
    if (!SEQ_RE.test(text) && MAP_KEY_RE.test(text)) break;
    if (SEQ_RE.test(text)) break;
    parts.push(stripComment(text));
    j += 1;
  }
  if (parts.length === 0) return { value: null, next: i };
  return { value: parts.join(' '), next: j };
}

// ---------------------------------------------------------------------------

export function parseYaml(text, sourceName = '<input>') {
  const where = (n) => `yaml-min[${sourceName}]: line ${n ?? '?'}: `;
  if (text.charCodeAt(0) === 0xFEFF) throw new YamlError(where(1) + 'UTF-8 BOM is outside the frozen corpus dialect');
  if (text.includes('\r')) throw new YamlError(where(1) + 'CR bytes are outside the frozen corpus dialect (LF-only corpus)');
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i += 1) {
    const lead = /^[ \t]*/.exec(lines[i])[0];
    if (lead.includes('\t')) throw new YamlError(where(i + 1) + 'tab indentation is outside the frozen corpus dialect');
    const t = lines[i].trim();
    if (t === '---' || t === '...') throw new YamlError(where(i + 1) + 'multi-document streams are outside the frozen corpus dialect');
  }
  const r = parseNode(lines, 0, 0);
  for (let k = r.next; k < lines.length; k += 1) {
    if (!isBlank(lines[k])) throw new YamlError(where(k + 1) + `trailing content after document: "${lines[k].slice(0, 40)}"`);
  }
  return r.value;
}

export function parseYamlFile(path) {
  return parseYaml(readFileSync(path, 'utf8'), path);
}

export { YamlError };
