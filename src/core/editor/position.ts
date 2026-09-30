/**
 * 位置记忆里「滚动」那一半的共用算术。
 *
 * 三个内核（Markdown / 代码 / 纯文本）的滚动容器各不相同 ——
 * `.qm-editor`、CodeMirror 的 `.cm-scroller`、textarea 自身 —— 但**算法只有这一份**：
 * 「取比例」和「按比例还原」都要处理「容器没有可滚动空间」这个除零情形，
 * 抄三份的结果一定是其中两份忘了判。
 *
 * 参数用**结构类型**而不是 `HTMLElement`：这样单测可以直接喂一个普通对象，
 * 不需要在 jsdom 里造出真实的布局（jsdom 根本没有布局引擎，
 * `scrollHeight` 恒等于 `clientHeight`，拿真实元素测等于什么都没测）。
 */

/** 只要能读出滚动几何的东西就行 —— `HTMLElement` 与测试里的普通对象都满足。 */
export interface ScrollMetrics {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
}

function clamp01(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.min(1, Math.max(0, value));
}

/**
 * 当前滚动比例（0–1）。
 *
 * 容器没有可滚动空间时（内容比视口短，`scrollHeight === clientHeight`）
 * 返回 0 而不是 `NaN` 或 `Infinity` —— 这个值会被写进存储、之后又读回来，
 * 一旦落进去一个 `NaN`，JSON 会把它变成 `null`，恢复时再乘一次就得到 `NaN`，
 * 最终表现是「打开文档后滚动条位置是随机的」。
 */
export function readScrollRatio(element: ScrollMetrics): number {
  const span = element.scrollHeight - element.clientHeight;
  if (!(span > 0)) return 0;
  return clamp01(element.scrollTop / span);
}

/**
 * 按比例还原滚动位置。
 *
 * 没有可滚动空间时**什么都不做**（而不是把 `scrollTop` 设成 0）：
 * 那可能是「内容还没渲染完」的中间态，硬写一次 0 会把用户原本的位置抹掉。
 */
export function applyScrollRatio(element: ScrollMetrics, ratio: number): void {
  const span = element.scrollHeight - element.clientHeight;
  if (!(span > 0)) return;
  element.scrollTop = clamp01(ratio) * span;
}
