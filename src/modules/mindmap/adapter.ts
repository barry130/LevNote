/**
 * `.mindmap` 信封 ⇄ mind-elixir 数据的**薄适配层**（ADR 0007 的后果 2）。
 *
 * 存在的理由只有一条：**内核的版本变化只能影响这个文件**。
 * 6.0.0 正在迭代（一个月内 next.1 → next.9），升级它不该让用户的文件失效。
 *
 * ## 契约是「带旧信封进去、拿新信封出来」
 *
 * ```
 * toKernelData(信封)                    → 内核数据        （打开时）
 * fromKernelData(上一份信封, 内核数据)   → 新信封          （保存时）
 * ```
 *
 * `fromKernelData` **必须**拿到上一份信封，这不是为了方便，是**必需**：
 * 内核的 `getData()` 返回的是一个**写死的字段列表**
 * （`nodeData` / `arrows` / `summaries` / `direction` / `theme` / `compact` / `meta`），
 * 我们展开到它上面的**自定义顶层键会原样消失**。
 * 那些键住在 `MindmapDocument.extra` 里，只有上一份信封有。
 *
 * 节点层没有这个问题：`nodeData` 是活对象，未知键会被 `getData()` 一并带回来。
 * 所以节点层的 `extra` 是**现读**的，顶层的是**继承**的 —— 差别是有原因的，不是随手写的。
 *
 * ## 两条映射规则
 *
 * 1. **内核认识的键，原样进出。** `hyperLink` / `tags` / `icons` / `image` /
 *    `metadata` / `branchColor` / `dangerouslySetInnerHTML` … 我们一个都不解释，
 *    但一个都不丢 —— 它们从节点的 `extra` 展开进内核，又被内核带回来。
 *    §7.1 的骨架刻意只固定最小公因式（`id` / `topic` / `children`），
 *    而「最小公因式」**不等于**「只许有这些」。
 * 2. **折叠状态是改名，不是两个语义。** 我们写 `collapsed`，内核写 `expanded`，
 *    两者互为反义。**只有一个方向会写出值**：`expanded === false` → `collapsed: true`。
 *    反过来 `expanded === true` 不写 —— 内核的默认就是展开，写出来只是噪声，
 *    而且会给「打开 → 保存」平白加上一次 diff。
 */

import type { MindElixirData, NodeObj } from 'mind-elixir';

import type { KernelRejectReason } from '../../core/editor/types.js';
import { parseMindmap } from '../../core/mindmap/parse.js';
import { MINDMAP_DEFAULT_LAYOUT, type MindmapDocument, type MindmapNode } from '../../core/mindmap/types.js';

/**
 * 内核的方向编号。
 *
 * 单独起一个别名而不是散落 `0 | 1 | 2 | 3`：这两张表是**互逆**的，
 * 而它们的一致性只能靠类型与测试保证 —— 让两处都写同一个别名，
 * 至少「有一处漏改」会在类型上就露出来。
 */
export type Direction = 0 | 1 | 2 | 3;

/**
 * 布局 → 内核方向。
 *
 * `direction` 是 mind-elixir 的概念（0 左 / 1 右 / 2 双向 / 3 向下），
 * `layout` 是我们的。这张表**必须**与 `MINDMAP_LAYOUTS` 的下标一一对应。
 */
const DIRECTION_BY_LAYOUT: ReadonlyMap<string, Direction> = new Map<string, Direction>([
  ['logicalStructureLeft', 0],
  ['logicalStructureRight', 1],
  [MINDMAP_DEFAULT_LAYOUT, 2],
  ['orgChart', 3],
]);

/**
 * 内核方向 → 布局。**下标就是方向编号**，顺序不能动。
 *
 * 默认方向刻意取 `2`（双向）而不是内核自己的默认值 `1`（向右）：
 * `layout` 缺省时我们要的是「中心发散的思维导图」，不是一张左对齐的树。
 */
const LAYOUT_BY_DIRECTION: readonly string[] = [
  'logicalStructureLeft',
  'logicalStructureRight',
  MINDMAP_DEFAULT_LAYOUT,
  'orgChart',
];

const DEFAULT_DIRECTION: Direction = 2;

/**
 * 节点上**不属于** `extra` 的键。
 *
 * - `parent`：内核自己维护的父引用（`getData()` 会删掉它，但我们再防一次 ——
 *   它是**循环引用**，一旦漏进 `extra`，`JSON.stringify` 会直接抛错）；
 * - `expanded`：折叠状态，由 `collapsed` 承载（改名，见文件头）。
 *
 * 其余「我们不认识」的键全部走 `extra`。
 */
const NON_EXTRA_NODE_KEYS: ReadonlySet<string> = new Set(['id', 'topic', 'children', 'note', 'collapsed', 'style', 'parent', 'expanded']);

/**
 * 顶层上**不属于** `extra` 的键。
 *
 * `theme` 与 `compact` 刻意**不在此列** —— 它们会留在 `extra` 里，
 * 但两个方向都不参与渲染（见 `toKernelData` / `fromKernelData` 里的说明）。
 */
const NON_EXTRA_TOP_KEYS: ReadonlySet<string> = new Set(['nodeData', 'direction']);

// ---------------------------------------------------------------------------
// 信封 → 内核
// ---------------------------------------------------------------------------

/**
 * 往内核节点上写一个「我们不认识」的键。
 *
 * 需要一次断言，因为 `NodeObj` 没有索引签名 —— 而**透传桶的全部意义**
 * 就是装我们类型里没有的键。把这个断言关在一个函数里，
 * 好过在四五个地方各写一遍（也好看清它只有这一处）。
 */
function setUnknown(target: NodeObj, key: string, value: unknown): void {
  (target as unknown as Record<string, unknown>)[key] = value;
}

function toKernelNode(node: MindmapNode): NodeObj {
  const children = node.children.map(toKernelNode);
  const out: NodeObj = { id: node.id, topic: node.topic, children };

  // 透传桶展开在前，已知字段写在后 —— 顺序让「已知字段赢」成为结构上的事实，
  // 而不是靠一句注释保证。（序列化器那边已经禁止同一个键有两个来源，
  // 所以这里其实是防御性的。）
  for (const [key, value] of Object.entries(node.extra)) {
    if (NON_EXTRA_NODE_KEYS.has(key)) continue;
    setUnknown(out, key, value);
  }

  if (node.note !== undefined && node.note !== '') out.note = node.note;
  if (node.style !== undefined && Object.keys(node.style).length > 0) out.style = node.style;
  if (node.collapsed === true) out.expanded = false;

  return out;
}

/**
 * 信封 → 内核数据。
 *
 * `theme` / `compact` 被**刻意跳过**：它们是视图偏好，不是文档内容。
 * 主题由轻笺的设置页统一管（ADR 0021），文件里再存一份会让
 * 「深色主题下打开一张图突然变亮」。它们仍然留在 `extra` 里不丢（见 `fromKernelData`），
 * 只是不参与渲染 —— 这是有意的取舍，写在 `docs/format-spec.md` §7.1。
 */
export function toKernelData(document: MindmapDocument): MindElixirData {
  const data: MindElixirData = {
    nodeData: toKernelNode(document.root),
    direction: DIRECTION_BY_LAYOUT.get(document.layout) ?? DEFAULT_DIRECTION,
  };

  for (const [key, value] of Object.entries(document.extra)) {
    if (NON_EXTRA_TOP_KEYS.has(key)) continue;
    if (key === 'theme' || key === 'compact') continue;
    // `arrows` / `summaries` / `meta` 在这里回到内核 —— 少了这一步，
    // 用户画的箭头与摘要会「存得住但看不见」。
    (data as unknown as Record<string, unknown>)[key] = value;
  }

  return data;
}

// ---------------------------------------------------------------------------
// 内核 → 信封
// ---------------------------------------------------------------------------

function fromKernelNode(node: NodeObj): MindmapNode {
  const extraEntries: [string, unknown][] = [];
  for (const [key, value] of Object.entries(node as unknown as Record<string, unknown>)) {
    if (NON_EXTRA_NODE_KEYS.has(key)) continue;
    extraEntries.push([key, value]);
  }

  const out: MindmapNode = {
    id: node.id,
    topic: node.topic,
    children: (node.children ?? []).map(fromKernelNode),
    // 用 `fromEntries` 而不是逐个赋值：它走 `CreateDataProperty`，
    // 于是 `__proto__` 这种键只会成为一个普通属性，而不会去改原型。
    extra: Object.fromEntries(extraEntries),
  };

  if (node.note !== undefined && node.note !== '') out.note = node.note;
  if (node.style !== undefined && Object.keys(node.style).length > 0) out.style = node.style;
  if (node.expanded === false) out.collapsed = true;

  return out;
}

/** 有值就写、没值就删。用 `delete` 而不是写 `undefined` —— 后者会序列化成 `null`。 */
function setOrDelete(bucket: Record<string, unknown>, key: string, value: unknown): void {
  const empty = value === undefined || (Array.isArray(value) && value.length === 0);
  if (empty) {
    delete bucket[key];
    return;
  }
  bucket[key] = value;
}

/**
 * 内核数据 → 新信封。
 *
 * ## `layout` 的一条特别规则
 *
 * `previous.layout` 认不出来时（别的工具 / 更高版本写的），**只要方向没被动过就原样保留**。
 * 「方向没被动过」是可判定的：认不出的布局打开时按默认方向渲染，
 * 所以 `direction === 默认` 就意味着用户没按过 `Ctrl+方向键`。
 *
 * 这条规则同时满足两个要求：方向没动 → 零 diff（原值还在）；
 * 方向动了 → 用户的动作赢（写映射值，此时原值本来就已不准确）。
 */
export function fromKernelData(previous: MindmapDocument, data: MindElixirData): MindmapDocument {
  const extra: Record<string, unknown> = { ...previous.extra };
  setOrDelete(extra, 'arrows', data.arrows);
  setOrDelete(extra, 'summaries', data.summaries);
  setOrDelete(extra, 'meta', data.meta);

  const direction = data.direction ?? DEFAULT_DIRECTION;
  const known = DIRECTION_BY_LAYOUT.has(previous.layout);

  return {
    root: fromKernelNode(data.nodeData),
    layout: known || direction !== DEFAULT_DIRECTION ? (LAYOUT_BY_DIRECTION[direction] ?? MINDMAP_DEFAULT_LAYOUT) : previous.layout,
    extra,
    // 按源文件的版本写回，不擅自升级（见 `MindmapDocument.version`）
    version: previous.version,
  };
}

// ---------------------------------------------------------------------------
// 「这份文件能不能安全地交给内核」
// ---------------------------------------------------------------------------

/**
 * 这棵树交给内核会不会出问题，以及**不会的话是因为什么**（ADR 0043）。
 *
 * `reason` 从前的类型是 `string`（一串写好的中文）。改成 `KernelRejectReason`
 * 之后它变成了**数据**：id 以字段的形式留着，话由外壳在渲染的那一刻才翻。
 * 两条理由：① 那串中文原本就没活着出去过 —— `canRenderInKernel` 只取 `.ok`，
 * 它一直是被扔掉的；② 一旦它变成契约上的字符串，语言就被钉死在解析那一刻，
 * 而解析可能发生在用户切语言之前。
 */
export type KernelSafety = { ok: true } | { ok: false; reason: KernelRejectReason };

/**
 * 节点 id 里不允许出现的字符。
 *
 * **不是我们挑剔，是内核的选择器受不了。** mind-elixir 的 `findEle(id)` 是
 * `el.querySelector('[data-nodeid="me' + id + '"]')` —— 而 `id` 是**拼进属性选择器**的。
 * 里面出现 `"` 会让选择器语法错误（直接抛 `SyntaxError`），
 * 出现 `\` 会变成转义符。**空格的没事**（在引号里），`#` / `.` 也没事。
 *
 * 为什么不在适配层里把它换掉？因为 id 是**旁表的键** ——
 * `arrows` / `summaries` 都按 id 引用节点。换掉 id 就得同步重写那些引用，
 * 而重写写错了的后果（箭头指向别的节点）比「这个文件用纯文本打开」严重得多。
 * 所以这里的选择是**如实拒绝**：`canHandle` 返回 false，
 * 用户拿到的是纯文本编辑器 —— 内容一个字节不少，还能顺手把 id 改掉。
 */
const UNSAFE_ID_CHARS = /["\\]/;

/**
 * 检查这棵树交给内核会不会出问题。
 *
 * 两条：id 里有危险字符、id 重复。
 *
 * **id 重复比看起来严重**：`findEle` 用 `querySelector`，它只返回**第一个**匹配。
 * 于是用户拖第二个节点，动的是第一个 —— 而界面上没有任何东西提示这件事。
 * 这属于「静默地把操作做到别的地方去」，正是硬约束 2 要拦的形态。
 *
 * 调用点是 `EditorAdapter.canHandle`（见 `index.ts`）：返回 `{ok: false}` 时
 * 注册表会继续往下找，最终落到纯文本兜底 —— **文件照样能打开**。
 */
export function checkKernelSafety(document: MindmapDocument): KernelSafety {
  const seen = new Set<string>();

  const walk = (node: MindmapNode): KernelRejectReason | null => {
    if (UNSAFE_ID_CHARS.test(node.id)) {
      return { kind: 'mindmap-unsafe-id', id: node.id };
    }
    if (seen.has(node.id)) {
      return { kind: 'mindmap-duplicate-id', id: node.id };
    }
    seen.add(node.id);
    for (const child of node.children) {
      const bad = walk(child);
      if (bad !== null) return bad;
    }
    return null;
  };

  const reason = walk(document.root);
  return reason === null ? { ok: true } : { ok: false, reason };
}

/**
 * 给 `canHandle` 用的窄化包装：只关心「能不能」。
 *
 * 注意它**丢掉了理由** —— 所以 `canHandleReason` 那条路不要走这里，
 * 走 `checkKernelSafety` 拿完整的 `KernelSafety`。
 */
export function canRenderInKernel(document: MindmapDocument): boolean {
  return checkKernelSafety(document).ok;
}

/**
 * 「这份 `.mindmap` 为什么不能交给导图内核」—— 返回 `null` 表示能。
 *
 * ## 为什么判定与理由必须是同一个函数
 *
 * `canHandle` 与 `canHandleReason` 是契约上的两个方法，外壳会分别调用它们。
 * 若各自实现一遍，它们迟早会漂移到「一个说不行、另一个说不出为什么」——
 * 而那正是 ADR 0043 要消灭的形态（横幅说的与实际不符）。
 * 所以这里是**唯一**的判定入口：`index.ts` 的两个方法各自从它的返回值取一半。
 *
 * 代价是 `canHandleReason` 被调用时会**再解析一次** JSON。
 * 那是刻意的：解析是纯函数，重算的结果必然一致；而缓存一份解析结果
 * 要引入「按什么键缓存、什么时候失效」这套状态，得不偿失。
 */
export function describeMindmapVeto(text: string): KernelRejectReason | null {
  // 新建的空文件不是「坏文件」，是「还没开始写」—— 与 `canHandle` 的口径一致
  if (text.trim() === '') return null;
  const parsed = parseMindmap(text);
  if (parsed.document === null) return { kind: 'mindmap-invalid' };
  const safety = checkKernelSafety(parsed.document);
  return safety.ok ? null : safety.reason;
}

/** 供测试与组件复用：布局 → 内核方向。 */
export function directionOf(layout: string): Direction {
  return DIRECTION_BY_LAYOUT.get(layout) ?? DEFAULT_DIRECTION;
}
