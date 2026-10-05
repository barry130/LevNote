/**
 * 代码内核的 React 挂载组件。
 *
 * 结构与 `MarkdownEditor.tsx` 完全同构（挂载点、生命周期、回调转发），
 * 这是刻意的：**两个内核的挂载方式长得一样**，才能说明适配层契约真的起作用了。
 * 如果这里不得不写一套不一样的东西，那说明契约漏了东西，该去改契约而不是在这里绕。
 */

import { useEffect, useRef } from 'react';

import type { EditorHandle, EditorMountProps } from '../../core/editor/types.js';
import { languageForPath } from './setup.js';
import { createCodeEditor, type CodeEditorInstance } from './setup.js';

function CodeEditor({
  initialValue,
  onChange,
  onReady,
  onCursorChange,
  onPositionChange,
  onError,
  readOnly,
  placeholder,
  wordWrap,
  path,
}: EditorMountProps & { path: string }) {
  const mountRef = useRef<HTMLDivElement | null>(null);

  // 用 ref 转发回调：它们每次渲染都是新函数，直接进依赖会让内核被反复重建
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onCursorChangeRef = useRef(onCursorChange);
  onCursorChangeRef.current = onCursorChange;
  const onPositionChangeRef = useRef(onPositionChange);
  onPositionChangeRef.current = onPositionChange;
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;

  const initialValueRef = useRef(initialValue);
  const readOnlyRef = useRef(readOnly);
  const placeholderRef = useRef(placeholder);
  const pathRef = useRef(path);
  const wrapRef = useRef(wordWrap);

  const instanceRef = useRef<CodeEditorInstance | null>(null);

  useEffect(() => {
    const root = mountRef.current;
    if (!root) return;

    let instance: CodeEditorInstance | null = null;

    try {
      instance = createCodeEditor({
        root,
        defaultValue: initialValueRef.current,
        language: languageForPath(pathRef.current),
        readOnly: readOnlyRef.current,
        wordWrap: wrapRef.current,
        placeholder: placeholderRef.current,
        onChange: (value) => onChangeRef.current(value),
        onCursorChange: (position) => onCursorChangeRef.current?.(position),
        onPositionChange: (position) => onPositionChangeRef.current?.(position),
      });
    } catch (thrown) {
      // 与 Markdown 内核不同，CodeMirror 是**同步**创建的：没有「创建到一半卸载」
      // 那个窗口期，所以这里不需要 disposed 检查就能正确报错。
      onErrorRef.current?.(thrown instanceof Error ? thrown : new Error(String(thrown)));
      return;
    }

    instanceRef.current = instance;

    const handle: EditorHandle = {
      getValue: () => instanceRef.current?.getValue() ?? '',
      setValue: (text) => instanceRef.current?.setValue(text),
      focus: () => instanceRef.current?.focus(),
      toHtml: async () => instanceRef.current?.toHtml() ?? null,
      // 往返保真自检（P6 第 29 项）：代码内核「源码即文档」，保存写回的就是
      // 原文本身 —— 往返是恒等
      roundTrip: async (text) => text,
      search: {
        // 转一层而不是直接把实例的 search 交出去：卸载后实例为 null，
        // 而外壳可能还握着这个 handle（React 的清理顺序不保证）。
        // 转一层让「已销毁」表现为「什么都不做」，而不是抛一个空引用错误。
        find: (options) => instanceRef.current?.search.find(options) ?? 0,
        reveal: (index) => instanceRef.current?.search.reveal(index),
        replace: (index, replacement) => instanceRef.current?.search.replace(index, replacement) ?? 0,
        replaceAll: (replacement) => instanceRef.current?.search.replaceAll(replacement) ?? 0,
        clear: () => instanceRef.current?.search.clear(),
      },
      goToOutlineItem: ({ level, text }) => instanceRef.current?.goToHeading(level, text) ?? false,
      revealSourceLocation: (location) => instanceRef.current?.revealSourceLocation(location) ?? false,
      // 与上面的 `search` 同一层转发：卸载后实例为 null，而外壳可能还握着
      // 这个 handle（React 的清理顺序不保证）。`getPosition` 返回 null 表示
      // 「此刻读不到」，外壳据此保留上一次的值，而不是覆盖成一个假的 0。
      getPosition: () => instanceRef.current?.getPosition() ?? null,
      setPosition: (position) => instanceRef.current?.setPosition(position),
    };
    onReadyRef.current(handle);

    return () => {
      instanceRef.current = null;
      instance?.destroy();
    };
    // 空依赖：内核只在挂载时创建一次，外部内容变更一律走 handle.setValue
  }, []);

  // 软换行是「视图偏好」而不是「文档属性」：切换它不该重建内核，
  // 否则光标位置与撤销栈会被重置 —— 用户只是想让长行折一下。
  useEffect(() => {
    if (wordWrap === undefined) return;
    instanceRef.current?.setWordWrap(wordWrap);
  }, [wordWrap]);

  return <div ref={mountRef} className="qm-code" />;
}

export default CodeEditor;
