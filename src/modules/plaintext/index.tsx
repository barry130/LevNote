/**
 * 纯文本内核 —— 兜底实现，**不注册进注册表**（见 registry.ts 的说明）。
 *
 * 它的职责不是「编辑文本」，而是**保证任何文件都能被打开**。
 * 打开一个不认识的二进制文件，用户看到的应该是乱码，而不是「打不开」——
 * 乱码可以解释，打不开无法解释。
 */

import { useEffect, useRef } from 'react';

import type { EditorAdapter, EditorHandle, EditorMountProps, EditorPosition, EditorSearch } from '../../core/editor/types.js';
import { applyScrollRatio, readScrollRatio } from '../../core/editor/position.js';
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
  onPositionChange,
  readOnly,
  placeholder,
}: EditorMountProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // 用 ref 转发回调，避免调用方每次渲染换函数就重建内核
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onCursorChangeRef = useRef(onCursorChange);
  onCursorChangeRef.current = onCursorChange;
  const onPositionChangeRef = useRef(onPositionChange);
  onPositionChangeRef.current = onPositionChange;

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
      getPosition: (): EditorPosition => ({
        offset: element.selectionStart ?? 0,
        textLength: element.value.length,
        scrollRatio: readScrollRatio(element),
      }),
      setPosition: (position) => {
        if (position.textLength === element.value.length) {
          const inside = Math.min(Math.max(position.offset, 0), element.value.length);
          element.setSelectionRange(inside, inside);
        }
        applyScrollRatio(element, position.scrollRatio);
      },
    };

    onReady(handle);
    reportCursor();
  }, [onReady]);

  /**
   * 上报位置。
   *
   * **一份，不是三份**：`input` / `scroll` / `select` 三个事件都要报同一份东西，
   * 各写一遍的下场是其中两份迟早漏掉一个字段（而漏掉 `textLength` 的表现是
   * 「恢复位置时永远只滚不动光标」，没人会想到是这里）。
   *
   * textarea 自己就是滚动容器（`.editor { overflow: auto }`），所以
   * `readScrollRatio` 直接读它 —— 不需要像另外两个内核那样区分「谁是 scroller」。
   *
   * 定义在组件体里而不是 effect 里：effect 依赖是 `[onReady]`，
   * 放在里面就只认第一次渲染的那份闭包。
   */
  const reportPosition = () => {
    const element = textareaRef.current;
    if (!element) return;
    onPositionChangeRef.current?.({
      offset: element.selectionStart ?? 0,
      // 字符串长度是算好的字段，O(1)
      textLength: element.value.length,
      scrollRatio: readScrollRatio(element),
    });
  };

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
      onChange={(event) => {
        onChangeRef.current(event.target.value);
        reportPosition();
      }}
      // 滚动、点击、键盘都会改位置，分开挂而不是合成一个 —— 它们的触发源不同，
      // 合成一个要写一堆判断。回调本身只是读三个字段 + 写 ref，代价可以忽略。
      onScroll={reportPosition}
      onSelect={reportPosition}
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
