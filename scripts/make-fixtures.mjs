/**
 * 生成测试夹具：各种编码 / BOM / 行尾的真实文件。
 *
 * 这些样本必须落成**真实字节**，不能用字符串字面量写 ——
 * 编码探测的 bug 只有在真实字节上才会暴露。
 * GBK / UTF-16 的字节是手工构造的（Node 的 TextEncoder 只会 UTF-8，也不该为它装依赖）。
 *
 * 用法：node scripts/make-fixtures.mjs
 */

import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const outDir = join(root, 'tests', 'fixtures');

/** 'D6 D0' -> Uint8Array */
const hex = (spec) => Uint8Array.from(spec.trim().split(/\s+/).map((byte) => parseInt(byte, 16)));

const utf8 = (text) => new TextEncoder().encode(text);

/** 把文本编成 UTF-16，附带 BOM */
const utf16 = (text, little) => {
  const body = new Uint8Array(text.length * 2);
  const view = new DataView(body.buffer);
  for (let i = 0; i < text.length; i += 1) view.setUint16(i * 2, text.charCodeAt(i), little);
  const bom = little ? [0xff, 0xfe] : [0xfe, 0xff];
  return Uint8Array.from([...bom, ...body]);
};

const concat = (...parts) => {
  const total = parts.reduce((sum, part) => sum + part.length, 0);
  const out = new Uint8Array(total);
  let offset = 0;
  for (const part of parts) {
    out.set(part, offset);
    offset += part.length;
  }
  return out;
};

const DOC_BLOCK_SAMPLE = [
  '# 格式与互操作 样本',
  '',
  '这份文件用来验证 doc-block 的往返：读进来再写回去，未改动的块应逐字节不变。',
  '',
  '```doc-block',
  'id: arch',
  'kind: diagram',
  'mode: reference',
  'src: diagrams/arch.drawio',
  'title: 整体架构',
  '---',
  '```',
  '',
  '中间夹一段正文。',
  '',
  '````doc-block',
  'id: snippet-py',
  'kind: code',
  'mode: inline',
  '---',
  '```python',
  'print("hello")',
  '```',
  '````',
  '',
  '末尾再来一个思维导图引用：',
  '',
  '```doc-block',
  'id: roadmap',
  'kind: mindmap',
  'mode: reference',
  'src: maps/roadmap.mindmap',
  '---',
  '```',
  '',
].join('\n');

const fixtures = [
  {
    name: 'plain-utf8.md',
    why: '最常见的情况：UTF-8 无 BOM，LF 行尾。作为基线。',
    bytes: utf8(DOC_BLOCK_SAMPLE),
  },
  {
    name: 'bom-utf8.md',
    why: '带 UTF-8 BOM。验证 BOM 被剥离、且保存时被原样写回（很多 Windows 工具依赖它）。',
    bytes: concat(utf8('\uFEFF'), utf8(DOC_BLOCK_SAMPLE)),
  },
  {
    name: 'crlf-gbk.md',
    why: 'GBK 编码 + CRLF 行尾。验证无 BOM 时的中文编码探测，以及保存不改行尾。',
    bytes: concat(
      hex('D6 D0 CE C4 B2 E2 CA D4 0D 0A'),
      hex('B5 DA B6 FE D0 D0 0D 0A'),
      hex('A1 B0 D2 FD BA C5 A1 B1 0D 0A'),
    ),
  },
  {
    name: 'utf16le.txt',
    why: 'UTF-16LE + BOM。这类文件在 Windows 上常见，且极易被误判成 UTF-8 而整篇乱码。',
    bytes: utf16('第一行\r\n第二行\r\n', true),
  },
  {
    name: 'utf16be.txt',
    why: 'UTF-16BE + BOM。验证大小端都被正确识别。',
    bytes: utf16('first\r\nsecond\r\n', false),
  },
  {
    name: 'mixed-endings.txt',
    why: 'LF 与 CRLF 混排。验证探测结果为 mixed，且一次保存不会擅自统一整篇文件。',
    bytes: utf8('a\nb\r\nc\nd\r\n'),
  },
  {
    name: 'empty.txt',
    why: '空文件。边界情况：不该崩、不该被当成二进制。',
    bytes: new Uint8Array(0),
  },
  {
    name: 'no-trailing-newline.md',
    why: '结尾没有换行符。验证写回时不擅自补一个换行（会让 diff 多出一行）。',
    bytes: utf8('# 没有结尾换行'),
  },
];

await mkdir(outDir, { recursive: true });

for (const fixture of fixtures) {
  await writeFile(join(outDir, fixture.name), fixture.bytes);
}

const manifest = fixtures.map((fixture) => ({ file: fixture.name, bytes: fixture.bytes.length, why: fixture.why }));
await writeFile(join(outDir, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');

console.log(`已生成 ${fixtures.length} 个夹具到 tests/fixtures/：`);
for (const fixture of fixtures) {
  console.log(`  ${fixture.name.padEnd(24)} ${String(fixture.bytes.length).padStart(5)} 字节`);
}
