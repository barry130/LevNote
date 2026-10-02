/**
 * ESLint 扁平配置。
 *
 * ## 为什么现在才加
 *
 * 代码里早就有 `// eslint-disable-next-line` 注释 —— 说明作者一直是按 ESLint 的
 * 规则在写，只是没把它配置起来。这意味着「有规则但不执行」：
 * 新加入的人不知道边界在哪，而漏掉的规则也没有任何东西会提醒。
 *
 * ## 规则选择的取向
 *
 * 只开**能抓到真问题**的规则，不开**风格**规则。
 * 缩进、引号、分号这些交给 `.editorconfig` 与格式化工具，
 * 混进 ESLint 会让「跑 lint」变成一件噪声很大、逐渐被忽略的事。
 *
 * 最重要的三条：
 * - `no-floating-promises`：异步调用没接住是这类应用最常见的真实 bug
 *   （保存失败被吞掉，用户以为存上了）。
 * - `no-misused-promises`：把 async 函数直接当事件处理器传，错误会静默消失。
 * - `eqeqeq`：`==` 在 `null` 判断上的宽松语义容易埋雷。
 *
 * ## 与 TypeScript 7 共存（**本配置最重要的一条背景**）
 *
 * 本项目用 TypeScript 7，而 TypeScript 7 **不提供 JS API**
 * （官方原话：*"While TypeScript 7.0 is here, it does not ship with an API."*）。
 * `typescript-eslint` 需要 API，因此它在 `versionMajorMinor >= 7` 时**直接抛错**，
 * 连纯语法规则都跑不起来 —— 这不是 peer 警告，是硬性 `throw`。
 *
 * 官方给的过渡方案是 npm 别名，让两者并存（见 `package.json`）：
 *
 * ```json
 * "typescript":            "npm:@typescript/typescript6@^6.0.2",  // 给工具用的 TS6 API
 * "@typescript/native":    "npm:typescript@^7.0.2"                // 给构建用的 TS7 tsc
 * ```
 *
 * 于是 `require('typescript')` 拿到 6.x（linter 可用），而 `npx tsc` 仍是 7.x
 * （原生编译器，快得多）。**顺带消掉了 peer 冲突**：`typescript-eslint` 的
 * peer 范围是 `>=4.8.4 <6.1.0`，别名后的 `typescript` 正好落在里面，
 * 因此不需要 `legacy-peer-deps`，`npm ci` 可以直接跑。
 *
 * **重审条件**：TypeScript 7.1 预计提供新 API；`typescript-eslint` 宣布支持
 * TS ≥ 7.1 后（跟踪 issue #10940），把这两条别名还原成普通的
 * `"typescript": "^7.x"` 即可。若某次升级后 lint 开始崩溃，
 * 第一件事是退到 `tseslint.configs.recommended`（纯语法规则），而不是加 `--force`。
 */

import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'src-tauri/target/**',
      'src-tauri/gen/**',
      'coverage/**',
      'docs/samples/**',
      // ESLint **不读 `.gitignore`** —— 本地生成的目录必须在这里再列一遍，
      // 否则 `npm run check` 会去 lint 它们。踩过两次：
      // ① `.v3-edge-*/` 是 Edge 的调试用户目录（每轮换一个新的 ——
      //    复用过的那份可能被沙箱锁住，删不掉也写不进），
      //    里面躺着几十个扩展的 `.js`，全都报「not found by the project service」；
      // ② `.v3-out/` 是 V3 跑出来的 JSON 与临时脚本。
      '.v3-edge-*/**',
      '.v3-out/**',
      // ③ `.v2-out/` 是 V2（mind-elixir 压测）跑出来的 JSON 读数。
      '.v2-out/**',
      // ④ `.v1-out/` 是 V1（drawio 查看器）下下来的第三方产物（viewer.min.js 2.7 MB、
      //    stencils.min.js 7.3 MB）、样本与读数，外加几个一次性的下载/诊断脚本。
      //    那些产物是**被观测的对象**，不是我们的代码 —— lint 它们只会得到一堆噪声。
      '.v1-out/**',
    ],
  },

  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,

  {
    languageOptions: {
      parserOptions: {
        // 让类型感知的规则能用上 tsconfig 的路径别名
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
      globals: { ...globals.browser, ...globals.node },
    },
    rules: {
      eqeqeq: ['error', 'always', { null: 'ignore' }],
      'no-console': ['warn', { allow: ['warn', 'error', 'info'] }],

      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/consistent-type-imports': ['error', { prefer: 'type-imports' }],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrors: 'all' },
      ],

      /**
       * `!` 在这类代码里通常意味着「我知道这个索引一定存在」，
       * 而 tsconfig 开了 `noUncheckedIndexedAccess` —— 两者是有意的搭配。
       */
      '@typescript-eslint/no-non-null-assertion': 'off',

      /**
       * `unbound-method` —— **开着**，而且这是本配置里最值得留下的一条。
       *
       * 它报的是「把对象上的方法拆下来当值单传」。React 里最常见的触发写法是
       * 回调 prop 用**方法简写**声明：
       *
       * ```ts
       * interface SidebarProps { onOpenFolder(): void }   // 方法签名 → 被认为「有 this」
       * export function Sidebar({ onOpenFolder }: SidebarProps) { ... }  // 解构 → 报错
       * ```
       *
       * 这条规则一开始是**关掉**的（在 React 组件里几乎全是误报）。
       * 后来把 11 个文件里的回调统一改成**属性签名**，它就不再误报了，
       * 于是打开。**关规则只是把噪声藏起来，改写法才是真的修好。**
       *
       * 顺带的好处：方法参数是**双变**的，属性参数是**逆变**的（`strictFunctionTypes`）。
       * 换成属性签名之后，「回调参数类型必须真的对得上」开始被编译器检查 ——
       * 实测这次改动没有暴露新的类型错误，说明之前的标注本来就是对的。
       *
       * **配套约定**（写在这里，因为它只对这类成员成立）：
       * - options / props 里的**回调** → 属性签名（它们会被取出来单传或解构）；
       * - **返回实例上的方法** → 保持方法简写（只会以 `instance.foo()` 调用，不会被拆下来）。
       */
      '@typescript-eslint/unbound-method': 'error',

      // 这几条在本项目里误报太多：Tauri 命令返回的 JSON 是运行时才知道形状的
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-argument': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      // `async` 函数在没有 await 时返回 Promise，是有意的接口约定（例如 DocumentProvider）
      '@typescript-eslint/require-await': 'off',
      // 空接口 / 空对象类型在「标记类型」的用法里是合理的
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-redundant-type-constituents': 'off',
      // 测试里大量使用 `!` 与假对象，这条规则会让测试文件变成噪声源
      '@typescript-eslint/no-explicit-any': 'warn',
    },
  },

  {
    files: ['**/*.test.ts', '**/*.test.tsx', 'tests/**/*'],
    rules: {
      '@typescript-eslint/no-floating-promises': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',

      /**
       * 关掉 `only-throw-error` —— 只关测试。
       *
       * 测试里的 `FakeProvider` 抛的是**普通对象字面量**：
       *
       * ```ts
       * if (!file) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
       * ```
       *
       * 这是**故意的，而且是对的做法**：Tauri 把 Rust 的 `Err(AppError)`
       * 序列化过来时，JS 侧拿到的就是这么一个普通对象。假对象的职责是复现
       * 真实边界的形状，而不是复现我们这边「抛什么」。用 `new AppError(...)`
       * 反而会让测试测不到「线上对象进来」这条路径。
       *
       * 生产代码里这条规则**保持开启** —— 见 `AppError` 类的注释：
       * 那边该抛真正的 `Error` 子类。区分开正是为了这个。
       */
      '@typescript-eslint/only-throw-error': 'off',
    },
  },

  {
    /**
     * React Hooks。
     *
     * **只开这两条，不用插件的 `recommended` 预设。**
     * v7 的 `recommended` 顺带把整套 React Compiler 规则（`purity`、`immutability`、
     * `set-state-in-render`、`preserve-manual-memoization` …）一起打开了。
     * 那套规则是按「已按编译器心智模型写好的代码」设计的，对一个还在用
     * `useRef` 手动挂载编辑器的项目会瞬间产出上百条意见 ——
     * 结果就是 lint 重新变成噪声，与「让它成为阻塞门」的目标正好相反。
     *
     * 开这两条的理由很具体：
     * - `rules-of-hooks`：在条件 / 循环里调 hook 是**真实崩溃**，不是风格问题。
     * - `exhaustive-deps`：代码里本来就有 4 处
     *   `eslint-disable-next-line react-hooks/exhaustive-deps` —— 说明作者
     *   一直在按这条规则写。之前插件没装，那些 disable 注释反而变成了
     *   「规则不存在」的报错。开上之后这些压制才真正生效、可被复查。
     *
     * **重审条件**：等编辑器挂载改成官方 `useSyncExternalStore` / 不再依赖
     * `useRef` + 手写 `useEffect` 之后，可以再评估接入 `recommended-latest`。
     */
    files: ['**/*.tsx'],
    plugins: { 'react-hooks': reactHooks },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
    },
  },

  {
    // 配置与脚本文件不在 tsconfig 的 include 里，开类型感知会直接报「找不到 project」
    files: ['*.config.*', 'scripts/**/*.mjs'],
    ...tseslint.configs.disableTypeChecked,
    rules: {
      ...tseslint.configs.disableTypeChecked.rules,
      // 命令行脚本的产出就是 stdout，打印是它的正常行为，不是「忘了删的调试语句」
      'no-console': 'off',
    },
  },
);
