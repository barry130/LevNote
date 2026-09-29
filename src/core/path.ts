/**
 * 路径的纯字符串处理。
 *
 * 刻意**不用 `node:path`**：这份代码要同时跑在 WebView（没有 Node 内置模块）与
 * vitest（有）里，而 Node 的 `path` 是平台相关的 —— 在 Linux 上跑测试时
 * `path.basename('C:\\a\\b.md')` 会返回整串。我们要的是「按用户机器上真实的样子
 * 处理 Windows 与 POSIX 两种风格」，所以自己按分隔符判断。
 *
 * 这些函数**不做文件系统访问**，也不做规范化（不解析 `..`、不合并重复分隔符）：
 * 路径规范化属于写入层（Rust 侧的原子写入会处理），这里只负责显示与拼接。
 */

/** 路径分隔符：出现反斜杠就按 Windows 风格处理，否则按 POSIX。 */
export function separatorOf(path: string): '\\' | '/' {
  return path.includes('\\') ? '\\' : '/';
}

export function basename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

/** 去掉扩展名的文件名，用作导出文档标题、标签页默认名。 */
export function titleOf(path: string): string {
  const name = basename(path);
  const dot = name.lastIndexOf('.');
  return dot <= 0 ? name : name.slice(0, dot);
}

/**
 * 扩展名（含点，小写）。没有扩展名时返回空串。
 *
 * `dot <= 0` 而不是 `dot < 0`：`.gitignore` 这种「整个名字就是扩展名」的文件
 * 不该被当成 `.gitignore` 扩展名，它是一个没有主名的隐藏文件。
 */
export function extensionOf(path: string): string {
  const name = basename(path);
  const dot = name.lastIndexOf('.');
  return dot <= 0 ? '' : name.slice(dot).toLowerCase();
}

/**
 * 认定为 Markdown 的扩展名（含点，小写）。
 *
 * ## 为什么这份名单在 `core` 而不是某个模块里
 *
 * 有**四个**地方需要同一个答案：
 * 1. Markdown 内核 —— 决定富文本视图归谁；
 * 2. 代码内核 —— `.md` 的**源码视图**要有 markdown 语法着色；
 * 3. 外壳 —— 决定哪些文档显示大纲面板；
 * 4. 将来 —— 导出、引用解析之类。
 *
 * 抄四份的下场是「给 `.mdx` 加了一处、忘了另外三处」，而这类不一致
 * 表现为「同一个文件在不同功能里被当成两种东西」，极难排查。
 * 放在 `core` 之后，任何一处要改名单都必须改这一个地方。
 *
 * ## 为什么刻意**不含 `.mdx`**
 *
 * MDX 里的 JSX 对 Milkdown 是语法错误，用 Markdown 内核打开会把
 * `import` / `<Component/>` 当成段落文本，一保存就把用户的 MDX 改坏了。
 * `.mdx` 会落到纯文本兜底，至少不破坏内容。
 */
export const MARKDOWN_EXTENSIONS: ReadonlySet<string> = new Set([
  '.md',
  '.markdown',
  '.mdown',
  '.mkd',
  '.mkdn',
  '.mdwn',
  '.mdtxt',
  '.mdtext',
]);

/** 路径是否属于 Markdown 家族。大小写不敏感（`extensionOf` 已统一小写）。 */
export function isMarkdownPath(path: string): boolean {
  return MARKDOWN_EXTENSIONS.has(extensionOf(path));
}

/**
 * 取所在目录。
 *
 * 返回 `null` 表示「拿不到目录」，也就是**不能猜**：光秃秃的文件名
 * （浏览器预览虚拟盘里的 `demo.md`）没有目录信息，硬拼出来的路径一定是错的。
 * 拿不到时调用方（导出内联、相对引用解析）应如实降级，而不是读一个猜出来的路径。
 */
export function dirname(path: string): string | null {
  const separator = separatorOf(path);
  const at = path.lastIndexOf(separator);

  // 常规情况：C:\notes\a.md → C:\notes；/home/u/a.md → /home/u
  if (at > 0) {
    const head = path.slice(0, at);
    // Windows 的 `C:\a.md` 会被切出 `C:`，而 `C:` 表示「C 盘的当前目录」，
    // 不是根目录 —— 必须补回分隔符才是 `C:\`
    return /^[A-Za-z]:$/.test(head) ? head + separator : head;
  }

  // 根目录下的 /a.md 或 \a.md
  if (at === 0) return separator;

  return null;
}

/** 把相对路径拼到目录上。`base` 为 null 时返回 null（不猜）。 */
export function joinPath(base: string | null, relative: string): string | null {
  if (base === null) return null;
  if (relative === '') return base;
  const separator = separatorOf(base);
  const trimmed = base.endsWith(separator) ? base.slice(0, -1) : base;
  return `${trimmed}${separator}${relative.replace(/^[\\/]+/, '')}`;
}

/**
 * 判断某个路径是否在某个目录之内。
 *
 * 用于文件树里「高亮当前文档所在位置」。刻意**不做符号链接解析** ——
 * 那是文件系统的事，纯字符串做不到也不该假装做到；这条只用来做 UI 提示，
 * 判断错了不会导致任何写入。
 */
export function isInside(path: string, directory: string): boolean {
  const separator = separatorOf(directory);
  const head = directory.endsWith(separator) ? directory : directory + separator;
  const a = separator === '\\' ? path.toLowerCase() : path;
  const b = separator === '\\' ? head.toLowerCase() : head;
  return a.startsWith(b);
}

/** 把路径拆成可显示的层级片段，用于面包屑。 */
export function segments(path: string): string[] {
  return path.split(/[\\/]+/).filter((part) => part !== '');
}
