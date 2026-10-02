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

// ---------------------------------------------------------------------------
// 剪贴板（粘贴图片的测试需要）
// ---------------------------------------------------------------------------

/**
 * jsdom **完全没有** `DataTransfer` 与 `ClipboardEvent`（实测 30.1.1）。
 *
 * 这不是「某个方法没实现」，而是两个类不存在 —— 所以不能用上面那个
 * 「补方法」的 `ensure` 思路糊过去，得给出一个能用的最小实现。
 *
 * ## 为什么必须补，而不是绕开
 *
 * 图片粘贴的入口是 ProseMirror 的 `handlePaste`，它唯一的输入就是
 * `event.clipboardData`。绕开它（比如直接调内部的插入函数）等于把
 * 「事件怎么进来、我们有没有拦下默认处理」整段跳过 —— 而那段恰恰是
 * 最容易写错的地方（不 `preventDefault` 的话，默认的 slice 解析会把
 * 一张图变成一段空文本插进去）。
 *
 * ## 只实现被测代码真正用到的那几个成员 —— 但「用到」要算上 ProseMirror
 *
 * 第一版只做了 `files` / `items` / `types`，结果一跑就炸：
 * ProseMirror 的 `editHandlers.paste` 在调我们的 `handlePaste` **之前**
 * 就会读 `clipboardData.getData('text/plain')`。所以 `getData` 是必需的 ——
 * 这不是「顺手多做一个」，而是**调用链上游的既有依赖**。
 *
 * 教训：假实现要做到「被测代码**整条调用链**用到的成员」，而不是
 * 「我自己那几行用到的成员」。判断依据是实际跑一次，不是读自己的代码。
 *
 * `setData` 一并给出，好让测试能构造「同时带图片与文本」的剪贴板。
 */
class FakeDataTransferItem {
  readonly kind = 'file';
  constructor(private readonly file: File) {}
  getAsFile(): File {
    return this.file;
  }
  getAsString(): void {
    // 剪贴板里的纯文本 —— 图片路径用不到
  }
}

class FakeDataTransferItemList {
  private readonly list: FakeDataTransferItem[] = [];

  add(file: File): FakeDataTransferItem {
    const item = new FakeDataTransferItem(file);
    this.list.push(item);
    return item;
  }

  get length(): number {
    return this.list.length;
  }

  item(index: number): FakeDataTransferItem | null {
    return this.list[index] ?? null;
  }

  [Symbol.iterator](): Iterator<FakeDataTransferItem> {
    return this.list[Symbol.iterator]();
  }
}

class FakeDataTransfer {
  readonly items = new FakeDataTransferItemList();

  /** 纯文本那一半。ProseMirror 的 paste 处理器会在我们之前读它。 */
  private readonly text = new Map<string, string>();

  /** 被测代码只做 `Array.from(data.files)`，所以数组足够冒充 `FileList`。 */
  get files(): FileList {
    return this.itemsAsFiles() as unknown as FileList;
  }

  get types(): string[] {
    const types = this.itemsAsFiles().map((file) => file.type);
    return [...types, ...this.text.keys()];
  }

  setData(format: string, value: string): void {
    this.text.set(format, value);
  }

  getData(format: string): string {
    return this.text.get(format) ?? '';
  }

  clearData(format?: string): void {
    if (format === undefined) this.text.clear();
    else this.text.delete(format);
  }

  private itemsAsFiles(): File[] {
    const out: File[] = [];
    for (const item of this.items) out.push(item.getAsFile());
    return out;
  }
}

class FakeClipboardEvent extends Event {
  readonly clipboardData: DataTransfer | null;

  constructor(type: string, init: { clipboardData?: DataTransfer | null } & EventInit = {}) {
    super(type, init);
    this.clipboardData = init.clipboardData ?? null;
  }
}

if (typeof window !== 'undefined') {
  ensure(globalThis, 'DataTransfer', FakeDataTransfer);
  ensure(globalThis, 'ClipboardEvent', FakeClipboardEvent);
}

// ---------------------------------------------------------------------------
// matchMedia（导图内核的构造函数要它）
// ---------------------------------------------------------------------------

/**
 * jsdom **没有** `window.matchMedia`（实测 30.1.1）。
 *
 * 它不是「某个方法没实现」，而是整个函数不存在 —— 而 mind-elixir 在
 * **构造函数里**就会调它两次（`dist/MindElixir.js` 实测）：
 *
 * - `matchMedia('(prefers-color-scheme: dark)')` —— 决定默认主题的明暗；
 * - `matchMedia('(hover: none), (pointer: coarse)')` —— 判断是不是触屏。
 *
 * 所以缺了它，导图内核**根本创建不出来**：`new MindElixir(...)` 直接抛
 * `window.matchMedia is not a function`。而这条错误会走 `onError`，
 * 表现是「导图打不开」—— 看起来像产品坏了，其实是测试环境少了一个接口。
 *
 * `matches: false` 是有意选的：它表示「没有深色偏好、不是触屏」，
 * 也就是**最保守的那一支**。返回 `true` 会让内核走进触屏分支，
 * 那条路上的行为（多选、长按菜单）在 jsdom 里没有意义。
 *
 * 其余成员（`addListener` / `addEventListener`）一并给出：真实浏览器的
 * `MediaQueryList` 上有它们，内核将来接上「系统主题变了」的监听时不用再回来补。
 */
if (typeof window !== 'undefined') {
  ensure(window, 'matchMedia', (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }));
}
