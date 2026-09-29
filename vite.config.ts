// 用 vitest/config 的 defineConfig：它扩展了 Vite 的配置类型，才有 `test` 字段。
// 运行时二者是同一个对象，不影响 Tauri 拉起 dev server。
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'node:url';

// Tauri 会以固定端口拉起 dev server；src-tauri 的变动不该触发前端热更新。
export default defineConfig({
  plugins: [react()],
  clearScreen: false,
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@qingmo/interop-spec': fileURLToPath(
        new URL('./packages/interop-spec/src/index.ts', import.meta.url),
      ),
    },
  },
  server: {
    port: 1420,
    strictPort: true,
    watch: { ignored: ['**/src-tauri/**'] },
  },
  build: {
    // WebView2 是 Chromium，WKWebView 是 WebKit，取两者都稳的基线。
    target: 'es2020',
    sourcemap: false,
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
