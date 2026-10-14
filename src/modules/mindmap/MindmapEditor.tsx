/**
 * 思维导图内核（mind-elixir 5.15.1）的 React 挂载组件。
 *
 * 结构与 `CodeEditor.tsx` / `MarkdownEditor.tsx` **同构**（挂载点、生命周期、回调转发）。
 * 这不是巧合，是适配层契约在起作用：如果第三个内核不得不写一套不一样的东西，
 * 那说明契约漏了东西，该去改契约而不是在这里绕。
 *
 * ## 两条变更通道，缺一不可
 *
 * 1. **主通道**：内核的 `operation` / `expandNode` / `changeDirection` 事件。
 *    这是内核**主动**告诉我们「数据变了」。
 * 2. **兜底通道**：`linkDiv` 事件 + 防抖后的快照比对。
 *
 * 为什么需要第二条：内核有**改了数据却不报事件**的操作。
 * 最典型的是 `expandNodeAll`（`Ctrl/Cmd + 点击折叠标记`，递归展开整棵子树）——
 * 它走的是 `refresh()` 那条路，**一个事件都不发**（读过 `MindElixir.js` 确认）。
 * 只挂第一条通道，用户展开整棵树之后保存，`collapsed` 会全部回到展开状态，
 * 而界面上看不出任何异常 —— 正是硬约束 2 要拦的形态。
 *
 * 兜底通道的代价被防抖吸收掉了：`linkDiv` 只在布局重画时触发，
 * 拖拽过程中频繁触发也只会在**停手 300ms 后**做一次比对。
 *
 * ## 为什么 `getValue()` 会「回退到原始文本」
 *
 * 见 `getValue` 里的注释 —— 那是 §7.2 第 4 条（`.drawio` 往返不篡改）对
 * `.mindmap` 的同一条要求：**没改过的文件，写回去必须逐字节一致**。
 * JSON 往返做不到这件事（键顺序、缩进、空格都会变），所以用「没改就交回原文」来保证。
 *
 * ## 切语言要重建内核（ADR 0049）
 *
 * 内核把右键菜单在**安装那一刻**就建成了 DOM：`dist/MindElixir.js:1394` 是
 * `E.innerHTML = \`<span>${文案}</span>\``，文案在那一刻就烤死了。
 * 所以「换了语言包对象」对已经建好的菜单**毫无作用** —— 只能重建内核。
 *
 * 于是 `locale` 进了下面那个 effect 的依赖数组。三件事得同时守住，重建才是划算的：
 *
 * 1. **不丢内容**（`latestRef`）。种子取「此刻保存会写出的那份文本」而不是
 *    `initialValue` —— 后者会让用户在这份文件上已经做的修改**静默回滚**：
 *    界面还画着改过的图，一保存就弹回打开时的样子。
 * 2. **不弄脏文件**（`getValue` 里写的是**交出去的那个值**）。种子一旦被
 *    规范形式污染，「打开 → 保存逐字节一致」这条当场破掉。
 * 3. **不让人以为图变了**（`scaleRef`）。不还原缩放的话，切一次语言视口就跳一下。
 *
 * 重建换来的好处是：这里可以放心用 `t()` 而**不必** `msg()` + `textOf()`。
 * ADR 0047 的判据是「这句译文会不会跨过下一次 `setLocale`」——在这里**不会**，
 * 因为跨过去之前内核已经重建了。重建是那条判据的兑现，不是它的副作用。
 * 没保住的三样如实写在这儿：**选中项、专注模式、平移位置**都不是公开 API，
 * 重建后回到初始状态（缩放与折叠状态是保住的）。
 */

import MindElixir, { type MindElixirInstance, type Operation } from 'mind-elixir';
import 'mind-elixir/style';
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from 'react';

import type { EditorHandle, EditorMountProps, EditorSearch } from '../../core/editor/types.js';
import { getLocale, msg, subscribeLocale, t, type LocalizedText } from '../../core/i18n/index.js';
import { parseMindmap } from '../../core/mindmap/parse.js';
import { serializeMindmap } from '../../core/mindmap/serialize.js';
import { emptyMindmapDocument } from '../../core/mindmap/template.js';
import { type LineEnding, type MindmapDocument, type MindmapImage, type MindmapNode, type MindmapTag } from '../../core/mindmap/types.js';
import { createFindPattern, findInText, replaceInText, type FindOptions } from '../../core/search/find.js';
import { directionOf, fromKernelData, toKernelData } from './adapter.js';
import { mindmapLangPack } from './lang-pack.js';
import { applyMarkerToggle, MINDMAP_MARKER_GROUPS, MINDMAP_MARKERS } from './markers.js';
import {
  activeFontSize,
  activeSwatchIndex,
  isBold,
  STYLE_FILL_COLORS,
  STYLE_FONT_SIZES,
  STYLE_TEXT_COLORS,
  type StyleKey,
  type StyleValues,
} from './node-style.js';
import {
  bytesToDataUrl,
  clampDisplaySize,
  collectImageNodeIds,
  collectImageUrls,
  decodeImageSize,
  extractImageFile,
  placeholderImage,
  prewarmImageCache,
} from './node-images.js';
import './mindmap.css';

/**
 * 兜底通道的防抖时长。
 *
 * 300ms 是「用户停手」与「多等一会儿」之间的折中：拖拽过程中 `linkDiv` 会被调很多次，
 * 每次都比对一遍是白烧 CPU；而 300ms 之后用户还在动，下一次 `linkDiv` 会重新计时。
 */
const VERIFY_DELAY = 300;

/**
 * 空文件长什么样。
 *
 * 新建一个 `.mindmap` 时磁盘上是 0 字节 —— 那不是一个「坏文件」，
 * 是「还没开始写」。给它一个中心主题，比让用户面对一个空白的纯文本编辑器友好。
 *
 * 注意 `canHandle` 那边**必须**放行空内容，否则这条走不到（见 `index.ts`）。
 *
 * 内容本身交给 `core/mindmap/template.ts` —— 与「新建文件时该写什么」
 * （`core/new-file-content.ts`）共用**同一份**构造，只差中心主题是谁
 * （新建时是用户自己刚起的文件名，这里是空文件的占位主题）。
 * 两份各自抄一份字段，格式演进时就会有一份先掉队。
 */
function createEmptyDocument(): MindmapDocument {
  return emptyMindmapDocument('中心主题');
}

type KernelTopic = ReturnType<MindElixirInstance['findEle']>;

/**
 * 一次查找命中的位置。**偏移是段内的**（相对那个节点的那一段文字），不是整篇的。
 * `field` 说的是命中落在哪段 —— `replace` 靠它把写回送到正确的通道
 * （topic 走 setNodeTopic，note 走 reshapeNode，ADR 0058）。
 */
interface TopicHit {
  id: string;
  from: number;
  to: number;
  field: 'topic' | 'note';
}

function findNodeById(root: MindmapNode, id: string): MindmapNode | null {
  if (root.id === id) return root;
  for (const child of root.children) {
    const found = findNodeById(child, id);
    if (found !== null) return found;
  }
  return null;
}

/**
 * 大纲跳转的目标定位（ADR 0052）：找到目标节点，连从根到它的整条祖先链一起交回 ——
 * 展开折叠祖先时要知道「谁挡在路上」，只找到节点本身不够。
 *
 * 匹配规则与契约注释一致：带 `id` 优先按 `id`（主题可以重名，`(level, text)`
 * 只能找到第一个）；没带就按「层级 + 文字」找第一个。返回的链是先序，
 * `[0]` 是根，最后一项是目标本身。
 */
function findTopicPath(root: MindmapNode, target: { id?: string; level: number; text: string }): MindmapNode[] | null {
  const seek = (node: MindmapNode, prefix: readonly MindmapNode[]): MindmapNode[] | null => {
    const path = [...prefix, node];
    const matched =
      target.id !== undefined ? node.id === target.id : path.length === target.level && node.topic === target.text;
    if (matched) return path;
    for (const child of node.children) {
      const hit = seek(child, path);
      if (hit !== null) return hit;
    }
    return null;
  };
  return seek(root, []);
}

/**
 * 在所有节点的 `topic` 和 `note` 里查找（note 自 ADR 0058 起一并搜）。
 *
 * ## 为什么逐节点查找，而不是拼成一篇大文本再查
 *
 * 拼成一篇会**产生跨节点的匹配**：「A 结尾的字符 + B 开头的字符」会被当成一处命中，
 * 而那一处在界面上根本不存在（两个节点之间隔着连线和留白）。
 * 逐节点查还顺带解决了偏移映射 —— 每个匹配的偏移天然就是它所属节点、
 * 所属字段的段内偏移，不需要维护一张「段起点表」。
 *
 * ## topic 先、note 后（同一节点内）
 *
 * 两条命中的偏移各自以自己那段文字为基准，塞进同一张表也没关系 ——
 * 前提是 `replace` 按 `field` 把写回分开路由（topic 走 setNodeTopic、
 * note 走 reshapeNode）。ADR 0056 只搜 topic 时登记的边界，本轮补的就是
 * 路由这一半：搜得到也换得了，才不是半成品。导航顺序仍是文档顺序
 * （先序遍历，节点内 topic 命中排在它的 note 命中前面）。
 */
function collectHits(root: MindmapNode, options: FindOptions): TopicHit[] {
  const hits: TopicHit[] = [];
  const walk = (node: MindmapNode): void => {
    for (const match of findInText(node.topic, options)) {
      hits.push({ id: node.id, from: match.from, to: match.to, field: 'topic' });
    }
    if (typeof node.note === 'string' && node.note !== '') {
      for (const match of findInText(node.note, options)) {
        hits.push({ id: node.id, from: match.from, to: match.to, field: 'note' });
      }
    }
    for (const child of node.children) walk(child);
  };
  walk(root);
  return hits;
}

/**
 * 主题桥接。
 *
 * 内核把 `cssVar` 里的键**内联**写到它自己的容器上（`container.style.setProperty`），
 * 所以外部 CSS 覆盖不了它 —— 必须从这个入口进去。
 * 值一律写成轻笺的 CSS 变量（而不是色值），于是深色主题下它自己就跟着变了，
 * 这个文件里永远不需要出现 `#rrggbb`。
 *
 * `type` 只是未覆盖变量的兜底底色（内核会在它之上叠 `cssVar`）；
 * `palette` 是主分支的取色环，用轻笺的品牌色。
 */
const LEVNOTE_THEME = {
  name: 'LevNote',
  type: 'light' as const,
  palette: ['#2e7d6a', '#3f6ea8', '#8a5a86', '#e2603f', '#7a8690', '#b4d4c6', '#16302a', '#209fb5'],
  cssVar: {
    '--bgcolor': 'var(--surface)',
    '--color': 'var(--ink)',
    '--main-color': 'var(--line-strong)',
    '--main-bgcolor': 'var(--surface)',
    '--main-bgcolor-transparent': 'var(--surface)',
    '--root-bgcolor': 'var(--bamboo)',
    '--root-color': '#ffffff',
    '--root-border-color': 'transparent',
    '--selected': 'var(--accent)',
    '--accent-color': 'var(--vermilion)',
    '--panel-color': 'var(--ink)',
    '--panel-bgcolor': 'var(--surface-raised)',
    '--panel-border-color': 'var(--line)',
  },
};

/**
 * 引用**外部文件**的节点图片数（`src` 是 `data:` 的会内嵌进导出的图里，
 * 其余的内核只会原样写一个引用）。
 *
 * 这是「绝不静默」的又一次兑现：外链图片导出去之后**不再是自包含的**
 * （PNG 里干脆画不出来），用户必须知道 —— 数出来交给 `exportImage` 的
 * `warnings`，而不是装作没看见。
 */
function countExternalImages(nodes: HTMLElement): number {
  let count = 0;
  for (const image of nodes.querySelectorAll('img')) {
    const src = image.getAttribute('src') ?? '';
    if (src !== '' && !src.startsWith('data:')) count += 1;
  }
  return count;
}

/**
 * 导出用的主题：把 `--bgcolor` 换成**此刻真实解析出的颜色**。
 *
 * 内核导出 SVG 时把 `theme.cssVar['--bgcolor']`（`'var(--surface)'`）**原样**
 * 写进底板矩形的 `fill` —— 而一份独立打开的 SVG 里没有 `--surface` 的定义，
 * `var()` 解析不出来，底板会黑掉。浏览器的自定义属性**计算值**会把 `var()`
 * 替换成具体色值（跟随当前浅色 / 深色主题），取那个。
 *
 * 解析不出来（空串、或还含着 `var(` —— 测试环境没有层叠样式就是这样）
 * 就**保留原样**：宁可导出底色可疑也不编一个颜色糊弄过去。
 *
 * 返回**新对象**而不是原地改：`mind.theme` 与模块级 `LEVNOTE_THEME` 是同一个
 * 引用，原地改会污染它，下一个实例就被牵连了。调用方负责用完还原。
 */
function withResolvedBackground(
  theme: MindElixirInstance['theme'],
  container: HTMLElement,
): MindElixirInstance['theme'] {
  const resolved = getComputedStyle(container).getPropertyValue('--bgcolor').trim();
  if (resolved === '' || resolved.includes('var(')) return theme;
  return { ...theme, cssVar: { ...theme.cssVar, '--bgcolor': resolved } };
}

/**
 * 备注徽章（ADR 0056）：`note` 内核零渲染 —— 数据里有、界面上一字不画
 * （Se 只画 text / icons / image / hyperLink / tags 五样）。徽章是它唯一
 * 看得见的形态；**创建**走右键「备注…」（还没有徽章可点），**改 / 看**
 * 走点徽章 —— 两个入口进的是同一个弹层。
 *
 * 幂等同步：内核每次重画（`linkDiv`：拖拽 / 展开 / undo / redo /
 * `reshapeNode` 都会触发）后跑一遍 —— 有备注的节点补徽章，没有的摘掉。
 * 徽章是 `me-tpc` 里的**兄弟元素**，不在内核认识的任何容器里，Se 重建
 * 时 `innerHTML = ""` 自然带走它，下一次同步再补 —— 内核从不经手它，
 * 也不需要。导出器按选择器挑元素，不认识 `button.mm-note-badge`，
 * 不会被带进 SVG / PNG —— 导出的是文件内容，徽章是编辑器部件
 * （与右键菜单同理，ADR 0056 写明了这条边界）。
 *
 * 点击读的是**此刻的 `nodeObj`** 而不是同步那一刻的快照：徽章可能比数据
 * 旧（弹层刚保存过、undo 刚回滚过），`nodeObj` 永远是真值 —— DOM 会
 * 说谎的教训，与 ADR 0055 的 🔗 同一条。空串不画徽章：空备注没有可
 * 指认的内容。
 */
function syncNoteBadges(nodes: HTMLElement, open: (nodeId: string, note: string, x: number, y: number) => void): void {
  for (const tpc of Array.from(nodes.querySelectorAll('me-tpc'))) {
    const note = (tpc as KernelTopic).nodeObj.note;
    const badge = tpc.querySelector('button.mm-note-badge');
    if (typeof note !== 'string' || note === '') {
      badge?.remove();
      continue;
    }
    if (badge !== null) continue;
    const created = document.createElement('button');
    created.type = 'button';
    created.className = 'mm-note-badge';
    created.textContent = '📝';
    created.title = t('ui.mindmap.noteBadgeTitle');
    created.addEventListener('click', (event) => {
      const owner = created.closest('me-tpc');
      if (owner === null) return;
      const current = (owner as KernelTopic).nodeObj.note;
      open((owner as KernelTopic).nodeObj.id, typeof current === 'string' ? current : '', event.clientX, event.clientY);
    });
    tpc.appendChild(created);
  }
}

/**
 * 数字字号的内联补丁（ADR 0059）。
 *
 * 内核渲染把 `style` 每个键**原样**塞给 CSSOM（`Se`，`MindElixir.js:344-347`），
 * 而 CSSOM 对**无单位的非零长度**按无效值忽略 —— 文件里存的
 * `"fontSize": 20` 因此在编辑器里画不出来；可内核的导出侧偏偏要的就是数字
 * （`LayoutSsr.js:47` 拼 `font-size: ${v}px`，字符串会被拼成非法值）。
 * 两头都要对，只能在这里补一次 `px`。
 *
 * 时机与备注徽章同一条（ADR 0056）：内核每次重画都会把内联样式重来一遍
 * （`refresh` 重建元素、`reshapeNode` 重跑 `Se`），所以挂在 `linkDiv` 上重放
 * —— 初始化一次 + 每次重画一次。带单位的字符串值（别的工具写的 `'20px'`）
 * 内核自己画得对，这里不碰。
 */
function syncStyleFontSizes(nodes: HTMLElement): void {
  for (const tpc of Array.from(nodes.querySelectorAll('me-tpc'))) {
    const topic = tpc as KernelTopic;
    const style = topic.nodeObj.style as Record<string, unknown> | undefined;
    const size = style?.fontSize;
    if (typeof size === 'number' && Number.isFinite(size)) topic.style.fontSize = `${size}px`;
  }
}

/**
 * camelCase 的样式键 → CSS 属性名。
 *
 * `removeProperty` 只认 CSS 属性名（`font-size`），不认 IDL 属性名（`fontSize`）
 * —— 拿 camelCase 去删会**静默什么都不删**。同一条转换内核自己也写了一遍
 * （`LayoutSsr.js:62` 给标签样式用）。
 */
function cssPropertyName(key: string): string {
  return key.replace(/[A-Z]/g, (char) => `-${char.toLowerCase()}`);
}

function MindmapEditor({ initialValue, onChange, onReady, onError, readOnly, resolveImageUrl, onImagePaste, onLinkClick }: EditorMountProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);

  // 回调每次渲染都是新函数，直接进依赖会让内核被反复重建
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;
  // 图片契约同一条纪律（照 `MarkdownEditor` 的 ref 转发）：闭包不烤死，
  // 内核 effect 里拿到的永远是「此刻」的那一份。
  const resolveImageUrlRef = useRef(resolveImageUrl);
  resolveImageUrlRef.current = resolveImageUrl;
  const onImagePasteRef = useRef(onImagePaste);
  onImagePasteRef.current = onImagePaste;
  // 链接激活契约（ADR 0055）同一条纪律：click 监听挂在内核 effect 里，
  // 拿的必须是「此刻」的外壳回调。
  const onLinkClickRef = useRef(onLinkClick);
  onLinkClickRef.current = onLinkClick;

  const initialValueRef = useRef(initialValue);
  const readOnlyRef = useRef(readOnly);

  /**
   * 语言。订阅在组件里，而不是等外壳那一圈 `localeTick` 顺带把内核刷一遍：
   * 内核不受 `memo` 保护，也不该依赖父组件有没有重渲染（那是一条看不见的耦合）。
   *
   * 这是全项目**第一处**真的订阅语言变化的消费方 —— 在此之前每个界面都在
   * 渲染那一刻现取 `t()`，所以切语言不需要谁通知它们。
   */
  const locale = useSyncExternalStore(subscribeLocale, getLocale);

  /**
   * 「此刻保存会写出的那份文本」，**跨重建**活着。
   *
   * 重建时不能拿 `initialValue` 重新解析：那样用户在这份文件上已经做的修改
   * 会被**静默回滚** —— 界面还画着改过的图，一保存就弹回打开时的样子。
   * 约定只有一条：**任何时刻它都等于「此刻 Ctrl+S 会写出的内容」**，
   * 所以 `emit` / `setValue` / `getValue` 三处都维护它，而重建直接拿它当种子。
   *
   * 头一次挂载时是 `null`，那时 `initialValue` 就是种子（用户还没机会改）。
   */
  const latestRef = useRef<string | null>(null);

  /**
   * 上一个内核的缩放，重建后还回去。
   *
   * `init()` 收尾会 `toCenter()`，不还原的话「切一次语言视口就跳一下」。
   * `scaleVal` 与 `scale()` 都是内核的公开 API（`dist/types/index.d.ts:68`、
   * `dist/types/methods.d.ts`），不是撬出来的内部状态。
   */
  const scaleRef = useRef(1);

  /**
   * 标记调色盘（ADR 0053）。右键菜单的「标记…」打开它；它是**内核模块自己的界面**
   * （与右键菜单同一条纪律），不进 `EditorHandle` —— 外壳没有对应的命令，
   * 也就没有理由为它开一条契约。
   *
   * `markerApplyRef` / `markerClearRef` 在下面的 effect 里被赋值，闭住**当前这颗
   * 内核**：语言切换会换内核（文件头三命门），ref 跟着换新函数，调色盘不用
   * 因此关掉 —— 下一次点击自然落在新的内核上。
   */
  const [markerPalette, setMarkerPalette] = useState<MarkerPaletteState | null>(null);
  const markerApplyRef = useRef<((nodeId: string, glyph: string) => readonly string[] | null) | null>(null);
  const markerClearRef = useRef<((nodeId: string) => readonly string[] | null) | null>(null);

  /**
   * 图片弹层（ADR 0054），与标记调色盘同一条纪律：内核模块自己的界面，
   * 不进 `EditorHandle`。`imageRemoveRef` 同 `markerApplyRef`：effect 里
   * 装上、闭住当前这颗内核，cleanup 时下岗。
   */
  const [imagePopover, setImagePopover] = useState<ImagePopoverState | null>(null);
  const imageRemoveRef = useRef<((nodeId: string) => boolean) | null>(null);

  /**
   * 链接 / 标签弹层（ADR 0055），与前两条内核自有 UI 同一条纪律：不进
   * `EditorHandle`，落盘 ref 在 effect 里装、cleanup 时下岗。标签的增删
   * 照 `markerApplyRef` 的先例**从内核现取现算**（不依赖弹层快照），返回
   * 新快照供弹层回显 —— 快照只负责「列出来」，真值永远在内核的
   * `nodeObj` 里。
   */
  const [linkPopover, setLinkPopover] = useState<LinkPopoverState | null>(null);
  const linkApplyRef = useRef<((nodeId: string, url: string | undefined) => boolean) | null>(null);
  const [tagPopover, setTagPopover] = useState<TagPopoverState | null>(null);
  const tagAddRef = useRef<((nodeId: string, text: string) => readonly MindmapTag[] | null) | null>(null);
  const tagRemoveRef = useRef<((nodeId: string, index: number) => readonly MindmapTag[] | null) | null>(null);

  /**
   * 备注弹层（ADR 0056），与前三条内核自有 UI 同一条纪律：不进 `EditorHandle`。
   * `noteApplyRef` 同 `linkApplyRef`：effect 里装上、闭住当前内核，cleanup 下岗。
   * 保存与移除都走 `reshapeNode({ note })` → `operation` → 既有 emit 主通道。
   */
  const [notePopover, setNotePopover] = useState<NotePopoverState | null>(null);
  const noteApplyRef = useRef<((nodeId: string, note: string | undefined) => boolean) | null>(null);

  /**
   * 样式弹层（ADR 0059），与前四条内核自有 UI 同一条纪律：不进 `EditorHandle`。
   * 落盘 ref 在 effect 里装、闭住当前内核，cleanup 下岗；两个出口都返回
   * **新的样式快照**供弹层回显（连挂多键是正常用法，亮着的色块必须跟着走）。
   */
  const [stylePopover, setStylePopover] = useState<StylePopoverState | null>(null);
  const styleApplyRef = useRef<((nodeId: string, key: StyleKey, value: string | number) => StyleValues | null) | null>(null);
  const styleClearRef = useRef<((nodeId: string) => StyleValues | null) | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    /**
     * 种子：第一次挂载是文件原文，之后每一次重建都是「此刻保存会写出的那份文本」
     * （见 `latestRef`）。走后者是切语言不丢内容的那一半。
     */
    const seed = latestRef.current ?? initialValueRef.current;

    // 空内容给一份默认文档；其余情况读不出来就是真读不出来 ——
    // 那条路 `canHandle` 已经拦掉了，走到这里说明有个我们没预料到的洞，
    // 如实报错好过白屏（「打开文件不能失败」这条承诺在注册表那一层兑现）。
    const empty = seed.trim() === '';
    const parsed = empty
      ? { document: createEmptyDocument(), lineEnding: '\n' as LineEnding }
      : parseMindmap(seed);
    const first = parsed.document;
    if (first === null) {
      onErrorRef.current?.(new Error('这不是一份能读的 .mindmap（缺少 root 或不是合法 JSON）'));
      return;
    }

    /**
     * 「原文」—— 只在用户没动过时交回它，好让「打开 → 保存」逐字节一致。
     *
     * **空文件时是 `null`**：那种情况我们并没有「原样交回」可言（原文是 0 字节），
     * 而用户眼前明明有一张带中心主题的图。交回空串等于「打开一份空白文件、
     * 看到一张图、保存后图没了」—— 一次静默丢内容。
     */
    const originalText = empty ? null : seed;
    const lineEnding = parsed.lineEnding;

    /**
     * 图片预热表（ADR 0054）：文件里的引用 → `data:` URL。挂在 effect 作用域：
     * `boot` 与粘贴通道、`setValue` 共享 —— 粘贴成功后往里补新条目，
     * 内核重渲染时 `imageProxy` 同步查表。
     */
    const imageCache = new Map<string, string>();
    let teardown: (() => void) | null = null;
    let cancelled = false;

    /**
     * 装内核（三条命门的注释在文件头）。预热完才调它；没有图片引用时立刻调。
     *
     * 返回 teardown 而不是把清理写在 effect 的 `return` 里，是因为**预热是异步的**：
     * cleanup 完全可能抢在装内核之前发生（卸载组件、或语言快速切了两下），
     * 那时什么都没装过，也就什么都不用拆 —— `teardown` 为 `null` 就是这个
     * 状态；装过了就一定拆。预热表填好之前就装内核的话，第一帧画出来的全是
     * 占位 —— 占位是「图不在这里」的如实陈述，不是「还没读完盘」的遮羞布。
     */
    const boot = (): (() => void) => {
      let document = first;

      let mind: MindElixirInstance;

      /**
       * 右键菜单「标记…」的 handler（ADR 0053）。三件事：
       *
       * 1. **自己藏菜单** —— 内核对 `extend` 项只转发事件、不藏（内置项各自的
       *    handler 里才有 `v.hidden = true`，见 `MindElixir.js:1399-1401`）；
       * 2. **记下右键到的节点** —— 菜单点击不改选中，`currentNode` 就是它
       *    （`showContextMenu` 之前内核已把它选上，`MindElixir.js:1202`）；
       *    根节点也一样能标 —— 内核对根只禁内置项，不禁 `extend` 项；
       * 3. 把**点击坐标**交给调色盘定位 —— 屏幕坐标，调色盘是 `position: fixed`。
       */
      const openMarkerPalette = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        setMarkerPalette({
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          icons: [...(topic.nodeObj.icons ?? [])],
        });
      };

      /**
       * 右键菜单「图片…」的 handler（ADR 0054），藏菜单 / 记节点 / 传坐标
       * 三件事与 `openMarkerPalette` 完全同构。
       *
       * 预览图在这里就解析好（占位也在这里烘焙）：弹层是纯展示，不该
       * 碰预热表之外的东西；预览用的就是这个节点此刻真正画着的那张图。
       */
      const openImagePopover = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        const raw = topic.nodeObj.image;
        if (raw === undefined) {
          setImagePopover({ kind: 'hint', x: event.clientX, y: event.clientY, nodeId: topic.nodeObj.id });
          return;
        }
        const preview =
          raw.url.startsWith('data:') || imageCache.has(raw.url)
            ? (imageCache.get(raw.url) ?? raw.url)
            : placeholderImage(raw.url);
        setImagePopover({
          kind: 'attached',
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          image: { url: raw.url, width: raw.width, height: raw.height },
          preview,
        });
      };

      /**
       * 右键菜单「链接…」的 handler（ADR 0055），藏菜单 / 记节点 / 传坐标
       * 三件事同构。`url` 是此刻的链接原文（没有就是空串）—— 弹层据此
       * 出「保存」还是「保存 + 移除」两套按钮，一个字段说得清。内核的
       * `hyperLink` 就是纯字符串，没有 title 之类的伴生数据可快照。
       */
      const openLinkPopover = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        const raw = topic.nodeObj.hyperLink;
        setLinkPopover({
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          url: typeof raw === 'string' ? raw : '',
        });
      };

      /**
       * 右键菜单「标签…」的 handler（ADR 0055），同构。`tags` 是**打开那
       * 一刻的快照**，只负责弹层里「列出来」；真正落盘的增删在 ref 通道里
       * 从内核的 `nodeObj` 现取现算（`markerApplyRef` 的先例），快照过期
       * 也不会写错。
       */
      const openTagPopover = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        setTagPopover({
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          // 断言一次（内核 `TagObj.style` 的类型写宽为运行时描述，见
          // `types.ts` 里 `MindmapTagObj` 的注释）：`getData()` 是 JSON
          // 深拷贝值，只可能是 JSON 值。
          tags: [...(topic.nodeObj.tags ?? [])] as MindmapTag[],
        });
      };

      /**
       * 右键菜单「备注…」的 handler（ADR 0056），与前四个同构。它是备注
       * 的**创建入口**：还没有备注的节点没有徽章可点，从零写第一条备注
       * 只能走这里。`note` 是打开那一刻的快照（没有就是空串），弹层据此
       * 出「保存」还是「保存 + 移除」两套按钮；落盘在 ref 通道里现取
       * `nodeObj`，快照过期也不会写错（`tagAddRef` 的先例）。
       */
      const openNotePopoverMenu = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        const raw = topic.nodeObj.note;
        setNotePopover({
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          note: typeof raw === 'string' ? raw : '',
        });
      };

      /**
       * 右键菜单「样式…」的 handler（ADR 0059），与前五个同构：藏菜单 / 记节点 /
       * 传坐标。`style` 是**打开那一刻的快照**，只负责弹层里「亮哪个」；
       * 真正落盘在 ref 通道里从内核的 `nodeObj` 现取现算（`markerApplyRef`
       * 的先例），快照过期也不会写错。
       */
      const openStylePopover = (event: MouseEvent): void => {
        const menu = event.target instanceof Element ? event.target.closest('.context-menu') : null;
        if (menu !== null) menu.setAttribute('hidden', '');
        const topic = mind.currentNode;
        if (topic == null) return;
        const raw = topic.nodeObj.style as Record<string, unknown> | undefined;
        setStylePopover({
          x: event.clientX,
          y: event.clientY,
          nodeId: topic.nodeObj.id,
          style: { ...(raw ?? {}) },
        });
      };

      /**
       * 粘贴图片（ADR 0054）。内核的 paste 监听先吃「复制的节点」那份 magic
       * JSON（`MindElixir.js:1016-1027`），吃不下才把**原始事件**交到这里 ——
       * 所以节点复制粘贴的既有行为一个字节都不受影响。
       *
       * 五步，每一步的失败都有归属：
       * 1. 只读直接不管（粘贴是编辑）；剪贴板里没有图片也不管（可能是
       *    别的内核/别的场景的粘贴，别抢事件）；
       * 2. 没有选中的主题 ⇒ 报 `imageNoTarget`（内核层知道的事）；
       * 3. `onImagePaste` 交回 `null` ⇒ **静默返回** —— 拒绝的原因只有外壳
       *    知道，它已经 notice 过了；内核再报一次就是两条消息（契约原话）；
       * 4. 解码不出尺寸 ⇒ 报 `imageDecodeFailed`，**什么都没挂上**；
       * 5. `findEle` / `reshapeNode` 抛错（目标刚被删/折叠）⇒ 报 `imageAttachFailed`。
       *
       * 成功的最后两步：把字节转成 `data:` URL 补进预热表（这一会话立刻可见，
       * 保存后再打开则走预热那条正路），然后 `reshapeNode` 落盘 —— 它
       * fire 的 `operation` 落进 emit 主通道，标脏 / 保存不用单独伺候
       * （与标记同一条路，ADR 0053 的先例）。
       */
      const onPaste = (event: ClipboardEvent): void => {
        if (readOnlyRef.current) return;
        const file = extractImageFile(event.clipboardData);
        if (file === null) return;
        event.preventDefault();
        const topic = mind.currentNode;
        if (topic == null) {
          onErrorRef.current?.(new Error(t('ui.mindmap.imageNoTarget')));
          return;
        }
        const topicId = topic.nodeObj.id;
        void (async () => {
          try {
            const bytes = await file.arrayBuffer();
            const src = await onImagePasteRef.current?.({ bytes, mimeType: file.type });
            if (src === null || src === undefined) return;
            const size = await decodeImageSize(bytes, file.type);
            if (size === null) {
              onErrorRef.current?.(new Error(t('ui.mindmap.imageDecodeFailed')));
              return;
            }
            const display = clampDisplaySize(size.width, size.height);
            imageCache.set(src, await bytesToDataUrl(bytes, file.type));
            // 落盘前重找一次元素：等这几步异步的工夫里节点可能被删/折叠，
            // `findEle` 对看不见的节点会 throw（ADR 0052 同款）——抓住、
            // 如实报「没挂上」，好过把图片挂进一个已经不存在的节点。
            const el = mind.findEle(topicId);
            await mind.reshapeNode(el, { image: { url: src, width: display.width, height: display.height } });
          } catch {
            onErrorRef.current?.(new Error(t('ui.mindmap.imageAttachFailed')));
          }
        })();
      };

      try {
        mind = new MindElixir({
          el: host,
          direction: directionOf(document.layout),
          // 只读时三样一起关：`editable` 管拖拽与新建，`contextMenu` 管右键菜单，
          // `keypress` 管 F2 / Enter 这些进编辑态的键。**编辑节点文字的入口只有 `keypress`**
          // （`editTopic` 本身不检查 `editable`，是调用它的地方检查的），
          // 所以这三个必须成套关 —— 少关一个就能改到内容。
          editable: !readOnlyRef.current,
          // 右键菜单：只读时整个关掉；能编辑时传**对象**而不是 `true`——
          // 传 `true` 用的是内核硬编码的英文包（`MindElixir.js:1379` 的 `Yn`）。
          // 传对象才能带上我们的语言包（ADR 0049）。`extend` 里的「标记…」
          // 「图片…」「链接…」「标签…」「备注…」追加在菜单末尾
          // （ADR 0053 / 0054 / 0055 / 0056），label 走 `t()` ——
          // 语言一变内核重建，这里自然跟着换语言，与 ADR 0049 同一条命门。
          contextMenu: readOnlyRef.current
            ? false
            : {
                focus: true,
                link: true,
                locale: mindmapLangPack(),
                extend: [
                  { name: t('ui.mindmap.markerMenu'), onclick: openMarkerPalette },
                  { name: t('ui.mindmap.imageMenu'), onclick: openImagePopover },
                  { name: t('ui.mindmap.linkMenu'), onclick: openLinkPopover },
                  { name: t('ui.mindmap.tagMenu'), onclick: openTagPopover },
                  { name: t('ui.mindmap.noteMenu'), onclick: openNotePopoverMenu },
                  { name: t('ui.mindmap.styleMenu'), onclick: openStylePopover },
                ],
              },
          keypress: !readOnlyRef.current,
          // 新建节点时的占位文字，同属内核那套界面文案（默认是 `"New Node"`）。
          newTopicName: t('ui.mindmap.newTopicName'),
          /**
           * 图片显示（ADR 0054）：内核每次渲染带图节点都会来问一次
           * （`MindElixir.js:357` 的 `o.src = this.imageProxy ? … : …`），
           * 所以查表必须**同步** —— 这也是预热为什么在装内核**之前**：
           * `data:` 内嵌图自包含，直接透传；文件引用查预热表；
           * 查不到（图丢了 / 是外链，ADR 0001 不去 fetch）给**带路径的占位**。
           * 三条出口都给得出东西，`src` 永远不为空。
           */
          imageProxy: (url: string): string =>
            url.startsWith('data:') ? url : (imageCache.get(url) ?? placeholderImage(url)),
          // 粘贴图片的入口（上面 `onPaste` 的注释说了整条链路）。
          pasteHandler: onPaste,
          // 内核自己的撤销栈：导图的编辑动作（新建 / 删除 / 拖动 / 改文字）都在内核里，
          // 外壳的撤销栈够不着它们，所以交给内核自己管。
          allowUndo: true,
          // 内核的右下角浮动工具条（缩放、回到中心）保留：它是缩放唯一的可见入口。
          toolBar: true,
          theme: LEVNOTE_THEME,
        });
        mind.init(toKernelData(document));
        // 把上一个内核的缩放还回去（`init()` 收尾会 `toCenter()`）。
        // 头一次挂载时 `scaleRef` 是 1，这句不动它，所以第一次的视口仍是内核自己算的。
        if (scaleRef.current !== 1) mind.scale(scaleRef.current);
      } catch (thrown) {
        onErrorRef.current?.(thrown instanceof Error ? thrown : new Error(String(thrown)));
        return () => {};
      }

      /**
       * **基准必须取「内核稳定之后」的那一份，不能取刚解析出来的那一份。**
       *
       * 内核跑完第一次 `layout()` 之后会给每个顶层节点补上 `direction`
       * （0 在左 / 1 在右 —— `dist/MindElixir.js` 第 308 行，按左右计数均分）。
       * 那是一份**确定性的产物**：同一棵树永远分配出同样的结果。但它毕竟不在
       * 用户的文件里，所以如果我们拿「解析结果」当基准，第一次比对就会把它
       * 当成「用户改了东西」—— 表现是**打开文件什么都没做，`getValue()` 就交回
       * 一份多出字段的文本**，而 §7.2 第 4 条要的正是「没改过就一个字节都不动」。
       *
       * 吸收进基准之后两件事同时成立：
       * - 「打开 → 保存」逐字节一致（`getValue` 交回原文）；
       * - 用户的左右分布**能存下来**（它留在 `extra` 里，`toKernelData` 时交回内核）。
       */
      const settled = fromKernelData(document, mind.getData());
      document = settled;
      let lastText = serializeMindmap(settled, { lineEnding });
      const canonicalOriginal = lastText;

      /**
       * 把内核当前的数据同步回信封，内容真的变了才通知外壳。
       *
       * `text === lastText` 这条判据同时干掉了三类假脏：
       * 内核在 `beginEdit` 时发的 `operation`、`linkDiv` 兜底通道与主通道的重复触发、
       * 以及用户改了又改回原样。
       */
      const emit = (): void => {
        const next = fromKernelData(document, mind.getData());
        const text = serializeMindmap(next, { lineEnding });
        if (text === lastText) return;
        document = next;
        lastText = text;
        latestRef.current = text;
        onChangeRef.current(text);
      };

      // --- 主通道 --------------------------------------------------------------
      const onOperation = (info: Operation): void => {
        // `beginEdit` 只把节点切进编辑态，**一个字节都没改**。
        // 把它当变更会让「点一下节点」就把文档标脏，然后 Ctrl+S 写回一份
        // 内容相同、格式不同的文件 —— 用户会觉得「我什么都没干，它为什么要保存」。
        if (info.name === 'beginEdit') return;
        emit();
      };
      mind.bus.addListener('operation', onOperation);
      // 折叠 / 展开只发这个事件（`expanded` 不在 `operation` 里）
      mind.bus.addListener('expandNode', emit);
      // `Ctrl + 方向键` 换布局
      mind.bus.addListener('changeDirection', emit);

      // --- 兜底通道 ------------------------------------------------------------
      let verifyTimer: ReturnType<typeof setTimeout> | null = null;
      const scheduleVerify = (): void => {
        if (verifyTimer !== null) clearTimeout(verifyTimer);
        verifyTimer = setTimeout(() => {
          verifyTimer = null;
          emit();
        }, VERIFY_DELAY);
      };
      mind.bus.addListener('linkDiv', scheduleVerify);

      // --- 标记调色盘的落盘通道（ADR 0053） -------------------------------------
      // 唯一入口是内核的公开 API `reshapeNode`：它同步重渲染主题载荷、fire
      // 'operation' —— 落进上面的 emit 主通道，于是「改了标记」与「改了文字」
      // 走同一条保存链路，标脏、写回、`getValue` 一概不用单独伺候。
      markerApplyRef.current = (nodeId, glyph) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          const next = applyMarkerToggle(el.nodeObj.icons ?? [], glyph);
          // 空数组照交，不是塞 undefined：「全清过」在文件里是 `"icons": []`
          // 这一行（见 serialize / adapter 的注释），把它折成 undefined
          // 就是保存后悄悄少一句话。
          void mind.reshapeNode(el, { icons: next });
          return next;
        } catch {
          // `findEle` 对看不见的节点会 throw（ADR 0052 那条）：调色盘开着时
          // 节点被折叠/删除都够不成正常路径，抓一个「改不进去」比白屏诚实。
          return null;
        }
      };
      markerClearRef.current = (nodeId) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          if ((el.nodeObj.icons ?? []).length === 0) return [];
          void mind.reshapeNode(el, { icons: [] });
          return [];
        } catch {
          return null;
        }
      };

      // --- 图片弹层的移除通道（ADR 0054） ---------------------------------------
      // 与标记同一条路：`reshapeNode({ image: undefined })` 一步到位。
      // `undefined` 会被 `JSON.stringify` 从数据里整个丢掉 —— 文件里这个键
      // 干净消失，不留下 `"image": null` 这种半句话。
      imageRemoveRef.current = (nodeId) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return false;
          if (el.nodeObj.image === undefined) return true;
          void mind.reshapeNode(el, { image: undefined });
          return true;
        } catch {
          return false;
        }
      };

      // --- 链接弹层的落盘通道（ADR 0055） ---------------------------------------
      // 同一条 `reshapeNode` 路：fire 的 `operation` 落进 emit 主通道。
      // `url === undefined` 是「移除」：`JSON.stringify` 把 undefined 整个
      // 丢掉，文件里这个键干净消失（同 image 的移除）。
      linkApplyRef.current = (nodeId, url) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return false;
          if (url === undefined) {
            if (el.nodeObj.hyperLink === undefined) return true;
            void mind.reshapeNode(el, { hyperLink: undefined });
            return true;
          }
          void mind.reshapeNode(el, { hyperLink: url });
          return true;
        } catch {
          return false;
        }
      };

      // --- 标签弹层的落盘通道（ADR 0055） ---------------------------------------
      // 整组替换（不是逐项增删），新组**从内核现取现算**（`markerApplyRef`
      // 的先例）：弹层快照可能过期，内核的 `nodeObj` 才是真值；返回新快照
      // 供弹层回显。空数组照交 —— 「全清过」在文件里是 `"tags": []` 这
      // 一行，折成 undefined 就是保存后悄悄少一句话（同 icons 的纪律）。
      tagAddRef.current = (nodeId, text) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          const next = [...(el.nodeObj.tags ?? [])] as MindmapTag[];
          // 重复原文是 no-op：列表里看得到它，再落一次盘只会 fire 一次
          // 同值 `operation`（假脏）。返回现值让弹层保持原样。
          if (next.some((tag) => (typeof tag === 'string' ? tag === text : tag.text === text))) return next;
          next.push(text);
          void mind.reshapeNode(el, { tags: next });
          return next;
        } catch {
          return null;
        }
      };
      tagRemoveRef.current = (nodeId, index) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          const next = [...(el.nodeObj.tags ?? [])] as MindmapTag[];
          if (index < 0 || index >= next.length) return next;
          next.splice(index, 1);
          void mind.reshapeNode(el, { tags: next });
          return next;
        } catch {
          return null;
        }
      };

      // --- 备注弹层的落盘通道（ADR 0056） ---------------------------------------
      // 仍是同一条 `reshapeNode` 路。空串是「移除」：`note` 的空串在徽章上
      // 没有可指认的内容（`syncNoteBadges` 对 '' 不画徽章），写回文件却会
      // 留一行 `"note": ""` —— 保存时折成 `undefined`，`JSON.stringify`
      // 把它整个丢掉，键干净消失（同 image / hyperLink 的移除）。
      noteApplyRef.current = (nodeId, note) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return false;
          if (note === undefined || note === '') {
            if (el.nodeObj.note === undefined || el.nodeObj.note === '') return true;
            void mind.reshapeNode(el, { note: undefined });
            return true;
          }
          void mind.reshapeNode(el, { note });
          return true;
        } catch {
          return false;
        }
      };

      // --- 样式弹层的落盘通道（ADR 0059） ---------------------------------------
      /**
       * 两件事决定了这个补丁长什么样，都不是风格问题：
       *
       * 1. **`reshapeNode` 是同步的**（`Xt = function(e,t){…}`，`MindElixir.js:507-515`
       *    里一个 `await` 都没有）：它同步合并、同步 `Se` 重画、同步
       *    `bus.fire('operation')` —— 而 `operation` 监听器**同步**跑完
       *    `emit()`（`getData()` → 适配层 → 序列化 → `onChange`）。
       *    也就是说**补丁交出去的那一刻就已经落盘了**，事后再补一刀数据侧
       *    来不及 —— 补丁本身必须是干净的最终形态。
       * 2. **合并是「加 / 改」语义**：`t.style = Object.assign(o.style, t.style)`
       *    （同处）—— 把某个键从补丁里**去掉**摘不掉它，`o.style` 里那份会
       *    原样留下。摘键只有一条路：**把值置成 `undefined`**（`Object.assign`
       *    照抄 `undefined`，`getData()` 走 JSON 时值为 `undefined` 的键被整个
       *    丢掉）—— 于是文件里不会出现 `"color": ""` 这种半句话，也不必等
       *    300ms 的兜底通道来补。
       *
       * 摘掉**最后一个**键时走的是 `{ style: undefined }` 而不是逐键置
       * `undefined`：后者留下的对象 `Object.keys` 仍然数得出键，适配层的
       * 「空对象不写」挡的是 `{}`、挡不住它，文件里就会留下一句
       * `"style": {}`（规范 §7.1 说它与「没有」等价，但等价不等于该写）。
       *
       * DOM 侧这一半内核修不了：`Se` 把每个键原样塞给 CSSOM
       * （`MindElixir.js:344-347`），CSSOM 对 `undefined` 赋值按**无效值忽略**
       * —— 旧样式会赖在元素上。所以摘除后自己 `removeProperty` 收尾，
       * 与 `syncStyleFontSizes` 补 px 是同一种分工：内核做不到的那一点点，
       * 由知道真实形状的我们补上。
       */
      styleApplyRef.current = (nodeId, key, value) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          const current = el.nodeObj.style as Record<string, unknown> | undefined;
          // 再点一次 = 摘掉。判据是**内核的真值**（不是弹层快照）：
          // 快照只负责「亮哪个」，写的那一刻现问（`markerApplyRef` 的先例）。
          const remove = current !== undefined && current[key] === value;

          if (!remove) {
            void mind.reshapeNode(el, { style: { [key]: value } });
            return { ...((el.nodeObj.style as Record<string, unknown> | undefined) ?? {}) };
          }

          const rest: Record<string, unknown> = { ...(current ?? {}) };
          delete rest[key];
          const emptied = Object.keys(rest).length === 0;
          void mind.reshapeNode(el, emptied ? { style: undefined } : { style: { ...rest, [key]: undefined } });
          // 内联样式的收尾（见上第 2 段）：内核不会替我们摘。
          el.style.removeProperty(cssPropertyName(key));
          // 合并出来那份里还留着 `key: undefined`（JSON 会丢掉它、文件是干净的），
          // 但留着会让 `Object.keys` 数得出一个「其实没有」的键 —— 顺手删掉，
          // 让内存里的形状与文件一致。这一步在 `emit()` **之后**，不影响落盘。
          if (!emptied) delete (el.nodeObj.style as Record<string, unknown>)[key];
          return rest;
        } catch {
          // `findEle` 对看不见的节点会 throw（ADR 0052 那条）：弹层开着时节点
          // 被折叠 / 删除都够不成正常路径，抓一个「改不进去」比白屏诚实。
          return null;
        }
      };

      /**
       * 清除 = **整个 `style` 字段撤掉**（`{ style: undefined }`，理由见上）。
       *
       * 语义是「回到默认外观」：词表之外的键（别的工具写的 `borderColor`
       * 一类）也一并清掉 —— 这是用户**显式**要的，§5 的「不丢未知键」防的是
       * 我们悄悄动它，不是拦用户自己清。已知边界与重审条件写进 ADR 0059。
       */
      styleClearRef.current = (nodeId) => {
        try {
          const el = mind.findEle(nodeId) as KernelTopic | undefined;
          if (el == null) return null;
          const style = el.nodeObj.style as Record<string, unknown> | undefined;
          if (style === undefined) return {};
          const keys = Object.keys(style);
          if (keys.length === 0) return {};
          void mind.reshapeNode(el, { style: undefined });
          // `Se` 因为 `t.style` 是假值整段跳过 —— 旧内联样式全都赖在元素上，
          // 逐个清掉（键名取自这一刀之前的快照，那时还没被撤掉）。
          for (const key of keys) el.style.removeProperty(cssPropertyName(key));
          return {};
        } catch {
          return null;
        }
      };

      // --- 备注徽章（ADR 0056） -------------------------------------------------
      // `note` 内核零渲染：数据里有、界面上一个字都不画（Se 只画
      // text/icons/image/hyperLink/tags 五样）。徽章是它唯一看得见的形态：
      // 有备注的节点挂 📝，点击直接开弹层（快捷新入口）；**从零创建**走右键
      // 「备注…」（还没有徽章可点）。同步时机 = 内核每次重画：初始化一次 +
      // `linkDiv`（拖拽/展开/undo/redo/reshapeNode 全都会触发它，探针实测
      // undo 后徽章随之消失/重现）。
      const openNotePopover = (nodeId: string, note: string, x: number, y: number): void => {
        setNotePopover({ x, y, nodeId, note });
      };
      const syncBadges = (): void => {
        syncNoteBadges(mind.nodes, openNotePopover);
      };
      mind.bus.addListener('linkDiv', syncBadges);
      syncBadges();

      // --- 数字字号的渲染补丁（ADR 0059） ---------------------------------------
      // 与徽章同一条兜底时机：内核每次重画都把内联样式重来一遍，数字字号那一半
      // 它画不出来（CSSOM 忽略无单位长度），所以挂在 `linkDiv` 上重放。
      const syncStyles = (): void => {
        syncStyleFontSizes(mind.nodes);
      };
      mind.bus.addListener('linkDiv', syncStyles);
      syncStyles();

      // --- 链接的点击激活（ADR 0055） -------------------------------------------
      // 内核只画 🔗、不给锚点挂任何监听（grep 全库只有渲染 `:366` 与导出
      // `:2687` 两处碰 `hyper-link`）：WebView 里裸点会整页导航走，所以必须
      // 我们自己拦。捕获阶段、无条件 `preventDefault` —— 「点开一个链接」
      // 是读不是编辑，**只读也激活**（markdown 正文链接要修饰键、这里
      // 是显式 chip 普通单击即走，差异写进 ADR 0055）。
      // URL 从 `nodeObj` 读而不是 DOM 的 `href`：DOM 会说谎（内核渲染
      // 策略变了 / 属性被内联改过），数据侧才是「文件里写了什么」。
      const onHyperLinkClick = (event: MouseEvent): void => {
        const target = event.target;
        if (!(target instanceof Element)) return;
        const anchor = target.closest('a.hyper-link');
        if (anchor === null) return;
        event.preventDefault();
        event.stopPropagation();
        const topic = anchor.closest('me-tpc');
        const url = topic instanceof Element ? (topic as KernelTopic).nodeObj.hyperLink : undefined;
        if (typeof url !== 'string' || url === '') return;
        // 内核就是拿 `innerText` 画的（Se，MindElixir.js:366）；jsdom 的
        // `textContent` 因此是空的 —— 读 innerText 才两头都拿得到。
        onLinkClickRef.current?.({
          href: url,
          text: (anchor as HTMLElement).innerText || anchor.textContent || '',
        });
      };
      host.addEventListener('click', onHyperLinkClick, true);

      // --- 查找 ---------------------------------------------------------------
      const search = createTopicSearch({
        readDocument: () => document,
        patchTopic: (id, topic) => {
          const node = findNodeById(document.root, id);
          if (node === null) return;
          node.topic = topic;
        },
        patchNote: (id, note) => {
          const node = findNodeById(document.root, id);
          if (node === null) return;
          if (note === undefined) delete node.note;
          else node.note = note;
        },
        showNote: (id, note) => {
          const el = mind.findEle(id) as KernelTopic | undefined;
          if (el == null) return;
          const rect = el.getBoundingClientRect();
          openNotePopover(id, note, rect.left, rect.bottom + 8);
        },
        hideNote: () => {
          setNotePopover(null);
        },
        mind,
      });

      const handle: EditorHandle = {
        getValue: () => {
          // 现取一次而不是返回上次 `emit` 的缓存：保存路径上「少一次同步」
          // 就是「少写一段用户刚打的字」，这个风险不值得省那几毫秒。
          const next = fromKernelData(document, mind.getData());
          document = next;
          lastText = serializeMindmap(next, { lineEnding });
          // 用户没动过 → 交回**原始文本**，让「打开 → 保存」逐字节一致。
          // 判据用的是「与内核稳定之后那份文档等价」而不是「文本相同」：
          // 用户改了又改回来，同样应该交回原文（内容确实没变）。
          const result = originalText !== null && lastText === canonicalOriginal ? originalText : lastText;
          // 记的是**交出去的那个值**，不是 `lastText`。
          // 两者可能不是同一串字节：对一份非规范形式的文件，`lastText` 是规范形式。
          // 种子要是被规范形式污染了，下一次保存就会把用户的文件重排版 ——
          // 「打开 → 保存逐字节一致」当场破掉，而界面上看不出任何异常。
          latestRef.current = result;
          return result;
        },

        setValue: (text) => {
          const next = text.trim() === '' ? { document: createEmptyDocument(), lineEnding } : parseMindmap(text);
          if (next.document === null) return;
          document = next.document;
          // `refresh()` 换数据但**不换方向**（读过源码确认），所以要显式设一次
          mind.direction = directionOf(document.layout);
          mind.refresh(toKernelData(document));
          lastText = serializeMindmap(document, { lineEnding: next.lineEnding });
          latestRef.current = lastText;
          /**
           * 新数据里的图片引用（ADR 0054）：`refresh` 那一刻表里还没有它们，
           * 画出来的是占位 —— 预热完把**看得见的**带图节点用空 patch 的
           * `reshapeNode` 重渲染一遍，占位换真图。折叠着的节点不重渲染
           * 也无妨：展开那一刻内核会重新问 `imageProxy`，自然查到新表。
           */
          const fresh = collectImageUrls(document.root).filter(
            (url) => !imageCache.has(url),
          );
          if (fresh.length === 0) return;
          const imageIds = collectImageNodeIds(document.root);
          void prewarmImageCache(fresh, resolveImageUrlRef.current).then((resolved) => {
            for (const [url, dataUrl] of resolved) imageCache.set(url, dataUrl);
            for (const id of imageIds) {
              try {
                void mind.reshapeNode(mind.findEle(id), {});
              } catch {
                // 看不见的节点（折叠 / 专注模式之外）下次渲染自然就对了。
              }
            }
          });
        },

        focus: () => host.focus(),

        /**
         * 导图没有「语义 HTML」可给 —— 它的呈现是一张由绝对定位的 div 与 SVG 组成的图，
         * 而 `toHtml()` 要的是能进导出文档的**内容**片段。
         *
         * 返回 `null` 是**合法结果**（见契约）：自包含导出会退回用 `<pre>` 呈现 JSON 原文，
         * 并如实说明，而不是伪造一份「看起来像图」的 HTML。
         * 真要做富导出，正解是内核的 `exportSvg()` 那条路 —— 那个决定已经做了
         * （ADR 0050，见下面的 `exportImage`），但它是「导一张图文件」，
         * 不是「给自包含 HTML 提供内容片段」，所以这里依旧返回 `null`。
         */
        toHtml: async () => null,

        /**
         * 把导图画成一张图（ADR 0050 —— `toHtml` 注释里说的「另一个决定」就是它）。
         *
         * 画法是内核的：PNG 走 `exportPng()`、SVG 走 `exportSvg()`，这里不重画、
         * 也不重排 —— 外壳复现不了内核的图，复现只会像得走样。这里只做两件
         * 内核自己不会做的事：把底色解析成真实色值（见 `withResolvedBackground`），
         * 以及数出引用外部文件的节点图片、把「这张图不是自包含的」如实带出去
         * （见 `countExternalImages`）。
         *
         * 返回 `null` 照契约语义如实传（这一次没能画出来，比如位图编码失败）。
         *
         * 节点插图（ADR 0054）让「外部图片」从理论变成了现实：本地文件引用
         * 在 PNG 转换里读不到（转换不去碰磁盘），在 SVG 里是原样 `<image href>`
         * —— 两条后果正是 `countExternalImages` 说的那两条。占位图（图丢了时
         * 画的那张）是 `data:` 内嵌，自包含，不会误报。
         */
        exportImage: async (format) => {
          const warnings: LocalizedText[] = [];
          const external = countExternalImages(mind.nodes);
          if (external > 0) {
            warnings.push(
              msg(
                format === 'png'
                  ? 'ui.mindmap.exportPngImagesExternal'
                  : 'ui.mindmap.exportSvgImagesExternal',
                { n: external },
              ),
            );
          }
          // 换底色 → 画 → 还原。`exportPng` 内部**先同步**调 `exportSvg` 拿到 SVG 串
          // 才开始异步转换（读过内核源码），所以还原不会抢在「画」之前发生。
          const previousTheme = mind.theme;
          mind.theme = withResolvedBackground(previousTheme, mind.container);
          try {
            const blob = format === 'png' ? await mind.exportPng() : mind.exportSvg();
            return blob === null ? null : { blob, warnings };
          } finally {
            mind.theme = previousTheme;
          }
        },

        search,

        /**
         * 大纲跳转（ADR 0052）。外壳的大纲面板把「根 → 分支 → 叶」平铺成一张表，
         * 点一条要把图带到那个主题跟前。
         *
         * ## 为什么按 `document`（信封）找，而不是内核自己的数据树
         *
         * 大纲条目的 id 来自**文件层**（面板用 `parseMindmap` 算的，见
         * `core/outline/mindmap-outline.ts`），这里与 `search` 的 `readDocument`
         * 同一个来源 —— 两侧的 id 在「打开 → 编辑 → 重建」全链路上一致
         * （缺 id 的文件两边都由同一个解析器补 `auto-` 前缀）。内核自己的
         * `nodeData` 反而不可靠：专注模式（右键菜单的「聚焦」）会把它换成子树，
         * 结构不是全的。
         *
         * ## 展开折叠的祖先是文档变更，标脏是诚实的
         *
         * `collapsed` 是文件字段（§7.1），展开它就是改文档 —— `expandNode` 发的
         * 事件走 `emit` 主通道，保存会把展开状态写进文件。这不是副作用，
         * 是这件事本来的样子。
         *
         * ## 逐段 try/catch，抓的是内核的 `throw`
         *
         * `findEle` 对看不见的节点（折叠中 / 专注模式的子树之外）会**抛错**而不是
         * 返回空（`MindElixir.js:341`）。抓住、交 `false` —— 外壳会如实说没跳过去，
         * 而不是白屏。
         *
         * 只读时折叠的祖先**不展开**（那是编辑），够不着的条目交 `false`；
         * 看得见的照跳 —— 选中不是编辑，不弄脏文件。
         */
        goToOutlineItem: ({ level, text, id }) => {
          const path = findTopicPath(document.root, { id, level, text });
          if (path === null) return false;
          const target = path[path.length - 1];
          if (target === undefined) return false;

          /**
           * 展开路上折叠着的祖先（自根向下）。两个如实的例外：
           *
           * - **根不展开**：内核的 `expandNode` 只认带「+/-」气泡的分支节点，
           *   对根调用会抛错（根没有那个气泡）；而内核对根的 `collapsed`
           *   本来就不做视觉隐藏 —— 子分支永远渲染，跳过去照样看得见。
           *   所以跳转照常成功，文件里的 `collapsed` 维持原样，不假装展开过。
           * - **只读不展开**：展开是编辑，够不着的条目交 `false`。
           */
          const ancestors = path.slice(0, -1);
          for (let index = 0; index < ancestors.length; index += 1) {
            const ancestor = ancestors[index];
            if (ancestor === undefined || ancestor.collapsed !== true) continue;
            if (index === 0) continue;
            if (readOnlyRef.current) return false;
            try {
              mind.expandNode(mind.findEle(ancestor.id), true);
            } catch {
              return false;
            }
          }
          try {
            const el = mind.findEle(target.id);
            mind.selectNode(el);
            mind.scrollIntoView(el);
          } catch {
            return false;
          }
          host.focus();
          return true;
        },
      };

      // 外壳从此拿到这条通道（契约方法都挂在 handle 上）。
      onReadyRef.current(handle);

      return () => {
        // 兜底通道还挂着一个待办的比对（用户拖完 300ms 内就切了语言）：
        // 那份改动还没进 `latestRef`。重建前先把它结掉，否则「拖一下、切语言」
        // 会丢掉那一下 —— 界面还画着动过的图，一保存就弹回去。
        // 用 `emit()` 而不是 `getValue()`：前者是每次用户操作都在跑的同一段纯数据代码，
        // 后者要在 cleanup 里包 try/catch 兜异常，那等于开一个静默吞错的洞（硬约束 2）。
        if (verifyTimer !== null) {
          clearTimeout(verifyTimer);
          verifyTimer = null;
          emit();
        }
        // 缩放留给下一个内核（见 `scaleRef`）。
        scaleRef.current = mind.scaleVal;
        // 标记 / 图片 / 链接 / 标签 / 备注五条内核自有 UI 的落盘通道跟着内核一起
        // 下岗（新 effect 会装新的）；弹层若还开着就关掉 —— 它闭住的那个
        // 内核已经销毁了。
        markerApplyRef.current = null;
        markerClearRef.current = null;
        imageRemoveRef.current = null;
        linkApplyRef.current = null;
        tagAddRef.current = null;
        tagRemoveRef.current = null;
        noteApplyRef.current = null;
        styleApplyRef.current = null;
        styleClearRef.current = null;
        setMarkerPalette(null);
        setImagePopover(null);
        setLinkPopover(null);
        setTagPopover(null);
        setNotePopover(null);
        setStylePopover(null);
        host.removeEventListener('click', onHyperLinkClick, true);
        mind.bus.removeListener('operation', onOperation);
        mind.bus.removeListener('expandNode', emit);
        mind.bus.removeListener('changeDirection', emit);
        mind.bus.removeListener('linkDiv', scheduleVerify);
        mind.bus.removeListener('linkDiv', syncBadges);
        mind.bus.removeListener('linkDiv', syncStyles);
        mind.destroy();
      };
    };

    // 先预热、后装内核 —— `boot` 头部的注释说了为什么这个顺序不能反。
    const imageUrls = collectImageUrls(first.root);
    const start = (): void => {
      if (cancelled) return;
      teardown = boot();
    };
    if (imageUrls.length === 0) {
      start();
    } else {
      void prewarmImageCache(imageUrls, resolveImageUrlRef.current).then((resolved) => {
        for (const [url, dataUrl] of resolved) imageCache.set(url, dataUrl);
        start();
      });
    }

    return () => {
      cancelled = true;
      teardown?.();
    };
    // 依赖 `locale`：语言变了要重建（内核在安装那一刻就把菜单文案烤死了，见文件头）。
    // 除此之外内核只活一次 —— 外部内容变更一律走 handle.setValue，不该在这里重建。
  }, [locale]);

  /**
   * 调色盘的三个出口。`useCallback` 只为引用稳定：`MarkerPalette` 里挂的
   * document 级监听以 `onClose` 为依赖，不稳定的话每点一次字形都要
   * 重挂一遍监听 —— 能用，但没必要。
   */
  const closeMarkerPalette = useCallback((): void => {
    setMarkerPalette(null);
  }, []);

  const toggleMarker = useCallback((nodeId: string, glyph: string): void => {
    // `null` = 这一下没落进去（节点已经够不着了）：快照不动，界面如实停在那里。
    const next = markerApplyRef.current?.(nodeId, glyph) ?? null;
    if (next === null) return;
    setMarkerPalette((state) => (state !== null && state.nodeId === nodeId ? { ...state, icons: [...next] } : state));
  }, []);

  const clearMarkers = useCallback((nodeId: string): void => {
    const next = markerClearRef.current?.(nodeId) ?? null;
    if (next === null) return;
    setMarkerPalette((state) => (state !== null && state.nodeId === nodeId ? { ...state, icons: [...next] } : state));
  }, []);

  /** 图片弹层的两个出口，引用稳定的理由同上。 */
  const closeImagePopover = useCallback((): void => {
    setImagePopover(null);
  }, []);

  const removeImage = useCallback((nodeId: string): void => {
    // `false` = 没移成（节点已经够不着了）：弹层如实留在原地，不假装成功。
    if (imageRemoveRef.current?.(nodeId) !== true) return;
    setImagePopover(null);
  }, []);

  /** 链接 / 标签弹层的出口，引用稳定的理由同上。 */
  const closeLinkPopover = useCallback((): void => {
    setLinkPopover(null);
  }, []);

  const saveLink = useCallback((nodeId: string, url: string): void => {
    if (linkApplyRef.current?.(nodeId, url) !== true) return;
    setLinkPopover(null);
  }, []);

  const removeLink = useCallback((nodeId: string): void => {
    // `false` = 没移成（节点已经够不着了）：弹层如实留在原地。
    if (linkApplyRef.current?.(nodeId, undefined) !== true) return;
    setLinkPopover(null);
  }, []);

  const closeTagPopover = useCallback((): void => {
    setTagPopover(null);
  }, []);

  const addTag = useCallback((nodeId: string, text: string): void => {
    // 空白不算标签（占位文案就写着「回车挂上」，空回车应该是无事发生）。
    const trimmed = text.trim();
    if (trimmed === '') return;
    // `null` = 没加成（节点已经够不着了）：快照不动，弹层如实停在原地。
    const next = tagAddRef.current?.(nodeId, trimmed) ?? null;
    if (next === null) return;
    setTagPopover((state) => (state !== null && state.nodeId === nodeId ? { ...state, tags: [...next] } : state));
  }, []);

  const removeTag = useCallback((nodeId: string, index: number): void => {
    const next = tagRemoveRef.current?.(nodeId, index) ?? null;
    if (next === null) return;
    setTagPopover((state) => (state !== null && state.nodeId === nodeId ? { ...state, tags: [...next] } : state));
  }, []);

  /** 备注弹层的出口，引用稳定的理由同上。 */
  const closeNotePopover = useCallback((): void => {
    setNotePopover(null);
  }, []);

  const saveNote = useCallback((nodeId: string, text: string): void => {
    // 落盘通道里空串已折成移除（见 `noteApplyRef` 的注释）。
    if (noteApplyRef.current?.(nodeId, text) !== true) return;
    setNotePopover(null);
  }, []);

  const removeNote = useCallback((nodeId: string): void => {
    // `false` = 没移成（节点已经够不着了）：弹层如实留在原地。
    if (noteApplyRef.current?.(nodeId, undefined) !== true) return;
    setNotePopover(null);
  }, []);

  /** 样式弹层的三个出口，引用稳定的理由同上面几族。 */
  const closeStylePopover = useCallback((): void => {
    setStylePopover(null);
  }, []);

  const applyStyle = useCallback((nodeId: string, key: StyleKey, value: string | number): void => {
    // `null` = 这一下没落进去（节点已经够不着了）：快照不动，界面如实停在那里。
    const next = styleApplyRef.current?.(nodeId, key, value) ?? null;
    if (next === null) return;
    setStylePopover((state) => (state !== null && state.nodeId === nodeId ? { ...state, style: next } : state));
  }, []);

  const clearStyle = useCallback((nodeId: string): void => {
    // `null` = 没清成（节点已经够不着了）：快照不动，界面如实停在那里。
    const next = styleClearRef.current?.(nodeId) ?? null;
    if (next === null) return;
    setStylePopover((state) => (state !== null && state.nodeId === nodeId ? { ...state, style: next } : state));
  }, []);

  return (
    <>
      <div ref={hostRef} className="qm-mindmap" tabIndex={-1} />
      {markerPalette !== null && (
        <MarkerPalette state={markerPalette} onToggle={toggleMarker} onClear={clearMarkers} onClose={closeMarkerPalette} />
      )}
      {imagePopover !== null && (
        <ImagePopover state={imagePopover} onRemove={removeImage} onClose={closeImagePopover} />
      )}
      {linkPopover !== null && (
        <LinkPopover state={linkPopover} onSave={saveLink} onRemove={removeLink} onClose={closeLinkPopover} />
      )}
      {notePopover !== null && (
        // key 挂节点 id：弹层草稿只在挂载时播种（useState 初值），
        // 键盘流 reveal 换节点时没有 mousedown 外点那一下自愈 ——
        // 不换挂载点就会把上一个节点的陈旧草稿端出来（ADR 0058）。
        <NotePopover key={notePopover.nodeId} state={notePopover} readOnly={readOnly === true} onSave={saveNote} onRemove={removeNote} onClose={closeNotePopover} />
      )}
      {tagPopover !== null && (
        <TagPopover state={tagPopover} onAdd={addTag} onRemove={removeTag} onClose={closeTagPopover} />
      )}
      {stylePopover !== null && (
        <StylePopover state={stylePopover} onApply={applyStyle} onClear={clearStyle} onClose={closeStylePopover} />
      )}
    </>
  );
}

/** 调色盘要显示的东西：位置（屏幕坐标）、目标节点、该节点当前的标记快照。 */
interface MarkerPaletteState {
  readonly x: number;
  readonly y: number;
  readonly nodeId: string;
  readonly icons: readonly string[];
}

interface MarkerPaletteProps {
  readonly state: MarkerPaletteState;
  readonly onToggle: (nodeId: string, glyph: string) => void;
  readonly onClear: (nodeId: string) => void;
  readonly onClose: () => void;
}

/** 估尺寸，只用来把调色盘夹在视口里（jsdom 没有布局，不能问元素自己）。 */
const PALETTE_EST_WIDTH = 232;
const PALETTE_EST_HEIGHT = 300;

/**
 * 标记调色盘（ADR 0053）。
 *
 * 画在**内核容器之外**（`MindmapEditor` 返回的兄弟节点）：内核在它自己的
 * 容器上监听点击/拖拽，够不着这里；它也不碰内核的 DOM —— 两边只通过
 * `reshapeNode` 这一个公开 API 说话（见 `markerApplyRef` 的注释）。
 *
 * 关闭 = 点外面 / Escape。**点字形不关**：一个主题常常一次标优先级、一次
 * 标色块，关了再开是折磨；摘掉也走同一下（再点一次 = 取消，XMind 同款）。
 */
function MarkerPalette({ state, onToggle, onClear, onClose }: MarkerPaletteProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && !rootRef.current.contains(event.target)) onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  // 夹进视口：`Math.max` 保住 8px 下限（窗口比调色盘还窄时也不贴边出界）。
  const left = Math.min(state.x, Math.max(8, window.innerWidth - PALETTE_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - PALETTE_EST_HEIGHT - 8));

  return (
    <div ref={rootRef} className="mm-marker-palette" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.markerMenu')}>
      {MINDMAP_MARKER_GROUPS.map((group) => (
        <div key={group.id} className="mm-marker-group">
          <div className="mm-marker-group-label">{t(group.labelKey)}</div>
          <div className="mm-marker-choices">
            {MINDMAP_MARKERS.filter((marker) => marker.group === group.id).map((marker) => (
              <button
                key={marker.glyph}
                type="button"
                className={state.icons.includes(marker.glyph) ? 'mm-marker active' : 'mm-marker'}
                title={t(marker.labelKey, marker.labelParams)}
                onClick={() => onToggle(state.nodeId, marker.glyph)}
              >
                {marker.glyph}
              </button>
            ))}
          </div>
        </div>
      ))}
      <button type="button" className="mm-marker-clear" onClick={() => onClear(state.nodeId)}>
        {t('ui.mindmap.markerClear')}
      </button>
    </div>
  );
}

/**
 * 图片弹层的两种状态（ADR 0054），判别字段 `kind`：
 *
 * - `'hint'`：主题上没图，弹层显示粘贴说明 —— 插图的入口只有粘贴这一条，
 *   弹层就是这个事实的操作说明；
 * - `'attached'`：有图，`preview` 是**预览图**（占位在 `openImagePopover` 里
 *   就烘焙好了 —— 弹层是纯展示，不碰预热表、不碰文件系统），
 *   「移除」是图片唯一的删除出口。`fit` 刻意不进来：它决定的是图怎么
 *   嵌进节点（缩放语义），弹层里原样显示预览就够了。
 */
type ImagePopoverState =
  | { readonly kind: 'hint'; readonly x: number; readonly y: number; readonly nodeId: string }
  | {
      readonly kind: 'attached';
      readonly x: number;
      readonly y: number;
      readonly nodeId: string;
      readonly image: MindmapImage;
      readonly preview: string;
    };

interface ImagePopoverProps {
  readonly state: ImagePopoverState;
  readonly onRemove: (nodeId: string) => void;
  readonly onClose: () => void;
}

/** 估尺寸的理由同 `PALETTE_EST_*`：夹视口用，jsdom 没布局可问。 */
const IMAGE_POPOVER_EST_WIDTH = 280;
const IMAGE_POPOVER_EST_HEIGHT = 320;

/**
 * 图片弹层（ADR 0054），形状照抄标记调色盘。
 *
 * 两种状态都有事可做，不静默：
 * - **没图**：告诉用户怎么粘（选中主题 + Ctrl+V）—— 插图的入口只有粘贴
 *   这一条，弹层就是这个事实的操作说明；
 * - **有图**：显示引用与预览，给「移除」—— 唯一的删除出口也在这里。
 */
function ImagePopover({ state, onRemove, onClose }: ImagePopoverProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && !rootRef.current.contains(event.target)) onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  const left = Math.min(state.x, Math.max(8, window.innerWidth - IMAGE_POPOVER_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - IMAGE_POPOVER_EST_HEIGHT - 8));

  return (
    <div ref={rootRef} className="mm-image-popover" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.imageMenu')}>
      {state.kind === 'hint' ? (
        <div className="mm-image-hint">{t('ui.mindmap.imagePasteHint')}</div>
      ) : (
        <>
          <img className="mm-image-preview" src={state.preview} alt={t('ui.mindmap.imageMenu')} />
          <div className="mm-image-meta">
            {t('ui.mindmap.imageInfo', { path: state.image.url, w: String(state.image.width), h: String(state.image.height) })}
          </div>
          <button type="button" className="mm-image-remove" onClick={() => onRemove(state.nodeId)}>
            {t('ui.mindmap.imageRemove')}
          </button>
        </>
      )}
    </div>
  );
}

/** 链接弹层要显示的东西：位置、目标节点、此刻的链接原文（空串 = 还没有）。 */
interface LinkPopoverState {
  readonly x: number;
  readonly y: number;
  readonly nodeId: string;
  readonly url: string;
}

interface LinkPopoverProps {
  readonly state: LinkPopoverState;
  readonly onSave: (nodeId: string, url: string) => void;
  readonly onRemove: (nodeId: string) => void;
  readonly onClose: () => void;
}

const LINK_POPOVER_EST_WIDTH = 300;
const LINK_POPOVER_EST_HEIGHT = 190;

/**
 * 链接弹层（ADR 0055）。关闭 / 夹视口 / 监听模式照 `MarkerPalette`；
 * 多出来的东西是输入框 —— 所以打开时把焦点直接放进输入框（另外两个
 * 弹层没有输入框，也就没有这一步）。
 *
 * `state.url` 决定按钮的套数：还没有链接就只有「保存」，有链接才补
 * 「移除」。保存的判据与按钮的禁用是同一条（trim 后非空）—— Enter
 * 那条路复用同一个 `save()`，不另写一份判断。
 */
function LinkPopover({ state, onSave, onRemove, onClose }: LinkPopoverProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [draft, setDraft] = useState(state.url);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && rootRef.current.contains(event.target)) return;
      onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  // 打开就是为了改：焦点落在输入框并全选现值 —— 覆写一条链接是常态。
  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const left = Math.min(state.x, Math.max(8, window.innerWidth - LINK_POPOVER_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - LINK_POPOVER_EST_HEIGHT - 8));

  const save = (): void => {
    const trimmed = draft.trim();
    if (trimmed === '') return;
    onSave(state.nodeId, trimmed);
  };

  return (
    <div ref={rootRef} className="mm-link-popover" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.linkMenu')}>
      <div className="mm-link-hint">{t('ui.mindmap.linkHint')}</div>
      <input
        ref={inputRef}
        className="mm-link-input"
        type="text"
        value={draft}
        placeholder={t('ui.mindmap.linkPlaceholder')}
        aria-label={t('ui.mindmap.linkMenu')}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') save();
        }}
      />
      <div className="mm-link-actions">
        <button type="button" className="mm-link-save" disabled={draft.trim() === ''} onClick={save}>
          {t('ui.mindmap.linkSave')}
        </button>
        {state.url !== '' && (
          <button type="button" className="mm-link-remove" onClick={() => onRemove(state.nodeId)}>
            {t('ui.mindmap.linkRemove')}
          </button>
        )}
      </div>
    </div>
  );
}

/** 标签弹层要显示的东西：位置、目标节点、打开那一刻的标签快照。 */
interface TagPopoverState {
  readonly x: number;
  readonly y: number;
  readonly nodeId: string;
  readonly tags: readonly MindmapTag[];
}

interface TagPopoverProps {
  readonly state: TagPopoverState;
  readonly onAdd: (nodeId: string, text: string) => void;
  readonly onRemove: (nodeId: string, index: number) => void;
  readonly onClose: () => void;
}

const TAG_POPOVER_EST_WIDTH = 280;
const TAG_POPOVER_EST_HEIGHT = 300;

/**
 * 标签弹层（ADR 0055）。**加一个不关弹层**：连着挂几个标签是常态，
 * 关了再开是折磨（「点字形不关」同一条理由）。每个 chip 带 ✕ 按索引
 * 移除 —— 快照与内核的一致性由 ref 通道保证（见 `tagRemoveRef`），
 * 这里只管把「点的是第几个」报上去。
 */
function TagPopover({ state, onAdd, onRemove, onClose }: TagPopoverProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [draft, setDraft] = useState('');

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && rootRef.current.contains(event.target)) return;
      onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const left = Math.min(state.x, Math.max(8, window.innerWidth - TAG_POPOVER_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - TAG_POPOVER_EST_HEIGHT - 8));

  const add = (): void => {
    const trimmed = draft.trim();
    if (trimmed === '') return;
    onAdd(state.nodeId, trimmed);
    setDraft('');
  };

  return (
    <div ref={rootRef} className="mm-tag-popover" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.tagMenu')}>
      <div className="mm-tag-list">
        {state.tags.map((tag, index) => (
          <span key={`${index}`} className="mm-tag-chip">
            {/* 文字单独一层：chip 的 textContent 就不是「文字+✕」混在一起 */}
            <span className="mm-tag-text">{typeof tag === 'string' ? tag : tag.text}</span>
            <button
              type="button"
              className="mm-tag-remove"
              aria-label={t('ui.mindmap.tagRemoveOne')}
              onClick={() => onRemove(state.nodeId, index)}
            >
              ✕
            </button>
          </span>
        ))}
        {state.tags.length === 0 && <div className="mm-tag-empty">{t('ui.mindmap.tagPlaceholder')}</div>}
      </div>
      <div className="mm-tag-entry">
        <input
          ref={inputRef}
          className="mm-tag-input"
          type="text"
          value={draft}
          placeholder={t('ui.mindmap.tagPlaceholder')}
          aria-label={t('ui.mindmap.tagMenu')}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') add();
          }}
        />
        <button type="button" className="mm-tag-add" disabled={draft.trim() === ''} onClick={add}>
          {t('ui.mindmap.tagAdd')}
        </button>
      </div>
    </div>
  );
}

/** 备注弹层要显示的东西：位置、目标节点、打开那一刻的备注原文（空串 = 还没有）。 */
interface NotePopoverState {
  readonly x: number;
  readonly y: number;
  readonly nodeId: string;
  readonly note: string;
}

interface NotePopoverProps {
  readonly state: NotePopoverState;
  readonly readOnly: boolean;
  readonly onSave: (nodeId: string, text: string) => void;
  readonly onRemove: (nodeId: string) => void;
  readonly onClose: () => void;
}

const NOTE_POPOVER_EST_WIDTH = 300;
const NOTE_POPOVER_EST_HEIGHT = 240;

/**
 * 备注弹层（ADR 0056）。关闭 / 夹视口照前三个弹层；正文是 textarea
 * （备注天然多行）。**Ctrl+Enter 保存**：备注里的普通 Enter 是换行，
 * 不能像链接那样把 Enter 拿去当保存 —— 按键写进了占位文案。空文本
 * 不能「保存」（要清空用「移除」），与链接弹层同一条判据。
 * 只读时整套按钮不渲染、textarea 只读 —— 读备注是读，不是编辑。
 */
function NotePopover({ state, readOnly, onSave, onRemove, onClose }: NotePopoverProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const [draft, setDraft] = useState(state.note);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && rootRef.current.contains(event.target)) return;
      onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const left = Math.min(state.x, Math.max(8, window.innerWidth - NOTE_POPOVER_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - NOTE_POPOVER_EST_HEIGHT - 8));

  const save = (): void => {
    const trimmed = draft.trim();
    if (trimmed === '') return;
    onSave(state.nodeId, trimmed);
  };

  return (
    <div ref={rootRef} className="mm-note-popover" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.noteBadgeTitle')}>
      <textarea
        ref={inputRef}
        className="mm-note-input"
        value={draft}
        placeholder={t('ui.mindmap.notePlaceholder')}
        aria-label={t('ui.mindmap.noteBadgeTitle')}
        readOnly={readOnly}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) save();
        }}
      />
      {!readOnly && (
        <div className="mm-note-actions">
          <button type="button" className="mm-note-save" disabled={draft.trim() === ''} onClick={save}>
            {t('ui.mindmap.noteSave')}
          </button>
          {state.note !== '' && (
            <button type="button" className="mm-note-remove" onClick={() => onRemove(state.nodeId)}>
              {t('ui.mindmap.noteRemove')}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/** 样式弹层要显示的东西：位置、目标节点、打开那一刻的样式快照。 */
interface StylePopoverState {
  readonly x: number;
  readonly y: number;
  readonly nodeId: string;
  /**
   * 打开那一刻的样式快照，之后由每次落盘的**返回值**刷新 —— 一个主题常常
   * 连配几样（颜色 + 字号 + 加粗），亮着的色块必须跟着走。它只负责
   * 「亮哪个」：真正「再点一次是不是摘掉」在落盘通道里从内核现问
   * （`markerApplyRef` 的先例），快照过期也写不错。
   */
  readonly style: StyleValues;
}

interface StylePopoverProps {
  readonly state: StylePopoverState;
  readonly onApply: (nodeId: string, key: StyleKey, value: string | number) => void;
  readonly onClear: (nodeId: string) => void;
  readonly onClose: () => void;
}

/** 估尺寸，只用来把弹层夹在视口里（jsdom 没有布局，不能问元素自己）。 */
const STYLE_EST_WIDTH = 244;
const STYLE_EST_HEIGHT = 292;

/**
 * 主题样式弹层（ADR 0059 —— 对标 XMind 的样式面板第一屏）。
 *
 * 与标记调色盘（0053）同一条纪律：画在**内核容器之外**（React 画的兄弟节点），
 * 不碰内核的 DOM，两边只通过 `reshapeNode` 这一个公开 API 说话。
 *
 * 关闭 = 点外面 / Escape。**点色块不关**：一个主题常常一次配颜色、一次配字号、
 * 一次加粗，关了再开是折磨；摘掉也走同一下（再点一次 = 取消，XMind 同款）。
 *
 * 亮不亮一律由 `node-style.ts` 的纯函数判（词表外的值不亮也不假装摘得掉）。
 */
function StylePopover({ state, onApply, onClear, onClose }: StylePopoverProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent): void => {
      if (rootRef.current === null) return;
      if (event.target instanceof Node && rootRef.current.contains(event.target)) return;
      onClose();
    };
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  const left = Math.min(state.x, Math.max(8, window.innerWidth - STYLE_EST_WIDTH - 8));
  const top = Math.min(state.y, Math.max(8, window.innerHeight - STYLE_EST_HEIGHT - 8));

  const textIndex = activeSwatchIndex(STYLE_TEXT_COLORS, state.style, 'color');
  const fillIndex = activeSwatchIndex(STYLE_FILL_COLORS, state.style, 'background');
  const size = activeFontSize(state.style);
  const bold = isBold(state.style);

  return (
    <div ref={rootRef} className="mm-style-popover" style={{ left, top }} role="dialog" aria-label={t('ui.mindmap.styleTitle')}>
      <div className="mm-style-row">
        <div className="mm-style-label">{t('ui.mindmap.styleText')}</div>
        <div className="mm-style-choices">
          {STYLE_TEXT_COLORS.map((hex, index) => (
            <button
              key={hex}
              type="button"
              className={index === textIndex ? 'mm-style-swatch mm-style-swatch-text active' : 'mm-style-swatch mm-style-swatch-text'}
              // 文字色用「那个颜色的 A」示意，不刷底色 —— 刷了就成了「背景」那一行的同款，
              // 两行看起来一模一样，用户只能靠读标签分辨。
              style={{ color: hex }}
              title={hex}
              aria-label={`${t('ui.mindmap.styleText')} ${hex}`}
              aria-pressed={index === textIndex}
              onClick={() => onApply(state.nodeId, 'color', hex)}
            >
              A
            </button>
          ))}
        </div>
      </div>
      <div className="mm-style-row">
        <div className="mm-style-label">{t('ui.mindmap.styleFill')}</div>
        <div className="mm-style-choices">
          {STYLE_FILL_COLORS.map((hex, index) => (
            <button
              key={hex}
              type="button"
              className={index === fillIndex ? 'mm-style-swatch mm-style-swatch-fill active' : 'mm-style-swatch mm-style-swatch-fill'}
              style={{ background: hex }}
              title={hex}
              aria-label={`${t('ui.mindmap.styleFill')} ${hex}`}
              aria-pressed={index === fillIndex}
              onClick={() => onApply(state.nodeId, 'background', hex)}
            />
          ))}
        </div>
      </div>
      <div className="mm-style-row">
        <div className="mm-style-label">{t('ui.mindmap.styleSize')}</div>
        <div className="mm-style-choices">
          {STYLE_FONT_SIZES.map((value) => (
            <button
              key={value}
              type="button"
              className={value === size ? 'mm-style-chip active' : 'mm-style-chip'}
              aria-pressed={value === size}
              onClick={() => onApply(state.nodeId, 'fontSize', value)}
            >
              {value}
            </button>
          ))}
        </div>
      </div>
      <div className="mm-style-row">
        <div className="mm-style-label">{t('ui.mindmap.styleBold')}</div>
        <div className="mm-style-choices">
          <button
            type="button"
            className={bold ? 'mm-style-chip mm-style-bold active' : 'mm-style-chip mm-style-bold'}
            title={t('ui.mindmap.styleBold')}
            aria-label={t('ui.mindmap.styleBold')}
            aria-pressed={bold}
            onClick={() => onApply(state.nodeId, 'fontWeight', 'bold')}
          >
            B
          </button>
        </div>
      </div>
      <button type="button" className="mm-style-clear" onClick={() => onClear(state.nodeId)}>
        {t('ui.mindmap.styleClear')}
      </button>
    </div>
  );
}

interface TopicSearchDeps {
  readDocument: () => MindmapDocument;
  patchTopic: (id: string, topic: string) => void;
  /** 信封侧同步写回 note（`undefined` = 清空即移除，0056 纪律）—— 同 `patchTopic` 的理由。 */
  patchNote: (id: string, note: string | undefined) => void;
  /** 把某节点的备注弹层亮出来 —— `reveal` 走到 note 命中时用（ADR 0058）。 */
  showNote: (id: string, note: string) => void;
  /** 关掉备注弹层 —— 任何写回都不该让陈旧弹层赖在屏上（ADR 0058）。 */
  hideNote: () => void;
  mind: MindElixirInstance;
}

/**
 * 导图里的查找替换。
 *
 * 与代码内核那份的差别只有「匹配在哪里找」：代码内核在**一篇文本**里找，
 * 这里在**每个节点的 topic 和 note** 里各找一遍（理由见 `collectHits`）。
 * 导航顺序同样是文档顺序（先序遍历），因为契约要求「顺序即导航顺序」。
 */
function createTopicSearch({ readDocument, patchTopic, patchNote, showNote, hideNote, mind }: TopicSearchDeps): EditorSearch {
  let hits: TopicHit[] = [];
  let options: FindOptions | null = null;
  let pattern: RegExp | null = null;

  const refind = (): number => {
    if (options === null) return 0;
    hits = collectHits(readDocument().root, options);
    pattern = createFindPattern(options);
    return hits.length;
  };

  const focusHit = (hit: TopicHit): KernelTopic | null => {
    const el = mind.findEle(hit.id) as KernelTopic | undefined;
    if (!el) return null;
    mind.selectNode(el);
    mind.scrollIntoView(el);
    return el;
  };

  return {
    find: (next) => {
      options = next;
      return refind();
    },

    reveal: (index) => {
      const hit = hits[index];
      if (!hit) return;
      focusHit(hit);
      // note 命中要把弹层亮出来 —— 命中落在弹层里，不亮就等于没定位到；
      // topic 命中把上一个亮着的弹层收掉，别让旧内容赖在屏上（ADR 0058）。
      if (hit.field === 'note') {
        const note = findNodeById(readDocument().root, hit.id)?.note ?? '';
        showNote(hit.id, note);
      } else {
        hideNote();
      }
    },

    replace: (index, replacement) => {
      const hit = hits[index];
      if (!hit || options === null) return hits.length;
      const node = findNodeById(readDocument().root, hit.id);
      const el = focusHit(hit);
      if (node === null || el === null) return hits.length;

      if (hit.field === 'note') {
        // note 路径：偏移是 note 段内的，写回走 reshapeNode —— 与备注弹层
        // 同一条路（ADR 0056）。替换到空串 = 清空即移除，空串不留在文件里。
        // 弹层必须关：它的草稿只在挂载时播种，note 被改后留着就是陈旧输入框。
        const current = node.note ?? '';
        const matched = current.slice(hit.from, hit.to);
        const insert = options.regex && pattern !== null ? matched.replace(new RegExp(pattern.source, pattern.flags.replace('g', '')), replacement) : replacement;
        const updated = current.slice(0, hit.from) + insert + current.slice(hit.to);
        const note = updated === '' ? undefined : updated;
        patchNote(hit.id, note);
        void mind.reshapeNode(el, { note });
        hideNote();
        return refind();
      }

      const current = node.topic;
      const matched = current.slice(hit.from, hit.to);
      const insert = options.regex && pattern !== null ? matched.replace(new RegExp(pattern.source, pattern.flags.replace('g', '')), replacement) : replacement;
      const updated = current.slice(0, hit.from) + insert + current.slice(hit.to);

      // 两边一起改：内核那边是用户看到的东西，信封那边是下一次 `refind` 的依据。
      // 内核的 `setNodeTopic` 是异步的，只改内核会让紧接着的 `refind` 读到旧文本，
      // 于是「替换全部」会漏掉后面的匹配 —— 而信封这一侧是同步的，正好补上这个空档。
      // 不等它是**有意**的：外壳的替换动作是同步接口，等下去就得把整个 `EditorSearch`
      // 改成异步的（那是契约级改动），而这里要的只是「界面立刻变」——
      // `setNodeTopic` 内部的 DOM 更新本来就是同步的，异步的只是它返回的 Promise。
      void mind.setNodeTopic(el, updated);
      patchTopic(hit.id, updated);
      return refind();
    },

    replaceAll: (replacement) => {
      if (options === null || hits.length === 0) return 0;

      if (options.regex) {
        // 正则模式下 `$1` 要展开，而捕获组信息在逐条替换时拿不到 ——
        // 对每个节点**每段**（topic / note）整段替换一次，语义最准
        // （与代码内核那边同一条理由）。topic 与 note 是两段独立文本，
        // 必须分开键控：同一节点的两段各有各的命中，混作一段会互相踩。
        const count = hits.length;
        const targets = new Map<string, { field: TopicHit['field']; id: string; text: string }>();
        for (const hit of hits) {
          const node = findNodeById(readDocument().root, hit.id);
          if (node === null) continue;
          const key = `${hit.field}:${hit.id}`;
          const source = targets.get(key)?.text ?? (hit.field === 'topic' ? node.topic : node.note ?? '');
          targets.set(key, { field: hit.field, id: hit.id, text: replaceInText(source, options, replacement).text });
        }
        for (const { field, id, text } of targets.values()) {
          const el = mind.findEle(id) as KernelTopic | undefined;
          if (field === 'topic') {
            patchTopic(id, text);
            if (el) void mind.setNodeTopic(el, text);
          } else {
            const note = text === '' ? undefined : text;
            patchNote(id, note);
            if (el) void mind.reshapeNode(el, { note });
          }
        }
        hideNote();
        hits = [];
        // 返回值是「替换掉的个数」—— 按命中数报：一条 topic 和一条 note
        // 是两处替换。旧版只数节点，topic/note 分家后这个语义才说得通。
        return count;
      }

      // 字面替换：按**节点 + 字段**分组，组内**倒序**处理。
      // 倒序是必须的 —— 同一段文字里有两处命中时，先替换前面那处会让
      // 后面那处的偏移失效（替换文本长度一变，后面的下标就全错了）。
      // topic 与 note 的偏移各算各的，分开分组才不会串线。
      const bySegment = new Map<string, { field: TopicHit['field']; id: string; list: TopicHit[] }>();
      for (const hit of hits) {
        const key = `${hit.field}:${hit.id}`;
        const group = bySegment.get(key);
        if (group === undefined) bySegment.set(key, { field: hit.field, id: hit.id, list: [hit] });
        else group.list.push(hit);
      }

      const count = hits.length;
      for (const { field, id, list } of bySegment.values()) {
        const node = findNodeById(readDocument().root, id);
        if (node === null) continue;
        let text = field === 'topic' ? node.topic : node.note ?? '';
        for (const hit of [...list].sort((a, b) => b.from - a.from)) {
          text = text.slice(0, hit.from) + replacement + text.slice(hit.to);
        }
        const el = mind.findEle(id) as KernelTopic | undefined;
        if (field === 'topic') {
          patchTopic(id, text);
          if (el) void mind.setNodeTopic(el, text);
        } else {
          const note = text === '' ? undefined : text;
          patchNote(id, note);
          if (el) void mind.reshapeNode(el, { note });
        }
      }
      hideNote();
      hits = [];
      return count;
    },

    clear: () => {
      hits = [];
      options = null;
      pattern = null;
    },
  };
}

export default MindmapEditor;
