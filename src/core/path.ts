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

import { msg, type LocalizedText } from './i18n/index.js';

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
 * 认定为思维导图的扩展名。
 *
 * 与 `MARKDOWN_EXTENSIONS` 同一个理由放在 `core`：**名单只有一份**。
 * 目前只有 `.mindmap` 一个，看起来不值得单开一个常量 —— 但 `.mindmap`
 * 会出现在至少三处（内核注册、文件对话框的过滤器、导出/引用解析），
 * 而这三处「各自写一个字符串字面量」正是硬约束 5 说的那种**双重落空**：
 * 改了一处、忘了另一处，表现是「同一个文件在不同功能里被当成两种东西」。
 *
 * 刻意**不把 `.xmind` 收进来**：那是另一个格式（zip 容器），
 * 拿 JSON 解析器去读它只会得到 `NOT_JSON`，不如让它落到纯文本兜底。
 */
export const MINDMAP_EXTENSIONS: ReadonlySet<string> = new Set(['.mindmap']);

/** 路径是否属于思维导图。大小写不敏感（`extensionOf` 已统一小写）。 */
export function isMindmapPath(path: string): boolean {
  return MINDMAP_EXTENSIONS.has(extensionOf(path));
}

/**
 * 认定为 draw.io 图纸的扩展名（ADR 0048）。
 *
 * 跟 `MINDMAP_EXTENSIONS` 同一个理由：这份名单会被内核注册、文件对话框的
 * 过滤器、导出/引用解析、新建模板四处读到，所以只留一份。
 *
 * 刻意**不把 `.xml` 收进来**：`.xml` 是全世界都在用的一种泛型容器，
 * 绝大多数 `.xml` 根本不是 draw.io 图纸。打开对话框里 `filterDrawio`
 * 确实也列了 `xml`（那是「用户就是从 draw.io 导出来的」这一路的入口），
 * 但**认领**一条路径要的是「打开后按图渲染」，因此这里只认 `.drawio`。
 * 落到 `.xml` 的文件照旧由代码内核按 XML 源码打开 —— 说错类型比不认领更糟。
 */
export const DRAWIO_EXTENSIONS: ReadonlySet<string> = new Set(['.drawio']);

/** 路径是否属于 draw.io 图纸。大小写不敏感（`extensionOf` 已统一小写）。 */
export function isDrawioPath(path: string): boolean {
  return DRAWIO_EXTENSIONS.has(extensionOf(path));
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

/**
 * 把相对路径拼到目录上。`base` 为 null 时返回 null（不猜）。
 *
 * ## `./` 前缀要被吃掉，`../` 不能
 *
 * `./a.png` 与 `a.png` 指的是同一个文件，所以拼出来的路径里不该留着 `./` ——
 * 它会一路走到错误消息和 UI 上（`C:\笔记\./a.assets/x.png`），既难看又让人怀疑。
 *
 * 而 `../` **必须保留**：它不是「多余的前缀」，是真正的语义（去上一级）。
 * 所以这里只剥「最多一个点 + 分隔符」，`..` 后面跟分隔符时正则不匹配、原样保留。
 *
 * > 这个不一致是真实存在过的：`core/export/inline-html.ts` 里有一份私有的同名函数
 * > 剥了 `./`，而这里没剥 —— 同一个名字两种行为。已在那边改名为 `normalizeJoin`
 * > 并注明差异（它还多做了一层重复分隔符的收敛）。
 *
 * ## 相对部分的分隔符统一成 `base` 的风格
 *
 * Markdown 里的图片地址**必须是正斜杠**（`./a.assets/x.png` 在 Windows 的
 * 记事本里也是这么写的），而文档路径在 Windows 上是反斜杠。直接拼会得到
 * `C:\笔记\a.assets/x.png` 这种**混排**的字符串。
 *
 * 操作系统不在乎（Win32 两种都认），但我们的代码在乎：这条路径会被当作
 * **字符串**用在「写进去」和「读回来」两条路上 ——
 * 写入时文件名是裸名（`joinPath(目录, 'image-x.png')`，得到反斜杠），
 * 回读时来自 Markdown 的相对路径带正斜杠。同一个文件两个字符串，
 * 任何按字符串比较的地方（缓存键、去重、相等判断）都会悄悄失配。
 *
 * 所以这里统一成 `base` 的风格。**只换风格，不做别的**：
 * 不折叠重复分隔符、不解析 `..` —— 那些仍然属于写入层的职责。
 */
export function joinPath(base: string | null, relative: string): string | null {
  if (base === null) return null;
  if (relative === '') return base;
  const separator = separatorOf(base);
  const trimmed = base.endsWith(separator) ? base.slice(0, -1) : base;
  const tail = relative.replace(/^\.?[\\/]+/, '').replace(/[\\/]/g, separator);
  return `${trimmed}${separator}${tail}`;
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

/**
 * 把 doc-block / 图片引用里的相对 `src` 解析成**绝对路径**（相对 docPath 所在目录）。
 *
 * 为什么不直接 `joinPath(dirname(docPath), src)`：`dirname` 会把分隔符信息
 * **丢掉**——`notes\main.md` 的目录是裸名 `notes`，`joinPath` 对没有分隔符的
 * base 只能按 `/` 拼，产出的 `notes/detail.md` 与磁盘上的 `notes\detail.md`
 * 是**两个字符串**，任何按字符串比较的地方都会悄悄失配（硬约束 12 的变体：
 * base 风格不明时，风格要向上取自完整的文档路径，而不是取自裸目录名）。
 *
 * 返回 null 表示 docPath 没有目录信息（光秃秃的文件名），不能猜。
 */
export function resolveRelative(docPath: string, relative: string): string | null {
  const directory = dirname(docPath);
  if (directory === null) return null;
  const joined = joinPath(directory, relative);
  if (joined === null) return null;
  const separator = separatorOf(docPath);
  if (separator === '\\') return joined.split('/').join('\\');
  if (separator === '/') return joined.split('\\').join('/');
  return joined;
}

/** 把路径拆成可显示的层级片段，用于面包屑。 */
export function segments(path: string): string[] {
  return path.split(/[\\/]+/).filter((part) => part !== '');
}

/**
 * Windows 保留的设备名。这些名字**带任何扩展名**都不能作文件名
 * （`CON.txt` 也保留），所以只查第一个点之前的主名。
 */
const RESERVED_STEMS: ReadonlySet<string> = new Set([
  'CON',
  'PRN',
  'AUX',
  'NUL',
  ...Array.from({ length: 9 }, (_, i) => `COM${i + 1}`),
  ...Array.from({ length: 9 }, (_, i) => `LPT${i + 1}`),
]);

/**
 * 校验「新建 / 重命名」里用户输入的名字。合法返回 null，否则返回**一句还没翻的话**。
 *
 * 规则对齐 Windows 资源管理器（本项目是 Windows 优先的桌面应用），且**宁严勿松**：
 * 这里放过去一个名字，磁盘层（Rust 命令）多半也会接受，但产出的文件在资源管理器里
 * 打不开、删不掉 —— 错误在很远的地方才暴露。输入会先被调用方 trim，这里再做一次兜底。
 *
 * 为什么在 `core` 而不是 UI：store 的编排层（`WorkspaceStore.createEntry` /
 * `renameEntry`）要在**碰磁盘之前**用同一份规则拦一道，测试也要钉同一份规则 ——
 * 放在组件里，「新建」与「重命名」迟早各写一份。
 *
 * ## 为什么返回 `LocalizedText` 而不是句子（ADR 0047）
 *
 * 它的返回值会**进 `tree.error` 那个 store 字段**，而 store 字段跨渲染活下来 ——
 * 返回句子的话，切语言时「这个名字非法」那句话会停在旧语言上
 * （和 ADR 0031 的 `BUILT_IN_COMMANDS`、ADR 0046 的记录本同一族）。
 * 所以交出去的是 `msg(...)`，翻译推迟到 Sidebar 渲染那一刻（`textOf`）。
 */
export function validateEntryName(rawName: string): LocalizedText | null {
  const name = rawName.trim();
  if (name === '') return msg('core.path.empty');
  if (name === '.' || name === '..') return msg('core.path.dotNames');
  if (/[\\/:*?"<>|]/.test(name)) return msg('core.path.invalidChars');
  for (const ch of name) {
    if (ch.charCodeAt(0) < 32) return msg('core.path.controlChars');
  }
  if (name.endsWith('.') || name.endsWith(' ')) return msg('core.path.trailingDotSpace');
  if (name.length > 200) return msg('core.path.tooLong');
  const stem = (name.split('.')[0] ?? '').toUpperCase();
  if (RESERVED_STEMS.has(stem)) return msg('core.path.reservedStem', { stem });
  return null;
}

