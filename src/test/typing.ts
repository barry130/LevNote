/**
 * 在 jsdom 里把文字「打」进一个 contenteditable 编辑器。
 *
 * ## 为什么需要它
 *
 * jsdom 没有真实键盘。直接往 `view.state` 里塞事务能造出文档状态，但会**绕过**
 * 整条输入链路：DOM 变更 → `MutationObserver` → `DOMObserver.flush()` →
 * `readDOMChange` → `handleTextInput` → `input` 事件 → 我们的 `markdownEmitPlugin`。
 * 而斜杠菜单、输入规则、变更上报恰恰全都挂在这条链上 —— 绕过它，测试就只验证了
 * 「我自己拼的事务能进 state」，那是同义反复。
 *
 * 所以这里做的是**浏览器做的同一件事**：改 DOM 文字、把 DOM 光标放到末尾、
 * 派发 `input`。剩下的交给 ProseMirror 自己读。
 *
 * ## 两个必须遵守的顺序
 *
 * 1. **改完 DOM 要先让出一个宏任务，再派发 `input`。**
 *    `MutationObserver` 的回调是微任务，ProseMirror 的 `DOMObserver` 靠它把变更
 *    记进队列；`input` 处理器只负责「立刻结算队列」。顺序反了，队列是空的，
 *    这次输入会被完全忽略 —— 而且不报错，表现为「测试里打字没反应」。
 * 2. **之后还要等一帧。** 浮层上报（选中工具条 / 斜杠菜单）是用
 *    `requestAnimationFrame` 合并的，`setTimeout` 走不到它。
 *
 * ## 它**不**负责的事
 *
 * 只处理 ProseMirror（`.ProseMirror`）。CodeMirror 的输入链路完全是另一套，
 * 硬凑在一起只会得到一个「看起来通用、两边都不准」的辅助函数。
 *
 * 落笔位置是「可编辑区里**最后一个文本块**」，没有文字节点就在它里面造一个。
 * 对单段落文档就是「在段落末尾打字」；对以空段落结尾的文档就是「在末尾那个
 * 空段落里打字」。这个定义简单、可预测，而且正是这些用例需要的
 * （斜杠菜单只在段落里触发）。
 */

/** 让 MutationObserver 结算 + 等一帧浮层上报。 */
export async function settleTyping(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => {
    requestAnimationFrame(() => resolve(undefined));
  });
}

/**
 * 找出「打字应该落进去」的那个文字节点，没有就在最后一个文本块里造一个。
 *
 * 只认 `p` / `h1`–`h6`：ProseMirror 把列表项渲染成 `<li><p>`、把引用渲染成
 * `<blockquote><p>`，所以这两个选择器已经覆盖了所有「能打字的地方」。
 * 刻意**不**回退到 `.ProseMirror` 本身 —— 往 contenteditable 直接塞裸文字
 * 是非法结构，ProseMirror 读到的会是一个和预期完全不同的变更。
 */
function caretTarget(editable: HTMLElement): Text {
  const blocks = editable.querySelectorAll('p, h1, h2, h3, h4, h5, h6');
  const host: Element = blocks[blocks.length - 1] ?? editable;

  const first = host.firstChild;
  if (first !== null && first.nodeType === Node.TEXT_NODE) return first as Text;

  // 空文本块：DOM 里只有一个 `<br class="ProseMirror-trailingBreak">`（没有文字节点）。
  // 把文字节点插到它前面 —— ProseMirror 会把这个多出来的文字读成一次输入。
  const created = document.createTextNode('');
  host.insertBefore(created, host.firstChild);
  return created;
}

/**
 * 逐字打进编辑器。
 *
 * 刻意**一个字符一次**而不是一次塞一整串：输入规则（`# ` → 标题、`- ` → 列表）
 * 与斜杠筛选都是逐键推进的，一次塞完会跳过中间状态。
 */
export async function typeInto(editable: HTMLElement, text: string): Promise<void> {
  for (const char of text) {
    const node = caretTarget(editable);
    node.nodeValue = `${node.nodeValue ?? ''}${char}`;

    const range = document.createRange();
    range.setStart(node, (node.nodeValue ?? '').length);
    range.collapse(true);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);

    // 先让 MutationObserver 把变更记进队列，再让 `input` 处理器结算它（顺序见文件头）
    await new Promise((resolve) => setTimeout(resolve, 0));
    editable.dispatchEvent(new Event('input', { bubbles: true }));
    await settleTyping();
  }
}
