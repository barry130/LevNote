/**
 * V1 · drawio `viewer.min.js` 能否**独立**渲染。
 *
 * 判据（`docs/open-decisions.md` 的 V1 行）：
 *   ① 能否独立运行（不依赖完整 bundle / 不依赖在线服务）；
 *   ② 能否正确渲染**多页**与**压缩变体**的 `.drawio`；
 *   ③ viewer + 必要 stencils 的实测体积（阈值 6 MB）。
 *
 * 三件事各有一条独立读数：
 *   ① **服务器侧的请求日志**（含 404）+ CDP 的 `Network.requestWillBeSent`
 *      + 页面里打点过的 XHR/fetch + Resource Timing —— 四条，且有一条**正对照**
 *      （`probeNetwork` 主动去取不存在的路径）来证明这套探针本身是通的；
 *   ② 渲染完把**标签文字**读回来跟样本的期望值比（不是「有没有 svg」这种弱断言），
 *      多页还要逐页切过去看标签**换了没有**；
 *   ③ 直接量文件字节数（原始 + gzip）。
 *
 * 用法：
 *   node scripts/v1-run.mjs --out .v1-out/v1.json
 *   node scripts/v1-run.mjs --viewer .v1-out/viewer-static.min.js
 */
import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';

import * as cdp from './lib/cdp.mjs';
import { createHarness } from './lib/harness.mjs';
import { buildSamples, EXPECTED } from './v1-samples.mjs';

const harness = createHarness();

const DEFAULTS = {
  viewer: '.v1-out/viewer.min.js',
  page: 'scripts/v1-page.html',
  port: 4328,
  cdp: 9222,
  browser: null,
  out: null,
  stencils: null,
  preloadStencils: null,
  keep: false,
  spawn: true,
  noblock: false,
};

const USAGE = `用法：node scripts/v1-run.mjs [选项]

  --viewer <路径>     查看器产物（默认 .v1-out/viewer.min.js）
  --page <路径>       测试页（默认 scripts/v1-page.html）
  --port <n>          静态服务端口（默认 4328）
  --cdp <n>           调试端口（默认 9222）
  --out <路径>        读数落盘

  --stencils <目录>   把 <目录>/{stencils,shapes}/** 喂到那些**本来会 404** 的路径上。
                      这是本脚本最重要的一组对照：同一个页面、同一份样本，
                      只改「那几个路径是 404 还是真文件」，就能量出
                      「不带 stencils 会怎样」与「带了会怎样」。

  --preload-stencils <文件>
                      额外把「全部 stencils 打成一包」的文件作为 <script> 预加载
                      （见测试页的 preloadStencils）。这是完全离线的第二条路。

  --noblock           不覆盖那些资源路径，看它**默认会去哪儿取**。

  --no-spawn          连一个已经开着的调试端口
  --keep              跑完不关浏览器
  --help              这份说明
`;

const argv = process.argv.slice(2);
// `--help` 不在 DEFAULTS 里，要先拦下来 —— 否则会被 parseArgs 当成未知参数抛错。
if (argv.includes('--help')) {
  process.stdout.write(USAGE);
  process.exit(0);
}
// `--no-spawn`（关掉自起浏览器）与 `--noblock` 都由 parseArgs 处理，见 cdp.mjs 的注释。
// **未知参数会抛错**：别把参数名拼错当成「它生效了」——
// `--preload-stencils` 曾经被静默吞掉，让一轮实验变成了伪装成实验的重复。
const args = cdp.parseArgs(argv, DEFAULTS, ['port', 'cdp']);

const notes = [];
const note = (text) => {
  notes.push(text);
  process.stderr.write(`[v1] ${text.replace(/\n/g, '\n     ')}\n`);
};

/** 量体积：原始 + gzip。安装包看的是 gzip 后的量级。 */
async function measureSize(path) {
  const raw = readFileSync(path);
  const { gzipSync } = await import('node:zlib');
  const gzipped = gzipSync(raw, { level: 9 });
  return { 原始字节: raw.length, gzip字节: gzipped.length };
}

const mb = (bytes) => Number((bytes / 1048576).toFixed(3));

/**
 * 把一个目录下的 `stencils/**` 与 `shapes/**` 映射到**测试页占位用的那些路径**上。
 *
 * 映射成同一批路径是刻意的：这样「带 stencils」与「不带 stencils」两轮之间
 * 唯一的差别就是**这些路径返回 404 还是返回文件**，页面一个字都不用改。
 * 两轮之间改得越少，结论越干净。
 */
function stencilRoutes(dir) {
  const routes = new Map();
  const root = resolve(dir);
  for (const kind of ['stencils', 'shapes']) {
    const base = join(root, kind);
    if (!existsSync(base)) continue;
    for (const relative of readdirSync(base, { recursive: true })) {
      const file = join(base, relative);
      if (!statSync(file).isFile()) continue;
      const route = `/__v1_blocked__/${kind}/${String(relative).split('\\').join('/')}`;
      const type = String(relative).endsWith('.xml')
        ? 'application/xml; charset=utf-8'
        : 'text/javascript; charset=utf-8';
      routes.set(route, { file: () => readFileSync(file), type });
    }
  }
  return routes;
}

/** 期望的标签是否都出现在渲染结果里（样本里每条断言都写明了应该看到什么）。 */
function checkLabels(expected, actual) {
  const missing = expected.filter((label) => !actual.includes(label));
  return { 期望: expected, 缺失: missing, 通过: missing.length === 0 };
}

async function main() {
  const viewerPath = resolve(args.viewer);
  const pagePath = resolve(args.page);
  if (!existsSync(viewerPath)) {
    throw new Error(
      `找不到查看器产物：${viewerPath}\n` +
        '先下下来（本机 curl 走 Schannel 握手会失败、Node 的 fetch 不认 http_proxy，\n' +
        '只有 Python urllib 能通，且必须显式设 User-Agent —— 见 .v1-out/dl.py）：\n' +
        '  <python> .v1-out/dl.py https://viewer.diagrams.net/js/viewer.min.js',
    );
  }

  const samples = buildSamples();
  note(`样本 ${samples.size} 份：${[...samples.keys()].join(', ')}`);

  const viewerBytes = await measureSize(viewerPath);
  note(
    `查看器 ${viewerPath}\n` +
      `  原始 ${mb(viewerBytes.原始字节)} MiB（${viewerBytes.原始字节} 字节）· ` +
      `gzip ${mb(viewerBytes.gzip字节)} MiB（${viewerBytes.gzip字节} 字节）`,
  );

  const routes = new Map([
    ['/', { body: readFileSync(pagePath), type: 'text/html; charset=utf-8' }],
    ['/index.html', { body: readFileSync(pagePath), type: 'text/html; charset=utf-8' }],
    ['/viewer.min.js', { body: readFileSync(viewerPath), type: 'text/javascript; charset=utf-8' }],
  ]);
  for (const [name, content] of samples) {
    routes.set(`/samples/${name}`, { body: content, type: 'application/xml; charset=utf-8' });
  }

  if (args.stencils !== null) {
    const extra = stencilRoutes(args.stencils);
    for (const [route, entry] of extra) routes.set(route, entry);
    let bytes = 0;
    for (const entry of extra.values()) bytes += entry.file().length;
    note(
      `--stencils：${extra.size} 个文件挂到占位路径上，合计 ${mb(bytes)} MiB（${args.stencils}）\n` +
        '  与「不带 stencils」那一轮相比，只有这几个路径的返回不同 —— 其余全一样',
    );
  }

  let preloadBytes = null;
  if (args.preloadStencils !== null) {
    const file = resolve(args.preloadStencils);
    if (!existsSync(file)) throw new Error(`找不到 --preload-stencils 的文件：${file}`);
    const content = readFileSync(file);
    preloadBytes = content.length;
    routes.set('/stencils.min.js', { body: content, type: 'text/javascript; charset=utf-8' });
    note(`--preload-stencils：${args.preloadStencils}（${mb(content.length)} MiB）会作为 <script> 预加载`);
  }

  await harness.startServer({ port: args.port, routes });
  const url = `http://127.0.0.1:${args.port}/${args.noblock ? '?noblock=1' : ''}`;
  note(`测试页 ${url}${args.noblock ? '（--noblock：不覆盖资源路径）' : ''}`);

  const http = `http://127.0.0.1:${args.cdp}`;
  if (args.spawn) {
    await harness.requireFreePort(http);
    const exe = harness.pickBrowser(args.browser);
    const started = harness.startBrowser({ exe, cdpPort: args.cdp, url });
    note(`浏览器 ${exe}（PID ${started.child.pid}）`);
  } else {
    note('--no-spawn：连一个已经开着的调试端口');
  }
  const version = await harness.waitForDebugPort(http);
  note(`调试端口 ${http} → ${version.Browser}`);

  await cdp.connect({
    http,
    pagePattern: new RegExp(`^http://127\\.0\\.0\\.1:${args.port}/`),
    hint: `测试页没起来？直接用浏览器打开 ${url}。`,
  });

  // CDP 侧的网络日志 —— 与页面内的打点互相独立
  const cdpRequests = [];
  await cdp.send('Network.enable');
  cdp.on('Network.requestWillBeSent', (params) => {
    cdpRequests.push({ url: params.request.url, type: params.type });
  });
  // 失败也要记：403/404 与「取到了在线资源」是两种完全不同的结论
  const cdpFailures = [];
  cdp.on('Network.loadingFailed', (params) => {
    cdpFailures.push({ error: params.errorText, blocked: params.blockedReason ?? null });
  });

  await cdp.waitFor('window.__v1 !== undefined', { timeout: 30000, label: '测试页里的 __v1 出现' });
  const viewerInfo = await cdp.evaluate(
    '({ hasGraphViewer: typeof GraphViewer !== "undefined", paths: window.__v1.state().paths })',
  );
  note(
    `页面就绪：GraphViewer ${viewerInfo.hasGraphViewer ? '有' : '⚠️ 没有'}\n` +
      `  路径占位 SHAPES_PATH=${viewerInfo.paths.SHAPES_PATH}\n` +
      `           STENCIL_PATH=${viewerInfo.paths.STENCIL_PATH}`,
  );
  if (!viewerInfo.hasGraphViewer) {
    throw new Error('页面里没有 GraphViewer —— 查看器脚本没加载成功，后面所有读数都没有意义');
  }

  if (preloadBytes !== null) {
    const preload = await cdp.evaluate('window.__v1.preloadStencils("/stencils.min.js")');
    if (preload.ok !== true) {
      throw new Error(`stencils.min.js 预加载失败：${JSON.stringify(preload)}`);
    }
    note(`stencils.min.js 已预加载（${mb(preloadBytes)} MiB）`);
  }

  // --- 正对照：先证明探针本身是通的 -----------------------------------------
  const probe = await cdp.evaluate('window.__v1.probeNetwork()');
  const probeSeen =
    probe.xhrAndFetch.length > 0 && probe.newResourceTiming.length > 0;
  note(
    `网络探针正对照：XHR/fetch 记到 ${probe.xhrAndFetch.length} 条、` +
      `Resource Timing 记到 ${probe.newResourceTiming.length} 条 → ${probeSeen ? '探针可用' : '⚠️ 探针可能瞎了'}`,
  );
  if (!probeSeen) {
    throw new Error(
      '网络探针没记到自己主动发出的请求 —— 那么「查看器没有发请求」这个结论是不可信的（探针瞎了而不是它没请求）。',
    );
  }

  // --- 逐样本渲染 -----------------------------------------------------------
  const results = [];
  for (const [name, xml] of samples) {
    const expected = EXPECTED.get(name);
    const negative = expected.throws instanceof RegExp;
    cdpRequests.length = 0;
    cdpFailures.length = 0;

    const first = await cdp.evaluate(`window.__v1.render(${JSON.stringify(xml)})`);
    const pages = first.pages ?? 0;

    // 逐页切过去，看标签有没有真的换 —— 「页数对」不等于「每页都画对了」。
    //
    // 页号取自**期望表**而不是 `0..pages`：有的样本根本没有 `<diagram>`
    // （查看器的页集合为 0，却仍然画出了内容），按页数循环会**一条断言都不做** ——
    // 那就是一个「看起来跑过了」的空转。
    const pageChecks = [];
    for (const index of Object.keys(expected.labels)
      .map(Number)
      .sort((a, b) => a - b)) {
      const state = index === 0 ? first : await cdp.evaluate(`window.__v1.selectPage(${index})`);
      const check = checkLabels(expected.labels[index], state.labels);
      pageChecks.push({
        页: index,
        名称: state.pageNames[index] ?? null,
        currentPage: state.currentPage,
        标签: state.labels,
        ...check,
      });
    }

    // 渲染之后再等一小会儿，把「懒加载才去取的资源」也收进来
    await cdp.sleep(600);
    const state = await cdp.evaluate('window.__v1.state()');
    const offlineRequests = harness.requests.filter((item) => item.status === 404);

    // 负样本：期望的是「**明确报错**」，而不是「画出一张空白画布」。
    // 这两者在弱断言下长得一模一样（都没有 svg），差别必须由错误消息担着。
    const 拒收命中 = negative
      ? (state.errors.find((line) => expected.throws.test(line)) ?? null)
      : null;

    const 页数通过 = expected.pages === null ? null : pages === expected.pages;
    const 页名通过 =
      expected.pageNames === null
        ? null
        : JSON.stringify(first.pageNames) === JSON.stringify(expected.pageNames);
    const 标签全部通过 = pageChecks.every((item) => item.通过);

    results.push({
      样本: name,
      负样本: negative,
      期望页数: expected.pages,
      实际页数: pages,
      页数通过,
      页名: first.pageNames,
      页名通过,
      逐页: pageChecks,
      标签全部通过,
      // 「形状对不对」要靠**跨样本对照**才能回答，所以留到循环之后再算（见下）。
      对照: expected.对照 ?? null,
      形状通过: null,
      形状对比: null,
      判定通过: null,
      拒收命中,
      画出来的: {
        svg: first.svg,
        rect: first.rect,
        path: first.path,
        ellipse: first.ellipse,
      },
      指纹: first.指纹,
      宿主文本: first.hostText,
      页面内错误: state.errors,
      页面内请求: state.requests,
      资源计时: state.resourceTiming,
      CDP请求: cdpRequests.slice(),
      CDP失败: cdpFailures.slice(),
      服务器404: offlineRequests,
    });
  }

  // --- 形状判据：只能靠对照 -------------------------------------------------
  //
  // 为什么不是「不等于某个实测指纹」：那样写过一版，然后**被自己的判据骗了**。
  // 预加载 `stencils.min.js` 那一轮（RUN C）的形状**仍然是回落的**
  // （rect 2 / path 3，与纯矩形一模一样），只因为 svg 总长度是 2827 而不是基准的 2809
  // —— 差 18 个字符 —— 就被判成「通过」。
  // **「不等于」这种判据对「差一点点」毫无分辨力**，而差一点点正是回落的样子。
  //
  // 换成与**几何完全相同**的纯矩形对照比结构：没有会飘的量参与判定。
  for (const item of results) {
    if (item.对照 === null) continue;
    const control = results.find((x) => x.样本 === item.对照);
    if (control === undefined) throw new Error(`找不到对照样本：${item.对照}`);
    const mine = item.指纹.标签数量;
    const theirs = control.指纹.标签数量;
    // stencil 的轮廓用 `<path>` 画，回落则是一个 `<rect>` → 矩形必须变少、路径必须变多。
    // **两个条件都要**：只比一个的话，另一个恰好相等就能骗过去。
    item.形状通过 = mine.rect < theirs.rect && mine.path > theirs.path;
    item.形状对比 = {
      对照样本: item.对照,
      本样本: { rect: mine.rect, path: mine.path },
      对照: { rect: theirs.rect, path: theirs.path },
    };
  }

  for (const item of results) {
    item.判定通过 = item.负样本
      ? item.拒收命中 !== null
      : item.页数通过 !== false &&
        item.页名通过 !== false &&
        item.标签全部通过 &&
        item.形状通过 !== false;
  }

  // --- 报告 -----------------------------------------------------------------
  const lines = [];
  lines.push('\n=== V1 · drawio 查看器独立渲染 ===');
  lines.push(`查看器：${args.viewer}`);
  lines.push(
    `体积：原始 ${mb(viewerBytes.原始字节)} MiB · gzip ${mb(viewerBytes.gzip字节)} MiB` +
      `（判据 ≤ 6 MB → ${viewerBytes.原始字节 <= 6 * 1024 * 1024 ? '通过' : '不通过'}）`,
  );
  lines.push(`网络探针正对照：${probeSeen ? '可用' : '不可用'}`);
  lines.push(
    `stencils 占位：${args.noblock ? '⚠️ 未占位（--noblock）' : '已占位（指向 404）'}` +
      `${args.stencils === null ? '' : ` + 真文件（${args.stencils}）`}` +
      `${args.preloadStencils === null ? '' : ` + 预加载包（${args.preloadStencils}）`}`,
  );
  lines.push('');
  lines.push(
    '样本'.padEnd(28) +
      '判定'.padEnd(8) +
      '页数'.padEnd(10) +
      '页名'.padEnd(8) +
      '标签'.padEnd(8) +
      '形状'.padEnd(8) +
      '请求'.padEnd(8) +
      '404',
  );
  for (const item of results) {
    lines.push(
      String(item.样本).padEnd(28) +
        (item.判定通过 ? '✅' : '❌').padEnd(8) +
        (item.期望页数 === null ? '—' : `${item.实际页数}/${item.期望页数}`).padEnd(10) +
        (item.页名通过 === null ? '—' : item.页名通过 ? '✅' : '❌').padEnd(8) +
        (item.负样本 ? '—' : item.标签全部通过 ? '✅' : '❌').padEnd(8) +
        (item.形状通过 === null ? '—' : item.形状通过 ? '✅' : '❌ 回落了').padEnd(8) +
        String(item.页面内请求.length).padEnd(8) +
        String(item.服务器404.length),
    );
  }
  lines.push('');
  lines.push('形状指纹（判「画的是不是那个形状」—— 标签对不上就说明不了这件事）：');
  for (const item of results) {
    lines.push(
      `  ${String(item.样本).padEnd(30)} ${JSON.stringify(item.指纹.标签数量)}  svg 长度 ${item.指纹.svg长度}`,
    );
    for (const d of item.指纹.path的d.slice(0, 4)) lines.push(`      d=${d}`);
  }
  lines.push('');
  for (const item of results) {
    lines.push(`【${item.样本}】`);
    if (item.负样本) {
      lines.push(
        `  负样本（证伪过的转义文本变体）→ ${item.拒收命中 === null ? '❌ 没报错（画了张空白画布？）' : '✅ 明确报错'}`,
      );
      if (item.拒收命中 !== null) lines.push(`  错误消息：${item.拒收命中.slice(0, 180)}`);
    }
    lines.push(`  页名：${JSON.stringify(item.页名)}`);
    if (item.形状对比 !== null) {
      lines.push(
        `  形状对比（对照 ${item.形状对比.对照样本}）：` +
          `本样本 rect ${item.形状对比.本样本.rect} / path ${item.形状对比.本样本.path}，` +
          `对照 rect ${item.形状对比.对照.rect} / path ${item.形状对比.对照.path}` +
          ` → ${item.形状通过 ? '✅ 结构变了（stencil 生效）' : '❌ 与纯矩形一模一样（回落了）'}`,
      );
    }
    for (const page of item.逐页) {
      lines.push(
        `  第 ${page.页} 页（${page.名称}）currentPage=${page.currentPage} 标签=${JSON.stringify(page.标签)}` +
          ` → ${page.通过 ? '✅' : '❌ 缺 ' + JSON.stringify(page.缺失)}`,
      );
    }
    if (item.宿主文本) lines.push(`  宿主文本：${item.宿主文本.slice(0, 200)}`);
    if (item.页面内错误.length > 0) lines.push(`  页面内错误：${JSON.stringify(item.页面内错误.slice(0, 4))}`);
    if (item.页面内请求.length > 0) lines.push(`  页面内请求：${JSON.stringify(item.页面内请求.slice(0, 6))}`);
    if (item.服务器404.length > 0) lines.push(`  服务器 404：${JSON.stringify(item.服务器404.slice(0, 6))}`);
    if (item.CDP请求.length > 0) {
      lines.push(`  CDP 请求：${JSON.stringify(item.CDP请求.map((x) => x.url).slice(0, 6))}`);
    }
  }

  const failed = results.filter((item) => !item.判定通过);
  lines.push('');
  lines.push(
    failed.length === 0
      ? `判定：${results.length} 个样本全部通过`
      : `判定：${results.length - failed.length}/${results.length} 通过，未过：${failed.map((x) => x.样本).join(', ')}`,
  );

  console.log(lines.join('\n'));
  console.log(notes.join('\n'));

  if (args.out !== null) {
    const outPath = resolve(args.out);
    mkdirSync(dirname(outPath), { recursive: true });
    writeFileSync(
      outPath,
      JSON.stringify(
        {
          元信息: {
            时间: new Date().toISOString(),
            查看器: args.viewer,
            体积: { ...viewerBytes, 原始MiB: mb(viewerBytes.原始字节), gzipMiB: mb(viewerBytes.gzip字节) },
            浏览器: version.Browser,
            网络探针正对照: probeSeen ? '可用' : '不可用',
          },
          样本: results,
          备注: notes,
        },
        null,
        2,
      ),
      'utf8',
    );
    console.log(`\n读数已落盘：${outPath}`);
  }

  harness.keepBrowserFlag = args.keep;
}

main()
  .then(() => harness.teardown({ keepBrowser: harness.keepBrowserFlag }))
  .catch(async (thrown) => {
    console.error('\n❌', thrown instanceof Error ? thrown.message : thrown);
    if (thrown instanceof Error && thrown.cause) console.error('   原因：', thrown.cause);
    await harness.teardown({ keepBrowser: harness.keepBrowserFlag });
    process.exitCode = 1;
  });
