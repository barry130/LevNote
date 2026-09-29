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
  BinaryWriteOutcome,
  DirEntry,
  DocumentProvider,
  FileStat,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from '../document/types.js';
import type { ExportReport } from './document.js';
import {
  describeExport,
  exportHtml,
  suggestedExportName,
  type ExportHtmlResult,
} from './export-document.js';

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

  /** 这个套件不测图片落盘；接口要求存在，写进同一个字节盘。 */
  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    if (this.bytes.has(path)) throw new Error('CONFLICT');
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

  /**
   * 带公式的文档：渲染结果必须真的进了导出文件。
   *
   * **这里不断言字体数量。** 真实实现走 `?url` + `fetch`，而 `fetch` 需要同源
   * 环境 —— vitest 的 node 环境里那个 URL 是 `/node_modules/…`，没有 base，
   * `fetch` 直接抛 `TypeError: Failed to parse URL`，于是这份测试里的字体
   * 一律「没能内联」。那是**测试环境的限制**，不是产品行为，钉住它只会
   * 把一个假象写进测试。字体那条链路由 `katex-css.test.ts`（注入 io）
   * 与 `inline-html.test.ts`（注入 io 走完整管线）覆盖。
   */
  it('带公式时渲染结果进文件，提示里报出渲染数', async () => {
    const provider = new FakeProvider();

    const result = await exportHtml({
      path: 'C:\\notes\\note.md',
      title: 'note',
      bodyHtml:
        '<p>前 <span class="qm-math qm-math-inline" data-type="math_inline" data-tex="x^2">$x^2$</span> 后</p>',
      fallbackText: '',
      provider,
    });

    const html = provider.writes[0]!.text;
    expect(html).toContain('katex-mathml');
    expect(html).not.toContain('$x^2$');
    expect(saved(result).message).toContain('1 个公式已渲染');
  });
});

// ---------------------------------------------------------------------------
// 导出提示
// ---------------------------------------------------------------------------

/** 一份什么都不做的报告。各条分支靠覆盖字段来构造。 */
function report(overrides: Partial<ExportReport> = {}): ExportReport {
  return {
    inlinedImages: 0,
    missingImages: 0,
    remoteImages: 0,
    highlightedBlocks: 0,
    renderedMath: 0,
    failedMath: 0,
    firstMathError: null,
    mathFonts: 0,
    missingMathFonts: 0,
    mathStyleBytes: 0,
    ...overrides,
  };
}

function summary(overrides: Partial<ExportReport> = {}, plainFallback = false): string {
  return describeExport({ target: 'C:\\out\\我的笔记.html', report: report(overrides), plainFallback });
}

describe('describeExport', () => {
  it('什么都没有时只说导出到了哪 —— 不编形容词', () => {
    expect(summary()).toBe('已导出 我的笔记.html');
  });

  it('路径只取文件名，不把整条路径甩给用户', () => {
    expect(summary()).not.toContain('C:\\');
  });

  it('内容里处理过什么就说什么', () => {
    expect(
      summary({ highlightedBlocks: 3, inlinedImages: 2, missingImages: 1, remoteImages: 1 }),
    ).toBe(
      '已导出 我的笔记.html；3 个代码块已着色；2 张图片已内联；1 张图片读不到，只保留了原链接；1 张图片来自网络，打开时需要联网',
    );
  });

  /**
   * 顺序是**刻意**的：先说自己要的那件事成了（导出到了哪），
   * 再说内容里有什么被处理过，最后说有什么**没做到**。
   */
  it('「没做到」的部分排在最后', () => {
    const message = summary({ highlightedBlocks: 1, missingImages: 1, missingMathFonts: 20 });
    expect(message.indexOf('已着色')).toBeLessThan(message.indexOf('读不到'));
    expect(message.indexOf('读不到')).toBeLessThan(message.indexOf('没能内联'));
  });

  describe('公式', () => {
    it('渲染成功时报出数量', () => {
      expect(summary({ renderedMath: 2 })).toBe('已导出 我的笔记.html；2 个公式已渲染');
    });

    /**
     * 字体的体积是用户该知道的**代价**，不是我们该藏的细节：一份带公式的
     * 文档会比同样长度的文档大 300 KB 上下，不说清楚的话用户只会觉得
     * 「这个导出是不是有 bug」。
     */
    it('字体内联成功时报出文件数与体积', () => {
      const message = summary({ renderedMath: 1, mathFonts: 20, mathStyleBytes: 300_000 });
      expect(message).toContain('公式字体已内联（20 个文件，约 293 KB）');
    });

    it('体积按 KB 四舍五入，不吐一串小数', () => {
      expect(summary({ mathFonts: 1, mathStyleBytes: 1023 })).toContain('约 1 KB');
    });

    /**
     * 这两条是**绝不静默**的红线：字体没内联意味着发出去的文件在别人机器上
     * 会掉字形，公式没渲染出来意味着那段内容只剩源码。都必须说出来。
     */
    it('渲染失败时给出第一条原因 —— 用户要知道回去改哪儿', () => {
      const message = summary({
        failedMath: 1,
        firstMathError: 'Undefined control sequence: \\foo（第 1 个字符起）',
      });
      expect(message).toContain('1 个公式渲染失败，只保留了源码');
      expect(message).toContain('Undefined control sequence: \\foo（第 1 个字符起）');
    });

    it('没有原因时不留下一个空括号', () => {
      const message = summary({ failedMath: 1 });
      expect(message).toContain('1 个公式渲染失败，只保留了源码');
      expect(message).not.toContain('（）');
    });

    it('字体没内联时必须说出来', () => {
      expect(summary({ missingMathFonts: 20 })).toContain(
        '20 个公式字体没能内联，导出的公式可能用系统字体显示',
      );
    });

    it('有公式但字体全缺时，两条消息同时出现', () => {
      const message = summary({ renderedMath: 3, missingMathFonts: 20 });
      expect(message).toContain('3 个公式已渲染');
      expect(message).toContain('20 个公式字体没能内联');
    });
  });

  it('纯文本兜底时说明', () => {
    expect(summary({}, true)).toBe('已导出 我的笔记.html；当前内核没有富文本结构，已按纯文本呈现');
  });
});
