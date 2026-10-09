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

/**
 * 打字机模式的滚动算术：把光标所在行挪到编辑区的**垂直正中**。
 *
 * 与上面两个函数同属一处的原因一样 —— Markdown 与代码内核都要它，
 * 而差一行的算术必须只有一份。参数全是普通数字（视口坐标系）：
 * 内核自己用 `coordsAtPos()` 取光标、用容器取几何，这里只做减法。
 *
 * 返回 `null` 表示**不用动**：容器还没布局（高度为 0，jsdom 或首帧）、
 * 拿到的几何不是数字，或者光标已经在正中（差值小于 1px —— 每敲一个字
 * 都重设一次 `scrollTop` 会跟浏览器的滚动合成器打架，也会吵醒位置记忆）。
 * 调用方拿到 `null` 直接返回，不写 `scrollTop`。
 */
export function typewriterScrollTop(
  caretViewportTop: number,
  scrollerViewportTop: number,
  currentScrollTop: number,
  clientHeight: number,
): number | null {
  if (!Number.isFinite(caretViewportTop) || !Number.isFinite(scrollerViewportTop)) return null;
  if (!Number.isFinite(currentScrollTop) || !Number.isFinite(clientHeight)) return null;
  if (!(clientHeight > 0)) return null;
  const caretContentTop = caretViewportTop - scrollerViewportTop + currentScrollTop;
  const target = Math.max(0, caretContentTop - clientHeight / 2);
  const delta = target - currentScrollTop;
  if (Math.abs(delta) < 1) return null;
  return target;
}
