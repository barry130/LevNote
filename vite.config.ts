// 用 vitest/config 的 defineConfig：它扩展了 Vite 的配置类型，才有 `test` 字段。
// 运行时二者是同一个对象，不影响 Tauri 拉起 dev server。
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { readFileSync } from 'node:fs';
import { fileURLToPath, URL } from 'node:url';
import type { Plugin as PostcssPlugin } from 'postcss';

import { woff2OnlyFontFaces } from './src/core/katex-fonts.js';

/**
 * 应用版本，从 `package.json` 读进来注入。
 *
 * 为什么不在设置页里写死一个字符串：写死的那个**一定会漂**（发版时改了
 * `package.json` 却忘了改界面），而漂了之后表现是「关于页显示的版本号是旧的」——
 * 一个没人会去核对的数字。从唯一来源读，就不可能不一致。
 *
 * 用 `readFileSync` 而不是 `import pkg from './package.json'`：后者会把整份
 * `package.json`（含全部依赖清单）打进产物，白送几 KB 给用户。
 */
const packageVersion = (JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf8')) as {
  version: string;
}).version;

/**
 * 编辑器那份 KaTeX CSS 只声明 woff2 字体。
 *
 * ## 不这么做会怎样
 *
 * `markdown.css` 里 `@import 'katex/dist/katex.min.css'`，而那份 CSS 的每条
 * `@font-face` 都指向三个文件（woff2 / woff / ttf）。Vite 会把**三个都**当成
 * 资源产出：实测 60 个文件、1.05 MB —— 而 woff 与 ttf 那 798 KB
 * **永远不会被取用**（目标环境只有 WebView2 与 WKWebView，两者从 2016 年起
 * 就支持 woff2）。它们会一直躺在安装包里，占 dist 的四分之一。
 *
 * ## 为什么是 PostCSS 插件，而不是 Vite 的 `transform` 钩子
 *
 * 一开始写成 `transform(code, id)` + `enforce: 'pre'`，**实测无效**：重建后
 * 产物里 woff/ttf 一个没少。根因是 `@import` 由 `postcss-import` 在 PostCSS
 * 管线内部读盘内联，**根本不经过 Vite 的模块图** —— 探针里
 * `katex/dist/katex.min.css` 从未以模块形式出现在 `resolveId` / `load` /
 * `transform` 的任何一次调用里（只有导出侧那条 `?raw` 走了）。钩子挂在那里
 * 等于挂在一个永远不会被经过的路口上。
 *
 * 而 PostCSS 插件看到的已经是内联完成的结果：探针在 `Once` 里数到 20 条
 * `@font-face`、60 个不同的 `url()`。`postcss-import` 由 Vite 插在用户插件的
 * **前面**，所以这里不用自己排序。
 *
 * ## 为什么是整份 AST 上的变换，而不是改 CSS 文本
 *
 * 「只留 woff2」这件事在 `woff2OnlyFontFaces` 里，并且对着**真实的**
 * `katex.min.css` 测过（含幂等性、缺 woff2 时抛错）。它**就地改 `decl.value`**、
 * 一个节点都不重建 —— 因为 Vite 解析 `url()` 用的是每个声明自己的
 * `declaration.source.input.file`，重建节点会把来源从 katex 目录改成
 * `markdown.css`，相对路径就解析不到了（实测：构建成功、字体 404）。
 * 那条弯路与结论记在该函数的注释里。
 */
const katexWoff2Only: PostcssPlugin = {
  postcssPlugin: 'qingmo:katex-woff2-only',
  Once(root, { result }) {
    // 只认编辑器那份。导出侧读的是 `?raw`（原样文本，不走 CSS 管线），
    // 它有自己的 data URI 内联逻辑，两条路刻意分开。
    if (!String(result.opts.from ?? '').includes('markdown.css')) return;

    // 格式不认识时它会抛错 —— 让构建停下来，而不是产出一份所有公式都掉字形的产物
    woff2OnlyFontFaces(root);
  },
};

// Tauri 会以固定端口拉起 dev server；src-tauri 的变动不该触发前端热更新。
export default defineConfig({
  plugins: [react()],
  css: { postcss: { plugins: [katexWoff2Only] } },
  clearScreen: false,
  define: {
    __QINGMO_VERSION__: JSON.stringify(packageVersion),
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@qingmo/interop-spec': fileURLToPath(
        new URL('./packages/interop-spec/src/index.ts', import.meta.url),
      ),
    },
  },
  server: {
    /**
     * **必须显式绑 IPv4，不能只写 `port`。**
     *
     * 只给 `port` 时 Vite 用 `localhost` 当 host，交给 `server.listen()` 解析。
     * 而这台机器上 `localhost` 只解析到 `::1`（IPv6）—— `ping localhost`
     * 回的就是 `[::1]` —— 于是 Vite **只监听 `[::1]:1420`**。
     *
     * WebView2（Chromium）解析 `localhost` 走的是 IPv4 `127.0.0.1`，
     * 于是**根本连不上 dev server**。表现是窗口内容区一片纯黑、
     * **没有任何报错**（不是「CSS 加载了但 React 没挂载」—— 那种情况
     * 会露出 `body` 的浅色底 `--surface-sunken`）。
     *
     * 实测（`--noproxy '*'` 绕开环境里的代理）：
     *   http://127.0.0.1:1420/  → 000（不可达）
     *   http://[::1]:1420/      → 200
     * 且 `netstat` 里 1420 上**没有 ESTABLISHED** —— 客户端从没连上过。
     *
     * 配套：`src-tauri/tauri.conf.json` 的 `devUrl` 也写成 `127.0.0.1`，
     * 两端都别再经过 `localhost` 这个名字。
     */
    host: '127.0.0.1',
    port: 1420,
    strictPort: true,
    watch: { ignored: ['**/src-tauri/**'] },
  },
  build: {
    // WebView2 是 Chromium，WKWebView 是 WebKit，取两者都稳的基线。
    target: 'es2020',
    sourcemap: false,
    /**
     * woff2 一律作为**独立文件**产出，绝不内联成 data URI。
     *
     * 导出带公式的文档时要把 20 个 KaTeX 字体内联进 HTML，字节是
     * `core/export/katex-css.ts` 用 `?url` 拿到 URL 之后 `fetch` 下来的。
     * 而 Vite 默认把小于 4 KB 的资源内联成 `data:` URI ——
     * `KaTeX_Size3-Regular.woff2` 正好是 3624 字节，会踩中这条线。
     *
     * 一旦它变成 data URI，`fetch('data:…')` 会被 Tauri 的 CSP 拦下来
     * （`connect-src 'self'` 不含 `data:`），于是**这个字体静默缺失**：
     * 导出的文件里 `\sum` 之类的大号符号会掉字形，而导出提示什么都不说
     * （它只知道「这个字体没拿到」，不知道「为什么」）。
     *
     * 返回 `false` = 永不内联；返回 `undefined` = 用默认规则。
     */
    assetsInlineLimit: (filePath) => (filePath.endsWith('.woff2') ? false : undefined),
  },
  test: {
    environment: 'node',
    include: ['packages/**/*.test.ts', 'packages/**/*.test.tsx', 'src/**/*.test.ts', 'src/**/*.test.tsx'],
    // 补 jsdom 缺失的 DOM 接口（`getClientRects` 等）。ProseMirror 的
    // `scrollIntoView()` 依赖它们，不补的话「查找跳转 / 工具条格式化」这类
    // 用例会在一个与被测代码无关的地方抛错。细节见该文件头部。
    setupFiles: ['./src/test/jsdom-gaps.ts'],
    // 导出路径要验证「版式 CSS 被内联进单文件 HTML」，而这条链路走的是
    // `import styles from './document-styles.css?raw'`。vitest 默认 css:false，
    // 会把 CSS 模块替换成空实现，`?raw` 也拿不到内容 —— 测试会假绿。
    css: true,
  },
});
