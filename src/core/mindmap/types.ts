/**
 * `.mindmap` 文件格式（`docs/format-spec.md` §7.1）—— 类型与常量。
 *
 * ## 为什么这一层在 `core` 而不在 `modules/mindmap`
 *
 * §7.1 的设计意图是「**结构自持，不绑定具体内核**」：`id` / `topic` / `children`
 * 是三家候选内核都有的最小公因式。既然如此，**文件格式就不该知道 mind-elixir 的存在** ——
 * 一旦它 import 了内核，换内核就等于让用户的文件失效，而那正是 §7.1 想避免的事。
 *
 * 所以分两层：
 * - **这一层**（`core/mindmap/`）：`.mindmap` ⇄ 内存里的树。纯函数、无内核、可单测。
 * - **适配层**（`src/modules/mindmap/adapter.ts`，ADR 0007 的后果 2）：
 *   这棵树 ⇄ mind-elixir 的 data。**内核版本变化只影响那一层**。
 *
 * 依赖方向也因此是对的：`modules/` → `core/`（硬约束 1）。
 *
 * ## 向前兼容：`extra` 透传桶
 *
 * `docs/format-spec.md` §5 已经定了规矩：
 * **「更高版本写的键，低版本必须原样保留。解析时落到 `extra`，序列化时照原样写回，
 * 同时给一条 `UNKNOWN_KEY` 提示。」** 这里照办，而且**顶层与每个节点各有一个桶**。
 *
 * 「没有这条，任何新增属性都是一次单向的数据损失」—— 用户在两台机器上装了不同版本的轻笺，
 * 文件来回拷不会掉信息。
 *
 * ## 一个必须记住的后果：`extra` **不能**跟着内核数据走
 *
 * V2 实测（`docs/open-decisions.md` 的「V2 实测记录」）：mind-elixir 的 `getData()`
 * 走带 replacer 的 `JSON.stringify`，**一次往返丢 40 处字段 + 顶层 4 项**。
 * 也就是说：**把 `extra` 塞进内核数据里，它一定会没。**
 *
 * 所以适配层的契约是「**带旧信封进去，拿新信封出来**」：
 * `(上一份 MindmapDocument, 新的内核 data) → 新的 MindmapDocument`，
 * 那些内核不认识的字段由上一份信封提供。**不是** `(内核 data) → MindmapDocument`。
 */

/** 文件标识。顶层 `format` 不是这个值的 JSON，就不是我们的导图。 */
export const MINDMAP_FORMAT = 'levnote.mindmap';

/**
 * 当前文件格式版本。
 *
 * > **一处规范内的不一致，在这里定死**：`docs/format-spec.md` §7.1 的标题写「(v0)」，
 * > 而它给出的示例写的是 `"version": 1`；§9 又讲「v0 → v1 的兼容承诺」。
 * > 三处说法互斥，实现必须挑一个。**挑示例里的 `1`** ——
 * > 示例是会被直接抄走的东西，而且「标题的 v0」可以读作「这份**规范草案**的第 0 版」，
 * > 与文件里的 `version` 字段本来就不是一回事。
 * > 规范那两处已按这个口径改写（见 §7.1 与 §9）。
 */
export const MINDMAP_VERSION = 1;

/**
 * 默认布局。§7.1 的示例写的就是它。
 *
 * 对应内核的**双向**逻辑结构图（`direction = 2`），也就是「思维导图」最典型的样子。
 * 刻意**不用** mind-elixir 自己的默认值（`direction = 1`，全部向右）：
 * 打开一张 `.mindmap` 时该看到的是中心发散，不是一张左对齐的树。
 */
export const MINDMAP_DEFAULT_LAYOUT = 'logicalStructure';

/**
 * 认得的布局。
 *
 * ## 为什么是四个值而不是一个
 *
 * 布局（= 内核的 `direction`）是**用户可以改的东西** ——
 * `Ctrl + 方向键` 就能换，那是内核自带的键位（见 `MindElixir.js` 的 keypress 表）。
 * 只认一个值意味着「用户按了 `Ctrl+←`、保存、再打开，方向回去了」——
 * 一次静默的数据丢失（硬约束 2）。
 *
 * 名字沿用 XMind 的术语，因为它们是**跨工具的通用叫法**，
 * 而不是 mind-elixir 的内部标识符 —— 将来换内核时这几个名字仍然讲得通。
 *
 * **顺序必须与内核的方向编号一致**（0 左 / 1 右 / 2 双向 / 3 向下）：
 * 适配层用下标直接查表（`modules/mindmap/adapter.ts` 的 `LAYOUT_BY_DIRECTION`），
 * 顺序错了会让「向右」显示成「向左」，而且**没有任何地方会报错**。
 * `adapter.test.ts` 有一条测试专门钉住这个顺序。
 */
export const MINDMAP_LAYOUTS: readonly string[] = [
  'logicalStructureLeft', // 0：全部向左
  'logicalStructureRight', // 1：全部向右
  'logicalStructure', // 2：双向（默认）
  'orgChart', // 3：向下（组织架构图）
];

/** 顶层「我们认识」的字段。不在此列的键进 `extra`。 */
export const KNOWN_TOP_KEYS: ReadonlySet<string> = new Set(['format', 'version', 'layout', 'root']);

/** 节点上「我们认识」的字段。不在此列的键进该节点的 `extra`。 */
export const KNOWN_NODE_KEYS: ReadonlySet<string> = new Set([
  'id',
  'topic',
  'note',
  'collapsed',
  'style',
  'icons',
  'image',
  'children',
]);

/** 图片在画布上的填充方式。与 mind-elixir 的 `objectFit` 取值一致，省略即不写。 */
export type MindmapImageFit = 'fill' | 'contain' | 'cover';

/**
 * 节点插图（ADR 0054）。字段与 mind-elixir 的 `NodeObj.image` 一致 ——
 * 不是抄内核，而是「一张图、一个显示框」本就是导图界的公共形状
 * （XMind / mind-elixir / MindMaster 皆然）。
 */
export interface MindmapImage {
  /** 相对本文件的路径，或 `data:` 内嵌图。**绝不存绝对路径**（见 §5）。 */
  readonly url: string;
  /** 显示宽度（px）。 */
  readonly width: number;
  /** 显示高度（px）。 */
  readonly height: number;
  /** 填充方式。可省。 */
  readonly fit?: MindmapImageFit;
}

/** 导图节点。字段与 §7.1 一致，外加一个透传桶。 */
export interface MindmapNode {
  id: string;
  topic: string;
  children: MindmapNode[];
  note?: string;
  collapsed?: boolean;
  /** 样式。**内容是不透明的** —— 我们不解释它，只负责原样保留。 */
  style?: Record<string, unknown>;
  /**
   * 标记 / 优先级（ADR 0053）：**显示字形本身**（emoji 一类字符），按先加在后。
   *
   * 与 `topic` 存显示文本同一条理由：字形就是内容，任何能排文本的内核
   * 都能显示它，于是这份文件不绑定「哪一个导图内核、哪一张标记词表」。
   * 我们的调色盘词表（`modules/mindmap/markers.ts`）只是**界面知识**，
   * 不是格式层语义：文件里出现词表之外的字符，照旧原样保留（§5），
   * 内核也照样把它排在主题旁边。
   *
   * 空数组是合法值（「全清过」也是一句话说得清的状态，序列化时原样写回）。
   */
  icons?: string[];
  /**
   * 节点插图（ADR 0054）。`url` 指向一张相对本文件的图（或 `data:` 内嵌图）；
   * `width` / `height` 是**显示尺寸**（px），不是原图尺寸 —— 粘贴时解码一次
   * 定下来，之后文件里存的就是「画出来多大」；`fit` 可省。
   *
   * 与 `topic` 存显示文本、`icons` 存显示字形同一条理由：**尺寸即内容** ——
   * 任何能排 `<img>` 的内核都认识「一张图、一个显示框」，于是这份文件
   * 不绑定「哪一个导图内核」。原图本身由外壳按 `ImagePasteRequest` /
   * `resolveImageUrl` 契约读写（`core/editor/types.ts`），格式层只见引用。
   *
   * `url` / `width` / `height` 三者**齐才有效**：内核缺一不画（据此设计占位图），
   * 解析时三者缺一或类型不对，**整值**转 `extra`（一个不丢）。
   */
  image?: MindmapImage;
  /**
   * 这个节点上**没认出来**的字段，原样保留（§5）。
   *
   * 刻意**必填**而不是可选：一个「可能是空对象」的可选字段，在适配层里
   * 最容易被写成 `extra ?? {}` 而**丢掉原有的键**（那正是静默丢数据）。
   * 必填之后，任何一条构造路径都必须显式交代它从哪来。
   *
   * **插入顺序有意义**：解析时按源文件的键顺序写入，序列化时按同样的顺序写出，
   * 所以「未识别的字段」在往返中不会被打乱相对次序。
   */
  extra: Record<string, unknown>;
}

/** 一份解析出来的 `.mindmap`。 */
export interface MindmapDocument {
  root: MindmapNode;
  /** 布局。认不出也照样保留（见 `MINDMAP_DEFAULT_LAYOUT`）。 */
  layout: string;
  /** 顶层没认出来的字段，原样保留。 */
  extra: Record<string, unknown>;
  /**
   * 源文件里读到的版本号。**写回时按它写，不擅自升级** ——
   * 把一个 v1 文件升级成 v2 会让老版本的轻笺读不了，而用户没要求过这件事。
   */
  version: number;
}

export type MindmapIssueSeverity = 'error' | 'warning';

export interface MindmapIssue {
  severity: MindmapIssueSeverity;
  code: MindmapIssueCode;
  message: string;
  /** 出问题的节点 id（有的话），UI 可以据此定位。 */
  nodeId?: string;
}

/**
 * 问题码。严重级别的含义沿用 §6 的定义：
 * **error = 照这样写下去，内容会丢或会失效；warning = 能工作，但你大概不是这个意思。**
 */
export type MindmapIssueCode =
  /** 不是合法 JSON。**致命** —— 只能落到纯文本兜底。 */
  | 'NOT_JSON'
  /** 顶层不是对象（是数组 / 字符串 / null…）。**致命**。 */
  | 'NOT_OBJECT'
  /** `format` 不是 `levnote.mindmap`。**致命** —— 这不是我们的文件。 */
  | 'WRONG_FORMAT'
  /** 缺 `root`，或它不是对象。**致命** —— 没有树可画。 */
  | 'MISSING_ROOT'
  /** `children` 不是数组。**error**：那个值只能被丢掉。 */
  | 'BAD_CHILDREN'
  /** 节点里的某个值不是对象，只能跳过。**error**。 */
  | 'BAD_NODE'
  /** `version` 比我们新。**warning** —— 照读，但可能有我们认不出的字段。 */
  | 'FUTURE_VERSION'
  /** `version` 不是数字。**warning** —— 按当前版本处理。 */
  | 'BAD_VERSION'
  /** 缺 `topic` 或它不是字符串。**warning** —— 补空串。 */
  | 'MISSING_TOPIC'
  /** 缺 `id` 或它不合法。**warning** —— 补一个**确定性**的自动 id。 */
  | 'MISSING_NODE_ID'
  /** 同一份文件里 id 重复。**warning** —— 树本身没问题，但按 id 索引的旁表会串。 */
  | 'DUPLICATE_NODE_ID'
  /** `layout` 不是我们认识的那个。**warning** —— 原样保留。 */
  | 'UNKNOWN_LAYOUT'
  /** 认出来的字段类型不对。**warning** —— 值转进 `extra`，一个都不丢。 */
  | 'BAD_FIELD'
  /** 见 §5。**warning** —— 值原样保留。 */
  | 'UNKNOWN_KEY';

/** 行尾风格。行尾是**文件属性**（§4），由知道目标文件的那一层传进来，序列化器不自己猜。 */
export type LineEnding = '\n' | '\r\n';

export interface ParseMindmapOptions {
  /** 自动补 id 的前缀。测试里固定它就能拿到确定的输出。 */
  autoIdPrefix?: string;
}

export interface ParseMindmapResult {
  /** 解析出来的文档。`null` = 致命问题，调用方应降级到纯文本兜底。 */
  document: MindmapDocument | null;
  issues: MindmapIssue[];
  /** 源文本的行尾风格，保存时原样传回序列化器。 */
  lineEnding: LineEnding;
  /**
   * 源文本是否**已经等于我们写出来的样子**。
   *
   * `false` 的含义是「这份文件是手写的 / 别的工具写的 / 更高版本写的」——
   * **不是**「它坏了」。它的用途只有一条：
   * 文档没有被编辑过就保存时，调用方应当**原样写回源文本**，而不是写序列化结果。
   * 这样「打开 → 保存」不产生 diff（§7.2 第 4 条对 `.drawio` 的要求，
   * 对 `.mindmap` 同样成立）。
   */
  canonical: boolean;
}

export interface SerializeMindmapOptions {
  /** 缩进空格数。默认 2，与 §7.1 的示例一致。 */
  indent?: number;
  /** 行尾。默认 `\n`。 */
  lineEnding?: LineEnding;
}
