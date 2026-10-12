/**
 * 图片导出编排的测试。
 *
 * 这里能钉住的与钉不住的（ADR 0050 的测试边界）：
 *
 * - 钉得住：先问存到哪再画（取消就不白画）、建议文件名带对后缀、
 *   字节**原样**落盘、内核的警告一条不少地拼进通报、
 *   每一种失败都如实报、绝不弹假成功。
 * - 钉不住：画出来的**像素**对不对。内核画图靠布局引擎与位图编码，
 *   测试环境没有（PNG 转换甚至根本跑不完），那部分靠 `pnpm run tauri dev`
 *   里的人工冒烟，写在这里只会得到一条假绿的测试。
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
import type { EditorImageExport } from '../editor/types.js';
import { msg } from '../i18n/index.js';
import { exportImage, type ExportImageResult } from './export-image.js';

/** 收窄到「成功」分支；失败时报出实际分支而不是一句类型错。 */
function saved(result: ExportImageResult): Extract<ExportImageResult, { status: 'saved' }> {
  if (result.status !== 'saved') throw new Error(`期望导出成功，实际是 ${result.status}`);
  return result;
}

class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  /** pickSavePath 收到的建议名，一次一条。 */
  readonly saveSuggestions: string[] = [];
  /** writeBinaryFile 落盘的每一次调用。 */
  readonly binaryWrites: { path: string; bytes: ArrayBuffer }[] = [];
  saveTarget: string | null = 'C:\\out\\导图.png';
  saveThrows = false;
  binaryWriteThrows = false;
  /** 伪造「少写了几个字节」的坏磁盘。 */
  shortBy = 0;
  private readonly bytes = new Map<string, ArrayBuffer>();

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
    if (this.binaryWriteThrows) throw new Error('磁盘满了');
    const stored = bytes.slice(0);
    this.binaryWrites.push({ path, bytes: stored });
    this.bytes.set(path, stored);
    return { path, size: bytes.byteLength - this.shortBy, mtimeMs: 2 };
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'levnote', version: 'test', os: 'win32', arch: 'x64', nowMs: 1 };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(suggestedName?: string): Promise<string | null> {
    if (suggestedName !== undefined) this.saveSuggestions.push(suggestedName);
    if (this.saveThrows) throw new Error('对话框炸了');
    return this.saveTarget;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(): Promise<void> {
    /* 这个套件不需要 */
  }
}

/** 造一个会记着自己被调了几次的 draw（无参 —— 格式在命令层就定死了）。 */
function fakeDraw(overrides: Partial<EditorImageExport> = {}) {
  let calls = 0;
  const image: EditorImageExport = {
    blob: new Blob([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], { type: 'image/png' }),
    warnings: [],
    ...overrides,
  };
  return {
    get calls() {
      return calls;
    },
    draw: () => {
      calls += 1;
      return Promise.resolve<EditorImageExport | null>(image);
    },
  };
}

describe('exportImage', () => {
  it('成功：建议名带格式后缀，字节原样落盘，内核的实话一条不少', async () => {
    const provider = new FakeProvider();
    const kernel = fakeDraw({
      warnings: [msg('ui.mindmap.exportPngImagesExternal', { n: 2 })],
    });

    const result = await exportImage({ title: '架构图', format: 'png', draw: kernel.draw, provider });

    expect(kernel.calls).toBe(1);
    expect(provider.saveSuggestions).toEqual(['架构图.png']);
    expect(saved(result).target).toBe('C:\\out\\导图.png');
    expect(provider.binaryWrites).toHaveLength(1);
    expect(provider.binaryWrites[0]!.path).toBe('C:\\out\\导图.png');
    // 字节原样：内核给什么就写什么，编排层不碰内容
    expect(Array.from(new Uint8Array(provider.binaryWrites[0]!.bytes))).toEqual([0x89, 0x50, 0x4e, 0x47]);
    // 通报 = 成功那一句 + 内核的警告，按序以「；」相接
    expect(saved(result).message).toBe('已导出 导图.png；有 2 张节点图片引用的是外部文件，PNG 里画不出来（转换时不去读它们）');
  });

  it('SVG 走同一条路，只是建议名换后缀', async () => {
    const provider = new FakeProvider();
    provider.saveTarget = 'C:\\out\\导图.svg';
    const { draw } = fakeDraw({
      blob: new Blob(['<svg …/>'], { type: 'image/svg+xml' }),
      warnings: [msg('ui.mindmap.exportSvgImagesExternal', { n: 1 })],
    });

    const result = await exportImage({ title: '导图', format: 'svg', draw, provider });

    expect(provider.saveSuggestions).toEqual(['导图.svg']);
    expect(saved(result).message).toContain('SVG 里保留的是原引用');
  });

  it('先问存到哪：用户取消就不画，也不写，也不弹假成功', async () => {
    const provider = new FakeProvider();
    provider.saveTarget = null;
    const kernel = fakeDraw();

    const result = await exportImage({ title: 'a', format: 'png', draw: kernel.draw, provider });

    expect(result.status).toBe('cancelled');
    expect(kernel.calls).toBe(0);
    expect(provider.binaryWrites).toHaveLength(0);
  });

  it('保存对话框弹不出来时如实报失败', async () => {
    const provider = new FakeProvider();
    provider.saveThrows = true;

    const result = await exportImage({ title: 'a', format: 'png', draw: fakeDraw().draw, provider });

    expect(result.status === 'failed' && result.message).toContain('无法弹出保存对话框');
  });

  it('内核画图时抛出，失败里带上原因', async () => {
    const provider = new FakeProvider();

    const result = await exportImage({
      title: 'a',
      format: 'png',
      draw: () => Promise.reject(new Error('位图编码失败')),
      provider,
    });

    expect(result.status === 'failed' && result.message).toContain('生成图片失败：位图编码失败');
    expect(provider.binaryWrites).toHaveLength(0);
  });

  it('内核说画不出来（null）时按失败报，不假装导出了一张空图', async () => {
    const provider = new FakeProvider();

    const result = await exportImage({
      title: 'a',
      format: 'png',
      draw: () => Promise.resolve(null),
      provider,
    });

    expect(result.status === 'failed' && result.message).toContain('内核没能把图画出来');
    expect(provider.binaryWrites).toHaveLength(0);
  });

  it('写盘失败时如实报失败', async () => {
    const provider = new FakeProvider();
    provider.binaryWriteThrows = true;

    const result = await exportImage({ title: 'a', format: 'png', draw: fakeDraw().draw, provider });

    expect(result.status === 'failed' && result.message).toContain('写入失败');
    expect(result.status === 'failed' && result.message).toContain('磁盘满了');
  });

  it('半成品必须说破：少写了字节就按失败报，还要带上删掉重试的路径', async () => {
    const provider = new FakeProvider();
    provider.shortBy = 2;

    const result = await exportImage({ title: 'a', format: 'png', draw: fakeDraw().draw, provider });

    expect(result.status === 'failed' && result.message).toContain('2 / 4 字节');
    expect(result.status === 'failed' && result.message).toContain('C:\\out\\导图.png');
  });

  it('内核没有实话时，通报就只有成功那一句', async () => {
    const provider = new FakeProvider();
    provider.saveTarget = 'C:\\out\\干净.png';

    const result = await exportImage({ title: '干净', format: 'png', draw: fakeDraw().draw, provider });

    expect(saved(result).message).toBe('已导出 干净.png');
  });
});
