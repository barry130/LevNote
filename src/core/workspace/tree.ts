/**
 * 文件树的纯逻辑部分。
 *
 * 拆出来是因为「哪些目录展开了、每个目录里有哪些子项」属于**状态**，
 * 而「按什么顺序、以什么缩进把它铺成一列」属于**渲染**。后者是纯函数，
 * 可以脱离 React 和文件系统单独测 —— 而排序规则恰好是最容易写错、
 * 又最容易被忽视的一处（目录混在文件里、`10.txt` 排在 `9.txt` 前面）。
 */

import type { DirEntry } from '../document/types.js';

/**
 * 默认折叠的目录名。
 *
 * 这份名单的存在意义是「打开一个真实项目时侧栏不要卡住」：
 * `node_modules` 动辄几万个子项，`target`（Rust 构建产物）在本项目里是
 * 2 GB / 数千文件。它们不是用户想浏览的内容。
 *
 * 刻意**只按名字匹配、不做递归扫描**：真正的忽略规则应该读 `.gitignore`，
 * 那是检索功能（P4）要解决的问题，这里只需要一个「别让侧栏被淹掉」的兜底。
 */
const DEFAULT_IGNORED = new Set([
  'node_modules',
  '.git',
  '.svn',
  '.hg',
  'target',
  'dist',
  'out',
  'build',
  '__pycache__',
  '.venv',
  'venv',
  '.next',
  '.nuxt',
  '.cache',
  '.idea',
  '.DS_Store',
  'Thumbs.db',
]);

export function isIgnoredName(name: string): boolean {
  return DEFAULT_IGNORED.has(name);
}

/**
 * 排序：目录在前，然后按名字。
 *
 * 用 `numeric: true` 让 `2.md` 排在 `10.md` 前面 —— 默认的字典序会把
 * `10.md` 放到 `2.md` 前，这是文件列表里最刺眼的一类错误。
 * `sensitivity: 'base'` 让大小写不敏感的排序稳定（`a.md` 与 `A.md` 相邻）。
 */
export function compareEntries(a: DirEntry, b: DirEntry): number {
  if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
  return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' });
}

/** 过滤 + 排序，得到真正会显示给用户的子项。 */
export function visibleEntries(entries: readonly DirEntry[], showIgnored = false): DirEntry[] {
  return entries
    .filter((entry) => showIgnored || !isIgnoredName(entry.name))
    .slice()
    .sort(compareEntries);
}

export interface TreeRow {
  readonly path: string;
  readonly name: string;
  readonly isDir: boolean;
  /** 缩进层级，根的直接子项是 0 */
  readonly depth: number;
  readonly expanded: boolean;
  /** 该目录已加载过子项（用于区分「空目录」与「还没展开」） */
  readonly loaded: boolean;
}

/**
 * 把「目录 → 子项」的映射铺成一行行可渲染的树。
 *
 * 只有**已展开**的目录才会贡献子行。某一级目录还没加载（`dirs` 里没有它）
 * 时照样输出这一行，只是没有子行 —— 这与「空目录」在视觉上要靠 `loaded`
 * 区分，否则用户会以为这个目录是空的。
 *
 * 不设深度上限：目录树深到需要截断时，用户的问题已经不是这个列表了。
 */
export function flattenTree(
  dirs: Readonly<Record<string, readonly DirEntry[]>>,
  root: string | null,
  expanded: readonly string[],
  showIgnored = false,
): TreeRow[] {
  if (root === null) return [];

  const expandedSet = new Set(expanded);
  const rows: TreeRow[] = [];

  const walk = (path: string, depth: number): void => {
    const entries = dirs[path];
    if (entries === undefined) return;
    for (const entry of visibleEntries(entries, showIgnored)) {
      const isExpanded = entry.isDir && expandedSet.has(entry.path);
      rows.push({
        path: entry.path,
        name: entry.name,
        isDir: entry.isDir,
        depth,
        expanded: isExpanded,
        loaded: entry.isDir ? dirs[entry.path] !== undefined : true,
      });
      if (isExpanded) walk(entry.path, depth + 1);
    }
  };

  walk(root, 0);
  return rows;
}

/** 在树里找从根到某个路径的祖先目录链（不含该路径本身）。用于「在侧栏里定位当前文件」。 */
export function ancestorsOf(root: string | null, path: string, separator: '\\' | '/'): string[] {
  if (root === null) return [];
  if (!path.startsWith(root)) return [];
  const rest = path.slice(root.length).replace(/^[\\/]+/, '');
  const parts = rest.split(/[\\/]+/).filter((part) => part !== '');
  if (parts.length <= 1) return [];

  const out: string[] = [];
  let cursor = root.endsWith(separator) ? root.slice(0, -1) : root;
  for (let i = 0; i < parts.length - 1; i += 1) {
    cursor = `${cursor}${separator}${parts[i] ?? ''}`;
    out.push(cursor);
  }
  return out;
}
