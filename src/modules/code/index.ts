/**
 * 代码 / 文本内核的适配器声明。
 *
 * ## 它同时承担「Markdown 的备胎」这个角色
 *
 * 注册表在 `canHandle` 否决一个内核后会**继续找下一个候选**（见 `registry.ts`）。
 * 带 YAML front matter 的 `.md` 正是这个机制的目标场景：Markdown 富文本内核
 * 会毁掉元数据，于是它退位；而这里认得 `.md`，接住它 ——
 * 用户拿到的是一份**有语法着色、有行号**的 Markdown 源码，
 * 而不是一个白底 textarea。
 *
 * 这就是「四类内核」架构真正的价值所在：**降级不是掉到最底，而是掉到次优**。
 *
 * ## 为什么扩展名名单是白名单而不是「所有文件」
 *
 * 兜底内核（`plaintext`）承诺「任何文件都能打开」，它是最后一道防线，
 * 必须保持**最简单、最不可能失败**。CodeMirror 的初始化比一个 textarea 复杂得多，
 * 让它去当兜底会把那条承诺的可靠性拉低。
 * 所以这里是白名单：认识的才接，不认识的仍走兜底。
 */

import type { EditorAdapter } from '../../core/editor/types.js';
import { extensionOf, MARKDOWN_EXTENSIONS } from '../../core/path.js';
import CodeEditor from './CodeEditor.js';

/**
 * 认定为「代码 / 文本」的扩展名。
 *
 * 与 `core/platform/desktop.ts` 的 `TEXT_EXTENSIONS`（对话框过滤器）**刻意分开**：
 * 那份是「用户可能想打开的文本类文件」（可以宽），
 * 这份是「交给 CodeMirror 处理」（必须保守，因为它决定用哪个内核）。
 * 两者的目标不同，合在一起会让其中一边被迫将就。
 *
 * 但 Markdown 那一段**必须**与 `core/path.ts` 共用同一份名单。两边不一致
 * 会造成双重落空：Markdown 内核认为「这不是 .md」（于是退位），
 * 代码内核也不认它（于是继续往下掉到纯文本）—— 用户拿到一个白底 textarea，
 * 而两个内核的代码看上去都没写错。
 */
const CODE_EXTENSIONS = new Set([
  // Markdown 源码形态（正常情况走 Markdown 内核，front matter 文件落到这里）
  ...MARKDOWN_EXTENSIONS,
  // 配置与数据
  '.json', '.jsonc', '.yml', '.yaml', '.toml', '.ini', '.cfg', '.conf', '.env', '.csv', '.tsv',
  // Web
  '.js', '.mjs', '.cjs', '.jsx', '.ts', '.mts', '.cts', '.tsx', '.css', '.scss', '.less', '.html', '.htm', '.vue', '.svelte',
  // 标记与图形
  '.xml', '.svg', '.drawio',
  // 系统与脚本
  '.sh', '.bash', '.zsh', '.fish', '.bat', '.cmd', '.ps1',
  // 语言
  '.py', '.pyw', '.rb', '.go', '.rs', '.java', '.kt', '.swift', '.c', '.h', '.cc', '.cpp', '.hpp', '.cs', '.php', '.lua', '.r', '.pl',
  // 查询与日志
  '.sql', '.log', '.diff', '.patch',
  // 纯文本
  '.txt', '.text',
]);

export const codeAdapter: EditorAdapter = {
  id: 'code',
  label: '代码 / 文本',
  matches: (path) => CODE_EXTENSIONS.has(extensionOf(path)),
  Component: CodeEditor,
};
