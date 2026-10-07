/**
 * 活动栏（Activity Bar）—— VSCode 最左边那条竖着的图标带。
 *
 * ## 它解决的问题不是「好看」
 *
 * 在这之前，侧栏、大纲、查找、命令面板、设置这五件事分别藏在：
 * 一个快捷键、一个快捷键、一个快捷键、一个工具栏按钮、一个工具栏按钮。
 * **快捷键不是发现功能的入口**（这是 M9 写设置页时定下的判断），
 * 而工具栏只能放得下高频动作。活动栏是那个「常驻的、一眼能看全的入口」——
 * 它把「这个应用有哪些面板」变成一个**看得见**的问题。
 *
 * ## 为什么它是独立组件而不是写在 App 里
 *
 * 它自己没有状态：选中态、开关、回调全部由外壳给。放在 App 里会让那个
 * 已经 600 行的渲染函数再长 40 行，而它和 App 的其他部分没有任何耦合 ——
 * 典型的「可以整块搬走」的东西。
 *
 * ## 选中态用 `aria-pressed` 而不是 class
 *
 * 侧栏与大纲是**开关**（可开可关），不是「当前在哪一页」。用 `aria-pressed`
 * 表达开关，样式也挂在同一个属性上（`[aria-pressed='true']`）——
 * 于是「看起来是选中的」和「读屏说是选中的」永远来自同一个信号，
 * 不会出现某天改了 class 忘了改无障碍属性。
 *
 * ## 每个按钮里都有一个 `.sr-only` 文字
 *
 * 两个原因，都不是「为了好看」：
 *
 * 1. **无障碍**：纯图标按钮必须有一个可访问名字。`title` 只是个提示，
 *    它不进可访问名（有的读屏会念，有的不念，取决于配置）。
 *    真实文字是唯一在所有读屏里都算数的做法。
 * 2. **可测**：项目里点按钮的测试辅助函数是按**文字**找元素的
 *    （`[...container.querySelectorAll('button')].find(b => b.textContent.trim() === text)`）。
 *    没有文字，那个入口在测试里就够不着 —— 而「够不着」最常见的处理方式
 *    是给测试开一个后门（按 class 找），那才是真正的坏味道。
 */

import {
  IconCommand,
  IconFiles,
  IconOutline,
  IconSearch,
  IconSearchFolder,
  IconSettings,
} from './icons.js';
import { t } from '../core/i18n/index.js';

export interface ActivityBarProps {
  readonly sidebarVisible: boolean;
  /** 侧栏里现在看的是文件树（而不是检索）。 */
  readonly filesView: boolean;
  /** 侧栏里现在看的是检索面板。 */
  readonly searchView: boolean;
  readonly outlineVisible: boolean;
  /** 当前文档没有大纲时（非 Markdown、或者内核不支持），按钮置灰。 */
  readonly canOutline: boolean;
  /** 没有打开的文档时查找无处可查，按钮置灰。 */
  readonly canFind: boolean;
  /** 没有打开文件夹（或平台不支持）时，跨文件检索无处可查。 */
  readonly canSearchFolder: boolean;
  readonly onToggleSidebar: () => void;
  readonly onSearchFolder: () => void;
  readonly onToggleOutline: () => void;
  readonly onFind: () => void;
  readonly onPalette: () => void;
  readonly onSettings: () => void;
}

export function ActivityBar({
  sidebarVisible,
  filesView,
  searchView,
  outlineVisible,
  canOutline,
  canFind,
  canSearchFolder,
  onToggleSidebar,
  onSearchFolder,
  onToggleOutline,
  onFind,
  onPalette,
  onSettings,
}: ActivityBarProps) {
  return (
    <nav className="activitybar" aria-label={t('ui.activitybar.groupLabel')}>
      <button
        type="button"
        className="activity-item"
        aria-pressed={sidebarVisible && filesView}
        title={t('ui.activitybar.filesTitle')}
        onClick={onToggleSidebar}
      >
        <IconFiles size={22} />
        <span className="sr-only">{t('ui.activitybar.files')}</span>
      </button>

      <button
        type="button"
        className="activity-item"
        aria-pressed={sidebarVisible && searchView}
        disabled={!canSearchFolder}
        title={canSearchFolder ? t('ui.activitybar.searchFolderTitle') : t('ui.activitybar.searchNoFolder')}
        onClick={onSearchFolder}
      >
        <IconSearchFolder size={22} />
        <span className="sr-only">{t('ui.activitybar.search')}</span>
      </button>

      <button
        type="button"
        className="activity-item"
        disabled={!canFind}
        title={canFind ? t('ui.activitybar.findTitle') : t('ui.activitybar.findNoDoc')}
        onClick={onFind}
      >
        <IconSearch size={22} />
        <span className="sr-only">{t('ui.activitybar.find')}</span>
      </button>

      <button
        type="button"
        className="activity-item"
        aria-pressed={outlineVisible && canOutline}
        disabled={!canOutline}
        title={canOutline ? t('ui.activitybar.outlineTitle') : t('ui.activitybar.outlineNoHeadings')}
        onClick={onToggleOutline}
      >
        <IconOutline size={22} />
        <span className="sr-only">{t('ui.activitybar.outline')}</span>
      </button>

      {/* 把设置与命令面板推到底部 —— VSCode 也是这个位置，
          因为它们是「偶尔用一次」的动作，不该占着视线中心 */}
      <span className="grow" />

      <button type="button" className="activity-item" title={t('ui.activitybar.paletteTitle')} onClick={onPalette}>
        <IconCommand size={22} />
        <span className="sr-only">{t('ui.activitybar.palette')}</span>
      </button>

      <button type="button" className="activity-item" title={t('ui.activitybar.settingsTitle')} onClick={onSettings}>
        <IconSettings size={22} />
        <span className="sr-only">{t('ui.activitybar.settings')}</span>
      </button>
    </nav>
  );
}
