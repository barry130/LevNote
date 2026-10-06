/**
 * 侧边栏文件树。
 *
 * 「打开文件夹」是 horseMD 的核心工作流之一，也是本产品从「单文件编辑器」
 * 变成「工作区编辑器」的关键一步。
 *
 * ## 三条刻意的取舍
 *
 * 1. **懒加载**：只有展开过的目录才去 `listDir`。一次性递归整个目录树在
 *    真实项目上会卡住（`node_modules` 就是几万个子项），而且绝大多数目录用户根本不会展开。
 * 2. **默认折叠重目录**：`node_modules` / `target` / `.git` 直接不显示
 *    （见 `tree.ts` 的 `DEFAULT_IGNORED`）。这不是「功能缺失」，
 *    而是「不让侧栏被淹掉」。用户可以一键显示出来。
 * 3. **不自动展开当前文件**：由 `WorkspaceStore.revealInTree` 在打开文件时
 *    精确展开祖先链，而不是把整棵树摊开 —— 后者会把用户的展开状态冲掉。
 *
 * ## 文件管理（新建 / 重命名 / 删除）
 *
 * 入口是**行上的悬停按钮**（目录多两个「新建」、所有行有「重命名 / 删除」）
 * 加头部的根目录「新建」—— 与本文件其余部分一致，图标是字符不是图标库。
 * 编辑走**行内输入**（不是弹窗）：输入的名字就长在它要出现的位置上，
 * Enter 确认、Esc 取消、点别处提交。
 *
 * 分工与斜杠菜单同构：**组件只管交互**（输入框、键盘、忙碌防重），
 * 「名字合不合法、磁盘动不动、标签页跟着怎么变」全在 `WorkspaceStore`
 * 的 `createEntry` / `renameEntry` / `deleteEntry`。提交失败时输入框**留在原地**
 * （错误显示在树顶的横幅里），用户可以改个名字再试 —— 清掉输入框等于
 * 让他重新打一遍自己刚打错的东西。
 */

import { Fragment, useEffect, useMemo, useRef, useState } from 'react';

import { basename } from '../core/path.js';
import { flattenTree, type TreeRow } from '../core/workspace/tree.js';
import type { TreeState } from '../core/workspace/store.js';

export interface SidebarProps {
  tree: TreeState;
  activePath: string | null;
  /** false（浏览器预览）时文件管理按钮整体不画 —— 画出来点不了是说谎。 */
  canManageFiles: boolean;
  onOpenFolder: () => void;
  onToggleDir: (path: string) => void;
  onOpenFile: (path: string) => void;
  onRefresh: () => void;
  onToggleIgnored: () => void;
  /** 新建文件 / 文件夹。返回 false 表示失败（原因已进树顶横幅），输入框留在原地。 */
  onCreateEntry: (dir: string, name: string, kind: 'file' | 'dir') => Promise<boolean>;
  /** 重命名。返回 false 同上。 */
  onRenameEntry: (path: string, newName: string) => Promise<boolean>;
  /**
   * 删除。**确认对话框在这一侧**（`App`）弹出：组件不持有平台对话框的引用，
   * 也不该替用户决定「警告文案里要不要说这是文件夹」。
   */
  onDeleteEntry: (path: string, isDir: boolean) => Promise<boolean>;
}

export function Sidebar({
  tree,
  activePath,
  canManageFiles,
  onOpenFolder,
  onToggleDir,
  onOpenFile,
  onRefresh,
  onToggleIgnored,
  onCreateEntry,
  onRenameEntry,
  onDeleteEntry,
}: SidebarProps) {
  const rows = useMemo(
    () => flattenTree(tree.dirs, tree.root, tree.expanded, tree.showIgnored),
    [tree.dirs, tree.root, tree.expanded, tree.showIgnored],
  );

  /** 正在进行的新建：目标目录 + 种类。非 null 时树上多出一行输入框。 */
  const [creating, setCreating] = useState<{ dir: string; kind: 'file' | 'dir' } | null>(null);
  /** 正在改名的路径。非 null 时那一行变成输入框。 */
  const [renaming, setRenaming] = useState<string | null>(null);

  const commitCreate = async (name: string): Promise<boolean> => {
    if (creating === null) return false;
    const ok = await onCreateEntry(creating.dir, name, creating.kind);
    if (ok) setCreating(null);
    return ok;
  };

  const commitRename = async (path: string, name: string): Promise<boolean> => {
    const ok = await onRenameEntry(path, name);
    if (ok) setRenaming(null);
    return ok;
  };

  return (
    <aside className="sidebar">
      <header className="sidebar-head">
        <span className="sidebar-title" title={tree.root ?? ''}>
          {tree.root === null ? '没有打开文件夹' : basename(tree.root)}
        </span>
        <span className="grow" />
        {tree.root !== null && (
          <>
            {canManageFiles && (
              <>
                <button
                  type="button"
                  title="在根目录新建文件"
                  onClick={() => setCreating({ dir: tree.root!, kind: 'file' })}
                >
                  ＋
                </button>
                <button
                  type="button"
                  title="在根目录新建文件夹"
                  onClick={() => setCreating({ dir: tree.root!, kind: 'dir' })}
                >
                  ⊞
                </button>
              </>
            )}
            <button type="button" title="刷新目录" onClick={onRefresh} disabled={tree.loading.length > 0}>
              ⟳
            </button>
            <button
              type="button"
              title={tree.showIgnored ? '隐藏 node_modules 等目录' : '显示 node_modules 等目录'}
              className={tree.showIgnored ? 'on' : ''}
              onClick={onToggleIgnored}
            >
              ◌
            </button>
          </>
        )}
        <button type="button" title="打开文件夹（Ctrl+Shift+O）" onClick={onOpenFolder}>
          ⌸
        </button>
      </header>

      {tree.error && <div className="sidebar-error">{tree.error.message}</div>}

      {tree.root === null ? (
        <div className="sidebar-empty">
          <p>打开一个文件夹，它的内容会显示在这里。</p>
          <button type="button" className="primary" onClick={onOpenFolder}>
            打开文件夹
          </button>
        </div>
      ) : (
        <div className="tree" role="tree">
          {creating !== null && creating.dir === tree.root && (
            <NameInputRow
              depth={0}
              placeholder={creating.kind === 'file' ? '新文件名（回车确认，Esc 取消）' : '新文件夹名（回车确认，Esc 取消）'}
              onCommit={commitCreate}
              onCancel={() => setCreating(null)}
            />
          )}
          {rows.map((row) => (
            <Fragment key={row.path}>
              {renaming === row.path ? (
                <NameInputRow
                  depth={row.depth}
                  initial={row.name}
                  onCommit={(name) => commitRename(row.path, name)}
                  onCancel={() => setRenaming(null)}
                />
              ) : (
                <TreeItem
                  row={row}
                  active={row.path === activePath}
                  canManage={canManageFiles}
                  onToggle={onToggleDir}
                  onOpen={onOpenFile}
                  onStartRename={() => setRenaming(row.path)}
                  onDelete={() => void onDeleteEntry(row.path, row.isDir)}
                  onCreateHere={(kind) => setCreating({ dir: row.path, kind })}
                />
              )}
              {/* 新建输入行紧跟目标目录：名字长在它将要出现的位置上 */}
              {creating !== null && row.isDir && creating.dir === row.path && (
                <NameInputRow
                  depth={row.depth + 1}
                  placeholder={creating.kind === 'file' ? '新文件名（回车确认，Esc 取消）' : '新文件夹名（回车确认，Esc 取消）'}
                  onCommit={commitCreate}
                  onCancel={() => setCreating(null)}
                />
              )}
            </Fragment>
          ))}
          {rows.length === 0 && creating === null && tree.loading.length === 0 && (
            <div className="sidebar-empty">
              <p>这个文件夹是空的（或只包含被折叠的目录）。</p>
            </div>
          )}
        </div>
      )}
    </aside>
  );
}

function TreeItem({
  row,
  active,
  canManage,
  onToggle,
  onOpen,
  onStartRename,
  onDelete,
  onCreateHere,
}: {
  row: TreeRow;
  active: boolean;
  canManage: boolean;
  onToggle: (path: string) => void;
  onOpen: (path: string) => void;
  onStartRename: () => void;
  onDelete: () => void;
  /** 仅目录行有：在这个目录里新建。 */
  onCreateHere: (kind: 'file' | 'dir') => void;
}) {
  return (
    <div
      role="treeitem"
      aria-expanded={row.isDir ? row.expanded : undefined}
      aria-selected={active}
      tabIndex={-1}
      className={`tree-row${active ? ' active' : ''}${row.isDir ? ' dir' : ''}`}
      // 缩进用内联 padding 而不是嵌套 DOM：树是扁平渲染的，
      // 嵌套 DOM 会让「重排时节点被卸载重建」发生得过于频繁（展开大目录时肉眼可见）
      style={{ paddingLeft: 8 + row.depth * 14 }}
      title={row.path}
      onClick={() => (row.isDir ? onToggle(row.path) : onOpen(row.path))}
    >
      <span className="tree-icon">{row.isDir ? (row.expanded ? '▾' : '▸') : ''}</span>
      <span className="tree-name">{row.name}</span>
      {/* 已展开但一个子项都没有的目录：给一个明确的说明，别让用户以为加载失败 */}
      {row.isDir && row.expanded && !row.loaded && <span className="tree-note">加载中…</span>}
      {canManage && (
        // 悬停才出现（见 styles.css）；stopPropagation 别让行点击抢走这一下
        <span className="tree-actions">
          {row.isDir && (
            <>
              <button
                type="button"
                title="在这新建文件"
                onClick={(event) => {
                  event.stopPropagation();
                  onCreateHere('file');
                }}
              >
                ＋
              </button>
              <button
                type="button"
                title="在这新建文件夹"
                onClick={(event) => {
                  event.stopPropagation();
                  onCreateHere('dir');
                }}
              >
                ⊞
              </button>
            </>
          )}
          <button
            type="button"
            title="重命名"
            onClick={(event) => {
              event.stopPropagation();
              onStartRename();
            }}
          >
            ✎
          </button>
          <button
            type="button"
            title="删除"
            onClick={(event) => {
              event.stopPropagation();
              onDelete();
            }}
          >
            ✕
          </button>
        </span>
      )}
    </div>
  );
}

/**
 * 新建 / 改名共用的行内输入行。
 *
 * 三条交互纪律：
 * - **挂载即聚焦**（改名时全选旧名）：既然行就在位置上，焦点就该在名字上；
 * - **空值提交 = 取消**：点别处关掉一个空输入框不该在树顶留一条「名称不能为空」；
 * - **提交进行中不再提交**：Enter 与失焦可能在同一帧里先后到达，
 *   没有这道闸会向 store 发两次同样的请求。
 */
function NameInputRow({
  depth,
  initial = '',
  placeholder,
  onCommit,
  onCancel,
}: {
  depth: number;
  /** 改名时传旧名（会被全选），新建时留空。 */
  initial?: string;
  placeholder?: string;
  onCommit: (name: string) => Promise<boolean>;
  onCancel: () => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [value, setValue] = useState(initial);
  const finishedRef = useRef(false);
  const busyRef = useRef(false);

  useEffect(() => {
    const input = inputRef.current;
    if (input === null) return;
    input.focus();
    input.select();
  }, []);

  const commit = (): void => {
    if (finishedRef.current || busyRef.current) return;
    const name = value.trim();
    // 空值提交（点别处关掉）按取消算 —— 不是一次要通报的失败
    if (name === '') {
      finishedRef.current = true;
      onCancel();
      return;
    }
    busyRef.current = true;
    void onCommit(name).then((ok) => {
      if (ok) finishedRef.current = true;
      else busyRef.current = false; // 失败：输入框留在原地，可以改名再试
    });
  };

  return (
    <div className="tree-row tree-input-row" style={{ paddingLeft: 8 + depth * 14 }}>
      <input
        ref={inputRef}
        value={value}
        placeholder={placeholder}
        spellCheck={false}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault();
            commit();
          } else if (event.key === 'Escape') {
            event.preventDefault();
            finishedRef.current = true;
            onCancel();
          }
        }}
        onBlur={commit}
      />
    </div>
  );
}
