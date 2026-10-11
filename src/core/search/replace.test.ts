// @vitest-environment node
import { describe, expect, it } from 'vitest';

import {
  replaceAcrossFiles,
  describeReplaceOutcome,
  renderReplaceReport,
  type ReplaceIo,
} from './replace.js';
import { msg, setLocale } from '../i18n/index.js';
import type { TextFile, WriteRequest } from '../document/types.js';

/** 假 IO：文本盘 + 记录每一次写请求（编码 / BOM 是否回传是重点断言）。 */
class FakeIo implements ReplaceIo {
  readonly files = new Map<string, TextFile>();
  readonly writes: WriteRequest[] = [];
  failRead = new Set<string>();
  failWrite = new Set<string>();
  readonlyPermissions = new Set<string>();

  put(path: string, text: string, overrides: Partial<TextFile> = {}): void {
    this.files.set(path, {
      path,
      text,
      encoding: 'UTF-8',
      bom: 'none',
      lineEnding: 'lf',
      size: text.length,
      mtimeMs: 1000,
      readonly: false,
      lossy: false,
      ...overrides,
    });
  }

  async readTextFile(path: string): Promise<TextFile> {
    if (this.failRead.has(path)) throw new Error('读不出来');
    const file = this.files.get(path);
    if (file === undefined) throw new Error(`没有这个文件：${path}`);
    return file;
  }

  async writeTextFile(request: WriteRequest): Promise<unknown> {
    if (this.failWrite.has(request.path)) throw new Error('磁盘拒绝写入');
    if (this.readonlyPermissions.has(request.path)) {
      throw { code: 'PERMISSION_DENIED', message: '文件是只读的' };
    }
    this.writes.push(request);
    const previous = this.files.get(request.path);
    this.files.set(request.path, { ...(previous as TextFile), text: request.text });
    return { path: request.path };
  }
}

const LITERAL = { query: 'foo', caseSensitive: false, wholeWord: false, regex: false };

describe('replaceAcrossFiles（A3 跨文件替换）', () => {
  it('替换、计数、写回的内容都对', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\a.md', 'foo bar foo');
    io.put('C:\\w\\b.md', 'no hit here');

    const outcome = await replaceAcrossFiles(['C:\\w\\a.md', 'C:\\w\\b.md'], LITERAL, 'qux', new Set(), io);

    expect(outcome.matchesReplaced).toBe(2);
    expect(outcome.filesReplaced).toBe(1);
    expect(io.files.get('C:\\w\\a.md')?.text).toBe('qux bar qux');
    // 没有命中的文件一个字节都不写
    expect(io.writes).toHaveLength(1);
  });

  it('编码、BOM 原样回传；混排行尾不传目标行尾', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\gbk.txt', '旧文本', { encoding: 'GBK', bom: 'utf-8', lineEnding: 'mixed' });

    await replaceAcrossFiles(['C:\\w\\gbk.txt'], { query: '旧', caseSensitive: false, wholeWord: false, regex: false }, '新', new Set(), io);

    expect(io.writes).toHaveLength(1);
    const request = io.writes[0]!;
    expect(request.encoding).toBe('GBK');
    expect(request.bom).toBe('utf-8');
    // mixed 不该被擅自统一掉
    expect(request.lineEnding).toBeUndefined();
  });

  it('打开且有未保存改动的文件整个跳过，磁盘上一个字节都不碰', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\dirty.md', 'foo');
    io.put('C:\\w\\clean.md', 'foo');

    const outcome = await replaceAcrossFiles(
      ['C:\\w\\dirty.md', 'C:\\w\\clean.md'],
      LITERAL,
      'qux',
      new Set(['C:\\w\\dirty.md']),
      io,
    );

    // 原因是 key 不是句子 —— 换语言时它跟着变（ADR 0047）
    expect(outcome.skipped).toEqual([{ path: 'C:\\w\\dirty.md', reason: msg('core.replace.skipDirty') }]);
    expect(outcome.filesReplaced).toBe(1);
    expect(io.writes.map((request) => request.path)).toEqual(['C:\\w\\clean.md']);
  });

  it('读不到与写不进（只读）逐个记进 failed，不中断其余文件', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\ok.md', 'foo');
    io.put('C:\\w\\locked.md', 'foo');
    io.put('C:\\w\\ghost.md', 'foo');
    io.failRead.add('C:\\w\\ghost.md');
    io.readonlyPermissions.add('C:\\w\\locked.md');

    const outcome = await replaceAcrossFiles(['C:\\w\\ok.md', 'C:\\w\\locked.md', 'C:\\w\\ghost.md'], LITERAL, 'qux', new Set(), io);

    expect(outcome.filesReplaced).toBe(1);
    expect(outcome.failed).toHaveLength(2);
    expect(outcome.failed.map((item) => item.path).sort()).toEqual(['C:\\w\\ghost.md', 'C:\\w\\locked.md']);
    expect(io.files.get('C:\\w\\ok.md')?.text).toBe('qux');
  });

  it('正则替换支持 $1 展开；非正则时 $ 是字面量', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\re.md', 'ab123 cd456');

    await replaceAcrossFiles(
      ['C:\\w\\re.md'],
      { query: '([a-z]+)(\\d+)', caseSensitive: false, wholeWord: false, regex: true },
      '$2$1',
      new Set(),
      io,
    );
    expect(io.files.get('C:\\w\\re.md')?.text).toBe('123ab 456cd');

    io.put('C:\\w\\lit.md', '价格$99');
    await replaceAcrossFiles(
      ['C:\\w\\lit.md'],
      { query: '价格$', caseSensitive: false, wholeWord: false, regex: false },
      '￥',
      new Set(),
      io,
    );
    expect(io.files.get('C:\\w\\lit.md')?.text).toBe('￥99');
  });

  it('大小写与全词的开关与检索同一套语义', async () => {
    const io = new FakeIo();
    io.put('C:\\w\\case.md', 'Foo foo Foobar');

    await replaceAcrossFiles(
      ['C:\\w\\case.md'],
      { query: 'foo', caseSensitive: true, wholeWord: false, regex: false },
      'bar',
      new Set(),
      io,
    );
    // 只有小写的 foo 命中；Foobar 的 foo 前缀不算（区分大小写下 Foo 与 Foobar 都不命中）
    expect(io.files.get('C:\\w\\case.md')?.text).toBe('Foo bar Foobar');
  });

  /**
   * 汇报交出去的是**结构**（ADR 0047），句子由 `renderReplaceReport` 在渲染的
   * 那一刻翻出来。下面两个用例钉住两件事：跳过与失败都要说出来，
   * 以及**同一份结构**在两种语言下渲染出不同的句子。
   */
  const mixed = describeReplaceOutcome({
    filesReplaced: 2,
    matchesReplaced: 5,
    skipped: [{ path: 'a', reason: msg('core.replace.skipDirty') }],
    failed: [{ path: 'b', reason: msg('core.replace.readFailed', { detail: 'EACCES' }) }],
  });

  it('结果描述把跳过与失败说出来（交出去的是结构，不是拼好的句子）', () => {
    expect(mixed.headline.map((line) => line.key)).toEqual([
      'core.replace.doneSummary',
      'core.replace.skippedN',
      'core.replace.failedN',
    ]);
    // 问题行**先失败后跳过** —— 失败比跳过更需要被看见
    expect(mixed.problems.map((problem) => problem.path)).toEqual(['b', 'a']);

    setLocale('zh-CN');
    expect(renderReplaceReport(mixed)).toBe(
      '替换了 5 处（2 个文件），跳过 1 个，失败 1 个\n- b：读不到这个文件（EACCES）\n- a：在打开的标签页里有未保存的改动',
    );
  });

  it('切语言后同一份汇报渲染出另一种语言 —— 不重建、不重新查（ADR 0047）', () => {
    setLocale('zh-CN');
    const zh = renderReplaceReport(mixed);
    setLocale('en');
    const en = renderReplaceReport(mixed);
    setLocale('zh-CN');

    expect(en).not.toBe(zh);
    // 抛物的原话当参数走，不翻译 —— 翻了就成撒谎
    expect(en).toContain("Couldn't read this file (EACCES)");
    // 连分隔符都来自 catalog：英文界面里不该画出一个中文逗号
    expect(en).toContain(', ');
    expect(en).not.toContain('，');
    expect(en).toContain('- b: ');
  });

  it('无事可做时是一整句，不混进逗号串', () => {
    const idle = describeReplaceOutcome({ filesReplaced: 0, matchesReplaced: 0, skipped: [], failed: [] });
    setLocale('zh-CN');
    expect(renderReplaceReport(idle)).toBe('没有需要替换的内容。');
  });
});
