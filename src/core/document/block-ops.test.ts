import { describe, expect, it } from 'vitest';

import { buildReferenceFence, extractNameCandidates, extractToReference, flattenReference } from './block-ops.js';
import type { BlockOpsDeps } from './block-ops.js';

/** 一套可微调的假依赖：文件系统是内存 Map，文档操作是记录下来的调用。 */
function makeDeps(overrides: Partial<BlockOpsDeps> = {}): BlockOpsDeps & {
  files: Map<string, string>;
  replacedSelections: string[];
  replacedFences: string[];
} {
  const files = new Map<string, string>();
  const replacedSelections: string[] = [];
  const replacedFences: string[] = [];
  return {
    files,
    replacedSelections,
    replacedFences,
    docPath: 'C:\\笔记\\main.md',
    selectedMarkdown: () => '被选中的正文。\n',
    replaceSelectionWith: (markdown) => replacedSelections.push(markdown),
    enclosingFence: () => ({ info: 'doc-block', content: 'id: a\nkind: markdown\nmode: reference\nsrc: notes/x.md\n---' }),
    replaceEnclosingFenceWith: (markdown) => replacedFences.push(markdown),
    readTextFile: async (path) => (files.has(path) ? { text: files.get(path) ?? '' } : null),
    writeTextFile: async (path, text) => {
      files.set(path, text);
    },
    notify: async () => undefined,
    ...overrides,
  };
}

describe('提取为引用（P6 第 28 项 L3 的外壳半边）', () => {
  it('把选区写成新文件，选区替换成指向它的 doc-block 引用', async () => {
    const deps = makeDeps();
    await extractToReference(deps);

    // 新文件落在文档同目录，内容就是选区原文
    expect(deps.files.has('C:\\笔记\\被选中的正文。.md')).toBe(false); // 标题不存在 → 回退名
    const written = [...deps.files.entries()];
    expect(written.length).toBe(1);
    expect(written[0]![0].startsWith('C:\\笔记\\')).toBe(true);
    expect(written[0]![0].endsWith('.md')).toBe(true);
    expect(written[0]![1]).toBe('被选中的正文。\n');

    // 引用围栏替换进文档，src 是正斜杠、相对路径
    expect(deps.replacedSelections.length).toBe(1);
    expect(deps.replacedSelections[0]).toContain('```doc-block');
    expect(deps.replacedSelections[0]).toContain(`src: ${written[0]![0].split('\\').pop()}`);
    expect(deps.replacedSelections[0]).not.toContain('\\');
  });

  it('名字被占用时自动加序号 —— 绝不覆盖已有文件', async () => {
    const deps = makeDeps();
    // 预先占住前两个候选名（excerpt.md、excerpt-2.md）
    deps.files.set('C:\\笔记\\excerpt.md', '别人的');
    deps.files.set('C:\\笔记\\excerpt-2.md', '别人的');
    await extractToReference(deps);

    const written = [...deps.files.keys()];
    expect(written).toContain('C:\\笔记\\excerpt-3.md');
    // 被占用的文件内容原样没动
    expect(deps.files.get('C:\\笔记\\excerpt.md')).toBe('别人的');
  });

  it('空选区说清楚原因，不做任何事', async () => {
    const notices: string[] = [];
    const deps = makeDeps({
      selectedMarkdown: () => null,
      notify: async (message) => {
        notices.push(message);
      },
    });
    await extractToReference(deps);
    expect(notices.length).toBe(1);
    expect(deps.replacedSelections.length).toBe(0);
    expect(deps.files.size).toBe(0);
  });

  it('候选名来自选区的第一个标题', () => {
    const candidates = extractNameCandidates('## 部署 手册/注意\n\n正文\n');
    expect(candidates.first).toBe('部署-手册-注意.md');
  });
});

describe('拉平引用（P6 第 28 项 L3 的外壳半边）', () => {
  it('reference 模式：读出引用文件，替换整个围栏', async () => {
    const deps = makeDeps();
    deps.files.set('C:\\笔记\\notes\\x.md', '# 被引用的文档\n\n正文。\n');
    await flattenReference(deps);

    expect(deps.replacedFences.length).toBe(1);
    expect(deps.replacedFences[0]).toContain('# 被引用的文档');
    // 被引用的文件还在
    expect(deps.files.get('C:\\笔记\\notes\\x.md')).toBe('# 被引用的文档\n\n正文。\n');
  });

  it('图表 / 导图引用拒绝拉平 —— 内容进正文是降级不是转换', async () => {
    const notices: string[] = [];
    const deps = makeDeps({
      enclosingFence: () => ({ info: 'doc-block', content: 'id: d\nkind: mindmap\nmode: reference\nsrc: m.mindmap\n---' }),
      notify: async (message) => {
        notices.push(message);
      },
    });
    await flattenReference(deps);
    expect(notices.length).toBe(1);
    expect(deps.replacedFences.length).toBe(0);
  });

  it('读不到引用文件时说清楚是哪一个', async () => {
    const notices: string[] = [];
    const deps = makeDeps({
      notify: async (message) => {
        notices.push(message);
      },
    });
    await flattenReference(deps);
    expect(notices[0]).toContain('notes/x.md');
    expect(deps.replacedFences.length).toBe(0);
  });
});

describe('引用围栏的组装', () => {
  it('属性顺序固定、src 是正斜杠、有标题时带上标题（规范 §4）', () => {
    const fence = buildReferenceFence('sub\\note.md', '小节');
    const lines = fence.split('\n');
    expect(lines[0]).toBe('```doc-block');
    expect(lines[1]).toMatch(/^id: /);
    expect(lines[2]).toBe('kind: markdown');
    expect(lines[3]).toBe('mode: reference');
    expect(lines[4]).toBe('src: sub/note.md');
    expect(lines[5]).toBe('title: 小节');
    expect(lines[6]).toBe('---');
    expect(lines[7]).toBe('```');
  });

  it('组装出来的围栏能被 parseDocBlock 原样读回（写与读是同一个约定）', async () => {
    const { parseDocBlock } = await import('../docblock.js');
    const fence = buildReferenceFence('note.md', null);
    // parseDocBlock 吃的是围栏内的内容
    const inner = fence.split('\n').slice(1, -1).join('\n');
    const parsed = parseDocBlock(inner);
    expect(parsed.attrs.kind).toBe('markdown');
    expect(parsed.attrs.mode).toBe('reference');
    expect(parsed.attrs.src).toBe('note.md');
    expect(parsed.errors).toEqual([]);
  });
});
