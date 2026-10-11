/**
 * 命令面板。
 *
 * ## 为什么值得做
 *
 * 它把「发现功能」的成本从「知道快捷键或找到按钮」降到「打出脑子里的词」。
 * 对本产品尤其重要：四类文档 + 两套内核 + 工作区，功能面比单文件编辑器宽得多，
 * 靠工具栏塞不下，靠菜单会长到没人看。
 *
 * ## 它为什么不会过期
 *
 * 列表直接由 `CommandRegistry` 生成 —— 注册一条命令，它就自动出现在这里。
 * 这是「命令是能力的唯一清单」这个设计的直接收益：
 * **面板里永远不会有指向已删除功能的幽灵条目，也永远不会有注册了却搜不到的功能。**
 *
 * 禁用的命令**显示但不可执行**（`enabled` 返回 false），并说明它为什么不可用。
 * 直接隐藏会让用户以为功能不存在。
 */

import { useEffect, useMemo, useRef, useState } from 'react';

import { formatKeybinding, type Platform } from '../core/commands/keybinding.js';
import { rankCommands } from '../core/commands/fuzzy.js';
import type { CommandRegistry } from '../core/commands/registry.js';
import { runCommandSafely } from '../core/commands/runner.js';
import type { Command, CommandContext } from '../core/commands/types.js';
import { t } from '../core/i18n/index.js';

export interface CommandPaletteProps {
  registry: CommandRegistry;
  context: CommandContext;
  platform: Platform;
  onClose: () => void;
}

interface Row {
  readonly command: Command;
  readonly enabled: boolean;
  readonly binding: string | null;
}

export function CommandPalette({ registry, context, platform, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [highlighted, setHighlighted] = useState(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const rows = useMemo<Row[]>(() => {
    // `dynamic` 命令（最近打开的文件之类）是数据不是能力，不进面板
    const all = registry.list().filter((command) => command.dynamic !== true);
    const ranked = rankCommands(query, all, (command) => ({
      title: command.title,
      keywords: command.keywords ?? [],
      id: command.id,
    }));

    return ranked.map(({ item }) => {
      // 不写初始值：try 与 catch 都会赋值，`= true` 是永远读不到的死赋值
      let enabled: boolean;
      try {
        enabled = item.enabled ? item.enabled(context) : true;
      } catch {
        // 一条命令的可用性判断写挂了，不该让整个面板打不开
        enabled = false;
      }
      return {
        command: item,
        enabled,
        binding: registry.displayBinding(item, platform, formatKeybinding),
      };
    });
    // context 每帧都是新对象，但它只是被传给 enabled 去读当前状态，
    // 不作为记忆键 —— 面板打开期间的列表内容只由 query 决定
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [registry, query, platform]);

  // 高亮项越界时收回到最后一项：搜索词变化会让列表变短
  useEffect(() => {
    setHighlighted((at) => Math.min(at, Math.max(0, rows.length - 1)));
  }, [rows.length]);

  const run = (row: Row | undefined): void => {
    if (!row || !row.enabled) return;
    onClose();
    // 关闭面板之后再执行：命令可能会打开另一个浮层（查找栏），
    // 先执行会让那个浮层在面板关闭前被渲染出来，视觉上闪一下
    // 与外壳那两个入口同一条兜底（ADR 0041）：面板里执行的命令失败了，
    // 也要说一句 —— 否则「选中回车、面板关了、什么都没发生」
    runCommandSafely(row.command, context);
  };

  return (
    <div
      className="palette-backdrop"
      onMouseDown={(event) => {
        // 只在点到遮罩本身时关闭；点在面板内部不关
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="palette" role="dialog" aria-label={t('ui.palette.title')}>
        <input
          ref={inputRef}
          className="palette-input"
          type="text"
          value={query}
          placeholder={t('ui.palette.placeholder')}
          spellCheck={false}
          onChange={(event) => {
            setQuery(event.target.value);
            setHighlighted(0);
          }}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown') {
              event.preventDefault();
              setHighlighted((at) => Math.min(at + 1, rows.length - 1));
            } else if (event.key === 'ArrowUp') {
              event.preventDefault();
              setHighlighted((at) => Math.max(at - 1, 0));
            } else if (event.key === 'Enter') {
              event.preventDefault();
              run(rows[highlighted]);
            } else if (event.key === 'Escape') {
              event.preventDefault();
              onClose();
            }
          }}
        />

        <div className="palette-list" ref={listRef}>
          {rows.length === 0 && <div className="palette-empty">{t('ui.palette.empty')}</div>}
          {rows.map((row, index) => (
            <button
              key={row.command.id}
              type="button"
              className={`palette-row${index === highlighted ? ' highlighted' : ''}${row.enabled ? '' : ' disabled'}`}
              disabled={!row.enabled}
              onMouseEnter={() => setHighlighted(index)}
              onClick={() => run(row)}
            >
              <span className="palette-category">{row.command.category}</span>
              <span className="palette-title">{row.command.title}</span>
              {!row.enabled && <span className="palette-note">{t('ui.palette.disabledNote')}</span>}
              <span className="grow" />
              {row.binding && <kbd className="palette-key">{row.binding}</kbd>}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
