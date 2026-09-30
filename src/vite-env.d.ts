/// <reference types="vite/client" />

/**
 * 应用版本，构建期由 `vite.config.ts` 的 `define` 注入。
 *
 * 来源是 `package.json` 的 `version`（唯一来源，见那里的注释）。
 * 声明放在这里而不是某个 `.ts` 里：它是**构建期常量**，不是一个模块，
 * 任何 `import` 都会变成运行时查找，而它其实在打包时就已经被替换成字面量了。
 */
declare const __QINGMO_VERSION__: string;
