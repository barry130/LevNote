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

  // ---- 文件 ---------------------------------------------------------------
  'command.file.open.title': '打开文件',
  'command.file.openFolder.title': '打开文件夹',
  'command.file.save.title': '保存',
  'command.file.saveAs.title': '另存为',
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

  // ---- 导航 ---------------------------------------------------------------
  'command.find.open.title': '在当前文档中查找',
  'command.search.inFolder.title': '在文件夹中检索',
  'command.palette.open.title': '命令面板',
  'command.ui.escape.title': '关闭浮层',

  // ---- 应用 ---------------------------------------------------------------
  'command.app.settings.title': '设置',

  // ---- 诊断 ---------------------------------------------------------------
  'command.diagnostics.roundTrip.title': '往返保真自检（保存会改写什么）',
} as const;

export type MessageKey = keyof typeof zhCN;

/** 已有目录的语言。加第二种语言时：加目录、扩这个联合类型、接设置页。 */
export type SupportedLocale = 'zh-CN';
