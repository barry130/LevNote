/**
 * 代码高亮 —— 编辑器与导出共用同一份实现。
 *
 * 为什么共用：如果编辑器用一套、导出用另一套，用户会看到「编辑器里是彩色，导出的文件里是黑白」
 * 这种说不通的差异。两者必须同源。
 *
 * 实现方式：highlight.js 输出的是**带 class 的 HTML 片段**，而 ProseMirror 的装饰需要的是
 * **字符区间**。所以这里把 HTML 片段反解成区间 + class 名，两处各取所需。
 */

import hljs from 'highlight.js/lib/common';

import { createLruCache } from './cache.js';

export interface HighlightToken {
  /** 相对于代码块文本起始位置的字符偏移 */
  from: number;
  to: number;
  /** highlight.js 的类名，如 `hljs-keyword` */
  className: string;
}

/** 反解 hljs 输出里的实体，使偏移量与源文本对得上。 */
function decodeEntities(raw: string): string {
  let text = raw.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;/g, "'");
  text = text.replace(/&#(\d+);/g, (_match, code: string) => String.fromCodePoint(Number(code)));
  text = text.replace(/&#x([0-9a-fA-F]+);/g, (_match, code: string) => String.fromCodePoint(parseInt(code, 16)));
  // &amp; 必须最后处理，否则会把上面替换出来的 & 又解码一遍
  return text.replace(/&amp;/g, '&');
}

/**
 * 把 hljs 的 HTML 片段转成字符区间列表。
 *
 * 不引入 HTML 解析器：hljs 的输出形式极窄（只有成对的 `<span class="...">`、纯文本和 `<br>`），
 * 手写扫描比拉一个解析器进来更可预测，也能在无 DOM 环境（单测）里跑。
 */
export function tokenizeHighlightedHtml(html: string): HighlightToken[] {
  const tokens: HighlightToken[] = [];
  const open: Array<{ className: string; from: number }> = [];
  let offset = 0;
  let cursor = 0;

  const consumeText = (raw: string) => {
    offset += decodeEntities(raw).length;
  };

  while (cursor < html.length) {
    const lt = html.indexOf('<', cursor);
    if (lt === -1) {
      consumeText(html.slice(cursor));
      break;
    }
    if (lt > cursor) consumeText(html.slice(cursor, lt));

    const gt = html.indexOf('>', lt);
    if (gt === -1) break;
    const tag = html.slice(lt + 1, gt);

    if (tag.startsWith('/')) {
      const entry = open.pop();
      if (entry) tokens.push({ from: entry.from, to: offset, className: entry.className });
    } else if (tag === 'br' || tag === 'br/' || tag === 'br /') {
      offset += 1;
    } else {
      const match = /class\s*=\s*"([^"]*)"/.exec(tag);
      open.push({ className: match?.[1] ?? '', from: offset });
    }
    cursor = gt + 1;
  }

  // 容错：hljs 输出正常是配平的，但不配平时不该丢掉已识别的部分
  while (open.length > 0) {
    const entry = open.pop();
    if (entry) tokens.push({ from: entry.from, to: offset, className: entry.className });
  }

  return tokens
    .filter((token) => token.className !== '' && token.to > token.from)
    // 先按起点，再按长度降序：外层区间在前，装饰嵌套的层级才稳定
    .sort((a, b) => a.from - b.from || b.to - a.to);
}

export interface HighlightResult {
  /** 带 span 的 HTML，可直接放进 `<code>` */
  html: string;
  /** 实际用于高亮的语言（自动识别时可能与我们猜的不同） */
  language: string;
  tokens: HighlightToken[];
}

/**
 * 高亮结果缓存。
 *
 * ## 为什么必须有 —— 这一项占了长文档打字成本的一半
 *
 * `codeHighlightPlugin` 的 `apply` 在**每次文档变更**时整篇重跑
 * （`tr.docChanged ? buildDecorations(tr.doc)`，见 `modules/markdown/plugin.ts`），
 * 于是每个 `code_block` 都要重新高亮一遍 —— 哪怕用户改的只是正文里的一个字。
 *
 * 而**语言不认识时会退回 `highlightAuto()`**，那是把全部约 35 种语言各试一遍。
 * 实测一份 1.5 万字文档（16 个代码块，其中 5 个 `mermaid` 围栏、共 605 字符）：
 *
 * | 项 | 耗时 |
 * |---|---|
 * | 5 个 `mermaid` 围栏（`hljs` 不认识 `mermaid` → 走 `highlightAuto`） | **20.8ms** |
 * | 其余 11 个块（都写了 hljs 认识的语言） | 5.2ms |
 * | 合计（整篇重跑一次） | **26.0ms** |
 *
 * V3 实测一次按键的内核同步成本是 34ms —— 这一项就占六成，
 * 而且全花在「重新算一遍用户根本没碰过的代码块」上。
 *
 * 缓存之后，打字时的整篇重跑从 26.0ms 掉到 **0.05ms**（只有正在编辑的那个块真的重算）。
 *
 * ## 为什么是 LRU
 *
 * 见 `./cache.ts` 的头部：这里的访问模式是「稳定块反复命中、临时 key 只读一次」，
 * 正是 LRU 相对 FIFO 有决定性优势的那种负载。
 *
 * ## 返回值是**共享**的
 *
 * 命中时返回同一个对象。目前两个调用方都只读（`plugin.ts` 读 `tokens`、
 * `export/inline-html.ts` 读 `html` / `language`），所以共享是安全的 ——
 * **新增调用方不得修改返回值**，那会污染后续所有命中。
 *
 * ## 上限 500
 *
 * 一份文档的代码块数远小于它，余量留给「同一个块被连续编辑」产生的历史版本。
 * 容量只影响内存，不影响正确性。
 */
const CACHE_LIMIT = 500;
const cache = createLruCache<HighlightResult>(CACHE_LIMIT);

/**
 * 缓存 key。用 `\0` 分隔 —— 语言标识与代码正文都不可能含它，
 * 所以 `('js', 'x')` 与 `('js\0x', '')` 不会撞在一起。
 */
function cacheKey(code: string, normalizedLanguage: string): string {
  return `${normalizedLanguage}\0${code}`;
}

/**
 * 高亮一段代码。
 *
 * `language` 不认识时**退回自动识别**而不是放弃高亮 —— 用户写了 ```js 之外的冷门语言，
 * 看到无色文本会以为功能坏了。
 *
 * 结果走 LRU 缓存（见上），命中时直接返回，不重算。
 */
export function highlightCode(code: string, language?: string): HighlightResult | null {
  if (code.trim() === '') return null;

  const normalized = (language ?? '').trim().toLowerCase();
  const key = cacheKey(code, normalized);

  const hit = cache.get(key);
  if (hit !== undefined) return hit;

  let html: string;
  let used: string;

  try {
    if (normalized !== '' && hljs.getLanguage(normalized)) {
      html = hljs.highlight(code, { language: normalized }).value;
      used = normalized;
    } else {
      const auto = hljs.highlightAuto(code);
      html = auto.value;
      used = auto.language ?? '';
    }
  } catch {
    // 高亮失败绝不能影响编辑 —— 退回不高亮。
    // 失败不进缓存：那是异常路径，不值得为它占一个位置，
    // 而且失败往往是「这段代码触发了 hljs 的某个边界」，重试未必还失败。
    return null;
  }

  const result: HighlightResult = { html, language: used, tokens: tokenizeHighlightedHtml(html) };
  cache.set(key, result);
  return result;
}

export { hljs };

/**
 * hljs 认识的全部语言标识（`highlight.js/lib/common` 的约 35 种），已排序。
 *
 * 给语言选择器当候选列表用。**数据来自 hljs 而不是我们自己维护一张表** ——
 * 自己维护的那张表会在某天和 `highlightCode` 真正认得的东西对不上，
 * 于是用户从列表里选了一个语言，高亮却退回自动识别（`highlightCode` 对不认识的
 * 语言就是这么做），而他看不出为什么。
 *
 * 排序用 `localeCompare` 而不是默认的码位序：语言标识全是 ASCII，
 * 两种排法结果一样，但写 `localeCompare` 表达了「这是给人看的顺序」。
 */
export function supportedLanguages(): string[] {
  return [...hljs.listLanguages()].sort((a, b) => a.localeCompare(b));
}
