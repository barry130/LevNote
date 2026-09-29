/**
 * 查找与替换的纯函数内核。
 *
 * 这一层**不认识编辑器**：它只在字符串上工作，返回字符偏移。
 * 三个内核（Markdown / 代码 / 纯文本）各自把偏移翻译成自己模型的坐标，
 * 于是「大小写敏感、全词、正则」这三条语义只实现一次，
 * 三个内核的行为天然一致 —— 不会出现「Markdown 里全词匹配生效、代码里不生效」。
 *
 * ## 为什么自己造正则而不是用 `String.prototype.replaceAll`
 *
 * 用户输入要经过三步变换（转义 → 加全词边界 → 套大小写标志），
 * 然后必须能**拿到每个匹配的位置**用于「上一个 / 下一个」。
 * `replaceAll` 只给结果不给位置；`replace` 配回调能拿到位置但拿不到「第几个」。
 * 自己建一个带 `g` 标志的正则、显式遍历，三件事都能做，也更好测。
 */

export interface FindOptions {
  readonly query: string;
  readonly caseSensitive: boolean;
  /** 只匹配完整单词（`\b` 两侧） */
  readonly wholeWord: boolean;
  /** 把 `query` 当正则表达式 */
  readonly regex: boolean;
}

export interface TextMatch {
  /** 匹配起始字符偏移（含） */
  readonly from: number;
  /** 匹配结束字符偏移（不含） */
  readonly to: number;
  readonly text: string;
}

export const DEFAULT_FIND_OPTIONS: FindOptions = {
  query: '',
  caseSensitive: false,
  wholeWord: false,
  regex: false,
};

export function hasQuery(options: FindOptions): boolean {
  return options.query !== '';
}

/**
 * 把用户输入编译成带 `g` 标志的正则。
 *
 * 返回 `null` 表示「这次查找没有意义」：查询为空，或用户写了一个非法的正则。
 * **非法正则不抛错**：用户正在输入 `(` 的中间态就是非法正则，
 * 抛错会让查找栏在他打字的过程中不停闪红。返回 null 让上层显示「无结果」即可。
 */
export function createFindPattern(options: FindOptions): RegExp | null {
  if (options.query === '') return null;

  let source = options.regex ? options.query : escapeRegExp(options.query);
  if (options.wholeWord) source = `\\b(?:${source})\\b`;

  try {
    return new RegExp(source, options.caseSensitive ? 'g' : 'gi');
  } catch {
    return null;
  }
}

export function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * 在一段文本里找出所有匹配。
 *
 * 两个必须处理的退化情况：
 * - **零宽匹配**（查询 `.*`、`a*`、`^`）会让 `lastIndex` 不前进，直接死循环。
 *   所以每次匹配后若 `to === from`，强制前进一个字符。
 * - 正则可能带 `y`（粘性）或用户写了带 `g` 的模式 —— 统一用新建的正则，
 *   并且每次调用都从 0 开始，不依赖外部状态。
 */
export function findInText(text: string, options: FindOptions): TextMatch[] {
  const pattern = createFindPattern(options);
  if (pattern === null) return [];
  return findWithPattern(text, pattern);
}

/** 已经编译好的模式可以复用，避免在每个文本节点上重复编译。 */
export function findWithPattern(text: string, pattern: RegExp): TextMatch[] {
  const out: TextMatch[] = [];
  // 每次都从头开始：调用方可能复用同一个正则遍历多个文本节点
  pattern.lastIndex = 0;

  for (;;) {
    const hit = pattern.exec(text);
    if (hit === null) break;

    const from = hit.index;
    const matched = hit[0];
    out.push({ from, to: from + matched.length, text: matched });

    // 零宽匹配必须强制前进，否则 exec 会永远返回同一个位置
    pattern.lastIndex = matched.length === 0 ? from + 1 : from + matched.length;
    if (pattern.lastIndex > text.length) break;
  }

  return out;
}

/**
 * 替换全部匹配。
 *
 * `replacement` 里支持 `$1` / `$&` 这类正则替换语法 —— 这是用户用正则查找时的合理预期。
 * 但在**非正则**模式下必须把 `$` 当字面量，否则用户搜 `价格$` 想替换成 `￥`
 * 会得到莫名其妙的结果。用函数形式的 replace 参数就能区分这两种情况。
 */
export function replaceInText(
  text: string,
  options: FindOptions,
  replacement: string,
): { text: string; count: number } {
  const pattern = createFindPattern(options);
  if (pattern === null) return { text, count: 0 };

  let count = 0;
  const next = text.replace(pattern, (...args: unknown[]) => {
    count += 1;
    return expandReplacement(replacement, args, options.regex);
  });

  return { text: next, count };
}

/**
 * 手工展开 `$1` / `$&` / `$$`。
 *
 * 不用 `String.prototype.replace` 的内建展开是因为它在非正则模式下也生效，
 * 而我们要的恰恰是「非正则模式下 `$` 是普通字符」。
 */
function expandReplacement(replacement: string, args: unknown[], asRegex: boolean): string {
  if (!asRegex) return replacement;

  // 参数形状：($&, $1..$n, offset, string, groups?)
  const whole = String(args[0] ?? '');
  const offset = typeof args[args.length - 2] === 'number' ? (args[args.length - 2] as number) : -1;
  void offset;
  const groups = args.slice(1, args.length - 2);

  return replacement.replace(/\$(\$|&|\d{1,2})/g, (_all, token: string) => {
    if (token === '$') return '$';
    if (token === '&') return whole;
    const index = Number.parseInt(token, 10);
    const value = groups[index - 1];
    return value === undefined ? '' : String(value);
  });
}

/** 匹配是否重叠无关紧要，但数量要能提前报出来（UI 显示「共 N 处」）。 */
export function countMatches(text: string, options: FindOptions): number {
  return findInText(text, options).length;
}
