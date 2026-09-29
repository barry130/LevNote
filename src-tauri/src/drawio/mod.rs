//! draw.io 宿主（P3 落地），本地静态资源服务器 + 独立窗口。
//!
//! 为什么不是 iframe：
//! draw.io 需要的是一整套带 `eval`、`blob:`、外部字体的运行环境，塞进主窗口的 iframe
//! 就等于把主窗口的 CSP 撕开一个大洞。所以走「独立 webview 窗口 + 127.0.0.1 随机端口」，
//! 主窗口 CSP 保持严格，两边只通过 IPC 交换 `.drawio` 的 XML。
//!
//! 未决风险 R1：draw.io 静态资源体积（数十 MB）与 Tauri「安装包小」的卖点冲突。
//! P0 的结论出来之前不引入资源，因此这里只留接口。

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
