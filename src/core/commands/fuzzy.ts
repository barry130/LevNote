/**
 * 命令面板的模糊匹配。
 *
 * ## 为什么不用现成的模糊搜索库
 *
 * 面板要搜的是**几十条命令**，不是几万行代码。这类规模下，
 * 一个 30 行的子序列匹配器比 `fuse.js`（几十 KB + 一堆可调参数）更合适：
 * 行为完全可预测，排序规则能写清理由，出问题时不需要读别人的源码。
 *
 * ## 评分规则（按重要性从高到低）
 *
 * 1. **连续匹配优于分散匹配** —— 搜 `save` 时 `保存(save)` 必须排在
 *    `s a v e` 前面。这条是**最重要的一条**：早期版本把「词首命中」的权重
 *    定得比「连续」高，结果是「每个字母都是一个词的开头」这种最不相关的匹配
 *    反而得分最高 —— 中文标题里夹杂一个英文词时，这会把正确结果挤下去。
 * 2. **整段连续命中额外加分** —— 也就是「子串命中」，它几乎总是用户想要的。
 * 3. **词首匹配加分** —— 搜 `os` 时 `open file` 比 `close` 更该靠前。
 * 4. **匹配位置越靠前越好**；
 * 5. **越短越好** —— 同样的匹配质量，短标题更可能是用户想要的。
 *
 * 中文命令名没有「词首」概念，因此规则 3 对中文基本不生效；
 * 中文靠 `keywords`（拼音）命中，这是把 `baocun` 打到「保存」上的唯一途径。
 */

export interface FuzzyMatch {
  readonly score: number;
  /** 命中字符在原文中的下标，供 UI 高亮 */
  readonly positions: readonly number[];
}

/**
 * 子序列匹配。返回 null 表示不匹配。
 *
 * 大小写不敏感：用户不会为了搜命令去管大小写。
 */
export function fuzzyMatch(query: string, text: string): FuzzyMatch | null {
  if (query === '') return { score: 0, positions: [] };
  if (text === '') return null;

  const haystack = text.toLowerCase();
  const needle = query.toLowerCase();

  const positions: number[] = [];
  let score = 0;
  let cursor = 0;
  let previousHit = -1;
  let allConsecutive = true;

  for (const char of needle) {
    const at = haystack.indexOf(char, cursor);
    if (at < 0) return null;

    positions.push(at);
    score += 10;

    /**
     * 连续命中：与上一个命中相邻。
     *
     * `previousHit >= 0` 这个前置判断不能省。少了它，**第一个字符永远会被当成
     * 「与上一个相邻」**（因为 `previousHit` 初值是 -1，而首字符的下标恰好是 0），
     * 于是每个查询都白拿一份连续加分 —— 而这份加分对「首字符在位置 0」的
     * 分散匹配（例如 `s a v e` 里的 `s`）比对真正的连续匹配（`保存 save` 里的 `save`）
     * 还要划算，排序结果正好反过来。
     */
    if (previousHit >= 0) {
      if (at === previousHit + 1) score += 14;
      else allConsecutive = false;
    }

    // 词首命中：开头，或前一个字符是分隔符
    if (at === 0 || isBoundary(haystack.charAt(at - 1))) score += 8;

    // 位置越靠前越好
    score -= Math.min(at, 12);

    previousHit = at;
    cursor = at + 1;
  }

  // 整段连续命中 = 子串命中。单独加一笔，让「连续」在整体排序上有决定性。
  // 单字符查询不加：对它来说「整段连续」是恒真的，加了等于给所有匹配同样加分。
  if (allConsecutive && needle.length > 1) score += 30;

  // 覆盖率高（匹配用了标题的大部分）再加一点分
  score += Math.round((needle.length / haystack.length) * 10);
  // 短标题优先
  score -= Math.min(text.length, 20);

  return { score, positions };
}

function isBoundary(char: string): boolean {
  return char === ' ' || char === '-' || char === '_' || char === '.' || char === '/' || char === ':';
}

export interface ScoredCandidate<T> {
  readonly item: T;
  readonly score: number;
  readonly positions: readonly number[];
  /** 命中的是哪个字段，供 UI 决定在哪显示高亮 */
  readonly field: 'title' | 'keywords' | 'id';
}

/**
 * 在一组候选里排序。
 *
 * 标题命中优先于关键词命中优先于 id 命中：用户打 `保存` 时，
 * 「保存」这条命令必须排在「导出为自包含 HTML」（关键词里有 `baocun` 吗？没有）
 * 之类的前面 —— 标题是用户最可能想的东西。
 *
 * 全部不命中时返回空数组（面板显示「没有匹配的命令」），
 * 而不是退化成「显示全部」—— 那会让用户以为搜索没生效。
 */
export function rankCommands<T>(
  query: string,
  items: readonly T[],
  fields: (item: T) => { title: string; keywords: readonly string[]; id: string },
): ScoredCandidate<T>[] {
  const trimmed = query.trim();
  if (trimmed === '') {
    return items.map((item) => ({ item, score: 0, positions: [], field: 'title' as const }));
  }

  const out: ScoredCandidate<T>[] = [];
  for (const item of items) {
    const { title, keywords, id } = fields(item);

    const byTitle = fuzzyMatch(trimmed, title);
    if (byTitle) {
      out.push({ item, score: byTitle.score + 200, positions: byTitle.positions, field: 'title' });
      continue;
    }

    const byKeyword = keywords.map((word) => fuzzyMatch(trimmed, word)).find((hit) => hit !== null) ?? null;
    if (byKeyword) {
      out.push({ item, score: byKeyword.score + 100, positions: byKeyword.positions, field: 'keywords' });
      continue;
    }

    const byId = fuzzyMatch(trimmed, id);
    if (byId) out.push({ item, score: byId.score, positions: byId.positions, field: 'id' });
  }

  return out.sort((a, b) => b.score - a.score);
}
