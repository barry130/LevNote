/**
 * 工作区检索面板。
 *
 * ## 它住在侧栏里，与文件树互斥
 *
 * 这是 VSCode 的做法，也是唯一不需要用户在两个面板之间做「谁该占哪块地方」
 * 决定的排法。面板本身**没有自己的状态** —— 全部在 `WorkspaceSearchStore` 里，
 * 因为那里才是能测到「防抖 / 取消 / 过期批次」这三件时序事的地方
 * （见 `core/search/workspace.ts` 的模块注释）。
 *
 * ## 三条刻意的呈现取舍
 *
 * 1. **进度与结果同时在场**。检索是流式的，用户在结果一条条冒出来的同时
 *    也要能看到「已经扫了多少」—— 只有结果没有进度，扫一个大目录时
 *    面板会显得像是卡住了；只有进度没有结果，用户会怀疑是不是没搜到。
 * 2. **「跳过」必须显示**。二进制、超大、读不到，三个数字都出现在收尾那行里。
 *    少了它，用户看不到某个文件时只能自己猜 —— 而「猜」正是硬约束 2 要消灭的东西。
 * 3. **空结果不是一句话，是一段可行动的说明**。检索最常见的困惑不是「没搜到」，
 *    而是「我明明知道它在里面」。所以空结果时明确提示那两个逃生口：
 *    关掉 `.gitignore` 尊重、检查文件夹是不是没打开。
 *
 * ## 替换（A3）
 *
 * 替换输入框只在**有结果**时可用 —— 「对着列表替换」而不是「对着工作区替换」：
 * 用户看到的确认框说的就是眼前这份结果。替换编排（确认框、脏文件清单、
 * provider 读写）在 `App`；面板只管输入与把结果那句话显示出来。
 */

import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';

import {
  describeSkips,
  describeSummary,
  flattenResults,
  splitPreview,
  type WorkspaceMatch,
  type WorkspaceSearchStore,
} from '../core/search/workspace.js';
import { basename } from '../core/path.js';

export interface SearchPanelProps {
  readonly store: WorkspaceSearchStore;
  /** 工作区根目录；null 表示还没打开文件夹。 */
  readonly root: string | null;
  /** 平台是否支持检索（浏览器预览不支持）。 */
  readonly canSearch: boolean;
  readonly onOpenMatch: (path: string, match: WorkspaceMatch) => void;
  readonly onClose: () => void;
  /**
   * 对给定文件做「全部替换」。返回给用户看的一句话（null = 用户取消了确认框）。
   * 编排在 App：那里有确认对话框与打开标签页的脏清单。
   */
  readonly onReplaceAll: (paths: readonly string[], replacement: string) => Promise<string | null>;
}

export function SearchPanel({ store, root, canSearch, onOpenMatch, onClose, onReplaceAll }: SearchPanelProps) {
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [replacement, setReplacement] = useState('');
  const [replacing, setReplacing] = useState(false);
  const [replaceMessage, setReplaceMessage] = useState<string | null>(null);

  // 打开面板就把焦点给输入框：这是用户打开它的唯一理由
  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const rows = flattenResults(state.files);
  const skipped = describeSkips(state.progress);
  const resultPaths = useMemo(() => state.files.map((file) => file.path), [state.files]);
  const canReplace = canSearch && root !== null && resultPaths.length > 0 && !replacing && state.query !== '';

  const handleReplace = (): void => {
    if (!canReplace) return;
    setReplacing(true);
    setReplaceMessage(null);
    void onReplaceAll(resultPaths, replacement).then((message) => {
      setReplacing(false);
      setReplaceMessage(message ?? null);
    });
  };

  return (
    <aside className="sidebar search-panel">
      <header className="sidebar-head">
        <span className="sidebar-title">在文件夹中检索</span>
        <span className="grow" />
        <button type="button" title="关闭检索面板" onClick={onClose}>
          ✕
        </button>
      </header>

      <div className="search-form">
        <div className="search-row">
          <input
            ref={inputRef}
            type="search"
            className="search-input"
            placeholder={root === null ? '先打开一个文件夹' : '检索内容…'}
            value={state.query}
            disabled={!canSearch || root === null}
            spellCheck={false}
            autoComplete="off"
            onChange={(event) => store.setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key !== 'Enter') return;
              event.preventDefault();
              // 回车 = 「我打完了」，不该再等防抖那 250ms
              store.searchNow();
            }}
          />
          <button
            type="button"
            title={state.running ? '停止检索' : '重新检索'}
            className={state.running ? 'search-stop' : ''}
            disabled={!canSearch || root === null || state.query === ''}
            onClick={() => (state.running ? store.cancel() : store.searchNow())}
          >
            {state.running ? '停止' : '检索'}
          </button>
        </div>

        <div className="search-toggles">
          <Toggle
            label="区分大小写"
            text="Aa"
            on={state.options.caseSensitive}
            onClick={() => store.setOption('caseSensitive', !state.options.caseSensitive)}
          />
          <Toggle
            label="全词匹配"
            text="ab|"
            on={state.options.wholeWord}
            onClick={() => store.setOption('wholeWord', !state.options.wholeWord)}
          />
          <Toggle
            label="正则表达式"
            text=".*"
            on={state.options.regex}
            onClick={() => store.setOption('regex', !state.options.regex)}
          />
          <Toggle
            label="包含被忽略的文件（.gitignore）"
            text="⊙"
            on={state.options.includeIgnored}
            onClick={() => store.setOption('includeIgnored', !state.options.includeIgnored)}
          />
        </div>

        <p className="search-hint">
          {state.options.includeIgnored
            ? '已关掉 .gitignore 规则；以 . 开头的隐藏项仍然跳过（否则 .git 会被整棵搜一遍）。'
            : '按 .gitignore 与 .ignore 跳过被忽略的文件。'}
        </p>

        {resultPaths.length > 0 && (
          <div className="search-row">
            <input
              type="search"
              className="search-input"
              placeholder="替换为…（留空 = 删除命中）"
              value={replacement}
              spellCheck={false}
              autoComplete="off"
              onChange={(event) => setReplacement(event.target.value)}
              onKeyDown={(event) => {
                if (event.key !== 'Enter') return;
                event.preventDefault();
                handleReplace();
              }}
            />
            <button
              type="button"
              title={`对结果里的 ${String(resultPaths.length)} 个文件做全部替换（打开且未保存的文件会跳过）`}
              disabled={!canReplace}
              onClick={handleReplace}
            >
              {replacing ? '替换中…' : '替换'}
            </button>
          </div>
        )}
        {replaceMessage !== null && <p className="search-hint">{replaceMessage}</p>}
      </div>

      {!canSearch && (
        <div className="search-note">工作区检索只在桌面版可用 —— 浏览器预览里没有真实目录。</div>
      )}

      {canSearch && root === null && (
        <div className="search-note">还没有打开文件夹，检索无从开始。按 Ctrl+Shift+O 打开一个。</div>
      )}

      {state.error && (
        <div className="sidebar-error">
          {state.error.message}
          {state.error.detail ? `（${state.error.detail}）` : ''}
        </div>
      )}

      {canSearch && root !== null && (
        <div className="search-status">
          {state.running && (
            <span className="search-progress">
              已扫 {state.progress.scannedFiles} 个文件 · 命中 {state.progress.totalMatches} 处…
            </span>
          )}
          {!state.running && state.summary !== null && <span>{describeSummary(state.summary)}</span>}
          {!state.running && state.summary === null && state.error === null && (
            <span>{state.query === '' ? '输入要检索的内容。' : '尚未检索。'}</span>
          )}
          {skipped !== null && <span className="search-skipped">{skipped}</span>}
        </div>
      )}

      {canSearch && root !== null && rows.length > 0 && (
        <div className="search-results" role="list">
          {rows.map((row) =>
            row.kind === 'file' ? (
              <div key={row.key} className="search-file" title={row.path}>
                <span className="search-file-name">{basename(row.path)}</span>
                <span className="search-file-count">{row.count}</span>
              </div>
            ) : (
              <button
                key={row.key}
                type="button"
                className="search-hit"
                title={`${row.path}:${String(row.match.line)}:${String(row.match.column + 1)}`}
                onClick={() => onOpenMatch(row.path, row.match)}
              >
                <span className="search-line-no">{row.match.line}</span>
                <span className="search-line-text">
                  {row.match.textOffset > 0 && <span className="search-ellipsis">…</span>}
                  <Preview match={row.match} />
                  {row.match.tailCut && <span className="search-ellipsis">…</span>}
                </span>
              </button>
            ),
          )}
        </div>
      )}

      {canSearch && root !== null && !state.running && state.summary !== null && rows.length === 0 && (
        <div className="search-note">
          <p>没有命中。</p>
          {!state.options.includeIgnored && (
            <p className="hint">
              如果确认这个文件夹里应该有，试试打开上面那个「⊙」——
              它会把被 .gitignore 跳过的文件也搜进来。
            </p>
          )}
        </div>
      )}
    </aside>
  );
}

/** 一行预览，命中部分高亮。 */
function Preview({ match }: { readonly match: WorkspaceMatch }) {
  const { before, matched, after } = splitPreview(match);
  return (
    <>
      {before}
      {matched === '' ? (
        // 零宽匹配（`^` / `\b` / `a*`）没有可高亮的字符。画一个细竖线，
        // 好过什么都不画 —— 后者会让「搜到了但看不到」变成一句空话
        <span className="search-caret" />
      ) : (
        <mark className="search-mark">{matched}</mark>
      )}
      {after}
    </>
  );
}

function Toggle({
  label,
  text,
  on,
  onClick,
}: {
  readonly label: string;
  readonly text: string;
  readonly on: boolean;
  readonly onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`search-toggle${on ? ' on' : ''}`}
      aria-pressed={on}
      title={label}
      onClick={onClick}
    >
      {text}
      {/* 纯符号按钮必须有一个可访问名（与活动栏同一条理由），
          而且项目的测试辅助函数是按文字找元素的 */}
      <span className="sr-only">{label}</span>
    </button>
  );
}
