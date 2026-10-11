/**
 * draw.io 图纸内核的适配器声明（ADR 0048）。
 *
 * 外壳只通过这个对象认识图纸内核 —— 它不知道底下是随包分发的那个
 * `viewer.min.js`，换渲染器只需要换掉 `renderDrawio` 那一层。
 *
 * ## 它与导图内核的差别，都写在取舍里
 *
 * | | `.mindmap` | `.drawio`（本轮） |
 * |---|---|---|
 * | 认不认得这份内容 | `canHandle` + `canHandleReason`（坏 JSON 要否决） | 一律认，交给渲染器如实报失败 |
 * | 源码视图 | `codeAdapter` | `codeAdapter`（同一条依赖，同一个理由） |
 * | 能不能直接编辑 | 能（mind-elixir 画布） | **不能**，面板上写明了 |
 * | 换句话说 | 「画布」 | 「查看器 + XML」 |
 *
 * `sourceView` 用代码内核的依据与 `modules/mindmap/index.ts` 完全一致：
 * `.drawio` 的源码形态就是那份 XML 原文，而轻笺已经有一个 XML/文本编辑器了。
 * 这描述的是一个真实存在的关系（「图纸的源码形态由代码内核呈现」），
 * 不是「代码内核知道图纸」。于是 `.drawio` 立刻拿到视图三态（图纸 / 源码 / 分屏），
 * 外壳只问 `sourceView` 在不在，不问它底下坐着谁。
 *
 * **而这就是本轮给用户的编辑入口**：在源码视图里改 XML，Ctrl+S 落盘，
 * 回到图纸视图立刻重画。不用改 `core/editor/types.ts` 的任何契约 ——
 * 文档的 text 就是 XML，内核交回的也是 XML，与 `.mindmap` 的 JSON 同一套。
 */

import type { EditorAdapter } from '../../core/editor/types.js';
import { isDrawioPath } from '../../core/path.js';
import { codeAdapter } from '../code/index.js';
import DrawioEditor from './DrawioEditor.js';

export const drawioAdapter: EditorAdapter = {
  id: 'drawio',
  label: 'draw.io 图纸',
  matches: isDrawioPath,

  // 刻意**不实现 `canHandle`**，理由写在 DrawioEditor.tsx 的文件头：
  // 查看器自己会把「不是合法的 XML」说出来，比适配层再判一遍更准，
  // 也不必为了它去顶 ADR 0043 那张 kind 表的 5 个上限。

  /**
   * 源码视图 = 代码内核。
   *
   * `.drawio` 也**不能**从 `CODE_EXTENSIONS` 里摘掉 —— 那会让
   * `sourceView: codeAdapter` 指着一个认不得 `.drawio` 的内核，
   * 于是「切到源码视图」打开的仍是一堆没有着色的文本（或者干脆兜底到纯文本）。
   * 代码内核认得 `.drawio` 正是这条依赖能成立的前提。
   */
  sourceView: codeAdapter,

  Component: DrawioEditor,
};