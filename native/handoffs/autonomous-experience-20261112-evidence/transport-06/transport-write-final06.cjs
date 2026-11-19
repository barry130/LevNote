'use strict';
// Pure byte transport from this authorized session's original tool results.
const fs = require('node:fs'), path = require('node:path');
const z = require('node:zlib'), crypto = require('node:crypto');
const session = '9e20bc49-fddb-490b-a752-c3b11da86ccc';
const sessionFile = 'C:/Users/canace/.dsh/sessions/--F-AIproject-LevNote--/' + session + '/session.v4.jsonl.zstd';
const boxRoot = '/home/user/levnote-dev-experience-20261112-01/delivery-06/';
const host = 'F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/transport-06';
if(process.platform!=='win32' || process.argv.length!==2 || path.resolve(process.cwd()).toLowerCase()!==path.resolve('F:/AIproject/LevNote').toLowerCase()) throw Error('Fixed Windows workspace, no arguments');
const specs = [
 ['final06-review.part01.b64',12150,'ada79f9ba7addf5ed6f3a856f37b770539ac1193224b0f0163fd5c062a2ab3c3'],
 ['final06-review.part02.b64',12150,'74c4ede6075f3c47d44cf05f7c306e5317c84dc57a8b0e529e1731c44b868b39'],
 ['final06-review.part03.b64',12150,'f5b92d0b0e0db787c3455f5525531938f3286fc02025df4c03630097dd6eaed5'],
 ['final06-review.part04.b64',12150,'a8937a62da2abd8d0c870abe5862856cdc77bf3edb7ba671924f0410fd67fa31'],
 ['final06-review.part05.b64',12150,'8f6fa83662cdc23f3e3dc47e2e743570e0d92b884b301170640e34d21a7ad631'],
 ['final06-review.part06.b64',12150,'85fce1d2bfe95c367dfabf7460536e6c476d7e11722f1978868e305a3e14ed78'],
 ['final06-review.part07.b64',12150,'b23b3de744144b944ec7c9e0a38067ea4d48df1d8935f2200c58b3bd449dd2e8'],
 ['final06-review.part08.b64',12150,'8d6eb8b3cd12ada3e0a7ac076dbf1a93443693bc41845f1d0976046399647c39'],
 ['final06-review.part09.b64',12150,'c95d5d103af1b9854f2e60410cb41d8fae0ddf639501f0682091e3969f2ffe3d'],
 ['final06-review.part10.b64',6359,'2c91b3491f456e68d59176303882a8de7b9fcfc835f65921805a81ef849d3e53'],
 ['transport-receipt.json',3913,'95d3554668b301547412089ffc053f982883b8d930247980f483dea099f171d8'],
 ['import-final06.py',10025,'ba50fdf157be36d97f07bc7841e0a127cdb15cd721c7ac1ba5eb53d3a55a5533'],
 ['decode-final06.py',3434,'e26b1eae706b05344163a0800a1720c671cf846f532996e5eb52407a967d9c68'],
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
   if(spec[0]==='decode-final06.py' && body.length===2550 && digest(body)==='41e7c775de745ce539e606dedc5d94006b9557f612b923c84e61a2d0e7e22e3c') continue; // explicitly superseded, known helper read
   throw Error('Original length/SHA mismatch '+spec[0]);
  }
  if(found.has(spec[0]) && !found.get(spec[0]).body.equals(body)) throw Error('Conflicting original results');
  found.set(spec[0],{body,ref:'session://'+session+'/event/'+e.seq});
 }
}
if(found.size!==specs.length) throw Error('Missing original read results: '+specs.filter(s=>!found.has(s[0])).map(s=>s[0]).join(','));
const encoded=specs.slice(0,10).map(s=>found.get(s[0]).body.toString('ascii').split('\n').join('')).join('');
if(encoded.length!==114280 || !/^[A-Za-z0-9+/]+={0,2}$/.test(encoded)) throw Error('Invalid complete base64');
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
const result={status:'FINAL06_OPAQUE_TRANSPORT_FILES_ONLY_NO_IMPORT_OR_SOURCE_WRITES',createdUTC:new Date().toISOString(),pid:process.pid,
 session,opaquePartCount:10,encodedCharacters:encoded.length,base64Decoded:false,
 files:targets.map(t=>({path:t.path,bytes:t.body.length,sha256:digest(t.body),sourceRef:t.sourceRef})),
 productCommands:0,currentSourceAndReportWrites:0};
const receiptBody=Buffer.from(JSON.stringify(result,null,2)+'\n');
fs.writeFileSync(receiptPath,receiptBody,{flag:'wx'});
if(!fs.readFileSync(receiptPath).equals(receiptBody)) throw Error('Receipt readback mismatch');
for(const t of targets) if(!fs.readFileSync(t.path).equals(t.body)) throw Error('Final inventory mismatch');
console.log(JSON.stringify({receipt:{path:receiptPath,bytes:fs.statSync(receiptPath).size,sha256:digest(fs.readFileSync(receiptPath))},...result},null,2));
