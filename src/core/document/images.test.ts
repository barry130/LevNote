// @vitest-environment node
/**
 * 图片的读取与落盘 —— 两个文档级动作。
 *
 * ## 这一组测试真正在钉的四条不变量
 *
 * 1. **绝不覆盖已存在的文件。** 撞名时换一个盐重试，重试到底就如实失败。
 *    覆盖的后果是毁掉用户磁盘上的一份文件，而界面上完全看不出来。
 * 2. **落盘字节数不对要说出来。** 写入被截断的话，用户会得到一张破图
 *    而没有任何线索 —— 这条断言把「静默的坏文件」变成一个明确的消息。
 * 3. **读不到就是 `null`，不抛。** 调用链的尽头是 NodeView 的一次渲染，
 *    抛出去只会变成一个没人处理的 Promise。
 * 4. **失败一定带一句给用户看的话。** `reason` 为空串等于静默失败，
 *    那是本项目的红线。
 */

import { describe, expect, it } from 'vitest';

import { textOf } from '../i18n/index.js';
import type {
  BinaryWriteOutcome,
  BomKind,
  DirEntry,
  DocumentProvider,
  FileStat,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from './types.js';
import { randomSalt, readImageAsUrl, writePastedImage } from './images.js';
import type { PasteImageResult } from './images.js';

/**
 * `reason` 现在交出去的是**还没翻的 key + 参数**（ADR 0047：文案快照那一族的
 * 最后一处 `t()` 结果烤死在了返回值里）。断言那句话之前先翻，
 * 翻的动作留在用例里 —— 生产代码里翻的地方只有渲染那一刻。
 */
function reasonOf(result: PasteImageResult): string {
  if (result.ok) throw new Error('这个用例期待落盘失败；`ok` 为真说明用例本身写错了');
  return textOf(result.reason);
}

/** 一张最小的 PNG 头，足够让 MIME 嗅探认出来。 */
const PNG_HEAD = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

interface FakeFile {
  text: string;
  encoding: string;
  bom: BomKind;
  mtimeMs: number;
  readonly: boolean;
}

/**
 * 内存文件系统。行为刻意贴近 Rust 侧：**目标已存在就拒绝写入**。
 *
 * 这一点必须照抄：如果假盘允许覆盖而真盘拒绝，那么「撞名」这条分支
 * 在测试里永远走不到，等到用户那儿才第一次触发。
 */
class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  readonly files = new Map<string, FakeFile>();
  readonly binaries = new Map<string, ArrayBuffer>();
  readonly writeCalls: string[] = [];
  /** 让写入按这个顺序失败：每一项是一次调用的结果。 */
  writeOutcomes: Array<'ok' | 'conflict' | 'io' | 'short'> = [];
  readThrows = false;
  private clock = 1_000;

  put(path: string, text = 'x'): void {
    this.files.set(path, { text, encoding: 'UTF-8', bom: 'none', mtimeMs: (this.clock += 10), readonly: false });
  }

  putBytes(path: string, bytes: number[]): void {
    this.binaries.set(path, new Uint8Array(bytes).buffer);
  }

  async readTextFile(path: string): Promise<TextFile> {
    const file = this.files.get(path);
    if (!file) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return {
      path,
      text: file.text,
      encoding: file.encoding,
      bom: file.bom,
      lineEnding: 'lf',
      size: file.text.length,
      mtimeMs: file.mtimeMs,
      readonly: file.readonly,
      lossy: false,
    };
  }

  async writeTextFile(request: WriteRequest): Promise<WriteOutcome> {
    this.put(request.path, request.text);
    return { path: request.path, size: request.text.length, mtimeMs: this.clock, encoding: 'UTF-8', bom: 'none', lineEnding: 'lf' };
  }

  async statFile(path: string): Promise<FileStat> {
    return { path, size: 0, mtimeMs: 1, readonly: false, isDir: false };
  }

  async listDir(): Promise<DirEntry[]> {
    return [];
  }

  async sniffEncoding(): Promise<string> {
    return 'UTF-8';
  }

  async readBinaryFile(path: string): Promise<ArrayBuffer> {
    if (this.readThrows) throw { code: 'NOT_FOUND', message: `读不到 ${path}`, detail: null };
    const bytes = this.binaries.get(path);
    if (!bytes) throw { code: 'NOT_FOUND', message: `读不到 ${path}`, detail: null };
    return bytes;
  }

  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    this.writeCalls.push(path);
    const outcome = this.writeOutcomes.shift() ?? 'ok';

    if (outcome === 'conflict') throw { code: 'CONFLICT', message: `已存在：${path}`, detail: null };
    if (outcome === 'io') throw { code: 'IO', message: '磁盘满了', detail: null };
    if (this.binaries.has(path)) throw { code: 'CONFLICT', message: `已存在：${path}`, detail: null };

    // 'short' 模拟「写了一半」：真盘上不会主动发生，但字节数对不上时
    // 调用方必须能发现，所以这里要能造出来
    const stored = outcome === 'short' ? bytes.slice(0, Math.max(0, bytes.byteLength - 1)) : bytes.slice(0);
    this.binaries.set(path, stored);
    return { path, size: stored.byteLength, mtimeMs: (this.clock += 10) };
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'levnote', version: 'test', os: 'win32', arch: 'x64', nowMs: this.clock };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(): Promise<string | null> {
    return null;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(): Promise<void> {
    /* 不需要 */
  }
}

const AT = new Date(2026, 8, 29, 14, 30, 22);

/** 每次调用换一个盐 —— 与真实调用方一致，也让「换名字重试」能被观察到。 */
function salts(...values: string[]): () => string {
  let index = 0;
  return () => values[index++] ?? 'ffff';
}

describe('readImageAsUrl', () => {
  it('把相对路径解析成本地绝对路径再读，拼成 data URI', async () => {
    const provider = new FakeProvider();
    // 键是清一色的反斜杠：Markdown 里写的是 `./读书笔记.assets/x.png`（正斜杠），
    // 而 `joinPath` 会把相对部分的分隔符统一成文档路径的风格（反斜杠）。
    // 这条统一是必需的，否则「写进去的路径」与「回读时算出来的路径」
    // 会是两个不同的字符串 —— 见 `core/path.test.ts` 里那条逐字符相等的用例
    provider.putBytes('C:\\笔记\\读书笔记.assets\\x.png', [...PNG_HEAD]);

    const url = await readImageAsUrl(provider, 'C:\\笔记\\读书笔记.md', './读书笔记.assets/x.png');
    expect(url).toBe(`data:image/png;base64,${btoa(String.fromCharCode(...PNG_HEAD))}`);
  });

  it('**按文件头判类型**，不信扩展名 —— 改过名的图也要能显示', async () => {
    const provider = new FakeProvider();
    // 内容其实是 PNG，名字却写着 .jpg
    provider.putBytes('C:\\n\\a.assets\\lying.jpg', [...PNG_HEAD]);

    const url = await readImageAsUrl(provider, 'C:\\n\\a.md', './a.assets/lying.jpg');
    expect(url).toContain('data:image/png;');
  });

  it('外部地址一律不读盘', async () => {
    const provider = new FakeProvider();
    provider.readThrows = true;
    for (const src of ['https://x/a.png', 'data:image/png;base64,AA', 'blob:abc', 'ftp://x/a.png']) {
      expect(await readImageAsUrl(provider, 'C:\\n\\a.md', src)).toBeNull();
    }
  });

  it('空的 src 不读盘', async () => {
    const provider = new FakeProvider();
    provider.readThrows = true;
    expect(await readImageAsUrl(provider, 'C:\\n\\a.md', '')).toBeNull();
  });

  it('文档没有目录信息时返回 null —— 相对路径无从谈起', async () => {
    const provider = new FakeProvider();
    provider.putBytes('x.png', [...PNG_HEAD]);
    expect(await readImageAsUrl(provider, 'demo.md', './x.png')).toBeNull();
  });

  it('**读不到就返回 null，不抛** —— 抛出去会变成一个没人处理的 Promise', async () => {
    const provider = new FakeProvider();
    provider.readThrows = true;
    await expect(readImageAsUrl(provider, 'C:\\n\\a.md', './missing.png')).resolves.toBeNull();
  });

  it('绝对路径的 src 直接用，不往文档目录上拼', async () => {
    const provider = new FakeProvider();
    provider.putBytes('D:\\pics\\x.png', [...PNG_HEAD]);

    const url = await readImageAsUrl(provider, 'C:\\n\\a.md', 'D:\\pics\\x.png');
    expect(url).toContain('data:image/png;');
  });
});

describe('writePastedImage', () => {
  const bytes = new Uint8Array([...PNG_HEAD, 0x01, 0x02]).buffer;

  it('写到文档旁边的资源目录，返回**写进文档的相对路径**', async () => {
    const provider = new FakeProvider();

    const result = await writePastedImage(provider, 'C:\\笔记\\读书笔记.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('a3f2'),
    });

    expect(result).toEqual({
      ok: true,
      src: './读书笔记.assets/image-20260929-143022-a3f2.png',
      path: 'C:\\笔记\\读书笔记.assets\\image-20260929-143022-a3f2.png',
      size: bytes.byteLength,
    });
    // 真的写进去了，而且字节一致
    expect(provider.binaries.get('C:\\笔记\\读书笔记.assets\\image-20260929-143022-a3f2.png')).toBeDefined();
  });

  it('**绝不覆盖**：撞名时换一个盐重试', async () => {
    const provider = new FakeProvider();
    provider.writeOutcomes = ['conflict'];

    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('aaaa', 'bbbb'),
    });

    expect(result.ok && result.src).toBe('./a.assets/image-20260929-143022-bbbb.png');
    expect(provider.writeCalls).toHaveLength(2);
  });

  it('重试到底就如实失败，并且说清原因', async () => {
    const provider = new FakeProvider();
    provider.writeOutcomes = ['conflict', 'conflict', 'conflict'];

    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('aaaa', 'bbbb', 'cccc', 'dddd'),
    });

    expect(result.ok).toBe(false);
    expect(reasonOf(result)).toContain('已存在');
    // 只试 3 次，不无限重试
    expect(provider.writeCalls).toHaveLength(3);
  });

  it('**不是「已存在」的错误不重试** —— 磁盘满了换名字也一样', async () => {
    const provider = new FakeProvider();
    provider.writeOutcomes = ['io'];

    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('aaaa'),
    });

    expect(result.ok).toBe(false);
    expect(reasonOf(result)).toContain('磁盘满了');
    expect(provider.writeCalls).toHaveLength(1);
  });

  it('**写入被截断时要说出来** —— 不然用户只会看到一张破图', async () => {
    const provider = new FakeProvider();
    provider.writeOutcomes = ['short'];

    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('aaaa'),
    });

    expect(result.ok).toBe(false);
    expect(reasonOf(result)).toContain('可能已损坏');
    // 消息里要带上路径，用户才知道该去删哪个文件
    expect(reasonOf(result)).toContain('image-20260929-143022-aaaa.png');
  });

  it('不认识的格式直接拒绝，连写都不写', async () => {
    const provider = new FakeProvider();

    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/tiff',
      at: AT,
      salt: salts('aaaa'),
    });

    expect(result.ok).toBe(false);
    expect(reasonOf(result)).toContain('不认识');
    expect(provider.writeCalls).toHaveLength(0);
  });

  it('MIME 是空串时也说得出话（不显示一个空的括号）', async () => {
    const provider = new FakeProvider();
    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: '  ',
      at: AT,
      salt: salts('aaaa'),
    });
    expect(reasonOf(result)).toContain('未知格式');
  });

  it('文档没保存过（没有目录信息）时拒绝，并告诉用户先保存', async () => {
    const provider = new FakeProvider();

    const result = await writePastedImage(provider, 'demo.md', {
      bytes,
      mimeType: 'image/png',
      at: AT,
      salt: salts('aaaa'),
    });

    expect(result.ok).toBe(false);
    expect(reasonOf(result)).toContain('保存');
    expect(provider.writeCalls).toHaveLength(0);
  });

  it('带参数的 MIME（`image/svg+xml;charset=utf-8`）也认', async () => {
    const provider = new FakeProvider();
    const result = await writePastedImage(provider, 'C:\\n\\a.md', {
      bytes,
      mimeType: 'image/svg+xml;charset=utf-8',
      at: AT,
      salt: salts('aaaa'),
    });
    expect(result.ok && result.src).toBe('./a.assets/image-20260929-143022-aaaa.svg');
  });
});

describe('randomSalt', () => {
  it('一定是 4 位十六进制 —— 长度不够会让文件名少一段', () => {
    for (let i = 0; i < 200; i += 1) {
      expect(randomSalt()).toMatch(/^[0-9a-f]{4}$/);
    }
  });

  it('不是常量（200 次里至少出现 100 个不同的值）', () => {
    const seen = new Set<string>();
    for (let i = 0; i < 200; i += 1) seen.add(randomSalt());
    expect(seen.size).toBeGreaterThan(100);
  });
});
