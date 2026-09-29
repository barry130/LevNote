import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { createDocumentProvider } from './core/platform/index.js';
import { WorkspaceStore } from './core/workspace/store.js';
import { App } from './ui/App.js';
import './ui/styles.css';

const container = document.getElementById('root');
if (!container) throw new Error('找不到 #root 容器，index.html 可能被改坏了');

// 工作区与文档内核都是普通对象而非 React 状态，便于将来把同一套逻辑复用到其他宿主（如 CLI）
const provider = createDocumentProvider();
const workspace = new WorkspaceStore(provider);

createRoot(container).render(
  <StrictMode>
    <App workspace={workspace} provider={provider} />
  </StrictMode>,
);
