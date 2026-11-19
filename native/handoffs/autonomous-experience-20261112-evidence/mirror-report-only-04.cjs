'use strict';
// Opaque report delivery only. Never parses/reconstructs or executes product source.
const fs = require('node:fs'), path = require('node:path'), crypto = require('node:crypto');
const root = path.resolve('F:/AIproject/LevNote');
const evidence = 'native/handoffs/autonomous-experience-20261112-evidence';
const incoming = path.join(root, evidence, 'incoming-report04');
const digest = b => crypto.createHash('sha256').update(b).digest('hex');
const checked = (p, bytes, sha256) => {
  const b = fs.readFileSync(p);
  if (b.length !== bytes || digest(b) !== sha256) throw Error('Pinned identity mismatch: ' + p);
  return b;
};
const safe = rel => {
  if (!rel.startsWith('native/handoffs/') || rel.includes('\\') || path.posix.isAbsolute(rel) || rel.split('/').some(v => v === '' || v === '..')) throw Error('Unsafe report target');
  const p = path.resolve(root, rel);
  if (!p.startsWith(root + path.sep)) throw Error('Report target outside workspace');
  return p;
};
const archive = path.join(incoming, 'report-mirror-04.tar.gz');
checked(archive, 8020, '6d15080e757588869916b9db473ea5b62038212c4971b89a29966a9ad5526a3d');
const manifestRel = evidence + '/report-only-04/report-inputs.json';
const manifestBody = checked(path.join(incoming, manifestRel), 728, '81661a734c44bf8c83846fca38c03f872b75bc79064206b5b4830b220f6e1d44');
const manifest = JSON.parse(manifestBody.toString('utf8'));
const sourceBody = checked(path.join(root, evidence, 'freeze-03/source-inputs.json'), 7681, 'ccc1461d5b26a34fe4c7788b4fff7dfa9d40cd11812bdd1ae2745f28586ae71b');
const sources = JSON.parse(sourceBody.toString('utf8'));
if (sources.files.length !== 8 || manifest.authorProductExecutions !== 0 || manifest.bytes !== 19105) throw Error('Unexpected pinned manifest');
for (const e of sources.files) checked(path.join(root, e.path), e.bytes, e.sha256);
// The original03 report remains immutable in the earlier exact-byte archive staging.
checked(path.join(root, evidence, 'incoming-03/native/handoffs/autonomous-experience-20261112.md'), 14089, 'ee31ec3a5cfa5be3b83422c6848830fb407a109d7d73ba50f6188b97027c123b');
const entries = [
  { path: 'native/handoffs/autonomous-experience-20261112.md', bytes: 19105, sha256: '5f16c8c368b043475bc3fab36217bbbd72de8748ce0751918d704033edfe55aa', previous: ['ee31ec3a5cfa5be3b83422c6848830fb407a109d7d73ba50f6188b97027c123b'] },
  { path: evidence + '/report-only-04/report-04.md', bytes: 19105, sha256: '5f16c8c368b043475bc3fab36217bbbd72de8748ce0751918d704033edfe55aa', previous: [] },
  { path: manifestRel, bytes: 728, sha256: '81661a734c44bf8c83846fca38c03f872b75bc79064206b5b4830b220f6e1d44', previous: [] }
];
// All targets are validated before the first write; changed old identities are rejected.
const targets = entries.map(e => {
  const body = checked(path.join(incoming, e.path), e.bytes, e.sha256), p = safe(e.path);
  const before = fs.existsSync(p) ? digest(fs.readFileSync(p)) : null;
  if (before !== null && before !== e.sha256 && !e.previous.includes(before)) throw Error('Unapproved previous report identity: ' + p);
  return { ...e, body, p, before };
});
for (const t of targets) {
  const now = fs.existsSync(t.p) ? digest(fs.readFileSync(t.p)) : null;
  if (now !== t.before) throw Error('Report changed after preflight: ' + t.p);
  fs.mkdirSync(path.dirname(t.p), {recursive: true});
  if (now !== t.sha256) fs.writeFileSync(t.p, t.body);
  checked(t.p, t.bytes, t.sha256);
}
for (const e of sources.files) checked(path.join(root, e.path), e.bytes, e.sha256);
const receiptPath = safe(evidence + '/report-only-04/working-mirror-receipt-04.json');
if (fs.existsSync(receiptPath)) throw Error('Create-only report receipt exists');
const result = {
  status: 'REPORT_ONLY_EXACT_BYTE_MIRROR_SOURCE03_UNCHANGED_NOT_PRODUCT_ACCEPTANCE',
  verifiedAtUTC: new Date().toISOString(), sourceRoot: manifest.sourceRoot,
  boxReportInputsSha256: digest(manifestBody), source03InputsSha256: digest(sourceBody),
  archive: {path: archive, bytes: 8020, sha256: '6d15080e757588869916b9db473ea5b62038212c4971b89a29966a9ad5526a3d'},
  targets: targets.map(({body, p, ...e}) => ({...e, localPath: p, verified: true})),
  finalSourceFiles: sources.files, original03ReportPreserved: true,
  productCommandsExecutedByAuthor: 0, session: '9e20bc49-fddb-490b-a752-c3b11da86ccc'
};
const receiptBytes = Buffer.from(JSON.stringify(result, null, 2) + '\n', 'utf8');
fs.writeFileSync(receiptPath, receiptBytes, {flag: 'wx'});
checked(receiptPath, receiptBytes.length, digest(receiptBytes));
console.log(JSON.stringify({receiptPath, receiptBytes: receiptBytes.length, receiptSha256: digest(receiptBytes), ...result}, null, 2));
