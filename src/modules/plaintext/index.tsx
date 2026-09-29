/**
 * 纯文本内核 —— 兜底实现，**不注册进注册表**（见 registry.ts 的说明）。
 *
 * 它的职责不是「编辑文本」，而是**保证任何文件都能被打开**。
 * 打开一个不认识的二进制文件，用户看到的应该是乱码，而不是「打不开」——
 * 乱码可以解释，打不开无法解释。
 */

import { useEffect, useRef } from 'react';

import type { EditorAdapter, EditorHandle, EditorMountProps, EditorSearch } from '../../core/editor/types.js';
import { findInText, replaceInText, type FindOptions, type TextMatch } from '../../core/search/find.js';

/**
 * textarea 的查找实现。
 *
 * 兜底内核的定位是「最后一道防线」，但它同时是**未知扩展名文件的日常编辑器** ——
 * 一个打不开查找的编辑器会让用户以为功能坏了，而不是「这个内核不支持」。
 * 它的实现也是最简单的一份（`value` 就是全部文本，字符偏移直接就是
 * `setSelectionRange` 的参数），所以没有理由不做。
 */
function createTextareaSearch(element: HTMLTextAreaElement): EditorSearch {
  let matches: TextMatch[] = [];
  let options: FindOptions | null = null;

  const refind = (): number => {
    matches = options === null ? [] : findInText(element.value, options);
    return matches.length;
  };

  const select = (index: number): void => {
    const hit = matches[index];
    if (!hit) return;
    element.focus();
    element.setSelectionRange(hit.from, hit.to);
  };

  return {
    find: (next) => {
      options = next;
      return refind();
    },

    reveal: (index) => select(index),

    replace: (index, replacement) => {
      const hit = matches[index];
      if (!hit || options === null) return matches.length;
      // 用 setRangeText 而不是拼字符串：它保留浏览器的原生撤销栈，
      // 用户按 Ctrl+Z 能退回替换前，而手拼字符串会把整个撤销历史清掉。
      element.setRangeText(replacement, hit.from, hit.to, 'end');
      element.dispatchEvent(new Event('input', { bubbles: true }));
      return refind();
    },

    replaceAll: (replacement) => {
      if (options === null) return 0;
      const result = replaceInText(element.value, options, replacement);
      if (result.count === 0) return 0;
      element.value = result.text;
      element.dispatchEvent(new Event('input', { bubbles: true }));
      matches = [];
      return result.count;
    },

    clear: () => {
      matches = [];
      options = null;
    },
  };
}

function PlainTextEditor({
  initialValue,
  onChange,
  onReady,
  onCursorChange,
  readOnly,
  placeholder,
}: EditorMountProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // 用 ref 转发回调，避免调用方每次渲染换函数就重建内核
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onCursorChangeRef = useRef(onCursorChange);
  onCursorChangeRef.current = onCursorChange;

  useEffect(() => {
    const element = textareaRef.current;
    if (!element) return;

    const reportCursor = () => {
      const upto = element.value.slice(0, element.selectionStart ?? 0);
      const lines = upto.split('\n');
      onCursorChangeRef.current?.({
        line: lines.length,
        column: (lines[lines.length - 1]?.length ?? 0) + 1,
      });
    };

    const handle: EditorHandle = {
      getValue: () => element.value,
      setValue: (text) => {
        if (element.value === text) return;
        // 尽量保住光标位置：重新加载后光标通常应停在原处而不是跳到开头
        const caret = element.selectionStart ?? 0;
        element.value = text;
        const next = Math.min(caret, text.length);
        element.setSelectionRange(next, next);
      },
      focus: () => element.focus(),
      // 纯文本没有富文本语义可导出，如实返回 null，由导出层退化为 <pre>
      toHtml: async () => null,
      search: createTextareaSearch(element),
    };

    onReady(handle);
    reportCursor();
  }, [onReady]);

  return (
    <textarea
      ref={textareaRef}
      className="editor"
      defaultValue={initialValue}
      readOnly={readOnly}
      spellCheck={false}
      autoComplete="off"
      autoCorrect="off"
      autoCapitalize="off"
      placeholder={placeholder}
      onChange={(event) => onChangeRef.current(event.target.value)}
      onKeyUp={() => {
        const element = textareaRef.current;
        if (!element) return;
        const upto = element.value.slice(0, element.selectionStart ?? 0);
        const lines = upto.split('\n');
        onCursorChangeRef.current?.({
          line: lines.length,
          column: (lines[lines.length - 1]?.length ?? 0) + 1,
        });
      }}
    />
  );
}

export const plainTextAdapter: EditorAdapter = {
  id: 'plaintext',
  label: '纯文本',
  // 兜底内核不参与匹配，由 registry 在无人认领时直接返回它
  matches: () => false,
  Component: PlainTextEditor,
};
