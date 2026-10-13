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
import { type LineEnding, type MindmapDocument, type MindmapImage, type MindmapNode } from '../../core/mindmap/types.js';
import { createFindPattern, findInText, replaceInText, type FindOptions } from '../../core/search/find.js';
import { directionOf, fromKernelData, toKernelData } from './adapter.js';
import { mindmapLangPack } from './lang-pack.js';
import { applyMarkerToggle, MINDMAP_MARKER_GROUPS, MINDMAP_MARKERS } from './markers.js';
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

/** 一次查找命中的位置。**偏移是段内的**（相对那个节点的 topic），不是整篇的。 */
interface TopicHit {
  id: string;
  from: number;
  to: number;
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
 * 在所有节点的 `topic` 里查找。
 *
 * ## 为什么逐节点查找，而不是拼成一篇大文本再查
 *
 * 拼成一篇会**产生跨节点的匹配**：「A 结尾的字符 + B 开头的字符」会被当成一处命中，
 * 而那一处在界面上根本不存在（两个节点之间隔着连线和留白）。
 * 逐节点查还顺带解决了偏移映射 —— 每个匹配的偏移天然就是它所属节点的段内偏移，
 * 不需要维护一张「段起点表」（那张表在 topic 里含换行时还会算错）。
 *
 * ## 只搜 `topic`
 *
 * 不搜 `note`：内核**不渲染** `note`（它在数据里，界面上看不见）。
 * 搜一个用户看不见的东西，会得到「找到 3 处，但一处都看不到」——
 * 那比不搜更让人困惑。
 */
function collectHits(root: MindmapNode, options: FindOptions): TopicHit[] {
  const hits: TopicHit[] = [];
  const walk = (node: MindmapNode): void => {
    for (const match of findInText(node.topic, options)) {
      hits.push({ id: node.id, from: match.from, to: match.to });
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

function MindmapEditor({ initialValue, onChange, onReady, onError, readOnly, resolveImageUrl, onImagePaste }: EditorMountProps) {
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
          // 「图片…」追加在菜单末尾（ADR 0053 / 0054），label 走 `t()` ——
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

      // --- 查找 ---------------------------------------------------------------
      const search = createTopicSearch({
        readDocument: () => document,
        patchTopic: (id, topic) => {
          const node = findNodeById(document.root, id);
          if (node === null) return;
          node.topic = topic;
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
        // 标记 / 图片两条内核自有 UI 的落盘通道跟着内核一起下岗（新 effect 会装新的）；
        // 弹层若还开着就关掉 —— 它闭住的那个内核已经销毁了。
        markerApplyRef.current = null;
        markerClearRef.current = null;
        imageRemoveRef.current = null;
        setMarkerPalette(null);
        setImagePopover(null);
        mind.bus.removeListener('operation', onOperation);
        mind.bus.removeListener('expandNode', emit);
        mind.bus.removeListener('changeDirection', emit);
        mind.bus.removeListener('linkDiv', scheduleVerify);
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

  return (
    <>
      <div ref={hostRef} className="qm-mindmap" tabIndex={-1} />
      {markerPalette !== null && (
        <MarkerPalette state={markerPalette} onToggle={toggleMarker} onClear={clearMarkers} onClose={closeMarkerPalette} />
      )}
      {imagePopover !== null && (
        <ImagePopover state={imagePopover} onRemove={removeImage} onClose={closeImagePopover} />
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

interface TopicSearchDeps {
  readDocument: () => MindmapDocument;
  patchTopic: (id: string, topic: string) => void;
  mind: MindElixirInstance;
}

/**
 * 导图里的查找替换。
 *
 * 与代码内核那份的差别只有「匹配在哪里找」：代码内核在**一篇文本**里找，
 * 这里在**每个节点的 topic** 里各找一遍（理由见 `collectHits`）。
 * 导航顺序同样是文档顺序（先序遍历），因为契约要求「顺序即导航顺序」。
 */
function createTopicSearch({ readDocument, patchTopic, mind }: TopicSearchDeps): EditorSearch {
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
    },

    replace: (index, replacement) => {
      const hit = hits[index];
      if (!hit || options === null) return hits.length;
      const node = findNodeById(readDocument().root, hit.id);
      const el = focusHit(hit);
      if (node === null || el === null) return hits.length;

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
        // 对每个节点整段替换一次，语义最准（与代码内核那边同一条理由）。
        const targets = new Map<string, string>();
        for (const hit of hits) {
          const node = findNodeById(readDocument().root, hit.id);
          if (node === null) continue;
          const source = targets.get(hit.id) ?? node.topic;
          targets.set(hit.id, replaceInText(source, options, replacement).text);
        }
        for (const [id, topic] of targets) {
          patchTopic(id, topic);
          const el = mind.findEle(id) as KernelTopic | undefined;
          if (el) void mind.setNodeTopic(el, topic);
        }
        hits = [];
        return targets.size;
      }

      // 字面替换：按**节点**分组，组内**倒序**处理。
      // 倒序是必须的 —— 同一条 topic 上有两处命中时，先替换前面那处会让
      // 后面那处的偏移失效（替换文本长度一变，后面的下标就全错了）。
      const byNode = new Map<string, TopicHit[]>();
      for (const hit of hits) {
        const list = byNode.get(hit.id);
        if (list === undefined) byNode.set(hit.id, [hit]);
        else list.push(hit);
      }

      const count = hits.length;
      for (const [id, list] of byNode) {
        const node = findNodeById(readDocument().root, id);
        if (node === null) continue;
        let topic = node.topic;
        for (const hit of [...list].sort((a, b) => b.from - a.from)) {
          topic = topic.slice(0, hit.from) + replacement + topic.slice(hit.to);
        }
        patchTopic(id, topic);
        const el = mind.findEle(id) as KernelTopic | undefined;
        if (el) void mind.setNodeTopic(el, topic);
      }
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
