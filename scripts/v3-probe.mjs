/**
 * V3（长文档输入延迟）的测量探针。
 *
 * **这不是一个模块，不要 import 它。** 整份文件是一段自执行脚本：
 * 打开 DevTools 的 Console，把下面的内容整段粘进去回车即可
 * （Chromium 第一次会拦一下，让你先输入 `allow pasting` —— 那是浏览器的
 * 通用防护，防的是「有人骗你粘贴代码」，与本文件无关）。
 *
 * ## 为什么必须这么做
 *
 * V3 的判据是「1 万字以上文档，输入延迟 ≤ 50ms」，而它没法用单测跑：
 *
 * - jsdom 没有布局引擎，也没有真实的绘制与计时（项目里所有单测都跑在它上面）；
 * - 用 `dispatchEvent(new KeyboardEvent('keydown'))` 这类**合成事件**插不进字符 ——
 *   ProseMirror 靠 `beforeinput` + `MutationObserver` 派发事务，合成事件走不到那条链；
 * - 项目里没有 benchmark 脚手架。
 *
 * （**但「自动化打字」并非绝对不通**：Playwright 的 `keyboard.type()` 走的是 CDP 的
 * `Input.dispatchKeyEvent`，是浏览器眼中的**可信输入**，会完整走一遍输入管线。
 * 代价是要装一份 Chromium，而且无头模式下的绘制读数不可信 —— 本项目还没走这条路。）
 *
 * ## 四个读数，各自回答不同的问题
 *
 * | 读数 | 含义 | 误差 |
 * |---|---|---|
 * | `输入延迟` | **主判据。** `beforeinput` → **出字那一帧**：用户实际感知的输入延迟 | **无**（Event Timing API 的定义） |
 * | `sync` | `beforeinput` → `input`：浏览器完成原生插入 + 内核在这一段里同步跑掉的成本 | **无** |
 * | `paint` | `input` → 出字那一帧 | **无** |
 * | `longtasks` | 主线程上 > 50ms 的独立任务 | **无** |
 *
 * `sync` 与 `paint` 是**分解**，但**不能相加** —— 它们量的是不同的那次绘制：
 * `beforeinput` 的条目量到「字上屏」那一帧，`input` 的条目量到它**之后**的下一次绘制
 * （内核把原生插入的文本改写回自己的模型之后）。实测 `sync 34 + paint 40 = 74ms`，
 * 而同一批数据的 `输入延迟` 只有 48ms —— 差的那 26ms 是内核的收尾工作，
 * 用户在 48ms 时已经看见字了。
 *
 * ## 2026-09-30 修正：主判据从 `paint` 改成 `beforeinput → 出字`
 *
 * 初版拿 `paint`（`input` → 下次绘制）当判据。**那是低估**：
 * `input` 事件在 `beforeinput` 之后才发，所以 `paint` 把
 * 「浏览器插入 + 内核同步响应」这一整段（实测 15–30ms，富文本档）漏在了外面。
 *
 * 现在直接量 `beforeinput` 的 Event Timing 条目 —— 它的 `duration` 定义就是
 * 「beforeinput → 它处理完之后的下一次绘制」，正好是要的那个数，
 * 而且**不需要我们自己拼两段**。`sync` / `paint` 降级为归因用的分解。
 *
 * ## `paint` / `输入延迟` 的取值被 Chromium 粗化到 8ms 的整数倍
 *
 * 实测拿到的是 16 / 24 / 32 / 40 / 64 / 88 —— **全是 8 的整数倍**。
 * 这是 Chromium 对 Event Timing `duration` 做的粗化（防计时侧信道），不是巧合。
 * 后果：**「≤ 50ms」实际等价于「≤ 48ms」**，而且 p50 与 p95 常常是同一个数
 * （24/24、40/40）—— 百分位在这个粒度上不总是有分辨力，读数要连着看。
 *
 * ## 量程下限是 16ms
 *
 * 另一个观察器（`durationThreshold: 16`）收到的条目数与 `0` 的**完全相同**，
 * 说明 **Chromium 把 `durationThreshold` 钳到了最小 16ms**（规范里就写的 16）。
 * 于是源码视图（CodeMirror）报出来的 16ms 是**顶到量程**，
 * 真实值 **≤ 16ms**，不能读成「正好 16ms」。
 *
 * ## 绝对读数会随机器背景负载整体漂移 —— 判据必须写明测量环境
 *
 * 实测：同一份文档、同一个视图、同一台机器，隔了二十分钟再跑，
 * `sync` 从 18.3ms 变成 34ms、源码档从 4.1ms 变成 7.3ms ——
 * **两个完全不同的内核按同一个比例（约 1.8×）一起变慢**，说明漂的是环境不是代码。
 * 原因是这台机器上同时跑着几套语言服务和一个前端 dev server。
 *
 * 这对 V3 是要害：判据是「≤ 50ms」，而读数能整体翻倍 —— **判定会随负载翻面**。
 * 所以：
 *
 * 1. **同批次内部**的横向比较（视图之间、落点之间）是可信的；
 * 2. 跨批次比绝对值必须**按校准基准归一化**（见 `v3-run.mjs` 的 `CALIBRATE`）；
 * 3. 记录结论时要**写上当时的校准基准**，否则这个数字没有意义。
 *
 * ## 事件计数：分辨「输入没进去」和「没有条目」
 *
 * 报告里的 `事件计数` 摆着四个数。不相等时要先解释清楚再采信读数：
 *
 * - `beforeinput` 少于实际敲入键数 → **输入压根没全进去**，这一档不成立；
 * - `beforeinput` 远多于 `input` → 内核拦掉了原生插入（`preventDefault`），
 *   这类内核测不到 `sync`；
 * - `输入延迟样本` 少于 `input` → Event Timing 漏了条目（多半被 16ms 下限挡掉），
 *   此时读数**偏向上界**，是保守的，可以采信。
 *
 * （实测踩过：源码视图敲 60 键只收到 33 个 `input`。补上这几个计数之前，
 * 这个现象与「输入没进去」完全无法区分。）
 *
 * ## 2026-09-30 修正：`paint` 改用 Event Timing API
 *
 * 初版用**双 `requestAnimationFrame`** 估「画完了没有」，并在注释里写「偏差 0–16.7ms」。
 * **那句话是错的，第一次实测把它照出来了**：rAF 回调跑在绘制**之前**，所以「双 rAF」意味着
 * 至少等到第 2 帧，而 `performance.now()` 读到的是第 2 帧回调的时刻 ——
 * 读数 = 实际耗时 + **1~2 帧**。60Hz 下就是 +16.7~33.3ms 的**固定底噪**。
 *
 * 第一次实测拿到 `p50 = 35.5ms`，而同一批数据的 `sync.p50` 只有 `17.2ms`、
 * 83 次输入只产生 4 条长任务 —— 35.5 几乎就是这个底噪本身。
 * 一份实际只要几毫秒的输入，被报成了「超出 50ms 判据」。
 *
 * 现在改用 **Event Timing API**（`PerformanceObserver` 的 `event` 条目）：
 * `entry.duration` 的定义就是「从事件时间戳到它处理完之后的下一次绘制」——
 * 这正是「输入延迟」的标准定义，也是 INP 指标的基础，**没有帧对齐误差**。
 * 双 rAF 保留为环境不支持时的回退，报告里会标明用的是哪一条路。
 *
 * ## 怎么用
 *
 * 1. 打开一份 1 万字以上的文档（`scripts/make-v3-longdoc.mjs` 能生成）；
 * 2. 把光标放到**文档中部**（开头与结尾的成本不同，这本身是一条线索）；
 * 3. 粘贴本文件内容回车 —— 控制台会打印「已挂载 N 个可编辑区」；
 * 4. 连续敲 30 个字符（别停太久，也不要敲得太快盖过测量）；
 * 5. 输入 `__v3()` 回车看统计。
 *
 * 三档视图都要测一遍（富文本 / 源码 / 分屏）。切视图之后 DOM 会重建，
 * 探针挂的监听会失效 —— **切完再粘一次探针**，或者调 `__v3.attach()` 重挂。
 *
 * ## 中文输入法怎么算
 *
 * 用拼音输入法时，按字母阶段发的是 `inputType: 'insertCompositionText'`
 * （DOM 里出现的是组合中的拼音），上屏时才是 `insertText`。两者都是真实成本，
 * 但组合阶段的 DOM 更新比上屏小得多。探针按 `inputType` 分开统计，所以不用自己拆。
 *
 * ## 报告里的「归因」那一行
 *
 * `sync` 达标而 `paint` 不达标，和两者都不达标，**指向完全不同的修改方向**：
 * 前者是样式/布局/绘制（或掉帧）的问题，后者才是 `plugin.ts` 里那几条 O(n) 的问题。
 * 这一行是自动判出来的，省得对着一堆百分位猜。
 */

(() => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    console.warn('[v3] 这段要在浏览器/WebView 的 Console 里跑，不是在 node 里');
    return;
  }

  /** 每条输入一条记录：`{ at, sync, paint, inputType, gap }`。 */
  const samples = [];
  /** `input` 事件留下的标记，用来给 Event Timing 的 entry 补上 `inputType` 与 `sync`。 */
  const marks = [];
  /**
   * `beforeinput` 留下的标记。它与 `marks` 分开，因为要配的是**另一个**事件类型的条目。
   *
   * 为什么要单独配 `beforeinput` 的条目：`sync`（`beforeinput → input`）与
   * `paint`（`input → 下次绘制`）是**两段不重叠的时间**，真正的「按键到出字」
   * 是两者之和。只用 `paint` 判定会**低估**延迟。而 Event Timing 对
   * `beforeinput` 也产生条目，它的 `duration` 定义就是「beforeinput → 出字那一帧」——
   * 这正是要量的那个数，且没有帧对齐误差。
   */
  const armedQueue = [];
  /** `beforeinput → 出字那一帧` 的样本。**这是主判据**，`sync` / `paint` 是它的分解。 */
  const latencies = [];
  const longtasks = [];
  const frameGaps = [];

  let mounted = 0;
  let usingEventTiming = false;
  /** 配不到 `input` 标记、被丢弃的 Event Timing 条目数（别的元素上的输入 / 历史条目）。 */
  let orphans = 0;
  let lastBeforeInputAt = 0;
  let armed = null;

  /**
   * 三个计数，用来分辨「输入没进去」和「输入进去了但没产生条目」。
   *
   * 这两件事的表现**完全一样**（样本数偏少），但含义相反：
   * 前者说明这一档的读数不可信，后者只是 Event Timing 少了些条目。
   * 实测踩过：源码视图（CodeMirror）敲 60 键只收到 33 个 `input` ——
   * 不把这两个数摆出来，就没法判断该不该采信。
   */
  let beforeInputCount = 0;
  let inputCount = 0;
  /** 配不到 `beforeinput` 标记的条目数（别处产生的 beforeinput）。 */
  let orphanBeforeInputEntries = 0;

  /**
   * 测量期间页面被隐藏/遮挡了多久。
   *
   * 这是**必须记**的一项：页面不可见时 Chromium 会挂起渲染管线，
   * 而 `paint` 的读数会窜到几百甚至上千毫秒 —— 那种数字**不像 0 那样一眼可疑**，
   * 只会被误读成「这一档特别慢」。实测踩过：跑到一半窗口被最小化，
   * 拿到 `paint.p95 = 1144ms`、`max = 1408ms`，而同一份文档正常时是 32ms。
   */
  let hiddenSince = 0;
  const hiddenPeriods = [];
  const onVisibilityChange = () => {
    if (document.hidden) {
      hiddenSince = performance.now();
    } else if (hiddenSince !== 0) {
      hiddenPeriods.push(Number((performance.now() - hiddenSince).toFixed(1)));
      hiddenSince = 0;
    }
  };
  document.addEventListener('visibilitychange', onVisibilityChange);

  const percentile = (values, q) => {
    if (values.length === 0) return 0;
    const ordered = [...values].sort((a, b) => a - b);
    const index = Math.min(ordered.length - 1, Math.floor(ordered.length * q));
    return ordered[index];
  };

  const describe = (values) => ({
    n: values.length,
    p50: Number(percentile(values, 0.5).toFixed(1)),
    p95: Number(percentile(values, 0.95).toFixed(1)),
    max: Number((values.length === 0 ? 0 : Math.max(...values)).toFixed(1)),
  });

  /** 取走与某个 `startTime` 最接近的标记（Event Timing 的 entry 是异步到的）。 */
  const takeNearest = (queue, startTime) => {
    let bestIndex = -1;
    let bestDelta = Number.POSITIVE_INFINITY;
    for (let i = 0; i < queue.length; i += 1) {
      const delta = Math.abs(queue[i].at - startTime);
      if (delta < bestDelta) {
        bestDelta = delta;
        bestIndex = i;
      }
    }
    if (bestIndex === -1 || bestDelta > 1) return null;
    return queue.splice(bestIndex, 1)[0];
  };

  const takeMark = (startTime) => takeNearest(marks, startTime);

  // -------------------------------------------------------------------------
  // 主测量：Event Timing API
  // -------------------------------------------------------------------------

  let eventObserver = null;
  try {
    eventObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        // 主判据：beforeinput → 出字那一帧
        if (entry.name === 'beforeinput') {
          const armedEntry = takeNearest(armedQueue, entry.startTime);
          if (armedEntry === null) {
            orphanBeforeInputEntries += 1;
            continue;
          }
          latencies.push({
            at: entry.startTime,
            latency: entry.duration,
            inputType: armedEntry.inputType,
            gap: armedEntry.gap,
          });
          continue;
        }

        if (entry.name !== 'input') continue;
        const mark = takeMark(entry.startTime);
        // 配不到标记的 entry **一律丢弃**。它是两种东西之一：别的元素上的输入事件
        // （对话框、搜索框 —— 我们的监听只挂在 contenteditable 上），
        // 或者 `buffered` 捞进来的历史条目。
        //
        // 上一版把它当成样本收下了（`inputType: 'unknown'`、`gap: 0`），于是它
        // **躲过了冷启动过滤**（`gap: 0` 落在窗口内），把 `paint.p95` 从 64ms
        // 污染成 104ms —— 而它的 `sync.n` 是 0，本该一眼看出不对。
        if (mark === null) {
          orphans += 1;
          continue;
        }
        samples.push({
          at: entry.startTime,
          // 定义即「事件 → 下一次绘制」，这就是输入延迟
          paint: entry.duration,
          // 内核 JS 成本：只有自己算得到，Event Timing 不提供
          sync: mark.sync,
          inputType: mark.inputType,
          gap: mark.gap,
        });
      }
    });
    // `durationThreshold` 默认是 104ms（只报慢的），必须显式设成 0 才拿得到全部样本。
    // `buffered: false`：只要安装之后产生的条目，历史条目配不到标记、只会变成噪声。
    eventObserver.observe({ type: 'event', buffered: false, durationThreshold: 0 });
    usingEventTiming = true;
  } catch {
    usingEventTiming = false;
  }

  /**
   * 量程下限核对：另开一个**只报 duration ≥ 16ms** 的观察器，和上面那个比数量。
   *
   * 实测所有取值都是 16 的整数倍、从没出现过 8 或 0，怀疑 Chromium 把
   * `durationThreshold` 钳到了最小 16ms（Event Timing 规范里就写了 16 这个下限）。
   * 两个观察器数量相同 = 确认下限是 16ms，那么**源码/分屏档报出来的 16ms
   * 就是「顶到量程下限」，真实值 ≤ 16ms**，不能读成「正好 16ms」。
   */
  let floorObserver = null;
  let floorCount = 0;
  try {
    floorObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.name === 'input' || entry.name === 'beforeinput') floorCount += 1;
      }
    });
    floorObserver.observe({ type: 'event', buffered: false, durationThreshold: 16 });
  } catch {
    floorObserver = null;
  }

  // -------------------------------------------------------------------------
  // 输入监听：拿 inputType，算内核成本
  // -------------------------------------------------------------------------

  const onBeforeInput = (event) => {
    beforeInputCount += 1;
    const now = performance.now();
    const gap = lastBeforeInputAt === 0 ? 0 : now - lastBeforeInputAt;
    lastBeforeInputAt = now;
    const inputType = event.inputType || 'unknown';
    armed = { at: now, timeStamp: event.timeStamp, inputType, gap };
    // 同一时刻也排进 `armedQueue`：`beforeinput` 的 Event Timing 条目要用 `event.timeStamp`
    // 去配（`performance.now()` 与它同源，但条目给的是事件自己的时间戳）。
    armedQueue.push({ at: event.timeStamp, inputType, gap });
    if (armedQueue.length > 400) armedQueue.shift();
  };

  const onInput = (event) => {
    inputCount += 1;
    const started = armed;
    armed = null;

    const inputType = (started === null ? '' : started.inputType) || event.inputType || 'unknown';
    const sync = started === null ? null : performance.now() - started.at;
    const gap = started === null ? 0 : started.gap;

    if (usingEventTiming) {
      // 交给 Event Timing 的 entry 去配 paint，这里只留标记
      marks.push({ at: event.timeStamp, sync, inputType, gap });
      if (marks.length > 400) marks.shift();
      return;
    }

    // 回退路径：没有 Event Timing 时只能自己用双 rAF 估「画完了没有」。
    // 读数含 1~2 帧底噪，报告里会标出来。
    if (started === null) return;
    if (typeof requestAnimationFrame !== 'function') {
      samples.push({ at: event.timeStamp, paint: sync, sync, inputType, gap });
      return;
    }
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        samples.push({ at: event.timeStamp, paint: performance.now() - started.at, sync, inputType, gap });
      });
    });
  };

  /** 已经挂过监听的根，防止重复 `attach()` 把每条输入算两遍。 */
  let attachedRoots = new WeakSet();

  const attach = () => {
    // 只挂真正可编辑的：分屏右栏是 `contenteditable="false"`，会被排除
    const roots = [...document.querySelectorAll('[contenteditable="true"]')];
    for (const root of roots) {
      if (attachedRoots.has(root)) continue;
      attachedRoots.add(root);
      // 捕获阶段：编辑器内部可能 stopPropagation，捕获能保证我们拿到
      root.addEventListener('beforeinput', onBeforeInput, true);
      root.addEventListener('input', onInput, true);
    }
    mounted = roots.length;
    console.log(`[v3] 已挂载 ${mounted} 个可编辑区${mounted === 0 ? '（没有？先打开一份文档）' : ''}`);
    return mounted;
  };

  // -------------------------------------------------------------------------
  // 旁证：长任务与帧率
  // -------------------------------------------------------------------------

  let longtaskObserver = null;
  try {
    longtaskObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) longtasks.push(Number(entry.duration.toFixed(1)));
    });
    longtaskObserver.observe({ entryTypes: ['longtask'] });
  } catch {
    // 环境不支持就不测这一项，别让它把整个探针拖垮
    longtaskObserver = null;
  }

  /**
   * 估刷新率。**它决定「掉一帧」值多少毫秒** —— 同样掉 4 帧，
   * 60Hz 是 67ms、144Hz 只有 28ms，结论会完全不同。
   */
  let lastFrameAt = 0;
  let frameProbe = 0;
  const sampleFrame = (timestamp) => {
    if (lastFrameAt !== 0) frameGaps.push(timestamp - lastFrameAt);
    lastFrameAt = timestamp;
    if (frameGaps.length < 60) frameProbe = requestAnimationFrame(sampleFrame);
  };
  if (typeof requestAnimationFrame === 'function') frameProbe = requestAnimationFrame(sampleFrame);

  const frameMs = () => (frameGaps.length === 0 ? 16.7 : Number(percentile(frameGaps, 0.5).toFixed(2)));

  // -------------------------------------------------------------------------
  // 报告
  // -------------------------------------------------------------------------

  /** `warmup` 用来剔除「停了很久之后的第一键」—— 那一次含冷启动成本。 */
  const report = ({ warmup = 1000 } = {}) => {
    const kept = samples.filter((s) => s.gap >= 0 && s.gap <= warmup);
    const dropped = samples.length - kept.length;

    const syncValues = kept.map((s) => s.sync).filter((v) => v !== null);
    const paintValues = kept.map((s) => s.paint);
    const sync = describe(syncValues);
    const paint = describe(paintValues);
    // 主判据：与 `kept` 用同一个冷启动窗口过滤
    const latencyValues = latencies.filter((s) => s.gap >= 0 && s.gap <= warmup).map((s) => s.latency);
    const latency = describe(latencyValues);
    const latencyOver = latencyValues.filter((v) => v > 50).length;
    /** threshold=0 那个观察器一共收到多少条条目（含配不上的）。用来和 16ms 观察器对账。 */
    const totalEntries = samples.length + orphans + latencies.length + orphanBeforeInputEntries;
    const ms = frameMs();

    const byType = {};
    for (const type of new Set(kept.map((s) => s.inputType))) {
      const subset = kept.filter((s) => s.inputType === type);
      byType[type] = {
        次数: subset.length,
        sync: describe(subset.map((s) => s.sync).filter((v) => v !== null)),
        paint: describe(subset.map((s) => s.paint)),
      };
    }

    return {
      测量方法: usingEventTiming
        ? 'Event Timing API（无帧对齐误差）'
        : `双 requestAnimationFrame（含 ${ms}~${(ms * 2).toFixed(1)}ms 底噪，结论不可直接采信）`,
      刷新率: frameGaps.length === 0 ? '未测到' : `约 ${Math.round(1000 / ms)}Hz（一帧 ${ms}ms）`,
      样本数: kept.length,
      剔除的冷启动键: dropped,
      丢弃的无关条目: orphans,
      事件计数: {
        beforeinput: beforeInputCount,
        input: inputCount,
        配对成功的条目: kept.length,
        输入延迟样本: latencyValues.length,
        配不上标记的条目: { input: orphans, beforeinput: orphanBeforeInputEntries },
        说明: '输入延迟样本数明显少于敲入键数 = Event Timing 漏了条目（多半是被 16ms 量程下限挡掉），不是输入没进去',
      },
      量程下限核对:
        floorObserver === null
          ? '这个环境不支持第二个观察器，没法核对'
          : floorCount !== totalEntries
            ? `threshold=0 收到 ${totalEntries} 条、threshold=16 收到 ${floorCount} 条 → 确有 <16ms 的条目被下限挡掉了`
            : latency.p50 <= 16
              ? `下限确认是 16ms（两个观察器都收到 ${floorCount} 条）。本档 输入延迟 p50 就是 16ms = 顶到量程，**真实值 ≤ 16ms，不是「正好 16ms」**`
              : `下限确认是 16ms（两个观察器都收到 ${floorCount} 条）。本档读数（输入延迟 p50 ${latency.p50}ms）在这之上，不受下限影响`,
      可见性中断:
        hiddenPeriods.length === 0 && !document.hidden
          ? '无（全程可见，读数可用）'
          : `⚠️ 发生了 ${hiddenPeriods.length + (document.hidden ? 1 : 0)} 次、累计约 ${(hiddenPeriods.reduce((a, b) => a + b, 0) + (document.hidden ? 1 : 0)).toFixed(0)}ms —— 渲染被挂起，**这一档的读数不可信**`,
      // 主判据：beforeinput → 出字那一帧。`sync` 与 `paint` 是它的两段分解。
      输入延迟: latency,
      sync,
      paint,
      分解说明:
        `sync（beforeinput→input）= ${sync.p50}ms，paint（input→那次绘制）= ${paint.p50}ms。` +
        '**两者不能相加** —— 它们量的是不同的那次绘制：' +
        '`beforeinput` 的条目量到的是**字上屏**的那一帧，而 `input` 的条目量到的是它之后的下一次绘制' +
        '（内核把原生插入的文本改写回自己的模型之后）。所以「用户什么时候看见字」看 `输入延迟`，' +
        '`sync` 与 `paint` 只用来判断成本落在哪一段。',
      '超过 50ms 的输入': `${latencyOver} / ${latencyValues.length}（${((latencyOver / Math.max(1, latencyValues.length)) * 100).toFixed(0)}%）`,
      长任务: { 条数: longtasks.length, 最长: longtasks.length === 0 ? 0 : Math.max(...longtasks) },
      按inputType分组: byType,
      归因:
        latencyValues.length === 0
          ? '还没有样本'
          : sync.p95 > 50
            ? '内核在这一段里同步跑的成本就超了 —— 对着 plugin.ts 的 O(n) 清单逐条查'
            : latency.p95 > 50
              ? '内核同步部分达标、慢在「input → 绘制」这一段 —— 瓶颈在样式/布局/绘制（或掉帧），不是 JS'
              : '两项都达标',
      结论:
        latencyValues.length === 0
          ? '还没有样本'
          : latency.p95 <= 50
            ? `✅ 输入延迟 p95 = ${latency.p95}ms，在判据内`
            : `❌ 输入延迟 p95 = ${latency.p95}ms，超出判据`,
    };
  };

  const api = {
    attach,
    report,
    reset: () => {
      samples.length = 0;
      marks.length = 0;
      armedQueue.length = 0;
      latencies.length = 0;
      longtasks.length = 0;
      frameGaps.length = 0;
      orphans = 0;
      orphanBeforeInputEntries = 0;
      beforeInputCount = 0;
      inputCount = 0;
      floorCount = 0;
      hiddenPeriods.length = 0;
      hiddenSince = 0;
      lastBeforeInputAt = 0;
      armed = null;
      console.log('[v3] 已清空');
    },
    stop: () => {
      eventObserver?.disconnect();
      floorObserver?.disconnect();
      longtaskObserver?.disconnect();
      if (typeof cancelAnimationFrame === 'function' && frameProbe !== 0) cancelAnimationFrame(frameProbe);
      for (const root of document.querySelectorAll('[contenteditable="true"]')) {
        root.removeEventListener('beforeinput', onBeforeInput, true);
        root.removeEventListener('input', onInput, true);
      }
      attachedRoots = new WeakSet();
      document.removeEventListener('visibilitychange', onVisibilityChange);
      console.log('[v3] 已卸载');
    },
    /** 原始样本，需要自己画图或做别的统计时用 */
    raw: () => ({ 输入: [...samples], 延迟: [...latencies] }),
  };

  // 调用 `__v3()` 等价于 `__v3.report()`
  const callable = (options) => api.report(options);
  Object.assign(callable, api);
  window.__v3 = callable;

  attach();
  console.log(
    `[v3] 采样中（${usingEventTiming ? 'Event Timing API' : '双 rAF 回退'}）。` +
      '敲 30 个字符后调 __v3() 看结果；__v3.reset() 清空；__v3.stop() 卸载。',
  );
})();
