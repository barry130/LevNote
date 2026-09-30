#!/usr/bin/env node
/**
 * 生成 V3（长文档输入延迟）的压测材料。
 *
 * 用法：
 *   node scripts/make-v3-longdoc.mjs
 *
 * ## 为什么要有这个脚本
 *
 * V3 的判据是「1 万字以上文档，输入延迟 ≤ 50ms」，而它**只能人手敲键盘测**
 * （jsdom 没有布局引擎也没有真实计时；ProseMirror 又不监听 keydown 插字，
 * 所以合成事件也造不出打字）。既然测量必须人工，至少把「准备一份够长、
 * 结构够杂的文档」这一步自动化掉 —— 否则每次重测都要重新凑一份。
 *
 * ## 为什么是两份而不是一份
 *
 * 单跑一份只能回答「过不过」。过不了的时候还需要回答「**是谁慢**」——
 * 否则拿到「超出 50ms」也不知道该改哪儿。两份文档构成一次干净的 A/B：
 *
 *   v3-longdoc.md        完整版（8 个代码块）
 *   v3-longdoc-nocode.md 对照组（代码块换成等量中文段落）
 *
 * 两份的**节点数与总字数接近、只差「有没有 code_block」**。于是两份的
 * 延迟之差就是 `codeHighlightPlugin` 里那次 `buildDecorations(tr.doc)`
 * （每次改文档整篇重跑 highlight.js）的净成本。详见 `plugin.ts`。
 *
 * 注意 A/B 的方向别做反：**把代码块的语言名删掉不是「去掉高亮」** ——
 * `highlightCode()` 对无语言输入会走 `hljs.highlightAuto()`，
 * 那是**更贵**的一条路（要跑多套语法并做 relevance 评分）。
 * 要归因就得让 `code_block` 这个节点本身消失。
 *
 * ## 两个刻意的选择
 *
 * 1. **文档不以 `---` 开头**。带 front matter 的 Markdown 文件按 ADR 0009
 *    会降级到代码内核，那样测到的就不是 Milkdown 了。
 * 2. **不写进版本库**（`.gitignore` 里已排除）。它是生成物，进 git 只会
 *    带来无意义的 diff 噪音；脚本本身才是资产。
 */

import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const outDir = join(here, '..', 'docs', 'samples');

/** 目标字数（Word 口径，与 `src/core/statistics.ts` 对齐）。 */
const TARGET_WORDS = 15_000;

// ---------------------------------------------------------------------------
// 内容池
// ---------------------------------------------------------------------------

/** 正文段落。刻意写成真实的技术笔记口吻 —— 压测时人要能读下去才知道自己在哪。 */
const PARAGRAPHS = [
  '编辑器的内核选型从来不是一个纯技术问题。它同时受许可、可维护性与「将来换掉它要付多少代价」三件事约束。一个成熟的内核能省下几个月的实现时间，但如果它的数据结构与我们的文档模型隔得太远，那么每一次打开与保存都要经过一次有损翻译，而这种损耗会在用户的文件里累积，最终表现为「用别的工具打开就不对了」。',
  '事务模型决定了插件能看见什么。在 ProseMirror 里，一次输入会变成一条或多条事务，每条事务带着一份「文档变了什么」的映射。插件可以在事务派发过程中拦截、改写或追加新的事务。这个能力很强，但也很容易用错：一旦在事务里做了与文档无关的副作用，撤销栈就会跟着乱掉，用户按一次撤销会回退两步。',
  '依赖方向是这套代码里最不可动摇的一条约束。内核不认识外壳，外壳也不许按类名去猜内核画了什么。所有跨层的沟通都必须经过一份显式的契约，契约里的每个方法都要能回答「如果我不加它，外壳是不是就得去读 DOM」。这条约束在短期内看起来像是自找麻烦，但它换来的是「换一个内核」时改动量可控。',
  '序列化与原文保真是同一件事的两面。富文本编辑器内部持有的是一棵树，而磁盘上放的是文本。把树写回文本时，任何「顺手规范化」都会改变用户没碰过的字节。这些改写大多是语义等价的，比如把下划线形式的标题改成井号形式，把行尾多余的空格收敛掉。问题不在等价，而在于用户看不见它发生。',
  '性能预算必须在写代码之前定下来，否则它永远会被排到最后。对编辑器来说，最敏感的那个数字是按键到出字的延迟。人眼对这件事的分辨能力比想象中强：超过大约五十毫秒，打字就会开始有「粘滞感」，而这种感觉一旦出现，用户会归因为「这个软件很重」，无论其他部分做得多好。',
  '测试策略要能回答「这条断言失败时，我该去哪一行找问题」。按模块数量铺测试是很容易的，但覆盖率数字本身不说明任何事。真正有用的是两类用例：一类钉住那些「写错了也不会报错、只会安静地做错事」的地方，另一类是跨层的端到端用例，因为接缝处的错误在单层测试里永远看不见。',
  '错误处理的第一原则是不要让它自己崩。外部输入的形状是不可信的：抛出来的可能是一个字符串、一个空值，或者一个没有 message 字段的对象。把这些都收敛成一个内部的错误类型，是让上层能写出一致提示的前提。反过来，只要有一处漏了收敛，那一处就会变成一个「点一下整个界面白掉」的入口。',
  '位置记忆看起来是个小功能，实际上是一堆取舍。光标的位置在内核的坐标系里是一个不透明的偏移量，而滚动位置是一个比例。这两个数分开存、分开恢复。当文档被外部改动过，光标偏移就可能指向完全不同的地方，这时候只恢复滚动、不对齐光标，比跳到一个错误的位置要好得多。',
  '样式表的所有权要按属性划分，而不是按组件划分。如果内核负责一段内容的排版，外壳又给它加了一条溢出裁剪规则，那么当内核某天忘了声明滚动时，表现不是「滚不动」，而是内容被安静地裁掉一截 —— 用户看不到后面还有内容。这类错误没有报错、没有日志，只能靠约定避免。',
  '构建产物的体积在桌面端不像网页端那样致命，但也不是可以无限放任的。每一次往主包里加东西，都要能回答它是不是首屏就要画出来。如果是，那就没有懒加载的余地，只能接受；如果不是，就应该走动态导入。这个判断没有中间地带，含糊过去的结果就是主包一年后翻一倍。',
  '文档往返是检验编辑器诚实的唯一标准。打开一份文件、一个字都不改、直接保存，磁盘上的字节应该完全一致。做不到这一点时，用户会开始怀疑这个工具，因为没有人愿意在一个会偷偷改自己文件的软件里写长文档。这类问题在单元测试里很难被发现，因为测试里的文件读取是假的，返回什么都是测试说了算。',
  '人工验收的边界要写清楚，不能含糊成「以后有空再看」。有一类结论只有真机才能得出：排版在真实渲染引擎里的观感、浮层贴在屏幕哪个位置、以及输入延迟到底是多少毫秒。这些东西在无布局引擎的测试环境里全都不存在，硬测只会得到一串没有意义的零。',
  '增量计算是长文档编辑器的核心难题。当用户敲下一个字符，理论上只有那一小段受影响的内容需要重新计算，但实际上很多实现会退回到「整篇重来」。整篇重来的代价与文档长度成正比，于是文档越写越长、编辑器越写越卡，而这个问题在文档还短的时候完全看不出来。',
  '缓存是另一种形式的取舍。把一份解析结果留在内存里能让重复访问变快，但缓存本身要占内存，而且必须有失效策略。没有上界的缓存迟早会吃掉几十兆内存，而用户只会看到「开了几个标签页之后变卡了」。按条目数设上界往往不够，因为条目的大小可以差几百倍。',
  '用户看不见的东西最容易被牺牲。一个功能是否正确，测试能覆盖；一个功能是否顺手，只有用的人知道。这两件事的优先级经常被搞反：团队会花很多时间把正确的功能做得更正确，却很少花时间去看它用起来是什么感觉。而后者才是用户真正会记住的部分。',
  '架构演进的正确方式是留下可以推翻的记录。每个决策都应该写清楚当时的约束、考虑了哪些备选、以及什么条件下应该重新审视它。没有这份记录，两年后的人只会看到一段「看起来很怪」的代码，然后把它改回更常见但更差的写法，因为他不知道当初为什么这么写。',
];

/** 代码块。语言覆盖 highlight.js 的常见集合，长度 18–30 行。 */
const CODE_BLOCKS = [
  {
    language: 'ts',
    lines: [
      'interface EditorPosition {',
      '  readonly offset: number;',
      '  readonly textLength: number;',
      '  readonly scrollRatio: number;',
      '}',
      '',
      'function restorePosition(handle: EditorHandle, saved: EditorPosition): boolean {',
      '  const current = handle.getPosition();',
      '  if (current.textLength !== saved.textLength) {',
      '    handle.setScrollRatio(saved.scrollRatio);',
      '    return false;',
      '  }',
      '  handle.setPosition(saved.offset);',
      '  handle.setScrollRatio(saved.scrollRatio);',
      '  return true;',
      '}',
      '',
      'export function remember(store: PositionStore, path: string, position: EditorPosition): void {',
      '  store.write(path, {',
      '    offset: position.offset,',
      '    textLength: position.textLength,',
      '    scrollRatio: clampRatio(position.scrollRatio),',
      '    savedAt: Date.now(),',
      '  });',
      '}',
    ],
  },
  {
    language: 'rust',
    lines: [
      'use std::io::Write;',
      'use std::path::{Path, PathBuf};',
      '',
      '#[derive(Debug, Clone)]',
      'pub struct WriteOutcome {',
      '    pub path: PathBuf,',
      '    pub bytes: usize,',
      '    pub mtime_ms: u64,',
      '}',
      '',
      'pub fn write_atomic(target: &Path, bytes: &[u8]) -> std::io::Result<WriteOutcome> {',
      '    let parent = target.parent().unwrap_or_else(|| Path::new("."));',
      '    let mut tmp = tempfile::NamedTempFile::new_in(parent)?;',
      '    tmp.write_all(bytes)?;',
      '    tmp.flush()?;',
      '    tmp.as_file().sync_all()?;',
      '',
      '    tmp.persist(target).map_err(|err| err.error)?;',
      '',
      '    let meta = std::fs::metadata(target)?;',
      '    Ok(WriteOutcome {',
      '        path: target.to_path_buf(),',
      '        bytes: bytes.len(),',
      '        mtime_ms: mtime_of(&meta),',
      '    })',
      '}',
    ],
  },
  {
    language: 'python',
    lines: [
      'from dataclasses import dataclass',
      'from typing import Iterable',
      '',
      '',
      '@dataclass(frozen=True)',
      'class Sample:',
      '    path: str',
      '    elapsed_ms: float',
      '    view: str',
      '',
      '',
      'def percentile(values: Iterable[float], q: float) -> float:',
      '    ordered = sorted(values)',
      '    if not ordered:',
      '        return 0.0',
      '    index = min(len(ordered) - 1, int(len(ordered) * q))',
      '    return ordered[index]',
      '',
      '',
      'def summarize(samples: list[Sample]) -> dict[str, float]:',
      '    values = [s.elapsed_ms for s in samples]',
      '    return {',
      '        "n": float(len(values)),',
      '        "p50": percentile(values, 0.50),',
      '        "p95": percentile(values, 0.95),',
      '        "max": max(values) if values else 0.0,',
      '    }',
    ],
  },
  {
    language: 'sql',
    lines: [
      'CREATE TABLE IF NOT EXISTS document (',
      '  id            INTEGER PRIMARY KEY AUTOINCREMENT,',
      '  path          TEXT    NOT NULL,',
      '  text_length   INTEGER NOT NULL DEFAULT 0,',
      '  scroll_ratio  REAL    NOT NULL DEFAULT 0.0,',
      '  saved_at      INTEGER NOT NULL,',
      '  UNIQUE (path)',
      ');',
      '',
      'CREATE INDEX IF NOT EXISTS idx_document_saved_at',
      '  ON document (saved_at DESC);',
      '',
      'SELECT',
      '  path,',
      '  text_length,',
      '  saved_at',
      'FROM document',
      'WHERE text_length >= 10000',
      '  AND saved_at > :since',
      'ORDER BY saved_at DESC',
      'LIMIT 50;',
    ],
  },
  {
    language: 'bash',
    lines: [
      '#!/usr/bin/env bash',
      'set -euo pipefail',
      '',
      'PORT=1420',
      'BASE="http://127.0.0.1:${PORT}"',
      '',
      'probe() {',
      '  local path="$1"',
      '  local code',
      '  code=$(curl -s -o /dev/null -w "%{http_code}" --noproxy "*" "${BASE}${path}")',
      '  printf "%-32s %s\\n" "${path}" "${code}"',
      '}',
      '',
      'probe /',
      'probe /src/main.tsx',
      'probe /src/ui/App.tsx',
      'probe /src/ui/styles.css',
      '',
      'if [ "$(curl -s -o /dev/null -w "%{http_code}" --noproxy "*" "${BASE}/")" != "200" ]; then',
      '  echo "dev server 不可达" >&2',
      '  exit 1',
      'fi',
      '',
      'echo "冒烟通过"',
    ],
  },
  {
    language: 'json',
    lines: [
      '{',
      '  "compilerOptions": {',
      '    "target": "ES2020",',
      '    "module": "ESNext",',
      '    "moduleResolution": "bundler",',
      '    "strict": true,',
      '    "noUncheckedIndexedAccess": true,',
      '    "exactOptionalPropertyTypes": true,',
      '    "noImplicitOverride": true,',
      '    "verbatimModuleSyntax": true,',
      '    "isolatedModules": true,',
      '    "skipLibCheck": true,',
      '    "noEmit": true',
      '  },',
      '  "include": ["src", "packages", "vite.config.ts"],',
      '  "exclude": ["node_modules", "dist", "src-tauri"]',
      '}',
    ],
  },
  {
    language: 'go',
    lines: [
      'package main',
      '',
      'import (',
      '\t"encoding/json"',
      '\t"log"',
      '\t"net/http"',
      '\t"time"',
      ')',
      '',
      'type report struct {',
      '\tPath      string  `json:"path"`',
      '\tElapsedMs float64 `json:"elapsedMs"`',
      '\tView      string  `json:"view"`',
      '}',
      '',
      'func handleReport(w http.ResponseWriter, r *http.Request) {',
      '\tif r.Method != http.MethodPost {',
      '\t\thttp.Error(w, "method not allowed", http.StatusMethodNotAllowed)',
      '\t\treturn',
      '\t}',
      '\tvar item report',
      '\tif err := json.NewDecoder(r.Body).Decode(&item); err != nil {',
      '\t\thttp.Error(w, err.Error(), http.StatusBadRequest)',
      '\t\treturn',
      '\t}',
      '\tlog.Printf("%s %s %.2fms", item.View, item.Path, item.ElapsedMs)',
      '\tw.WriteHeader(http.StatusNoContent)',
      '}',
    ],
  },
  {
    language: 'css',
    lines: [
      '.editor-split {',
      '  display: flex;',
      '  flex: 1;',
      '  min-height: 0;',
      '}',
      '',
      '.editor-pane {',
      '  display: flex;',
      '  flex-direction: column;',
      '  flex-basis: 0;',
      '  min-width: 0;',
      '  min-height: 0;',
      '  /* 刻意不声明 overflow —— 滚动归内核，见硬约束 13 */',
      '}',
      '',
      '.editor-divider {',
      '  position: relative;',
      '  flex: none;',
      '  width: 1px;',
      '  background: var(--line);',
      '  cursor: col-resize;',
      '}',
      '',
      '.editor-divider::after {',
      '  content: "";',
      '  position: absolute;',
      '  inset: 0 -3px;',
      '}',
    ],
  },
];

/** 夹在正文里的其他块类型 —— 让文档结构杂一些，贴近真实长文。 */
const EXTRAS = [
  ['引用', ['> 判据是「1 万字以上文档，输入延迟 ≤ 50ms」。超出就换方案，而不是硬优化。', '> 这条写进 ADR 0004 的重审条件里，是当时就定下来的。']],
  ['无序列表', ['- 第一件事是把被测链路读清楚，知道每个按键到底跑了哪些线性成本的活', '- 第二件事是分清「过不过」与「是谁慢」，后者才决定改哪儿', '- 第三件事是把结论回填到决策记录里，否则下次还会再问一遍']],
  ['有序列表', ['1. 造一份够长的文档', '2. 在中部连续输入，记录按键到出字的耗时', '3. 换一种结构再测一遍，两次的差就是那个因素的成本']],
  ['表格', ['| 视图 | 内核 | 每次按键的额外成本 |', '|---|---|---|', '| 富文本 | Milkdown | 整篇重跑高亮 + 整篇序列化 |', '| 源码 | CodeMirror | 无 |', '| 分屏 | 两者 | 左栏的成本 + 右栏整篇重解析 |']],
  ['公式', ['行内公式 $O(n)$ 与独立公式：', '', '$$', 'T(n) = \\sum_{i=1}^{n} c_i \\cdot f_i(n)', '$$']],
  ['围栏', ['```mermaid', 'flowchart LR', '  A[按键] --> B[事务派发]', '  B --> C[高亮重建]', '  B --> D[整篇序列化]', '  B --> E[光标位置换算]', '  C --> F[重排与绘制]', '  D --> F', '  E --> F', '```']],
];

// ---------------------------------------------------------------------------
// 组装
// ---------------------------------------------------------------------------

const FENCE = '`'.repeat(3);

function codeBlock(index) {
  const block = CODE_BLOCKS[index % CODE_BLOCKS.length];
  return [`${FENCE}${block.language}`, ...block.lines, FENCE, ''];
}

/** 替身用的中文句子（含标点约 105 字）。 */
const STAND_IN_SENTENCE =
  '这一段是对照组里的替身内容，它占据的位置原本是一个代码块。' +
  '这里刻意用连续的中文把它填满，长度与那段代码逐字符相当，' +
  '目的是让两份文档的字符数、节点数与位置空间都接近，' +
  '从而把「有没有代码块」这一个变量单独隔离出来，' +
  '而不是把文档长度差异也算进两次测量的差里。';

/**
 * 对照组用的替身：把代码块换成一整段中文，**字符数精确对齐**。
 *
 * 为什么必须精确：ProseMirror 的位置空间（`doc.content.size`）由字符数与
 * 节点边界共同决定，而 `markdownEmitPlugin` 的整篇序列化成本与它成正比。
 * 如果替身比代码块短，对照组就会因为「文档更小」而天然更快 ——
 * 那次测量差里就混进了长度差异，归因结论不成立。
 */
function codeBlockStandIn(index) {
  const block = CODE_BLOCKS[index % CODE_BLOCKS.length];
  const target = block.lines.join('\n').length;

  let text = '';
  while (text.length < target) text += STAND_IN_SENTENCE;
  text = text.slice(0, target);

  // 收在一个完整的句号上，读起来不别扭（截断点太靠前就不收，宁可留半句）
  const lastStop = text.lastIndexOf('。');
  if (lastStop > target * 0.8) text = text.slice(0, lastStop + 1);

  return [text, ''];
}

/**
 * 组装文档。
 *
 * `sections` 为 `null` 时按目标字数自动增长；给了数字就按那个节数铺 ——
 * 对照组靠它拿到**与完整版完全相同的节数**（节点数对齐）。
 */
function build({ withCode, sections = null }) {
  const lines = [
    '# 长文档输入延迟压测材料',
    '',
    '> 本文件由 `scripts/make-v3-longdoc.mjs` 生成，用于 V3（长文档输入延迟）验证。',
    '> 打开之后请**定位到文档中部**再开始打字 —— 文档开头的成本与结尾不同，',
    '> 这一点本身就是一条归因线索（见脚本注释）。',
    '',
  ];

  let section = 0;
  let codeIndex = 0;

  // 先按目标字数铺正文，再收尾 —— 字数由脚本自己算，不靠人估。
  // `sections === null` 时按字数增长，否则按指定节数（对照组走这条）。
  while (sections === null ? wordCount(lines.join('\n')) < TARGET_WORDS : section < sections) {
    section += 1;
    lines.push(`## 第 ${section} 节`, '');

    for (let i = 0; i < 3; i += 1) {
      lines.push(PARAGRAPHS[(section * 3 + i) % PARAGRAPHS.length], '');
    }

    const extra = EXTRAS[section % EXTRAS.length];
    lines.push(...extra[1], '');

    // 每 3 节放一个代码块 —— 密度接近一份真实的技术长文
    if (section % 3 === 0) {
      // 必须展开：`push(数组)` 会把整个数组当成**一个元素**，
      // `join('\n')` 时被 `toString()` 成逗号连接的一行，代码块就塌成单行了
      lines.push(...(withCode ? codeBlock(codeIndex) : codeBlockStandIn(codeIndex)));
      codeIndex += 1;
    }
  }

  lines.push('## 结尾', '', '这是文档的最后一段。如果在这里打字的延迟明显低于中部，说明成本与光标位置相关。', '');
  return { text: lines.join('\n'), sections: section };
}

// ---------------------------------------------------------------------------
// 字数口径（`src/core/statistics.ts` 的精简复刻，只用于打印进度）
// ---------------------------------------------------------------------------

function isCjk(cp) {
  return (
    (cp >= 0x3040 && cp <= 0x30ff) ||
    (cp >= 0x3400 && cp <= 0x4dbf) ||
    (cp >= 0x4e00 && cp <= 0x9fff) ||
    (cp >= 0xf900 && cp <= 0xfaff) ||
    (cp >= 0xac00 && cp <= 0xd7af) ||
    (cp >= 0x20000 && cp <= 0x2ebef) ||
    (cp >= 0x3001 && cp <= 0x303f) ||
    (cp >= 0xff01 && cp <= 0xff60) ||
    (cp >= 0xffe0 && cp <= 0xffe6)
  );
}

function wordCount(text) {
  let cjk = 0;
  for (const ch of text) {
    const cp = ch.codePointAt(0);
    if (cp !== undefined && isCjk(cp)) cjk += 1;
  }
  const latin = text.match(/[A-Za-z0-9]+(?:['\u2019_-][A-Za-z0-9]+)*/g)?.length ?? 0;
  return cjk + latin;
}

// ---------------------------------------------------------------------------
// 落盘
// ---------------------------------------------------------------------------

mkdirSync(outDir, { recursive: true });

const full = build({ withCode: true });
// 对照组复用完整版的节数 —— 节点数对齐，两次测量只差「有没有 code_block」
const control = build({ withCode: false, sections: full.sections });

const targets = [
  ['v3-longdoc.md', full.text],
  ['v3-longdoc-nocode.md', control.text],
];

for (const [name, text] of targets) {
  writeFileSync(join(outDir, name), text, 'utf8');
  const lines = text.split('\n').length;
  console.log(
    `${name.padEnd(24)} ${String(wordCount(text)).padStart(7)} 字  ${String(lines).padStart(5)} 行  ${String(text.length).padStart(7)} 字符`,
  );
}

// 自检。两条都是「产物看起来正常、但测的东西不是你想测的」那类错误：
// 前者是 `push(数组)` 不展开导致代码块塌成单行（这个坑真踩过），
// 后者是 front matter 会让整个文件降级到代码内核（ADR 0009）。
for (const [name, text] of targets) {
  const fenceLines = text.split('\n').filter((line) => line.startsWith(FENCE)).length;
  if (fenceLines % 2 !== 0) {
    throw new Error(`${name}: 行首围栏 ${fenceLines} 个（奇数）—— 代码块没有配对，检查有没有漏写 ...展开`);
  }
  if (text.startsWith('---')) {
    throw new Error(`${name}: 以 --- 开头会被判定成 front matter 文件并降级到代码内核（ADR 0009），测不到 Milkdown`);
  }
  console.log(`${name}: ${fenceLines} 个围栏（配对 ✓），非 front matter ✓`);
}

console.log('');
console.log(`两份都是 ${full.sections} 节（节点数对齐）`);
const wordDelta = Math.abs(wordCount(full.text) - wordCount(control.text));
const charDelta = Math.abs(full.text.length - control.text.length);
console.log(`字符数相差 ${charDelta}（${((charDelta / full.text.length) * 100).toFixed(1)}%）`);
console.log(`字数相差   ${wordDelta}（${((wordDelta / wordCount(full.text)) * 100).toFixed(1)}%）`);
console.log('');
console.log('判据只看**字符数**：延迟跟的是字符数与位置空间（序列化要遍历全部文本、');
console.log('textBetween 要按位置数走），不是 Word 口径字数。字数这一栏两者差得多，');
console.log('是因为中文替身每个字符都算 1 字，而代码块里的拉丁词整串只算 1 个 ——');
console.log('这条差异与每次按键的成本无关，所以不影响归因。');
console.log('');
if (charDelta / full.text.length > 0.05) {
  console.log('⚠️  字符数差超过 5%，A/B 的可比性会打折 —— 请调整内容池后重跑');
} else {
  console.log('✅ 字符数与节点数都已对齐，可以做「有没有代码块」这一条 A/B');
}
