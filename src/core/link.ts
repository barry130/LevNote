/**
 * Markdown 链接地址的分类（纯函数）。
 *
 * ## 为什么需要它
 *
 * 文档里的 `href` 是一个**字符串**，而「点开它」这件事有四种完全不同的后果：
 * 在本文档里跳锚点、打开工作区里另一个文件、交给外部程序、或者根本不该点。
 * 这个判断只依赖 `href` 本身（不含文档路径），所以它是纯函数 ——
 * 而「相对谁」那一步归外壳（只有外壳知道文档在哪，见 `EditorMountProps.onLinkClick`）。
 *
 * ## 与 Milkdown 的白名单不是同一份名单
 *
 * `@milkdown/preset-commonmark` 内部有一个 `sanitizeLinkHref`，它决定
 * **一个地址能不能写进 DOM 的 `href` 属性**（挡 `javascript:` / `data:` 这类
 * 存储型 XSS）。这里的 `EXTERNAL_SCHEMES` 决定的是
 * **我们愿不愿意把它主动交出去**（写进剪贴板、将来交给系统浏览器）。
 *
 * 两者刻意不共用：目的不同，且这份**更窄**（只有 `http` / `https` / `mailto`）。
 * 宽进严出：渲染时认得的协议照常显示，我们主动处理的只有这三个。
 */

/** 我们**主动交出去**的协议。刻意比渲染白名单窄 —— 见文件头。 */
const EXTERNAL_SCHEMES: ReadonlySet<string> = new Set(['http:', 'https:', 'mailto:']);

/** 形如 `scheme:` 的前缀。`C:` 也会命中，所以盘符必须先判。 */
const SCHEME = /^([A-Za-z][A-Za-z0-9+.-]*):/;

/**
 * 浏览器在解析 URL 的协议时**会忽略**的字符。
 *
 * 它们常被用来把危险协议偷渡过朴素的前缀检查 —— `java\tscript:` 在浏览器里
 * 仍然解析成 `javascript:`。这里必须做同样的归一化，否则分类结论与浏览器
 * 的真实行为不一致，而这个函数的全部意义就是「预测浏览器会怎么理解它」。
 *
 * ## 为什么这里有一份、而不是复用 Milkdown 的那一份
 *
 * `@milkdown/preset-commonmark` 内部有一个同样功能的私有函数，但它
 * **不是公开导出**，而且 `core/` 不许依赖任何具体内核（硬约束 1）——
 * 依赖方向是 `modules/` → `core/`。所以这里是一份**独立的、更窄的实现**：
 * 它只服务「判断该不该主动交出去」，不承担「能不能写进 DOM」的职责。
 * 两者要盯的是同一个浏览器行为，改动时请一起看。
 */
function isIgnoredChar(code: number): boolean {
  return (
    code <= 0x20 ||
    (code >= 0x7f && code <= 0xa0) ||
    (code >= 0x200b && code <= 0x200d) ||
    code === 0x2028 ||
    code === 0x2029 ||
    code === 0xfeff
  );
}

/** 只用于**探测协议**：探测完仍然用原文，绝不把归一化后的串交出去。 */
function stripIgnoredChars(input: string): string {
  let out = '';
  for (const char of input) {
    if (!isIgnoredChar(char.codePointAt(0) ?? 0)) out += char;
  }
  return out;
}

/**
 * Windows 盘符绝对路径（`C:\a\b.md` 或 `C:/a/b.md`）。
 *
 * **必须早于协议识别**：`C:` 与 `https:` 长得一模一样，先按协议判的话
 * 盘符路径会被认成「不支持的协议」，用户点自己写的绝对路径链接只会得到一句
 * 「协议不受支持」—— 一个把可用功能说成不可用的错。
 */
const DRIVE_ABSOLUTE = /^[A-Za-z]:[\\/]/;

export type LinkTarget =
  /** 本文档内的锚点（`#标题`）。`anchor` 已解码。 */
  | { readonly kind: 'anchor'; readonly anchor: string }
  /**
   * 指向一个文件。`path` **还没有**解析成绝对路径 ——
   * 「相对谁」要当前文档的路径，那归外壳。
   *
   * `absolute` 为 true 表示 `path` 本身就是绝对路径（盘符 / UNC / POSIX 根），
   * 此时外壳**不能**走 `resolveRelative`（它会剥掉开头的分隔符）。
   */
  | {
      readonly kind: 'file';
      readonly path: string;
      readonly absolute: boolean;
      readonly anchor: string | null;
    }
  /** 外部地址（`http` / `https` / `mailto`）。 */
  | { readonly kind: 'external'; readonly href: string }
  /** 空地址，或只有一个 `#` —— 常见于「先占个位」的写法。 */
  | { readonly kind: 'empty' }
  /** 有明确协议但不在白名单里（`javascript:` / `data:` / `ftp:` / `tel:` …）。 */
  | { readonly kind: 'unsupported'; readonly href: string };

/**
 * 把地址拆成「路径部分」与「锚点部分」。
 *
 * 只按**第一个** `#` 切：`#` 之后的 `#` 是锚点文字的一部分（标题里带 `#`
 * 是合法的，Milkdown 的 slug 也不会把它转义）。
 */
export function splitFragment(href: string): { path: string; anchor: string | null } {
  const at = href.indexOf('#');
  if (at < 0) return { path: href, anchor: null };
  return { path: href.slice(0, at), anchor: href.slice(at + 1) };
}

/**
 * 锚点的百分号解码，失败时**原样返回**。
 *
 * 解码是为了与浏览器对齐：导出成 HTML 之后点 `#%E6%A0%87%E9%A2%98`，
 * 浏览器是解码后再去匹配 `id` 的，所以编辑器里也该这样匹配，
 * 否则同一份文档在编辑器与导出页里跳向不同的地方。
 *
 * 手写的地址里 `%` 不罕见（`#100%-覆盖率`），`decodeURIComponent` 会抛 ——
 * 那是用户的原文，不是错误，原样用就好。
 */
export function decodeAnchor(raw: string): string {
  if (!raw.includes('%')) return raw;
  try {
    return decodeURIComponent(raw);
  } catch {
    return raw;
  }
}

/** 锚点部分归一化：空串等同于「没有锚点」。 */
function anchorOrNull(anchor: string | null): string | null {
  return anchor === null || anchor === '' ? null : decodeAnchor(anchor);
}

/**
 * 判断一个 `href` 指向哪一类目标。
 *
 * 顺序在这里是有意义的，三条都不能换位置：
 * 1. **盘符先于协议** —— `C:` 长得像协议（见 `DRIVE_ABSOLUTE`）；
 * 2. **先拆锚点再判其余** —— 否则 `other.md#sec` 会被当成一个含 `#` 的文件名；
 * 3. **空路径 + 有锚点 = 本文档锚点** —— `#标题` 没有路径部分，这是它唯一的写法。
 */
export function classifyLink(href: string): LinkTarget {
  const trimmed = href.trim();
  if (trimmed === '') return { kind: 'empty' };

  const { path, anchor } = splitFragment(trimmed);

  // 没有路径部分：要么是本文档锚点，要么是 `#` / 空地址
  if (path === '') {
    const decoded = anchorOrNull(anchor);
    return decoded === null ? { kind: 'empty' } : { kind: 'anchor', anchor: decoded };
  }

  // 盘符与协议都在**归一化后**的串上探测（见 `stripIgnoredChars`）：
  // 归一化只用于判断，交出去的始终是原文
  const probe = stripIgnoredChars(path);

  if (DRIVE_ABSOLUTE.test(probe)) {
    return { kind: 'file', path, absolute: true, anchor: anchorOrNull(anchor) };
  }

  const scheme = SCHEME.exec(probe)?.[1];
  if (scheme !== undefined) {
    const lower = `${scheme.toLowerCase()}:`;
    return EXTERNAL_SCHEMES.has(lower)
      ? { kind: 'external', href: trimmed }
      : { kind: 'unsupported', href: trimmed };
  }

  // 以分隔符开头 = 绝对路径（POSIX 根 `/a/b`、UNC `\\server\share`）
  return { kind: 'file', path, absolute: /^[\\/]/.test(probe), anchor: anchorOrNull(anchor) };
}
