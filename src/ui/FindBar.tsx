/**
 * 查找 / 替换栏。
 *
 * 它是**外壳**，不是内核的一部分：所有匹配与替换都交给 `EditorHandle.search`，
 * 这一层只负责「用户输入 → 动作序列」的翻译与状态显示。
 * 因此三个内核（Markdown / 代码 / 纯文本）共用这一个界面，
 * 行为差异只来自内核自己实现的匹配语义。
 *
 * ## 两个必须做对的地方
 *
 * 1. **每次输入都重新查找**，并且立刻定位到第一个匹配。
 *    这是「边打边看」的基础；如果要求按回车才搜，用户会以为功能没反应。
 * 2. **内核不支持查找时如实说明**。给一个能点但没反应的按钮，
 *    比明说「不支持」糟糕得多 —— 用户会反复点，然后怀疑软件坏了。
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import type { EditorHandle } from '../core/editor/types.js';
import type { FindOptions } from '../core/search/find.js';

export interface FindBarProps {
  handle: EditorHandle | null;
  onClose(): void;
}

export function FindBar({ handle, onClose }: FindBarProps) {
  const [query, setQuery] = useState('');
  const [replacement, setReplacement] = useState('');
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [wholeWord, setWholeWord] = useState(false);
  const [regex, setRegex] = useState(false);
  const [showReplace, setShowReplace] = useState(false);
  const [count, setCount] = useState(0);
  const [index, setIndex] = useState(-1);
  const [notice, setNotice] = useState('');

  const inputRef = useRef<HTMLInputElement | null>(null);
  const search = handle?.search ?? null;

  const options: FindOptions = { query, caseSensitive, wholeWord, regex };

  // 打开查找栏就聚焦输入框：用户按 Ctrl+F 之后一定是想打字
  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  /**
   * 条件变化就重跑一次查找。
   *
   * 依赖里带上 `search`：切标签页会换一个内核实例，旧实例上的匹配列表对新内核无效。
   * 不重跑的话，用户切回来会看到「1 / 7」但按下一个毫无反应。
   */
  useEffect(() => {
    if (!search) {
      setCount(0);
      setIndex(-1);
      return;
    }
    if (query === '') {
      search.clear();
      setCount(0);
      setIndex(-1);
      setNotice('');
      return;
    }

    const total = search.find(options);
    setCount(total);
    setIndex(total > 0 ? 0 : -1);
    // 正则非法时 find 会返回 0（见 find.ts 的 createFindPattern）——
    // 这时要给一个明确的提示，否则用户会以为「这个正则没有匹配」
    setNotice(regex && total === 0 ? '正则无效，或没有匹配' : '');
    if (total > 0) search.reveal(0);
    // options 是每帧新建的对象，逐字段列出依赖，避免每次渲染都重跑
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, query, caseSensitive, wholeWord, regex]);

  const step = useCallback(
    (delta: number) => {
      if (!search || count === 0) return;
      const next = ((index + delta) % count + count) % count;
      setIndex(next);
      search.reveal(next);
    },
    [search, count, index],
  );

  const replaceCurrent = useCallback(() => {
    if (!search || count === 0 || index < 0) return;
    const total = search.replace(index, replacement);
    setCount(total);
    setIndex(total > 0 ? Math.min(index, total - 1) : -1);
  }, [search, count, index, replacement]);

  const replaceEvery = useCallback(() => {
    if (!search || count === 0) return;
    const done = search.replaceAll(replacement);
    setCount(0);
    setIndex(-1);
    setNotice(`已替换 ${done} 处`);
  }, [search, count, replacement]);

  const close = useCallback(() => {
    search?.clear();
    onClose();
  }, [search, onClose]);

  if (!search) {
    return (
      <div className="findbar">
        <span className="grow">
          当前编辑器内核不支持在文档内查找。切换到 Markdown 或代码内核后可用。
        </span>
        <button type="button" onClick={onClose}>
          关闭
        </button>
      </div>
    );
  }

  return (
    <div className="findbar">
      <input
        ref={inputRef}
        className="find-input"
        type="text"
        value={query}
        placeholder="查找…"
        spellCheck={false}
        onChange={(event) => setQuery(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault();
            step(event.shiftKey ? -1 : 1);
          } else if (event.key === 'Escape') {
            event.preventDefault();
            close();
          }
        }}
      />

      <span className="find-count">{count === 0 ? (query === '' ? '' : '无结果') : `${index + 1} / ${count}`}</span>

      <button
        type="button"
        className={caseSensitive ? 'on' : ''}
        title="区分大小写"
        onClick={() => setCaseSensitive((value) => !value)}
      >
        Aa
      </button>
      <button type="button" className={wholeWord ? 'on' : ''} title="全词匹配" onClick={() => setWholeWord((value) => !value)}>
        ab
      </button>
      <button type="button" className={regex ? 'on' : ''} title="正则表达式" onClick={() => setRegex((value) => !value)}>
        .*
      </button>

      <span className="sep">│</span>
      <button type="button" title="上一个（Shift+Enter）" disabled={count === 0} onClick={() => step(-1)}>
        ↑
      </button>
      <button type="button" title="下一个（Enter）" disabled={count === 0} onClick={() => step(1)}>
        ↓
      </button>

      <span className="sep">│</span>
      <button type="button" className={showReplace ? 'on' : ''} title="显示替换" onClick={() => setShowReplace((value) => !value)}>
        替换
      </button>

      {showReplace && (
        <>
          <input
            className="find-input"
            type="text"
            value={replacement}
            placeholder="替换为…"
            spellCheck={false}
            onChange={(event) => setReplacement(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                replaceCurrent();
              }
            }}
          />
          <button type="button" disabled={count === 0} onClick={replaceCurrent}>
            替换
          </button>
          <button type="button" disabled={count === 0} onClick={replaceEvery}>
            全部替换
          </button>
        </>
      )}

      {notice !== '' && <span className="find-notice">{notice}</span>}

      <span className="grow" />
      <button type="button" title="关闭（Esc）" onClick={close}>
        ×
      </button>
    </div>
  );
}
