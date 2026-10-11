/**
 * V4 · 随包的 `viewer.min.js` **能不能起出真正的 draw.io 编辑器**。
 *
 * ## 这次要回答的是一个具体的事实问题，不是性能问题
 *
 * 一直以来我们把 `public/drawio/viewer.min.js` 当成**只读查看器**（ADR 0027），
 * 于是「`.drawio` 可编辑」被排到 ROADMAP 的 P5 第 26 项「编辑器按需下载」，
 * 而那条路的前提是**能下载**。
 *
 * 但探针扫过这个文件之后发现：它是**完整的 draw.io 应用**，不只是查看器 ——
 * `Editor` / `Graph` / `EditorUi` / `EmbedFile` / `App` 都是这个文件里的顶层符号，
 * 连 `EditorUi.prototype.installMessageHandler`（embed 的 postMessage 协议）
 * 与 `initializeEmbedMode` 都在里面。缺的只是**入口**：文件末尾那一行只调
 * `Editor.initMath()` 与 `GraphViewer.processElements()`。
 *
 * 所以要问的是：**把入口补上之后，编辑器真的能用吗？**
 *
 * ## 四条读数，各自可证伪
 *
 *   ① `new EditorUi(document.body)` + `initializeEmbedMode()` 能不能不抛异常；
 *   ② 父窗口 `postMessage` 一份 XML 过去，iframe 里**真的画出图了吗**
 *      （数的是 `mxGraphModel` 里的 cell，不是「页面上有没有 svg」这种弱断言）；
 *   ③ 在编辑器里改一下再保存，XML **真的回到父窗口了吗**（`{event:'save', xml}`）——
 *      这一条决定了「内核能不能把 XML 当文档内容经 `onChange` 交回外壳落盘」；
 *   ④ 整个过程有没有碰网络（`Network.requestWillBeSent` 的全量记录）。
 *
 * 用法：
 *   node scripts/v4-run.mjs --out .v1-out/v4.json
 */
import { mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

import * as cdp from './lib/cdp.mjs';
import { createHarness } from './lib/harness.mjs';

const harness = createHarness();

const DEFAULTS = {
  host: 'scripts/v4-host.html',
  editor: 'scripts/v4-editor.html',
  assets: 'public/drawio',
  port: 4329,
  cdp: 9229,
  browser: null,
  out: null,
  keep: false,
  spawn: true,
};

const USAGE = `用法：node scripts/v4-run.mjs [选项]

  --host <路径>      宿主页（默认 scripts/v4-host.html）
  --editor <路径>    编辑器页（默认 scripts/v4-editor.html；它挂在 /drawio/editor.html 这条路由上，
                  于是页面里的相对路径与随包分发时一致 —— 探针页的路径必须一致）
  --assets <目录>    viewer.min.js 与 shapes/stencils 所在目录（默认 public/drawio）
  --port <n>         静态服务端口（默认 4329）
  --cdp <n>          调试端口（默认 9229）
  --browser <路径>   浏览器可执行文件（默认按 harness 的候选顺序找）
  --out <路径>       读数落盘
  --keep             跑完不关浏览器
  --no-spawn         不自己起浏览器（连已经在跑的）
`;

/** 一张最小的图：一个圆角矩形 + 一条标题文字。 */
const SAMPLE = [
  '<mxfile host="probe">',
  '<diagram name="页1" id="p1">',
  '<mxGraphModel dx="800" dy="600" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="850" pageHeight="1100" math="0" shadow="0">',
  '<root>',
  '<mxCell id="0" />',
  '<mxCell id="1" parent="0" />',
  '<mxCell id="2" value="探针标题" style="rounded=1;whiteSpace=wrap;html=1;" vertex="1" parent="1">',
  '<mxGeometry x="160" y="120" width="240" height="80" as="geometry" />',
  '</mxCell>',
  '</root>',
  '</mxGraphModel>',
  '</diagram>',
  '</mxfile>',
].join('');

const args = cdp.parseArgs(process.argv.slice(2), DEFAULTS, ['port', 'cdp']);
if (process.argv.includes('--help')) {
  console.log(USAGE);
  process.exit(0);
}

const cdpHttp = `http://127.0.0.1:${args.cdp}`;
const pageUrl = `http://127.0.0.1:${args.port}/v4-host.html`;

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.xml': 'application/xml; charset=utf-8',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.css': 'text/css; charset=utf-8',
};

/** 把 `public/drawio` 下的东西按原路径喂给静态服务（404 也记账）。 */
function assetRoutes(root, into) {
  const walk = (dir) => {
    for (const name of readdirSync(dir)) {
      const full = join(dir, name);
      if (statSync(full).isDirectory()) {
        walk(full);
        continue;
      }
      const route = `/drawio/${relative(root, full).replace(/\\/g, '/')}`;
      const dot = route.lastIndexOf('.');
      const type = MIME[route.slice(dot)] ?? 'application/octet-stream';
      into.set(route, { file: () => readFileSync(full), type });
    }
  };
  walk(root);
  return into;
}

/** 全量网络记录。判据是「有没有一条 URL 的 host 不是 127.0.0.1」。 */
const requests = [];
cdp.on('Network.requestWillBeSent', (params) => {
  requests.push(params.request.url);
});

const fail = (message, cause) => {
  console.error(message);
  process.exitCode = 1;
  if (cause) console.error(cause);
};

// 只在 try / catch 两条路末尾各赋值一次，`finally` 之后才读。
// 0 = 四条判据全过，2 = 跑完了但有判据没过，1 = 没跑到能下结论的地方。
let exitCode;
try {
  const routes = new Map();
  routes.set('/v4-host.html', { file: () => readFileSync(resolve(args.host)), type: MIME['.html'] });
  // 编辑器页住在 `/drawio/` 下面（与随包分发时的相对路径一致），
  // 于是它页面里的 `viewer.min.js`、`shapes/` 解析到的也是 `/drawio/…`
  const editorBody = () => readFileSync(resolve(args.editor));
  routes.set('/drawio/editor.html', { file: editorBody, type: MIME['.html'] });
  assetRoutes(resolve(args.assets), routes);

  await harness.requireFreePort(cdpHttp);
  await harness.startServer({ port: args.port, routes });

  if (args.spawn) {
    const exe = harness.pickBrowser(args.browser);
    harness.startBrowser({ exe, cdpPort: args.cdp, url: pageUrl });
    await harness.waitForDebugPort(cdpHttp);
  }

  await cdp.connect({ http: cdpHttp, pagePattern: /v4-host\.html/ });
  await cdp.send('Network.enable');
  await cdp.send('Log.enable').catch(() => undefined);

  // ① 编辑器起没起来
  await cdp.waitFor('window.send', { label: '宿主页脚本' });
  await cdp.waitFor(
    `(() => { const w = document.getElementById('frame').contentWindow;
        return w && (w.__drawioHostReady === true || w.__drawioHostError != null); })()`,
    { timeout: 40000, label: 'EditorUi 起起来（__drawioHostReady 或 __drawioHostError）' },
  );
  const bootError = await cdp.evaluate('document.getElementById("frame").contentWindow.__drawioHostError ?? null');
  const bootReady = await cdp.evaluate('document.getElementById("frame").contentWindow.__drawioHostReady === true');
  const globals = await cdp.evaluate('document.getElementById("frame").contentWindow.__drawioGlobals ?? null');
  const read1 = { hostError: bootError, ready: bootReady, globals };
  if (bootError !== null) {
    // 起不来就没必要往下跑：后面三条读数会全都是「因为没编辑器所以没有」，
    // 混进报告里只会让人以为那三条也测过了。
    console.log(JSON.stringify({ boot: read1 }, null, 2));
    throw new Error(`EditorUi 没起来：${bootError}`);
  }

  // ② 一份 XML 送过去，里面真的画出图了吗
  const sampleLiteral = JSON.stringify(SAMPLE);
  await cdp.evaluate(`window.send({ action: 'load', xml: ${sampleLiteral}, autosave: 0 })`);
  await cdp.waitFor('window.__log.length > 0', { label: '编辑器回话' });
  const read2 = {
    messages: await cdp.evaluate('window.__log.slice(0, 4)'),
    // 模型里有几个 cell —— 「文件读进来了」与「画出来了」是两回事
    cells: await cdp.evaluate(
      `document.getElementById('frame').contentWindow.__drawioHostUi.editor.graph.getModel()
         .getChildCount(document.getElementById('frame').contentWindow.__drawioHostUi.editor.graph.getModel().getRoot())`,
    ),
    // 标签文字有没有真的画到画布上（渲染完成的判据）
    drawnLabel: await cdp.evaluate(
      `(document.getElementById('frame').contentDocument.body.textContent || '').includes('探针标题')`,
    ),
  };

  // ③ 在编辑器里加一个形状，再保存 —— XML 回不回得来
  await cdp.evaluate(
    `(function () {
       const w = document.getElementById('frame').contentWindow;
       const graph = w.__drawioHostUi.editor.graph;
       graph.insertVertex(graph.getDefaultParent(), null, '探针加的', 80, 60, 160, 60,
         'rounded=0;whiteSpace=wrap;html=1;');
       return true;
     })()`,
  );
  const beforeSave = await cdp.evaluate('window.__log.length');
  await cdp.evaluate(`document.getElementById('frame').contentWindow.__drawioHostUi.saveFile()`);
  let savedXml = null;
  for (let i = 0; i < 40 && savedXml === null; i += 1) {
    savedXml = await cdp.evaluate(
      `(() => {
         for (const raw of window.__log.slice(${beforeSave})) {
           try { const m = JSON.parse(raw); if (m.event === 'save' && m.xml) return m.xml; } catch {}
         }
         return null;
       })()`,
    );
    if (savedXml === null) await cdp.sleep(150);
  }
  const read3 = { gotSaveBack: savedXml !== null, hasAddedLabel: savedXml !== null && savedXml.includes('探针加的') };

  // ④ 有没有碰网络
  const offHost = requests.filter((url) => !url.startsWith('http://127.0.0.1:'));
  const read4 = { total: requests.length, offHost, notFound: harness.requests.filter((r) => r.status === 404) };

  const report = {
    ranAt: new Date().toISOString(),
    boot: read1,
    load: read2,
    save: read3,
    network: read4,
    verdict: {
      editorBoots: read1.ready && read1.hostError === null,
      diagramRenders: read2.drawnLabel === true,
      editComesBack: read3.gotSaveBack === true && read3.hasAddedLabel === true,
      staysOffline: read4.offHost.length === 0,
    },
  };

  console.log(JSON.stringify(report, null, 2));
  if (args.out) {
    mkdirSync(resolve(args.out, '..'), { recursive: true });
    writeFileSync(resolve(args.out), JSON.stringify(report, null, 2));
    console.error(`\n读数已落盘：${args.out}`);
  }
  exitCode = Object.values(report.verdict).every(Boolean) ? 0 : 2;
} catch (thrown) {
  fail('探针跑完了，但没跑到能下结论的地方。', thrown instanceof Error ? thrown.message : String(thrown));
  exitCode = 1;
} finally {
  await harness.teardown({ keepBrowser: args.keep });
}

process.exit(exitCode);