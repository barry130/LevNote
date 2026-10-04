/**
 * `doc-block` 的解析（格式规范 `docs/format-spec.md` §2–§3 的实现）。
 *
 * `doc-block` 是四类文档互相引用的载体（ADR 0002）：一个信息串为
 * `doc-block` 的**普通围栏代码块**，属性区写 `key: value`，`---` 结束，
 * 之后是载荷。不认得它的渲染器把它当普通代码块 —— 内容可见、语义可推断，
 * 这正是选围栏而不是 front-matter / 行内标记的理由。
 *
 * ## 这一层只解析「属性区 + 载荷」，不管围栏
 *
 * 调用方（Markdown 内核的 NodeView）拿到的是**已经**被 Markdown 解析器
 * 当成代码块切开的内容 —— 围栏的配对、加长、嵌套是 CommonMark 的事，
 * 这里只剩两件事：把 `key: value` 行读成结构，找到 `---` 结束标记并把
 * 载荷原样切出来。所以规范 §3 的 `UNTERMINATED_FENCE`（围栏没闭合）在
 * 这一层**不可能发生**；能发生的只有 `MISSING_TERMINATOR`。
 *
 * ## 向前兼容落在 `extra`
 *
 * 更高版本写的键，低版本必须原样保留（规范 §5）——不认识的键进
 * `attrs.extra`，同时报一条 `UNKNOWN_KEY`（可能是拼写错误，也可能是新特性）。
 * **绝不丢弃**：丢一条键就是一次单向的数据损失。
 */

/** 属性区认识的键。其余键进 `extra`（规范 §2.3「其它」）。 */
const KNOWN_KEYS = ['id', 'kind', 'mode', 'src', 'title'] as const;

/** 规范 §2.3 的 `kind` 取值。`text` 是未知 kind 的兜底。 */
export type DocBlockKind = 'markdown' | 'code' | 'mindmap' | 'diagram' | 'image' | 'text';

export interface DocBlockAttrs {
  readonly id: string | null;
  readonly kind: string | null;
  readonly mode: string | null;
  readonly src: string | null;
  readonly title: string | null;
  /** 不认识的键，**原样保留**（规范 §5 向前兼容）。 */
  readonly extra: Readonly<Record<string, string>>;
}

/** 解析层面能发现的错误。渲染方据此决定「照常渲染」还是「说明原因」。 */
export type DocBlockParseError = 'MISSING_TERMINATOR' | 'UNKNOWN_KEY';

export interface ParsedDocBlock {
  readonly attrs: DocBlockAttrs;
  /** `---` 之后的载荷，**逐字节原样**（不做任何规范化）。没有结束标记时是 `null`。 */
  readonly payload: string | null;
  readonly errors: readonly DocBlockParseError[];
}

/** 规范 §3.2：属性区的结束标记是「一整行 trim 后等于 `---`」。 */
function isTerminator(line: string): boolean {
  return line.trim() === '---';
}

/**
 * 属性行是 `key: value`。**键后必须紧跟冒号**；没有冒号的行不是属性行 ——
 * 它出现在结束标记之前时按「属性区不认识的东西」处理：跳过，不进 extra
 * 也不报错。理由：属性区混进一句注释不该让整个块渲染失败，
 * 而把它塞进任何一个值的语义都是错的。
 */
function parseAttrLine(line: string): { key: string; value: string } | null {
  const match = /^([a-z0-9-]+):\s*(.*)$/.exec(line);
  if (match === null) return null;
  return { key: match[1]!, value: match[2] ?? '' };
}

/**
 * 值可能被双引号包着（序列化器对含首尾空白或引号的值加引号，规范 §4）。
 * 解析端把**成对**的引号去掉；只有一侧有引号的不算引用，照原样保留 ——
 * 那多半是值本身含有引号字符。
 */
function unquote(value: string): string {
  if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) {
    return value.slice(1, -1);
  }
  return value;
}

/**
 * 解析一个 doc-block 的**内容**（围栏之内、外壳之外的原文）。
 *
 * 永远不抛：内容可能来自任何用户写的文件，解析失败要通过
 * `errors` 说出来，而不是让渲染链路上出现一个没人接的异常。
 */
export function parseDocBlock(text: string): ParsedDocBlock {
  const lines = text.split('\n');

  let attrs: DocBlockAttrs = { id: null, kind: null, mode: null, src: null, title: null, extra: {} };
  const errors: DocBlockParseError[] = [];
  const unknownKeys = new Set<string>();

  let terminatorIndex = -1;
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]!;
    if (line.trim() === '') continue;
    if (isTerminator(line)) {
      terminatorIndex = index;
      break;
    }

    const attr = parseAttrLine(line);
    if (attr === null) continue;

    if ((KNOWN_KEYS as readonly string[]).includes(attr.key)) {
      attrs = { ...attrs, [attr.key]: unquote(attr.value) };
    } else {
      unknownKeys.add(attr.key);
      attrs = { ...attrs, extra: { ...attrs.extra, [attr.key]: unquote(attr.value) } };
    }
  }

  if (unknownKeys.size > 0) errors.push('UNKNOWN_KEY');

  // 没找到结束标记：属性区到行尾全是属性，载荷不存在。
  // 这是渲染时必须说出来的事 —— 用户多半是漏写了那一行 `---`
  if (terminatorIndex === -1) {
    errors.push('MISSING_TERMINATOR');
    return { attrs, payload: null, errors };
  }

  const payload = lines.slice(terminatorIndex + 1).join('\n');
  return { attrs, payload, errors };
}

/**
 * 这个 doc-block 是不是一块**可渲染的图表引用**（P5 第 25 项的形态）：
 *
 * - `kind: diagram` —— 渲染器只认这一类；
 * - `mode: reference`（或缺省 —— 规范 §2.3 说默认语义是 reference）且 `src` 非空。
 *
 * `mode: inline`（载荷就是文档内容）不在这一步的射程里，返回 `null`
 * 让它按普通代码块显示 —— 那是 ADR 0002 定下的降级形态，不是丢失。
 */
export function referencedDiagramSrc(parsed: ParsedDocBlock): string | null {
  if (parsed.attrs.kind !== 'diagram') return null;
  const mode = parsed.attrs.mode ?? 'reference';
  if (mode !== 'reference') return null;
  const src = parsed.attrs.src;
  if (src === null || src.trim() === '') return null;
  return src.trim();
}
