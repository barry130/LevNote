// @vitest-environment jsdom
/**
 * 导出动作的测试。
 *
 * 这里测的是「动作的**结果**」——落盘内容、用户取消、失败时的说法。
 * 「导出的 HTML 长什么样」由 `inline-html.test.ts` 与 `App.test.tsx` 钉住，
 * 不在这里重复。
 */

import { describe, expect, it } from 'vitest';

import type {
  DirEntry,
  DocumentProvider,
  FileStat,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from '../document/types.js';
import { exportHtml, suggestedExportName, type ExportHtmlResult } from './export-document.js';

/**
 * 收窄到「成功」分支。
 *
 * 比 `expect(result.status).toBe('saved')` 之后再访问 `result.message` 好：
 * 后者在类型上仍然可能是 cancelled 分支，TS 会报错；用断言函数既能收窄类型，
 * 又在失败时给出比「类型不对」更有意义的报错。
 */
function saved(result: ExportHtmlResult): Extract<ExportHtmlResult, { status: 'saved' }> {
  if (result.status !== 'saved') throw new Error(`期望导出成功，实际是 ${result.status}`);
  return result;
}

class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  readonly writes: WriteRequest[] = [];
  saveTarget: string | null = 'C:\\out\\结果.html';
  saveThrows = false;
  writeThrows = false;
  private readonly bytes = new Map<string, ArrayBuffer>();

  putBytes(path: string, bytes: number[]): void {
    this.bytes.set(path, new Uint8Array(bytes).buffer as ArrayBuffer);
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
    if (this.writeThrows) throw new Error('磁盘满了');
    this.writes.push(request);
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

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'qingmo', version: 'test', os: 'win32', arch: 'x64', nowMs: 1 };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(): Promise<string | null> {
    if (this.saveThrows) throw new Error('对话框炸了');
    return this.saveTarget;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(): Promise<void> {
    /* 不需要 */
  }
}

describe('suggestedExportName', () => {
  it('用不带扩展名的文件名加 .html', () => {
    expect(suggestedExportName('C:\\notes\\架构说明.md')).toBe('架构说明.html');
  });
});

describe('exportHtml', () => {
  it('写出完整 HTML，并如实通报做了什么', async () => {
    const provider = new FakeProvider();
    provider.putBytes('C:\\notes\\img\\a.png', [0x89, 0x50, 0x4e, 0x47]);

    const result = await exportHtml({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml: '<h1>标题</h1><pre data-language="js"><code>const a = 1;</code></pre><img src="img/a.png">',
      fallbackText: '',
      provider,
    });

    expect(saved(result).target).toBe('C:\\out\\结果.html');
    expect(provider.writes).toHaveLength(1);

    const written = provider.writes[0]!;
    expect(written.path).toBe('C:\\out\\结果.html');
    expect(written.text.startsWith('<!DOCTYPE html>')).toBe(true);
    expect(written.text).toContain('<h1');
    // 代码高亮与图片内联都在导出侧补上
    expect(written.text).toContain('hljs-keyword');
    expect(written.text).toContain('data:image/png;base64,');
    // 版式内联，不引用外部资源
    expect(written.text).not.toContain('<link');

    // 导出文件固定 UTF-8 无 BOM + LF：跟随源文档的 GBK / CRLF 会让它在别人机器上乱码
    expect(written.encoding).toBe('UTF-8');
    expect(written.bom).toBe('none');
    expect(written.lineEnding).toBe('lf');

    expect(saved(result).message).toContain('1 张图片已内联');
    expect(saved(result).message).toContain('1 个代码块已着色');
  });

  it('用户取消保存对话框时什么都不写，也不弹假成功', async () => {
    const provider = new FakeProvider();
    provider.saveTarget = null;

    const result = await exportHtml({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml: '<p>正文</p>',
      fallbackText: '',
      provider,
    });

    expect(result.status).toBe('cancelled');
    expect(provider.writes).toHaveLength(0);
  });

  it('内核给不出结构时退化为 <pre>，并在通报里说明', async () => {
    const provider = new FakeProvider();
    const result = await exportHtml({
      path: 'C:\\notes\\note.txt',
      title: 'note',
      bodyHtml: null,
      fallbackText: 'if (a < b) {\n  返回值\n}\n',
      provider,
    });

    const html = provider.writes[0]!.text;
    expect(html).toContain('qm-plain');
    // 原文里的尖括号必须转义，否则导出文件被撕坏
    expect(html).toContain('if (a &lt; b) {');
    expect(saved(result).message).toContain('按纯文本呈现');
  });

  it('写入失败时给出失败结果，而不是抛出去', async () => {
    const provider = new FakeProvider();
    provider.writeThrows = true;

    const result = await exportHtml({
      path: 'C:\\n\\a.md',
      title: 'a',
      bodyHtml: '<p>x</p>',
      fallbackText: '',
      provider,
    });

    expect(result.status === 'failed' && result.message).toContain('磁盘满了');
  });

  it('弹不出保存对话框时也如实报失败', async () => {
    const provider = new FakeProvider();
    provider.saveThrows = true;

    const result = await exportHtml({
      path: 'C:\\n\\a.md',
      title: 'a',
      bodyHtml: '<p>x</p>',
      fallbackText: '',
      provider,
    });

    expect(result.status === 'failed' && result.message).toContain('保存对话框');
  });
});
