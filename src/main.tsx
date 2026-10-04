import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { createDocumentProvider } from './core/platform/index.js';
import { WorkspaceStore } from './core/workspace/store.js';
import { registerBuiltInEditors } from './modules/index.js';
import { App } from './ui/App.js';
import './ui/styles.css';

const container = document.getElementById('root');
if (!container) throw new Error('找不到 #root 容器，index.html 可能被改坏了');

// 工作区与文档内核都是普通对象而非 React 状态，便于将来把同一套逻辑复用到其他宿主（如 CLI）
const provider = createDocumentProvider();
const workspace = new WorkspaceStore(provider);

/**
 * 内核注册完成**之前**不许渲染第一帧（P3 第 18 项的「注册等待态」）。
 *
 * 注册表现在是异步的（三个内核走动态 import、各自成 chunk），而
 * `resolveAdapter` 是同步查表：表还是空的就渲染，`.md` 会被静默兜到
 * 纯文本上，等内核到了再「跳」回富文本 —— 那正是硬约束 3 要杜绝的
 * 「被遮蔽的打开路径」。这里等的就是那张表。
 *
 * `registerBuiltInEditors()` 自身**不会 reject**（单个内核加载失败只会被
 * 记进 `editorRegistrationFailures()`，由 App 的横幅说出来 —— 少一个内核
 * 是降级，降级不许静默，但也不该把整个应用挡在门外：纯文本兜底是静态的，
 * 永远可用）。这层 `catch` 只兜「意外」—— 到了这一步说明启动本身坏了，
 * index.html 里的引导位会如实说。
 */
void registerBuiltInEditors().then(
  () => {
    createRoot(container).render(
      <StrictMode>
        <App workspace={workspace} provider={provider} />
      </StrictMode>,
    );
  },
  (thrown: unknown) => {
    const slot = document.getElementById('boot-status');
    const message = thrown instanceof Error ? thrown.message : String(thrown);
    if (slot !== null) {
      slot.textContent = `启动失败：${message}`;
      slot.removeAttribute('hidden');
    } else {
      throw thrown;
    }
  },
);
