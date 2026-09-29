//! 导出（P2 起逐步落地）。
//!
//! 三条导出路线，优先级由「桌面端没有分享链接」这个前提决定（风险 R8）：
//!
//! 1. **自包含 HTML**（P2，最重要）：单文件、内联 CSS 与图片、双击即开。
//!    它承担了「本地不支持分享」之后「把成果发给别人」的角色，属于一等公民。
//!    实现在前端（序列化文档树 → HTML 模板），Rust 侧只负责原子落盘。
//! 2. **SVG / 图表 → PNG**：用 `resvg` + `usvg` crate 编进二进制。
//!    机器上没有 ImageMagick / Inkscape / rsvg-convert，且不该让用户去装 ——
//!    纯 Rust 实现是唯一无外部依赖的选择。字体需内嵌或用系统字体，
//!    中文缺字回退是这块最容易翻车的地方，P2 必须用中英混排样本验一遍。
//! 3. **Markdown → PDF**：走 WebView 的打印管线（WebView2 `PrintToPdf` /
//!    WKWebView 打印），复用渲染结果而不是另做一套排版引擎，保证「所见即所得」。

use crate::error::{AppError, AppResult};

/// SVG 字符串 → PNG 字节。P2 用 `resvg` 实现。
pub fn svg_to_png(_svg: &str, _scale: f32) -> AppResult<Vec<u8>> {
    Err(AppError::not_supported("SVG → PNG 导出将在 P2 提供"))
}

/// 自包含 HTML 已由前端生成，这里只负责原子写入并返回落盘信息。
pub fn write_export(target: &str, bytes: &[u8]) -> AppResult<u64> {
    let path = std::path::PathBuf::from(target);
    super::fs::atomic::write_atomic(&path, bytes)
}
