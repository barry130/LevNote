/**
 * 单文档面板：告示条 + 编辑器 + 状态栏。
 *
 * ## 它为什么自己订阅 store，而不是从外壳收一堆 props
 *
 * 这一层要的东西全部来自**一份文档**的状态（脏、冲突、编码、行尾、光标、统计），
 * 把它们提到外壳再传下来，等于让外壳同时持有「工作区状态」和「活动文档状态」
 * 两套东西，而后者在切标签页时还要跟着换 —— 那是外壳最容易写错的地方。
 *
 * 反过来，让面板自己订阅「我这份文档」之后：
 * - 外壳完全不需要知道活动文档的任何字段；
 * - 切标签页就是换一个 `store`，其余一切自动正确；
 * - 面板的每一行代码都只依赖一个 store，不依赖它现在是第几个标签页。
 *
 * 唯一的对外输出是 `onHandle`：外壳持有编辑器命令面来执行导出、查找这类命令。
 */

import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';

import type { BomKind, LineEndingName } from '../core/document/types.js';
import type { ConflictChoice, DocumentStore } from '../core/document/store.js';
import type { CursorPosition, EditorAdapter, EditorHandle } from '../core/editor/types.js';
import { resolveAdapter } from '../core/editor/registry.js';
import { analyze, formatStatistics } from '../core/statistics.js';
import { EDITOR_FALLBACK } from '../modules/index.js';

const ENCODINGS: Array<{ label: string; encoding: string; bom: BomKind }> = [
  { label: 'UTF-8', encoding: 'UTF-8', bom: 'none' },
  { label: 'UTF-8（含 BOM）', encoding: 'UTF-8', bom: 'utf-8' },
  { label: 'GBK', encoding: 'GBK', bom: 'none' },
  { label: 'GB18030', encoding: 'GB18030', bom: 'none' },
  { label: 'Big5', encoding: 'Big5', bom: 'none' },
  { label: 'UTF-16 LE', encoding: 'UTF-16LE', bom: 'utf-16le' },
  { label: 'UTF-16 BE', encoding: 'UTF-16BE', bom: 'utf-16be' },
  { label: 'windows-1252', encoding: 'windows-1252', bom: 'none' },
];

const LINE_ENDINGS: Array<{ value: LineEndingName; label: string }> = [
  { value: 'lf', label: 'LF' },
  { value: 'crlf', label: 'CRLF' },
  { value: 'cr', label: 'CR' },
];

const STATUS_TEXT = {
  clean: '已保存',
  dirty: '有未保存的改动',
  saving: '处理中…',
  conflict: '磁盘上有更新的版本',
  error: '出错',
} as const;

function encodingKey(encoding: string, bom: BomKind): string {
  const hit = ENCODINGS.find((item) => item.encoding.toUpperCase() === encoding.toUpperCase() && item.bom === bom);
  if (hit) return `${hit.encoding}|${hit.bom}`;
  // 磁盘上是探测出来的编码（比如 chardetng 猜的 GB18030），不在下拉里也不该显示空白
  return `other|${encoding}|${bom}`;
}

export interface DocumentPaneProps {
  store: DocumentStore;
  wordWrap: boolean;
  /** 编辑器就绪时交出命令面；卸载或换文档时交回 null。 */
  onHandle: (handle: EditorHandle | null) => void;
  /** 解析出的内核变化时通知外壳（状态栏之外的地方也可能要用）。 */
  onAdapterChange?: (adapter: EditorAdapter | null) => void;
}

export function DocumentPane({ store, wordWrap, onHandle, onAdapterChange }: DocumentPaneProps) {
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const { doc, status, error, conflict, busy, notice } = state;

  const handleRef = useRef<EditorHandle | null>(null);
  /**
   * `handleRef` 里这个句柄属于哪一次会话。
   *
   * 必须有它：换文件时新的内核是**异步**创建的，而同步的 effect 会先跑。
   * 只凭 `handleRef.current` 存在就往上写，会写到一个已经被销毁的旧内核上。
   */
  const handleSessionRef = useRef<number | null>(null);
  const lastReportedRef = useRef<string | null>(null);
  const lastPushedRef = useRef<string | null>(null);

  const [cursor, setCursor] = useState<CursorPosition | null>(null);
  const [editorError, setEditorError] = useState<string | null>(null);
  /** 内核启动失败的会话号。用会话号而不是布尔值，换文件后自然重新尝试富文本内核。 */
  const [failedSession, setFailedSession] = useState<number | null>(null);

  const session = doc?.session ?? 0;
  const kernelFailed = failedSession !== null && failedSession === session;

  /**
   * 选内核。
   *
   * 依赖刻意只写 `session`：`canHandle` 关心的是文件开头（front matter 之类），
   * 与用户后续的编辑无关。若依赖写成 `doc.text`，每敲一个字都会重跑一次
   * 「注册表匹配 + front matter 探测」，没有任何意义。
   */
  const { adapter, downgraded } = useMemo(() => {
    const current = store.getSnapshot().doc;
    if (!current) return { adapter: null, downgraded: false };
    // 内核起不来时不把用户堵在「打不开」上：退回兜底内核，内容照常可读可改
    if (kernelFailed) return { adapter: EDITOR_FALLBACK, downgraded: false };
    const byContent = resolveAdapter(current.path, EDITOR_FALLBACK, current.text);
    const byName = resolveAdapter(current.path, EDITOR_FALLBACK);
    // 扩展名认得、内容却让首选内核退位 —— 这是一次有理由的降级，必须告诉用户
    return { adapter: byContent, downgraded: byContent.id !== byName.id };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, store, kernelFailed]);

  useEffect(() => {
    onAdapterChange?.(adapter);
  }, [adapter, onAdapterChange]);

  // 卸载（换标签页 / 关标签页）时把命令面交回去，避免外壳握着一个已销毁的内核
  useEffect(() => () => onHandle(null), [onHandle]);

  const EditorComponent = adapter?.Component ?? null;

  // ---- 内核 ↔ 外壳 的双向同步 -------------------------------------------

  const handleEditorChange = useCallback(
    (value: string) => {
      lastReportedRef.current = value;
      store.updateText(value);
    },
    [store],
  );

  const handleEditorReady = useCallback(
    (handle: EditorHandle) => {
      handleRef.current = handle;
      handleSessionRef.current = session;
      // 内核刚才是用 `doc.text` 挂载的，所以两边已经一致，不需要再推一次。
      // 同步这两个基准，是为了让下面那个 effect 不会在刚挂载时就误判成外部变更。
      const text = store.getSnapshot().doc?.text ?? '';
      lastReportedRef.current = text;
      lastPushedRef.current = text;
      onHandle(handle);
    },
    [session, store, onHandle],
  );

  useEffect(() => {
    if (!doc) return;
    // 新内核还没就绪（或者启动失败了），此时 handleRef 里要么是空的、
    // 要么指向上一份文档那个已经被销毁的实例，都不能碰
    if (handleSessionRef.current !== session) return;
    const handle = handleRef.current;
    if (!handle) return;
    // 内核自己报的：外壳这边已经把内容记下了，不要再推回去（回声）
    if (doc.text === lastReportedRef.current) return;
    // 外壳自己刚推过的：不要推第二遍
    if (doc.text === lastPushedRef.current) return;
    // 走到这里说明 doc.text 是被**外部**改的：重新加载、冲突解决、或换文件
    handle.setValue(doc.text);
    lastPushedRef.current = doc.text;
  }, [doc, session]);

  // 光标位置随文档切换重置，避免新文档还显示上一份的行列
  useEffect(() => {
    setCursor(null);
  }, [session]);

  const handleEditorError = useCallback(
    (thrown: Error) => {
      setEditorError(thrown.message);
      setFailedSession(store.getSnapshot().doc?.session ?? null);
    },
    [store],
  );

  // 只依赖文本本身：光标移动、脏标记翻转之类不该触发重新统计
  const docText = doc?.text;
  const stats = useMemo(() => (docText === undefined ? null : analyze(docText)), [docText]);

  const activeEncodingKey = doc ? encodingKey(doc.encoding, doc.bom) : '';

  return (
    <>
      {conflict && (
        <div className="banner warn">
          <strong>磁盘上的文件已被其他程序修改。</strong>
          <span className="grow">{conflict.message}</span>
          <button type="button" onClick={() => void store.resolveConflict('reload')} disabled={busy}>
            用磁盘版本覆盖我的
          </button>
          <button type="button" onClick={() => void store.resolveConflict('save-copy')} disabled={busy}>
            另存为副本
          </button>
          <button type="button" onClick={() => void store.resolveConflict('overwrite')} disabled={busy}>
            用我的覆盖磁盘
          </button>
        </div>
      )}

      {doc?.lossy && (
        <div className="banner info">
          <span className="grow">
            这个文件的编码是自动猜出来的，可能猜错。若下方出现乱码，请在状态栏手动指定编码。
          </span>
        </div>
      )}

      {downgraded && (
        <div className="banner info">
          <span className="grow">
            这个文件的开头是 YAML front matter。用富文本内核打开会把 <code>---</code> 当成分隔线、
            把 <code>键: 值</code> 当成正文，保存时元数据就没了。
            所以这个文件<strong>以「{adapter?.label ?? '纯文本'}」打开</strong>，内容不会有任何改动。
          </span>
        </div>
      )}

      {editorError && (
        <div className="banner error">
          <span className="grow">
            <strong>编辑器内核没能启动。</strong> {editorError}
            {kernelFailed ? '（已退回纯文本内核，内容不会丢）' : ''}
          </span>
          <button type="button" onClick={() => setEditorError(null)}>
            知道了
          </button>
        </div>
      )}

      {error && (
        <div className="banner error">
          <span className="grow">
            <strong>{error.code}</strong> · {error.message}
            {error.detail ? `（${error.detail}）` : ''}
          </span>
          <button type="button" onClick={store.clearError}>
            知道了
          </button>
        </div>
      )}

      <main className="editor-area">
        {doc && EditorComponent ? (
          <EditorComponent
            // 只在换文件时重建内核：撤销栈、光标、滚动位置都该跟着换。
            // 「另存为」换了路径但 session 不变，所以不会被重建 —— 否则光标会跳回开头。
            key={session}
            initialValue={doc.text}
            path={doc.path}
            readOnly={doc.readonly}
            wordWrap={wordWrap}
            placeholder="在这里写。支持 Markdown、代码 / 文本、思维导图（.mindmap）与 draw.io 图表（.drawio）四类文档。"
            onChange={handleEditorChange}
            onReady={handleEditorReady}
            onCursorChange={setCursor}
            onError={handleEditorError}
          />
        ) : doc ? (
          <div className="empty">
            <h2>找不到能打开这个文档的内核</h2>
            <p>这不应该发生 —— 纯文本兜底内核会处理任何文件。请把这份文件路径反馈给开发者。</p>
          </div>
        ) : null}
      </main>

      <footer className="statusbar">
        <span className={`status-pill ${status}`}>{STATUS_TEXT[status]}</span>

        {doc && (
          <>
            <span className="sep">│</span>
            <span className="kernel" title="当前使用的编辑器内核">
              {adapter?.label ?? '无内核'}
            </span>

            <span className="sep">│</span>
            <label>
              编码{' '}
              <select
                value={activeEncodingKey}
                onChange={(event) => {
                  const value = event.target.value;
                  if (value.startsWith('other|')) return;
                  const hit = ENCODINGS.find((item) => `${item.encoding}|${item.bom}` === value);
                  if (!hit) return;
                  // 编码与 BOM 必须一起改，否则会出现「GBK + UTF-8 BOM」这类不可能的组合
                  store.setFormat(hit.encoding, hit.bom);
                }}
              >
                {activeEncodingKey.startsWith('other|') && (
                  <option value={activeEncodingKey}>
                    {doc.encoding}
                    {doc.bom === 'none' ? '' : ' + BOM'}
                  </option>
                )}
                {ENCODINGS.map((item) => (
                  <option key={`${item.encoding}|${item.bom}`} value={`${item.encoding}|${item.bom}`}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>

            <label>
              行尾{' '}
              <select value={doc.lineEnding} onChange={(event) => store.setLineEnding(event.target.value as LineEndingName)}>
                {doc.lineEnding === 'mixed' && <option value="mixed">混合</option>}
                {LINE_ENDINGS.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>

            {/* 内核不报光标位置时整段隐藏，而不是显示一个假的 1:1 */}
            {cursor && (
              <>
                <span className="sep">│</span>
                <span>
                  第 {cursor.line} 行，第 {cursor.column} 列
                </span>
              </>
            )}
            <span className="sep">│</span>
            <span
              className="statistics"
              title="字数按 Microsoft Word 的口径：中日韩文字与标点各算一字，拉丁文按词计"
            >
              {stats ? formatStatistics(stats) : ''}
            </span>
            {doc.readonly && (
              <>
                <span className="sep">│</span>
                <span>只读文件</span>
              </>
            )}
          </>
        )}

        <span className="grow" />
        {notice && <span className="hint">{notice}</span>}
        <span className="sep">│</span>
        <span className="hint">Ctrl+S 保存 · Ctrl+P 命令面板</span>
      </footer>
    </>
  );
}

export type { ConflictChoice };
