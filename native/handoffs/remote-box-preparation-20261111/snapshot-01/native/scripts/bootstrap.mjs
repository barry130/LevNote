// A project-local toolchain keeps the existing Tauri build and system Go untouched.
import https from 'node:https';
import { createWriteStream } from 'node:fs';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { pipeline } from 'node:stream/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const tools = path.join(root, 'tools');
const filename = 'go1.27.3.windows-amd64.zip';
function request(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 6) return reject(new Error('Too many redirects'));
    const req = https.get(url, response => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        resolve(request(new URL(response.headers.location, url), redirects + 1));
      } else if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`HTTP ${response.statusCode}: ${url}`));
      } else resolve(response);
    });
    req.setTimeout(60000, () => req.destroy(new Error('Download timed out')));
    req.on('error', reject);
  });
}
await mkdir(tools, { recursive: true });
await writeFile(path.join(tools, 'go.mod'), 'module levnote.local/toolchain-cache\n\ngo 1.27.3\n');
const response = await request('https://go.dev/dl/?mode=json&include=all');
let json = '';
for await (const chunk of response) json += chunk;
const release = JSON.parse(json).find(r => r.version === 'go1.27.3');
const archive = release?.files.find(f => f.filename === filename);
if (!archive) throw new Error('The pinned Go release is not in the official release manifest');
const target = path.join(tools, filename);
console.log(`Downloading ${filename} (${Math.round(archive.size / 1048576)} MiB)`);
await pipeline(await request(`https://go.dev/dl/${filename}`), createWriteStream(target));
const digest = createHash('sha256').update(await readFile(target)).digest('hex');
if (digest !== archive.sha256) throw new Error('Go archive SHA256 mismatch; do not extract');
await writeFile(path.join(tools, 'go-toolchain.json'), JSON.stringify({ ...archive, verified: true }, null, 2));
console.log(`Verified SHA256 ${digest}`);
