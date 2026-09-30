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
  onPositionChange,
  onSelectionChange,
  onSlashChange,
  onTableChange,
  onCodeBlockChange,
  resolveImageUrl,
  onImagePaste,
  onImageClick,
  onDiagramOpen,
  onError,
  readOnly,
  placeholder,
  codeLineNumbers = true,
}: EditorMountProps) {
  const mountRef = useRef<HTMLDivElement | null>(null);

  // 用 ref 转发回调。这些回调在每次 React 渲染时都是新函数，
  // 若直接进 effect 依赖，内核会被反复重建（每敲一个字重建一次编辑器）。
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onCursorChangeRef = useRef(onCursorChange);
  onCursorChangeRef.current = onCursorChange;
  const onPositionChangeRef = useRef(onPositionChange);
  onPositionChangeRef.current = onPositionChange;
  const onSelectionChangeRef = useRef(onSelectionChange);
  onSelectionChangeRef.current = onSelectionChange;
  const onSlashChangeRef = useRef(onSlashChange);
  onSlashChangeRef.current = onSlashChange;
  const onTableChangeRef = useRef(onTableChange);
  onTableChangeRef.current = onTableChange;
  const onCodeBlockChangeRef = useRef(onCodeBlockChange);
  onCodeBlockChangeRef.current = onCodeBlockChange;
  const resolveImageUrlRef = useRef(resolveImageUrl);
  resolveImageUrlRef.current = resolveImageUrl;
  const onImagePasteRef = useRef(onImagePaste);
  onImagePasteRef.current = onImagePaste;
  const onImageClickRef = useRef(onImageClick);
  onImageClickRef.current = onImageClick;
  const onDiagramOpenRef = useRef(onDiagramOpen);
  onDiagramOpenRef.current = onDiagramOpen;
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
          onPositionChange: (position) => onPositionChangeRef.current?.(position),
          onSelectionChange: (selection) => onSelectionChangeRef.current?.(selection),
          onSlashChange: (state) => onSlashChangeRef.current?.(state),
          onTableChange: (state) => onTableChangeRef.current?.(state),
          onCodeBlockChange: (state) => onCodeBlockChangeRef.current?.(state),
          // 这两个都**必须**转发（而不是直接传 `resolveImageUrl` / `onImagePaste`）：
          // 内核的 NodeView 与 handlePaste 在创建时就把它们捕获了，直接传的话
          // 之后 React 重渲染产生的新函数永远不会被内核看到 ——
          // 而这两个回调恰恰会变（文档路径变了、provider 换了）。
          resolveImageUrl: async (src) => (await resolveImageUrlRef.current?.(src)) ?? null,
          onImagePaste: async (request) => (await onImagePasteRef.current?.(request)) ?? null,
          // 点击不必转发成新函数：它没有返回值，也不会被内核异步持有 ——
          // 但内核**只在构造 NodeView 时**读一次这个属性，所以仍然要经 ref，
          // 否则切标签页之后点开的是旧文档的灯箱
          onImageClick: (image) => onImageClickRef.current?.(image),
          // 与 onImageClick 同属「内核只在构造 NodeView 时读一次」的那类回调
          onDiagramOpen: (diagram) => onDiagramOpenRef.current?.(diagram),
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
          table: created.table,
          codeBlock: created.codeBlock,
          goToOutlineItem: ({ level, text }) => created.goToHeading(level, text),
          getPosition: () => created.getPosition(),
          setPosition: (position) => created.setPosition(position),
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

  // 行号的显隐只是一个类名：行号槽由 NodeView 建好之后不再变动，
  // 关掉它只是让 CSS 不画。所以这个偏好**不需要**进上面那个 effect 的依赖表 ——
  // 改它不会重建内核，光标、滚动位置、撤销栈都原样保留
  return <div ref={mountRef} className={codeLineNumbers ? 'qm-editor' : 'qm-editor qm-no-line-numbers'} />;
}

export default MarkdownEditor;
