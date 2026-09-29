//! 全文检索（P4 落地）。
//!
//! 设计要点提前记在这里，免得实现时再纠：
//! - 大目录必须流式 + 可取消，不能用 `Vec<Result>` 一次性收集；
//! - 二进制文件靠「前 8KB 是否含 NUL」快速跳过，不做完整解码；
//! - 尊重 `.gitignore` 与用户的黑名单（`node_modules` / `target` 等）默认排除；
//! - 结果按文件聚合后分块推给前端，UI 边搜边出，不等全量。

use crate::error::{AppError, AppResult};

pub fn search_in_workspace(_root: &str, _query: &str) -> AppResult<()> {
    Err(AppError::not_supported("全文检索将在 P4 提供"))
}
