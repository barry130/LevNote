/**
 * 内置命令清单。
 *
 * 这里是**能力的唯一清单**：工具栏按钮、命令面板、快捷键都从这一份派生。
 * 因此不存在「加了按钮忘了加快捷键」或「面板里搜不到这个功能」的可能。
 *
 * 命名约定：`域.动作`。域只用于面板分组与将来的快捷键自定义界面。
 */

import type { CommandRegistry } from './registry.js';
import type { Command, CommandContext } from './types.js';
import type { BlockFormat, InlineFormat } from '../editor/types.js';
import { ENCODINGS, encodingSlug, encodingLabel } from '../encodings.js';
import { describeRoundTrip, formatRoundTripReport } from '../fidelity.js';
import { t } from '../i18n/index.js';
import { extractToReference, flattenReference } from '../document/block-ops.js';
import { asAppError } from '../document/types.js';
import { isDesktop, printHtml } from '../platform/index.js';
import { basename, dirname, joinPath } from '../path.js';
import { buildSampleDocument, SAMPLE_DOCUMENT_NAME } from '../sample-document.js';

function activeDocument(ctx: CommandContext) {
  return ctx.workspace.activeStore?.getSnapshot().doc ?? null;
}

function hasDocument(ctx: CommandContext): boolean {
  return activeDocument(ctx) !== null;
}

interface ExportInput {
  readonly path: string;
  readonly title: string;
  readonly bodyHtml: string | null;
  readonly fallbackText: string;
}

/**
 * 两种导出（自包含 HTML / 打印 PDF）共用的输入组装。
 *
 * 组装只做一件事：问内核要语义 HTML（要不到就给原文）。渲染在导出模块里，
 * 出口在各自的动作里 —— 两边共用同一份 `buildSelfContainedHtml` 结果，
 * 「导出的 HTML」和「打印出来的 PDF」永远不会长得不一样。
 */
async function composeExportInput(ctx: CommandContext): Promise<ExportInput | null> {
  const doc = activeDocument(ctx);
  if (!doc) return null;

  const handle = ctx.ui.editor();

  // toHtml() 返回 null 是**合法结果**（纯文本内核就是如此），不是错误。
  // 抛错也按 null 处理：导出成纯文本总比导不出来强。
  let bodyHtml: string | null;
  try {
    bodyHtml = handle ? await handle.toHtml() : null;
  } catch {
    bodyHtml = null;
  }

  return {
    path: doc.path,
    title: doc.displayName.replace(/\.[^.]+$/, ''),
    bodyHtml,
    fallbackText: doc.text,
  };
}

async function runExport(ctx: CommandContext): Promise<void> {
  const input = await composeExportInput(ctx);
  if (input === null) return;

  /**
   * 导出模块**用在时才加载**（P3 第 18 项）：它拖着 hljs 与整套 remark
   * （为了公式与代码块的 HTML 化），而导出是一个低频动作 —— 让主包为了
   * 它多解析三百 KB 不划算。动态 import 之余没有任何行为变化。
   */
  const { exportHtml } = await import('../export/export-document.js');
  const result = await exportHtml({ ...input, provider: ctx.provider });

  if (result.status === 'saved') await ctx.ui.notify(result.message, t('core.cmd.exportDone'));
  else if (result.status === 'failed') await ctx.ui.notify(result.message, t('core.cmd.exportFailed'));
}

async function runPrintExport(ctx: CommandContext): Promise<void> {
  const input = await composeExportInput(ctx);
  if (input === null) return;

  // 与 HTML 导出共用同一个按需加载理由：渲染模块不进主包（P3 第 18 项）
  const { printDocument } = await import('../export/print-document.js');
  const result = await printDocument({ ...input, provider: ctx.provider, printHtml });

  if (result.status === 'opened') await ctx.ui.notify(result.message, t('core.cmd.printTitle'));
  else await ctx.ui.notify(result.message, t('core.cmd.exportFailed'));
}

/**
 * 往返保真自检（P6 第 29 项，ADR 0033）：拿磁盘上的原文走一遍内核自己的
 * 「解析 → 序列化」，与原文逐行比对，如实报告「不改一个字保存会改写什么」。
 *
 * 只读不写 —— 它是给用户知情权的一个诊断动作，不是「修复」：
 * 报告里的差异是内核**故意保留**的规范化（表格重排之类），改不改由人决定。
 */
async function runRoundTripCheck(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  if (doc === null) return;
  const handle = ctx.ui.editor();
  if (handle?.roundTrip === undefined) return;

  // 比对的基准是**磁盘那份**（savedText），不是当前编辑内容 ——
  // 用户改过的部分是「他的改动」，不属于「内核会改写什么」这个问题
  let roundTripped: string;
  try {
    roundTripped = await handle.roundTrip(doc.savedText);
  } catch (thrown) {
    const reason = thrown instanceof Error ? thrown.message : String(thrown);
    await ctx.ui.notify(t('core.cmd.roundTripFailed', { reason }), t('core.cmd.roundTripTitle'));
    return;
  }

  const report = describeRoundTrip(doc.savedText, roundTripped);
  await ctx.ui.notify(formatRoundTripReport(report), t('core.cmd.roundTripTitle'));
}

/**
 * 「提取为引用」的编排：把内核的文档操作与 provider 的文件操作接在一起。
 *
 * 依赖逐个注入而不是让 `block-ops.ts` 反过来 import 这里的 ctx ——
 * 这样编排逻辑可以被假依赖单测钉住（见 `block-ops.test.ts`）。
 */
async function runExtractReference(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  const transform = ctx.ui.editor()?.blockTransform;
  if (doc === null || transform === undefined) return;
  await extractToReference({
    docPath: doc.path,
    selectedMarkdown: () => transform.selectedMarkdown(),
    replaceSelectionWith: (markdown) => transform.replaceSelectionWith(markdown),
    enclosingFence: () => transform.enclosingFence(),
    replaceEnclosingFenceWith: (markdown) => transform.replaceEnclosingFenceWith(markdown),
    readTextFile: async (path) => {
      try {
        return await ctx.provider.readTextFile(path);
      } catch {
        return null;
      }
    },
    writeTextFile: (path, text) => ctx.provider.writeTextFile({ path, text, create: true }).then(() => undefined),
    notify: (message, title) => ctx.ui.notify(message, title),
  });
}

/** 「拉平引用」的编排。与 `runExtractReference` 同一条分工。 */
async function runInlineReference(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  const transform = ctx.ui.editor()?.blockTransform;
  if (doc === null || transform === undefined) return;
  await flattenReference({
    docPath: doc.path,
    selectedMarkdown: () => transform.selectedMarkdown(),
    replaceSelectionWith: (markdown) => transform.replaceSelectionWith(markdown),
    enclosingFence: () => transform.enclosingFence(),
    replaceEnclosingFenceWith: (markdown) => transform.replaceEnclosingFenceWith(markdown),
    readTextFile: async (path) => {
      try {
        return await ctx.provider.readTextFile(path);
      } catch {
        return null;
      }
    },
    writeTextFile: (path, text) => ctx.provider.writeTextFile({ path, text, create: true }).then(() => undefined),
    notify: (message, title) => ctx.ui.notify(message, title),
  });
}

/**
 * 「格式」与「段落」两组命令是**机械生成的**（每条只差一个格式名/块名），
 * 手写二十遍同样的 enabled 与 run 只会藏起共性。数组用显式类型标注 ——
 * `as const` 的异构数组解构不出可选的 `level`，标了类型反而直白；
 * id 刻意是**字面量联合**而不是 string：`t()` 的参数是精确的 key 联合，
 * 模板字面量在字面量联合上分发后恰好对得上（写错 key 是编译错误）。
 */
const INLINE_FORMATS = [
  // 命令 id 用「加粗」，内核的 InlineFormat 用 strong —— 两套词各归各的用户
  { id: 'bold', inline: 'strong' },
  { id: 'italic', inline: 'emphasis' },
  { id: 'strikethrough', inline: 'strikethrough' },
  { id: 'inlineCode', inline: 'inlineCode' },
] as const satisfies readonly { id: string; inline: InlineFormat }[];

const INLINE_FORMAT_COMMANDS: readonly Command[] = INLINE_FORMATS.map(({ id, inline }) => ({
  id: `format.${id}`,
  title: t(`command.format.${id}.title`),
  category: t('command.category.format'),
  keywords: [id, 'geshi'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.toggleInline(inline);
  },
}));

type ParagraphBlockId =
  | 'paragraph.plainText'
  | 'paragraph.heading1'
  | 'paragraph.heading2'
  | 'paragraph.heading3'
  | 'paragraph.heading4'
  | 'paragraph.heading5'
  | 'paragraph.heading6'
  | 'paragraph.blockquote'
  | 'paragraph.bulletList'
  | 'paragraph.orderedList'
  | 'paragraph.codeBlock';

const PARAGRAPH_BLOCK_COMMANDS: readonly Command[] = (
  [
    { id: 'paragraph.plainText', block: 'paragraph', level: undefined },
    { id: 'paragraph.heading1', block: 'heading', level: 1 },
    { id: 'paragraph.heading2', block: 'heading', level: 2 },
    { id: 'paragraph.heading3', block: 'heading', level: 3 },
    { id: 'paragraph.heading4', block: 'heading', level: 4 },
    { id: 'paragraph.heading5', block: 'heading', level: 5 },
    { id: 'paragraph.heading6', block: 'heading', level: 6 },
    { id: 'paragraph.blockquote', block: 'blockquote', level: undefined },
    { id: 'paragraph.bulletList', block: 'bulletList', level: undefined },
    { id: 'paragraph.orderedList', block: 'orderedList', level: undefined },
    { id: 'paragraph.codeBlock', block: 'codeBlock', level: undefined },
  ] as readonly { id: ParagraphBlockId; block: BlockFormat; level: number | undefined }[]
).map(({ id, block, level }) => ({
  id,
  title: t(`command.${id}.title`),
  category: t('command.category.paragraph'),
  keywords: [id, 'duanluo'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.toggleBlock(block, level);
  },
}));

const PARAGRAPH_INSERT_COMMANDS: readonly Command[] = (
  ['table', 'mathBlock', 'diagram', 'drawio', 'hr'] as const
).map((insertable) => ({
  id: `paragraph.${insertable}`,
  title: t(`command.paragraph.${insertable}.title`),
  category: t('command.category.paragraph'),
  keywords: [insertable, 'charu', 'duanluo'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.insertBlock(insertable);
  },
}));

const THEME_COMMANDS: readonly Command[] = (['system', 'paper', 'ink'] as const).map((preference) => ({
  id: `theme.${preference}`,
  title: t(`command.theme.${preference}.title`),
  category: t('command.category.theme'),
  keywords: [preference, 'theme', 'zhuti'],
  run: (ctx: CommandContext) => {
    ctx.ui.setTheme(preference);
  },
}));

/**
 * 「编码」菜单的命令：一份编码一个「转为 …」。
 *
 * 标题直接用 `core/encodings.ts` 里的 `label` —— 那份数据本身就是界面文案
 * （状态栏的下拉显示的也是它），这里不再造第二个名字。可勾选状态由外壳
 * 按文档当前的编码判断（`menuChecked`），命令本身无状态。
 */
const ENCODING_COMMANDS: readonly Command[] = ENCODINGS.map((option) => ({
  id: `encoding.set.${encodingSlug(option.encoding, option.bom)}`,
  title: encodingLabel(option),
  category: t('command.category.encoding'),
  keywords: [option.encoding, 'bianma', 'encoding', '编码'],
  enabled: (ctx: CommandContext) => {
    const doc = activeDocument(ctx);
    return doc !== null && !doc.readonly;
  },
  run: (ctx: CommandContext) => {
    ctx.ui.setEncoding(option.encoding, option.bom);
  },
}));

/** 「编码」菜单要的命令 id 清单 —— 与命令定义同生共死，菜单不自己拼 id。 */
export const ENCODING_COMMAND_IDS: readonly string[] = ENCODING_COMMANDS.map((command) => command.id);

export const BUILT_IN_COMMANDS: readonly Command[] = [
  // ---- 文件 -------------------------------------------------------------
  {
    id: 'file.open',
    title: t('command.file.open.title'),
    category: t('command.category.file'),
    keybinding: 'Mod+O',
    keywords: ['open', 'dakaishejian'],
    run: (ctx) => {
      void ctx.workspace.openViaDialog();
    },
  },
  {
    id: 'file.openFolder',
    title: t('command.file.openFolder.title'),
    category: t('command.category.file'),
    keybinding: 'Mod+Shift+O',
    keywords: ['open folder', 'workspace', 'gongzuoqu'],
    run: (ctx) => {
      void ctx.workspace.openFolder();
    },
  },
  {
    /**
     * 新建文件：选位置、起名、落一个空文件、打开。
     *
     * 本产品**没有「无标题缓冲区」**——文档生命周期从磁盘上的一份文件开始
     * （打开/保存/监听/冲突检测全都以路径为主键），先造一个内存里的「新文档」
     * 再补一次另存为，等于在每条生命周期链路上都加一个「还没有路径」的状态。
     * 所以「新建」被实现成「直接在磁盘上起一个空文件」，语义反而更诚实。
     *
     * 绝不静默覆盖（硬约束 2）：另存为对话框允许指到一个已有文件上，
     * 新建撞上已有文件时明确说明并停手，而不是把它清空。
     */
    id: 'file.new',
    title: t('command.file.new.title'),
    category: t('command.category.file'),
    keywords: ['new file', 'create', 'xinjian'],
    run: (ctx) => {
      void (async () => {
        const path = await ctx.provider.pickSavePath(t('core.cmd.untitledName'));
        if (path === null) return;
        try {
          await ctx.provider.statFile(path);
          await ctx.ui.notify(t('core.cmd.newFileExists'), t('core.cmd.newFileTitle'));
          return;
        } catch {
          // stat 失败 = 文件不存在，正是「新建」想要的情形
        }
        try {
          await ctx.provider.writeTextFile({ path, text: '', create: true });
        } catch {
          await ctx.ui.notify(t('core.cmd.newFileFailed'), t('core.cmd.newFileTitle'));
          return;
        }
        await ctx.workspace.open(path);
      })();
    },
  },
  {
    id: 'file.save',
    title: t('command.file.save.title'),
    category: t('command.category.file'),
    keybinding: 'Mod+S',
    keywords: ['save', 'baocun'],
    enabled: (ctx) => {
      const snapshot = ctx.workspace.activeStore?.getSnapshot();
      if (snapshot === undefined || snapshot.doc === null) return false;
      if (snapshot.doc.readonly) return false;
      // 正在写盘时不再放行：连按 Ctrl+S 会排两次写，
      // 而第二次拿的 mtime 基准还是旧的，会误报一次「磁盘上的文件已被修改」
      return !snapshot.busy;
    },
    run: (ctx) => {
      void ctx.workspace.save();
    },
  },
  {
    id: 'file.saveAs',
    title: t('command.file.saveAs.title'),
    category: t('command.category.file'),
    keybinding: 'Mod+Shift+S',
    keywords: ['save as', 'lingcunwei'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.saveAs();
    },
  },
  {
    id: 'file.saveAll',
    title: t('command.file.saveAll.title'),
    category: t('command.category.file'),
    keywords: ['save all', 'baocun', '保存全部'],
    // 一个脏标签页都没有时禁用 —— 与 file.save 的「不忙时才放行」同一原则：
    // 显示为禁用告诉用户「现在没有要保存的东西」，而不是点了没反应
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.some((tab) => tab.dirty),
    run: (ctx) => {
      void ctx.workspace.saveAll();
    },
  },
  {
    id: 'file.reload',
    title: t('command.file.reload.title'),
    category: t('command.category.file'),
    keywords: ['reload', 'chongxinjiazai'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.reload();
    },
  },
  {
    id: 'file.closeTab',
    title: t('command.file.closeTab.title'),
    category: t('command.category.file'),
    keybinding: 'Mod+W',
    keywords: ['close', 'guanbi'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.closeActiveTab(),
  },
  {
    id: 'file.exportHtml',
    title: t('command.file.exportHtml.title'),
    category: t('command.category.export'),
    keybinding: 'Mod+Shift+E',
    keywords: ['export', 'html', 'daochu'],
    enabled: hasDocument,
    run: runExport,
  },
  {
    id: 'file.exportPdf',
    title: t('command.file.exportPdf.title'),
    category: t('command.category.export'),
    keywords: ['pdf', 'print', 'dayin', 'daochu'],
    // 浏览器预览没有打印子窗口，禁用比「点了报错」诚实；
    // printHtml 里还有一道明确拒绝，防的是别的入口直接调动作。
    enabled: (ctx) => hasDocument(ctx) && isDesktop(),
    run: runPrintExport,
  },

  // ---- 标签页 -----------------------------------------------------------
  {
    id: 'tab.next',
    title: t('command.tab.next.title'),
    category: t('command.category.tab'),
    keybinding: 'Ctrl+Tab',
    keywords: ['next tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(1),
  },
  {
    id: 'tab.prev',
    title: t('command.tab.prev.title'),
    category: t('command.category.tab'),
    keybinding: 'Ctrl+Shift+Tab',
    keywords: ['previous tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(-1),
  },

  // ---- 视图 -------------------------------------------------------------
  {
    id: 'view.toggleSidebar',
    title: t('command.view.toggleSidebar.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+B',
    keywords: ['sidebar', 'cebianlan'],
    run: (ctx) => ctx.ui.toggleSidebar(),
  },
  {
    /**
     * 活动栏的「文件浏览器」。
     *
     * 它没有快捷键，也不是 `view.toggleSidebar` 的别名 —— 两者要的是两种行为：
     * `Ctrl+B` 收起 / 放出侧栏（**保留**正在看的那一栏），
     * 而这个按钮的语义是「我要看文件树」（侧栏正显示检索时应当换过去）。
     *
     * 仍然走命令而不是直接调 UI：活动栏上五个入口的可用性全部来自
     * `CommandRegistry.enabled`，这里是那套约定的第六个。
     */
    id: 'view.filesView',
    title: t('command.view.filesView.title'),
    category: t('command.category.view'),
    keywords: ['files', 'explorer', 'wenjianliulanqi', '文件树'],
    run: (ctx) => ctx.ui.openFilesView(),
  },
  {
    id: 'view.toggleOutline',
    title: t('command.view.toggleOutline.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+Shift+L',
    keywords: ['outline', 'dagang'],
    run: (ctx) => ctx.ui.toggleOutline(),
  },
  {
    id: 'view.toggleSource',
    title: t('command.view.toggleSource.title'),
    category: t('command.category.view'),
    /**
     * 与 horseMD 的「源码模式切换」同一个键位。
     *
     * `Ctrl+/` 在这里是安全的：`isAllowedInEditable` 放行带修饰键的快捷键，
     * 而 `/` 本身不带修饰键时在编辑器里就是普通输入，不会被抢走。
     */
    keybinding: 'Mod+/',
    keywords: ['source', 'markdown', 'yuanma', '源码', 'markdown source'],
    // 只有首选内核声明了 `sourceView` 的文档才可用（目前是 Markdown）。
    // 用禁用而不是隐藏：隐藏会让用户以为这个功能不存在。
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.toggleSourceView(),
  },
  {
    id: 'edit.extractReference',
    title: t('command.edit.extractReference.title'),
    category: t('command.category.edit'),
    keywords: ['extract', 'reference', 'doc-block', 'tiqu', 'yinyong'],
    // 只有 Markdown 内核实现了 blockTransform（doc-block 是 Markdown 的语法）；
    // 其它内核禁用而不是隐藏 —— 与 view.toggleSource 同一条原则
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.blockTransform !== undefined,
    run: (ctx) => runExtractReference(ctx),
  },
  {
    id: 'edit.inlineReference',
    title: t('command.edit.inlineReference.title'),
    category: t('command.category.edit'),
    keywords: ['flatten', 'inline', 'doc-block', 'laping', 'zhankai'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.blockTransform !== undefined,
    run: (ctx) => runInlineReference(ctx),
  },
  /**
   * 撤销 / 重做（菜单入口）。
   *
   * **刻意没有默认快捷键**：编辑器内部的 Ctrl+Z / Ctrl+Shift+Z 由内核自己的
   * keymap 处理（ProseMirror 的 history 插件、CodeMirror 的 historyKeymap），
   * 外壳若再注册一个全局 Mod+Z，`isAllowedInEditable` 放行带修饰键的绑定，
   * 两次撤销会在同一按键上叠加执行。所以这两个命令只走菜单 / 面板入口；
   * 用户确实想给它们自定义全局键时，可以在设置页录制（后果自负）。
   */
  {
    id: 'edit.undo',
    title: t('command.edit.undo.title'),
    category: t('command.category.edit'),
    keywords: ['undo', 'chexiao'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.undo !== undefined,
    run: (ctx) => ctx.ui.editor()?.undo?.(),
  },
  {
    id: 'edit.redo',
    title: t('command.edit.redo.title'),
    category: t('command.category.edit'),
    keywords: ['redo', 'chongzuo'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.redo !== undefined,
    run: (ctx) => ctx.ui.editor()?.redo?.(),
  },
  {
    /**
     * 复制为 HTML。
     *
     * `toHtml()` 返回 null（纯文本内核）或抛错时退回**复制原文**，并把这次
     * 降级说出来（硬约束 2）—— 用户得知道剪贴板里拿到的不是格式化结果。
     */
    id: 'edit.copyHtml',
    title: t('command.edit.copyHtml.title'),
    category: t('command.category.edit'),
    keywords: ['copy html', 'fuzhi', '剪贴板'],
    enabled: hasDocument,
    run: (ctx) => {
      void (async () => {
        const doc = activeDocument(ctx);
        const handle = ctx.ui.editor();
        if (doc === null || handle === null) return;

        let html: string | null;
        try {
          html = await handle.toHtml();
        } catch {
          html = null;
        }

        try {
          await navigator.clipboard.writeText(html ?? doc.text);
        } catch {
          await ctx.ui.notify(t('core.cmd.clipCopyDenied'), t('command.edit.copyHtml.title'));
          return;
        }
        await ctx.ui.notify(
          html === null
            ? t('core.cmd.clipNoHtml')
            : t('core.cmd.clipHtmlOk'),
          t('command.edit.copyHtml.title'),
        );
      })();
    },
  },
  /**
   * 三档视图的直接入口。
   *
   * ## 为什么是 `Ctrl+1/2/3` 而不是 `Ctrl+/` 那种循环
   *
   * 循环是**有状态的**：想从富文本切到分屏，得先知道「现在在哪一档、
   * 还要按几下」。三档之后这个心智负担明显变大（两档时循环很自然，
   * 三档就开始猜）。数字键是「直接跳过去」，与 VSCode 的编辑器组切换同一个思路。
   *
   * `Ctrl+/` 保留：它在两档之间来回仍然是最高频的动作，不该被迫多按一个键。
   *
   * ## `enabled` 用的是同一个判据
   *
   * 源码与分屏都要求首选内核声明了 `sourceView`（目前是 Markdown），
   * 所以三个命令里有两个共用 `canToggleSourceView()` —— 名字说的是
   * 「能不能切到源码那一侧」，而分屏正是「源码那一侧 + 预览」。
   * 富文本那一档永远可用（兜底内核也能渲染），所以它没有 `enabled`。
   */
  {
    id: 'view.rich',
    title: t('command.view.rich.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+1',
    keywords: ['rich', 'wysiwyg', 'fuwenben', '预览', '所见即所得'],
    run: (ctx) => ctx.ui.setViewMode('rich'),
  },
  {
    id: 'view.source',
    title: t('command.view.source.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+2',
    keywords: ['source', 'yuanma', '源码', 'markdown'],
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.setViewMode('source'),
  },
  {
    id: 'view.split',
    title: t('command.view.split.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+3',
    keywords: ['split', 'side by side', 'bingshu', '分屏', '双栏', 'fenping'],
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.setViewMode('split'),
  },
  {
    id: 'view.cycleTheme',
    title: t('command.view.cycleTheme.title'),
    category: t('command.category.view'),
    keybinding: 'Mod+Shift+T',
    keywords: ['theme', 'zhuti', 'dark', 'light'],
    run: (ctx) => ctx.ui.cycleTheme(),
  },
  {
    id: 'view.toggleWordWrap',
    title: t('command.view.toggleWordWrap.title'),
    category: t('command.category.view'),
    keywords: ['word wrap', 'huanhang', '换行', '软换行'],
    run: (ctx) => ctx.ui.toggleWordWrap(),
  },
  {
    id: 'view.toggleCodeLineNumbers',
    title: t('command.view.toggleCodeLineNumbers.title'),
    category: t('command.category.view'),
    keywords: ['line numbers', 'hanghao', '行号'],
    run: (ctx) => ctx.ui.toggleCodeLineNumbers(),
  },
  {
    /**
     * 专注 / 打字机（Typora 的两个写作模式）。
     *
     * 与 `view.toggleWordWrap` 同属「设置层开关」：命令只翻偏好，
     * 哪些内核消费它由各内核决定（Markdown 两个都认；代码内核只认打字机；
     * 纯文本与导图忽略）。菜单项不按内核禁用 —— 与「行号」同一先例：
     * 偏好是全局的，当前文档用不上时切了也没害处。
     */
    id: 'view.toggleFocusMode',
    title: t('command.view.toggleFocusMode.title'),
    category: t('command.category.view'),
    keywords: ['focus', 'zhuanzhu', '专注', '沉浸'],
    run: (ctx) => ctx.ui.toggleFocusMode(),
  },
  {
    id: 'view.toggleTypewriterMode',
    title: t('command.view.toggleTypewriterMode.title'),
    category: t('command.category.view'),
    keywords: ['typewriter', 'daziji', '打字机'],
    run: (ctx) => ctx.ui.toggleTypewriterMode(),
  },
  {
    id: 'view.toggleFullscreen',
    title: t('command.view.toggleFullscreen.title'),
    category: t('command.category.view'),
    // F11 是全屏的通用键位；功能键在输入框里不产生文本，isAllowedInEditable 放行
    keybinding: 'F11',
    keywords: ['fullscreen', 'quanping', '全屏'],
    run: (ctx) => {
      void ctx.ui.toggleFullscreen();
    },
  },

  // ---- 导航 -------------------------------------------------------------
  {
    id: 'find.open',
    title: t('command.find.open.title'),
    category: t('command.category.navigate'),
    keybinding: 'Mod+F',
    keywords: ['find', 'search', 'chazhao'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.openFind(),
  },
  {
    id: 'search.inFolder',
    title: t('command.search.inFolder.title'),
    category: t('command.category.navigate'),
    /**
     * 与 VSCode 同一个键位。
     *
     * `Ctrl+Shift+F` 在这里是安全的：`isAllowedInEditable` 放行带修饰键的快捷键，
     * 而 `F` 不带修饰键时在编辑器里就是普通输入。
     */
    keybinding: 'Mod+Shift+F',
    keywords: ['search in folder', 'workspace search', 'grep', 'quanwen', 'sousuo', '全文检索', '在文件夹中查找'],
    // 没有文件夹就无处可查。用**禁用**而不是隐藏：隐藏会让用户以为功能不存在
    // （而它只是需要一个前置条件，命令面板里那行提示正好告诉他缺什么）
    enabled: (ctx) => ctx.workspace.getSnapshot().tree.root !== null,
    run: (ctx) => ctx.ui.openSearchFolder(),
  },
  {
    id: 'palette.open',
    title: t('command.palette.open.title'),
    category: t('command.category.navigate'),
    keybinding: 'Mod+P',
    keywords: ['command palette', 'minglingmianban'],
    run: (ctx) => ctx.ui.openPalette(),
  },
  {
    id: 'app.settings',
    title: t('command.app.settings.title'),
    category: t('command.category.app'),
    keybinding: 'Mod+,',
    keywords: ['settings', 'preferences', 'shezhi', 'pianhao'],
    run: (ctx) => ctx.ui.openSettings(),
  },

  // ---- 诊断 ---------------------------------------------------------------
  {
    id: 'diagnostics.roundTrip',
    title: t('command.diagnostics.roundTrip.title'),
    category: t('command.category.diagnostics'),
    keywords: ['roundtrip', 'fidelity', 'diagnostics', 'wangfan', 'baozhen'],
    // 内核没实现 roundTrip（导图内核）时禁用 —— 这是「没有这个能力」，
    // 不是「这份文档用不上」，但一致的呈现仍是禁用而非隐藏
    enabled: (ctx) => {
      const doc = activeDocument(ctx);
      return doc !== null && typeof ctx.ui.editor()?.roundTrip === 'function';
    },
    run: runRoundTripCheck,
  },
  // ---- 格式 ---------------------------------------------------------------
  // 四条都只是 `formatting` 契约的薄封装：能不能用问内核（Markdown 实现了它，
  // 导图/代码内核没有），格式切换本身由内核完成（含 USER_EDIT_META，与
  // 选中浮层工具条是同一条路径 —— 菜单只是多一个入口，不多一条语义）。
  ...INLINE_FORMAT_COMMANDS,

  // ---- 段落 ---------------------------------------------------------------
  // 块级格式走 toggleBlock（把**光标所在块**换过去，与 Typora 的段落菜单同一
  // 语义）；表格 / 数学块 / 图表 / 分隔线没有「所在块」可换，走 insertBlock
  // （斜杠菜单的同一入口，删 `/query` 那步对菜单场景无害 —— 不在触发态时它
  // 就等价于普通插入）。
  ...PARAGRAPH_BLOCK_COMMANDS,
  ...PARAGRAPH_INSERT_COMMANDS,

  // ---- 主题 ---------------------------------------------------------------
  // setTheme 与 cycleTheme 都是现成的 UiActions；三条直接入口给菜单用
  // （「我要暗黑」是两步循环里的第三步，循环键留着给快切）。
  ...THEME_COMMANDS,

  // ---- 编码 ---------------------------------------------------------------
  // 与状态栏下拉同一份候选清单（core/encodings.ts）；只改元数据，保存时生效
  ...ENCODING_COMMANDS,

  {
    id: 'app.about',
    title: t('command.app.about.title'),
    category: t('command.category.app'),
    keywords: ['about', 'version', 'guanyu', 'banben'],
    // 构建期注入的常量必须写成 `import.meta.env.X`（硬约束 14）——
    // dev server 不替换裸标识符，写错的表现是「点一下关于整个应用就没了」
    run: (ctx) => {
      void ctx.ui.notify(t('core.cmd.aboutBody', { version: String(import.meta.env.LEVNOTE_VERSION ?? '') }), t('command.app.about.title'));
    },
  },

  /**
   * 示例文档（B1）：往工作区根写入 `轻笺示例.md` 并打开。
   *
   * 与 `file.new` 同一条编排（stat → 写 → open），也守同一条红线：
   * **已存在就不再写** —— 用户可能已经改过这份示例，覆盖就是毁内容
   * （硬约束 2）。想重置就删掉文件再点一次，这与示例文档里写的话一致。
   */
  {
    id: 'help.sampleDocument',
    title: t('command.help.sampleDocument.title'),
    category: t('command.category.help'),
    keywords: ['sample', 'example', 'demo', 'shili', '示例', '导览'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tree.root !== null,
    run: (ctx) => {
      void (async () => {
        const root = ctx.workspace.getSnapshot().tree.root;
        if (root === null) return;
        const path = joinPath(root, SAMPLE_DOCUMENT_NAME);
        if (path === null) return;
        try {
          await ctx.provider.statFile(path);
        } catch {
          // stat 失败 = 还没有这份文件，写入它
          try {
            await ctx.provider.writeTextFile({ path, text: buildSampleDocument(), create: true });
          } catch (thrown) {
            const error = asAppError(thrown);
            await ctx.ui.notify(t('core.cmd.sampleCreateFailed', { reason: error.message }), t('core.cmd.sampleTitle'));
            return;
          }
        }
        await ctx.workspace.open(path);
      })();
    },
  },

  {
    id: 'ui.escape',
    title: t('command.ui.escape.title'),
    category: t('command.category.navigate'),
    keybinding: 'Escape',
    keywords: ['cancel', 'close'],
    run: (ctx) => {
      ctx.ui.closePalette();
      ctx.ui.closeFind();
      ctx.ui.closeSettings();
    },
  },
];

/** 「最近打开」进菜单的条数（持久化上界更大，见 core/recent.ts）。 */
export const RECENT_MENU_SLOTS = 10;

/**
 * 最近打开的文件（A5）→ 动态命令。
 *
 * 标题是**数据**：`文件名 — 所在目录`（不同目录里的同名文件要分得开），
 * 与编码命令用 `option.label` 当标题同一条先例。`dynamic: true` 让它们
 * 进菜单、不进命令面板与快捷键表 —— 那两处是能力清单，不是数据清单。
 *
 * id 带槽位号（`file.recent.0..9`）：列表顺序变了，命令**原位覆盖**，
 * 菜单项的位置不漂；尾部空出来的槽位由调用方注销。
 */
export function buildRecentFileCommands(paths: readonly string[], open: (path: string) => void): readonly Command[] {
  return paths.slice(0, RECENT_MENU_SLOTS).map((path, index) => {
    const name = basename(path);
    const dir = dirname(path);
    return {
      id: `file.recent.${index}`,
      title: dir === null ? name : `${name} — ${dir}`,
      category: t('command.category.file'),
      keywords: [name, 'recent', 'zuijin', '最近打开'],
      dynamic: true,
      run: () => open(path),
    } satisfies Command;
  });
}

export function registerBuiltInCommands(registry: CommandRegistry): void {
  for (const command of BUILT_IN_COMMANDS) registry.register(command);
}
