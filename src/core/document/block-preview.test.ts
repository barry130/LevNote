// @vitest-environment node
/**
 * 引用摘要的计算（P6 第 28 项 L2，ADR 0036 决策 3）。
 *
 * 摘要是外壳的活（文件在外壳的领域里），所以这里只注入假的读文件依赖 ——
 * 钉住的是**每种 kind 算出什么结构**与**失败以数据返回**这两件事；
 * 「画得对不对」在内核的 `docblock-card.test.ts` 里钉。
 */

import { describe, expect, it } from 'vitest';

import { buildDocBlockPreview, type BlockPreviewDeps } from './block-preview.js';

function depsWith(files: Record<string, string>): BlockPreviewDeps {
  return {
    readTextFile: async (path) => {
      const text = files[path];
      return text === undefined ? null : { text };
    },
  };
}

describe('buildDocBlockPreview', () => {
  it('markdown：标题树（剥掉行内标记）+ 字数', async () => {
    const deps = depsWith({
      'C:\\n\\detail.md': '# 部署手册\n\n先装依赖，再跑 `npm run build`。\n\n## 前置要求\n\n## 步骤\n',
    });
    const preview = await buildDocBlockPreview(deps, 'C:\\n\\main.md', { src: 'detail.md', kind: 'markdown' });
    expect(preview).toEqual({
      kind: 'markdown',
      wordCount: expect.any(Number),
      headings: [
        { level: 1, text: '部署手册' },
        { level: 2, text: '前置要求' },
        { level: 2, text: '步骤' },
      ],
    });
    if (preview.kind === 'markdown') expect(preview.wordCount).toBeGreaterThan(0);
  });

  it('markdown：标题多于 8 条时截断 —— 摘要只负责「认出文件」', async () => {
    const body = Array.from({ length: 12 }, (_, i) => `## 第 ${i + 1} 节\n`).join('\n');
    const deps = depsWith({ '/n/x.md': `# 开头\n\n${body}` });
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'x.md', kind: 'markdown' });
    if (preview.kind !== 'markdown') throw new Error(`kind 应是 markdown，得到 ${preview.kind}`);
    expect(preview.headings).toHaveLength(8);
  });

  it('text：行数 + 首 40 行片段，截断时如实说明', async () => {
    const lines = Array.from({ length: 60 }, (_, i) => `line ${i + 1}`);
    const deps = depsWith({ '/n/data.txt': lines.join('\n') });
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'data.txt', kind: 'text' });
    if (preview.kind !== 'text') throw new Error(`kind 应是 text，得到 ${preview.kind}`);
    expect(preview.lineCount).toBe(60);
    expect(preview.snippet).toContain('line 1');
    expect(preview.snippet).toContain('line 40');
    expect(preview.snippet).not.toContain('line 41');
    expect(preview.snippet).toContain('共 60 行');
  });

  it('mindmap：根主题 + 节点数', async () => {
    const mindmap = JSON.stringify({
      format: 'qingmo.mindmap',
      version: 1,
      root: { id: 'r', topic: '根', children: [{ id: 'a', topic: '甲', children: [] }] },
    });
    const deps = depsWith({ '/n/map.mindmap': mindmap });
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'map.mindmap', kind: 'mindmap' });
    expect(preview).toEqual({ kind: 'mindmap', nodeCount: 2, rootTitle: '根' });
  });

  it('解析不了的相对路径（宿主文档没有目录信息）以 error 返回，不抛', async () => {
    const deps = depsWith({});
    const preview = await buildDocBlockPreview(deps, 'main.md', { src: 'x.md', kind: 'markdown' });
    expect(preview.kind).toBe('error');
  });

  it('读不到文件以 error 返回，不抛 —— 失败原因要画得出来', async () => {
    const deps = depsWith({});
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'ghost.md', kind: 'markdown' });
    expect(preview).toEqual({ kind: 'error', reason: '读不到 ghost.md' });
  });

  it('mindmap 内容坏了以 error 返回 —— 兜底语义与内核一致', async () => {
    const deps = depsWith({ '/n/map.mindmap': '不是 JSON' });
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'map.mindmap', kind: 'mindmap' });
    expect(preview.kind).toBe('error');
  });

  it('不认识的 kind 按 text 兜底 —— 卡片能画出来的 kind 才会走到这里', async () => {
    const deps = depsWith({ '/n/a.txt': '只有一行' });
    const preview = await buildDocBlockPreview(deps, '/n/main.md', { src: 'a.txt', kind: 'spreadsheet' });
    expect(preview.kind).toBe('text');
  });
});
