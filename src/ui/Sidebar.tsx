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
 */

import { useMemo } from 'react';

import { basename } from '../core/path.js';
import { flattenTree, type TreeRow } from '../core/workspace/tree.js';
import type { TreeState } from '../core/workspace/store.js';

export interface SidebarProps {
  tree: TreeState;
  activePath: string | null;
  onOpenFolder: () => void;
  onToggleDir: (path: string) => void;
  onOpenFile: (path: string) => void;
  onRefresh: () => void;
  onToggleIgnored: () => void;
}

export function Sidebar({
  tree,
  activePath,
  onOpenFolder,
  onToggleDir,
  onOpenFile,
  onRefresh,
  onToggleIgnored,
}: SidebarProps) {
  const rows = useMemo(
    () => flattenTree(tree.dirs, tree.root, tree.expanded, tree.showIgnored),
    [tree.dirs, tree.root, tree.expanded, tree.showIgnored],
  );

  return (
    <aside className="sidebar">
      <header className="sidebar-head">
        <span className="sidebar-title" title={tree.root ?? ''}>
          {tree.root === null ? '没有打开文件夹' : basename(tree.root)}
        </span>
        <span className="grow" />
        {tree.root !== null && (
          <>
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
          {rows.map((row) => (
            <TreeItem
              key={row.path}
              row={row}
              active={row.path === activePath}
              onToggle={onToggleDir}
              onOpen={onOpenFile}
            />
          ))}
          {rows.length === 0 && tree.loading.length === 0 && (
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
  onToggle,
  onOpen,
}: {
  row: TreeRow;
  active: boolean;
  onToggle: (path: string) => void;
  onOpen: (path: string) => void;
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
    </div>
  );
}
