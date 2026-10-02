//! draw.io 宿主（P3 落地），本地静态资源服务器 + 独立窗口。
//!
//! 为什么不是 iframe：
//! draw.io 需要的是一整套带 `eval`、`blob:`、外部字体的运行环境，塞进主窗口的 iframe
//! 就等于把主窗口的 CSP 撕开一个大洞。所以走「独立 webview 窗口 + 127.0.0.1 随机端口」，
//! 主窗口 CSP 保持严格，两边只通过 IPC 交换 `.drawio` 的 XML。
//!
//! 未决风险 R1：draw.io 静态资源体积（数十 MB）与 Tauri「安装包小」的卖点冲突。
//! **2026-10-02 实测（V1）：只读查看器这一条路是小的 —— `viewer.min.js` 2.654 MiB
//! （gzip 0.684），按库内置 bpmn/flowchart 后合计 2.914 MiB。数十 MB 那个量级属于
//! 「完整编辑器」，而它走的是按需下载（ADR 0005）。R1 的射程因此收窄到编辑器那一侧。**
//! 详见 `docs/decisions/0005-drawio-integration-route.md` 文末的「V1 实测记录」。
//!
//! 那条记录里还有两条 P5 落地时必须遵守的硬要求，别在实现时忘掉：
//! ① 那 10 个资源路径（`SHAPES_PATH` / `STENCIL_PATH` / …）**必须全部指到本地** ——
//!    不占位时实测会去 `viewer.diagrams.net` 取资源，即「纯本地」的产品会静默联网；
//! ② stencil 取不到时 mxGraph **不报错**，会把形状画成普通矩形而标签照旧 ——
//!    必须在 UI 上说清楚（硬约束 2「绝不静默」）。

use crate::error::{AppError, AppResult};

pub struct DrawioHost {
    pub port: u16,
    pub url: String,
}

/// 启动本地资源服务。计划返回随机端口，避免与其他程序冲突，并只监听 127.0.0.1。
pub fn start_host() -> AppResult<DrawioHost> {
    Err(AppError::not_supported(
        "draw.io 宿主将在 P3 提供；需先完成 P0 的资源体积评估（风险 R1）",
    ))
}
