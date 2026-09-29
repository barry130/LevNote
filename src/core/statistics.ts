/**
 * 文本统计：行数 / 字数 / 字符数 / 阅读时长。
 *
 * 纯函数，不依赖任何内核 —— 富文本、源码、思维导图都该用同一套口径，
 * 否则同一份内容在两个视图里显示两个数，用户会不知道该信哪个。
 *
 * ## 为什么「字数」不能是 `text.length`
 *
 * `text.length` 是 **UTF-16 码元数**：汉字正好是 1 个码元（看着对得上），
 * 但 emoji 是 2 个、组合 emoji 更多，拉丁文更是「一个字母算一个字」。
 * 拿它当字数，在混排文档上会明显偏大。
 *
 * ## 字数的口径（**这是一个需要写下来的定义**）
 *
 * 中文写作工具之间并不统一，所以这里明确取 **Microsoft Word「字数」的口径** ——
 * 它是中文用户最常拿来对照的那个数：
 *
 * | 内容 | 计法 | 例 |
 * |---|---|---|
 * | 中日韩文字（表意文字 / 假名 / 韩文音节） | 每个 1 字 | `你好` → 2 |
 * | 中日韩标点与全角符号 | **也算 1 字** | `你好，世界。` → 6 |
 * | 拉丁字母 / 数字的连续串 | 整串算 1 个词 | `hello world` → 2 |
 * | 半角标点、空白 | 不计 | |
 *
 * 标点计入是有意的：你敲了 500 个字符，期望看到的就接近 500。
 * 若将来想改成「不含标点」，改 `isCjkPunctuation` 的调用点即可，只此一处。
 *
 * ## 为什么不用 `Intl.Segmenter` 做字素簇切分
 *
 * 字素簇更「正确」（组合 emoji 算 1 个字符），但它依赖 ICU，而且状态栏是
 * **每次内容变化都要重算**的 —— 对长文档（V3 关注的正是这条路径）是不必要的开销。
 * 这里改成**单次遍历码点 + 跳过零宽/组合附加符号**：够准、够快、无环境依赖。
 *
 * 已知简化：`👨‍👩‍👧` 计为 3（三个 emoji 各 1，两个零宽连接符不计），而不是 1。
 */

export interface TextStatistics {
  /** 行数。空文本算 1 行 —— 与状态栏「第 1 行，第 1 列」的光标显示保持一致。 */
  readonly lines: number;
  /** 字数（口径见文件头）。 */
  readonly words: number;
  /** 字符数，**不含**空白。 */
  readonly characters: number;
  /** 字符数，**含**空白与换行。 */
  readonly charactersWithSpaces: number;
  /** 估算阅读时长（分钟）。空文本为 0。 */
  readonly readingMinutes: number;
}

/** 拉丁词：字母/数字串，允许内部出现 `'`、`-`、`_`（`don't`、`well-known`、`snake_case`）。 */
const LATIN_WORD = /[A-Za-z0-9]+(?:['\u2019_-][A-Za-z0-9]+)*/g;

/** 阅读速度。中文按字、拉丁按词，分开算再相加 —— 两者速度差一倍以上。 */
const CJK_CHARS_PER_MINUTE = 300;
const LATIN_WORDS_PER_MINUTE = 200;

/** 中日韩文字：表意文字（含扩展区）、假名、韩文音节。 */
function isCjkLetter(cp: number): boolean {
  return (
    (cp >= 0x3040 && cp <= 0x30ff) || // 平假名 / 片假名
    (cp >= 0x3400 && cp <= 0x4dbf) || // 中日韩统一表意文字扩展 A
    (cp >= 0x4e00 && cp <= 0x9fff) || // 中日韩统一表意文字
    (cp >= 0xf900 && cp <= 0xfaff) || // 兼容表意文字
    (cp >= 0xac00 && cp <= 0xd7af) || // 韩文音节
    (cp >= 0x20000 && cp <= 0x2ebef) // 扩展 B–F（需要按码点判断，`charCodeAt` 拿不到）
  );
}

/** 中日韩标点与全角符号。按 Word 的口径计入字数。 */
function isCjkPunctuation(cp: number): boolean {
  return (
    (cp >= 0x3001 && cp <= 0x303f) || // 、。〈〉《》「」『』… 等（U+3000 是表意空格，下面按空白先拦掉）
    (cp >= 0xff01 && cp <= 0xff60) || // 全角 ！＂＃…Ｚ 等
    (cp >= 0xffe0 && cp <= 0xffe6) // 全角 ￠￡￥￦ 等
  );
}

/**
 * 不单独算一个「字符」的码点。
 *
 * 它们在人眼里不占位置，但会让码点计数虚高（组合 emoji 尤其明显）。
 */
function isZeroWidthOrCombining(cp: number): boolean {
  return (
    (cp >= 0x0300 && cp <= 0x036f) || // 组合附加符号
    (cp >= 0x200b && cp <= 0x200f) || // 零宽空格 / 零宽不连字 / 零宽连接符（组合 emoji 靠它）/ 方向标记
    cp === 0xfe0e ||
    cp === 0xfe0f || // 变体选择符（文本/emoji 呈现）
    (cp >= 0x1f3fb && cp <= 0x1f3ff) // 肤色修饰符
  );
}

function isWhitespace(cp: number): boolean {
  return (
    cp === 0x20 ||
    cp === 0x09 ||
    cp === 0x0a ||
    cp === 0x0d ||
    cp === 0xa0 || // 不换行空格
    cp === 0x3000 // 表意空格（全角空格）
  );
}

/**
 * 统计一份文本。
 *
 * **单次遍历**：长文档上每次内容变化都会调它，所以刻意不做多次正则扫描，
 * 也不物化中间数组。
 */
export function analyze(text: string): TextStatistics {
  let lines = 1;
  let characters = 0;
  let charactersWithSpaces = 0;
  let cjkChars = 0;

  for (const ch of text) {
    // `for...of` 按码点迭代，所以代理对（emoji、扩展区汉字）是一轮一次
    const cp = ch.codePointAt(0);
    if (cp === undefined) continue;

    if (ch === '\n') lines += 1;
    if (isZeroWidthOrCombining(cp)) continue;

    charactersWithSpaces += 1;
    if (isWhitespace(cp)) continue;

    characters += 1;
    if (isCjkLetter(cp) || isCjkPunctuation(cp)) cjkChars += 1;
  }

  const latinWords = text.match(LATIN_WORD)?.length ?? 0;
  const words = cjkChars + latinWords;

  // 用 round 而不是 ceil：ceil 会让 301 字显示成 2 分钟、300 字显示成 1 分钟，
  // 差一个字跳一档，看起来像坏了。round 的档位边界落在半分钟处，读数平滑得多。
  // 有内容时至少 1 分钟 —— 「约 0 分钟」不是信息。
  const minutes = cjkChars / CJK_CHARS_PER_MINUTE + latinWords / LATIN_WORDS_PER_MINUTE;
  const readingMinutes = words === 0 ? 0 : Math.max(1, Math.round(minutes));

  return { lines, words, characters, charactersWithSpaces, readingMinutes };
}

/**
 * 把统计结果格式化成状态栏那一行。
 *
 * 放在这里而不是 UI 里，是为了让「状态栏显示什么」也能被测试覆盖 ——
 * 空文档该显示成什么样，是个容易写错又没人会去点的边界。
 */
export function formatStatistics(stats: TextStatistics): string {
  const parts = [`${stats.lines} 行`, `${stats.words} 字`, `${stats.characters} 字符`];
  // 空文档不显示「约 0 分钟」—— 那不是信息，是噪声
  if (stats.readingMinutes > 0) parts.push(`约 ${stats.readingMinutes} 分钟`);
  return parts.join(' · ');
}
