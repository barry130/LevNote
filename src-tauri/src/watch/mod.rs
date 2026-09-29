//! 文件监听（P3 落地）。
//!
//! 目标：既支持「当前文档被外部程序改写」的冲突提示，也支持打开目录树时自动刷新。
//! 计划：`notify` 做底层事件，`notify-debouncer-mini` 收口（编辑器保存一次会触发多个事件），
//! 事件通过 `app.emit("fs://changed", payload)` 推给前端。
//!
//! P1 不启用，所以这里不做半成品 —— 前端 `watchPath` 会拿到 NOT_SUPPORTED，
//! 而 P1 的冲突检测走的是「保存时比对 mtime」这条不需要监听也能工作的路子。

use crate::error::{AppError, AppResult};

pub fn start_watching(_path: &str) -> AppResult<()> {
    Err(AppError::not_supported(
        "文件监听将在 P3 提供；当前阶段请在保存时依赖 mtime 冲突检测",
    ))
}
