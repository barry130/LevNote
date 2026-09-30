/**
 * 设置页。
 *
 * ## 它解决的是「偏好没有落脚点」
 *
 * 在它之前，主题只能靠 `Ctrl+Shift+T` 循环、侧栏靠 `Ctrl+B`、行号藏在代码块的
 * 语言面板里 —— 每一条都是「知道的人才知道」。而**快捷键不是发现功能的入口**，
 * 它是熟练之后的加速器。一个只能靠快捷键改的偏好，等于没有。
 *
 * ## 快捷键一览是**派生的**，不是手写的
 *
 * 与命令面板同一条道理：列表直接由 `CommandRegistry` 生成。
 * 手写一份的结局必然是某天加了命令忘了补这张表，而**没人会发现**
 * （用户不会去核对设置页里的快捷键是不是全的）。
 *
 * ## 为什么没有「保存 / 取消」按钮
 *
 * 每一项都是即时生效的（主题、行号、位置记忆都是）。加了保存按钮反而要回答
 * 「点了取消之后主题回不回去」，而那个问题的答案没人猜得准。
 * 唯一的例外是「清除位置记忆」—— 它是一次**破坏性动作**，所以单独放、
 * 带确认。
 */

import { useEffect, useMemo, useRef } from 'react';

import { formatKeybinding, type Platform } from '../core/commands/keybinding.js';
import type { CommandRegistry } from '../core/commands/registry.js';
import type { Command } from '../core/commands/types.js';
import { askConfirm } from '../core/platform/index.js';
import type { ViewSettings } from '../core/settings.js';
import { THEME_LABELS, THEME_ORDER } from '../core/theme/theme.js';

export interface SettingsDialogProps {
  settings: ViewSettings;
  /** 改一项偏好。外壳负责落盘与生效，设置页只负责说「用户改了什么」。 */
  onChange: (patch: Partial<ViewSettings>) => void;
  registry: CommandRegistry;
  platform: Platform;
  /** 清空全部位置记忆。**破坏性**，界面上要带确认。 */
  onClearPositions: () => void;
  onClose: () => void;
}

interface ShortcutGroup {
  readonly category: string;
  readonly rows: ReadonlyArray<{ readonly command: Command; readonly binding: string | null }>;
}

function Toggle({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint: string;
  checked: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <label className="settings-toggle">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span>
        <span className="settings-toggle-label">{label}</span>
        <span className="settings-hint">{hint}</span>
      </span>
    </label>
  );
}

export function SettingsDialog({
  settings,
  onChange,
  registry,
  platform,
  onClearPositions,
  onClose,
}: SettingsDialogProps) {
  const dialogRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  /**
   * Esc 关闭。
   *
   * 挂在 `window` 的**捕获阶段**并 `stopPropagation` —— 与浮层菜单同一条理由
   * （硬约束 9）：默认监听的话事件会先经过编辑器，而编辑器可能正拿着焦点。
   * 捕获 + 拦下之后，这一次 Esc 就只归设置页处理。
   *
   * 与 `ui.escape` 命令并存不是重复：命令那条要经过全局快捷键表，
   * 而这里这条保证「哪怕焦点在对话框内部、或者某条命令被禁用」也关得掉。
   * 两条都只是把 `settingsOpen` 设成 false，重复执行是幂等的。
   */
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      onClose();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [onClose]);

  const groups = useMemo<ShortcutGroup[]>(() => {
    const byCategory = new Map<string, Array<{ command: Command; binding: string | null }>>();
    for (const command of registry.list()) {
      const binding = registry.displayBinding(command, platform, formatKeybinding);
      // 没有快捷键的命令不进这张表：它是「快捷键一览」，不是第二份命令面板
      if (binding === null) continue;
      const bucket = byCategory.get(command.category);
      const row = { command, binding };
      if (bucket) bucket.push(row);
      else byCategory.set(command.category, [row]);
    }
    return [...byCategory].map(([category, rows]) => ({ category, rows }));
  }, [registry, platform]);

  return (
    <div
      className="palette-backdrop"
      onMouseDown={(event) => {
        // 只在点到遮罩本身时关闭；点在对话框内部不关
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="settings" role="dialog" aria-label="设置" tabIndex={-1} ref={dialogRef}>
        <header className="settings-head">
          <h2>设置</h2>
          <span className="grow" />
          <button type="button" onClick={onClose}>
            关闭
          </button>
        </header>

        <div className="settings-body">
          <section className="settings-section">
            <h3>外观</h3>
            <label className="settings-row">
              <span className="settings-row-label">主题</span>
              <select
                value={settings.theme}
                onChange={(event) => {
                  const value = event.target.value;
                  // 不用 `as` 硬转：取值来自我们自己渲染的 option，但类型收窄
                  // 还是走同一份判定，免得将来选项列表变了这里悄悄接受一个非法值
                  const hit = THEME_ORDER.find((item) => item === value);
                  if (hit) onChange({ theme: hit });
                }}
              >
                {THEME_ORDER.map((item) => (
                  <option key={item} value={item}>
                    {THEME_LABELS[item]}
                  </option>
                ))}
              </select>
            </label>
          </section>

          <section className="settings-section">
            <h3>编辑器</h3>
            <Toggle
              label="代码块显示行号"
              hint="只影响 Markdown 文档里的代码块。代码 / 文本文档始终有行号。"
              checked={settings.codeLineNumbers}
              onChange={(next) => onChange({ codeLineNumbers: next })}
            />
            <Toggle
              label="软换行"
              hint="长行折到下一行，而不是横向滚动。只对代码 / 文本文档有效。"
              checked={settings.wordWrap}
              onChange={(next) => onChange({ wordWrap: next })}
            />
            <Toggle
              label="恢复上次的标签页"
              hint="重新打开时自动打开上次的标签页，没保存的改动也还在。关掉之后不再写会话记录 —— 异常退出时未保存的内容会丢。"
              checked={settings.restoreSession}
              onChange={(next) => onChange({ restoreSession: next })}
            />
            <Toggle
              label="记住文档位置"
              hint="重新打开时回到上次的光标与滚动位置。位置只存在本机，不会上传。"
              checked={settings.rememberPositions}
              onChange={(next) => onChange({ rememberPositions: next })}
            />
            <div className="settings-row">
              <span className="settings-hint">关掉上面的开关只是不再读写，已有的记忆还在。</span>
              <button
                type="button"
                onClick={() => {
                  // 破坏性动作必须问一次。`askConfirm` 走平台对话框，
                  // 桌面端是原生确认框 —— 浏览器预览下退化成 window.confirm
                  void (async () => {
                    const yes = await askConfirm('清除所有文档的位置记忆？此操作不可撤销。');
                    if (yes) onClearPositions();
                  })();
                }}
              >
                清除位置记忆
              </button>
            </div>
          </section>

          <section className="settings-section">
            <h3>快捷键</h3>
            <p className="settings-hint">
              这张表由命令注册表生成，与命令面板、工具栏同源 —— 不会出现「加了功能这里没更新」。
            </p>
            {groups.map((group) => (
              <div className="settings-keys" key={group.category}>
                <div className="settings-keys-category">{group.category}</div>
                {group.rows.map(({ command, binding }) => (
                  <div className="settings-keys-row" key={command.id}>
                    <span>{command.title}</span>
                    <span className="grow" />
                    <kbd className="palette-key">{binding}</kbd>
                  </div>
                ))}
              </div>
            ))}
          </section>

          <section className="settings-section">
            <h3>关于</h3>
            <div className="settings-row">
              <span className="settings-row-label">轻墨</span>
              <span className="settings-hint">版本 {import.meta.env.QINGMO_VERSION}</span>
            </div>
            <p className="settings-hint">
              纯本地编辑器：没有账号、没有网络请求、没有遥测。文档只存在你自己的磁盘上。
            </p>
          </section>
        </div>
      </div>
    </div>
  );
}
