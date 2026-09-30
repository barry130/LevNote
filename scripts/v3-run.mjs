/**
 * V3 的**自动化驱动器** —— 用 CDP 驱动真实 WebView，不需要人往 Console 里粘东西。
 *
 * ## 为什么要有它
 *
 * `v3-probe.mjs` 是一段要人工粘贴、人工敲字的脚本。人工做一次两次没问题，
 * 但 V3 有**四个场景**（三档视图 × 起点/中部）要跑，还要在改代码之后能重跑 ——
 * 手工流程的代价是「每次都要重新对齐姿势」，而姿势不一致，读数就不可比。
 *
 * ## 它怎么绕过那两个「人工」
 *
 * | 人工步骤 | 这里怎么做 |
 * |---|---|
 * | 打开一份 1 万字以上的文档 | 用 `Page.addScriptToEvaluateOnNewDocument` 在**新文档的第一个脚本之前**把会话种进 `localStorage`，然后 reload —— 应用自己的会话恢复会把它打开 |
 * | 把光标放到中部 | 把编辑器的滚动容器滚到 50%，然后在它的矩形中心派发一次**真实鼠标点击**（`Input.dispatchMouseEvent`）。落到哪一段由浏览器自己算，不由我们猜 |
 * | 敲字 | `Input.dispatchKeyEvent`（带 `text`），也就是 Playwright `keyboard.type()` 走的那条 CDP 路径 —— **浏览器眼中的可信输入**，会完整走一遍 `beforeinput` → `MutationObserver` → `input` 管线 |
 * | 粘探针 | `Runtime.evaluate` 整份 `v3-probe.mjs` 的源码 |
 *
 * ## 为什么种会话要绕一圈
 *
 * 「先 `localStorage.setItem` 再 `location.reload()`」是**不行的**：
 * 应用在 `beforeunload` 里会把当前会话（空的）写回去，正好把我们种的那份覆盖掉。
 * `addScriptToEvaluateOnNewDocument` 的注入点在新文档的**最早**时刻，
 * 晚于旧文档的 `beforeunload`、早于应用的任何一行代码 —— 这是唯一没有竞态的时机。
 *
 * ## 用法
 *
 * 先起带远程调试端口的宿主（见 `docs/open-decisions.md` 的 V3 一节）。**两条路**：
 *
 * ```bash
 * # 桌面版（真 WebView2）—— 需要在 `src-tauri/tauri.conf.json` 里给窗口加
 * # `"additionalBrowserArgs": "… --remote-debugging-port=9222"`
 * # （**注意**：`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` 环境变量**没有用**，
 * #  见下面「环境变量的坑」）
 * npx tauri dev -c scripts/v3-dev.conf.json
 *
 * # 或者真实 Edge（同一个 Chromium 引擎，见下）。
 * # 这几条 --disable-* 是**必须**的：少了它们，窗口被别的窗口盖住时
 * # Chromium 会挂起渲染管线，读数会窜到上千毫秒（见下面「页面必须可见」）
 * msedge.exe --remote-debugging-port=9222 --user-data-dir=<空目录> \
 *   --disable-features=CalculateNativeWinOcclusion,msEdgeSidebarV2 \
 *   --disable-backgrounding-occluded-windows \
 *   --disable-renderer-backgrounding --disable-background-timer-throttling \
 *   http://127.0.0.1:1420/
 * ```
 *
 * 然后：
 *
 * ```bash
 * node scripts/v3-run.mjs --file "F:/…/docs/samples/v3-longdoc.md" --mode rich --at middle
 * node scripts/v3-run.mjs --provider browser --fresh --file … --mode source --at start --out .v3-out/x.json
 * node scripts/v3-summary.mjs            # 把 .v3-out/ 里的读数汇总成一张表
 * ```
 *
 * `--fresh` 先 reload 再重开文档 —— 上一轮敲进去的字符会留在文档里
 * （探针只负责测量、不负责还原），不重载的话每跑一档就往上叠 60 个字符，
 * 场景之间就不再可比。
 *
 * ## 页面必须可见（否则读数会「看起来只是慢」）
 *
 * 页面被隐藏或完全遮挡时 Chromium 挂起渲染管线，`paint` 会窜到 **1144ms / 1408ms** ——
 * 而那**不像 0 那样一眼可疑**，只会被读成「这一档特别慢」。所以脚本做了三件事：
 * 起 Edge 时带 `--disable-features=CalculateNativeWinOcclusion`
 * `--disable-backgrounding-occluded-windows`；连接后把窗口从最小化恢复并提前台
 * （`Page.bringToFront` 单独不够 —— Windows 不许后台进程抢前台）；
 * 以及**全程跟踪 `visibilitychange`**，丢过前台就判失败、**不落盘**。
 *
 * ## 环境变量的坑（踩过，写下来省下一次）
 *
 * `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9222` **不会生效**。
 * 原因是 `wry` 无条件调用 `options.set_additional_browser_arguments(...)`
 * （没给就填默认的 `--disable-features=msWebOOUI,msPdfOOUI,msSmartScreenProtection`），
 * 而 WebView2 的约定是：**宿主显式设了这个属性，环境变量就被忽略**。
 * 所以只能走窗口配置里的 `additionalBrowserArgs`。
 *
 * ## `--provider browser` 是什么
 *
 * 桌面版在有些环境里起不来（本工具环境就是这样：窗口建出来了、进程也 Responding，
 * 但 WebView2 的子进程一个都没有）。此时可以拿 `npm run dev` + 真实 Edge 顶上 ——
 * **同一个 Chromium 引擎、同一套布局与绘制**，差的只是宿主外壳。
 * 代价是文档来源换成了浏览器预览的内存虚拟盘，所以打开文件要走
 * `<input type="file">`，这里用 CDP 的 `Page.setInterceptFileChooserDialog`
 * 把文件喂进去（也就是 Playwright `setInputFiles` 走的那条路）。
 *
 * **它测的是同一件事**：输入延迟由「内核 JS + 样式/布局/绘制」决定，
 * 与文档来源无关。而且它有一个校验锚点 —— 人工在真 WebView2 上跑过的那一轮，
 * 同一份文档 / 同一个视图 / 同一个落点的读数是 `sync` p50 14.7 / `paint` p50 32。
 * 两条路对得上，才说明这条路可信。
 *
 * ## 两个它自己会检查的前提（都不通过就报错退出，不会给你一份假数据）
 *
 * 1. **输入真的进了内核** —— 注入探针之前先挂一个独立的 `beforeinput` 计数器，
 *    敲一个字符，读计数器。为 0 说明这条输入路径没走通，此时**读数会全是 0**，
 *    而 0 看起来像「性能极好」—— 这是最危险的一种假绿。
 * 2. **刷新率合理** —— 窗口被遮挡时 Chromium 会节流渲染，`paint` 会变成几十上百毫秒。
 *    报告里带刷新率，脚本会在低于 30Hz 时警告。
 */

import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';

const CDP_HTTP = process.env.V3_CDP ?? 'http://127.0.0.1:9222';
const PROBE = readFileSync(new URL('./v3-probe.mjs', import.meta.url), 'utf8');

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

function parseArgs(argv) {
  const out = {
    file: null,
    provider: 'desktop',
    mode: 'rich',
    at: 'middle',
    keys: 40,
    interval: 70,
    warmup: 1000,
    settle: 1500,
    out: null,
    keep: false,
    fresh: false,
  };
  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    if (!token.startsWith('--')) continue;
    const name = token.slice(2);
    if (name === 'keep' || name === 'fresh') {
      out[name] = true;
      continue;
    }
    const value = argv[i + 1];
    i += 1;
    if (name === 'keys' || name === 'interval' || name === 'warmup' || name === 'settle') out[name] = Number(value);
    else out[name] = value;
  }
  return out;
}

const args = parseArgs(process.argv.slice(2));
if (!args.file) {
  console.error('用法：node scripts/v3-run.mjs --file <文档绝对路径> [--mode rich|source|split] [--at start|middle|end] [--keys 40]');
  process.exit(2);
}
const FILE = resolve(args.file);

/** 过程备注。失败时也要打出来 —— 不然只剩一句「没走通」，什么线索都没有。 */
const notes = [];

// ---------------------------------------------------------------------------
// 极简 CDP 客户端
// ---------------------------------------------------------------------------

let nextId = 1;
const pending = new Map();
const eventHandlers = new Map();
let socket = null;

function send(method, params = {}) {
  const id = nextId;
  nextId += 1;
  return new Promise((resolvePromise, reject) => {
    pending.set(id, { resolve: resolvePromise, reject });
    socket.send(JSON.stringify({ id, method, params }));
  });
}

/** 订阅一个 CDP 事件。同一个方法只留最后一个处理器 —— 这个脚本一次只等一件事。 */
function on(method, handler) {
  eventHandlers.set(method, handler);
}

async function evaluate(expression) {
  const result = await send('Runtime.evaluate', {
    expression,
    returnByValue: true,
    awaitPromise: true,
    userGesture: true,
  });
  if (result.exceptionDetails) {
    const detail = result.exceptionDetails.exception?.description ?? result.exceptionDetails.text;
    throw new Error(`页面里求值失败：${detail}`);
  }
  return result.result.value;
}

const sleep = (ms) => new Promise((done) => setTimeout(done, ms));

async function waitFor(expression, { timeout = 20000, label = expression } = {}) {
  const deadline = Date.now() + timeout;
  for (;;) {
    if (await evaluate(`!!(${expression})`)) return;
    if (Date.now() > deadline) throw new Error(`等超时了：${label}`);
    await sleep(150);
  }
}

async function connect() {
  let targets;
  try {
    const response = await fetch(`${CDP_HTTP}/json/list`);
    targets = await response.json();
  } catch (thrown) {
    throw new Error(
      `连不上 ${CDP_HTTP}。先起一个带远程调试端口的宿主：\n` +
        `  桌面版：npx tauri dev -c scripts/v3-dev.conf.json\n` +
        `          （窗口配置里的 additionalBrowserArgs 带 --remote-debugging-port=9222。\n` +
        `           注意 WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 环境变量**没有用** ——\n` +
        `           wry 无条件调 set_additional_browser_arguments，宿主设了属性就忽略环境变量）\n` +
        `  浏览器：见本文件头部那段 msedge 命令（--disable-* 那几条是必须的）\n` +
        `原始错误：${thrown instanceof Error ? thrown.message : String(thrown)}`,
      { cause: thrown },
    );
  }

  const page = targets.find((item) => item.type === 'page' && /127\.0\.0\.1:1420|tauri:|localhost/.test(item.url));
  if (!page) {
    const seen = targets.map((item) => `${item.type} ${item.url}`).join('\n  ');
    throw new Error(`找不到应用页面。当前目标：\n  ${seen || '（一个都没有）'}`);
  }

  // 把抢前台的杂页关掉（Edge 首次启动会弹 `edge://sync-confirmation-dialog/` 之类的页）。
  // 它们是**同一个窗口里的另一个标签页**，会让应用页变成后台页 ——
  // 而后台页的 rAF 被挂起、Event Timing 一条都不产生，读数会全是 0。
  for (const item of targets) {
    if (item.type === 'page' && /^(edge|devtools|chrome):/.test(item.url)) {
      await fetch(`${CDP_HTTP}/json/close/${item.id}`).catch(() => undefined);
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

  // 应用页必须是**活动标签页**，否则 Chromium 挂起它的渲染管线。
  //
  // 光 `Page.bringToFront` 不够：窗口如果被最小化了，它拉不回来。
  // 实测踩过 —— 跑到一半窗口被最小化，`paint` 直接窜到 1144ms / 1408ms，
  // 而那**不像 0 那样一眼可疑**（0 会被「样本数为 0」的警告兜住，
  // 上千毫秒只会看起来像「这一档特别慢」）。所以先用 Browser 域把窗口
  // 恢复成 normal 再提前台。
  try {
    const { windowId } = await send('Browser.getWindowForTarget', { targetId: page.id });
    await send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'normal' } });
  } catch {
    // 有些宿主（WebView2）没有 Browser 域。拉不回来也不致命 ——
    // 下面的可见性闸门会挡住不可信的读数。
  }
  await send('Page.bringToFront');

  // 浏览器预览模式下 `notify` 会 `window.alert(...)`，而它会**阻塞页面**
  // （连带阻塞我们的 `Runtime.evaluate`）。一律自动确认掉。
  on('Page.javascriptDialogOpening', () => {
    void send('Page.handleJavaScriptDialog', { accept: true }).catch(() => undefined);
  });

  return page;
}

// ---------------------------------------------------------------------------
// 键鼠
// ---------------------------------------------------------------------------

const MOD_CTRL = 2;

const KEYS = {
  Home: { code: 'Home', vk: 36 },
  End: { code: 'End', vk: 35 },
  KeyO: { code: 'KeyO', vk: 79, key: 'o' },
  Digit1: { code: 'Digit1', vk: 49, key: '1' },
  Digit2: { code: 'Digit2', vk: 50, key: '2' },
  Digit3: { code: 'Digit3', vk: 51, key: '3' },
};

async function pressKey(name, { ctrl = false } = {}) {
  const spec = KEYS[name];
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

const LETTERS = 'abcdefghijklmnopqrstuvwxyz';

/**
 * 敲一个 ASCII 字符。
 *
 * 走 `Input.dispatchKeyEvent` + `text` —— 这正是 Playwright `keyboard.type()`
 * 走的那条路，浏览器把它当成真实按键，`beforeinput` / `input` 都会发出来。
 * 不用 `Input.insertText`：那条路更短，会跳过键盘事件那一层，
 * 而我们要测的是**用户在键盘上打字**的成本。
 */
async function typeChar(character) {
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

async function clickAt(x, y) {
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y, button: 'none' });
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
}

// ---------------------------------------------------------------------------
// 页面内的辅助代码（都是字符串，交给 Runtime.evaluate）
// ---------------------------------------------------------------------------

/** 找到「真正可编辑的那一个」元素。分屏右栏是 `contenteditable="false"`，天然被排除。 */
const FIND_EDITABLE = `
(() => {
  const all = [...document.querySelectorAll('[contenteditable="true"]')];
  return all.find((el) => el.closest('.editor-pane') !== null) ?? all[0] ?? null;
})()
`;

/**
 * 把编辑区滚到指定比例，返回**一个保证落在可见编辑区内**的点。
 *
 * 滚动容器不一定是 `contenteditable` 本身：富文本内核的滚动在 `.qm-editor` 上，
 * 代码内核在 `.cm-scroller` 上。所以**往上走，找第一个真的能滚的祖先** ——
 * 这比写死类名稳（硬约束 11：外壳不许按类名去认内核画的东西）。
 *
 * **为什么要取「交集」而不是编辑区矩形的中心**：编辑区元素和内容一样高
 * （一份 2 万字的文档约 4.7 万像素），它的矩形中心在 `ratio = 0` 时算出来是
 * y ≈ 23500px —— 视口外面，点击落空、焦点丢失，接下来敲的字一个都进不去内核。
 * 这正是 `--at start` / `--at end` 第一次跑挂掉的原因（`middle` 是碰巧：
 * 内容中点 ≈ 视口中心）。取「内容矩形 ∩ 滚动容器可见矩形」的中心就稳了。
 */
const SCROLL_TO = (ratio) => `
(() => {
  const editable = ${FIND_EDITABLE};
  if (!editable) return null;
  let scroller = editable;
  while (scroller && scroller.scrollHeight <= scroller.clientHeight + 4) scroller = scroller.parentElement;
  const host = (scroller && scroller !== document.body && scroller !== document.documentElement) ? scroller : editable;
  host.scrollTop = (host.scrollHeight - host.clientHeight) * ${ratio};

  const e = editable.getBoundingClientRect();
  const c = host.getBoundingClientRect();
  const top = Math.max(e.top, c.top);
  const bottom = Math.min(e.bottom, c.bottom);
  const left = Math.max(e.left, c.left);
  const right = Math.min(e.right, c.right);
  const usable = bottom - top >= 20 && right - left >= 20;
  const box = usable
    ? { top, bottom, left, right }
    : { top: c.top, bottom: c.bottom, left: c.left, right: c.right };
  return {
    x: (box.left + box.right) / 2,
    y: (box.top + box.bottom) / 2,
    visibleHeight: Math.round(box.bottom - box.top),
    usedIntersection: usable,
    scrolled: host !== editable,
  };
})()
`;

/** 状态栏里的「第 N 行，第 M 列」与「字数」。用它证明光标真的落到了我们想要的地方。 */
const READ_STATUS = `
(() => {
  const bar = document.querySelector('.statusbar');
  const text = bar ? bar.textContent : '';
  const line = /第\\s*(\\d+)\\s*行，第\\s*(\\d+)\\s*列/.exec(text);
  const stats = (bar && bar.querySelector('.statistics')) ? bar.querySelector('.statistics').textContent : '';
  const kernel = (bar && bar.querySelector('.kernel')) ? bar.querySelector('.kernel').textContent : '';
  return { line: line ? Number(line[1]) : null, column: line ? Number(line[2]) : null, stats, kernel, title: document.title, visibility: document.visibilityState, hidden: document.hidden };
})()
`;

/** 文档总行数：拿渲染出来的块级元素数量估一个比例，只为「光标是不是在中部」提供一个旁证。 */
const DOC_SHAPE = `
(() => {
  const editable = ${FIND_EDITABLE};
  if (!editable) return null;
  return { blocks: editable.querySelectorAll(':scope > *').length, chars: editable.textContent.length };
})()
`;

/**
 * **校准基准**：在页面里跑一段固定的纯 CPU 工作负载，取中位数。
 *
 * 为什么必须有它：实测同一份文档、同一个视图、同一台机器，
 * 两个批次之间 `sync` 从 18.3ms 变成 34ms、源码档从 4.1ms 变成 7.3ms ——
 * **两个内核按同一个比例（约 1.8×）一起变慢**，而中间只过了二十分钟。
 * 原因是这台机器上同时跑着几套语言服务和一个前端 dev server，负载在漂。
 *
 * 后果很严重：V3 的判据是「≤ 50ms」，而读数能整体翻倍 ——
 * **判定会随负载翻面**。所以每轮都跑一遍这个基准，
 * 跨批次比较时按它归一化；同一批次内部本来就是可比的。
 *
 * 取 5 次的中位数（第 3 小）：第一次含 JIT 预热，单次又太容易被抢占打断。
 */
const CALIBRATE = `
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
  window.__v3sink = sink;
  return Number(runs[2].toFixed(2));
})()
`;

// ---------------------------------------------------------------------------
// 主流程
// ---------------------------------------------------------------------------

function seedSessionSource(file) {
  const session = {
    tabs: [{ path: file, draft: null, at: Date.now() }],
    activePath: file,
    root: null,
    droppedDrafts: 0,
  };
  // 注意这里**不是** JSON.stringify(session) 直接塞：要嵌进一段源码里，
  // 所以先 stringify 成字符串、再 stringify 一次做字面量。
  return `try { localStorage.setItem('qingmo.session.v1', ${JSON.stringify(JSON.stringify(session))}); } catch (e) {}`;
}

const baseName = (path) => path.split(/[\\/]/).pop() ?? path;

/** 桌面版：靠应用自己的会话恢复把文档打开（种进 localStorage 再 reload）。 */
async function openViaSessionSeed(notes) {
  if (await evaluate(`document.title.includes(${JSON.stringify(baseName(FILE))})`)) {
    notes.push('目标文档已经开着，跳过种会话这一步');
    return;
  }
  const injected = await send('Page.addScriptToEvaluateOnNewDocument', { source: seedSessionSource(FILE) });
  await send('Page.reload', { ignoreCache: false });
  // 注入的脚本只该影响这一次导航，之后立刻撤掉，免得污染后续场景
  await sleep(1200);
  await send('Page.removeScriptToEvaluateOnNewDocument', { identifier: injected.identifier });
}

/**
 * 浏览器预览：走应用自己的「打开文件」命令，把文件从 CDP 喂进
 * `<input type="file">`（`Page.setInterceptFileChooserDialog` + `DOM.setFileInputFiles`，
 * 也就是 Playwright `setInputFiles` 走的那条路）。
 *
 * 比种会话更长，但**走的是真实的产品路径** —— 顺带证明了「打开文件」这条链路是通的。
 */
async function openViaFileChooser(notes) {
  if (await evaluate(`document.title.includes(${JSON.stringify(baseName(FILE))})`)) {
    notes.push('目标文档已经开着，跳过打开这一步');
    return;
  }

  await send('Page.setInterceptFileChooserDialog', { enabled: true });
  let resolveChooser;
  const chooser = new Promise((resolvePromise) => {
    resolveChooser = resolvePromise;
  });
  on('Page.fileChooserOpened', resolveChooser);

  await clickAt(200, 200);
  await pressKey('KeyO', { ctrl: true });

  const event = await Promise.race([chooser, sleep(8000).then(() => null)]);
  if (event === null) {
    await send('Page.setInterceptFileChooserDialog', { enabled: false });
    throw new Error('没有等到文件选择事件 —— Ctrl+O 可能没被应用接住，或者拦截没生效');
  }
  await send('DOM.setFileInputFiles', { files: [FILE], backendNodeId: event.backendNodeId });
  await send('Page.setInterceptFileChooserDialog', { enabled: false });
  notes.push('已通过文件选择把文档喂进去（浏览器预览没有磁盘）');
}

async function main() {
  const page = await connect();
  notes.push(`页面：${page.url}`);

  // 渲染管线必须真的在跑。页面不可见时 Chromium 会挂起 rAF，
  // Event Timing 一条都不会产生 —— 读数会全是 0，而 0 看起来像「极快」。
  const visibility = await evaluate('document.visibilityState');
  if (visibility !== 'visible') {
    throw new Error(
      `页面不是可见状态（visibilityState = ${visibility}）—— 此时渲染管线被挂起，读数会全是 0。\n` +
        '排查方向：窗口被最小化、被别的窗口完全遮挡，或者应用页不是活动标签页。',
    );
  }

  // --- 1. 让应用打开目标文档 -------------------------------------------------
  // `--fresh`：先 reload。上一轮敲进去的字符会留在文档里（探针只负责测量，
  // 不负责还原），不重载的话每跑一档就往上叠 40 个字符 —— 场景之间就不再可比。
  // 浏览器预览的内存虚拟盘会随 reload 一起丢掉，所以紧接着还要重新喂文件。
  if (args.fresh) {
    await send('Page.reload', { ignoreCache: false });
    await sleep(1500);
    notes.push('已重载页面（--fresh），文档从干净状态重新打开');
  }

  if (args.provider === 'browser') await openViaFileChooser(notes);
  else await openViaSessionSeed(notes);

  await waitFor(`document.querySelector('.shell') !== null`, { label: '应用外壳' });
  await waitFor(`!document.body.textContent.includes('正在恢复上次的标签页')`, { label: '会话恢复结束' });
  await waitFor(FIND_EDITABLE, { label: '编辑器内核就绪' });
  await sleep(600);

  const opened = await evaluate(READ_STATUS);
  notes.push(`打开：${opened.title} · 内核 ${opened.kernel.trim()}`);
  notes.push(`页面可见性：${opened.visibility}（hidden=${opened.hidden}）—— 隐藏/被遮挡时 Chromium 会挂起 rAF，读数会全是 0`);

  // --- 2. 切到目标视图 -------------------------------------------------------
  const MODE_KEY = { rich: 'Digit1', source: 'Digit2', split: 'Digit3' };
  await clickAt(200, 200); // 先让窗口拿到焦点，键盘事件才会进应用
  await pressKey(MODE_KEY[args.mode], { ctrl: true });
  await sleep(900);
  await waitFor(FIND_EDITABLE, { label: `${args.mode} 视图下的编辑器` });
  await sleep(400);

  const afterSwitch = await evaluate(READ_STATUS);
  notes.push(`视图 ${args.mode} → 内核 ${afterSwitch.kernel.trim()}`);

  // --- 3. 落光标 -------------------------------------------------------------
  const shape = await evaluate(DOC_SHAPE);
  const ratio = args.at === 'start' ? 0 : args.at === 'end' ? 1 : 0.5;
  const box = await evaluate(SCROLL_TO(ratio));
  if (box === null) throw new Error('找不到编辑区，没法落光标');
  notes.push(
    `滚到 ${ratio} → 点击 (${Math.round(box.x)}, ${Math.round(box.y)})，可见编辑区高 ${box.visibleHeight}px` +
      `${box.usedIntersection ? '' : '（退化到容器矩形）'}${box.scrolled ? '' : '（没找到滚动容器）'}`,
  );
  await clickAt(box.x, box.y);
  await sleep(400);

  // 点击之后焦点必须真的在编辑器里，否则后面敲的字一个都进不去，
  // 而表现是「读数全是 0」—— 看起来像性能极好。
  const focused = await evaluate(
    `(() => { const el = ${FIND_EDITABLE}; return el !== null && (el === document.activeElement || el.contains(document.activeElement)); })()`,
  );
  if (!focused) throw new Error('点击之后焦点不在编辑器里 —— 光标落不进去，后面敲的字会被丢掉');
  notes.push('点击后焦点在编辑器内');

  if (args.at === 'start') await pressKey('Home', { ctrl: true });
  if (args.at === 'end') await pressKey('End', { ctrl: true });
  await sleep(400);

  const atCursor = await evaluate(READ_STATUS);
  if (atCursor.line === null) throw new Error('状态栏没有报出行号 —— 光标可能没落进编辑器');
  notes.push(`光标：第 ${atCursor.line} 行（共约 ${shape.blocks} 个块、${shape.chars} 字符）· ${atCursor.stats}`);

  // --- 4. 前提检查：输入真的进得去吗 ----------------------------------------
  await evaluate(`
    (() => {
      window.__v3pre = 0;
      const el = ${FIND_EDITABLE};
      window.__v3preTarget = el;
      window.__v3preHandler = () => { window.__v3pre += 1; };
      el.addEventListener('beforeinput', window.__v3preHandler, true);
    })()
  `);
  await typeChar('z');
  await sleep(250);
  const reached = await evaluate('window.__v3pre');
  await evaluate(`
    (() => {
      const el = window.__v3preTarget;
      if (el && window.__v3preHandler) el.removeEventListener('beforeinput', window.__v3preHandler, true);
      delete window.__v3preTarget;
      delete window.__v3preHandler;
    })()
  `);

  if (reached === 0) {
    throw new Error(
      'CDP 的按键没有变成 beforeinput —— 这条输入路径没走通。\n' +
        '读数会全是 0，而 0 看起来像「性能极好」，所以这里直接停。\n' +
        '排查方向：窗口是否真的拿到了焦点（`Input.dispatchKeyEvent` 需要页面在前台）、\n' +
        '编辑器是否还没就绪、或者 ProseMirror 把这一层拦掉了。',
    );
  }
  notes.push(`前提检查通过：CDP 按键 → beforeinput（计数器 ${reached}）`);

  // 把刚才那个探路字符撤掉，别污染被测文档
  await send('Input.dispatchKeyEvent', { type: 'keyDown', modifiers: MOD_CTRL, key: 'z', code: 'KeyZ', windowsVirtualKeyCode: 90, nativeVirtualKeyCode: 90 });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', modifiers: MOD_CTRL, key: 'z', code: 'KeyZ', windowsVirtualKeyCode: 90, nativeVirtualKeyCode: 90 });
  await sleep(400);

  // --- 5. 装探针，敲字 -------------------------------------------------------
  // 敲字**之前**先量一次校准基准，敲完再量一次 —— 两者差得多说明这一轮里负载就变了
  const baselineBefore = await evaluate(CALIBRATE);
  notes.push(`校准基准（敲字前）：${baselineBefore}ms —— 跨批次比较要按它归一化`);

  await evaluate(PROBE);
  // 探针在**加载时就自己 attach 过一次**了，这里不要再调一次 ——
  // 重复挂监听会让每条输入产生两个标记，其中一个 `sync` 是 null，
  // 而它配不上 Event Timing 的条目就变成孤儿噪声。只读一下数量做核对。
  const editableCount = await evaluate(`document.querySelectorAll('[contenteditable="true"]').length`);
  notes.push(`探针已挂载，页面上可编辑区 ${editableCount} 个`);
  if (editableCount !== 1) {
    notes.push(`⚠️ 可编辑区有 ${editableCount} 个 —— 探针会把每一个都算一遍，读数可能被重复计数，请人工确认`);
  }

  await evaluate('window.__v3.reset()');
  await sleep(300);

  const typed = [];
  for (let i = 0; i < args.keys; i += 1) {
    const character = LETTERS[i % LETTERS.length];
    await typeChar(character);
    typed.push(character);
    await sleep(args.interval);
  }
  await sleep(args.settle);

  const report = await evaluate(`window.__v3(${JSON.stringify({ warmup: args.warmup })})`);
  const raw = await evaluate('window.__v3.raw()');
  const baselineAfter = await evaluate(CALIBRATE);
  const drift = baselineBefore === 0 ? 0 : Math.abs(baselineAfter - baselineBefore) / baselineBefore;
  notes.push(
    `校准基准（敲字后）：${baselineAfter}ms（相对敲字前变化 ${(drift * 100).toFixed(0)}%）` +
      `${drift > 0.3 ? ' —— ⚠️ 这一轮里负载就变了，读数可信度打折' : ''}`,
  );

  // 测量期间丢过前台 → `paint` 被渲染节流污染（实测出现过 1144ms / 1408ms），
  // 而这种数字**不像 0 那样一眼可疑**，只会被读成「这一档特别慢」。
  // 所以这一档不落盘，直接判失败。
  if (typeof report.可见性中断 === 'string' && report.可见性中断.startsWith('⚠️')) {
    console.error(JSON.stringify(report, null, 2));
    throw new Error(
      '测量期间页面被隐藏或遮挡过 —— 渲染管线被挂起，paint 读数不可信。\n' +
        '这一档不落盘。让窗口保持可见、不被其它窗口盖住，然后重跑。',
    );
  }

  const payload = {
    场景: { 文档: FILE.split(/[\\/]/).pop(), 视图: args.mode, 落点: args.at, 敲入: typed.join(''), 间隔ms: args.interval },
    校准基准ms: { 敲字前: baselineBefore, 敲字后: baselineAfter },
    备注: notes,
    报告: report,
    原始样本: raw,
  };

  if (args.out) {
    const outPath = resolve(args.out);
    mkdirSync(dirname(outPath), { recursive: true });
    writeFileSync(outPath, JSON.stringify(payload, null, 2), 'utf8');
  }
  console.log(JSON.stringify(payload, null, 2));

  const frameHz = Number(/(\d+)Hz/.exec(report.刷新率)?.[1] ?? 0);
  if (frameHz > 0 && frameHz < 30) {
    console.warn(`\n⚠️ 刷新率只有 ${frameHz}Hz —— 窗口很可能被遮挡或最小化了，Chromium 在节流渲染。这份读数不能采信。`);
  }
  if (report.样本数 === 0) {
    console.warn('\n⚠️ 一个样本都没有 —— 报告里的 0 不代表「很快」，请检查上面那几条备注。');
  }

  if (!args.keep) {
    // 把探针留下（下一次注入会重置），但**不要**留下任何脏状态：
    // 应用会把草稿写进会话存储，那会让下一次场景从「改过的文本」开始
    await evaluate(`try { localStorage.setItem('qingmo.session.v1', JSON.stringify({ tabs: [], activePath: null, root: null, droppedDrafts: 0 })); } catch (e) {}`);
  }
}

main().then(
  () => {
    socket?.close();
    process.exit(0);
  },
  (thrown) => {
    console.error(`\n❌ ${thrown instanceof Error ? thrown.message : String(thrown)}`);
    if (notes.length > 0) {
      console.error('\n跑到哪一步了：');
      for (const note of notes) console.error(`  · ${note}`);
    }
    socket?.close();
    process.exit(1);
  },
);
