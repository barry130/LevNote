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
 * ## 三个读数，各自回答不同的问题
 *
 * | 读数 | 含义 | 误差 |
 * |---|---|---|
 * | `sync` | `beforeinput` → `input`：**内核在一个事件循环里跑完的成本**（事务派发 + 整篇序列化上报 + 位置换算 + React 同步部分） | **无** |
 * | `paint` | 事件时间戳 → **下一次绘制**：用户实际感知的输入延迟 | **无**（Event Timing API 的定义） |
 * | `longtasks` | 主线程上 > 50ms 的独立任务 | **无** |
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
  const longtasks = [];
  const frameGaps = [];

  let mounted = 0;
  let usingEventTiming = false;
  /** 配不到 `input` 标记、被丢弃的 Event Timing 条目数（别的元素上的输入 / 历史条目）。 */
  let orphans = 0;
  let lastInputAt = 0;
  let armed = null;

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
  const takeMark = (startTime) => {
    let bestIndex = -1;
    let bestDelta = Number.POSITIVE_INFINITY;
    for (let i = 0; i < marks.length; i += 1) {
      const delta = Math.abs(marks[i].at - startTime);
      if (delta < bestDelta) {
        bestDelta = delta;
        bestIndex = i;
      }
    }
    if (bestIndex === -1 || bestDelta > 1) return null;
    return marks.splice(bestIndex, 1)[0];
  };

  // -------------------------------------------------------------------------
  // 主测量：Event Timing API
  // -------------------------------------------------------------------------

  let eventObserver = null;
  try {
    eventObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
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

  // -------------------------------------------------------------------------
  // 输入监听：拿 inputType，算内核成本
  // -------------------------------------------------------------------------

  const onBeforeInput = (event) => {
    armed = { at: performance.now(), inputType: event.inputType || 'unknown' };
  };

  const onInput = (event) => {
    const started = armed;
    armed = null;

    const inputType = (started === null ? '' : started.inputType) || event.inputType || 'unknown';
    const sync = started === null ? null : performance.now() - started.at;
    const gap = started === null ? 0 : started.at - lastInputAt;
    if (started !== null) lastInputAt = started.at;

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

  const attach = () => {
    // 只挂真正可编辑的：分屏右栏是 `contenteditable="false"`，会被排除
    const roots = [...document.querySelectorAll('[contenteditable="true"]')];
    for (const root of roots) {
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
    const over = paintValues.filter((v) => v > 50).length;
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
      sync,
      paint,
      '超过 50ms 的输入': `${over} / ${kept.length}（${((over / Math.max(1, kept.length)) * 100).toFixed(0)}%）`,
      长任务: { 条数: longtasks.length, 最长: longtasks.length === 0 ? 0 : Math.max(...longtasks) },
      按inputType分组: byType,
      归因:
        paintValues.length === 0
          ? '还没有样本'
          : sync.p95 > 50
            ? '内核 JS 同步成本本身就超了 —— 对着 plugin.ts 的 O(n) 清单逐条查'
            : paint.p95 > 50
              ? '内核 JS 达标、慢在「内核跑完 → 绘制」这一段 —— 瓶颈在样式/布局/绘制（或掉帧），不是 JS'
              : '两项都达标',
      结论:
        paintValues.length === 0
          ? '还没有样本'
          : paint.p95 <= 50
            ? `✅ p95 = ${paint.p95}ms，在判据内`
            : `❌ p95 = ${paint.p95}ms，超出判据`,
    };
  };

  const api = {
    attach,
    report,
    reset: () => {
      samples.length = 0;
      marks.length = 0;
      longtasks.length = 0;
      frameGaps.length = 0;
      orphans = 0;
      lastInputAt = 0;
      console.log('[v3] 已清空');
    },
    stop: () => {
      eventObserver?.disconnect();
      longtaskObserver?.disconnect();
      if (typeof cancelAnimationFrame === 'function' && frameProbe !== 0) cancelAnimationFrame(frameProbe);
      for (const root of document.querySelectorAll('[contenteditable="true"]')) {
        root.removeEventListener('beforeinput', onBeforeInput, true);
        root.removeEventListener('input', onInput, true);
      }
      console.log('[v3] 已卸载');
    },
    /** 原始样本，需要自己画图或做别的统计时用 */
    raw: () => [...samples],
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
