# 0013 · 在 TypeScript 7 之上运行 ESLint

- **状态**：已采纳
- **日期**：2026-09-29
- **相关**：[0011](./0011-code-kernel-and-graceful-downgrade.md)、[`../open-decisions.md`](../open-decisions.md)

---

## 背景

本项目使用 **TypeScript 7.0.2**。TypeScript 7 是原生编译器重写，**它不提供 JS API**
（官方原文：*"While TypeScript 7.0 is here, it does not ship with an API."*）。

而 `typescript-eslint` 需要 API。它的入口有一道硬守卫：

```js
// node_modules/typescript-eslint/dist/index.js
const ts = __importStar(require("typescript"));
const [versionMajor] = ts.versionMajorMinor.split('.').map(Number);
if (versionMajor >= 7) {
  console.error('typescript-eslint does not support TS 7.0. …');
  throw new Error('typescript-eslint does not support TS 7.0.');
}
```

注意这是**模块加载期的 `throw`**，不是 peer 警告 —— 也就是说
**连纯语法规则（不需要类型信息的那一半）也跑不起来**。

同时，仓库里当时**根本没有 `eslint.config.js`**，而源码里已经散落着
`// eslint-disable-next-line react-hooks/exhaustive-deps` 这类注释。
两者合起来是一个很隐蔽的状态：**有规则、有抑制注释，但没有东西在执行规则**。
新加入的人看不到边界在哪，漏掉的规则也没有任何东西会提醒。

---

## 选项

| | 方案 | 评价 |
|---|---|---|
| A | 把项目降级到 TypeScript 6 | 为了 lint 放弃原生编译器的速度，本末倒置 |
| B | `--legacy-peer-deps` / `--force` 强行装 | **没用**。冲突不是 peer 警告，是运行时硬 `throw`，绕不过去 |
| C | 官方别名方案：让 TS6 与 TS7 并存 | ✅ 采纳 |
| D | 放弃 ESLint，只保留 `tsc` | 等于承认那些 disable 注释永远无效；规则彻底失效 |

---

## 决策

**采用 C —— 按官方过渡方案用 npm 别名让两个 TypeScript 并存：**

```json
"devDependencies": {
  "typescript":         "npm:@typescript/typescript6@^6.0.2",
  "@typescript/native": "npm:typescript@^7.0.2"
}
```

- `node_modules/typescript` → TS **6** 的 API，供 `typescript-eslint` 等工具使用；
- `node_modules/@typescript/native` → TS **7**，提供 `tsc` 可执行文件。

因为别名后的包名不同，两者不冲突；npm 会同时装下。

---

## 为什么

1. **B 是错的，而且错得不容易发现。** 直觉会认为「peer 冲突 → `--legacy-peer-deps`」。
   但这里的冲突被 `typescript-eslint` 自己在运行期检查了，加 `--force` 只会
   让安装通过、运行时照样抛错，把一个「装不上」的问题变成一个「装上了但一跑就炸」
   的问题 —— 后者更难查。

2. **C 顺带解决了 peer 冲突。** `typescript-eslint@8` 的 peer 范围是
   `>=4.8.4 <6.1.0`。别名之后 `typescript` 解析到 6.0.x，**正好落在范围里**，
   因此 `npm ci` 不需要任何 `legacy-peer-deps`，CI 可以直接跑。

3. **D 的代价被低估了。** 代码里那些 disable 注释是一份「作者知道规则存在」的证据。
   放弃 lint 不是「少一道检查」，而是让已有的意图变成死代码。

---

## 实测结果

| 检查 | 结果 |
|---|---|
| `require('typescript').version` | `6.0.2` ← linter 拿到的 |
| `npx tsc --version` | `Version 7.0.2` ← 构建拿到的 |
| `node_modules/.bin` | 同时有 `tsc` 与 `tsc6` |
| `npx eslint --version` | `v10.11.0` |
| `npx eslint .` | 能跑出真实结果（见下），不再抛错 |

---

## 代价（如实记录）

1. **`package.json` 里 `typescript` 这个名字指向 TS 6**，直接读依赖表的人会困惑。
   缓解：两条别名都带注释说明，且本 ADR 记录原因。
2. **类型感知的规则实际跑在 TS 6 的类型系统上。** 若代码使用了 TS 7 独有的语法，
   linter 可能解析失败。当前语料未触发，但这是一条真实存在的边界。
3. **这是一个过渡期方案。** 官方明说 TS 7.1 会提供「一个新的、不同的 API」。
4. **lint 与 `tsc` 可能对同一段代码给出不同的类型**（2026-09-29 实测到）。
   lint 跑在 TS 6 上、`tsc` 跑在 TS 7 上，两者的控制流分析并不等价。
   已撞到过一次：`src/modules/markdown/setup.ts` 里「在回调中给外部 `let` 赋值 →
   回到外层收窄 → 再做 `+ 1`」这种写法，**TS 6 把变量收窄成 `null`，
   排除 null 后得到 `never`**，而 TS 7 正确得到 `number`。
   结果是 `tsc --noEmit` 通过、lint 报错。
   影响：**「`tsc` 通过」不再蕴含「lint 通过」**，两条命令都必须跑
   （`npm run check` 现在就是这么做的）。
   遇到这类分歧时**改写法**，不要去赌哪个版本的 CFA 更聪明。

---

## 重审条件

- **TypeScript 7.1 提供新 API、且 `typescript-eslint` 宣布支持 TS ≥ 7.1 之后**
  （跟踪 [typescript-eslint#10940](https://github.com/typescript-eslint/typescript-eslint/issues/10940)），
  把两条别名还原为普通的 `"typescript": "^7.x"`，并删掉本节。
- **若某次升级后 lint 开始崩溃**：第一件事是退到 `tseslint.configs.recommended`
  （纯语法规则，不需要类型信息），**而不是**加 `--force`。见上面「为什么」第 1 条。

---

## 首次 lint 的真实结果 —— 以及后续的清零

`npx eslint .` 首轮 → **64 problems（61 errors, 3 warnings）**。
其中 31 项是 `unbound-method` 的误报（当时先关掉规则），**剩余 33 项是真的**。

**2026-09-29：33 项全部清零，并且 `unbound-method` 也被重新打开。**
配置里不再有任何被关掉的规则，`npm run check` 与 CI 都会因 lint 失败而失败。
下表是每一条的最终去向：

| 数量 | 规则 | 最终处理 |
|---|---|---|
| 31 | `@typescript-eslint/unbound-method` | **重新打开**。11 个文件里的回调 prop 从方法简写改成属性签名（`onOpenFolder: () => void`）。附带的**逆变**检查没有暴露新的类型错误 —— 说明原来的标注本来就是对的 |
| 11 | `@typescript-eslint/only-throw-error` | 生产代码引入 `AppError` 类（`extends Error`，与 Rust 的 `AppError::new` 对称）；测试里的 `FakeProvider` **保留抛普通对象**并加文件级豁免 —— 那是在模拟 Tauri 边界的 serde 输出，抛对象才是保真 |
| 7 | `@typescript-eslint/no-unnecessary-type-assertion` | `lint:fix` 自动去除。顺带发现 `document/store.ts` 的 `TargetLineEnding` 因此变成未使用导入（已删） |
| 4 | `react-hooks/exhaustive-deps`「规则不存在」 | 装上 `eslint-plugin-react-hooks`。**只开 `rules-of-hooks` + `exhaustive-deps`**，不开它的 `recommended` —— 那会连带打开整套 React Compiler 规则（`purity` / `immutability` / `set-state-in-render` …），瞬间产出上百条意见，等于把 lint 重新变成噪声 |
| 3 | `@typescript-eslint/no-base-to-string` | 改成 `typeof` 收窄。`String(对象)` 会得到 `'[object Object]'`，那种「看起来有值其实是垃圾」的结果比空串更难查 |
| 2 | `no-useless-assignment` | 去掉 `let x = <初值>` 的死初值（try 与 catch 两条路径都会赋值） |
| 2 | `@typescript-eslint/consistent-type-imports` | `lint:fix`；`registry.test.ts` 的 `typeof import(...)` 改成静态 `import type`（类型被完全擦除，不影响 `vi.resetModules()`） |
| 2 | `no-console`（warn） | `scripts/**` 的 override 里关掉 —— 命令行脚本的产出就是 stdout |
| 1 | `@typescript-eslint/restrict-plus-operands` | 提取 `findHeadingPos()` 辅助函数。**这一条本身不重要，重要的是它暴露了上面「代价」第 4 点的双编译器分歧** |
| 2 | 未使用的 disable 指令（warn） | 自动清除，并清掉留下的纯空白行 |

> 这张表本身就是「有 lint 之后才能拿到的信息」。在此之前，上面每一项都无从得知。
>
> 一处勘误：上表最初列过一条「解析错误 ×1」。在后续的稳定复现里没有出现过，
> 重新归类后那 33 项可以被完整解释，不再单列。推断是首轮运行时 `projectService`
> 的冷启动产物，**没有进一步追查** —— 记在这里是为了不让它无声消失。
