/**
 * 图片的 NodeView：把文档里写的 `src` 换成**真的能加载的 URL**。
 *
 * ## 为什么图片需要一个 NodeView
 *
 * 文档里写的是 `./a.assets/x.png`，而 WebView 的 base URL 是应用自身 ——
 * 把这一串直接塞进 `<img src>` 会去请求应用自己的资源目录，得到 404。
 * 相对路径的解析要「文档在哪」，读文件要平台能力，**两样都只有外壳有**
 * （内核不许碰文件系统）。所以默认渲染出来的 `<img>` 必然是破的。
 *
 * NodeView 是唯一能改这件事的地方：它是叶子节点，没有 `contentDOM`，
 * 整个 DOM 由我们负责。于是可以在 `src` 变化时异步向外壳要一个可用的 URL
 * 再设上去，拿不到就画一个**说明原因的占位**。
 *
 * ## 为什么不像表格那样用装饰
 *
 * 装饰的输入必须是**同步**算出来的（它住在 state 里），而读文件是异步的。
 * 硬要塞进去就得把「已解析的 URL」做成一份 state，还要处理
 * 「文件在磁盘上被改了」的失效 —— 那比一个 NodeView 复杂得多。
 *
 * ## 这个占位不是装饰品
 *
 * 图片读不到的原因有四种，且**都必须说清楚**（本项目「绝不静默」的红线）：
 * 文档没有目录信息、文件不存在、格式读不出来、外壳没有这个能力。
 * 画一个空框或破图图标等于把「为什么」藏起来 —— 而用户下一步要做的
 * 恰恰是「去把那张图找回来」，他需要知道路径。
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import type { EditorView, NodeView } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import type { ImageInfo } from '../../core/editor/types.js';

/**
 * ProseMirror 的图片节点名（对齐 preset-commonmark 的 schema）。
 *
 * 与 `CODE_BLOCK` 同样的处境：这个字符串静默地出现在 NodeView 注册与
 * `selection.ts` 的 `NODE_FORMATS` 里，上游改名不会有任何报错。
 */
export const IMAGE = 'image';

/** 解析 `src` 的能力，由外壳提供。拿不到就是「这个平台没法显示本地图片」。 */
export type ImageUrlResolver = (src: string) => Promise<string | null>;

/** 点开一张图片的能力，由外壳提供（灯箱）。 */
export type ImageClickHandler = (image: ImageInfo) => void;

/**
 * 交给 NodeView 的两项外壳能力。
 *
 * 两项合成一个对象、用一个取值函数取，而不是两个并列的取值函数：
 * 它们**总是同时被提供或同时缺失**（都是「外壳支持图片吗」的答案），
 * 分开传只会让调用点多写一次 `() => options.x`。
 */
export interface ImageViewHandlers {
  readonly resolve?: ImageUrlResolver;
  readonly onClick?: ImageClickHandler;
}

const imageKey = new PluginKey('levnote-image');

/** `attrs` 是 `any`（ProseMirror 的定义），取值一律收窄，不把未知当字符串用。 */
function readString(attrs: ProseNode['attrs'], key: string): string {
  const value: unknown = attrs[key];
  return typeof value === 'string' ? value : '';
}

/**
 * 图片的视图。
 *
 * 状态机只有三格，用 `dom` 上的类名表示，CSS 负责画：
 *
 * - `is-loading` —— 正在向外壳要 URL（本地图片要读盘，不是瞬时的）
 * - `is-loaded` —— 拿到了，`<img>` 有 `src`
 * - `is-failed` —— 拿不到，显示带路径的说明
 */
class ImageView implements NodeView {
  readonly dom: HTMLElement;
  private readonly el: HTMLImageElement;
  private readonly note: HTMLElement;

  /** 已经发起解析的那个 `src`；用来判断 `update` 里要不要重新解析。 */
  private pending: string | null = null;
  /**
   * 竞态令牌。
   *
   * 用户可能连改两次路径，而读盘是异步的 —— 先发的请求后回来就会把
   * **过期的 URL** 盖到新的图上。每次发起解析都自增，回来时对不上就丢弃。
   * 这与 `DocumentPane` 的 `handleKeyRef` 是同一类问题（异步结果属于哪一次请求）。
   */
  private token = 0;
  private alive = true;
  /**
   * 当前状态。**同时**记在字段里，而不只是写进 `classList`。
   *
   * 点击处理要知道「现在是不是加载好了」，从 `classList.contains(...)` 反推
   * 等于把「状态」这个概念只存在 DOM 上 —— 而 DOM 是渲染结果，不是真相。
   * 一个字段的成本，换来「问状态」不需要读 DOM。
   */
  private state: 'is-loading' | 'is-loaded' | 'is-failed' = 'is-loading';

  constructor(
    private node: ProseNode,
    private readonly getHandlers: () => ImageViewHandlers,
  ) {
    this.dom = document.createElement('span');
    this.dom.className = 'qm-img';

    this.el = document.createElement('img');
    this.el.className = 'qm-img-el';
    // 与 ProseMirror 自带 image 的 toDOM 一致，节点拖拽才认得它
    this.el.draggable = true;
    this.el.alt = readString(node.attrs, 'alt');

    /**
     * 点开灯箱。
     *
     * 监听挂在**我们自己造的这个元素**上，而不是让外壳去编辑器容器上做事件委托 ——
     * 后者要求外壳认识 `qm-img-el` 这个类名，而外壳不许 import 具体内核。
     * 见 `EditorMountProps.onImageClick` 的注释。
     *
     * `stopPropagation` 是必要的：不拦的话这次点击会继续走到 ProseMirror 的
     * 选区处理上，把光标放到图片旁边 —— 用户只是想看大图，却顺带改了光标位置。
     *
     * 只有**真的加载出来了**才响应：占位上点一下弹出一个空白灯箱，
     * 比不弹更让人困惑（明明写着「图片读不到」）。
     */
    this.el.addEventListener('click', (event) => {
      const handler = this.getHandlers().onClick;
      if (handler === undefined || this.state !== 'is-loaded') return;
      event.preventDefault();
      event.stopPropagation();
      handler({
        url: this.el.src,
        src: readString(this.node.attrs, 'src'),
        alt: readString(this.node.attrs, 'alt'),
        title: readString(this.node.attrs, 'title'),
      });
    });

    // 占位里的文字**不可编辑**：它是我们画的说明，不是文档内容。
    // 少了这一句，用户可以在占位里打字，而那些字既存不进文档也删不掉。
    this.note = document.createElement('span');
    this.note.className = 'qm-img-note';
    this.note.contentEditable = 'false';

    this.dom.append(this.el, this.note);
    this.applyMeta();
    void this.load();
  }

  /** alt / title 是纯属性，直接写，不需要重新读盘。 */
  private applyMeta(): void {
    const title = readString(this.node.attrs, 'title');
    this.el.alt = readString(this.node.attrs, 'alt');
    if (title === '') this.el.removeAttribute('title');
    else this.el.setAttribute('title', title);
  }

  private setState(state: 'is-loading' | 'is-loaded' | 'is-failed'): void {
    this.state = state;
    this.dom.classList.remove('is-loading', 'is-loaded', 'is-failed');
    this.dom.classList.add(state);
  }

  private async load(): Promise<void> {
    const src = readString(this.node.attrs, 'src');
    this.pending = src;

    if (src === '') {
      this.fail('图片没有地址');
      return;
    }

    const resolver = this.getHandlers().resolve;
    if (resolver === undefined) {
      this.fail(`无法显示本地图片：${src}`);
      return;
    }

    const mine = (this.token += 1);
    this.setState('is-loading');
    // 读盘不是瞬时的（一次 IPC + 一次 base64）。这段时间里说一句「在读」，
    // 好过让那个位置一片空白 —— 空白看起来像「图没了」
    this.note.textContent = '正在读取图片…';

    let url: string | null;
    try {
      url = await resolver(src);
    } catch {
      // 外壳理论上不会抛（它内部已经把失败收敛成 null），但这里不能假设 ——
      // 一个未捕获的拒绝会变成一个没人处理的 Promise
      url = null;
    }

    // 过期（src 变了）或已销毁：丢掉这次结果，不要写进 DOM
    if (!this.alive || mine !== this.token) return;

    if (url === null || url === '') {
      this.fail(`图片读不到：${src}`);
      return;
    }

    this.el.src = url;
    this.note.textContent = '';
    this.setState('is-loaded');
  }

  private fail(message: string): void {
    this.el.removeAttribute('src');
    this.note.textContent = message;
    this.setState('is-failed');
  }

  update(node: ProseNode): boolean {
    // 只处理图片节点。别的类型交给 ProseMirror 重建视图 ——
    // 返回 true 说「我还能用」会把一个不同类型的节点塞给这个视图
    if (node.type.name !== IMAGE) return false;

    const nextSrc = readString(node.attrs, 'src');
    const changed = nextSrc !== this.pending;
    this.node = node;
    this.applyMeta();

    // `update` 在**每一次文档变更**时都会被调用（包括在别处打字）。
    // 只有路径真的变了才重新读盘，否则每敲一个字都要发起一次文件读取
    if (changed) void this.load();

    return true;
  }

  /**
   * 这个 NodeView 里的所有变更都由我们自己产生（换 `src`、改类名、写说明），
   * 一律忽略。
   *
   * 图片是叶子节点，没有 `contentDOM`，所以「不在 contentDOM 里就不算」这条
   * 简化成了「全都不算」。少了它，ProseMirror 会把我们写进 `<img>` 的
   * **data URI 当成用户输入读进文档** —— 存出来的 Markdown 里会多出一串 base64。
   */
  ignoreMutation(): boolean {
    return true;
  }

  destroy(): void {
    this.alive = false;
  }
}

/**
 * 注册图片的 NodeView。
 *
 * `getHandlers` 是一个**取值函数**而不是一个值：外壳的回调在每次 React 渲染时
 * 都是新的，而插件只在创建时装配一次。传函数就永远读得到当下那一个
 * （与 `MarkdownEditor.tsx` 里用 ref 转发回调是同一个理由）。
 */
export function imageViewPlugin(getHandlers: () => ImageViewHandlers) {
  return $prose(
    () =>
      new Plugin({
        key: imageKey,
        props: {
          nodeViews: {
            [IMAGE]: (node, _view: EditorView) => new ImageView(node, getHandlers),
          },
        },
      }),
  );
}
