/**
 * 内置命令清单。
 *
 * 这里是**能力的唯一清单**：工具栏按钮、命令面板、快捷键都从这一份派生。
 * 因此不存在「加了按钮忘了加快捷键」或「面板里搜不到这个功能」的可能。
 *
 * 命名约定：`域.动作`。域只用于面板分组与将来的快捷键自定义界面。
 */

import { exportHtml } from '../export/export-document.js';
import type { CommandRegistry } from './registry.js';
import type { Command, CommandContext } from './types.js';

function activeDocument(ctx: CommandContext) {
  return ctx.workspace.activeStore?.getSnapshot().doc ?? null;
}

function hasDocument(ctx: CommandContext): boolean {
  return activeDocument(ctx) !== null;
}

async function runExport(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  if (!doc) return;

  const handle = ctx.ui.editor();

  // toHtml() 返回 null 是**合法结果**（纯文本内核就是如此），不是错误。
  // 抛错也按 null 处理：导出成纯文本总比导不出来强。
  // 不写初始值：try 与 catch 两条路径都会赋值，`= null` 是永远读不到的死赋值。
  let bodyHtml: string | null;
  try {
    bodyHtml = handle ? await handle.toHtml() : null;
  } catch {
    bodyHtml = null;
  }

  const result = await exportHtml({
    path: doc.path,
    title: doc.displayName.replace(/\.[^.]+$/, ''),
    bodyHtml,
    fallbackText: doc.text,
    provider: ctx.provider,
  });

  if (result.status === 'saved') await ctx.ui.notify(result.message, '导出完成');
  else if (result.status === 'failed') await ctx.ui.notify(result.message, '导出失败');
}

export const BUILT_IN_COMMANDS: readonly Command[] = [
  // ---- 文件 -------------------------------------------------------------
  {
    id: 'file.open',
    title: '打开文件',
    category: '文件',
    keybinding: 'Mod+O',
    keywords: ['open', 'dakaishejian'],
    run: (ctx) => {
      void ctx.workspace.openViaDialog();
    },
  },
  {
    id: 'file.openFolder',
    title: '打开文件夹',
    category: '文件',
    keybinding: 'Mod+Shift+O',
    keywords: ['open folder', 'workspace', 'gongzuoqu'],
    run: (ctx) => {
      void ctx.workspace.openFolder();
    },
  },
  {
    id: 'file.save',
    title: '保存',
    category: '文件',
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
    title: '另存为',
    category: '文件',
    keybinding: 'Mod+Shift+S',
    keywords: ['save as', 'lingcunwei'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.saveAs();
    },
  },
  {
    id: 'file.reload',
    title: '从磁盘重新加载',
    category: '文件',
    keywords: ['reload', 'chongxinjiazai'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.reload();
    },
  },
  {
    id: 'file.closeTab',
    title: '关闭当前标签页',
    category: '文件',
    keybinding: 'Mod+W',
    keywords: ['close', 'guanbi'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.closeActiveTab(),
  },
  {
    id: 'file.exportHtml',
    title: '导出为自包含 HTML',
    category: '导出',
    keybinding: 'Mod+Shift+E',
    keywords: ['export', 'html', 'daochu'],
    enabled: hasDocument,
    run: runExport,
  },

  // ---- 标签页 -----------------------------------------------------------
  {
    id: 'tab.next',
    title: '下一个标签页',
    category: '标签页',
    keybinding: 'Ctrl+Tab',
    keywords: ['next tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(1),
  },
  {
    id: 'tab.prev',
    title: '上一个标签页',
    category: '标签页',
    keybinding: 'Ctrl+Shift+Tab',
    keywords: ['previous tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(-1),
  },

  // ---- 视图 -------------------------------------------------------------
  {
    id: 'view.toggleSidebar',
    title: '切换侧边栏',
    category: '视图',
    keybinding: 'Mod+B',
    keywords: ['sidebar', 'cebianlan'],
    run: (ctx) => ctx.ui.toggleSidebar(),
  },
  {
    id: 'view.toggleOutline',
    title: '切换大纲',
    category: '视图',
    keybinding: 'Mod+Shift+L',
    keywords: ['outline', 'dagang'],
    run: (ctx) => ctx.ui.toggleOutline(),
  },
  {
    id: 'view.toggleSource',
    title: '切换源码模式',
    category: '视图',
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
    id: 'view.cycleTheme',
    title: '切换主题',
    category: '视图',
    keybinding: 'Mod+Shift+T',
    keywords: ['theme', 'zhuti', 'dark', 'light'],
    run: (ctx) => ctx.ui.cycleTheme(),
  },

  // ---- 导航 -------------------------------------------------------------
  {
    id: 'find.open',
    title: '在当前文档中查找',
    category: '导航',
    keybinding: 'Mod+F',
    keywords: ['find', 'search', 'chazhao'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.openFind(),
  },
  {
    id: 'palette.open',
    title: '命令面板',
    category: '导航',
    keybinding: 'Mod+P',
    keywords: ['command palette', 'minglingmianban'],
    run: (ctx) => ctx.ui.openPalette(),
  },
  {
    id: 'app.settings',
    title: '设置',
    category: '应用',
    keybinding: 'Mod+,',
    keywords: ['settings', 'preferences', 'shezhi', 'pianhao'],
    run: (ctx) => ctx.ui.openSettings(),
  },
  {
    id: 'ui.escape',
    title: '关闭浮层',
    category: '导航',
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
