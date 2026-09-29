/**
 * doc-block 解析器。
 *
 * 手写行扫描而不是上 YAML/Markdown 库，原因：
 * - 规则足够浅，手写可预测、零依赖、错误信息能精确到行列；
 * - 用通用 YAML 会把「属性区」意外地扩展成一门语言，用户手改时更容易写错；
 * - 解析必须容忍残缺输入（用户正在打字时编辑器要能实时解析）。
 */

import {
  DOC_BLOCK_FENCE_MIN,
  DOC_BLOCK_TERMINATOR,
  ID_PATTERN,
  isDocBlockMode,
  isDocKind,
  KNOWN_ATTR_KEYS,
  type DocBlock,
  type DocBlockAttrs,
  type DocBlockMode,
  type DocKind,
  type InteropIssue,
  type InteropIssueCode,
  type ParseResult,
} from './types.js';

interface Line {
  content: string;
  start: number;
  end: number;
}

const OPEN_RE = /^(`{3,})[ \t]*doc-block[ \t]*$/;
const CLOSE_RE = /^(`{3,})[ \t]*$/;
const HEADER_RE = /^([A-Za-z][A-Za-z0-9_-]*)[ \t]*:[ \t]*(.*)$/;

/** 按任意换行风格切行，同时记录每行的原文偏移。 */
export function splitLines(text: string): Line[] {
  const lines: Line[] = [];
  const re = /\r\n|\n|\r/g;
  let start = 0;
  let match: RegExpExecArray | null;

  while ((match = re.exec(text)) !== null) {
    lines.push({ content: text.slice(start, match.index), start, end: match.index + match[0].length });
    start = match.index + match[0].length;
  }
  if (start < text.length) {
    lines.push({ content: text.slice(start), start, end: text.length });
  }
  return lines;
}

/** 去掉成对的引号，方便用户写 `src: "a b.drawio"`。 */
function unquote(value: string): string {
  const v = value.trim();
  if (v.length >= 2) {
    const first = v[0];
    const last = v[v.length - 1];
    if ((first === '"' && last === '"') || (first === "'" && last === "'")) {
      return v.slice(1, -1);
    }
  }
  return v;
}

interface HeaderOutcome {
  attrs: DocBlockAttrs;
  issues: InteropIssue[];
  /** 属性区最后一行的结束偏移，用于无 `---` 时的兜底 range。 */
  endOffset: number;
}

function parseHeader(lines: Line[]): HeaderOutcome {
  const issues: InteropIssue[] = [];
  const raw: Record<string, string> = {};

  for (const line of lines) {
    if (line.content.trim() === '') continue;

    const m = HEADER_RE.exec(line.content);
    if (!m) {
      issues.push({
        severity: 'warning',
        code: 'HEADER_LINE_IGNORED',
        message: `属性区里这一行不是 key: value 形式，已忽略：${line.content.trim().slice(0, 40)}`,
        offset: line.start,
      });
      continue;
    }

    const key = m[1]!.toLowerCase();
    const value = unquote(m[2]!);

    if (!(KNOWN_ATTR_KEYS as readonly string[]).includes(key)) {
      issues.push({
        severity: 'warning',
        code: 'UNKNOWN_KEY',
        message: `不认识的属性键 "${key}"，已原样保留（可能是更高版本写的，或拼写有误）`,
        offset: line.start,
      });
    }
    // 重复键：后者胜出，并提示。同名键重复通常是手改时的笔误。
    raw[key] = value;
  }

  const id = raw['id'] ?? '';
  const kindRaw = raw['kind'] ?? '';
  const modeRaw = raw['mode'] ?? 'reference';
  const src = raw['src'];
  const title = raw['title'];

  let kind: DocKind = 'text';
  if (kindRaw === '') {
    issues.push({
      severity: 'error',
      code: 'UNKNOWN_KIND',
      message: '缺少 kind，无法判定这是哪类文档；已按纯文本处理以避免破坏内容',
      offset: lines[0]?.start ?? 0,
    });
  } else if (isDocKind(kindRaw)) {
    kind = kindRaw;
  } else {
    issues.push({
      severity: 'error',
      code: 'UNKNOWN_KIND',
      message: `不认识的 kind "${kindRaw}"；已按纯文本处理以避免破坏内容`,
      offset: lines[0]?.start ?? 0,
    });
  }

  let mode: DocBlockMode = 'reference';
  if (isDocBlockMode(modeRaw)) {
    mode = modeRaw;
  } else {
    issues.push({
      severity: 'error',
      code: 'UNKNOWN_MODE',
      message: `不认识的 mode "${modeRaw}"，已按 reference 处理`,
      offset: lines[0]?.start ?? 0,
    });
  }

  const extra: Record<string, string> = {};
  for (const [key, value] of Object.entries(raw)) {
    if (!(KNOWN_ATTR_KEYS as readonly string[]).includes(key)) extra[key] = value;
  }

  const attrs: DocBlockAttrs = { id, kind, mode, extra };
  if (src !== undefined && src !== '') attrs.src = src;
  if (title !== undefined && title !== '') attrs.title = title;

  return { attrs, issues, endOffset: lines[lines.length - 1]?.end ?? 0 };
}

/**
 * 解析整篇文档里所有 doc-block。
 *
 * 不做「全文校验」以外的加工 —— 返回的 `raw` 与 `range` 保证可以无损回填。
 */
export function parseDocBlocks(text: string): ParseResult {
  const lines = splitLines(text);
  const blocks: DocBlock[] = [];
  const issues: InteropIssue[] = [];

  const pushIssue = (code: InteropIssueCode, severity: 'error' | 'warning', message: string, offset: number, blockId?: string) => {
    issues.push(blockId === undefined ? { severity, code, message, offset } : { severity, code, message, offset, blockId });
  };

  let i = 0;
  while (i < lines.length) {
    const openLine = lines[i]!;
    const open = OPEN_RE.exec(openLine.content);
    if (!open) {
      i += 1;
      continue;
    }

    const fenceLength = Math.max(open[1]!.length, DOC_BLOCK_FENCE_MIN);
    const blockStart = openLine.start;

    // ---- 找分隔行 ---
    const headerLines: Line[] = [];
    let terminatorIndex = -1;
    let j = i + 1;
    while (j < lines.length) {
      const cur = lines[j]!;
      if (cur.content.trim() === DOC_BLOCK_TERMINATOR) {
        terminatorIndex = j;
        break;
      }
      const close = CLOSE_RE.exec(cur.content);
      if (close && close[1]!.length >= fenceLength) break;
      headerLines.push(cur);
      j += 1;
    }

    // ---- 找结束围栏 ---
    let closeIndex = -1;
    let k = terminatorIndex >= 0 ? terminatorIndex + 1 : j;
    while (k < lines.length) {
      const cur = lines[k]!;
      const close = CLOSE_RE.exec(cur.content);
      if (close && close[1]!.length >= fenceLength) {
        closeIndex = k;
        break;
      }
      k += 1;
    }

    const header = parseHeader(headerLines);
    const headerEnd = header.endOffset || openLine.end;

    if (terminatorIndex < 0) {
      const at = closeIndex >= 0 ? lines[closeIndex]!.start : text.length;
      pushIssue(
        'MISSING_TERMINATOR',
        'error',
        `缺少属性区结束行 "${DOC_BLOCK_TERMINATOR}"，无法区分属性与内容`,
        at,
      );
    }

    const payloadStart = terminatorIndex >= 0 ? lines[terminatorIndex]!.end : headerEnd;
    let payloadEnd: number;
    if (closeIndex >= 0) {
      payloadEnd = lines[closeIndex]!.start;
    } else {
      payloadEnd = text.length;
      pushIssue(
        'UNTERMINATED_FENCE',
        'error',
        '块没有结束围栏，解析结果可能吞掉了后面的正文',
        blockStart,
      );
    }

    const payload = terminatorIndex >= 0 ? text.slice(payloadStart, Math.max(payloadStart, payloadEnd)) : '';
    // range 右端取「结束围栏这一行的内容末尾」，不吞掉它后面的换行 ——
    // 这样 raw 与 serializeDocBlock 的输出严格相等，块之间的空行归属调用方，
    // spliceBlocks 写回时才能原样还原整篇文本。
    const blockEnd = closeIndex >= 0
      ? lines[closeIndex]!.start + lines[closeIndex]!.content.length
      : text.length;

    blocks.push({
      attrs: header.attrs,
      payload,
      range: { start: blockStart, end: blockEnd },
      raw: text.slice(blockStart, blockEnd),
      fenceLength,
    });

    issues.push(...header.issues.map((issue) => (issue.blockId === undefined ? { ...issue, blockId: header.attrs.id } : issue)));

    i = closeIndex >= 0 ? closeIndex + 1 : lines.length;
  }

  return { blocks, issues };
}

/** 校验：id 合法性、重复、路径写法、模式与内容的匹配。 */
export function validateDocBlocks(blocks: DocBlock[]): InteropIssue[] {
  const issues: InteropIssue[] = [];
  const seen = new Map<string, number>();

  for (const block of blocks) {
    const { attrs, payload, range } = block;
    const at = range.start;
    const id = attrs.id;

    if (id === '') {
      issues.push({ severity: 'error', code: 'MISSING_ID', message: '块缺少 id；id 用于交叉引用与去重，必需', offset: at });
    } else if (!ID_PATTERN.test(id)) {
      issues.push({
        severity: 'error',
        code: 'BAD_ID',
        message: `id "${id}" 不合法：只允许字母、数字、下划线、连字符，且需以字母或数字开头（最长 64）`,
        offset: at,
        blockId: id,
      });
    }

    if (id !== '') {
      const first = seen.get(id);
      if (first === undefined) {
        seen.set(id, at);
      } else {
        issues.push({
          severity: 'error',
          code: 'DUPLICATE_ID',
          message: `id "${id}" 在本文档中重复出现（首次出现于偏移 ${first}）`,
          offset: at,
          blockId: id,
        });
      }
    }

    if (attrs.mode === 'reference') {
      const src = attrs.src;
      if (src === undefined) {
        issues.push({
          severity: 'error',
          code: 'MISSING_SRC',
          message: 'reference 模式必须给 src（真实内容在旁边的独立文件里）',
          offset: at,
          blockId: id,
        });
      } else if (src.trim() === '') {
        issues.push({ severity: 'error', code: 'EMPTY_SRC', message: 'src 为空', offset: at, blockId: id });
      } else {
        if (/^[A-Za-z]:[\\/]/.test(src) || src.startsWith('/') || src.startsWith('\\')) {
          issues.push({
            severity: 'error',
            code: 'ABSOLUTE_SRC',
            message: `src "${src}" 是绝对路径；文档一旦移动或分享给别人就会失效，请改成相对路径`,
            offset: at,
            blockId: id,
          });
        }
        if (src.includes('\\')) {
          issues.push({
            severity: 'warning',
            code: 'BACKSLASH_IN_SRC',
            message: `src "${src}" 含反斜杠；规范要求统一用正斜杠，以便跨平台`,
            offset: at,
            blockId: id,
          });
        }
        if (escapesRoot(src)) {
          issues.push({
            severity: 'warning',
            code: 'PATH_ESCAPES_ROOT',
            message: `src "${src}" 向上跳出了文档所在目录；分享时容易少带文件`,
            offset: at,
            blockId: id,
          });
        }
      }
      if (payload.trim() !== '') {
        issues.push({
          severity: 'warning',
          code: 'PAYLOAD_IN_REFERENCE',
          message: 'reference 模式下分隔行之后还有内容；这部分不会生效，可能让读者误解',
          offset: at,
          blockId: id,
        });
      }
    }

    if (attrs.mode === 'inline') {
      if (payload.trim() === '') {
        issues.push({
          severity: 'warning',
          code: 'EMPTY_PAYLOAD_IN_INLINE',
          message: 'inline 模式但载荷为空；建议改成 reference 模式',
          offset: at,
          blockId: id,
        });
      }
      if (attrs.kind === 'image') {
        issues.push({
          severity: 'warning',
          code: 'IMAGE_MUST_REFERENCE',
          message: '图片请用 reference 模式存路径，不要内联二进制',
          offset: at,
          blockId: id,
        });
      }
    }
  }

  return issues;
}

/** 相对路径是否向上越出根目录。纯字符串推演，不碰文件系统。 */
export function escapesRoot(src: string): boolean {
  const normalized = src.replace(/\\/g, '/');
  let depth = 0;
  for (const segment of normalized.split('/')) {
    if (segment === '' || segment === '.') continue;
    if (segment === '..') {
      depth -= 1;
      if (depth < 0) return true;
    } else {
      depth += 1;
    }
  }
  return false;
}
