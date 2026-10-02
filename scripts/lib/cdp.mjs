/**
 * 极简 CDP 客户端 —— `v2-run.mjs` / `v3-run.mjs` 共用的那一层。
 *
 * ## 为什么抽出来
 *
 * V2 与 V3 量的是完全不同的东西（导图内核 vs Markdown 内核），
 * 但它们要解决的是**同一组环境问题**，而且每一组都是踩出来的：
 *
 * | 问题 | 不解决会怎样 |
 * |---|---|
 * | 窗口被遮挡 / 最小化时 Chromium 挂起渲染管线 | 读数窜到上千毫秒，而那**不像 0 那样一眼可疑** |
 * | 抢前台的杂页（`edge://sync-confirmation-dialog/`） | 应用页变成后台页，rAF 被挂起，读数全是 0 |
 * | `Runtime.evaluate` 抛错被当成「返回 undefined」 | 拿到的是一份看着正常的假数据 |
 * | 敲字走 `Input.insertText` 而不是 `dispatchKeyEvent` | 跳过键盘事件那一层，测的不是「打字」 |
 *
 * 这些不是「两处各写一遍也没关系」的东西 —— 它们的失败形态都是**安静的**，
 * 抄第二份时漏掉一条，得到的是一份看起来正常的数据。
 *
 * ## 它**不**包含什么
 *
 * 页面里要跑什么、怎么让目标内容出现、怎么判断「输入真的进去了」——
 * 那些是各脚本自己的事（V3 要种会话 / 喂文件，V2 只要导航到一个本地页面）。
 * 这里只负责「连上、发命令、敲键、读数」。
 *
 * ## 状态是模块级的，不是工厂返回的对象
 *
 * 刻意的：这样调用方写的是 `send(...)` / `evaluate(...)`，
 * 与抽出来之前**一字不差** —— 抽取本身不会引入一批机械的调用点改动，
 * 而那批改动正是最容易在「没法跑一遍验证」的时候出错的地方。
 */

// `WebSocket` 用 Node 22 自带的全局实现（`v3-run.mjs` 一直就是这么用的）。
// 曾经在这里写过一行 `import { WebSocket } from 'node:ws-shim';` —— 那个模块
// 不存在，而且**不需要**：Node 22 的全局 `WebSocket` 就是 WHATWG 那个。
// 记在这里是因为它长得像个「正统做法」，抄的人会以为少了个依赖。

// ---------------------------------------------------------------------------
// 参数解析
// ---------------------------------------------------------------------------

/**
 * `--key value` / `--flag` 形式的最小解析器。
 *
 * `numberKeys` 里列出的键会被转成数字 —— 字符串 `'40'` 与数字 `40` 在
 * `for (let i = 0; i < args.keys; i++)` 里都能跑，但在报告里一个是字符串，
 * 会让「同一场景两次读数」的对比脚本出岔子。
 */
export function parseArgs(argv, defaults, numberKeys = []) {
  const out = { ...defaults };
  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    if (!token.startsWith('--')) continue;
    const name = token.slice(2);
    if (!(name in defaults)) continue;
    if (typeof defaults[name] === 'boolean') {
      out[name] = true;
      continue;
    }
    const value = argv[i + 1];
    i += 1;
    out[name] = numberKeys.includes(name) ? Number(value) : value;
  }
  return out;
}

// ---------------------------------------------------------------------------
// 连接与消息
// ---------------------------------------------------------------------------

let nextId = 1;
const pending = new Map();
const eventHandlers = new Map();
let socket = null;

/**
 * 发一条 CDP 命令。
 *
 * **必须有超时。** 没有超时的版本在实测里把一次压测挂死了 19 分钟：
 * 宿主（或驱动脚本自己）一旦卡住，`send` 返回的 Promise 永远不 settle，
 * 脚本就静静地挂在那儿 —— 而它**既不报错、也不退出**，
 * 唯一的症状是「跑了很久还没结果」。一个能自己失败的超时，
 * 比一个沉默的挂起好得多（这正是「绝不静默」在测量脚本里的形态）。
 *
 * 120 秒是刻意的宽裕值：`Runtime.evaluate` 里跑的是「一次 linkDiv」
 * 或「敲一个字」，正常都在毫秒级；超过 120 秒就一定是卡住了，不是慢。
 */
export function send(method, params = {}, { timeoutMs = 120000 } = {}) {
  const id = nextId;
  nextId += 1;
  return new Promise((resolvePromise, reject) => {
    const timer = setTimeout(() => {
      pending.delete(id);
      reject(
        new Error(
          `CDP ${method} 超过 ${timeoutMs}ms 没有回应 —— 宿主卡住了，或者这条命令不会被满足。\n` +
            `（不是「慢」：这个脚本里的每条命令正常都在毫秒级。读数不落盘，请重跑。）`,
        ),
      );
    }, timeoutMs);
    pending.set(id, {
      resolve: (value) => {
        clearTimeout(timer);
        resolvePromise(value);
      },
      reject: (error) => {
        clearTimeout(timer);
        reject(error);
      },
    });
    socket.send(JSON.stringify({ id, method, params }));
  });
}

/** 订阅一个 CDP 事件。同一个方法只留最后一个处理器 —— 这些脚本一次只等一件事。 */
export function on(method, handler) {
  eventHandlers.set(method, handler);
}

export async function evaluate(expression) {
  const result = await send('Runtime.evaluate', {
    expression,
    returnByValue: true,
    awaitPromise: true,
    userGesture: true,
  });
  // 页面里抛错时 `result.result.value` 是 undefined —— 不判这一下，
  // 调用方拿到的是「一个看起来正常的空值」，而不是「出错了」
  if (result.exceptionDetails) {
    const detail = result.exceptionDetails.exception?.description ?? result.exceptionDetails.text;
    throw new Error(`页面里求值失败：${detail}`);
  }
  return result.result.value;
}

export const sleep = (ms) => new Promise((done) => setTimeout(done, ms));

export async function waitFor(expression, { timeout = 20000, label = expression } = {}) {
  const deadline = Date.now() + timeout;
  for (;;) {
    if (await evaluate(`!!(${expression})`)) return;
    if (Date.now() > deadline) throw new Error(`等超时了：${label}`);
    await sleep(150);
  }
}

/**
 * 连上一个已经开着远程调试端口的宿主。
 *
 * `pagePattern` 挑出**目标页**（一个宿主可能有多个标签页）。
 * 挑错页的后果是「读数全是 0」而不是报错 —— 所以挑不到时会把当前所有目标
 * 打出来，而不是安静地连第一个。
 */
export async function connect({
  http = 'http://127.0.0.1:9222',
  pagePattern,
  hint = '',
  closeStrayPages = true,
  autoAcceptDialogs = true,
} = {}) {
  let targets;
  try {
    const response = await fetch(`${http}/json/list`);
    targets = await response.json();
  } catch (thrown) {
    throw new Error(
      `连不上 ${http}。先起一个带远程调试端口的宿主。\n${hint}\n` +
        `原始错误：${thrown instanceof Error ? thrown.message : String(thrown)}`,
      { cause: thrown },
    );
  }

  const page = targets.find((item) => item.type === 'page' && pagePattern.test(item.url));
  if (!page) {
    const seen = targets.map((item) => `${item.type} ${item.url}`).join('\n  ');
    throw new Error(`找不到目标页面（正则 ${pagePattern}）。当前目标：\n  ${seen || '（一个都没有）'}`);
  }

  // 抢前台的杂页是**同一个窗口里的另一个标签页**，会让目标页变成后台页 ——
  // 而后台页的 rAF 被挂起、Event Timing 一条都不产生，读数会全是 0。
  if (closeStrayPages) {
    for (const item of targets) {
      if (item.type === 'page' && /^(edge|devtools|chrome):/.test(item.url)) {
        await fetch(`${http}/json/close/${item.id}`).catch(() => undefined);
      }
    }
  }

  socket = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((resolvePromise, reject) => {
    socket.addEventListener('open', resolvePromise, { once: true });
    socket.addEventListener('error', () => reject(new Error('WebSocket 打不开')), { once: true });
  });
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    if (message.id === undefined) {
      if (message.method) eventHandlers.get(message.method)?.(message.params);
      return;
    }
    const entry = pending.get(message.id);
    if (!entry) return;
    pending.delete(message.id);
    if (message.error) entry.reject(new Error(`${message.error.message}（${JSON.stringify(message.error.data ?? null)}）`));
    else entry.resolve(message.result);
  });

  await send('Page.enable');
  await send('Runtime.enable');
  await send('DOM.enable');

  /**
   * 目标页必须是**活动标签页**，否则 Chromium 挂起它的渲染管线。
   *
   * 光 `Page.bringToFront` 不够：窗口如果被最小化了，它拉不回来。
   * 实测踩过 —— 跑到一半窗口被最小化，`paint` 直接窜到 1144ms / 1408ms，
   * 而那**不像 0 那样一眼可疑**（0 会被「样本数为 0」的警告兜住，
   * 上千毫秒只会看起来像「这一档特别慢」）。所以先用 Browser 域把窗口
   * 恢复成 normal 再提前台。
   */
  try {
    const { windowId } = await send('Browser.getWindowForTarget', { targetId: page.id });
    await send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'normal' } });
  } catch {
    // 有些宿主（WebView2）没有 Browser 域。拉不回来也不致命 ——
    // 各脚本自己的可见性闸门会挡住不可信的读数。
  }
  await send('Page.bringToFront');

  // 有些路径会 `window.alert(...)`，而它会**阻塞页面**（连带阻塞 `Runtime.evaluate`）。
  // 一律自动确认掉。
  if (autoAcceptDialogs) {
    on('Page.javascriptDialogOpening', () => {
      void send('Page.handleJavaScriptDialog', { accept: true }).catch(() => undefined);
    });
  }

  return page;
}

/** 收尾。脚本正常 / 异常退出时都该调。 */
export function close() {
  socket?.close();
  socket = null;
}

// ---------------------------------------------------------------------------
// 键鼠
// ---------------------------------------------------------------------------

export const MOD_CTRL = 2;

export const KEYS = {
  Home: { code: 'Home', vk: 36 },
  End: { code: 'End', vk: 35 },
  KeyO: { code: 'KeyO', vk: 79, key: 'o' },
  Digit1: { code: 'Digit1', vk: 49, key: '1' },
  Digit2: { code: 'Digit2', vk: 50, key: '2' },
  Digit3: { code: 'Digit3', vk: 51, key: '3' },
  Enter: { code: 'Enter', vk: 13 },
  Tab: { code: 'Tab', vk: 9 },
  Delete: { code: 'Delete', vk: 46 },
  Backspace: { code: 'Backspace', vk: 8 },
  Escape: { code: 'Escape', vk: 27 },
};

export async function pressKey(name, { ctrl = false } = {}) {
  const spec = KEYS[name];
  if (spec === undefined) throw new Error(`KEYS 里没有 ${name}`);
  const modifiers = ctrl ? MOD_CTRL : 0;
  const key = spec.key ?? name;
  await send('Input.dispatchKeyEvent', {
    type: 'keyDown',
    modifiers,
    key,
    code: spec.code,
    windowsVirtualKeyCode: spec.vk,
    nativeVirtualKeyCode: spec.vk,
  });
  await send('Input.dispatchKeyEvent', {
    type: 'keyUp',
    modifiers,
    key,
    code: spec.code,
    windowsVirtualKeyCode: spec.vk,
    nativeVirtualKeyCode: spec.vk,
  });
}

export const LETTERS = 'abcdefghijklmnopqrstuvwxyz';

/**
 * 敲一个 ASCII 字符。
 *
 * 走 `Input.dispatchKeyEvent` + `text` —— 这正是 Playwright `keyboard.type()`
 * 走的那条路，浏览器把它当成真实按键，`beforeinput` / `input` 都会发出来。
 * 不用 `Input.insertText`：那条路更短，会跳过键盘事件那一层，
 * 而我们要测的是**用户在键盘上打字**的成本。
 */
export async function typeChar(character) {
  const upper = character.toUpperCase();
  const vk = upper.charCodeAt(0);
  await send('Input.dispatchKeyEvent', {
    type: 'keyDown',
    key: character,
    code: `Key${upper}`,
    text: character,
    unmodifiedText: character,
    windowsVirtualKeyCode: vk,
    nativeVirtualKeyCode: vk,
  });
  await send('Input.dispatchKeyEvent', {
    type: 'keyUp',
    key: character,
    code: `Key${upper}`,
    windowsVirtualKeyCode: vk,
    nativeVirtualKeyCode: vk,
  });
}

export async function clickAt(x, y) {
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y, button: 'none' });
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
}

// ---------------------------------------------------------------------------
// 页面内的通用量具
// ---------------------------------------------------------------------------

/**
 * **校准基准**：在页面里跑一段固定的纯 CPU 工作负载，取 5 次的中位数。
 *
 * 为什么必须有它：实测同一份文档、同一个视图、同一台机器，
 * 两个批次之间 `sync` 从 18.3ms 变成 34ms —— **两个不同的内核按同一个比例
 * （约 1.8×）一起变慢**，而中间只过了二十分钟。原因是这台机器上同时跑着
 * 几套语言服务和一个前端 dev server，负载在漂。
 *
 * 后果很严重：判据是「≤ 50ms」，而读数能整体翻倍 —— **判定会随负载翻面**。
 * 所以每轮都跑一遍这个基准，跨批次比较时按它归一化；同一批次内部本来就可比。
 *
 * 取 5 次的中位数（第 3 小）：第一次含 JIT 预热，单次又太容易被抢占打断。
 */
export const CALIBRATE = `
(() => {
  const runs = [];
  let sink = 0;
  for (let r = 0; r < 5; r += 1) {
    const t0 = performance.now();
    let x = 0;
    for (let i = 0; i < 6000000; i += 1) x = (x + i) % 1000003;
    sink += x;
    runs.push(performance.now() - t0);
  }
  runs.sort((a, b) => a - b);
  window.__calibrationSink = sink;
  return Number(runs[2].toFixed(2));
})()
`;
