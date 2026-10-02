/**
 * V2 的**自动化驱动器** —— 用 CDP 驱动真实 Chromium，量 mind-elixir 在 500 节点下的延迟。
 *
 * ## 一句话：它量的是「用户在这个规模上会做、而代价随规模增长的事」
 *
 * 判据写在 `docs/open-decisions.md` 的 V2 那行：「输入到反馈 ≤ 50ms」。
 * 但**照字面量「打字」会得到一个正确的假绿** ——
 * 内核的编辑框是一个原生 `contenteditable`，只在 `blur` 上挂监听，
 * 敲字这件事它一点都不参与，所以那个读数与节点数结构性地无关。
 * 这件事的证据与推理写在 `v2-probe.mjs` 的头部。
 *
 * 所以这里量九档，其中五档是用户路径、四档是归因：
 *
 * | 档 | 类别 | 为什么有它 |
 * |---|---|---|
 * | `type` | 用户路径 | 字面判据。量它是为了把「与节点数无关」**证明**出来，而不是猜 |
 * | `commit` | 用户路径 | 编辑框里回车 → `blur` → 提交 + `linkDiv()`：用户最常撞到的一下 |
 * | `child` / `sibling` / `remove` | 用户路径 | 结构操作，每次都要重画全部连线 |
 * | `refresh` / `linkDiv` / `getData` / `addChildDirect` | 归因 | 直接调 API，把「浏览器事件派发」与「内核自己的开销」分开 |
 *
 * 归因那一层是刻意的。V3 的教训是「测出来不合格之后要先归因再动手，而归因要量、不要猜」——
 * 只测「按 Tab 到出画面」这个总数，超标了也不知道该怪谁。
 *
 * ## 每次读数都配一条「这次操作真的发生了」的证据
 *
 * 这是 V3 那条「0 看起来像性能极好」的直接延伸。如果 Tab 没被内核接住，
 * `keydown` 的 `duration` 会是 0 或很小 —— **一个漂亮的假读数**。
 * 所以每一档之后都去数节点数 / 看编辑框 / 读 topic，
 * 对不上就把这一档标成**失败**，而不是「很快」。
 *
 * ## 环境纪律（全部来自 V3，一条都不是可选的）
 *
 * | 问题 | 不解决会怎样 |
 * |---|---|
 * | 窗口被遮挡 / 最小化时 Chromium 挂起渲染管线 | 读数窜到上千毫秒，而那**不像 0 那样一眼可疑** |
 * | 抢前台的杂页（`edge://…`） | 页面变成后台页，rAF 被挂起，读数全是 0 |
 * | `Runtime.evaluate` 抛错被当成「返回 undefined」 | 拿到的是一份看着正常的假数据 |
 * | 敲字走 `Input.insertText` 而不是 `dispatchKeyEvent` | 跳过键盘事件那一层，测的不是「打字」 |
 * | 绝对读数随机器负载整体漂移 | 判定会随负载翻面 —— 所以每轮都记校准基准 |
 *
 * 这些都在 `scripts/lib/cdp.mjs` 里，与 V3 共用同一份实现。
 *
 * ## 用法
 *
 * ```bash
 * node scripts/v2-run.mjs                                   # 88 与 500 节点 × 两种形状，全部九档
 * node scripts/v2-run.mjs --nodes 500 --shapes wide --ops child,linkDiv --reps 5
 * node scripts/v2-run.mjs --no-spawn                        # 连一个已经开着的调试端口
 * node scripts/v2-run.mjs --out .v2-out/run.json
 * ```
 *
 * 它**自己起浏览器、自己起静态服务**，一条命令跑完。压测台是
 * `scripts/v2-page.html` + 一个几十行的 http 服务（只喂 `v2-page.html` 与
 * mind-elixir 的 `dist/` 两个文件），所以这条路上没有打包器、没有 npm install，
 * 也就没有「构建配置改坏了导致读数变了」这一类噪声。
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';

import * as cdp from './lib/cdp.mjs';
import { createHarness } from './lib/harness.mjs';

const PROBE = readFileSync(new URL('./v2-probe.mjs', import.meta.url), 'utf8');
const PAGE = new URL('./v2-page.html', import.meta.url);

/**
 * 静态服务、起浏览器、探端口、收尾都在 `lib/harness.mjs`（V1 与 V2 共用）。
 *
 * **收尾必须走一条与 happy path 分开的幂等路径** —— 这条是踩出来的：
 * 原先收尾写在 happy path 末尾，一抛异常就被跳过，而 Node 的 http server
 * 句柄会钉住事件循环，进程永不退出：既不打错误、也不退出，只是静静占着
 * 9222 与 4318 两个端口。实测卡了 1 分 42 秒仍不退，而且这就是之前那两次
 * 「跑了很久只输出前缀、最后被 SIGTERM」的真相 —— 当时误判成「被测页面卡死」，
 * 还专门写了外部探针去问页面，其实页面是好的。
 */
const harness = createHarness();

/** 隔离的 node 工作区 —— ADR 0007：验证用的依赖不进本仓库 `package.json`。 */
const DEFAULT_PKG = 'C:/Users/canace/.workbuddy-ai/binaries/node/workspace/node_modules/mind-elixir';

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

const DEFAULTS = {
  pkg: DEFAULT_PKG,
  nodes: '88,500',
  shapes: 'balanced,wide',
  ops: 'type,commit,child,sibling,remove,refresh,linkDiv,getData,addChildDirect',
  reps: 10,
  typeRuns: 3,
  keys: 30,
  interval: 60,
  settle: 400,
  port: 4318,
  cdp: 9222,
  browser: null,
  out: null,
  keep: false,
  // 自起浏览器是默认行为；`--no-spawn` 由 `parseArgs` 通用处理（见 cdp.mjs 的注释）。
  spawn: true,
  fresh: false,
};

/**
 * 形状写成 `balanced@12` / `wide`。
 *
 * **分支因子是可调的，而且必须可调。** 实测 `linkDiv` 的成本由「顶层分支数」决定
 * （主循环里每轮写一次 `style.borderColor` 再读一次 `offsetWidth`，强制同步布局），
 * 而不是总节点数 —— 于是「500 节点卡不卡」**没有唯一答案**：
 * `balanced@6` 的 500 节点全部 ≤ 40ms，`wide`（499 个顶层分支）的同一份代码要 240ms 以上。
 * 只测一种形状，就会把一个「形状相关的结论」说成一个普遍结论。
 */
function parseShapes(text) {
  const shapes = String(text)
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
    .map((item) => {
      const [kind, branch] = item.split('@');
      if (kind === 'wide') return { kind: 'wide', branch: 0, label: 'wide' };
      if (kind === 'balanced') {
        const value = branch === undefined ? 6 : Number(branch);
        if (!Number.isFinite(value) || value < 2) throw new Error(`balanced 的分支因子要 ≥ 2，收到 ${String(branch)}`);
        return { kind: 'balanced', branch: value, label: `balanced@${value}` };
      }
      throw new Error(`不认识的形状：${item}（只认 balanced@k 与 wide）`);
    });
  if (shapes.length === 0) throw new Error('--shapes 一个有效值都没有');
  return shapes;
}

function parseArgv(argv) {
  // `--no-spawn`（关掉自起浏览器）由 parseArgs 通用处理 —— `spawn` 已在 DEFAULTS 里。
  // 以前这里是 `args.spawn = !argv.includes('--no-spawn')`，两个脚本各写一遍；
  // 抽进 parseArgs 的同时也把「`--preload-stencils` 被静默吞掉」那个坑一起堵了。
  const args = cdp.parseArgs(argv, DEFAULTS, ['reps', 'typeRuns', 'keys', 'interval', 'settle', 'port', 'cdp']);
  args.nodes = String(args.nodes)
    .split(',')
    .map((item) => Number(item.trim()))
    .filter((item) => Number.isFinite(item) && item > 0);
  args.shapes = parseShapes(args.shapes);
  args.ops = String(args.ops)
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item in OPS);
  if (args.nodes.length === 0) throw new Error('--nodes 一个有效值都没有');
  if (args.ops.length === 0) throw new Error('--ops 一个有效值都没有');
  return args;
}

// ---------------------------------------------------------------------------
// 九档操作
// ---------------------------------------------------------------------------

/**
 * 每一档的 `验证` 都返回 `null`（通过）或一句「为什么不成立」。
 *
 * **这一列是这个脚本里最值钱的东西。** 它把「读数」与「读数是在什么前提下量的」绑在一起：
 * 没有它，一次没被内核接住的按键会产出一个 0ms 的漂亮数字，
 * 而报告上看起来与「内核很快」一模一样。
 */
const OPS = {
  type: {
    说明: '编辑框里敲字（字面判据；预期与节点数无关）',
    类别: 'session',
  },
  commit: {
    说明: '编辑框里回车 → blur → 提交 topic + linkDiv()',
    类别: 'repeat',
    键: 'Enter',
    准备: 'edit',
    验证: (before, after) => {
      if (before.editBox !== true) return '准备阶段没有出现编辑框';
      if (after.editBox !== false) return '编辑框没消失（回车没被内核接住？）';
      if (after.firstChildTopic !== 'xyz') return `提交后 topic 是 ${JSON.stringify(after.firstChildTopic)}，期望 "xyz"`;
      if (after.firstChildRenderedText !== 'xyz') {
        return `重画后的文本是 ${JSON.stringify(after.firstChildRenderedText)}，期望 "xyz"`;
      }
      return null;
    },
  },
  child: {
    说明: 'Tab 加子节点',
    类别: 'repeat',
    键: 'Tab',
    准备: 'select',
    验证: (before, after) =>
      after.nodes === before.nodes + 1 ? null : `节点数 ${before.nodes} → ${after.nodes}，期望 +1`,
  },
  sibling: {
    说明: 'Enter 加兄弟节点',
    类别: 'repeat',
    键: 'Enter',
    准备: 'select',
    验证: (before, after) =>
      after.nodes === before.nodes + 1 ? null : `节点数 ${before.nodes} → ${after.nodes}，期望 +1`,
  },
  remove: {
    说明: 'Delete 删除选中节点',
    类别: 'repeat',
    键: 'Delete',
    准备: 'select',
    验证: (before, after) =>
      after.nodes === before.nodes - 1 ? null : `节点数 ${before.nodes} → ${after.nodes}，期望 −1`,
  },
  refresh: {
    说明: '直接调 refresh()（整篇重画；归因）',
    类别: 'direct',
    调用: 'opRefresh',
  },
  linkDiv: {
    说明: '直接调 linkDiv()（只重画连线；归因）',
    类别: 'direct',
    调用: 'opLinkDiv',
  },
  getData: {
    说明: '直接调 getDataString()（每次撤销入栈都要跑；归因）',
    类别: 'direct',
    调用: 'opGetData',
  },
  addChildDirect: {
    说明: '直接调 addChild()（不含浏览器事件开销；归因）',
    类别: 'direct',
    调用: 'opAddChildDirect',
    复位: true,
    验证: (before, after) =>
      after.nodes === before.nodes + 1 ? null : `节点数 ${before.nodes} → ${after.nodes}，期望 +1`,
  },
};

// ---------------------------------------------------------------------------
// 压测台的路由表（静态服务本身在 `lib/harness.mjs`）
// ---------------------------------------------------------------------------

const pageRoutes = (pkgDir) =>
  new Map([
    ['/', { body: readFileSync(PAGE), type: 'text/html; charset=utf-8' }],
    ['/index.html', { body: readFileSync(PAGE), type: 'text/html; charset=utf-8' }],
    [
      '/mind-elixir/MindElixir.js',
      { file: () => readFileSync(resolve(pkgDir, 'dist/MindElixir.js')), type: 'text/javascript; charset=utf-8' },
    ],
    [
      '/mind-elixir/MindElixir.css',
      { file: () => readFileSync(resolve(pkgDir, 'dist/MindElixir.css')), type: 'text/css; charset=utf-8' },
    ],
  ]);

// ---------------------------------------------------------------------------
// 单元格：一种形状 × 一个节点数
// ---------------------------------------------------------------------------

/**
 * 读浏览器自己的性能计数器。
 *
 * 它回答的是**机制**问题，而不是「快不快」：`type` 那一档在 `wide · 500 节点` 下
 * 越敲越慢（第 30 个字符要 350ms），而内核在那条路径上**一行代码都没跑** ——
 * 所以只能是浏览器的样式/布局在变贵。`LayoutCount` / `RecalcStyleDuration`
 * 的增量能把这句话从猜测变成读数。
 */
const METRIC_NAMES = [
  'LayoutCount',
  'RecalcStyleCount',
  'LayoutDuration',
  'RecalcStyleDuration',
  'ScriptDuration',
  'TaskDuration',
  'JSHeapUsedSize',
];

async function readMetrics() {
  const { metrics } = await cdp.send('Performance.getMetrics');
  const out = {};
  for (const metric of metrics) {
    if (METRIC_NAMES.includes(metric.name)) out[metric.name] = metric.value;
  }
  return out;
}

const diffMetrics = (before, after) =>
  Object.fromEntries(Object.keys(after).map((name) => [name, Number((after[name] - before[name]).toFixed(4))]));

/**
 * 页面必须**真的持有焦点**。
 *
 * `document.visibilityState === 'visible'` 不够 —— 窗口可以可见但被别的窗口盖住，
 * 那时 `#input-box` 收不到焦点，`Input.dispatchKeyEvent` 的字符会落到别处。
 * 第一次跑就丢过一档：`type` 的 30 个字符敲到一半编辑框 blur 了（内核在 blur 里
 * 把编辑框移除），整档作废。
 *
 * **调用时机是这个函数的一半。** 最初只在单元格开头调一次，结果 `balanced@75`
 * 那一档在开头通过了、到打字时已经失焦 —— 中间隔着 `mount()` 与 500ms 的刷新率测量，
 * 足够 Windows 把前台还给别的窗口。焦点是**「打字」这个动作**的前置条件，
 * 就得在打字之前查，不是在「这一档开始」之前查。
 */
async function ensureFocus(label) {
  for (let attempt = 0; attempt < 6; attempt += 1) {
    if (await cdp.evaluate('document.hasFocus()')) return;
    await cdp.send('Page.bringToFront');
    await cdp.sleep(400);
  }
  throw new Error(
    `${label}：页面拿不到焦点（document.hasFocus() 一直是 false）。\n` +
      '此时按键会落到别的窗口上，读数会全是 0。让这个窗口保持在前台再跑。',
  );
}

async function runCell({ shape, nodes, ops, args }) {
  const record = { 形状: shape.label, 节点数: nodes, 挂载ms: [], 操作: {} };

  const mount = async () => {
    const info = await cdp.evaluate(
      `window.__v2page.mount(${nodes}, ${JSON.stringify(shape.kind)}, ${shape.branch})`,
    );
    record.挂载ms.push(info.syncMs);
    return info;
  };

  // --- 前提检查（不通过就直接停，不给一份假数据）----------------------------
  const first = await mount();
  if (first.nodes !== nodes) {
    throw new Error(`造出来的节点数是 ${first.nodes}，期望 ${nodes} —— 构造器有问题，后面的读数没有意义`);
  }
  if (first.domTopics !== nodes) {
    throw new Error(`DOM 里只有 ${first.domTopics} 个 me-tpc，但数据里有 ${nodes} 个节点 —— 图没有真的画全`);
  }
  if (first.lines === 0) {
    throw new Error('连线 SVG 里一个元素都没有 —— linkDiv 没跑，后面量的是个空壳');
  }
  const info = await cdp.evaluate('window.__v2page.info()');
  if (info.hostBox.w < 100 || info.hostBox.h < 100) {
    throw new Error(`宿主尺寸是 ${info.hostBox.w}×${info.hostBox.h} —— 这不是一次真实的排版渲染，读数没有意义`);
  }
  if (info.visibility !== 'visible') {
    throw new Error(`页面不可见（visibilityState = ${info.visibility}）—— 渲染管线被挂起，读数会全是 0`);
  }
  record.初始化 = { ...first, 宿主尺寸: info.hostBox, 地图尺寸: info.mapBox };

  const hz = await cdp.evaluate('window.__v2.measureFrameHz(500)');
  record.刷新率 = hz;

  // --- 输入路径检查：分辨「输入没进去」与「进去了但没有条目」------------------
  //
  // 两个都要防：
  //
  // ① **必须轮询**，不能敲完等固定一小会儿就读。第一次跑踩到了：
  //    `wide · 500 节点` 下那一帧要 568ms，等 200ms 读出来 `beforeinput` 条目数是 0，
  //    于是判定写成「Event Timing 没产生条目」—— 而实际上只是**还没到那一帧**。
  //    一个「读数是对的、结论是错的」的典型：判定行会让人以为 Event Timing 不可用。
  //
  // ② **必须重试**，且每轮重试前重新要一次焦点。焦点是打字的前置条件，而它会在
  //    `mount()` + 500ms 刷新率测量这段时间里被 Windows 抢走（实测 `balanced@75`
  //    就是这样：单元格开头 `ensureFocus` 通过，打字时 `hasFocus=false`）。
  //    不重试的话，一次瞬时失焦会**杀掉整轮扫描** —— 八个形状测了零个。
  let arrived = false;
  let beforeInputEntries = 0;
  let afterProbe = null;
  const attempts = [];
  for (let attempt = 0; attempt < 3 && !arrived; attempt += 1) {
    await ensureFocus(`${shape.label} · ${nodes} 节点 · 打字前`);
    await cdp.evaluate('window.__v2page.startEditing()');
    await cdp.evaluate('window.__v2.reset()');
    await cdp.typeChar('z');
    const entryDeadline = Date.now() + 4000;
    for (;;) {
      const rawProbe = await cdp.evaluate('window.__v2.raw()');
      beforeInputEntries = (rawProbe.事件.beforeinput ?? []).length;
      if (beforeInputEntries > 0 || Date.now() > entryDeadline) break;
      await cdp.sleep(200);
    }
    afterProbe = await cdp.evaluate('window.__v2page.info()');
    arrived = typeof afterProbe.editBoxText === 'string' && afterProbe.editBoxText.includes('z');
    attempts.push({ 第: attempt + 1, arrived, hasFocus: afterProbe.hasFocus, editBox: afterProbe.editBox });
  }
  record.输入路径 = {
    字符到达编辑框: arrived,
    beforeinput条目数: beforeInputEntries,
    尝试: attempts,
    判定: arrived && beforeInputEntries > 0
      ? 'CDP 按键 → beforeinput，两条读数都可用'
      : arrived
        ? '按键到了编辑框，但 4 秒内没等到 beforeinput 条目 —— 只有「输入 → 帧」可用'
        : '⚠️ 按键没有到达编辑框 —— 后面所有读数都会是 0，而 0 看起来像性能极好',
  };
  if (!arrived) {
    throw new Error(
      'CDP 按键没有变成编辑框里的字符（已重试 3 轮，每轮重新要过焦点）。\n' +
        '此时所有读数都会是 0，而 0 看起来像「性能极好」，所以这里直接停。\n' +
        `排查方向：页面有没有拿到焦点（hasFocus=${afterProbe.hasFocus}）、` +
        `编辑框有没有起来（editBox=${afterProbe.editBox}）、` +
        `activeElement=${afterProbe.activeElement}`,
    );
  }

  // --- 逐档测量 -------------------------------------------------------------
  for (const opName of ops) {
    const op = OPS[opName];
    const cell = { 说明: op.说明, 类别: op.类别, 样本: [], 失败: [] };
    // 逐档打进度。只按「单元格」打的话，一个单元格里九档、几百次重复，
    // 卡住时看到的只是「跑了很久」—— 实测就这样白等了 19 分钟，
    // 最后还得靠外部探针去问页面才知道卡在哪。
    process.stderr.write(`\r  ${shape.label} · ${nodes} · ${opName} …`.padEnd(60));

    if (op.类别 === 'session') {
      // 敲字这一档跑**多轮**，每轮一条逐字符序列。
      //
      // 这是被实测逼出来的：`wide · 500 节点` 下第一轮是「前 6 个字符慢、
      // 之后稳定在 0.4ms」，另一轮是「前面稳定、后面越敲越慢到 350ms」——
      // **形态完全相反**。只看一次就会把其中一种形态当成结论。
      // 多轮 + 逐字符序列能立刻把「这是页面的性质」与「这是环境噪声」分开。
      for (let run = 0; run < args.typeRuns; run += 1) {
        await mount();
        const started = await cdp.evaluate('window.__v2page.startEditing()');
        if (started.box !== true || started.focused !== true) {
          cell.失败.push(`第 ${run + 1} 轮：编辑框没起来 / 没拿到焦点（box=${started.box} focused=${started.focused}）`);
          continue;
        }
        await cdp.evaluate('window.__v2.reset()');
        const metricsBefore = await readMetrics();
        const typed = [];
        let lostAt = null;
        let blurSnapshot = null;
        for (let i = 0; i < args.keys; i += 1) {
          const character = cdp.LETTERS[i % cdp.LETTERS.length];
          // 每个字符都单独打一次点 —— 否则这一档只有 Event Timing 一条读数，
          // 而万一 Chromium 不给 `beforeinput` 条目，这一档就完全空了。
          await cdp.evaluate(`window.__v2.arm(${JSON.stringify(character)})`);
          await cdp.typeChar(character);
          typed.push(character);
          // 逐字符打进度到 stderr。没有它的话，这种「越敲越慢」的档跑到一半
          // 被掐掉时**连卡在第几个字符都不知道** —— 而这正是最需要知道的一件事。
          //
          // 刻意**不**在这里多读一次探针：那会多一次 `Runtime.evaluate` 往返，
          // 占用主线程、还会改变下一个字符相对 vsync 的相位。
          process.stderr.write(`\r  type 第 ${run + 1} 轮 ${i + 1}/${args.keys} 「${character}」…`);
          // 每敲一个字都确认编辑框还在：整档敲完才发现丢了的话，
          // 只知道「丢了」、不知道**第几个字符丢的**，而「第 3 个丢」与
          // 「第 28 个丢」指向完全不同的原因（焦点被抢 / 内核自己关了）。
          if (lostAt === null && (await cdp.evaluate('document.getElementById("input-box") !== null')) !== true) {
            lostAt = i + 1;
            blurSnapshot = (await cdp.evaluate('window.__v2page.info()')).editBlur;
          }
          await cdp.sleep(args.interval);
        }
        await cdp.sleep(args.settle);
        const metricsAfter = await readMetrics();
        const raw = await cdp.evaluate('window.__v2.raw()');
        const after = await cdp.evaluate('window.__v2page.info()');
        cell.样本.push({
          序号: run + 1,
          敲入: typed.join(''),
          事件: raw.事件,
          到帧列表: raw.输入到帧.map((item) => item.ms),
          丢失于第几个字符: lostAt,
          失焦快照: blurSnapshot,
          之后: { editBoxText: after.editBoxText, editBox: after.editBox, activeElement: after.activeElement },
          资源增量: diffMetrics(metricsBefore, metricsAfter),
        });
        if (lostAt !== null) {
          const why = blurSnapshot === null
            ? '（没记到失焦快照）'
            : `失焦时 hasFocus=${blurSnapshot.hasFocus}、visibility=${blurSnapshot.visibility}、activeElement=${blurSnapshot.activeElement}`;
          cell.失败.push(
            `第 ${run + 1} 轮：敲到第 ${lostAt} 个字符时编辑框就没了（blur → 内核把它移除）${why} —— 后面那些字符没进内核`,
          );
        }
      }
    } else if (op.类别 === 'repeat') {
      for (let i = 0; i < args.reps; i += 1) {
        process.stderr.write(`\r  ${shape.label} · ${nodes} · ${opName} 第 ${i + 1}/${args.reps} 次 …`.padEnd(60));
        await mount();

        // `beforeInfo` **必须在「摆好姿势之后、按键之前」取**。
        // 第一次跑在 `mount()` 之后立刻取，于是 `commit` 档 11 次全判失败
        // （「准备阶段没有出现编辑框」）—— 因为那时编辑框确实还没被创建。
        // 一个纯粹由取值时机造成的假失败，比假绿好一点，但同样是噪声。
        let beforeInfo;
        if (op.准备 === 'edit') {
          const started = await cdp.evaluate('window.__v2page.startEditing()');
          if (started.box !== true || started.focused !== true) {
            cell.失败.push(`第 ${i + 1} 次：编辑框没起来 / 没拿到焦点（box=${started.box} focused=${started.focused}）`);
            continue;
          }
          for (const character of ['x', 'y', 'z']) {
            await cdp.typeChar(character);
            await cdp.sleep(30);
          }
          await cdp.sleep(80);
          beforeInfo = await cdp.evaluate('window.__v2page.info()');
        } else {
          await cdp.evaluate('window.__v2page.selectFirstLeaf()');
          const focused = await cdp.evaluate('window.__v2page.focusContainer()');
          if (focused !== true) {
            cell.失败.push(`第 ${i + 1} 次：地图容器没拿到焦点，按键会落到别处`);
            continue;
          }
          await cdp.sleep(60);
          beforeInfo = await cdp.evaluate('window.__v2page.info()');
        }

        await cdp.evaluate('window.__v2.reset()');
        await cdp.evaluate(`window.__v2.arm(${JSON.stringify(op.键)})`);
        await cdp.pressKey(op.键);
        await cdp.sleep(160);

        const raw = await cdp.evaluate('window.__v2.raw()');
        const after = await cdp.evaluate('window.__v2page.info()');
        const problem = op.验证(beforeInfo, after);
        if (problem !== null) {
          cell.失败.push(`第 ${i + 1} 次：${problem}`);
          continue;
        }
        cell.样本.push({
          序号: i + 1,
          事件: raw.事件,
          到帧列表: raw.输入到帧.map((item) => item.ms),
          节点数: after.nodes,
        });
      }
      if (cell.样本.length !== args.reps) {
        cell.失败.push(`只有 ${cell.样本.length}/${args.reps} 次通过验证`);
      }
    } else {
      for (let i = 0; i < args.reps; i += 1) {
        process.stderr.write(`\r  ${shape.label} · ${nodes} · ${opName} 第 ${i + 1}/${args.reps} 次 …`.padEnd(60));
        let beforeInfo;
        if (op.复位 === true) {
          await mount();
          beforeInfo = await cdp.evaluate('window.__v2page.info()');
        }
        const result = await cdp.evaluate(`window.__v2page.${op.调用}()`);
        if (op.验证 !== undefined) {
          const after = await cdp.evaluate('window.__v2page.info()');
          const problem = op.验证(beforeInfo, after);
          if (problem !== null) {
            cell.失败.push(`第 ${i + 1} 次：${problem}`);
            continue;
          }
        }
        cell.样本.push({ 序号: i + 1, ...result });
      }
    }

    record.操作[opName] = cell;
  }

  return record;
}

// ---------------------------------------------------------------------------
// 汇总
// ---------------------------------------------------------------------------

const percentile = (sorted, p) => {
  if (sorted.length === 0) return null;
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil(p * sorted.length) - 1));
  return sorted[index];
};

const summarize = (values) => {
  const sorted = [...values].sort((a, b) => a - b);
  return {
    样本: sorted.length,
    最小: percentile(sorted, 0),
    p50: percentile(sorted, 0.5),
    p95: percentile(sorted, 0.95),
    最大: sorted.length === 0 ? null : sorted[sorted.length - 1],
  };
};

/**
 * 把一档的所有样本折成一行读数。
 *
 * **Event Timing 与手动打点是两条独立的读数，不相加** —— 它们量的不是同一条线
 * （见 `v2-probe.mjs` 头部）。这里只把它们各自汇总出来，判断留给读报告的人。
 *
 * 样本的形状统一成三种键，好让汇总不必知道档位：
 * - `事件`：`{ 事件名: [duration] }`，Event Timing 来的；
 * - `到帧列表` / `到帧ms`：手动打点来的（前者是「一次操作产生多条」，后者是「一条」）；
 * - `syncMs`：直接调 API 的同步耗时。
 */
function foldOp(cell) {
  const paintByEvent = {};
  const sync = [];
  const toFrame = [];
  const manualFrame = [];
  for (const sample of cell.样本) {
    if (sample.事件 !== undefined) {
      for (const [name, list] of Object.entries(sample.事件)) (paintByEvent[name] ??= []).push(...list);
    }
    if (sample.syncMs !== undefined) sync.push(sample.syncMs);
    if (sample.到帧ms !== undefined && sample.到帧ms !== null) toFrame.push(sample.到帧ms);
    if (sample.到帧列表 !== undefined) manualFrame.push(...sample.到帧列表);
  }
  const byEvent = {};
  for (const [name, list] of Object.entries(paintByEvent)) byEvent[name] = summarize(list);
  return {
    通过次数: cell.样本.length,
    失败: cell.失败,
    按事件名: byEvent,
    同步耗时: summarize(sync),
    到帧: summarize(toFrame),
    输入到帧: summarize(manualFrame),
  };
}

function renderTable(cells) {
  const lines = [];
  const span = (summary) => (summary.样本 === 0 ? '—' : `${summary.p50} / ${summary.p95}`);
  const pad = (text, width) => (text.length >= width ? `${text} ` : text.padEnd(width));
  for (const cell of cells) {
    const mounts = summarize(cell.挂载ms);
    lines.push('');
    lines.push(`【${cell.形状} · ${cell.节点数} 节点】挂载 ${mounts.p50}ms（p50，共 ${mounts.样本} 次）· 刷新率 ${cell.刷新率.刷新率}`);
    lines.push(`  输入路径：${cell.输入路径.判定}`);
    lines.push(
      `  宿主 ${cell.初始化.宿主尺寸.w}×${cell.初始化.宿主尺寸.h}px` +
        ` · 地图 ${cell.初始化.地图尺寸?.w ?? '?'}×${cell.初始化.地图尺寸?.h ?? '?'}px` +
        ` · 连线元素 ${cell.初始化.lines} 个`,
    );
    lines.push('');
    lines.push(
      `  ${pad('档', 18)}${pad('通过/总', 9)}${pad('同步ms', 17)}${pad('到帧ms', 17)}${pad('输入→帧', 17)}输入→绘制（按事件名 p50/p95）`,
    );
    for (const [name, cellOp] of Object.entries(cell.操作)) {
      const folded = foldOp(cellOp);
      const total = folded.通过次数 + cellOp.失败.length;
      const events = Object.entries(folded.按事件名)
        .map(([eventName, summary]) => `${eventName} ${span(summary)}`)
        .join(' · ');
      lines.push(
        `  ${pad(name, 18)}${pad(`${folded.通过次数}/${total}`, 9)}` +
          `${pad(span(folded.同步耗时), 17)}${pad(span(folded.到帧), 17)}` +
          `${pad(span(folded.输入到帧), 17)}${events || '—'}`,
      );
      if (folded.失败.length > 0) {
        lines.push(`      ⚠️ 失败 ${folded.失败.length} 次：${folded.失败.slice(0, 3).join('；')}`);
      }
      // 敲字这一档的**逐字符序列**是最有信息量的一行：只看 p50/p95 会把
      // 「均匀地慢」和「越敲越慢」读成同一件事，而它们指向完全不同的原因。
      // 多轮并列是为了让「形态可复现吗」一眼可见。
      if (cellOp.类别 === 'session') {
        for (const sample of cellOp.样本) {
          const sequence = sample.到帧列表;
          if (!Array.isArray(sequence) || sequence.length === 0) continue;
          const head = sequence.slice(0, 6).join(' ');
          const tail = sequence.slice(-6).join(' ');
          const lost = sample.丢失于第几个字符 === null ? '' : ` ⚠️ 第 ${sample.丢失于第几个字符} 个字符起编辑框没了`;
          lines.push(`      第 ${sample.序号} 轮 输入→帧：前 6 个 ${head} ｜ 后 6 个 ${tail}${lost}`);
          const delta = sample.资源增量;
          if (delta !== undefined) {
            lines.push(
              `        浏览器计数增量：布局 ${delta.LayoutCount} 次 / ${delta.LayoutDuration?.toFixed(1)}ms` +
                ` · 样式重算 ${delta.RecalcStyleCount} 次 / ${delta.RecalcStyleDuration?.toFixed(1)}ms` +
                ` · 脚本 ${delta.ScriptDuration?.toFixed(1)}ms`,
            );
          }
        }
      }
    }
  }
  return lines.join('\n');
}

// ---------------------------------------------------------------------------
// 主流程
// ---------------------------------------------------------------------------

async function main() {
  const args = parseArgv(process.argv.slice(2));
  const notes = [];
  /**
   * 记一条备注，**同时立刻打到 stderr**。
   *
   * 这个脚本的前半段（起静态服务 → 起浏览器 → 连 CDP → 注入探针 → 量校准基准）
   * 原本只在**最后**统一打印，于是「卡在哪一步」完全看不见 ——
   * 实测卡过一次，等了 19 分钟只看到一行「跑了很久」，最后还得另写一个探针
   * 去问页面才知道页面是好的、卡的是脚本自己。
   * 一条能自己失败的超时 + 一条会立刻打出来的进度，缺一不可。
   */
  const note = (text) => {
    notes.push(text);
    process.stderr.write(`[v2] ${text.replace(/\n/g, '\n     ')}\n`);
  };
  const http = `http://127.0.0.1:${args.cdp}`;

  if (!existsSync(resolve(args.pkg, 'dist/MindElixir.js'))) {
    throw new Error(
      `找不到 mind-elixir：${args.pkg}\n` +
        '用 --pkg 指到它的包目录。装法（ADR 0007：不进本仓库 package.json）：\n' +
        '  cd C:/Users/canace/.workbuddy-ai/binaries/node/workspace && node.exe --version\n' +
        '  <npm> install mind-elixir',
    );
  }
  const pkgVersion = JSON.parse(readFileSync(resolve(args.pkg, 'package.json'), 'utf8')).version;
  note(`mind-elixir ${pkgVersion}（${args.pkg}）`);

  await harness.startServer({ port: args.port, routes: pageRoutes(args.pkg) });
  const url = `http://127.0.0.1:${args.port}/`;
  note(`压测台 ${url}`);

  if (args.spawn) {
    await harness.requireFreePort(http);
    const exe = harness.pickBrowser(args.browser);
    const started = harness.startBrowser({ exe, cdpPort: args.cdp, url });
    note(`浏览器 ${exe}（PID ${started.child.pid}，profile ${started.profile}）`);
  } else {
    note('--no-spawn：连一个已经开着的调试端口（页面应当是压测台）');
  }

  const version = await harness.waitForDebugPort(http);
  note(`调试端口 ${http} → ${version.Browser}`);
  note(`浏览器命令（--no-spawn 时照这条起）：\n  ${harness.browser?.exe ?? '<浏览器>'} --remote-debugging-port=${args.cdp} --user-data-dir=<空目录> --disable-features=CalculateNativeWinOcclusion --disable-backgrounding-occluded-windows --disable-renderer-backgrounding --disable-background-timer-throttling ${url}`);

  await cdp.connect({
    http,
    pagePattern: new RegExp(`^http://127\\.0\\.0\\.1:${args.port}/`),
    hint: `压测台没起来？直接用浏览器打开 ${url}，或让本脚本自己起（去掉 --no-spawn）。`,
  });

  if (args.fresh) {
    await cdp.send('Page.reload', { ignoreCache: false });
    await cdp.sleep(1500);
    note('已重载页面（--fresh）');
  }

  await cdp.waitFor('window.__v2ready === true', { label: '压测台就绪', timeout: 30000 });
  await cdp.waitFor('document.visibilityState === "visible"', { label: '页面可见', timeout: 10000 });
  await cdp.send('Performance.enable');

  await ensureFocus('开跑前');

  // 上一轮留下的探针先卸掉：它的 PerformanceObserver 还在观察，
  // 重新注入一份会让**每个事件被观察两次**（新报告只读新的桶，不会重复计数，
  // 但白留一个观察器，且容易让人误以为读数被算了两次）。
  await cdp.evaluate('window.__v2 && window.__v2.stop && window.__v2.stop()');
  await cdp.evaluate(PROBE);
  note('探针已注入');

  const baselineBefore = await cdp.evaluate(cdp.CALIBRATE);
  note(`校准基准（开跑前）：${baselineBefore}ms —— 跨批次比较要按它归一化`);

  const cells = [];
  const failedCells = [];
  for (const shape of args.shapes) {
    for (const nodes of args.nodes) {
      process.stderr.write(`\r跑 ${shape.label} · ${nodes} 节点 …`);
      try {
        await ensureFocus(`${shape.label} · ${nodes} 节点`);
        cells.push(await runCell({ shape, nodes, ops: args.ops, args }));
      } catch (error) {
        // 一个单元格失败**不**该杀掉整轮扫描。
        //
        // 实测：八档的形状扫描在第一档（`balanced@75`）因为一次瞬时失焦抛了异常，
        // 于是「跑了八个形状」变成「一个都没测」。失败必须留下痕迹（下面会单独列出来，
        // 且不进表格），但已经测到的七个形状是有效资产，不能一起丢掉。
        failedCells.push({
          形状: shape.label,
          节点数: nodes,
          错误: error instanceof Error ? error.message : String(error),
        });
        note(`❌ ${shape.label} · ${nodes} 节点整档失败：${error instanceof Error ? error.message.split('\n')[0] : String(error)}`);
      }
    }
  }
  process.stderr.write('\r');

  const baselineAfter = await cdp.evaluate(cdp.CALIBRATE);
  const drift = baselineBefore === 0 ? 0 : Math.abs(baselineAfter - baselineBefore) / baselineBefore;
  notes.push(
    `校准基准（跑完）：${baselineAfter}ms（相对开跑前变化 ${(drift * 100).toFixed(0)}%）` +
      `${drift > 0.3 ? ' —— ⚠️ 这一轮里负载就变了，跨形状的绝对比较要打折' : ''}`,
  );

  const payload = {
    元信息: {
      时间: new Date().toISOString(),
      'mind-elixir': pkgVersion,
      节点数: args.nodes,
      形状: args.shapes.map((item) => item.label),
      档位: args.ops,
      重复次数: args.reps,
      敲入字符数: args.keys,
      校准基准ms: { 开跑前: baselineBefore, 跑完: baselineAfter },
      页面: url,
    },
    单元格: cells,
    失败单元格: failedCells,
    备注: notes,
  };

  console.log('\n=== V2 · mind-elixir 压测 ===');
  console.log(notes.join('\n'));
  console.log(renderTable(cells));

  if (failedCells.length > 0) {
    console.log(`\n【整档失败：${failedCells.length}】—— 下面这些形状没有读数，不是「通过」`);
    for (const item of failedCells) {
      console.log(`  ❌ ${item.形状} · ${item.节点数} 节点\n     ${item.错误.replace(/\n/g, '\n     ')}`);
    }
  }

  if (args.out !== null) {
    const outPath = resolve(args.out);
    mkdirSync(dirname(outPath), { recursive: true });
    writeFileSync(outPath, JSON.stringify(payload, null, 2), 'utf8');
    console.log(`\n原始读数已落盘：${outPath}`);
  }

  harness.keepBrowserFlag = args.keep;
}

// 收尾**不写在 `main()` 里**。main 里提前 throw 的地方有七八处（找不到包、找不到浏览器、
// 焦点拿不到…），靠「每一处都记得收尾」是不可能守住的 —— 实测就是这么漏的。
// 成败两条路都走同一个幂等的 `teardown`。
main()
  .then(() => harness.teardown({ keepBrowser: harness.keepBrowserFlag }))
  .catch(async (thrown) => {
    console.error('\n❌', thrown instanceof Error ? thrown.message : thrown);
    if (thrown instanceof Error && thrown.cause) console.error('   原因：', thrown.cause);
    await harness.teardown({ keepBrowser: harness.keepBrowserFlag });
    process.exitCode = 1;
  });
