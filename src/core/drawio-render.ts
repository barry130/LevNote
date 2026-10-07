/**
 * 把一段 `.drawio` XML 画进容器 —— 内置只读查看器的渲染入口（P5 第 25 项）。
 *
 * ## 为什么查看器是**运行时脚本注入**，而不是打包进 bundle
 *
 * `viewer.min.js` 有 2.654 MiB，是 draw.io 官方**未改动**的构建产物
 * （ADR 0005：上游不接受 PR，改动只允许在配置层）。它不能进主包 ——
 * 静态 import 会让每一个打开轻墨的人先解析 2.6 MB，哪怕一篇文档一张图都没有。
 * 所以文件随应用分发（`public/drawio/`，纯本地、无网络），但**只在文档里
 * 真的出现 `doc-block` 图表引用时**才用 `<script>` 注入一次。
 *
 * ## 那十个资源路径是「无网络」的唯一开关
 *
 * CDN 版产物把每条资源路径硬写成绝对地址并可覆盖
 * （`window.X = window.X || "https://viewer.diagrams.net/…"`）。加载前必须把
 * 十条路径全部指到本地 —— 漏一条，打开一张带 stencil 的图就会**静默联网**，
 * 与 ADR 0001 直接冲突。清单与 V1 实测的对照写在 ADR 0027。
 *
 * ## 为什么是 `createViewerForElement` 而不是 `processElements`
 *
 * `processElements` 是给静态 HTML 用的批量入口，不把**查看器实例**交出来；
 * 只有 `createViewerForElement` 有回调，多页信息、渲染是否成功都要从实例这边看
 * （V1 的测试页就是这么驱动的，`scripts/v1-page.html`）。
 *
 * ## 为什么渲染**不接主题**
 *
 * mermaid 的颜色由渲染时的主题决定，所以 `renderMermaid` 必须收主题；
 * drawio 的颜色**写在图自己的 XML 里**，画布恒为白 —— 换明暗重画只会
 * 得到同一张图。刻意不设 `theme` 参数：与 `renderMermaid` 的「不设默认值」
 * 是同一条纪律，只不过方向相反 —— 这里连参数都不该有。
 */

import { t } from './i18n/index.js';

/**
 * 查看器全局暴露的那个入口。
 *
 * 做成**结构类型**：测试里造一个假实现不必把 viewer.min.js 的全部导出凑齐。
 */
export interface DrawioViewerLike {
  readonly createViewerForElement: (
    element: HTMLElement,
    onReady: (viewer: unknown) => void,
  ) => void;
}

/**
 * 渲染需要的两件外部能力，做成注入的（与 `MermaidIo` 同一条理由）：
 * `loadViewer` 换掉之后测试不必真的去加载 2.6 MB 的脚本；
 * `settle` 单独存在是因为「等待渲染收尾」与「加载」是两件事 ——
 * 真实查看器画完后还会补一次布局（等两帧），而测试里要的是立刻返回。
 */
export interface DrawIoIo {
  readonly loadViewer: () => Promise<DrawioViewerLike>;
  /** 渲染回调之后的收尾等待。真实实现等两帧；测试注入立刻完成。 */
  readonly settle: () => Promise<void>;
}

/** 渲染成功。`svg` 是当前页的 `<svg>` 标记，供「放大看」复用；空页是 `null`。 */
export interface DrawioRenderOk {
  readonly ok: true;
  readonly svg: string | null;
}

/** 渲染失败。`message` 是可以直接显示给用户的一句话 —— 失败绝不吞掉。 */
export interface DrawioRenderFail {
  readonly ok: false;
  readonly message: string;
}

export type DrawioRender = DrawioRenderOk | DrawioRenderFail;


/** 查看器回调一直没有回来时的兜底等待。真画一张大图也不该超过它。 */
const RENDER_TIMEOUT_MS = 15_000;

// ---------------------------------------------------------------------------
// 真实环境下的加载
// ---------------------------------------------------------------------------

/** 随应用分发的查看器与形状资源（`public/drawio/`，见 `vite.config.ts` 的 publicDir）。 */
export const DRAWIO_ASSET_BASE = `${import.meta.env.BASE_URL}drawio`;

/**
 * 加载前必须占住的资源路径 —— **全部十条**。
 *
 * 这份清单与 `scripts/v1-page.html` 的占位一一对应；上游产物里每一条都是
 * `window.X || "绝对地址"` 的写法，所以只要我们先赋值，它就不会去联网。
 * `DRAWIO_BASE_URL` / `DRAWIO_LOG_URL` 是新式命名（遥测与代理），给空串
 * 让查看器走「没配置」的分支，而不是去拼一个 `undefined` 出来的坏地址。
 */
function installResourcePaths(): void {
  const w = window as unknown as Record<string, unknown>;
  w['SHAPES_PATH'] = `${DRAWIO_ASSET_BASE}/shapes`;
  w['STENCIL_PATH'] = `${DRAWIO_ASSET_BASE}/stencils`;
  w['STYLE_PATH'] = `${DRAWIO_ASSET_BASE}/styles`;
  w['PROXY_URL'] = '';
  w['DRAW_MATH_URL'] = '';
  w['GRAPH_IMAGE_PATH'] = `${DRAWIO_ASSET_BASE}/img`;
  w['RESOURCES_PATH'] = `${DRAWIO_ASSET_BASE}/resources`;
  w['RESOURCE_BASE'] = `${DRAWIO_ASSET_BASE}/resources/grapheditor`;
  w['mxImageBasePath'] = `${DRAWIO_ASSET_BASE}/mxgraph/images`;
  w['mxBasePath'] = `${DRAWIO_ASSET_BASE}/mxgraph`;
  w['DRAWIO_BASE_URL'] = '';
  w['DRAWIO_LOG_URL'] = '';
  // 拦住自动处理：脚本末尾没有这个回调会立刻 processElements()，
  // 而我们要自己控制「什么时候、对哪个元素」渲染（V1 测试页同款）
  w['onDrawioViewerLoad'] = () => undefined;
}

/**
 * 「加载一次」的缓存，与 `mermaid-render.cachedLoader` 同一条规则：
 * 缓存 **Promise** 去重并发；**失败不缓存**，修好之后下一次渲染会重试。
 */
const loadViewerOnce = (): Promise<DrawioViewerLike> => {
  return new Promise((resolve, reject) => {
    installResourcePaths();
    const script = document.createElement('script');
    script.src = `${DRAWIO_ASSET_BASE}/viewer.min.js`;
    script.onload = () => {
      const viewer = (window as unknown as Record<string, unknown>)['GraphViewer'];
      if (viewer === undefined || viewer === null) {
        reject(new Error(t('core.render.drawioNoEntry')));
        return;
      }
      resolve(viewer as DrawioViewerLike);
    };
    script.onerror = () => {
      reject(new Error(t('core.render.drawioLoadFailed')));
    };
    document.head.appendChild(script);
  });
};

let pendingViewer: Promise<DrawioViewerLike> | null = null;

function defaultLoadViewer(): Promise<DrawioViewerLike> {
  pendingViewer ??= loadViewerOnce().catch((thrown: unknown) => {
    pendingViewer = null;
    throw thrown;
  });
  return pendingViewer;
}

function defaultSettle(): Promise<void> {
  // 两帧：mxGraph 在回调之后还会补一次布局（标签、自适应）——
  // V1 实测就是这么取到最终 DOM 的
  return new Promise((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve());
    });
  });
}

const DEFAULT_IO: DrawIoIo = { loadViewer: defaultLoadViewer, settle: defaultSettle };

// ---------------------------------------------------------------------------
// 渲染
// ---------------------------------------------------------------------------

/**
 * 把一份 `.drawio` XML 画进容器。
 *
 * 容器的**内容归调用方管理**：本函数往里追加一个 `.mxgraph` 元素（查看器
 * 画在里面），重画前由调用方清空。成功时容器里的查看器 DOM **留着** ——
 * 多页切换、缩放那些交互就在它上面；返回的 `svg` 只是给「放大看」的快照。
 */
export async function renderDrawio(
  xml: string,
  container: HTMLElement,
  io: DrawIoIo = DEFAULT_IO,
): Promise<DrawioRender> {
  if (xml.trim() === '') return { ok: false, message: t('core.render.drawioEmpty') };

  let viewer: DrawioViewerLike;
  try {
    viewer = await io.loadViewer();
  } catch (thrown) {
    return { ok: false, message: t('core.render.drawioFailed', { reason: describe(thrown) }) };
  }

  const host = document.createElement('div');
  host.className = 'mxgraph';
  // 与 V1 测试页同一份配置：`nav` 给出工具栏（多页切换、缩放），`resize` 自适应
  host.setAttribute('data-mxgraph', JSON.stringify({ nav: true, resize: true, xml }));
  container.appendChild(host);

  try {
    await withTimeout(
      (async () => {
        await new Promise<void>((resolve, reject) => {
          try {
            viewer.createViewerForElement(host, () => resolve());
          } catch (thrown) {
            // 拒绝理由必须是 Error：非 Error 的抛出物裹一层，外层统一 describe
            reject(thrown instanceof Error ? thrown : new Error(String(thrown)));
          }
        });
        await io.settle();
      })(),
    );
  } catch (thrown) {
    // 查看器渲染失败时会把错误写进元素（如「不是合法的 XML」），留着它比清空好
    const detail = host.textContent?.trim();
    const reason = describe(thrown);
    return { ok: false, message: detail !== '' ? `${reason} ${detail}` : reason };
  }

  const svg = host.querySelector('svg');
  return { ok: true, svg: svg === null ? null : svg.outerHTML };
}

async function withTimeout(run: Promise<void>): Promise<void> {
  let timer: ReturnType<typeof setTimeout> | null = null;
  try {
    await Promise.race([
      run,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error(t('core.render.timeout'))), RENDER_TIMEOUT_MS);
      }),
    ]);
  } finally {
    if (timer !== null) clearTimeout(timer);
  }
}

/** 兜底：`String(thrown)` 对非 Error 也能给出一句话，而不是 `[object Object]`。 */
function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
