/**
 * `.mindmap` → 内存里的树（`docs/format-spec.md` §7.1 / §5 / §6）。
 *
 * ## 三条不可动摇的规矩
 *
 * 1. **绝不静默**（硬约束 2）：认不出的东西进 `extra` 并给提示；类型不对的字段
 *    按 `BAD_FIELD` 提示后**转进 `extra`**（一个都不丢）；只有真的没法表示的东西
 *    才降级成空值，而那时一定是 **error** 级。
 * 2. **打开任何文件都不能失败**（硬约束 3）：本函数**不抛异常**。致命问题返回
 *    `document: null`，由外壳降级到纯文本兜底并说明原因。
 * 3. **不擅自升级**：写回时用**源文件里读到的**版本号，不改成当前版本 ——
 *    把 v1 升级成 v2 会让装了老版本的机器读不了，而用户没要求过这件事。
 *
 * ## `canonical` 是干什么的
 *
 * 它是「源文本是否已经等于我们写出来的样子」。**它不是「文件坏没坏」的判据** ——
 * 一份手写的、缩进用 4 空格的文件只是 `canonical: false`，完全正常。
 *
 * 它的唯一用途：文档**没有被编辑过**就保存时，调用方应当原样写回源文本，
 * 而不是写序列化结果。JSON 的往返做不到逐字节保真（键顺序、缩进、空行都可能变），
 * 所以「打开即保存不产生 diff」这条保证**不能靠序列化器**，只能靠「没改就不重写」。
 */

import { t } from '../i18n/index.js';
import {
  KNOWN_NODE_KEYS,
  KNOWN_TOP_KEYS,
  MINDMAP_DEFAULT_LAYOUT,
  MINDMAP_FORMAT,
  MINDMAP_LAYOUTS,
  MINDMAP_VERSION,
  type LineEnding,
  type MindmapDocument,
  type MindmapIssue,
  type MindmapIssueCode,
  type MindmapNode,
  type ParseMindmapOptions,
  type ParseMindmapResult,
} from './types.js';
import { serializeMindmap } from './serialize.js';

/**
 * 只认「普通对象」。
 *
 * 数组、`null`、以及 `JSON.parse` 永远产不出的东西都不算 ——
 * 但**不检查原型**：`JSON.parse` 产出的对象原型就是 `Object.prototype`，
 * 而用 `Object.create(null)` 造出来的桶在 `JSON.stringify` 下行为一致。
 */
function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * 显式的数组判据。
 *
 * 不直接用 `Array.isArray(x)`：它在 TS 里把 `unknown` 窄化成 `any[]`，
 * 于是一路 `any` 泄漏到循环变量上，被 `no-unsafe-*` 系列规则抓住。
 * 用一个返回 `unknown[]` 的类型谓词，就一个 `any` 都不会出现。
 */
function isUnknownArray(value: unknown): value is unknown[] {
  return Array.isArray(value);
}

/** 行尾是**文件属性**（§4）。整篇第一个换行决定它，混排时不猜（保存层按原样写回）。 */
export function detectLineEnding(text: string): LineEnding {
  const at = text.indexOf('\n');
  return at > 0 && text.charAt(at - 1) === '\r' ? '\r\n' : '\n';
}

/**
 * 给错误消息用的一句话描述。**只用于人看**，不参与任何判断。
 *
 * 末尾刻意**不写** `String(value)` 兜底：`String()` 落到对象上会得到
 * `[object Object]`（`no-base-to-string` 抓的就是这个），而那时它既不比「对象」更清楚，
 * 也会把「未知类型」这件事盖掉。所以只对 number / boolean 求值，其余一律只报 `typeof`。
 */
function describe(value: unknown): string {
  if (value === null) return 'null';
  if (value === undefined) return 'undefined';
  if (Array.isArray(value)) return t('core.mm.descArray', { n: value.length });
  if (typeof value === 'string') {
    return t('core.mm.descString', { value: JSON.stringify(value.length > 40 ? `${value.slice(0, 40)}…` : value) });
  }
  if (typeof value === 'object') return t('core.mm.descObject');
  if (typeof value === 'number' || typeof value === 'boolean') return `${typeof value} ${String(value)}`;
  return typeof value;
}

export function parseMindmap(text: string, options: ParseMindmapOptions = {}): ParseMindmapResult {
  const autoIdPrefix = options.autoIdPrefix ?? 'auto-';
  const issues: MindmapIssue[] = [];
  const lineEnding = detectLineEnding(text);

  /** 致命问题：没有树可画，调用方应降级到纯文本兜底。 */
  const fatal = (code: MindmapIssueCode, message: string): ParseMindmapResult => {
    issues.push({ severity: 'error', code, message });
    return { document: null, issues, lineEnding, canonical: false };
  };

  let raw: unknown;
  try {
    raw = JSON.parse(text) as unknown;
  } catch (error) {
    return fatal('NOT_JSON', t('core.mm.notJson', { reason: error instanceof Error ? error.message : String(error) }));
  }
  if (!isPlainObject(raw)) return fatal('NOT_OBJECT', t('core.mm.notObject', { described: describe(raw) }));
  if (raw.format !== MINDMAP_FORMAT) {
    return fatal('WRONG_FORMAT', t('core.mm.wrongFormat', { described: describe(raw.format), expected: MINDMAP_FORMAT }));
  }

  // --- version --------------------------------------------------------------
  // 非数字 → 按当前版本处理并提示；比我们新 → 照读 + 提示（§5 向前兼容）。
  let version = MINDMAP_VERSION;
  const rawVersion = raw.version;
  if (typeof rawVersion === 'number' && Number.isFinite(rawVersion)) {
    version = rawVersion;
    if (rawVersion > MINDMAP_VERSION) {
      issues.push({
        severity: 'warning',
        code: 'FUTURE_VERSION',
        message: t('core.mm.versionNewer', { version: rawVersion, current: MINDMAP_VERSION }),
      });
    }
  } else if (rawVersion !== undefined) {
    issues.push({
      severity: 'warning',
      code: 'BAD_VERSION',
      message: t('core.mm.versionNotNumber', { described: describe(rawVersion), current: MINDMAP_VERSION }),
    });
  }

  // --- layout ---------------------------------------------------------------
  let layout = MINDMAP_DEFAULT_LAYOUT;
  const rawLayout = raw.layout;
  if (typeof rawLayout === 'string' && rawLayout !== '') {
    layout = rawLayout;
    if (!MINDMAP_LAYOUTS.includes(rawLayout)) {
      issues.push({
        severity: 'warning',
        code: 'UNKNOWN_LAYOUT',
        message: t('core.mm.layoutUnknown', { layout: JSON.stringify(rawLayout), fallback: MINDMAP_DEFAULT_LAYOUT }),
      });
    }
  } else if (rawLayout !== undefined) {
    issues.push({
      severity: 'warning',
      code: 'BAD_FIELD',
      message: t('core.mm.layoutNotString', { described: describe(rawLayout), fallback: MINDMAP_DEFAULT_LAYOUT }),
    });
  }

  // --- 树 -------------------------------------------------------------------
  const seenIds = new Set<string>();
  let autoCounter = 0;

  /** 补一个**确定性**的 id。确定性很重要：否则「打开 → 保存 → 再打开」每次都会产生 diff。 */
  const synthesizeId = (): string => {
    let candidate = `${autoIdPrefix}${autoCounter}`;
    autoCounter += 1;
    while (seenIds.has(candidate)) {
      candidate = `${autoIdPrefix}${autoCounter}`;
      autoCounter += 1;
    }
    return candidate;
  };

  const parseNode = (value: unknown, path: string): MindmapNode | null => {
    if (!isPlainObject(value)) {
      issues.push({
        severity: 'error',
        code: 'BAD_NODE',
        message: t('core.mm.nodeNotObject', { path, described: describe(value) }),
      });
      return null;
    }

    // --- id：结构必需，坏值只能换掉（不能进 extra，否则会写出重复键）------
    const rawId = value.id;
    let id: string;
    if (typeof rawId === 'string' && rawId !== '') {
      id = rawId;
    } else {
      id = synthesizeId();
      issues.push({
        severity: 'warning',
        code: 'MISSING_NODE_ID',
        nodeId: id,
        message: t('core.mm.idMissing', { path, described: describe(rawId), filled: JSON.stringify(id) }),
      });
    }
    if (seenIds.has(id)) {
      issues.push({
        severity: 'warning',
        code: 'DUPLICATE_NODE_ID',
        nodeId: id,
        message: t('core.mm.idDuplicate', { id: JSON.stringify(id) }),
      });
    }
    seenIds.add(id);

    // --- topic：结构必需，数字/布尔转成字符串（比丢掉好）-------------------
    const rawTopic = value.topic;
    let topic: string;
    if (typeof rawTopic === 'string') {
      topic = rawTopic;
    } else if (typeof rawTopic === 'number' || typeof rawTopic === 'boolean') {
      topic = String(rawTopic);
      issues.push({
        severity: 'warning',
        code: 'MISSING_TOPIC',
        nodeId: id,
        message: t('core.mm.topicCoerced', { path, described: describe(rawTopic), topic: JSON.stringify(topic) }),
      });
    } else {
      topic = '';
      issues.push({
        severity: 'warning',
        code: 'MISSING_TOPIC',
        nodeId: id,
        message: t('core.mm.topicMissing', { path, described: describe(rawTopic) }),
      });
    }

    // --- extra：先收未识别的键，再收「认出来了但类型不对」的键 ---------------
    // 两处都走**同一条数组**，最后一次性 `Object.fromEntries`。
    // 不直接 `extra[key] = value`：`key` 万一是 `__proto__`，赋值会去改原型而不是建键
    // （`Object.fromEntries` 用的是 CreateDataProperty，只建自己的键）。
    const extraEntries: [string, unknown][] = [];
    const unknownKeys: string[] = [];
    for (const key of Object.keys(value)) {
      if (KNOWN_NODE_KEYS.has(key)) continue;
      extraEntries.push([key, value[key]]);
      unknownKeys.push(key);
    }
    if (unknownKeys.length > 0) {
      issues.push({
        severity: 'warning',
        code: 'UNKNOWN_KEY',
        nodeId: id,
        message: t('core.mm.unknownKeys', { id: JSON.stringify(id), n: unknownKeys.length, keys: unknownKeys.join('、') }),
      });
    }

    const node: MindmapNode = { id, topic, children: [], extra: {} };

    /**
     * 认出来的**可选**字段：类型对就上，类型不对就转进 `extra`（**一个都不丢**）。
     *
     * 三段各写一遍而不抽一个泛型小工具：抽取版本要用 `as` 断言把 `unknown` 掰成目标类型，
     * 而那正是绕过类型检查的地方 —— 三个字段换一个类型漏洞不划算。
     */
    const badField = (key: string, candidate: unknown): void => {
      extraEntries.push([key, candidate]);
      issues.push({
        severity: 'warning',
        code: 'BAD_FIELD',
        nodeId: id,
        message: t('core.mm.fieldWrongType', { path, key, described: describe(candidate) }),
      });
    };

    const rawNote = value.note;
    if (typeof rawNote === 'string') node.note = rawNote;
    else if (rawNote !== undefined) badField('note', rawNote);

    const rawCollapsed = value.collapsed;
    if (typeof rawCollapsed === 'boolean') node.collapsed = rawCollapsed;
    else if (rawCollapsed !== undefined) badField('collapsed', rawCollapsed);

    const rawStyle = value.style;
    if (isPlainObject(rawStyle)) node.style = rawStyle;
    else if (rawStyle !== undefined) badField('style', rawStyle);

    /**
     * 标记（ADR 0053）：只认「字符串数组」，**不认词表** —— 词表是界面知识
     * （`modules/mindmap/markers.ts`），格式层一个字符都不解释。
     * 数组里混进非字符串（整条进 `extra`，一个不丢）按 `badField` 惯例办；
     * **不逐项过滤**：半好半坏的数组静默丢掉坏的那几项，正是「假装读进来了」。
     */
    const rawIcons = value.icons;
    if (isUnknownArray(rawIcons) && rawIcons.every((entry): entry is string => typeof entry === 'string')) node.icons = rawIcons;
    else if (rawIcons !== undefined) badField('icons', rawIcons);

    // --- children ------------------------------------------------------------
    const rawChildren = value.children;
    if (rawChildren === undefined) {
      // 叶子节点。正常，不提示。
    } else if (isUnknownArray(rawChildren)) {
      for (let index = 0; index < rawChildren.length; index += 1) {
        const child = parseNode(rawChildren[index], `${path}.children[${index}]`);
        if (child !== null) node.children.push(child);
      }
    } else {
      issues.push({
        severity: 'error',
        code: 'BAD_CHILDREN',
        nodeId: id,
        message: t('core.mm.childrenNotArray', { path, described: describe(rawChildren) }),
      });
    }

    node.extra = Object.fromEntries(extraEntries);
    return node;
  };

  const rawRoot = raw.root;
  if (!isPlainObject(rawRoot)) return fatal('MISSING_ROOT', t('core.mm.notObject', { described: describe(rawRoot) }));
  const root = parseNode(rawRoot, 'root');
  if (root === null) return fatal('MISSING_ROOT', t('core.mm.rootMissing'));

  // --- 顶层 extra -----------------------------------------------------------
  const topEntries: [string, unknown][] = [];
  const unknownTop: string[] = [];
  for (const key of Object.keys(raw)) {
    if (KNOWN_TOP_KEYS.has(key)) continue;
    topEntries.push([key, raw[key]]);
    unknownTop.push(key);
  }
  if (unknownTop.length > 0) {
    issues.push({
      severity: 'warning',
      code: 'UNKNOWN_KEY',
      message: t('core.mm.unknownKeysTop', { n: unknownTop.length, keys: unknownTop.join('、') }),
    });
  }

  const document: MindmapDocument = { root, layout, extra: Object.fromEntries(topEntries), version };

  // --- canonical ------------------------------------------------------------
  // 「源文本是否已经等于我们写出来的样子」。序列化理论上不会抛（树里的值都来自 JSON.parse），
  // 但这里是**解析函数**，绝不能因为一条提示性的判据而把打开文件搞崩 —— 所以兜住。
  // 两条路都显式赋值，不写初值：写了初值反而会被 `no-useless-assignment` 抓出来
  // （那个初值永远读不到）。
  let canonical: boolean;
  try {
    canonical = serializeMindmap(document, { lineEnding }) === text;
  } catch {
    canonical = false;
  }

  return { document, issues, lineEnding, canonical };
}
