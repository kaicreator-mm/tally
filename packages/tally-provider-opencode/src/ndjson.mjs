/**
 * NDJSON event-stream parsing for `opencode run --format json` stdout.
 *
 * Observed envelope (opencode-ai@1.19.6, evidence: PR #471 runtimes/opencode,
 * 2026-11-19 field runs; do not invent additional vocabulary):
 *
 *   {"type":"step_start"|"text"|"tool_use"|"step_finish",
 *    "timestamp":<ms>,"sessionID":"ses_*","part":{...}}
 *
 * Fail-safe rules:
 *   - noise before the first '{' of a line is skipped;
 *   - lines that do not parse as JSON objects are ignored (never thrown);
 *   - a truncated trailing line (no closing '\n') is dropped at end();
 *   - unknown envelope `type` values are ignored (the stream may grow new
 *     types in future releases; observing them is not this adapter's job).
 *
 * Aggregates:
 *   - summary: the text of the LAST `text` part (the final answer);
 *   - usage: element-wise sum of `step_finish` part.tokens
 *     {input, output, reasoning, cache:{read, write}};
 *   - sessionId: the last observed ses_* session handle;
 *   - sawJSON: whether at least one well-formed envelope was observed
 *     (drives the plain-text degrade decision in the adapter).
 */

function addToken(target, source, sourceKey, targetKey) {
  const value = source === null || source === undefined ? undefined : source[sourceKey];
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    target[targetKey] += value;
  }
}

function addTokens(usage, tokens) {
  if (!tokens || typeof tokens !== 'object' || Array.isArray(tokens)) return;
  addToken(usage, tokens, 'input', 'inputTokens');
  addToken(usage, tokens, 'output', 'outputTokens');
  addToken(usage, tokens, 'reasoning', 'reasoningTokens');
  addToken(usage, tokens.cache, 'read', 'cacheReadTokens');
  addToken(usage, tokens.cache, 'write', 'cacheWriteTokens');
}

function zeroUsage() {
  return {inputTokens: 0, outputTokens: 0, reasoningTokens: 0,
    cacheReadTokens: 0, cacheWriteTokens: 0};
}

export function createNdjsonEventSink() {
  let buffer = '';
  let summary = null;
  let sessionId = null;
  let sawJSON = false;
  let eventCount = 0;
  const usage = zeroUsage();

  function handleLine(line) {
    const start = line.indexOf('{');
    if (start === -1) return; // pure noise line
    let envelope;
    try {
      envelope = JSON.parse(line.slice(start));
    } catch {
      return; // garbage / truncated mid-stream line: ignored, never thrown
    }
    if (!envelope || typeof envelope !== 'object' || Array.isArray(envelope)) return;
    if (typeof envelope.type !== 'string') return;
    const part = envelope.part;
    if (!part || typeof part !== 'object' || Array.isArray(part)) return;
    sawJSON = true;
    eventCount += 1;
    if (typeof envelope.sessionID === 'string' && envelope.sessionID.startsWith('ses_')) {
      sessionId = envelope.sessionID;
    }
    switch (envelope.type) {
      case 'text':
        if (typeof part.text === 'string') summary = part.text;
        break;
      case 'step_finish':
        addTokens(usage, part.tokens);
        break;
      case 'step_start':
      case 'tool_use':
      default:
        break; // observed-but-unused and unknown types: ignored (fail-safe)
    }
  }

  return {
    /** Feed a raw stdout chunk; partial lines are buffered across pushes. */
    push(chunk) {
      if (typeof chunk !== 'string') return;
      buffer += chunk;
      let newlineAt = buffer.indexOf('\n');
      while (newlineAt !== -1) {
        const line = buffer.slice(0, newlineAt).replace(/\r$/, '');
        buffer = buffer.slice(newlineAt + 1);
        handleLine(line);
        newlineAt = buffer.indexOf('\n');
      }
    },
    /** Finalize: a truncated trailing line is dropped (safe). */
    end() {
      buffer = '';
    },
    snapshot() {
      return {summary, sessionId, sawJSON, eventCount, usage: {...usage}};
    },
  };
}

/** Convenience wrapper: parse one whole stdout string. */
export function parseNdjsonRunOutput(text) {
  const sink = createNdjsonEventSink();
  sink.push(typeof text === 'string' ? text : '');
  sink.end();
  return sink.snapshot();
}
