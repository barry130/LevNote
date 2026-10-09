// @vitest-environment jsdom
/**
 * 模块装配的**接线测试**。
 *
 * 为什么单独开一个文件：`markdown-core.test.ts` 测的是「内核能不能正确解析/序列化」，
 * 内核的测试全绿而应用不工作，最常见的原因就是**装配错了** —— 注册顺序反了、
 * 两个模块各自维护了一份扩展名名单、源码视图指向了不存在的东西。
 * 这些都不是任何单个模块的内部逻辑能暴露出来的问题。
 *
 * 所以这里只问三类问题：名单是不是同一份、源码视图指向谁、注册表里到底有什么。
 * 不测任何解析行为 —— 那是内核测试的事。
 */

import { beforeAll, describe, expect, it } from 'vitest';

import { listAdapters, resolveAdapter } from '../core/editor/registry.js';
import { MARKDOWN_EXTENSIONS } from '../core/path.js';
import { codeAdapter } from './code/index.js';
import { languageForPath } from './code/setup.js';
import { EDITOR_FALLBACK, registerBuiltInEditors } from './index.js';
import { markdownAdapter } from './markdown/index.js';
import { mindmapAdapter } from './mindmap/index.js';

describe('Markdown 扩展名名单只有一份', () => {
  /**
   * 这条测试的价值在于它**跨模块**：名单本来抄在三个地方
   * （Markdown 内核、代码内核、外壳的大纲判断），源码视图又需要第四处。
   * 抄多份的真实后果是双重落空 —— Markdown 内核认为「这不是 .md」于是退位，
   * 代码内核也不认它于是继续往下掉，用户拿到一个白底 textarea，
   * 而两个内核的代码看上去都没写错。收敛到 `core/path.ts` 之后由这条测试钉住。
   */
  it('名单里的每一个扩展名，Markdown 内核与代码内核都认', () => {
    for (const ext of MARKDOWN_EXTENSIONS) {
      const path = `笔记${ext}`;
      expect(markdownAdapter.matches(path), `${ext} 应被 Markdown 内核认领`).toBe(true);
      // 代码内核认领它是为了**源码视图**（front matter 文件则是靠这个接住降级）
      expect(codeAdapter.matches(path), `${ext} 应被代码内核认领`).toBe(true);
    }
  });

  it('每一个 Markdown 扩展名在源码视图里都有 markdown 语法着色', () => {
    // 漏掉 `.mdown` 这类冷门后缀的表现是「同一个文档切到源码视图突然没有着色」，
    // 用户会以为源码视图坏了 —— 而不是以为「这个后缀没配语法包」。
    for (const ext of MARKDOWN_EXTENSIONS) {
      expect(languageForPath(`笔记${ext}`), ext).toBe('markdown');
    }
  });
});

describe('源码视图的接线', () => {
  it('Markdown 内核的源码视图就是代码内核', () => {
    // 这条断言看着平淡，但它就是 Ctrl+/ 能不能用的全部依据：
    // 外壳靠 `primary.sourceView !== undefined` 决定那个开关亮不亮。
    expect(markdownAdapter.sourceView).toBe(codeAdapter);
  });

  it('代码内核与纯文本内核没有源码视图 —— 它们本身就是源码', () => {
    expect(codeAdapter.sourceView).toBeUndefined();
    expect(EDITOR_FALLBACK.sourceView).toBeUndefined();
  });

  it('源码视图不注册进注册表 —— 否则同一个 .md 会有两个合法答案', async () => {
    await registerBuiltInEditors();

    // 顺序即优先级：markdown 在前（`.md` 归富文本内核），
    // mindmap 在中间（`.mindmap` 归导图内核），
    // code 在**最后**（它是备胎，认得前两个认得的所有扩展名，排前面会把它们全吃掉）。
    // 源码视图**不在**这张表里，它只挂在适配器对象上由外壳显式取用 ——
    // 塞进注册表的话，`resolveAdapter('a.md')` 就有了两个合法答案，
    // 哪一个先匹配到取决于注册顺序，那是没法推理的。
    expect(listAdapters().map((adapter) => adapter.id)).toEqual(['markdown', 'mindmap', 'code']);
  });

  it('导图内核的源码视图也是代码内核 —— 于是它自动获得视图三态', () => {
    // `.mindmap` 的源码形态就是 JSON，而轻笺已经有一个 JSON 编辑器了。
    // 外壳只看 `sourceView` 在不在，所以这一条就是「导图能不能切到源码视图」的全部依据。
    expect(mindmapAdapter.sourceView).toBe(codeAdapter);
    expect(languageForPath('图.mindmap')).toBe('json');
  });
});

describe('导图的备胎链', () => {
  // 注册是异步的（内核走动态 import，见 index.ts），断言前必须等它落地
  beforeAll(async () => {
    await registerBuiltInEditors();
  });

  const pick = (text?: string): string => resolveAdapter('图.mindmap', EDITOR_FALLBACK, text).id;

  const body = (over: Record<string, unknown> = {}): string =>
    JSON.stringify({ format: 'levnote.mindmap', version: 1, root: { id: 'n1', topic: '根', children: [] }, ...over });

  it('正常的 .mindmap 归导图内核', () => {
    expect(pick(body())).toBe('mindmap');
  });

  it('空文件也归导图内核 —— 那不是坏文件，是「还没开始写」', () => {
    expect(pick('')).toBe('mindmap');
    expect(pick('\n  \n')).toBe('mindmap');
  });

  it('读不出来的 .mindmap 落到代码内核，而不是兜底纯文本', () => {
    // 一级比一级少，但一级都不会打不开。退到代码内核意味着用户看到的
    // 是一份**有 JSON 着色**的源码，能直接改；退到纯文本则是白底 textarea。
    expect(pick('{ 这不是 JSON')).toBe('code');
    expect(pick(JSON.stringify({ format: '别的格式', version: 1, root: {} }))).toBe('code');
    expect(pick(JSON.stringify({ format: 'levnote.mindmap', version: 1 }))).toBe('code');
  });

  it('节点 id 重复的文件也落到代码内核 —— 拖动会作用到别的节点上', () => {
    const duplicated = body({
      root: {
        id: 'n1',
        topic: '根',
        children: [
          { id: 'same', topic: '甲', children: [] },
          { id: 'same', topic: '乙', children: [] },
        ],
      },
    });
    expect(pick(duplicated)).toBe('code');
  });

  it('代码内核认得 .mindmap —— 备胎要认得正主认得的东西', () => {
    expect(codeAdapter.matches('图.mindmap')).toBe(true);
    expect(mindmapAdapter.matches('图.mindmap')).toBe(true);
    // 反过来不成立：导图内核不该去抢 `.md`
    expect(mindmapAdapter.matches('笔记.md')).toBe(false);
  });
});
