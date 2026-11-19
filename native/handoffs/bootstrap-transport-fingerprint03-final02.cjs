'use strict';
// Box-authored bootstrap: recover the reviewed writer from original persisted tool bytes only.
const fs=require('node:fs'), path=require('node:path'), z=require('node:zlib');
const crypto=require('node:crypto'), child=require('node:child_process');
const session="a753a04c-1383-4870-877c-e70639eb0422";
const sessionFile='C:/Users/canace/.dsh/sessions/--F-AIproject-LevNote--/'+session+'/session.v4.jsonl.zstd';
const boxRoot="/home/user/levnote-dev-reliability-20261112-01/delivery/document-fingerprint-review-03/";
const host="F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03";
const writerName='transport-write-fingerprint03-final02.cjs', ownName='bootstrap-transport-fingerprint03-final02.cjs';
const pin=[5891, "f09642a1361f3107114f6b2daa85e921e8941e2cced29de9005de6afeb11e8d2"];
if(process.platform!=='win32'||process.argv.length!==2||path.resolve(process.cwd()).toLowerCase()!==path.resolve('F:/AIproject/LevNote').toLowerCase()) throw Error('Fixed Windows workspace, no arguments');
const digest=b=>crypto.createHash('sha256').update(b).digest('hex');
const ownBody=fs.readFileSync(__filename), calls=new Map(), found=new Map();
const compressed=fs.readFileSync(sessionFile);
for(let offset=0;offset<compressed.length;) {
 const frame=z.zstdDecompressSync(compressed.subarray(offset),{info:true});
 if(!(frame.engine.bytesWritten>0)) throw Error('No zstd progress');
 offset+=frame.engine.bytesWritten;
 for(const line of frame.buffer.toString('utf8').split('\n')) {
  if(!line) continue;
  const e=JSON.parse(line);
  if(e.type==='tool/call'&&e.data.name.endsWith('mcp__box__read_file')) {
   const a=JSON.parse(e.data.arguments);
   const name=[writerName,ownName].find(n=>boxRoot+n===a.path);
   if(name&&(a.offset===1||a.offset===undefined)) calls.set(e.data.callId,name);
  }
  if(e.type!=='tool/result'||!e.data.message||!calls.has(e.data.message.toolCallId)) continue;
  const m=e.data.message,name=calls.get(m.toolCallId);
  if(m.isError) throw Error('Original bootstrap/writer read failed');
  const rows=m.content.filter(v=>v.type==='text').map(v=>v.text).join('\n').trimEnd().split('\n'), lines=[];
  for(const row of rows) {
   const r=/^\s*(\d+)\t(.*)$/.exec(row);
   if(!r||Number(r[1])!==lines.length+1) throw Error('Invalid original numbered helper row');
   lines.push(r[2]);
  }
  const body=Buffer.from(lines.join('\n')+'\n');
  if(name===ownName&&!body.equals(ownBody)) throw Error('Bootstrap differs from original box tool result');
  if(name===writerName&&(body.length!==pin[0]||digest(body)!==pin[1])) continue;
  if(found.has(name)&&!found.get(name).body.equals(body)) throw Error('Conflicting original helper results');
  found.set(name,{body,ref:'session://'+session+'/event/'+e.seq});
 }
}
if(found.size!==2) throw Error('Original reviewed bootstrap/writer tool results incomplete');
const target=path.join(host,writerName), receipt=path.join(host,'bootstrap-receipt.json');
for(const p of [target,receipt]) if(fs.existsSync(p)) throw Error('Create-only bootstrap target exists: '+p);
fs.mkdirSync(host,{recursive:true});
fs.writeFileSync(target,found.get(writerName).body,{flag:'wx'});
if(!fs.readFileSync(target).equals(found.get(writerName).body)) throw Error('Writer fullbytes readback mismatch');
const run=child.spawnSync(process.execPath,[target],{cwd:'F:/AIproject/LevNote',encoding:'utf8',maxBuffer:1024*1024});
process.stdout.write(run.stdout||'');
process.stderr.write(run.stderr||'');
const result={status:run.status===0?'OPAQUE_WRITER_EXIT_ZERO':'OPAQUE_WRITER_FAILED_DO_NOT_REPEAT',pid:process.pid,writerPid:run.pid,writerExit:run.status,signal:run.signal,error:run.error?String(run.error):null,
 bootstrap:{path:__filename,bytes:ownBody.length,sha256:digest(ownBody),sourceRef:found.get(ownName).ref},
 writer:{path:target,bytes:pin[0],sha256:pin[1],sourceRef:found.get(writerName).ref},productCommands:0,base64Decoded:false,currentSourceAndReportWrites:0};
const body=Buffer.from(JSON.stringify(result,null,2)+'\n');
fs.writeFileSync(receipt,body,{flag:'wx'});
if(!fs.readFileSync(receipt).equals(body)) throw Error('Bootstrap receipt fullbytes readback mismatch');
console.log(JSON.stringify({receipt:{path:receipt,bytes:body.length,sha256:digest(body)},...result},null,2));
if(run.status!==0) process.exitCode=1;
