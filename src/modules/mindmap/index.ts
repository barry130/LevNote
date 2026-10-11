/**
 * 思维导图内核的适配器声明。
 *
 * 外壳只通过这个对象认识导图内核 —— 它不知道底下是 mind-elixir（5.15.1，
 * 精确锁版本，理由见 ADR 0007），换内核只需要换掉 `Component` 与 `adapter.ts`。
 *
 * ## 它有两个入口，一个出口
 *
 * 入口一：`.mindmap` 扩展名（`core/path.ts` 的 `isMindmapPath`）。
 * 入口二：**坏掉的 `.mindmap`** —— 见下面 `canHandle` 的说明。
 *
 * 出口是纯文本兜底（`EDITOR_FALLBACK`）：任何一份 `.mindmap` 最终都能被打开，
 * 最差也是「一份能看能改的 JSON 源码」。
 */

import type { EditorAdapter, KernelRejectReason } from '../../core/editor/types.js';
import { isMindmapPath } from '../../core/path.js';
import { codeAdapter } from '../code/index.js';
import { describeMindmapVeto } from './adapter.js';
import MindmapEditor from './MindmapEditor.js';

export const mindmapAdapter: EditorAdapter = {
  id: 'mindmap',
  label: '思维导图',
  matches: isMindmapPath,

  /**
   * 拿到内容之后的第二道确认。
   *
   * 三种情况会**否决**自己，让注册表继续往下找（最终落到代码内核或纯文本）：
   *
   * 1. **不是合法 JSON / 缺 `root` / `format` 不对** —— 那不是一份能画的图。
   *    退到代码内核后用户看到的是一份有 JSON 着色的源码，能直接改；
   *    而如果这里硬接下来，用户会看到一个空白的导图画布，连内容在哪都找不到。
   * 2. **节点 id 里有引号或反斜杠** —— 内核的 `findEle` 是
   *    `querySelector('[data-nodeid="me' + id + '"]')`，那些字符会让选择器语法错误。
   *    理由与「为什么不在适配层里换掉 id」写在 `adapter.ts` 的 `checkKernelSafety` 上。
   * 3. **节点 id 重复** —— `querySelector` 只返回第一个匹配，于是拖第二个节点
   *    动的是第一个，而界面上没有任何提示。
   *
   * 三条都属于「宁可降级，不可静默出错」：降级是用户看得见的（源码视图），
   * 而出错是看不见的。
   *
   * **理由由下面那个 `canHandleReason` 说出来**（ADR 0043）。两者都走
   * `describeMindmapVeto`，所以「判定的结果」与「说出来的理由」不可能分叉。
   */
  canHandle: (_path, text) => describeMindmapVeto(text) === null,

  /**
   * 否决时到底是因为哪一条（ADR 0043）。
   *
   * 没有它，外壳只能对**所有**降级说同一句话 —— 而写死的那句是 front matter 的，
   * 于是「id 重复的 `.mindmap` 被退到代码内核」时，横幅上会出现一句
   * 与眼前文件毫无关系的解释。那比沉默更糟：用户会拿它去改文件。
   */
  canHandleReason: (_path, text): KernelRejectReason | null => describeMindmapVeto(text),

  /**
   * 源码视图 = 代码内核。
   *
   * `.mindmap` 的「源码形态」就是它的 JSON 原文，而轻笺已经有一个 JSON 编辑器了。
   * 与 Markdown 内核那边是**同一条依赖**（markdown → code），理由也一样：
   * 这描述的是一个真实存在的关系（「导图的源码形态由代码内核呈现」），
   * 而不是「代码内核知道导图」。
   *
   * 于是 `.mindmap` 自动获得 M10 的视图三态（富文本 / 源码 / 分屏）——
   * 外壳只问 `sourceView` 在不在，不问它底下坐着谁。
   */
  sourceView: codeAdapter,

  Component: MindmapEditor,
};
