/**
 * 代码块的语言选择器。
 *
 * ## 它长在哪儿、为什么长在那儿
 *
 * 挂在代码块**右上角那个语言标签**的正下方、向右贴住标签（`horizontal: 'end'`）。
 * 语言标签本身由内核的 NodeView 画（见 `modules/markdown/code-block-view.ts`），
 * 面板由外壳画 —— 与选中工具条 / 斜杠菜单 / 表格工具条完全同一套分工：
 * **内核只报状态与几何，外壳画界面**。
 *
 * 与另外三个浮层不同的一点：这个面板是**点击触发**的，而且自己带一个输入框。
 * 所以它不用 `window` 上的捕获键盘（那个是给「焦点还在编辑器里」的斜杠菜单用的）——
 * 焦点就在下面这个输入框上，`onKeyDown` 直接收得到。
 *
 * ## 三个「关上」的时机
 *
 * - **Esc** —— 用户明确说不要了；
 * - **点面板外面** —— 常规的浮层行为；
 * - **编辑器滚动** —— 面板锚在标签上，而标签会跟着内容跑。
 *   不关的话面板会停在原地和标签分家，看起来像飘走了。
 *   原生 `<select>` 的下拉也是这个行为。
 *
 * 滚动那条要排掉「面板自己那个列表在滚」：`scroll` 不冒泡，但捕获阶段
 * 一样能收到它，不排掉的话用户一滚候选列表面板就没了。
 *
 * ## 为什么输入框用 `focus({ preventScroll: true })`
 *
 * 不传这个的话，聚焦可能让浏览器去滚动祖先容器把输入框露出来 ——
 * 而那次滚动会被上面那条监听接住，面板**刚打开就自己关上**。
 * 这个 bug 只在某些布局下复现，属于「偶发、难查」那一类，所以从一开始就堵掉。
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';

import { rankCommands } from '../core/commands/fuzzy.js';
import type { CodeBlockState, EditorCodeBlock } from '../core/editor/types.js';
import { supportedLanguages } from '../core/highlight/hljs.js';
import { computeFloatingPlacement } from './floating.js';

interface LanguageRow {
  /** 写进围栏 info string 的语言标识。**空串 = 清除语言**。 */
  readonly id: string;
  readonly label: string;
  /** 拼音 / 别名，供模糊搜索。中文名没有「词首」概念，靠这个才搜得到。 */
  readonly keywords: readonly string[];
}

/** 「纯文本」那一项：它对应 `setLanguage('')`，即把围栏的 info string 清掉。 */
const PLAIN_ROW: LanguageRow = {
  id: '',
  label: '纯文本',
  keywords: ['text', 'txt', 'plain', 'chunwenben', 'wenben'],
};

/**
 * 常见语言的中文名与别名。
 *
 * **只补常见的十几种**，其余的照原样显示 `hljs` 的语言标识 ——
 * 给 `gams`（通用代数建模系统）编一个中文名，只会让它更难被找到，
 * 因为用户脑子里本来就是 `gams`。
 *
 * 数据本身来自 `supportedLanguages()`（也就是 `hljs` 自己），这里**不另立一份清单**：
 * 另立的那一份迟早会和真正能高亮的语言对不上，而那种错是静默的 ——
 * 用户从列表里选了一个语言，高亮却退回自动识别，他看不出为什么。
 */
const LABELS: Readonly<Record<string, { label: string; keywords: readonly string[] }>> = {
  bash: { label: 'Bash', keywords: ['shell', 'sh', '命令行'] },
  c: { label: 'C', keywords: [] },
  cpp: { label: 'C++', keywords: ['cplusplus'] },
  csharp: { label: 'C#', keywords: ['csharp', 'dotnet'] },
  css: { label: 'CSS', keywords: ['样式'] },
  diff: { label: 'Diff', keywords: ['patch', '补丁'] },
  go: { label: 'Go', keywords: ['golang'] },
  graphql: { label: 'GraphQL', keywords: [] },
  ini: { label: 'INI', keywords: ['toml', 'conf', '配置'] },
  java: { label: 'Java', keywords: [] },
  javascript: { label: 'JavaScript', keywords: ['js', '前端'] },
  json: { label: 'JSON', keywords: [] },
  kotlin: { label: 'Kotlin', keywords: ['kt'] },
  less: { label: 'Less', keywords: [] },
  lua: { label: 'Lua', keywords: [] },
  makefile: { label: 'Makefile', keywords: ['make'] },
  markdown: { label: 'Markdown', keywords: ['md'] },
  objectivec: { label: 'Objective-C', keywords: ['objc'] },
  perl: { label: 'Perl', keywords: [] },
  php: { label: 'PHP', keywords: [] },
  'php-template': { label: 'PHP 模板', keywords: ['phptemplate', '模板'] },
  python: { label: 'Python', keywords: ['py'] },
  'python-repl': { label: 'Python 交互式', keywords: ['repl', 'jupyter'] },
  r: { label: 'R', keywords: [] },
  ruby: { label: 'Ruby', keywords: ['rb'] },
  rust: { label: 'Rust', keywords: ['rs'] },
  scss: { label: 'SCSS', keywords: ['sass'] },
  shell: { label: 'Shell', keywords: ['bash', 'sh', '命令行'] },
  sql: { label: 'SQL', keywords: ['数据库'] },
  swift: { label: 'Swift', keywords: [] },
  typescript: { label: 'TypeScript', keywords: ['ts'] },
  vbnet: { label: 'VB.NET', keywords: ['visualbasic'] },
  wasm: { label: 'WebAssembly', keywords: ['wat', '汇编'] },
  xml: { label: 'XML / HTML', keywords: ['html', 'svg'] },
  yaml: { label: 'YAML', keywords: ['yml'] },
};

const LANGUAGE_ROWS: readonly LanguageRow[] = [
  PLAIN_ROW,
  ...supportedLanguages()
    /*
     * `plaintext` 不列：上面那一项「纯文本」就是它，但两者在 markdown 里
     * 写出来并不一样（```` ``` ```` 与 ```` ```plaintext ````）。
     * 两个都列会让人以为要选两次。已经在用 `plaintext` 的文件仍然可以
     * 通过下面那个「使用 “…”」选项选回去。
     */
    .filter((id) => id !== 'plaintext')
    .map((id) => ({ id, label: LABELS[id]?.label ?? id, keywords: LABELS[id]?.keywords ?? [] })),
];

export interface CodeBlockPickerProps {
  state: CodeBlockState;
  codeBlock: EditorCodeBlock;
  /** 代码块是否显示行号。这是外壳的视图偏好，不是文档内容。 */
  lineNumbers: boolean;
  onToggleLineNumbers: (next: boolean) => void;
}

export function CodeBlockPicker({ state, codeBlock, lineNumbers, onToggleLineNumbers }: CodeBlockPickerProps) {
  const { anchor, language } = state;
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const [query, setQuery] = useState('');
  const [placement, setPlacement] = useState<{ left: number; top: number } | null>(null);

  const rows = useMemo<LanguageRow[]>(() => {
    const typed = query.trim();

    /**
     * 空查询时把**当前语言**补到最前面，前提是它不在候选里。
     *
     * ```` ```js ```` 就是这种情况：hljs 的正式标识是 `javascript`，
     * 而用户（以及绝大多数文档）写的是 `js`。不补这一行的话，面板打开时
     * 「当前用的是哪个」无处可显示 —— 所有行都不高亮，用户会以为
     * 这份文档的语言没被认出来。补上之后它就是一个普通候选项：
     * 点它等于原样保留，点 `JavaScript` 才会把围栏改成正式标识。
     */
    const current: LanguageRow[] =
      typed === '' && language !== '' && !LANGUAGE_ROWS.some((row) => row.id === language)
        ? [{ id: language, label: language, keywords: [] }]
        : [];

    const hits = rankCommands(query, LANGUAGE_ROWS, (item) => ({
      title: item.label,
      keywords: item.keywords,
      id: item.id,
    })).map((hit) => hit.item);

    if (typed === '' || LANGUAGE_ROWS.some((row) => row.id === typed)) return [...current, ...hits];
    // 原样用用户输入的值。少了这一项，「hljs 不认识的围栏」（`mermaid` 之类）
    // 就没法从界面上选出来 —— 用户会被卡在列表里找不到自己的语言
    return [...hits, { id: typed, label: `使用 “${typed}”`, keywords: [] }];
  }, [query, language]);

  /**
   * 初始高亮落在**当前语言**那一行上，而不是永远从第一行开始。
   *
   * 这里用 `rows` 的初值算一次即可（`useState` 的初始化函数只在首次渲染跑）——
   * 之后改筛选词时由输入框的 `onChange` 把它重置回 0，
   * 所以**不需要**一个 `[query]` 的 effect（那样反而会在挂载时把初始高亮冲掉）。
   */
  const [highlighted, setHighlighted] = useState(() => {
    const at = rows.findIndex((row) => row.id === language);
    return at === -1 ? 0 : at;
  });

  // 列表变短时把高亮收回范围内
  useEffect(() => {
    setHighlighted((at) => Math.min(at, Math.max(0, rows.length - 1)));
  }, [rows.length]);

  useLayoutEffect(() => {
    const node = rootRef.current;
    if (node === null) return;

    const { width, height } = node.getBoundingClientRect();

    // 摆在语言标签**下方**，并向右贴住标签 —— 标签在代码块右上角，
    // 面板要往**左**展开才不会伸到正文栏外面去（`horizontal: 'end'` 的理由）
    setPlacement(
      computeFloatingPlacement(
        anchor,
        { width, height },
        { vertical: 'below', horizontal: 'end' },
        { width: window.innerWidth, height: window.innerHeight },
      ),
    );
  }, [anchor]);

  // 先聚焦（并且不触发滚动），再挂下面那个滚动监听 —— 顺序不能反，
  // 否则聚焦引起的那次滚动会把刚打开的面板关掉
  useEffect(() => {
    inputRef.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      const node = rootRef.current;
      if (node === null || !(event.target instanceof Node) || node.contains(event.target)) return;
      codeBlock.close();
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [codeBlock]);

  useEffect(() => {
    const onScroll = (event: Event) => {
      const node = rootRef.current;
      // 面板自己那个候选列表也会滚。捕获阶段收得到它，得排掉
      if (node !== null && event.target instanceof Node && node.contains(event.target)) return;
      codeBlock.close();
    };
    // 捕获阶段：`scroll` 不冒泡，挂在 `window` 上只有捕获才收得到
    window.addEventListener('scroll', onScroll, true);
    return () => window.removeEventListener('scroll', onScroll, true);
  }, [codeBlock]);

  const apply = useCallback(
    (id: string) => {
      // 关面板与写文档都在内核里一次做完 —— 文档坐标只有它知道
      codeBlock.setLanguage(id);
    },
    [codeBlock],
  );

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    // 输入法组合期间不要抢键：中文输入时 Enter 是「上屏」
    if (event.nativeEvent.isComposing) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      codeBlock.close();
      return;
    }

    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      if (rows.length === 0) return;
      const step = event.key === 'ArrowDown' ? 1 : -1;
      // 循环而不是停在两端：候选很少，撞到底再按一下没反应会显得卡住了
      setHighlighted((at) => (at + step + rows.length) % rows.length);
      return;
    }

    if (event.key === 'Enter') {
      const row = rows[highlighted];
      if (row === undefined) return;
      event.preventDefault();
      apply(row.id);
    }
  };

  return (
    <div
      ref={rootRef}
      className="cb-picker"
      role="dialog"
      aria-label="代码语言"
      style={{
        left: placement?.left ?? 0,
        top: placement?.top ?? 0,
        visibility: placement === null ? 'hidden' : 'visible',
      }}
    >
      <input
        ref={inputRef}
        type="text"
        className="cb-search"
        value={query}
        placeholder="搜索语言…"
        aria-label="搜索语言"
        role="combobox"
        aria-expanded="true"
        aria-controls="cb-picker-list"
        spellCheck={false}
        onChange={(event) => {
          setQuery(event.target.value);
          // 筛选词一变就回到第一项：不回的话，用户翻了几下再改搜索词，
          // 高亮会停在一条和当前搜索毫无关系的项上
          setHighlighted(0);
        }}
        onKeyDown={onKeyDown}
      />

      <div className="cb-list" id="cb-picker-list" role="listbox" aria-label="候选语言">
        {rows.length === 0 ? (
          <div className="cb-empty">没有匹配的语言</div>
        ) : (
          rows.map((row, index) => (
            <button
              key={row.id === '' ? 'plain' : row.id}
              type="button"
              role="option"
              aria-selected={index === highlighted}
              className={`cb-item${index === highlighted ? ' highlighted' : ''}${row.id === language ? ' current' : ''}`}
              // 不拦这一下，焦点会离开输入框，接着触发上面那个「点面板外面」的监听 ——
              // 面板会在 click 到达按钮之前被卸载（与斜杠菜单同一个坑）
              onMouseDown={(event) => event.preventDefault()}
              onMouseEnter={() => setHighlighted(index)}
              onClick={() => apply(row.id)}
            >
              <span className="cb-label">{row.label}</span>
              {/* 标识与名字相同时不重复画一遍（```` ```js ```` 补出来的那一行就是这样） */}
              {row.id === '' || row.id === row.label ? null : <span className="cb-id">{row.id}</span>}
            </button>
          ))
        )}
      </div>

      <label className="cb-toggle">
        <input
          type="checkbox"
          checked={lineNumbers}
          onChange={(event) => onToggleLineNumbers(event.target.checked)}
        />
        显示行号
      </label>
    </div>
  );
}
