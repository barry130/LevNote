/// <reference types="vite/client" />

/**
 * 应用版本，构建期由 `vite.config.ts` 的 `define` 注入。
 *
 * 来源是 `package.json` 的 `version`（唯一来源，见 `vite.config.ts` 的注释）。
 *
 * ## 为什么挂在 `import.meta.env` 上，而不是一个裸的全局常量
 *
 * 一开始声明的是 `declare const __QINGMO_VERSION__: string`。它**只在 build
 * 与 vitest 里被替换**，dev server 不认 —— 实测请求 `/src/ui/SettingsDialog.tsx`
 * 拿回来的还是裸标识符。于是开发时渲染到那一行就抛 `ReferenceError`，
 * 没有错误边界，React 把整棵树卸掉：**点一下设置，应用直接没了**。
 *
 * 写成 `import.meta.env.*` 之后三个环境都替换；而且它是**属性访问**，
 * 万一将来 define 漏了，结果只是 `undefined`，不会把应用带走。
 *
 * 声明放在这里而不是某个 `.ts` 里：它是**构建期常量**，不是一个模块，
 * 任何 `import` 都会变成运行时查找，而它其实在打包时就已经被替换成字面量了。
 */
interface ImportMetaEnv {
  readonly QINGMO_VERSION: string;
}
