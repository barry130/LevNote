/**
 * Markdown 内核的适配器声明。
 *
 * 外壳只通过这个对象认识 Markdown 内核 —— 它不知道底下是 Milkdown，
 * 换内核（比如将来换成自研的）只需要换掉 `Component`。
 */

import type { EditorAdapter } from '../../core/editor/types.js';
import { isMarkdownPath } from '../../core/path.js';
import { codeAdapter } from '../code/index.js';
import { hasFrontMatter } from './frontmatter.js';
import MarkdownEditor from './MarkdownEditor.js';

export { hasFrontMatter };

export const markdownAdapter: EditorAdapter = {
  id: 'markdown',
  label: 'Markdown',
  matches: isMarkdownPath,
  // 扩展名对得上，但内容是 front matter 文档 —— 交给纯文本内核，别毁人家元数据
  canHandle: (_path, text) => !hasFrontMatter(text),
  /**
   * 源码视图就是代码内核。
   *
   * 这是一个**模块间的依赖**（markdown → code），刻意保留：它描述的是一个
   * 真实存在的关系 ——「Markdown 文档的源码形态，由代码内核来呈现」。
   * 反过来写（让代码内核去声明「我是 Markdown 的源码视图」）是错的，
   * 因为那是 Markdown 内核的属性，不是代码内核的属性。
   *
   * 方向不会成环：代码内核不认识 Markdown 内核，也不需要认识。
   * 语法包也是现成的 —— `languageForPath('notes.md')` 给的就是 markdown。
   */
  sourceView: codeAdapter,
  Component: MarkdownEditor,
};
