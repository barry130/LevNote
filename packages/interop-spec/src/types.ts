/**
 * 块级互操作规范 v0 —— 类型定义。
 *
 * 设计目标（按优先级）：
 * 1. **不丢**：解析 → 序列化必须能原样还原，且未改动的块序列化后与原文逐字节相同。
 * 2. **降级不崩**：不认识的 Markdown 渲染器把它当普通围栏代码块显示，内容可见而非消失。
 * 3. **可手改**：纯文本编辑器里能直接改，语法浅到不需要文档。
 * 4. **向前兼容**：未来版本新增的键，老版本原样保留在 `extra` 里，不丢。
 */

/** 四类文档 + 图片 + 纯文本兜底。四类一等公民：markdown / code / mindmap / diagram。 */
export const DOC_KINDS = ['markdown', 'code', 'mindmap', 'diagram', 'image', 'text'] as const;
export type DocKind = (typeof DOC_KINDS)[number];

/**
 * - `reference`：只存路径，真实内容在旁边的独立文件里。默认模式。
 *   这是「四类文档各自独立新建 / 修改 / 保存」的基础形态。
 * - `inline`：内容直接内联在块里。用于「这个片段太小，不值得单开一个文件」的例外情况。
 *   内联存的是**明文**（如 .drawio 的 XML），不是压缩数据 —— 保证纯文本也能看懂。
 */
export const DOC_BLOCK_MODES = ['reference', 'inline'] as const;
export type DocBlockMode = (typeof DOC_BLOCK_MODES)[number];

/** 围栏的信息串。认得它的渲染器走我们的逻辑，不认得的按代码块渲染。 */
export const DOC_BLOCK_INFO = 'doc-block';

/** 围栏最少反引号数，遵循 CommonMark。 */
export const DOC_BLOCK_FENCE_MIN = 3;

/** 属性区与载荷区的分隔行。 */
export const DOC_BLOCK_TERMINATOR = '---';

/** 已知属性键。不在此列的键会被保留进 `extra`。 */
export const KNOWN_ATTR_KEYS = ['id', 'kind', 'mode', 'src', 'title'] as const;

export interface DocBlockAttrs {
  /** 稳定标识，文档内唯一。用于交叉引用与「同一个图被多处引用」的去重。 */
  id: string;
  kind: DocKind;
  mode: DocBlockMode;
  /** 相对当前文档所在目录的路径，统一用正斜杠。reference 模式必填。 */
  src?: string;
  title?: string;
  /** 未识别的键原样保留，保证向前兼容不丢信息。 */
  extra: Record<string, string>;
}

export interface SourceRange {
  start: number;
  end: number;
}

export interface DocBlock {
  attrs: DocBlockAttrs;
  /** inline 模式的载荷；reference 模式为空串。 */
  payload: string;
  /** 在原文中的字符区间，便于编辑器做精准替换而不是全文重排。 */
  range: SourceRange;
  /** 原始文本片段。未改动时直接回填，避免「打开即重排」污染 git diff。 */
  raw: string;
  /** 原始围栏反引号个数，序列化时尽量沿用。 */
  fenceLength: number;
}

export type IssueSeverity = 'error' | 'warning';

export interface InteropIssue {
  severity: IssueSeverity;
  code: InteropIssueCode;
  message: string;
  /** 原文偏移，UI 可直接定位。 */
  offset: number;
  blockId?: string;
}

export type InteropIssueCode =
  | 'MISSING_ID'
  | 'BAD_ID'
  | 'DUPLICATE_ID'
  | 'UNKNOWN_KIND'
  | 'UNKNOWN_MODE'
  | 'UNKNOWN_KEY'
  | 'MISSING_SRC'
  | 'EMPTY_SRC'
  | 'ABSOLUTE_SRC'
  | 'BACKSLASH_IN_SRC'
  | 'PATH_ESCAPES_ROOT'
  | 'MISSING_TERMINATOR'
  | 'UNTERMINATED_FENCE'
  | 'HEADER_LINE_IGNORED'
  | 'PAYLOAD_IN_REFERENCE'
  | 'EMPTY_PAYLOAD_IN_INLINE'
  | 'IMAGE_MUST_REFERENCE';

export interface ParseResult {
  blocks: DocBlock[];
  issues: InteropIssue[];
}

/** 建块时的入参，除 kind 外全部可选。 */
export interface CreateDocBlockInput {
  kind: DocKind;
  id?: string;
  mode?: DocBlockMode;
  src?: string;
  title?: string;
  payload?: string;
  extra?: Record<string, string>;
}

export function isDocKind(value: string): value is DocKind {
  return (DOC_KINDS as readonly string[]).includes(value);
}

export function isDocBlockMode(value: string): value is DocBlockMode {
  return (DOC_BLOCK_MODES as readonly string[]).includes(value);
}

/** id 允许的字符集：够宽，同时避免空格/引号在路径与锚点里引发歧义。 */
export const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/;
