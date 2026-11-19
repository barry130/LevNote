'use strict';
// Byte transport only. Reads this authorized session's tool results, never executes product code.
const fs = require('node:fs');
const path = require('node:path');
const z = require('node:zlib');
const crypto = require('node:crypto');
const sessionPath = 'C:/Users/canace/.dsh/sessions/--F-AIproject-LevNote--/9e20bc49-fddb-490b-a752-c3b11da86ccc/session.v4.jsonl.zstd';
const tag = process.argv[2] || '01';
const transfers = {
  '01': { prefix: '/home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-01/mirror-01.part', expectedSize: 50873, expectedHash: '91b48f6504c42eb48966d3878bee3351dcab8dc64cc056a1b7df8a1de8664641', encodedLength: 67832, lastPartLength: 7832, partCount: 6, partPattern: /mirror-01\.part(0[1-6])\.b64$/, archiveName: 'mirror-01.tar.gz' },
  '03': { prefix: '/home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-03/mirror-03.part', expectedSize: 48175, expectedHash: '520e57570ca53d598a434f5fdbeaa477d64921b8b46d234671f9a9156acd79b4', encodedLength: 64236, lastPartLength: 4236, partCount: 6, partPattern: /mirror-03\.part(0[1-6])\.b64$/, archiveName: 'mirror-03.tar.gz' },
  'report04': { prefix: '/home/user/levnote-dev-experience-20261112-01/report-only-04/report-mirror-04.part', expectedSize: 8020, expectedHash: '6d15080e757588869916b9db473ea5b62038212c4971b89a29966a9ad5526a3d', encodedLength: 10696, lastPartLength: 10696, partCount: 1, partPattern: /report-mirror-04\.part(01)\.b64$/, archiveName: 'report-mirror-04.tar.gz' }
};
if (!Object.hasOwn(transfers, tag)) throw Error('Unrecognized authorized transfer');
const { prefix, expectedSize, expectedHash, encodedLength, lastPartLength, partCount, partPattern, archiveName } = transfers[tag];
const workspace = path.resolve('F:/AIproject/LevNote');
const stage = path.join(workspace, 'native/handoffs/autonomous-experience-20261112-evidence/incoming-' + tag);
const digest = b => crypto.createHash('sha256').update(b).digest('hex');
const calls = new Map(), chunks = new Map(), refs = new Map();
const compressed = fs.readFileSync(sessionPath);
for (let offset = 0; offset < compressed.length;) {
  const frame = z.zstdDecompressSync(compressed.subarray(offset), { info: true });
  if (!(frame.engine.bytesWritten > 0)) throw Error('No zstd frame progress');
  offset += frame.engine.bytesWritten;
  for (const line of frame.buffer.toString('utf8').split('\n')) {
    if (!line) continue;
    const e = JSON.parse(line);
    if (e.type === 'tool/call' && e.data.name.endsWith('mcp__box__read_file')) {
      const a = JSON.parse(e.data.arguments);
      const match = typeof a.path === 'string' && a.path.startsWith(prefix) && partPattern.exec(a.path);
      if (match) calls.set(e.data.callId, Number(match[1]));
    }
    if (e.type !== 'tool/result' || !calls.has(e.data.message.toolCallId)) continue;
    const m = e.data.message, part = calls.get(m.toolCallId);
    if (m.isError) throw Error('Chunk tool returned error');
    const text = m.content.map(v => v.type === 'text' ? v.text : '').join('\n');
    const rows = text.trimEnd().split('\n');
    let n = 0, value = '';
    for (const row of rows) {
      const match = /^\s*(\d+)\t([A-Za-z0-9+/=]+)$/.exec(row);
      if (!match || Number(match[1]) !== ++n) throw Error('Invalid numbered base64 row in part ' + part);
      value += match[2];
    }
    if (value.length !== (part === partCount ? lastPartLength : 12000)) throw Error('Unexpected chunk length');
    if (chunks.has(part) && chunks.get(part) !== value) throw Error('Conflicting repeated chunk');
    chunks.set(part, value); refs.set(part, 'session://9e20bc49-fddb-490b-a752-c3b11da86ccc/event/' + e.seq);
  }
}
if (chunks.size !== partCount) throw Error('Incomplete chunk set');
const encoded = Array.from({length:partCount}, (_,i)=>chunks.get(i+1)).join('');
const archive = Buffer.from(encoded, 'base64');
if (encoded.length !== encodedLength || archive.length !== expectedSize || digest(archive) !== expectedHash) throw Error('Archive byte identity mismatch');
const tar = z.gunzipSync(archive), entries = []; let pax = {};
const readString = b => b.toString('utf8').replace(/\0.*$/s, '');
for (let offset = 0; offset + 512 <= tar.length;) {
  const header = tar.subarray(offset, offset + 512); offset += 512;
  if (header.every(v=>v===0)) break;
  const size = parseInt(readString(header.subarray(124,136)).trim(),8);
  if (!Number.isSafeInteger(size) || size < 0 || offset+size > tar.length) throw Error('Invalid tar size');
  const stored = parseInt(readString(header.subarray(148,156)).trim(),8);
  let sum=0; for(let i=0;i<512;i++) sum += i>=148&&i<156 ? 32 : header[i];
  if (stored !== sum) throw Error('Invalid tar header checksum');
  const body = Buffer.from(tar.subarray(offset, offset+size)); offset += Math.ceil(size/512)*512;
  const type=String.fromCharCode(header[156]);
  if (type === 'x') {
    pax={}; for(let p=0;p<body.length;){const space=body.indexOf(32,p), len=Number(body.subarray(p,space).toString('ascii'));
      if(!(len>0)||p+len>body.length) throw Error('Invalid PAX record');
      const record=body.subarray(space+1,p+len-1).toString('utf8'), eq=record.indexOf('=');p+=len;pax[record.slice(0,eq)]=record.slice(eq+1);}
    continue;
  }
  if (type !== '0' && type !== '\0') throw Error('Unexpected tar entry type '+type);
  const pre=readString(header.subarray(345,500));
  const name=pax.path || (pre ? pre+'/' : '')+readString(header.subarray(0,100)); pax={};
  if(!name.startsWith('native/') || name.includes('\\') || path.posix.isAbsolute(name) || name.split('/').some(v=>v==='..'||v==='')) throw Error('Unsafe tar path');
  if(entries.some(e=>e.path===name))throw Error('Duplicate tar path');
  entries.push({path:name,bytes:body.length,sha256:digest(body),body});
}
// Preflight every destination before any writes. Immutable repeated transfers must match exactly.
const targets=[{path:path.join(stage,archiveName),body:archive},...entries.map(e=>({path:path.join(stage,e.path),body:e.body}))];
for(const t of targets)if(fs.existsSync(t.path)&&digest(fs.readFileSync(t.path))!==digest(t.body))throw Error('Existing staging identity conflict: '+t.path);
for(const t of targets){fs.mkdirSync(path.dirname(t.path),{recursive:true});if(!fs.existsSync(t.path))fs.writeFileSync(t.path,t.body);if(digest(fs.readFileSync(t.path))!==digest(t.body))throw Error('Write verification failure');}
const inventory=entries.map(({body,...e})=>e);
console.log(JSON.stringify({archive:{path:path.join(stage,archiveName),bytes:archive.length,sha256:digest(archive)},chunkRefs:Object.fromEntries(refs),entries:inventory},null,2));
