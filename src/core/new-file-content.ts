/**
 * 「新建一个文件时，该往里写什么」——**一条出口**（ADR 0048）。
 *
 * ## 为什么要有这一层
 *
 * 「新建」在这个产品里发生过一次真实分歧：命令面板的 `file.new` 走
 * `writeTextFile({ text: '' })`，侧栏的「新建文件」走 Rust 的 `create_file`，
 * 两处各写各的。结果是**同一个 `.mindmap` 从两条路出来不一样** ——
 * 一条落模板、一条落空文件，而两边都觉得自己是对的。
 *
 * 分歧的代价不在「空文件」本身：`.mindmap` 的空文件**完全合法**
 * （`describeMindmapVeto` 对空文本返回 `null`，画布会接管）。
 * 代价在**将来每加一种有格式的文件类型，都要改两个地方**，
 * 而且很容易只改一处 —— 那就是一个「看起来能开、打开是空白」的类型。
 *
 * 所以这里收成一条：按**路径**决定初始内容，调用方只管拿 `path` 来问。
 *
 * ## 判据是**扩展名**，不是内核
 *
 * 判断用的全是 `core/path.ts` 的 `is*Path`，没有一个 import 来自 `modules/` ——
 * `core/` 不该知道有哪些内核（硬约束 1，ADR 0007 后果 2 在格式层上的延续）。
 * 顺序上「先判是不是我们认识的格式，不是就落空文件」：
 * 空文件对**任何**扩展名都是合法起点，所以未知类型不需要额外解释。
 */

import { isDrawioPath, isMindmapPath, titleOf } from './path.js';
import { drawioTemplateXml } from './drawio.js';
import { emptyMindmapText } from './mindmap/template.js';

/**
 * 新建 `path` 时该写进去的初始内容。
 *
 * **中心主题默认取文件名**（`会议记录.mindmap` → 中心节点写「会议记录」）：
 * 那不是我们替用户编的内容，是他自己刚起的名字；
 * 而且导图画布打开时正需要一个能看见、点进去就能改的靶子。
 *
 * 用户把文件名叫成 `.mindmap`（纯后缀、没有主名）时，`titleOf` 按它自己的
 * `dot <= 0` 判定会把整个 `.mindmap` 留下，于是中心主题就是 `.mindmap` ——
 * 那仍然是用户自己起的名字，**替他修饰反而是替他编内容**，所以照原样用。
 *
 * `.drawio` 落一份**真的能画出来**的最小图纸（`drawioTemplateXml`）：与导图
 * 模板同一个理由 —— 落一个空文件的话，用户打开看到的是一句「这份图纸还是空的」，
 * 而「新建」本该给一份能立刻上手的起点。顺带这一份让 `<svg>` 立刻就有内容，
 * 「放大看图」那条路也就通了。
 */
export function newFileContent(path: string): string {
  if (isMindmapPath(path)) return emptyMindmapText(titleOf(path));
  if (isDrawioPath(path)) return drawioTemplateXml();
  return '';
}