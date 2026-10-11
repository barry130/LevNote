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
import { renderReplaceReport, type ReplaceReport } from '../core/search/replace.js';
import { t } from '../core/i18n/index.js';

export interface SearchPanelProps {
  readonly store: WorkspaceSearchStore;
  /** 工作区根目录；null 表示还没打开文件夹。 */
  readonly root: string | null;
  /** 平台是否支持检索（浏览器预览不支持）。 */
  readonly canSearch: boolean;
  readonly onOpenMatch: (path: string, match: WorkspaceMatch) => void;
  readonly onClose: () => void;
  /**
   * 对给定文件做「全部替换」。返回**汇报的结构**（null = 用户取消了确认框）——
   * 句子由本组件在渲染的那一刻翻出来（ADR 0047）。
   * 编排在 App：那里有确认对话框与打开标签页的脏清单。
   */
  readonly onReplaceAll: (paths: readonly string[], replacement: string) => Promise<ReplaceReport | null>;
}

export function SearchPanel({ store, root, canSearch, onOpenMatch, onClose, onReplaceAll }: SearchPanelProps) {
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [replacement, setReplacement] = useState('');
  const [replacing, setReplacing] = useState(false);
  // 存的是**汇报的结构**，不是句子（ADR 0047）：翻译留到渲染那一刻，
  // 所以切语言之后这行字会跟着变，而不是留在旧语言里。
  const [replaceReport, setReplaceReport] = useState<ReplaceReport | null>(null);

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
    setReplaceReport(null);
    void onReplaceAll(resultPaths, replacement).then((report) => {
      setReplacing(false);
      setReplaceReport(report ?? null);
    });
  };

  return (
    <aside className="sidebar search-panel">
      <header className="sidebar-head">
        <span className="sidebar-title">{t('ui.searchpanel.title')}</span>
        <span className="grow" />
        <button type="button" title={t('ui.searchpanel.close')} onClick={onClose}>
          ✕
        </button>
      </header>

      <div className="search-form">
        <div className="search-row">
          <input
            ref={inputRef}
            type="search"
            className="search-input"
            placeholder={root === null ? t('ui.searchpanel.needFolder') : t('ui.searchpanel.placeholder')}
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
            title={state.running ? t('ui.searchpanel.stopTitle') : t('ui.searchpanel.rerunTitle')}
            className={state.running ? 'search-stop' : ''}
            disabled={!canSearch || root === null || state.query === ''}
            onClick={() => (state.running ? store.cancel() : store.searchNow())}
          >
            {state.running ? t('ui.searchpanel.stop') : t('ui.searchpanel.run')}
          </button>
        </div>

        <div className="search-toggles">
          <Toggle
            label={t('ui.find.caseSensitive')}
            text="Aa"
            on={state.options.caseSensitive}
            onClick={() => store.setOption('caseSensitive', !state.options.caseSensitive)}
          />
          <Toggle
            label={t('ui.find.wholeWord')}
            text="ab|"
            on={state.options.wholeWord}
            onClick={() => store.setOption('wholeWord', !state.options.wholeWord)}
          />
          <Toggle
            label={t('ui.find.regex')}
            text=".*"
            on={state.options.regex}
            onClick={() => store.setOption('regex', !state.options.regex)}
          />
          <Toggle
            label={t('ui.searchpanel.includeIgnored')}
            text="⊙"
            on={state.options.includeIgnored}
            onClick={() => store.setOption('includeIgnored', !state.options.includeIgnored)}
          />
        </div>

        <p className="search-hint">
          {state.options.includeIgnored
            ? t('ui.searchpanel.ignoredHintOn')
            : t('ui.searchpanel.ignoredHintOff')}
        </p>

        {resultPaths.length > 0 && (
          <div className="search-row">
            <input
              type="search"
              className="search-input"
              placeholder={t('ui.searchpanel.replacePlaceholder')}
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
              title={t('ui.searchpanel.replaceTitle', { n: resultPaths.length })}
              disabled={!canReplace}
              onClick={handleReplace}
            >
              {replacing ? t('ui.searchpanel.replacing') : t('ui.searchpanel.replace')}
            </button>
          </div>
        )}
        {replaceReport !== null && <p className="search-hint">{renderReplaceReport(replaceReport)}</p>}
      </div>

      {!canSearch && (
        <div className="search-note">{t('ui.searchpanel.desktopOnly')}</div>
      )}

      {canSearch && root === null && (
        <div className="search-note">{t('ui.searchpanel.noFolder')}</div>
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
              {t('ui.searchpanel.progress', { files: state.progress.scannedFiles, matches: state.progress.totalMatches })}
            </span>
          )}
          {!state.running && state.summary !== null && <span>{describeSummary(state.summary)}</span>}
          {!state.running && state.summary === null && state.error === null && (
            <span>{state.query === '' ? t('ui.searchpanel.typeQuery') : t('ui.searchpanel.notRun')}</span>
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
          <p>{t('ui.searchpanel.noMatches')}</p>
          {!state.options.includeIgnored && <p className="hint">{t('ui.searchpanel.noMatchesHint')}</p>}
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
