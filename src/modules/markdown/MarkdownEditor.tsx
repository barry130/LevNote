/**
 * Markdown 内核的 React 挂载组件。
 *
 * 这一层只做三件事：挂载点、生命周期、回调转发。内核的一切都在 `setup.ts` 里，
 * 这里不认识 Milkdown，只认识 `EditorMountProps`。
 */

import { useEffect, useRef } from 'react';

import type { EditorHandle, EditorMountProps } from '../../core/editor/types.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';
import './markdown.css';

function MarkdownEditor({
  initialValue,
  onChange,
  onReady,
  onCursorChange,
  onSelectionChange,
  onError,
  readOnly,
  placeholder,
}: EditorMountProps) {
  const mountRef = useRef<HTMLDivElement | null>(null);

  // 用 ref 转发回调。这些回调在每次 React 渲染时都是新函数，
  // 若直接进 effect 依赖，内核会被反复重建（每敲一个字重建一次编辑器）。
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onCursorChangeRef = useRef(onCursorChange);
  onCursorChangeRef.current = onCursorChange;
  const onSelectionChangeRef = useRef(onSelectionChange);
  onSelectionChangeRef.current = onSelectionChange;
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;

  const initialValueRef = useRef(initialValue);
  const readOnlyRef = useRef(readOnly);
  const placeholderRef = useRef(placeholder);

  useEffect(() => {
    const root = mountRef.current;
    if (!root) return;

    let disposed = false;
    let instance: MarkdownEditorInstance | null = null;

    void (async () => {
      try {
        const created = await createMarkdownEditor({
          root,
          defaultValue: initialValueRef.current,
          readOnly: readOnlyRef.current,
          placeholder: placeholderRef.current,
          onChange: (markdown) => onChangeRef.current(markdown),
          onCursorChange: (position) => onCursorChangeRef.current?.(position),
          onSelectionChange: (selection) => onSelectionChangeRef.current?.(selection),
        });

        // 内核是异步创建的：还没创建完就卸载了（切文件、StrictMode 双调用），
        // 必须立刻销毁，否则会留下一个挂在已卸载 DOM 上的编辑器实例
        if (disposed) {
          await created.destroy();
          return;
        }
        instance = created;

        const handle: EditorHandle = {
          getValue: () => created.getMarkdown(),
          setValue: (text) => created.setMarkdown(text),
          focus: () => created.focus(),
          // 语义 HTML 片段。Milkdown 能给出结构，所以这里返回字符串而不是 null
          toHtml: async () => created.toHtml(),
          search: created.search,
          formatting: created.formatting,
          goToOutlineItem: ({ level, text }) => created.goToHeading(level, text),
        };
        onReadyRef.current(handle);
      } catch (thrown) {
        if (disposed) return;
        onErrorRef.current?.(thrown instanceof Error ? thrown : new Error(String(thrown)));
      }
    })();

    return () => {
      disposed = true;
      if (instance) void instance.destroy();
    };
    // 空依赖：内核只在挂载时创建一次。
    // 外部内容变更一律走 handle.setValue，改这个组件的 key 才会重建。
  }, []);

  return <div ref={mountRef} className="qm-editor" />;
}

export default MarkdownEditor;
