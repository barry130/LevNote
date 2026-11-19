'use strict';
// Prepared for Root review only. Two create-only byte writes; never audit, decode, import or run product code.
const fs = require('node:fs'), path = require('node:path');
const z = require('node:zlib'), crypto = require('node:crypto');
const session = 'a753a04c-1383-4870-877c-e70639eb0422';
const sessionFile = 'C:/Users/canace/.dsh/sessions/--F-AIproject-LevNote--/' + session + '/session.v4.jsonl.zstd';
const sourcePath = '/home/user/levnote-dev-reliability-20261112-01/delivery/document-fingerprint-review-03/audit-fingerprint03-host-import-01.py';
const workspace = 'F:/AIproject/LevNote';
const host = workspace + '/native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03';
const script = path.join(host, 'mirror-fingerprint03-host-auditor-01.cjs');
const target = path.join(host, 'audit-fingerprint03-host-import-01.py');
const receiptPath = path.join(host, 'auditor-mirror-receipt-01.json');
const expected = {bytes: 17502, sha256: '3117d3174e22565119306cad7c585c76f53b85a3b0e428363f5e67ac2260bedd'};
const digest = b => crypto.createHash('sha256').update(b).digest('hex');
const ident = b => ({bytes: b.length, sha256: digest(b)});
const samePath = (a, b) => path.resolve(a).toLowerCase() === path.resolve(b).toLowerCase();
if (process.platform !== 'win32' || process.argv.length !== 2 || !samePath(process.cwd(), workspace) || !samePath(__filename, script)) {
 throw Error('Fixed Windows workspace and mirror script, no arguments');
}
function plainDirectory(p) {
 for (let q = path.resolve(p);;) {
  const info = fs.lstatSync(q);
  if (!info.isDirectory() || info.isSymbolicLink()) throw Error('Non-plain fixed directory: ' + q);
  const parent = path.dirname(q);
  if (parent === q) break;
  q = parent;
 }
 if (!samePath(fs.realpathSync.native(p), p)) throw Error('Fixed directory redirects');
}
function absent(p) {
 let exists = true;
 try { fs.lstatSync(p); } catch (e) { if (e.code === 'ENOENT') exists = false; else throw e; }
 if (exists) throw Error('Create-only target exists: ' + p);
}
function readRegular(p) {
 const info = fs.lstatSync(p);
 if (!info.isFile() || info.isSymbolicLink()) throw Error('Non-regular fixed file: ' + p);
 return fs.readFileSync(p);
}
function fixedTransportInventory() {
 const result = {};
 for (const name of fs.readdirSync(host).sort()) {
  if (name === path.basename(target) || name === path.basename(receiptPath)) continue;
  result[name] = ident(readRegular(path.join(host, name)));
 }
 return result;
}
plainDirectory(host);
absent(target); absent(receiptPath);
const startedUTC = new Date().toISOString();
const originalTransport = fixedTransportInventory();
const calls = new Map(), fullReads = [], partialReads = [];
const decoder = new TextDecoder('utf-8', {fatal: true});
let authoritativeBody = null, frames = 0, events = 0;
function seq(e) {
 if (!Number.isSafeInteger(e.seq) || e.seq < 1) throw Error('Original event has no exact sequence');
 return e.seq;
}
function visit(line) {
 if (line.length === 0) return;
 const e = JSON.parse(decoder.decode(line));
 events++;
 if (e.type === 'tool/call' && typeof e.data?.name === 'string' && e.data.name.endsWith('mcp__box__read_file')) {
  const a = JSON.parse(e.data.arguments);
  if (a.path === sourcePath) {
   if (typeof e.data.callId !== 'string' || calls.has(e.data.callId)) throw Error('Missing or duplicate original callId');
   const full = a.offset === 1 && Number.isInteger(a.limit) && a.limit >= 295;
   calls.set(e.data.callId, {callId: e.data.callId, callSeq: seq(e), offset: a.offset ?? null,
                             limit: a.limit ?? null, full, results: 0});
  }
 }
 if (e.type === 'tool/result' && e.data?.message && calls.has(e.data.message.toolCallId)) {
  const m = e.data.message, call = calls.get(m.toolCallId);
  call.results++;
  if (call.results !== 1) throw Error('Duplicate result for exact original callId');
  const reference = {callId: call.callId, callSeq: call.callSeq, resultSeq: seq(e),
                     callRef: 'session://' + session + '/event/' + call.callSeq,
                     resultRef: 'session://' + session + '/event/' + e.seq, offset: call.offset, limit: call.limit};
  if (!call.full) {
   // Explicitly counted partial reads are never used as byte candidates.
   partialReads.push(reference);
  } else {
   if (m.isError || !Array.isArray(m.content)) throw Error('Full original file read returned error or no content');
   const text = m.content.filter(v => v.type === 'text').map(v => v.text).join('\n');
   const rows = text.trimEnd().split('\n'), lines = [];
   for (const row of rows) {
    const match = /^\s*(\d+)\t(.*)$/.exec(row);
    if (!match || Number(match[1]) !== lines.length + 1) throw Error('Full original numbered body missing, truncated or out of order');
    lines.push(match[2]);
   }
   if (lines.length !== 295) throw Error('Full original body is not all numbered lines 1-295');
   const body = Buffer.from(lines.join('\n') + '\n', 'utf8');
   if (body.length !== expected.bytes || digest(body) !== expected.sha256) throw Error('Full original byte/pin conflict; no older result fallback');
   if (authoritativeBody !== null && !authoritativeBody.equals(body)) throw Error('Conflicting full originals; no choosing an older read');
   authoritativeBody = body;
   fullReads.push({...reference, numberedLines: 295, ...ident(body)});
  }
 }
}
const compressed = readRegular(sessionFile);
let pending = Buffer.alloc(0);
for (let offset = 0; offset < compressed.length;) {
 const frame = z.zstdDecompressSync(compressed.subarray(offset), {info: true});
 const consumed = frame.engine.bytesWritten;
 if (!Number.isSafeInteger(consumed) || consumed <= 0 || consumed > compressed.length - offset) throw Error('No valid zstd frame progress');
 offset += consumed; frames++;
 const chunk = pending.length ? Buffer.concat([pending, frame.buffer]) : frame.buffer;
 let start = 0, end;
 while ((end = chunk.indexOf(10, start)) !== -1) { visit(chunk.subarray(start, end)); start = end + 1; }
 pending = Buffer.from(chunk.subarray(start));
}
if (pending.length) visit(pending);
for (const call of calls.values()) if (call.full && call.results !== 1) throw Error('Missing result for a full original call; no fallback');
if (!authoritativeBody || fullReads.length < 1) throw Error('No complete pinned original auditor body found');
// Every full same-path original was validated, not just the selected/newest/oldest one.
fullReads.sort((a, b) => a.resultSeq - b.resultSeq);
partialReads.sort((a, b) => a.callSeq - b.callSeq);
if (JSON.stringify(fixedTransportInventory()) !== JSON.stringify(originalTransport)) throw Error('Original fixed transport changed before mirror');
absent(target); absent(receiptPath); // Complete both-target preflight immediately before any wx write.
fs.writeFileSync(target, authoritativeBody, {flag: 'wx'});
if (!readRegular(target).equals(authoritativeBody)) throw Error('Auditor original-byte mirror readback mismatch');
const result = {status: 'TASK11_PRIMARY03_AUDITOR_BYTE_MIRROR_ONLY_NO_AUDIT_IMPORT_OR_PRODUCT_EXECUTED',
 startedUTC, endedUTC: new Date().toISOString(), pid: process.pid, argv: process.argv, cwd: process.cwd(),
 script: {path: script, ...ident(readRegular(script))}, sourceSession: session, sourceSessionFile: sessionFile,
 exactOriginalToolPath: sourcePath, zstdFramesRead: frames, eventsRead: events,
 allFullSamePathOriginalsValidatedNoOlderFallback: true, fullOriginalReadCount: fullReads.length, fullOriginalReads: fullReads,
 partialExactPathCallsIgnored: [...calls.values()].filter(c => !c.full).length, partialResultsObserved: partialReads.length,
 partialOriginalReadsNotByteCandidates: partialReads,
 mirroredAuditor: {path: target, ...ident(authoritativeBody)}, mirrorReceiptPath: receiptPath,
 completeBothTargetsAbsentBeforeAnyWrite: true, createdFileCount: 2, originalTransportFilesBefore: originalTransport,
 productCommands: 0, spawns: 0, sourceAndCurrentReportWrites: 0, decoderOrImporterOrAuditorExecuted: false,
 notWindowsOrWholeApplicationAcceptance: true};
const receiptBody = Buffer.from(JSON.stringify(result, null, 2) + '\n', 'utf8');
absent(receiptPath);
fs.writeFileSync(receiptPath, receiptBody, {flag: 'wx'});
if (!readRegular(receiptPath).equals(receiptBody) || !readRegular(target).equals(authoritativeBody)) throw Error('Final two-file mirror readback mismatch');
if (JSON.stringify(fixedTransportInventory()) !== JSON.stringify(originalTransport)) throw Error('Original fixed transport changed during mirror');
console.log(JSON.stringify({...result, receipt: {path: receiptPath, ...ident(readRegular(receiptPath))},
                           originalTransportFilesAfterByteExact: true, exitCode: 0}, null, 2));
