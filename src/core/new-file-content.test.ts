// @vitest-environment node
/**
 * 「新建文件时该写什么」这条出口的测试（ADR 0048）。
 *
 * 钉三件事：
 * 1. **`.mindmap` 落的是一张真的导图**，不是一段长得像导图的文本 ——
 *    用的是解析器自己的眼睛（`parseMindmap`），不是对这个模块的断言。
 * 2. **往返是规范形式**（`canonical`）：新文件打开后一个字没改就保存不产生 diff。
 * 3. **别的类型仍然是空文件**：这一条是「未知类型不需要额外解释」的保证。
 *
 * 另外钉住内核安全的两个条件（id 不含 `"` / `\`、id 不重复）——
 * 判据抄自 `modules/mindmap/adapter.ts` 的 `UNSAFE_ID_CHARS`，
 * **不能 import 那一层**（core 不该知道内核，硬约束 1）。
 *
 * `.drawio` 那一组在下面：判据是「新文件一打开就该画得出来、且不顶一条警告」——
 * 所以用 `core/drawio.ts` 自己的 `missingStencilLibs` 量，不是照抄模板里的字面量。
 */

import { describe, expect, it } from 'vitest';

import { missingStencilLibs } from './drawio.js';
import { parseMindmap } from './mindmap/parse.js';
import {
  MINDMAP_DEFAULT_LAYOUT,
  MINDMAP_FORMAT,
  MINDMAP_VERSION,
  type MindmapNode,
} from './mindmap/types.js';
import { newFileContent } from './new-file-content.js';

/** 与 `modules/mindmap/adapter.ts` 的 `UNSAFE_ID_CHARS` 同一条判据。 */
function collectIds(text: string): string[] {
  const document = parseMindmap(text).document;
  if (document === null) throw new Error('模板没能解析成导图');
  const ids: string[] = [];
  const walk = (node: MindmapNode): void => {
    ids.push(node.id);
    node.children.forEach(walk);
  };
  walk(document.root);
  return ids;
}

describe('新建文件的初始内容', () => {
  it('★ .mindmap 落的是一张解析器认得的导图，不是长得像的文本', () => {
    const text = newFileContent('C:\\notes\\会议记录.mindmap');

    const parsed = parseMindmap(text);
    expect(parsed.issues).toEqual([]);
    expect(parsed.document).not.toBeNull();
    expect(parsed.document?.version).toBe(MINDMAP_VERSION);
    expect(parsed.document?.layout).toBe(MINDMAP_DEFAULT_LAYOUT);
    // 中心主题就是文件名 —— 用户自己刚起的名字，不是我们编的内容
    expect(parsed.document?.root.topic).toBe('会议记录');
    expect(parsed.document?.root.children).toEqual([]);
    expect(text).toContain(MINDMAP_FORMAT);
  });

  it('★ 模板往返是规范形式：打开后一个字没改就保存不产生 diff', () => {
    expect(parseMindmap(newFileContent('会议.mindmap')).canonical).toBe(true);
  });

  it('★ 模板的 id 对内核是安全的（不含引号与反斜杠，且不重复）', () => {
    const ids = collectIds(newFileContent('会议.mindmap'));
    expect(ids).toHaveLength(1);
    expect(ids[0]).not.toMatch(/["\\]/);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('别的类型仍然是空文件 —— 空文件对任何扩展名都是合法起点', () => {
    expect(newFileContent('C:\\notes\\笔记.md')).toBe('');
    expect(newFileContent('C:\\notes\\a.ts')).toBe('');
    expect(newFileContent('C:\\notes\\没有后缀')).toBe('');
  });

  it('大小写不敏感：.MINDMAP 与 .mindmap 落同一份模板', () => {
    expect(newFileContent('C:\\notes\\会议.MINDMAP')).toBe(newFileContent('C:\\notes\\会议.mindmap'));
  });
});

describe('新建的 .drawio 落的是一张画得出来的图纸', () => {
  it('★ 模板不引用任何未内置的形状库 —— 否则新文件一打开就顶一条警告', () => {
    // 判据用 `core/drawio.ts` 自己的那双眼（`missingStencilLibs`），
    // 不是对本模块的断言：它扫的是 XML 里 `shape=mxgraph.<库名>.` 这些写法。
    // 随包只内置 bpmn 与 flowchart（ADR 0027 / P5-24），模板刻意只用内建形状。
    expect(missingStencilLibs(newFileContent('C:\\notes\\流程.drawio'))).toEqual([]);
  });

  it('★ 模板是一个 mxfile，且图形真的在里面（不是空壳）', () => {
    const xml = newFileContent('C:\\notes\\流程.drawio');

    expect(xml).toContain('<mxfile');
    // 一个 `<mxCell` 都没有 = 空图纸，那正是新建模板要避免的状态：
    // 用户按了「新建图纸」，看到的是一句「这份图纸还是空的」。
    expect(xml).toContain('<mxCell');
    expect(xml).not.toContain('双击');
  });

  it('别的类型仍然是空文件 —— 加了一个 .drawio 分支，没有把 .xml 也拽进来', () => {
    // `.xml` 刻意不认领（`isDrawioPath` 不收它）：泛型容器，绝大多数 .xml 不是图纸。
    expect(newFileContent('C:\\notes\\配置.xml')).toBe('');
  });

  it('大小写不敏感：.DRAWIO 与 .drawio 落同一份模板', () => {
    expect(newFileContent('C:\\notes\\流程.DRAWIO')).toBe(newFileContent('C:\\notes\\流程.drawio'));
  });
});