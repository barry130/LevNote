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
 * 高亮一段代码。
 *
 * `language` 不认识时**退回自动识别**而不是放弃高亮 —— 用户写了 ```js 之外的冷门语言，
 * 看到无色文本会以为功能坏了。
 */
export function highlightCode(code: string, language?: string): HighlightResult | null {
  if (code.trim() === '') return null;

  const normalized = (language ?? '').trim().toLowerCase();
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
    // 高亮失败绝不能影响编辑 —— 退回不高亮
    return null;
  }

  return { html, language: used, tokens: tokenizeHighlightedHtml(html) };
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
