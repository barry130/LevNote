/**
 * Markdown 内核的适配器声明。
 *
 * 外壳只通过这个对象认识 Markdown 内核 —— 它不知道底下是 Milkdown，
 * 换内核（比如将来换成自研的）只需要换掉 `Component`。
 */

import type { EditorAdapter } from '../../core/editor/types.js';
import { hasFrontMatter } from './frontmatter.js';
import MarkdownEditor from './MarkdownEditor.js';

export { hasFrontMatter };

/**
 * 认定为 Markdown 的扩展名。
 *
 * 刻意**不含 `.mdx`**：MDX 里的 JSX 对 Milkdown 是语法错误，
 * 用 Markdown 内核打开会把 `import` / `<Component/>` 当成段落文本，
 * 一保存就把用户的 MDX 改坏了。`.mdx` 会落到纯文本兜底，至少不破坏内容。
 */
const MARKDOWN_EXTENSIONS = new Set(['.md', '.markdown', '.mdown', '.mkd', '.mkdn', '.mdwn', '.mdtxt', '.mdtext']);

function extensionOf(path: string): string {
  const name = path.split(/[\\/]/).pop() ?? path;
  const dot = name.lastIndexOf('.');
  return dot <= 0 ? '' : name.slice(dot).toLowerCase();
}


export const markdownAdapter: EditorAdapter = {
  id: 'markdown',
  label: 'Markdown',
  matches: (path) => MARKDOWN_EXTENSIONS.has(extensionOf(path)),
  // 扩展名对得上，但内容是 front matter 文档 —— 交给纯文本内核，别毁人家元数据
  canHandle: (_path, text) => !hasFrontMatter(text),
  Component: MarkdownEditor,
};
