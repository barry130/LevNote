/**
 * V2（mind-elixir 500 节点压测）的测量探针。
 *
 * **这不是一个模块，不要 import 它。** 整份文件是一段自执行脚本，
 * 由 `v2-run.mjs` 用 `Runtime.evaluate` 注入 —— 也可以人工粘进 Console。
 *
 * 它与 `v3-probe.mjs` 是**兄弟**，共用同一套定义与同一条纪律
 * （Event Timing 的 `duration` 就是「事件 → 它处理完之后的那次绘制」、
 * 绝对读数随机器负载漂移所以必须记校准基准、读数被量化所以别抠小数位）。
 * 不同的只有「量哪些事件」。
 *
 * ---
 *
 * ## 先读这一段：**打字这一档很可能是一个「正确的假绿」**
 *
 * 读内核源码（`dist/MindElixir.js`）可以看到：
 *
 * - 进入编辑时，内核造的是一个 `div#input-box`，`contentEditable = "plaintext-only"`；
 * - 它**只在 `blur` 上挂了监听**（提交 topic、`linkDiv()`），
 *   **没有任何 `input` / `beforeinput` 监听**；
 * - 节点本体（`me-tpc`）在编辑期间被设成 `opacity: 0`，编辑框是浮在它上面的另一个 div。
 *
 * 也就是说：**敲字这件事内核一点都不参与**，全由浏览器在原生 contenteditable 上完成。
 * 后果是「打字延迟」这个读数与节点数**结构性地无关** —— 500 个节点和 3 个节点
 * 会给出同一个数，而那个数很好看。它是**真实读数**、**没有测量误差**，
 * 但它只回答「浏览器往一个 div 里插一个字符要多久」，
 * **不回答「这个导图内核在 500 节点下卡不卡」**。
 *
 * 这与 V3 踩过的那个坑是同一类（用 `paint` 当判据，量了半条链路）：
 * 读数完全合理，指标选错了。
 *
 * 所以 V2 的主判据**不能只是打字**，必须是「用户在这个规模上会做、而代价随规模增长的事」：
 *
 * | 档 | 触发 | 为什么它是这一档的判据 |
 * |---|---|---|
 * | `type` | 编辑框里敲字符 | 字面判据。**预期与节点数无关**，量它是为了把这件事**证明**出来而不是猜 |
 * | `commit` | 编辑框里回车（→ `blur` → 提交 + `linkDiv()`） | 「敲完一按回车就卡住」是用户实际会遇到的形态 |
 * | `child` / `sibling` / `remove` | 选中节点后 Tab / Enter / Delete | 结构操作，每次都会重画全部连线 |
 * | `refresh` / `getData` / `linkDiv` | 直接调 API | **归因**：把「浏览器事件派发开销」与「内核自己的开销」分开 |
 *
 * ## 两条读数，量的不是同一条线
 *
 * | 读数 | 定义 | 来源 | 偏差 |
 * |---|---|---|---|
 * | `输入→绘制` | 事件 → 它处理完之后的那次**绘制** | Event Timing 的 `entry.duration` | 无（按定义），但被粗化 |
 * | `输入→帧` | 事件 → **下一帧开始** | 自己打点：捕获阶段记 `event.timeStamp`，`rAF` 回调里减 | **系统性略小**（rAF 在绘制之前跑），且被 vsync 量化 |
 *
 * 两条都留着，是因为 **Event Timing 不保证覆盖 `keydown`** ——
 * 规范里的「要暴露的事件类型」清单与 Chromium 的实现未必一致。
 * 万一 `keydown` 一个条目都不产生，`输入→帧` 就是唯一一条退路。
 * **它们不是用来互相印证到小数位的**：两条都被量化（8ms / 16.7ms），
 * 小差值没有意义，看的是量级与「随节点数怎么变」。
 *
 * ## 量化：别抠小数位
 *
 * V3 实测 Event Timing 的 `duration` 全是 **8 的整数倍**（Chromium 的计时粗化）。
 * 后果：判据「≤ 50ms」实际等价于「≤ 48ms」，且 p50 与 p95 常常是同一个数。
 * `输入→帧` 则被 vsync 量化到约 16.7ms。
 * 所以报告里给的是 p50 / p95 / 最大，**并且永远要连着「这一档做了多少次」一起看**。
 *
 * ## 每次读数都必须配上「这次操作真的发生了」的证据
 *
 * 这是 V3 那条「0 看起来像性能极好」的直接延伸，在 V2 里更要紧：
 * 如果 Tab 没被内核接住，`keydown` 的 `duration` 会是 0 或很小 —— **一个漂亮的假读数**。
 * 所以驱动器在每一档之后都会去数节点数 / 看编辑框还在不在，
 * 对不上就把这一档标成**失败**而不是「很快」。
 */

(() => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    console.warn('[v2] 这段要在浏览器里跑，不是在 node 里');
    return;
  }

  /** 按事件名分桶的 Event Timing 条目：`name -> [{ duration, startTime }]`。 */
  const buckets = new Map();
  /** 手动打点的「输入 → 下一帧开始」，`{ key, ms }`。 */
  const inputToFrame = [];
  const longtasks = [];
  const frameGaps = [];
  const visibilityBreaks = [];

  let eventObserver = null;
  let longtaskObserver = null;
  let usingEventTiming = false;
  let frameProbe = 0;
  let pendingArm = null;
  let armedHandler = null;
  let hiddenSince = 0;

  // -------------------------------------------------------------------------
  // Event Timing
  // -------------------------------------------------------------------------

  const onEventEntries = (list) => {
    for (const entry of list.getEntries()) {
      const record = { duration: entry.duration, startTime: entry.startTime };
      const bucket = buckets.get(entry.name);
      if (bucket === undefined) buckets.set(entry.name, [record]);
      else bucket.push(record);
    }
  };

  function attachEventTiming() {
    if (eventObserver !== null) return;
    try {
      eventObserver = new PerformanceObserver(onEventEntries);
      // `durationThreshold` 默认是 **104ms**（只报慢的）—— 不显式设成 0 就一个样本都收不到，
      // 而「收不到样本」的表现是报告里样本数为 0，看起来像「没有慢事件」。
      // 另外 Chromium 会把它钳到最小 16ms（规范里就写了 16），所以再低也没有用。
      eventObserver.observe({ type: 'event', buffered: false, durationThreshold: 0 });
      usingEventTiming = true;
    } catch (thrown) {
      eventObserver = null;
      usingEventTiming = false;
      console.warn('[v2] Event Timing 不可用，只能用「输入 → 帧」那条退路：', thrown);
    }
  }

  // -------------------------------------------------------------------------
  // 手动打点：输入 → 下一帧开始
  // -------------------------------------------------------------------------

  /**
   * 给一个键名装一次打点。**下一次**该键的 `keydown` 会被量一次。
   *
   * 捕获阶段（`true`）是刻意的：内核的监听在 `me-tab` 那条冒泡路径上，
   * 捕获阶段先到，所以 `t0` 记的是**内核开始干活之前**的时刻。
   */
  function arm(key) {
    pendingArm = { key };
    if (armedHandler !== null) return;
    armedHandler = (event) => {
      if (pendingArm === null || event.key !== pendingArm.key) return;
      const armed = pendingArm;
      pendingArm = null;
      // `event.timeStamp` 与 `performance.now()` 同源，比在回调里现取 `now()` 更准 ——
      // 事件从产生到派发到我们手上也有一小段。
      const t0 = event.timeStamp;
      requestAnimationFrame(() => {
        inputToFrame.push({ key: armed.key, ms: Number((performance.now() - t0).toFixed(2)) });
      });
    };
    window.addEventListener('keydown', armedHandler, true);
  }

  // -------------------------------------------------------------------------
  // 长任务 / 刷新率 / 可见性
  // -------------------------------------------------------------------------

  function attachLongtask() {
    if (longtaskObserver !== null) return;
    try {
      longtaskObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) longtasks.push(Number(entry.duration.toFixed(2)));
      });
      longtaskObserver.observe({ type: 'longtask', buffered: false });
    } catch {
      longtaskObserver = null;
    }
  }

  /** 估刷新率。它决定「掉一帧」值多少毫秒 —— 同样掉 4 帧，60Hz 是 66ms、120Hz 是 33ms。 */
  function measureFrameHz(durationMs = 500) {
    frameGaps.length = 0;
    return new Promise((done) => {
      const started = performance.now();
      let last = 0;
      const tick = (now) => {
        if (last !== 0) frameGaps.push(now - last);
        last = now;
        if (now - started < durationMs) frameProbe = requestAnimationFrame(tick);
        else {
          frameProbe = 0;
          const sorted = [...frameGaps].sort((a, b) => a - b);
          const median = sorted.length === 0 ? 0 : sorted[Math.floor(sorted.length / 2)];
          done({
            样本: sorted.length,
            中位帧间隔ms: Number(median.toFixed(2)),
            刷新率: median === 0 ? '未测到' : `约 ${Math.round(1000 / median)}Hz（一帧 ${median.toFixed(1)}ms）`,
          });
        }
      };
      frameProbe = requestAnimationFrame(tick);
    });
  }

  function onVisibilityChange() {
    const hidden = document.hidden;
    if (hidden && hiddenSince === 0) hiddenSince = performance.now();
    else if (!hidden && hiddenSince !== 0) {
      visibilityBreaks.push({ from: hiddenSince, ms: Number((performance.now() - hiddenSince).toFixed(2)) });
      hiddenSince = 0;
    }
  }

  // -------------------------------------------------------------------------
  // 统计
  // -------------------------------------------------------------------------

  const percentile = (sorted, p) => {
    if (sorted.length === 0) return null;
    const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil(p * sorted.length) - 1));
    return sorted[index];
  };

  function summarize(values) {
    const sorted = [...values].sort((a, b) => a - b);
    return {
      样本: sorted.length,
      最小: percentile(sorted, 0),
      p50: percentile(sorted, 0.5),
      p95: percentile(sorted, 0.95),
      最大: sorted.length === 0 ? null : sorted[sorted.length - 1],
    };
  }

  function eventSummary() {
    const out = {};
    for (const [name, records] of buckets) {
      out[name] = summarize(records.map((r) => r.duration));
    }
    return out;
  }

  // -------------------------------------------------------------------------
  // 生命周期
  // -------------------------------------------------------------------------

  function reset() {
    buckets.clear();
    inputToFrame.length = 0;
    longtasks.length = 0;
    frameGaps.length = 0;
    visibilityBreaks.length = 0;
    hiddenSince = 0;
    pendingArm = null;
  }

  function report() {
    const byEvent = eventSummary();
    const notes = [];

    if (!usingEventTiming) notes.push('⚠️ Event Timing 不可用，报告里没有「输入 → 绘制」，只有「输入 → 帧」');
    if (document.hidden) notes.push('⚠️ 此刻页面是隐藏的 —— 渲染管线被挂起，读数不可信');
    if (visibilityBreaks.length > 0) {
      notes.push(
        `⚠️ 测量期间页面被隐藏过 ${visibilityBreaks.length} 次（合计 ` +
          `${visibilityBreaks.reduce((sum, item) => sum + item.ms, 0).toFixed(0)}ms）—— 这一档的读数不可信`,
      );
    }
    const seen = Object.keys(byEvent).sort();
    if (seen.length > 0) notes.push(`本次收到条目的事件类型：${seen.join(' / ')}`);

    return {
      主判据_输入到绘制: byEvent,
      退路_输入到帧: summarize(inputToFrame.map((item) => item.ms)),
      长任务: summarize(longtasks),
      可见性中断: visibilityBreaks.length === 0 ? '无' : `⚠️ ${visibilityBreaks.length} 次`,
      备注: notes,
    };
  }

  function stop() {
    eventObserver?.disconnect();
    longtaskObserver?.disconnect();
    if (frameProbe !== 0) cancelAnimationFrame(frameProbe);
    if (armedHandler !== null) window.removeEventListener('keydown', armedHandler, true);
    document.removeEventListener('visibilitychange', onVisibilityChange);
    eventObserver = null;
    longtaskObserver = null;
    armedHandler = null;
    frameProbe = 0;
  }

  const api = {
    attach: () => {
      attachEventTiming();
      attachLongtask();
      document.addEventListener('visibilitychange', onVisibilityChange);
      return { 事件计时: usingEventTiming ? 'Event Timing API' : '不可用' };
    },
    reset,
    report,
    stop,
    arm,
    measureFrameHz,
    /** 原始样本：自己画图或做别的统计时用。 */
    raw: () => ({
      事件: Object.fromEntries([...buckets].map(([name, records]) => [name, records.map((r) => r.duration)])),
      输入到帧: [...inputToFrame],
      长任务: [...longtasks],
      可见性中断: [...visibilityBreaks],
    }),
  };

  const callable = () => api.report();
  Object.assign(callable, api);
  window.__v2 = callable;

  api.attach();
  console.log(`[v2] 探针已挂载（${usingEventTiming ? 'Event Timing API' : '无 Event Timing，只有输入→帧'}）`);
})();
