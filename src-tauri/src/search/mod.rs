//! 全文检索（P4 第 23 项）。
//!
//! ## 形状
//!
//! ```text
//! 前端 ──invoke(search_in_workspace, { request, onEvent })──▶ Rust
//!      ◀──onEvent: Channel<SearchBatch>（× N）────────────────
//!      ◀──invoke 的返回值：SearchSummary（这一次的结局）──────
//!      ──invoke(cancel_search, { searchId })──▶ 置位，遍历下一轮就停
//! ```
//!
//! ## 四条定下来的取舍
//!
//! 1. **流式**。结果不攒到最后一次性返回：一个大目录扫十几秒，中途什么都看不到
//!    是最难受的形态。批次按「攒够 12 个有命中的文件」或「过了 150ms」推，
//!    后者保证**一个命中都没有时进度也在动**。
//! 2. **可取消**。取消不是「把结果丢掉」而是**真的停下遍历** ——
//!    否则用户改一个字就要多扫一遍全目录，机器风扇替他记账。
//!    实现在 `SearchState`：一个 id 一个 `AtomicBool`，遍历每处理一个文件看一眼。
//! 3. **尊重 `.gitignore`**。用 `ignore`（ripgrep 的库）而不是自己实现，
//!    理由见 `Cargo.toml` 与 `walk.rs` 的模块注释。面板上有一个
//!    「包含被忽略的文件」的开关 —— 那是这条规则的**逃生口**，
//!    少了它，「结果里没有我的文件」就变成一件无法自救的事。
//! 4. **错误分两个通道**。**开工前的错误**（没有文件夹、检索词为空、正则写错）
//!    与**中途失败**都走命令的 `Err`；**结局**（正常结束 / 被取消 / 达到上限）
//!    走命令的返回值。通道只负责**批次**这一件事。
//!
//!    一开始的设计是把结局也做成一个 `Done` 事件从通道推，于是前端有两条
//!    互相竞速的终局路径（`invoke` 的 promise 与通道的事件），得靠「谁先到」去猜。
//!    现在只有一条：`await invoke(...)` 拿到的就是结局。
//!    代价是**通道上可能还有一批在路上**（Rust 发完最后一批就返回了），
//!    所以前端在收到结局之后**不许让进度回退** —— 那两行在
//!    `core/search/workspace.ts` 里，配了测试。
//!
//! ## 列号为什么是 UTF-16 码元
//!
//! 命中要能跳过去，跳转的落点是「第几行第几列」。这个「列」必须是一个
//! **两侧都精确**的单位，而两侧用的是不同的坐标系：
//!
//! - Rust 侧天然是字节偏移（`regex` 给的就是字节）；
//! - JS 侧天然是 UTF-16 码元（`String` 下标、`CodeMirror` 的 `ch`、
//!   `textarea.selectionStart` 全是它）。
//!
//! 选 UTF-16 码元，是因为**转换只需要在 Rust 侧做一次**，而且做得准
//! （`char::len_utf16`）；反过来在 JS 侧做就要逐字符数一遍。
//! 用「字符数」这种看着更中性的单位反而最糟：中文与 emoji 的 `char` 计数
//! 与 JS 下标都对不上，而**差一位**的表现是「跳到命中右边一格」，
//! 没人会怀疑是单位问题。

pub mod matcher;
pub mod walk;

use std::collections::{HashMap, VecDeque};
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, MutexGuard};

use serde::{Deserialize, Serialize};
use tauri::ipc::Channel;
use tauri::{AppHandle, Manager, State};

use crate::error::{AppError, AppResult};
use matcher::{LineHit, Matcher};
use walk::SearchOptions;

/// 命中上限的默认值。
///
/// 2000 的取法：面板是给人看的，不是给机器算的 —— 超过两千条没人会一条条翻。
/// 到了上限会**如实说**「已达上限，结果可能不全」，而不是安静地截断。
pub const DEFAULT_MAX_RESULTS: usize = 2000;

/// 单文件字节上限的默认值。
///
/// 4 MiB：比它大的多半是日志、导出数据、被压缩过的东西。跳过它们让一次检索
/// 的耗时可预期，而跳过的数量会出现在收尾统计里。
pub const DEFAULT_MAX_FILE_BYTES: u64 = 4 * 1024 * 1024;

/// 「取消得比注册还早」时先记下来的 id 数量上限。
///
/// 前端在 `invoke` 返回之前就点了停止，那一瞬间 Rust 侧还没有这个 id。
/// 直接丢弃这次取消 = 用户点了停止、遍历却继续跑到底。所以先记下来，
/// 等 `register` 时补上。用环形队列封顶，避免一条永远不来的检索
/// 在表里留下一个删不掉的名字。
const EARLY_CANCEL_CAPACITY: usize = 64;

/// 一次检索的请求。`searchId` 由**前端**生成 —— 它必须在 `invoke` 之前就知道
/// 自己叫这个名字，否则「还没拿到 id 就想取消」是一个无解的状态。
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchRequest {
    pub search_id: u64,
    pub root: String,
    pub query: String,
    #[serde(default)]
    pub case_sensitive: bool,
    #[serde(default)]
    pub whole_word: bool,
    #[serde(default)]
    pub regex: bool,
    #[serde(default)]
    pub include_ignored: bool,
    pub max_results: Option<usize>,
    pub max_file_bytes: Option<u64>,
}

/// 一个文件里的全部命中。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct FileMatches {
    pub path: String,
    pub matches: Vec<LineHit>,
}

/// 一批结果 + 截至此刻的进度。
///
/// 进度与结果**打在一起**：分成两个事件的话，前端要维护两份状态，
/// 而它们之间的一致性（「这个进度对应的是哪一批结果」）没人保证。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchBatch {
    pub files: Vec<FileMatches>,
    pub matched_files: u64,
    pub total_matches: u64,
    pub scanned_files: u64,
    pub scanned_bytes: u64,
    pub skipped_binary: u64,
    pub skipped_large: u64,
    pub skipped_unreadable: u64,
}

/// 检索是怎么结束的。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum SearchReason {
    /// 走完了整个目录树。
    Completed,
    /// 用户点了停止（或接收端断开）。
    Cancelled,
    /// 到了命中上限。**结果可能不全**，面板必须说出来。
    Truncated,
}

/// 收尾统计。它同时是 `search_in_workspace` 的返回值。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchSummary {
    pub reason: SearchReason,
    pub matched_files: u64,
    pub total_matches: u64,
    pub scanned_files: u64,
    pub scanned_bytes: u64,
    /// 前 8KB 含 NUL，当成二进制跳过。
    pub skipped_binary: u64,
    /// 超过单文件上限。
    pub skipped_large: u64,
    /// 读不出来（权限、断链、解码失败）。
    pub skipped_unreadable: u64,
    pub elapsed_ms: u64,
}

/// 在跑的检索的取消位。
#[derive(Default)]
pub struct SearchState {
    flags: Mutex<HashMap<u64, Arc<AtomicBool>>>,
    early_cancel: Mutex<VecDeque<u64>>,
}

fn lock<T>(mutex: &Mutex<T>) -> MutexGuard<'_, T> {
    // 这些锁里没有会 panic 的代码，中毒只可能来自别的线程的 panic。
    // 即便如此也不该把整个检索链路带崩 —— 拿到里面的数据继续用。
    mutex
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

impl SearchState {
    fn register(&self, id: u64) -> Arc<AtomicBool> {
        let early = {
            let mut early = lock(&self.early_cancel);
            match early.iter().position(|&item| item == id) {
                Some(position) => {
                    early.remove(position);
                    true
                }
                None => false,
            }
        };
        let flag = Arc::new(AtomicBool::new(early));
        lock(&self.flags).insert(id, Arc::clone(&flag));
        flag
    }

    fn unregister(&self, id: u64) {
        lock(&self.flags).remove(&id);
    }

    /// 返回「这个 id 现在有没有在跑的检索」。
    fn cancel(&self, id: u64) -> bool {
        let found = {
            let flags = lock(&self.flags);
            match flags.get(&id) {
                Some(flag) => {
                    flag.store(true, Ordering::Relaxed);
                    true
                }
                None => false,
            }
        };
        if found {
            return true;
        }

        // 还没注册：先记下来，等 `register` 时补上（见 `EARLY_CANCEL_CAPACITY`）
        let mut early = lock(&self.early_cancel);
        if !early.contains(&id) {
            early.push_back(id);
        }
        while early.len() > EARLY_CANCEL_CAPACITY {
            early.pop_front();
        }
        false
    }
}

fn validate_root(raw: &str) -> AppResult<PathBuf> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return Err(AppError::invalid_path("还没有打开文件夹，检索无从开始"));
    }
    let path = PathBuf::from(trimmed);
    if !path.is_dir() {
        return Err(AppError::invalid_path(format!(
            "不是文件夹：{}",
            path.display()
        )));
    }
    Ok(path)
}

/// 在一个文件夹里检索。
///
/// `on_event` 上推的是**批次**；**结局**是这个函数的返回值。
/// 分成两条路的理由见模块注释第 4 条。
#[tauri::command]
pub async fn search_in_workspace(
    app: AppHandle,
    request: SearchRequest,
    on_event: Channel<SearchBatch>,
) -> AppResult<SearchSummary> {
    let root = validate_root(&request.root)?;
    let matcher = Matcher::build(
        &request.query,
        request.case_sensitive,
        request.whole_word,
        request.regex,
    )?;

    let options = SearchOptions {
        include_ignored: request.include_ignored,
        max_results: request.max_results.unwrap_or(DEFAULT_MAX_RESULTS).max(1),
        max_file_bytes: request
            .max_file_bytes
            .unwrap_or(DEFAULT_MAX_FILE_BYTES)
            .max(1),
    };

    let search_id = request.search_id;
    let cancel = app.state::<SearchState>().register(search_id);

    // 遍历是阻塞的、且可能跑很久 —— 放进阻塞线程池，不要占住 async 运行时的工作线程
    let worker = tauri::async_runtime::spawn_blocking(move || {
        let mut emit = |batch: SearchBatch| on_event.send(batch).is_ok();
        walk::run(&root, &matcher, options, &cancel, &mut emit)
    });

    let outcome = worker.await;
    app.state::<SearchState>().unregister(search_id);

    outcome.map_err(|error| AppError::internal(format!("检索线程异常结束：{error}")))
}

/// 取消一次检索。返回「这个 id 现在有没有在跑的检索」。
///
/// 返回 `false` 不代表失败：它多半是「取消得比注册还早」，那一次取消已经被记下来，
/// 会在检索真正开始时生效（见 `EARLY_CANCEL_CAPACITY`）。
#[tauri::command]
pub fn cancel_search(state: State<'_, SearchState>, search_id: u64) -> bool {
    state.cancel(search_id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cancel_before_register_is_remembered() {
        let state = SearchState::default();
        assert!(!state.cancel(7));
        // 注册时应当拿到一个**已经置位**的取消标志
        let flag = state.register(7);
        assert!(flag.load(Ordering::Relaxed));
        state.unregister(7);
    }

    #[test]
    fn cancel_after_register_sets_the_flag() {
        let state = SearchState::default();
        let flag = state.register(1);
        assert!(!flag.load(Ordering::Relaxed));
        assert!(state.cancel(1));
        assert!(flag.load(Ordering::Relaxed));
        state.unregister(1);
    }

    #[test]
    fn early_cancel_list_is_bounded() {
        let state = SearchState::default();
        for id in 0..(EARLY_CANCEL_CAPACITY as u64 * 3) {
            state.cancel(id);
        }
        // 最早的那几个已经被挤出去了：它们再来注册时不会被误取消
        assert!(!state.register(0).load(Ordering::Relaxed));
        // 最近的一个还在
        let last = EARLY_CANCEL_CAPACITY as u64 * 3 - 1;
        assert!(state.register(last).load(Ordering::Relaxed));
    }

    #[test]
    fn unregister_forgets_the_flag() {
        let state = SearchState::default();
        let flag = state.register(3);
        state.unregister(3);
        assert!(!state.cancel(3));
        assert!(!flag.load(Ordering::Relaxed));
    }

    #[test]
    fn validate_root_rejects_empty_and_files() {
        assert!(validate_root("  ").is_err());
        assert!(validate_root("Cargo.toml").is_err());
    }
}
