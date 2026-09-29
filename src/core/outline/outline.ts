/**
 * 从 Markdown 源码里提取标题大纲。
 *
 * 刻意**不经过编辑器内核**：大纲要能在一个还没挂载内核的文档上算出来
 * （标签页预览、保存前的快照），而且它必须是纯函数才好测。
 * 代价是它得自己认一遍语法 —— 但只需要认「哪些行是标题」这一件事。
 *
 * ## 必须跳过的两种「假标题」
 *
 * 1. **围栏代码块内部**。` ```js ` 里写 `# 这是注释` 是极常见的写法，
 *    不跳过的话大纲里会冒出一堆代码注释。围栏的配对规则按 CommonMark：
 *    结束围栏必须同一种字符、长度不小于开始、且后面只有空白。
 * 2. **YAML front matter**。`---` 之间可能有多行 `title:` 之类的内容，
 *    虽然它一般不以 `#` 开头，但 front matter 里出现注释 `# xxx` 是合法的 YAML 注释。
 * 3. **缩进 4 空格以上的行**。那是缩进代码块，不是标题。
 */

export interface OutlineItem {
  /** 1–6 */
  readonly level: number;
  /** 用于显示的标题文字（已剥掉行内标记） */
  readonly text: string;
  /** 1 基行号，用于跳转 */
  readonly line: number;
}

interface Fence {
  readonly char: '`' | '~';
  readonly length: number;
}

const ATX = /^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$/;
const SETEXT = /^ {0,3}(=+|-+)[ \t]*$/;

/** 围栏开始行：```` ```js ````、`~~~~`。信息串里不能有反引号（CommonMark 的限制）。 */
const FENCE_OPEN = /^ {0,3}(`{3,}|~{3,})(.*)$/;

/**
 * 剥掉标题里的行内标记，只留给人看的文字。
 *
 * 只处理最常见的几种：图片、链接、行内代码、强调。**不做完整 Markdown 解析** ——
 * 大纲是导航用的，`**重点**` 显示成 `**重点**` 与显示成 `重点` 都不影响点击跳转，
 * 多写一百行解析器换不到对应的价值。
 */
export function stripInlineMarkup(text: string): string {
  return (
    text
      // 图片必须在链接之前处理，否则 ![alt](src) 会被链接规则吃掉一部分
      .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
      .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
      .replace(/\[([^\]]*)\]\[[^\]]*\]/g, '$1')
      .replace(/`+([^`]*)`+/g, '$1')
      .replace(/(\*\*|__)(.*?)\1/g, '$2')
      .replace(/(\*|_)(.*?)\1/g, '$2')
      .replace(/~~(.*?)~~/g, '$1')
      .replace(/\\([\\`*_{}[\]()#+\-.!])/g, '$1')
      .trim()
  );
}

/** 判断某一行是不是「setext 下划线」：需要上一行是非空正文，且上上行不是标题。 */
function isSetextUnderline(line: string, previous: string | null): boolean {
  if (previous === null) return false;
  const trimmed = previous.trim();
  if (trimmed === '') return false;
  // 上一行本身是标题或列表项时，`---` 更可能是分隔线 / 列表延续，不当作 setext
  if (ATX.test(previous)) return false;
  if (/^ {0,3}([-*+]|\d+[.)])[ \t]/.test(previous)) return false;
  return SETEXT.test(line);
}

export function extractOutline(markdown: string): OutlineItem[] {
  const items: OutlineItem[] = [];
  const lines = markdown.split('\n');
  let fence: Fence | null = null;
  let inFrontMatter = false;
  let frontMatterEnd = -1;

  // front matter 只在文件最开头成立：第一行必须是恰好 `---`
  if (lines.length > 0 && lines[0]?.trim() === '---') {
    inFrontMatter = true;
    for (let i = 1; i < lines.length; i += 1) {
      const line = lines[i] ?? '';
      if (line.trim() === '---' || line.trim() === '...') {
        frontMatterEnd = i;
        break;
      }
    }
    // 找不到结束标记：不把它当 front matter（否则整篇文档都进不了大纲）
    if (frontMatterEnd === -1) inFrontMatter = false;
  }

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i] ?? '';

    if (inFrontMatter) {
      if (i >= frontMatterEnd) inFrontMatter = false;
      continue;
    }

    // ---- 围栏状态机 ----
    if (fence !== null) {
      const closing = new RegExp(`^ {0,3}\\${fence.char}{${fence.length},}[ \\t]*$`);
      if (closing.test(line)) fence = null;
      continue;
    }

    const open = FENCE_OPEN.exec(line);
    if (open) {
      const marker = open[1] ?? '';
      // 反引号围栏的信息串里不允许再出现反引号（CommonMark 明文规定）
      if (open[1]?.[0] !== '`' || !(open[2] ?? '').includes('`')) {
        fence = { char: marker[0] as '`' | '~', length: marker.length };
        continue;
      }
    }

    // ---- ATX ----
    const atx = ATX.exec(line);
    if (atx) {
      const level = (atx[1] ?? '').length;
      // 结尾的 # 是装饰性的：`## 标题 ##` 的标题是「标题」
      const raw = (atx[2] ?? '').replace(/[ \t]+#+[ \t]*$/, '');
      items.push({ level, text: stripInlineMarkup(raw), line: i + 1 });
      continue;
    }

    // ---- setext ----
    if (isSetextUnderline(line, i > 0 ? (lines[i - 1] ?? null) : null)) {
      const raw = (lines[i - 1] ?? '').trim();
      items.push({ level: (line.trim()[0] === '=' ? 1 : 2), text: stripInlineMarkup(raw), line: i });
    }
  }

  return items;
}

/**
 * 给大纲项标出「最深的公共层级」以外需要缩进的量。
 *
 * 真实文档里 `#` 之后直接跳到 `###` 很常见，直接按 level 缩进会留下空档。
 * 这里按「出现过的层级」重排成连续深度，让大纲看起来是紧凑的树。
 */
export function outlineDepths(items: readonly OutlineItem[]): number[] {
  const seen: number[] = [];
  return items.map((item) => {
    while (seen.length > 0 && (seen[seen.length - 1] ?? 0) >= item.level) seen.pop();
    seen.push(item.level);
    return seen.length - 1;
  });
}
