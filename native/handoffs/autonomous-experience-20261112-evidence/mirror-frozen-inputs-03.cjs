'use strict';
// Opaque byte delivery only; no source reconstruction, formatting or product execution.
const fs = require('node:fs'), path = require('node:path'), crypto = require('node:crypto');
const root = path.resolve('F:/AIproject/LevNote');
const evidence = 'native/handoffs/autonomous-experience-20261112-evidence';
const incoming = path.join(root, evidence, 'incoming-03');
const receiptRel = evidence + '/freeze-03/mirror-inputs.json';
const manifestRel = evidence + '/freeze-03/source-inputs.json';
const archive = path.join(incoming, 'mirror-03.tar.gz');
const hash = b => crypto.createHash('sha256').update(b).digest('hex');
const checked = (p, bytes, sha256) => {
  const b = fs.readFileSync(p);
  if (b.length !== bytes || hash(b) !== sha256) throw Error('Pinned input identity mismatch: ' + p);
  return b;
};
checked(archive, 48175, '520e57570ca53d598a434f5fdbeaa477d64921b8b46d234671f9a9156acd79b4');
const receiptBody = checked(path.join(incoming, receiptRel), 2902, 'f0340241dff6096cc58bc674eb7f49496fcf14ece6287c87b3ae0d4784d0cf2e');
const manifestBody = checked(path.join(incoming, manifestRel), 7681, 'ccc1461d5b26a34fe4c7788b4fff7dfa9d40cd11812bdd1ae2745f28586ae71b');
const receipt = JSON.parse(receiptBody.toString('utf8')), manifest = JSON.parse(manifestBody.toString('utf8'));
if (receipt.entries.length !== 10 || manifest.files.length !== 8) throw Error('Unexpected pinned inventory shape');
const entries = [...receipt.entries, {path: receiptRel, bytes: receiptBody.length, sha256: hash(receiptBody), allowedPreviousSha256: []}];
const safe = rel => {
  if (!rel.startsWith('native/') || rel.includes('\\') || path.posix.isAbsolute(rel) || rel.split('/').some(v => v === '' || v === '..')) throw Error('Unsafe receipt path');
  const target = path.resolve(root, rel);
  if (!target.startsWith(root + path.sep)) throw Error('Destination outside workspace');
  return target;
};
// Read and validate every target's allowed old identity before writing any target.
const targets = entries.map(e => {
  const body = checked(path.join(incoming, e.path), e.bytes, e.sha256), target = safe(e.path);
  const before = fs.existsSync(target) ? hash(fs.readFileSync(target)) : null;
  if (before !== null && before !== e.sha256 && !e.allowedPreviousSha256.includes(before)) throw Error('Unapproved old identity: ' + target);
  return {...e, body, target, before};
});
for (const e of manifest.files) {
  safe(e.path);
  const replacing = targets.find(t => t.path === e.path);
  if (!replacing) checked(path.join(root, e.path), e.bytes, e.sha256);
}
for (const t of targets) {
  const now = fs.existsSync(t.target) ? hash(fs.readFileSync(t.target)) : null;
  if (now !== t.before) throw Error('Destination changed after preflight: ' + t.target);
  fs.mkdirSync(path.dirname(t.target), {recursive: true});
  if (now !== t.sha256) fs.writeFileSync(t.target, t.body);
  checked(t.target, t.bytes, t.sha256);
}
for (const e of manifest.files) checked(path.join(root, e.path), e.bytes, e.sha256);
const finalRel = evidence + '/working-mirror-receipt-03.json', finalPath = path.join(root, finalRel);
const output = {
  status: 'EXACT_BYTE_WORKING_MIRROR_VERIFIED_NOT_PRODUCT_ACCEPTANCE',
  sourceRoot: manifest.sourceRoot,
  verifiedAtUTC: new Date().toISOString(),
  archive: {path: archive, bytes: 48175, sha256: '520e57570ca53d598a434f5fdbeaa477d64921b8b46d234671f9a9156acd79b4'},
  boxMirrorInputsSha256: hash(receiptBody),
  sourceInputsSha256: hash(manifestBody),
  targets: targets.map(({body, target, ...e}) => ({...e, localPath: target, verified: true})),
  finalSourceFiles: manifest.files,
  productCommandsExecutedByAuthor: 0,
  originalFreeze01Preserved: true,
  session: '9e20bc49-fddb-490b-a752-c3b11da86ccc'
};
if (fs.existsSync(finalPath)) {
  const old = JSON.parse(fs.readFileSync(finalPath, 'utf8'));
  if (old.sourceInputsSha256 !== output.sourceInputsSha256 || old.archive.sha256 !== output.archive.sha256) throw Error('Immutable final receipt conflicts');
  console.log(JSON.stringify({receiptPath: finalPath, receiptSha256: hash(fs.readFileSync(finalPath)), status: old.status, finalSourceFiles: manifest.files}, null, 2));
} else {
  const b = Buffer.from(JSON.stringify(output, null, 2) + '\n', 'utf8');
  fs.writeFileSync(finalPath, b, {flag: 'wx'});
  checked(finalPath, b.length, hash(b));
  console.log(JSON.stringify({receiptPath: finalPath, receiptBytes: b.length, receiptSha256: hash(b), ...output}, null, 2));
}
