/**
 * 补上 jsdom **确实缺失**、而 ProseMirror 依赖的 DOM 接口。
 *
 * **这些不是被测代码的缺陷，是测试环境的缺口** —— 真实 WebView 里这些方法都在。
 * 不补的话，任何触发 `scrollIntoView()` 的操作（查找跳转、工具条格式化）
 * 都会抛 `target.getClientRects is not a function`，看起来像编辑器坏了，
 * 实际是 jsdom 少实现了一个接口。
 *
 * ## 缺的到底是哪几个（实测 jsdom 30.1.1，不是猜的）
 *
 * | 接口 | jsdom |
 * |---|---|
 * | `Range.prototype.getClientRects` | **缺** |
 * | `Range.prototype.getBoundingClientRect` | **缺** |
 * | `HTMLElement.prototype.scrollIntoView` | **缺** |
 * | `Element.prototype.getClientRects` | 有 |
 * | `Element.prototype.getBoundingClientRect` | 有（无布局引擎，返回全 0） |
 *
 * 所以下面用 `ensure()` **只补缺口**，绝不覆盖 jsdom 已有的实现 ——
 * 否则将来 jsdom 补上了，我们反而用一个更假的版本把它盖掉，
 * 而且没人会注意到（测试照样绿）。
 *
 * ## 为什么补成「空」而不是编一组坐标
 *
 * 编一个矩形会让 `coordsAtPos()` 返回看似合理的位置，于是测试开始断言一些
 * 在真实环境里未必成立的东西 —— 那是「假绿」。返回空列表则让 `singleRect()`
 * 退回到 `getBoundingClientRect()`（全 0），谁也不会据此得出错误结论。
 *
 * 这个文件通过 `vite.config.ts` 的 `test.setupFiles` 全局加载，所以必须对
 * `environment: 'node'` 的那些用例无害 —— 那里根本没有 `Range` / `HTMLElement`。
 */

/** 只在目标确实没有这个成员时才补。返回是否真的补了（便于人工核对）。 */
function ensure(target: object, name: string, value: unknown): boolean {
  const existing = (target as Record<string, unknown>)[name];
  if (typeof existing === 'function') return false;
  Object.defineProperty(target, name, { value, writable: true, configurable: true });
  return true;
}

const EMPTY_RECT_LIST = (): DOMRectList => [] as unknown as DOMRectList;
const ZERO_RECT = (): DOMRect => ({
  x: 0,
  y: 0,
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  width: 0,
  height: 0,
  toJSON: () => ({}),
});

if (typeof Range !== 'undefined') {
  ensure(Range.prototype, 'getClientRects', EMPTY_RECT_LIST);
  ensure(Range.prototype, 'getBoundingClientRect', ZERO_RECT);
}

if (typeof HTMLElement !== 'undefined') {
  ensure(HTMLElement.prototype, 'scrollIntoView', () => undefined);
}
