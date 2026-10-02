/**
 * 真实浏览器压测/验证的**共用外壳**：静态服务、起浏览器、探端口、收尾。
 *
 * V2（mind-elixir 压测）与 V1（drawio 查看器）本来是各写一份的，
 * 而这几段的正确写法里有好几条是**踩出来的**（见下面每处的注释）——
 * 抄一份就等于把踩过的坑再留一个副本。所以抽到这里，两边共用。
 *
 * 与 `cdp.mjs` 的分工：那个只管「连上之后怎么说话」，这个管「怎么把宿主拉起来、怎么干净地收掉」。
 */
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';

import * as cdp from './cdp.mjs';

export const BROWSER_CANDIDATES = [
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
];

/** 端口上有没有东西在应答（不问是什么，只问「通不通」）。 */
export async function portAnswers(http) {
  try {
    await fetch(`${http}/json/version`);
    return true;
  } catch {
    return false;
  }
}

export async function waitForDebugPort(http, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    try {
      const response = await fetch(`${http}/json/version`);
      if (response.ok) return await response.json();
    } catch {
      // 端口还没起来，继续等
    }
    if (Date.now() > deadline) throw new Error(`等了 ${timeoutMs}ms，${http} 还是连不上`);
    await cdp.sleep(300);
  }
}

/**
 * 起一个浏览器，指向 `url`，开远程调试端口。
 *
 * 几条启动参数**必须**有：少了它们，窗口被别的窗口盖住时 Chromium 会挂起渲染管线，
 * 读数会窜到上千毫秒 —— 而那不像 0 那样一眼可疑，只会被读成「这一档特别慢」。
 */
export function spawnBrowser({ exe, cdpPort, url, windowSize = '1440,900', windowPosition = '40,40' }) {
  const profile = resolve(tmpdir(), `qm-harness-profile-${Date.now()}`);
  const args = [
    `--remote-debugging-port=${cdpPort}`,
    `--user-data-dir=${profile}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-extensions',
    '--disable-features=CalculateNativeWinOcclusion,msEdgeSidebarV2',
    '--disable-backgrounding-occluded-windows',
    '--disable-renderer-backgrounding',
    '--disable-background-timer-throttling',
    `--window-size=${windowSize}`,
    `--window-position=${windowPosition}`,
    url,
  ];
  const child = spawn(exe, args, { detached: true, stdio: 'ignore' });
  child.unref();
  return { child, profile, exe };
}

/**
 * 一个**只绑 127.0.0.1** 的静态服务。
 *
 * `routes`：`Map<路径, { file } 或 { body, type }>`。
 * 未命中的路径一律 404 —— 但**记进 `requests` 里**，因为「它去取了什么」正是
 * V1 要测的东西（CDN 版的查看器会把 stencils / shapes 的路径硬写成绝对地址，
 * 404 与「取到了在线资源」是两种完全不同的结论）。
 */
export function createHarness() {
  let server = null;
  let browser = null;
  let keepBrowserFlag = false;
  /** 收到过的每一个请求（含 404）。判「有没有联网取资源」靠它。 */
  let requests = [];

  const startServer = async ({ port, routes }) => {
    requests = [];
    const nodeServer = createServer((request, response) => {
      const route = new URL(request.url ?? '/', 'http://127.0.0.1').pathname;
      const entry = routes.get(route);
      if (entry === undefined) {
        requests.push({ path: route, status: 404 });
        response.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' });
        response.end('not found');
        return;
      }
      requests.push({ path: route, status: 200 });
      const body = entry.body !== undefined ? entry.body : entry.file();
      response.writeHead(200, { 'content-type': entry.type });
      response.end(body);
    });
    server = await new Promise((done, fail) => {
      nodeServer.once('error', fail);
      nodeServer.listen(port, '127.0.0.1', () => done(nodeServer));
    });
    return server;
  };

  const startBrowser = (options) => {
    browser = spawnBrowser(options);
    return browser;
  };

  /** 幂等：happy path 与 catch 路径都会调，重复调不报错。 */
  const teardown = async ({ keepBrowser = false } = {}) => {
    if (!keepBrowser && browser !== null) {
      await cdp.send('Browser.close').catch(() => undefined);
    }
    cdp.close();
    if (server !== null) {
      await new Promise((done) => server.close(done));
      server = null;
    }
    if (browser !== null && keepBrowser) {
      console.log(`浏览器留着没关（--keep）：PID ${browser.child.pid}，profile ${browser.profile}`);
    }
    browser = null;
  };

  /**
   * 起浏览器**之前**必须先探端口。
   *
   * 一个残留的浏览器会把端口占着，新起的那个绑不上，而 `waitForDebugPort` 会
   * **高高兴兴地连上旧的那个** —— 于是脚本测的是上一次的页面（甚至是被改过的代码），
   * 而所有输出看起来完全正常。这类「静默连错对象」比报错难查得多。
   */
  const requireFreePort = async (http) => {
    if (await portAnswers(http)) {
      throw new Error(
        `${http} 上已经有一个调试端口在应答 —— 大概率是上一次跑残留的浏览器。\n` +
          '新起的浏览器绑不上这个端口，脚本会静默连到旧的那个（测错页面）。\n' +
          '先清掉它：关掉那个浏览器窗口；或改用 --no-spawn 明确表示「就用这个」。',
      );
    }
  };

  const pickBrowser = (explicit) => {
    const exe = explicit ?? BROWSER_CANDIDATES.find((candidate) => existsSync(candidate));
    if (exe === undefined) {
      throw new Error(
        `找不到浏览器。用 --browser 指定，或加 --no-spawn 自己起一个。\n试过：\n  ${BROWSER_CANDIDATES.join('\n  ')}`,
      );
    }
    return exe;
  };

  return {
    get server() {
      return server;
    },
    get browser() {
      return browser;
    },
    get requests() {
      return requests;
    },
    get keepBrowserFlag() {
      return keepBrowserFlag;
    },
    set keepBrowserFlag(value) {
      keepBrowserFlag = value;
    },
    startServer,
    startBrowser,
    pickBrowser,
    requireFreePort,
    waitForDebugPort,
    teardown,
  };
}
