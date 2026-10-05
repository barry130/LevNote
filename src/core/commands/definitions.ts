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
import { describeRoundTrip, formatRoundTripReport } from '../fidelity.js';
import { t } from '../i18n/index.js';
import { isDesktop, printHtml } from '../platform/index.js';

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

  if (result.status === 'saved') await ctx.ui.notify(result.message, '导出完成');
  else if (result.status === 'failed') await ctx.ui.notify(result.message, '导出失败');
}

async function runPrintExport(ctx: CommandContext): Promise<void> {
  const input = await composeExportInput(ctx);
  if (input === null) return;

  // 与 HTML 导出共用同一个按需加载理由：渲染模块不进主包（P3 第 18 项）
  const { printDocument } = await import('../export/print-document.js');
  const result = await printDocument({ ...input, provider: ctx.provider, printHtml });

  if (result.status === 'opened') await ctx.ui.notify(result.message, '打印 / 导出 PDF');
  else await ctx.ui.notify(result.message, '导出失败');
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
    await ctx.ui.notify(`往返保真自检失败：${reason}`, '往返保真');
    return;
  }

  const report = describeRoundTrip(doc.savedText, roundTripped);
  await ctx.ui.notify(formatRoundTripReport(report), '往返保真');
}

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

export function registerBuiltInCommands(registry: CommandRegistry): void {
  for (const command of BUILT_IN_COMMANDS) registry.register(command);
}
