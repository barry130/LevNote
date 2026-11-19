'use strict';
// Pure byte transport from this authorized session's original tool results.
const fs = require('node:fs'), path = require('node:path');
const z = require('node:zlib'), crypto = require('node:crypto');
const session = "a753a04c-1383-4870-877c-e70639eb0422";
const sessionFile = 'C:/Users/canace/.dsh/sessions/--F-AIproject-LevNote--/' + session + '/session.v4.jsonl.zstd';
const boxRoot = "/home/user/levnote-dev-reliability-20261112-01/delivery/document-fingerprint-review-03/";
const host = "F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03";
if(process.platform!=='win32' || process.argv.length!==2 || path.resolve(process.cwd()).toLowerCase()!==path.resolve('F:/AIproject/LevNote').toLowerCase()) throw Error('Fixed Windows workspace, no arguments');
const specs = [
 [
  "fingerprint-review03.part01.b64",
  12150,
  "68f8ca742bb564943966c795d1d1af852e5c58c3274d399f350d4e96eb342fb2"
 ],
 [
  "fingerprint-review03.part02.b64",
  12150,
  "a9d7464d5411abad45dc587168e7cc83f4399d23bd41555430639726c689fb37"
 ],
 [
  "fingerprint-review03.part03.b64",
  12150,
  "93ad4d5d9e5e8896108b31091a0c1d1fb00daee162b40cd53fdd36aea3acba32"
 ],
 [
  "fingerprint-review03.part04.b64",
  12150,
  "e69d74919b3a129bea581d92089331b790a04c1a7ef35a5f747a73d373dcb111"
 ],
 [
  "fingerprint-review03.part05.b64",
  12150,
  "fdb446619417a21b8194be7f77cb04747bf091c15520f2dab9e116d9484edd35"
 ],
 [
  "fingerprint-review03.part06.b64",
  12150,
  "4313fc30297f2f92dcd026f1a3e93b81e92e72afde4252160b7cc11df2440238"
 ],
 [
  "fingerprint-review03.part07.b64",
  12150,
  "ea60cbd06c65f335e3b80d0d4f7dd5c3e4f3789378171aeec2c3a4355a2468f6"
 ],
 [
  "fingerprint-review03.part08.b64",
  12150,
  "2b556a9c3ecf81d883bec42c0a30cc2c95621c1d62bf0be207af0ead45770081"
 ],
 [
  "fingerprint-review03.part09.b64",
  7015,
  "c92c8f3b9d0285c59d97ed6e21daa4624a1d27ad6e3aa3a648617b8fd9bd971d"
 ],
 [
  "transport-receipt.json",
  4436,
  "b0eac7f7153071e34de45bb203a34d8febf63276b3f6ca65c760402abdc12d44"
 ],
 [
  "decode-fingerprint03.py",
  3398,
  "7a937f762e843116967c145dd2e8f2341bfbe58e74c6dbc1dbdb5d38d24bbd0e"
 ],
 [
  "import-fingerprint03-final.py",
  11251,
  "77c89b41848ec420ee27a4813318f74cc86059839e4e5f465a8cd5698ae28d13"
 ]
];
const digest = b => crypto.createHash('sha256').update(b).digest('hex');
const calls = new Map(), found = new Map();
const compressed = fs.readFileSync(sessionFile);
for (let offset=0; offset<compressed.length;) {
 const frame = z.zstdDecompressSync(compressed.subarray(offset),{info:true});
 if (!(frame.engine.bytesWritten>0)) throw Error('No zstd progress');
 offset += frame.engine.bytesWritten;
 for (const line of frame.buffer.toString('utf8').split('\n')) {
  if (!line) continue;
  const e=JSON.parse(line);
  if(e.type==='tool/call' && e.data.name.endsWith('mcp__box__read_file')) {
   const a=JSON.parse(e.data.arguments);
   const spec=specs.find(s=>boxRoot+s[0]===a.path);
   if(spec && (a.offset===1 || a.offset===undefined)) calls.set(e.data.callId,spec);
  }
  if(e.type!=='tool/result' || !e.data.message || !calls.has(e.data.message.toolCallId)) continue;
  const m=e.data.message, spec=calls.get(m.toolCallId);
  if(m.isError) throw Error('Original file read returned error');
  const text=m.content.filter(v=>v.type==='text').map(v=>v.text).join('\n');
  const rows=text.trimEnd().split('\n');
  const lines=[];
  for(const row of rows) {
   const match=/^\s*(\d+)\t(.*)$/.exec(row);
   if(!match || Number(match[1])!==lines.length+1) throw Error('Invalid numbered original row '+spec[0]);
   lines.push(match[2]);
  }
  const body=Buffer.from(lines.join('\n')+'\n','utf8');
  if(body.length!==spec[1] || digest(body)!==spec[2]) {
   throw Error('Original length/SHA mismatch '+spec[0]);
  }
  if(found.has(spec[0]) && !found.get(spec[0]).body.equals(body)) throw Error('Conflicting original results');
  found.set(spec[0],{body,ref:'session://'+session+'/event/'+e.seq});
 }
}
if(found.size!==specs.length) throw Error('Missing original read results: '+specs.filter(s=>!found.has(s[0])).map(s=>s[0]).join(','));
const encoded=specs.slice(0,9).map(s=>found.get(s[0]).body.toString('ascii').split('\n').join('')).join('');
if(encoded.length!==102928 || !/^[A-Za-z0-9+/]+={0,2}$/.test(encoded)) throw Error('Invalid complete base64');
const targets=specs.map(s=>({path:path.join(host,s[0]),body:found.get(s[0]).body,sourceRef:found.get(s[0]).ref}));
const receiptPath=path.join(host,'transport-mirror-receipt.json');
// Complete all-target absence preflight before any opaque writes; no retries/overwrite.
for(const t of [...targets,{path:receiptPath}]) if(fs.existsSync(t.path)) throw Error('Create-only target exists: '+t.path);
for(const t of targets) {
 if(fs.existsSync(t.path)) throw Error('Target appeared after preflight');
 fs.mkdirSync(path.dirname(t.path),{recursive:true});
 fs.writeFileSync(t.path,t.body,{flag:'wx'});
 if(!fs.readFileSync(t.path).equals(t.body)) throw Error('Opaque write readback mismatch');
}
const result={status:'TASK11_PRIMARY03_OPAQUE_TRANSPORT_FILES_ONLY_NO_IMPORT_OR_SOURCE_WRITES',createdUTC:new Date().toISOString(),pid:process.pid,
 session,opaquePartCount:9,encodedCharacters:encoded.length,base64Decoded:false,
 files:targets.map(t=>({path:t.path,bytes:t.body.length,sha256:digest(t.body),sourceRef:t.sourceRef})),
 productCommands:0,currentSourceAndReportWrites:0};
const receiptBody=Buffer.from(JSON.stringify(result,null,2)+'\n');
fs.writeFileSync(receiptPath,receiptBody,{flag:'wx'});
if(!fs.readFileSync(receiptPath).equals(receiptBody)) throw Error('Receipt readback mismatch');
for(const t of targets) if(!fs.readFileSync(t.path).equals(t.body)) throw Error('Final inventory mismatch');
console.log(JSON.stringify({receipt:{path:receiptPath,bytes:fs.statSync(receiptPath).size,sha256:digest(fs.readFileSync(receiptPath))},...result},null,2));
