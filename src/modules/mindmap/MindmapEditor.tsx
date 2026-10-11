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
 */

import MindElixir, { type MindElixirInstance, type Operation } from 'mind-elixir';
import 'mind-elixir/style';
import { useEffect, useRef } from 'react';

import type { EditorHandle, EditorMountProps, EditorSearch } from '../../core/editor/types.js';
import { parseMindmap } from '../../core/mindmap/parse.js';
import { serializeMindmap } from '../../core/mindmap/serialize.js';
import { emptyMindmapDocument } from '../../core/mindmap/template.js';
import { type LineEnding, type MindmapDocument, type MindmapNode } from '../../core/mindmap/types.js';
import { createFindPattern, findInText, replaceInText, type FindOptions } from '../../core/search/find.js';
import { directionOf, fromKernelData, toKernelData } from './adapter.js';
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

function MindmapEditor({ initialValue, onChange, onReady, onError, readOnly }: EditorMountProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);

  // 回调每次渲染都是新函数，直接进依赖会让内核被反复重建
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;

  const initialValueRef = useRef(initialValue);
  const readOnlyRef = useRef(readOnly);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    // 空内容给一份默认文档；其余情况读不出来就是真读不出来 ——
    // 那条路 `canHandle` 已经拦掉了，走到这里说明有个我们没预料到的洞，
    // 如实报错好过白屏（「打开文件不能失败」这条承诺在注册表那一层兑现）。
    const empty = initialValueRef.current.trim() === '';
    const parsed = empty
      ? { document: createEmptyDocument(), lineEnding: '\n' as LineEnding }
      : parseMindmap(initialValueRef.current);
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
    const originalText = empty ? null : initialValueRef.current;
    const lineEnding = parsed.lineEnding;

    let document = first;

    let mind: MindElixirInstance;
    try {
      mind = new MindElixir({
        el: host,
        direction: directionOf(document.layout),
        // 只读时三样一起关：`editable` 管拖拽与新建，`contextMenu` 管右键菜单，
        // `keypress` 管 F2 / Enter 这些进编辑态的键。**编辑节点文字的入口只有 `keypress`**
        // （`editTopic` 本身不检查 `editable`，是调用它的地方检查的），
        // 所以这三个必须成套关 —— 少关一个就能改到内容。
        editable: !readOnlyRef.current,
        contextMenu: !readOnlyRef.current,
        keypress: !readOnlyRef.current,
        // 内核自己的撤销栈：导图的编辑动作（新建 / 删除 / 拖动 / 改文字）都在内核里，
        // 外壳的撤销栈够不着它们，所以交给内核自己管。
        allowUndo: true,
        // 内核的右下角浮动工具条（缩放、回到中心）保留：它是缩放唯一的可见入口。
        toolBar: true,
        theme: LEVNOTE_THEME,
      });
      mind.init(toKernelData(document));
    } catch (thrown) {
      onErrorRef.current?.(thrown instanceof Error ? thrown : new Error(String(thrown)));
      return;
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
        return originalText !== null && lastText === canonicalOriginal ? originalText : lastText;
      },

      setValue: (text) => {
        const next = text.trim() === '' ? { document: createEmptyDocument(), lineEnding } : parseMindmap(text);
        if (next.document === null) return;
        document = next.document;
        // `refresh()` 换数据但**不换方向**（读过源码确认），所以要显式设一次
        mind.direction = directionOf(document.layout);
        mind.refresh(toKernelData(document));
        lastText = serializeMindmap(document, { lineEnding: next.lineEnding });
      },

      focus: () => host.focus(),

      /**
       * 导图没有「语义 HTML」可给 —— 它的呈现是一张由绝对定位的 div 与 SVG 组成的图，
       * 而 `toHtml()` 要的是能进导出文档的**内容**片段。
       *
       * 返回 `null` 是**合法结果**（见契约）：自包含导出会退回用 `<pre>` 呈现 JSON 原文，
       * 并如实说明，而不是伪造一份「看起来像图」的 HTML。
       * 真要做富导出，正解是内核的 `exportSvg()` 那条路，那是另一个决定（要处理
       * `foreignObject` 的兼容性与样式内联），不在「接入内核」这件事的射程内。
       */
      toHtml: async () => null,

      search,
    };

    onReadyRef.current(handle);

    return () => {
      if (verifyTimer !== null) clearTimeout(verifyTimer);
      mind.bus.removeListener('operation', onOperation);
      mind.bus.removeListener('expandNode', emit);
      mind.bus.removeListener('changeDirection', emit);
      mind.bus.removeListener('linkDiv', scheduleVerify);
      mind.destroy();
    };
    // 空依赖：内核只在挂载时创建一次，外部内容变更一律走 handle.setValue
  }, []);

  return <div ref={hostRef} className="qm-mindmap" tabIndex={-1} />;
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
