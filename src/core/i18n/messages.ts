/**
 * 文案目录（P3 第 17 项，ADR 0031）。
 *
 * ## 这是什么、这不是什么
 *
 * 这是 i18n 的**骨架**：把用户可见文案收进唯一目录、用类型安全的方式取用，
 * 让「换一种语言」从全局扫字符串变成「加一份目录」。目前只有 zh-CN 一份
 * —— ROADMAP 的原话是「先抽文案，不急翻译」，目录里没有的第二语言不设置、
 * 不假装存在。
 *
 * ## key 约定
 *
 * - 命令标题：`command.<命令 id>.title`（`file.open` → `command.file.open.title`）；
 * - 命令分组：`command.category.<分组名>`；
 * - key 是**稳定标识**，改显示文案只改值；改 key 就要连同调用点一起改 ——
 *   这是有意的：key 出现在调用点旁边，搜得到、对得上。
 *
 * ## 为什么不引入 i18n 库
 *
 * 骨架只需要「查表 + 插值 + 回退」三件事，四十行以内；i18next 这类库带来
 * 的复数规则、日期格式化、懒加载命名空间，在只有一种语言时全是死重。
 * 何时引入见 ADR 0031 的重审条件。
 */

export const zhCN = {
  // ---- 命令分组 -----------------------------------------------------------
  'command.category.file': '文件',
  'command.category.export': '导出',
  'command.category.tab': '标签页',
  'command.category.view': '视图',
  'command.category.navigate': '导航',
  'command.category.app': '应用',
  'command.category.diagnostics': '诊断',
  'command.category.edit': '编辑',
  'command.category.paragraph': '段落',
  'command.category.format': '格式',
  'command.category.theme': '主题',
  'command.category.encoding': '编码',
  'command.category.help': '帮助',

  // ---- 菜单栏（下拉菜单的名字；与命令分组不是一个东西 ——
  //      菜单是外壳的导航结构，分组是命令面板的归类） ------------------------
  'menu.file': '文件',
  'menu.edit': '编辑',
  'menu.paragraph': '段落',
  'menu.format': '格式',
  'menu.search': '搜索',
  'menu.view': '视图',
  'menu.encoding': '编码',
  'menu.theme': '主题',
  'menu.help': '帮助',

  // ---- 文件 ---------------------------------------------------------------
  'command.file.new.title': '新建文件',
  'command.file.open.title': '打开文件',
  'command.file.openFolder.title': '打开文件夹',
  'command.file.save.title': '保存',
  'command.file.saveAs.title': '另存为',
  'command.file.saveAll.title': '保存全部',
  'command.file.reload.title': '从磁盘重新加载',
  'command.file.closeTab.title': '关闭当前标签页',

  // ---- 导出 ---------------------------------------------------------------
  'command.file.exportHtml.title': '导出为自包含 HTML',
  'command.file.exportPdf.title': '打印 / 导出为 PDF',

  // ---- 标签页 ---------------------------------------------------------------
  'command.tab.next.title': '下一个标签页',
  'command.tab.prev.title': '上一个标签页',

  // ---- 视图 ---------------------------------------------------------------
  'command.view.toggleSidebar.title': '切换侧边栏',
  'command.view.filesView.title': '文件浏览器',
  'command.view.toggleOutline.title': '切换大纲',
  'command.view.toggleSource.title': '切换源码模式',
  'command.view.rich.title': '富文本视图',
  'command.view.source.title': '源码视图',
  'command.view.split.title': '源码 + 预览（并排）',
  'command.view.cycleTheme.title': '切换主题',
  'command.view.toggleWordWrap.title': '切换自动换行',
  'command.view.toggleCodeLineNumbers.title': '切换代码行号',
  'command.view.toggleFullscreen.title': '切换全屏',

  // ---- 导航 ---------------------------------------------------------------
  'command.find.open.title': '在当前文档中查找',
  'command.search.inFolder.title': '在文件夹中检索',
  'command.palette.open.title': '命令面板',
  'command.ui.escape.title': '关闭浮层',

  // ---- 应用 ---------------------------------------------------------------
  'command.app.settings.title': '设置',

  // ---- 帮助 ---------------------------------------------------------------
  'command.help.sampleDocument.title': '打开示例文档',

  // ---- 诊断 ---------------------------------------------------------------
  'command.diagnostics.roundTrip.title': '往返保真自检（保存会改写什么）',
  'command.edit.extractReference.title': '提取为引用',
  'command.edit.inlineReference.title': '拉平引用',
  'command.edit.undo.title': '撤销',
  'command.edit.redo.title': '重做',
  'command.edit.copyHtml.title': '复制为 HTML',

  // ---- 格式（选中文字的行内格式；只有实现了 formatting 契约的内核可用） ----
  'command.format.bold.title': '加粗',
  'command.format.italic.title': '斜体',
  'command.format.strikethrough.title': '删除线',
  'command.format.inlineCode.title': '行内代码',

  // ---- 段落（光标所在块的格式与插入；同上，Markdown 内核可用） -------------
  'command.paragraph.plainText.title': '正文',
  'command.paragraph.heading1.title': '一级标题',
  'command.paragraph.heading2.title': '二级标题',
  'command.paragraph.heading3.title': '三级标题',
  'command.paragraph.heading4.title': '四级标题',
  'command.paragraph.heading5.title': '五级标题',
  'command.paragraph.heading6.title': '六级标题',
  'command.paragraph.blockquote.title': '引用',
  'command.paragraph.bulletList.title': '无序列表',
  'command.paragraph.orderedList.title': '有序列表',
  'command.paragraph.codeBlock.title': '代码块',
  'command.paragraph.table.title': '表格',
  'command.paragraph.mathBlock.title': '数学块',
  'command.paragraph.diagram.title': 'Mermaid 图表',
  'command.paragraph.drawio.title': 'draw.io 图表',
  'command.paragraph.hr.title': '分隔线',

  // ---- 主题 ---------------------------------------------------------------
  'command.theme.system.title': '跟随系统',
  'command.theme.paper.title': '明亮（纸白）',
  'command.theme.ink.title': '暗黑（墨黑）',

  // ---- 应用 ---------------------------------------------------------------
  'command.app.about.title': '关于轻墨',
} as const;

export type MessageKey = keyof typeof zhCN;

/** 已有目录的语言。加第三种语言时：加目录、扩这个联合类型、接设置页。 */
export type SupportedLocale = 'zh-CN' | 'en';

/** 语言清单（设置页的下拉选项与 settings.ts 的校验共用这一份）。 */
export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['zh-CN', 'en'];

/**
 * 英文目录（2026-10-07 接入设置页 —— 第二份目录落地）。
 *
 * 类型是 `Record<MessageKey, string>`：**漏 key 是编译错误** —— 英文目录
 * 刻意要求全量对应，回退链（缺 key 落回 zh-CN）是给「将来只翻了一部分的
 * 语言」准备的，不是给偷懒准备的。
 *
 * 目前覆盖**命令系统**（菜单 / 命令面板 / 快捷键表）。设置页与各面板的
 * 文案还在 zh-CN 目录里没抽出来 —— 迁移是分批的机械活，每迁一批，
 * 那块界面就在两种语言下都成立。
 */
export const en: Record<MessageKey, string> = {
  // ---- 命令分组 -----------------------------------------------------------
  'command.category.file': 'File',
  'command.category.export': 'Export',
  'command.category.tab': 'Tabs',
  'command.category.view': 'View',
  'command.category.navigate': 'Navigate',
  'command.category.app': 'App',
  'command.category.diagnostics': 'Diagnostics',
  'command.category.edit': 'Edit',
  'command.category.paragraph': 'Paragraph',
  'command.category.format': 'Format',
  'command.category.theme': 'Theme',
  'command.category.encoding': 'Encoding',
  'command.category.help': 'Help',

  // ---- 菜单栏 -------------------------------------------------------------
  'menu.file': 'File',
  'menu.edit': 'Edit',
  'menu.paragraph': 'Paragraph',
  'menu.format': 'Format',
  'menu.search': 'Search',
  'menu.view': 'View',
  'menu.encoding': 'Encoding',
  'menu.theme': 'Theme',
  'menu.help': 'Help',

  // ---- 文件 ---------------------------------------------------------------
  'command.file.new.title': 'New File',
  'command.file.open.title': 'Open File…',
  'command.file.openFolder.title': 'Open Folder…',
  'command.file.save.title': 'Save',
  'command.file.saveAs.title': 'Save As…',
  'command.file.saveAll.title': 'Save All',
  'command.file.reload.title': 'Reload from Disk',
  'command.file.closeTab.title': 'Close Tab',

  // ---- 导出 ---------------------------------------------------------------
  'command.file.exportHtml.title': 'Export as Self-Contained HTML',
  'command.file.exportPdf.title': 'Print / Export as PDF',

  // ---- 标签页 ---------------------------------------------------------------
  'command.tab.next.title': 'Next Tab',
  'command.tab.prev.title': 'Previous Tab',

  // ---- 视图 ---------------------------------------------------------------
  'command.view.toggleSidebar.title': 'Toggle Sidebar',
  'command.view.filesView.title': 'File Explorer',
  'command.view.toggleOutline.title': 'Toggle Outline',
  'command.view.toggleSource.title': 'Toggle Source Mode',
  'command.view.rich.title': 'Rich Text View',
  'command.view.source.title': 'Source View',
  'command.view.split.title': 'Source + Preview (Side by Side)',
  'command.view.cycleTheme.title': 'Cycle Theme',
  'command.view.toggleWordWrap.title': 'Toggle Word Wrap',
  'command.view.toggleCodeLineNumbers.title': 'Toggle Code Line Numbers',
  'command.view.toggleFullscreen.title': 'Toggle Full Screen',

  // ---- 导航 ---------------------------------------------------------------
  'command.find.open.title': 'Find in Document',
  'command.search.inFolder.title': 'Search in Folder',
  'command.palette.open.title': 'Command Palette',
  'command.ui.escape.title': 'Close Overlays',

  // ---- 应用 ---------------------------------------------------------------
  'command.app.settings.title': 'Settings',

  // ---- 帮助 ---------------------------------------------------------------
  'command.help.sampleDocument.title': 'Open Sample Document',

  // ---- 诊断 ---------------------------------------------------------------
  'command.diagnostics.roundTrip.title': 'Round-trip Fidelity Check (What Saving Would Rewrite)',
  'command.edit.extractReference.title': 'Extract as Reference',
  'command.edit.inlineReference.title': 'Flatten Reference',
  'command.edit.undo.title': 'Undo',
  'command.edit.redo.title': 'Redo',
  'command.edit.copyHtml.title': 'Copy as HTML',

  // ---- 格式 ---------------------------------------------------------------
  'command.format.bold.title': 'Bold',
  'command.format.italic.title': 'Italic',
  'command.format.strikethrough.title': 'Strikethrough',
  'command.format.inlineCode.title': 'Inline Code',

  // ---- 段落 ---------------------------------------------------------------
  'command.paragraph.plainText.title': 'Body Text',
  'command.paragraph.heading1.title': 'Heading 1',
  'command.paragraph.heading2.title': 'Heading 2',
  'command.paragraph.heading3.title': 'Heading 3',
  'command.paragraph.heading4.title': 'Heading 4',
  'command.paragraph.heading5.title': 'Heading 5',
  'command.paragraph.heading6.title': 'Heading 6',
  'command.paragraph.blockquote.title': 'Blockquote',
  'command.paragraph.bulletList.title': 'Bullet List',
  'command.paragraph.orderedList.title': 'Ordered List',
  'command.paragraph.codeBlock.title': 'Code Block',
  'command.paragraph.table.title': 'Table',
  'command.paragraph.mathBlock.title': 'Math Block',
  'command.paragraph.diagram.title': 'Mermaid Diagram',
  'command.paragraph.drawio.title': 'draw.io Diagram',
  'command.paragraph.hr.title': 'Horizontal Rule',

  // ---- 主题 ---------------------------------------------------------------
  'command.theme.system.title': 'Follow System',
  'command.theme.paper.title': 'Light (Paper)',
  'command.theme.ink.title': 'Dark (Ink)',

  // ---- 应用 ---------------------------------------------------------------
  'command.app.about.title': 'About Qingmo',
};
