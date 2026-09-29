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
import type { ConflictChoice, DocumentStore, ViewMode } from '../core/document/store.js';
import type {
  CursorPosition,
  EditorAdapter,
  EditorFormatting,
  EditorHandle,
  SelectionSnapshot,
} from '../core/editor/types.js';
import { resolveAdapter } from '../core/editor/registry.js';
import { analyze, formatStatistics } from '../core/statistics.js';
import { EDITOR_FALLBACK } from '../modules/index.js';
import { SelectionToolbar } from './SelectionToolbar.js';

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

/**
 * 面板向上汇报的「视图信息」。
 *
 * 外壳为什么需要它：工具栏上那个「源码 / 富文本」按钮既要判断**能不能切**，
 * 又要显示**现在在哪一边**。这两件事都住在文档状态里，而外壳**刻意不订阅
 * 文档状态** —— 它只订阅工作区，否则用户每敲一个字，整个外壳（侧栏、标签栏、
 * 工具栏）都要跟着重渲染。所以由面板把外壳真正需要的两个字段报上来，
 * 而不是让外壳去偷看某一份文档的内部状态。
 *
 * `adapter` 给的是**首选内核**（不考虑源码视图），这是刻意的：
 * 切到源码模式之后，当前内核变成了代码内核，而代码内核自己没有 `sourceView` ——
 * 若把「当前内核」报上去，按钮会在切过去的那一刻自己变成禁用，用户再也切不回来。
 */
export interface PaneViewInfo {
  /** 首选内核。null 表示这份文档没有解析出内核（正常情况下不会发生）。 */
  readonly adapter: EditorAdapter | null;
  readonly viewMode: ViewMode;
}

export interface DocumentPaneProps {
  store: DocumentStore;
  wordWrap: boolean;
  /** 编辑器就绪时交出命令面；卸载或换文档时交回 null。 */
  onHandle: (handle: EditorHandle | null) => void;
  /** 首选内核或视图模式变化时通知外壳；卸载时以 null 撤回。 */
  onViewInfoChange?: (info: PaneViewInfo | null) => void;
}

export function DocumentPane({ store, wordWrap, onHandle, onViewInfoChange }: DocumentPaneProps) {
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const { doc, status, error, conflict, busy, notice, viewMode } = state;

  const handleRef = useRef<EditorHandle | null>(null);
  /**
   * `handleRef` 里这个句柄属于**哪一次内核实例**，形如 `会话号:视图模式`。
   *
   * 必须有它：换文件、切视图都会重建内核，而新的内核是**异步**创建的，
   * 同步的 effect 会先跑。只凭 `handleRef.current` 存在就往上写，
   * 会写到一个已经被销毁的旧内核上。
   *
   * 把「视图模式」也算进这个标识，是因为切视图时**会话号不变** ——
   * 只用会话号的话，那一段异步窗口里 `handleRef` 会被误认为「有效」。
   */
  const handleKeyRef = useRef<string | null>(null);
  const lastReportedRef = useRef<string | null>(null);
  const lastPushedRef = useRef<string | null>(null);

  const [cursor, setCursor] = useState<CursorPosition | null>(null);
  const [editorError, setEditorError] = useState<string | null>(null);
  /** 内核启动失败的那一次实例标识。用标识而不是布尔值，换文件 / 切视图后自然重新尝试。 */
  const [failedKey, setFailedKey] = useState<string | null>(null);

  /**
   * 浮层工具条的两份输入：当前选区，和内核交出来的格式化命令面。
   *
   * 都放在 state 里而不是只留 ref：它们直接决定**画什么**。
   * 分开两个 state 是因为它们的生命周期不同 ——
   * 选区每次移动都换，命令面一个内核实例只有一个。
   */
  const [selection, setSelection] = useState<SelectionSnapshot | null>(null);
  const [formatting, setFormatting] = useState<EditorFormatting | null>(null);

  const session = doc?.session ?? 0;
  /**
   * 这次内核实例的标识。
   *
   * 加上视图模式换来一个「自然重试」：一次「富文本内核起不来」不该把用户
   * 永久锁在兜底内核上 —— 他至少该能切到源码视图，用代码内核把内容看全。
   */
  const kernelKey = `${session}:${viewMode}`;
  const kernelFailed = failedKey !== null && failedKey === kernelKey;

  /**
   * 选内核（**首选内核**，与视图模式无关）。
   *
   * 依赖刻意只写 `session`：`canHandle` 关心的是文件开头（front matter 之类），
   * 与用户后续的编辑无关。若依赖写成 `doc.text`，每敲一个字都会重跑一次
   * 「注册表匹配 + front matter 探测」，没有任何意义。
   *
   * 之所以把「首选内核」单独算出来而不是直接算「用哪个」：外壳需要首选内核
   * 才能回答「能不能切源码」，而切过去之后当前内核已经不是它了。
   */
  const { primary, downgraded } = useMemo(() => {
    const current = store.getSnapshot().doc;
    if (!current) return { primary: null, downgraded: false };
    const byContent = resolveAdapter(current.path, EDITOR_FALLBACK, current.text);
    const byName = resolveAdapter(current.path, EDITOR_FALLBACK);
    // 扩展名认得、内容却让首选内核退位 —— 这是一次有理由的降级，必须告诉用户
    return { primary: byContent, downgraded: byContent.id !== byName.id };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, store]);

  /**
   * 实际渲染用的内核。
   *
   * 优先级：**内核起不来 → 兜底** ＞ 源码模式 → 首选内核的 `sourceView` ＞ 首选内核。
   * 「起不来」排在最前，因为一个创建失败的实例渲染不出任何东西，
   * 而兜底内核至少保证内容可读可改 —— **打开文件绝不能失败**。
   */
  const adapter = useMemo(() => {
    if (primary === null) return null;
    if (kernelFailed) return EDITOR_FALLBACK;
    if (viewMode === 'source') return primary.sourceView ?? primary;
    return primary;
  }, [primary, viewMode, kernelFailed]);

  useEffect(() => {
    onViewInfoChange?.({ adapter: primary, viewMode });
  }, [primary, viewMode, onViewInfoChange]);

  /**
   * 卸载时把视图信息撤回。
   *
   * 少了这一步，切标签页的那一帧工具栏会拿着**上一个标签页**的内核信息去决定
   * 「能不能切源码」—— 表现为按钮闪一下（比如从 `.md` 切到 `.py` 时，
   * 按钮先亮着再变灰）。React 会把这次清理与下一个面板的首次上报放在同一批里，
   * 所以用户看不到中间态。
   */
  useEffect(() => () => onViewInfoChange?.(null), [onViewInfoChange]);

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
      handleKeyRef.current = kernelKey;
      // 错误属于**上一个创建失败的实例**。新实例就绪说明界面已经能用了，
      // 把过期错误留在横幅上，用户会以为「现在还是坏的」。
      setEditorError(null);
      // 内核刚才是用 `doc.text` 挂载的，所以两边已经一致，不需要再推一次。
      // 同步这两个基准，是为了让下面那个 effect 不会在刚挂载时就误判成外部变更。
      const text = store.getSnapshot().doc?.text ?? '';
      lastReportedRef.current = text;
      lastPushedRef.current = text;
      // 只在内核真的交出了格式化能力时才留下它。纯文本兜底内核没有这个字段 ——
      // 那时浮层根本不出现，而不是出现一个点了没反应的
      setFormatting(handle.formatting ?? null);
      onHandle(handle);
    },
    [kernelKey, store, onHandle],
  );

  useEffect(() => {
    if (!doc) return;
    // 新内核还没就绪（或者启动失败了），此时 handleRef 里要么是空的、
    // 要么指向上一份文档 / 上一次视图那个已经被销毁的实例，都不能碰
    if (handleKeyRef.current !== kernelKey) return;
    const handle = handleRef.current;
    if (!handle) return;
    // 内核自己报的：外壳这边已经把内容记下了，不要再推回去（回声）
    if (doc.text === lastReportedRef.current) return;
    // 外壳自己刚推过的：不要推第二遍
    if (doc.text === lastPushedRef.current) return;
    // 走到这里说明 doc.text 是被**外部**改的：重新加载、冲突解决、或换文件
    handle.setValue(doc.text);
    lastPushedRef.current = doc.text;
  }, [doc, session, kernelKey]);

  // 光标位置随文档切换重置，避免新文档还显示上一份的行列
  useEffect(() => {
    setCursor(null);
  }, [session]);

  /**
   * 换一次内核实例就清掉失败标记 —— 也就是**每次显式切视图都重试目标内核**。
   *
   * 少了这一步会留下一个永久失败标记：用户切到源码视图、再切回富文本，
   * 看到的仍然是兜底内核，而且**没有任何解释**（错误横幅属于上一个实例，
   * 早被撤掉了）。那是这个功能最糟的失败形态 —— 界面不对，还不说为什么。
   * 重试一次的成本是一次内核创建，而用户已经明确表达了「我要看那一边」。
   */
  useEffect(() => {
    setFailedKey(null);
    // 换内核实例 = 换了一份文档或换了一边视图，上一个实例报上来的选区与命令面
    // 都已经失效。不清的话，切标签页的那一帧会拿旧选区画出一个位置正确的浮层 ——
    // 位置对了反而更糟，用户会以为它属于新文档
    setSelection(null);
    setFormatting(null);
  }, [kernelKey]);

  const handleEditorError = useCallback(
    (thrown: Error) => {
      setEditorError(thrown.message);
      // 记的是**这一次实例**的标识，而不是「当下 store 里的 session」——
      // 后者在切视图之后会指向另一个内核，把失败标记盖到它头上。
      // （已经卸载的实例不会报错：`MarkdownEditor` 的 effect 里有 `disposed` 检查。）
      setFailedKey(kernelKey);
    },
    [kernelKey],
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
            所以这个文件<strong>以「{primary?.label ?? '纯文本'}」打开</strong>，内容不会有任何改动。
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
            // 换文件、换视图都要重建内核：撤销栈、光标、滚动位置都该跟着换。
            // 「另存为」换了路径但 session 与视图都不变，所以不会被重建 ——
            // 否则用户刚改完按 Ctrl+Shift+S，光标会跳回开头。
            key={kernelKey}
            initialValue={doc.text}
            path={doc.path}
            readOnly={doc.readonly}
            wordWrap={wordWrap}
            placeholder="在这里写。支持 Markdown、代码 / 文本、思维导图（.mindmap）与 draw.io 图表（.drawio）四类文档。"
            onChange={handleEditorChange}
            onReady={handleEditorReady}
            onCursorChange={setCursor}
            onSelectionChange={setSelection}
            onError={handleEditorError}
          />
        ) : doc ? (
          <div className="empty">
            <h2>找不到能打开这个文档的内核</h2>
            <p>这不应该发生 —— 纯文本兜底内核会处理任何文件。请把这份文件路径反馈给开发者。</p>
          </div>
        ) : null}

        {/* 浮层是编辑器区域的兄弟节点，不是子节点 —— 内核挂载时会替换掉自己那层容器的内容，
            放进同一层会被它抹掉。`position: fixed` 让它脱离 flex 布局，不会撑开高度 */}
        {selection !== null && formatting !== null && (
          <SelectionToolbar snapshot={selection} formatting={formatting} />
        )}
      </main>

      <footer className="statusbar">
        <span className={`status-pill ${status}`}>{STATUS_TEXT[status]}</span>

        {doc && (
          <>
            <span className="sep">│</span>
            <span className="kernel" title="当前使用的编辑器内核">
              {adapter?.label ?? '无内核'}
              {/* 源码模式下说明「这是谁的源码」：否则一份 .md 的状态栏写着
                  「代码 / 文本」，用户会以为打开的文件不对 */}
              {viewMode === 'source' && primary?.sourceView ? ` · ${primary.label} 源码` : ''}
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
