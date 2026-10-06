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

import { useEffect, useMemo, useRef, useState } from 'react';

import { formatKeybinding, isSafeGlobalSpec, specFromEvent, type Platform } from '../core/commands/keybinding.js';
import type { CommandRegistry } from '../core/commands/registry.js';
import type { Command } from '../core/commands/types.js';
import { askConfirm } from '../core/platform/index.js';
import {
  clampEditorFontSize,
  clampEditorLineHeight,
  sanitizeEditorFontFamily,
  EDITOR_FONT_SIZE_MAX,
  EDITOR_FONT_SIZE_MIN,
  EDITOR_LINE_HEIGHT_MAX,
  EDITOR_LINE_HEIGHT_MIN,
  type ViewSettings,
} from '../core/settings.js';
import { THEME_LABELS, THEME_ORDER } from '../core/theme/theme.js';
import { SUPPORTED_LOCALES } from '../core/i18n/messages.js';

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
  readonly rows: ReadonlyArray<{
    readonly command: Command;
    readonly binding: string | null;
    /** 用户改过这条命令的快捷键（含显式清除）—— 「恢复默认」按钮只在这种行出现 */
    readonly overridden: boolean;
  }>;
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
  // 快捷键录制（P6 第 31 项）：正在改哪条命令、已经录到的写法、录制期错误
  const [editingId, setEditingId] = useState<string | null>(null);
  const [recorded, setRecorded] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

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
   *
   * **录制快捷键时这条要让路**：Esc 在录制状态下是「取消录制」，
   * 不是「关闭设置」。两个监听同为 window 捕获、按注册顺序先后执行，
   * 所以这里必须显式检查录制状态，否则录不了任何含 Esc 语义的东西。
   */
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (editingId !== null) return;
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      onClose();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [onClose, editingId]);

  /**
   * 录制监听：编辑中的每一次按键（捕获阶段，先于全局快捷键表）被收成
   * 规范写法。只按修饰键时 `specFromEvent` 给 null —— 继续等真正的键名。
   * `preventDefault` + `stopPropagation` 让录制期间的全局快捷键全部失效
   * （否则录 `Mod+S` 的那一刻文档就先被保存了）。
   */
  useEffect(() => {
    if (editingId === null) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        setEditingId(null);
        setRecorded(null);
        setError(null);
        return;
      }
      const spec = specFromEvent(event, platform);
      if (spec === null) return;
      event.preventDefault();
      event.stopPropagation();
      setRecorded(spec);
      // 单字母 / 数字键在输入框里会被打字占用（见 isAllowedInEditable 的
      // 白名单理由），录进来就是「设置完打不了字」—— 当场说明，不让保存
      setError(isSafeGlobalSpec(spec, platform) ? null : '单个字母或数字键会被输入占用，请带上 Ctrl、Alt 之类的修饰键。');
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [editingId, platform]);

  /** 把录制结果写进偏好；如果这个写法已被别的命令占用，把它**显式顶掉**（置空）。 */
  const saveRecording = () => {
    if (editingId === null || recorded === null || error !== null) return;
    const next: Record<string, string> = { ...settings.keybindings, [editingId]: recorded };
    const owner = registry.findOwnerOf(recorded, platform, editingId);
    if (owner !== null) next[owner.id] = '';
    onChange({ keybindings: next });
    setEditingId(null);
    setRecorded(null);
    setError(null);
  };

  /** 回到内置默认；若默认写法已被别的命令覆盖占用，同样把它显式顶掉。 */
  const restoreDefault = (command: Command) => {
    const next: Record<string, string> = { ...settings.keybindings };
    if (!Object.prototype.hasOwnProperty.call(next, command.id)) return;
    delete next[command.id];
    if (command.keybinding) {
      const owner = registry.findOwnerOf(command.keybinding, platform, command.id);
      if (owner !== null) next[owner.id] = '';
    }
    onChange({ keybindings: next });
  };

  /** 冲突提示：录到的写法此刻被谁占着（编辑自己除外）。 */
  const conflictOwner = useMemo(() => {
    if (editingId === null || recorded === null || error !== null) return null;
    return registry.findOwnerOf(recorded, platform, editingId);
  }, [editingId, recorded, error, registry, platform]);

  const groups = useMemo<ShortcutGroup[]>(() => {
    /**
     * 与旧版不同：**所有命令都进这张表**。过去「没有快捷键的命令不进」
     * 是因为表只是「一览」；现在每一条都可以被指派快捷键，
     * 不显示出来的命令就永远指派不了 —— 那才是真正的「知道的人才知道」。
     *
     * 唯一的例外是 `dynamic` 命令（最近打开的文件之类）：它们是数据不是能力，
     * 每个文件一行会把这张表淹掉，而用户也不会想给「打开某个具体文件」
     * 指派快捷键。
     */
    const byCategory = new Map<string, Array<{ command: Command; binding: string | null; overridden: boolean }>>();
    for (const command of registry.list()) {
      if (command.dynamic === true) continue;
      const binding = registry.displayBinding(command, platform, formatKeybinding);
      const overridden = Object.prototype.hasOwnProperty.call(settings.keybindings, command.id);
      const bucket = byCategory.get(command.category);
      const row = { command, binding, overridden };
      if (bucket) bucket.push(row);
      else byCategory.set(command.category, [row]);
    }
    return [...byCategory].map(([category, rows]) => ({ category, rows }));
    // 语言切换后这张表的标题来自重新注册过的命令 —— 重挂由外壳的 key 保证，
    // 不用在这里塞一个 linter 会当成「多余」的 locale 依赖
  }, [registry, platform, settings.keybindings]);

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
              {/* 标签刻意双语：文案迁移分批进行，这一行在两种语言下都要能被认出来 */}
              <span className="settings-row-label">语言 / Language</span>
              <select
                value={settings.locale}
                onChange={(event) => {
                  const value = event.target.value;
                  const hit = SUPPORTED_LOCALES.find((item) => item === value);
                  if (hit) onChange({ locale: hit });
                }}
              >
                {SUPPORTED_LOCALES.map((item) => (
                  <option key={item} value={item}>
                    {item === 'zh-CN' ? '简体中文' : 'English'}
                  </option>
                ))}
              </select>
            </label>
            <p className="settings-hint">
              目前覆盖菜单 / 命令面板 / 快捷键表，其余界面将分批迁移。
              Covers menus, command palette &amp; shortcuts; the rest follows in batches.
            </p>
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
            <div className="settings-row">
              <span className="settings-row-label">字号</span>
              <input
                type="number"
                min={EDITOR_FONT_SIZE_MIN}
                max={EDITOR_FONT_SIZE_MAX}
                value={settings.editorFontSize}
                onChange={(event) => onChange({ editorFontSize: clampEditorFontSize(Number(event.target.value)) })}
              />
              <span className="settings-hint">px（{EDITOR_FONT_SIZE_MIN}–{EDITOR_FONT_SIZE_MAX}），三类编辑器一起变</span>
            </div>
            <label className="settings-row">
              <span className="settings-row-label">正文字体</span>
              <input
                type="text"
                value={settings.editorFontFamily}
                placeholder="内置默认"
                onChange={(event) => onChange({ editorFontFamily: sanitizeEditorFontFamily(event.target.value) })}
              />
              <span className="settings-hint">留空 = 内置；代码与代码块始终等宽</span>
            </label>
            <div className="settings-row">
              <span className="settings-row-label">行高</span>
              <input
                type="number"
                min={EDITOR_LINE_HEIGHT_MIN}
                max={EDITOR_LINE_HEIGHT_MAX}
                step={0.05}
                value={settings.editorLineHeight}
                onChange={(event) => onChange({ editorLineHeight: clampEditorLineHeight(Number(event.target.value)) })}
              />
              <span className="settings-hint">{EDITOR_LINE_HEIGHT_MIN}–{EDITOR_LINE_HEIGHT_MAX}</span>
            </div>
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
              点「改」后按下新的组合键；被顶掉的原命令会显示为「未设置」。
            </p>
            {groups.map((group) => (
              <div className="settings-keys" key={group.category}>
                <div className="settings-keys-category">{group.category}</div>
                {group.rows.map(({ command, binding, overridden }) => (
                  <div className="settings-keys-row" key={command.id}>
                    <span>{command.title}</span>
                    <span className="grow" />
                    {editingId === command.id ? (
                      <>
                        <kbd className="palette-key">{recorded ?? '按下新的组合键…'}</kbd>
                        <button type="button" disabled={recorded === null || error !== null} onClick={saveRecording}>
                          保存
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setEditingId(null);
                            setRecorded(null);
                            setError(null);
                          }}
                        >
                          取消
                        </button>
                      </>
                    ) : (
                      <>
                        <kbd className="palette-key">{binding ?? '未设置'}</kbd>
                        <button
                          type="button"
                          onClick={() => {
                            setEditingId(command.id);
                            setRecorded(null);
                            setError(null);
                          }}
                        >
                          改
                        </button>
                        {overridden && (
                          <button type="button" onClick={() => restoreDefault(command)}>
                            恢复默认
                          </button>
                        )}
                      </>
                    )}
                  </div>
                ))}
              </div>
            ))}
            {editingId !== null && error !== null && (
              <p className="settings-hint settings-keys-conflict">{error}</p>
            )}
            {editingId !== null && error === null && conflictOwner !== null && (
              <p className="settings-hint settings-keys-conflict">
                「{conflictOwner.title}」当前也用这个组合键 —— 保存后它将显示为「未设置」。
              </p>
            )}
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
