// @vitest-environment jsdom
/**
 * 打印 / 导出 PDF 动作的测试（P3 第 16 项，ADR 0030）。
 *
 * 钉的是「动作的结果」：交给打印窗口的是不是**与 HTML 导出同一份渲染**、
 * 失败时的说法、如实通报的缺项。「打印窗口本身」在 Rust 侧（`print.rs`），
 * 这一层只认注入的 `printHtml` 能力 —— 与 provider 的注入同一条理由。
 */

import { describe, expect, it } from 'vitest';

import type {
  BinaryWriteOutcome,
  DirEntry,
  DocumentProvider,
  FileStat,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from '../document/types.js';
import { printDocument, type PrintDocumentResult } from './print-document.js';

function opened(result: PrintDocumentResult): Extract<PrintDocumentResult, { status: 'opened' }> {
  if (result.status !== 'opened') throw new Error(`期望打印窗口打开，实际是 ${result.status}`);
  return result;
}

class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  private readonly bytes = new Map<string, ArrayBuffer>();

  putBytes(path: string, bytes: number[]): void {
    this.bytes.set(path, new Uint8Array(bytes).buffer);
  }

  async readTextFile(path: string): Promise<TextFile> {
    return {
      path,
      text: '',
      encoding: 'UTF-8',
      bom: 'none',
      lineEnding: 'lf',
      size: 0,
      mtimeMs: 1,
      readonly: false,
      lossy: false,
    };
  }

  async writeTextFile(request: WriteRequest): Promise<WriteOutcome> {
    return { path: request.path, size: request.text.length, mtimeMs: 2, encoding: 'UTF-8', bom: 'none', lineEnding: 'lf' };
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
    const bytes = this.bytes.get(path);
    if (!bytes) throw new Error('ENOENT');
    return bytes;
  }

  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    this.bytes.set(path, bytes.slice(0));
    return { path, size: bytes.byteLength, mtimeMs: 2 };
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'qingmo', version: 'test', os: 'win32', arch: 'x64', nowMs: 1 };
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

describe('printDocument', () => {
  it('交给打印窗口的是与 HTML 导出同一份自包含渲染', async () => {
    const provider = new FakeProvider();
    provider.putBytes('C:\\notes\\img\\a.png', [0x89, 0x50, 0x4e, 0x47]);
    const printed: Array<{ html: string; title: string }> = [];

    const result = await printDocument({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml: '<h1>标题</h1><pre data-language="js"><code>const a = 1;</code></pre><img src="img/a.png">',
      fallbackText: '',
      provider,
      printHtml: async (html, title) => {
        printed.push({ html, title });
      },
    });

    expect(printed).toHaveLength(1);
    expect(printed[0]?.title).toBe('note');
    const html = printed[0]?.html ?? '';
    // 渲染与 HTML 导出同源：正文、代码着色、图片内联、版式全在同一份里
    expect(html).toContain('<h1');
    expect(html).toContain('hljs-keyword');
    expect(html).toContain('data:image/png;base64,');
    expect(html).not.toContain('<link');
    expect(opened(result).message).toContain('另存为 PDF');
  });

  it('富文本内核不在时如实退化为原文呈现，不伪造排版', async () => {
    const provider = new FakeProvider();
    const printed: string[] = [];

    const result = await printDocument({
      path: 'C:\\notes\\raw.txt',
      title: 'raw',
      bodyHtml: null,
      fallbackText: '第一行 < 里面的尖括号是内容',
      provider,
      printHtml: async (html) => {
        printed.push(html);
      },
    });

    expect(result.status).toBe('opened');
    const html = printed[0] ?? '';
    expect(html).toContain('第一行 &lt; 里面的尖括号是内容');
  });

  it('缺项如实通报 —— 读不到的图片保留原链接并说出来', async () => {
    const provider = new FakeProvider(); // 没放任何图片字节
    const printed: string[] = [];

    const result = await printDocument({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml: '<p>正文</p><img src="img/丢了.png">',
      fallbackText: '',
      provider,
      printHtml: async (html) => {
        printed.push(html);
      },
    });

    expect(result.status).toBe('opened');
    expect(result.message).toContain('1 张图片读不到');
    // 原链接保留在 HTML 里，打印时可见为破图，但「哪里坏了」是明说的
    expect(printed[0]).toContain('img/丢了.png');
  });

  it('打印窗口打开失败如实报失败，不弹假成功', async () => {
    const provider = new FakeProvider();

    const result = await printDocument({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml: '<p>正文</p>',
      fallbackText: '',
      provider,
      printHtml: () => Promise.reject(new Error('打不开打印窗口：窗口创建失败')),
    });

    expect(result.status).toBe('failed');
    expect(result.message).toContain('窗口创建失败');
  });
});
