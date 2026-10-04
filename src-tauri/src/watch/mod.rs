//! 文件监听（P3 第 14 项，ADR 0029）。
//!
//! ## 形状
//!
//! ```text
//! 前端 ──invoke(watch_replace, { paths })──▶ Rust：把监视集合**整体替换**成 paths
//!      ◀──invoke 返回值：没监视成功的路径（通常为空）────
//! Rust ──emit("fs://changed", { paths })──▶ 前端：防抖窗口内的变更路径批量
//! ```
//!
//! 前端每次「打开文件夹 / 开关标签页」之后把**想要**的监视集合发过来
//! （工作区根目录 + 不在根目录下的打开文件），Rust 侧对当前集合做差量
//! watch / unwatch。**整表替换**而不是 start/stop：前端不用记「哪个还开着」，
//! 重复发送幂等，崩溃重启后发一次全量即收敛。
//!
//! ## 三条定下来的取舍
//!
//! 1. **`notify` 底层事件 + `notify-debouncer-mini` 收口**。保存一次文件会
//!    触发多个事件（写前截断、写内容、元数据……），不收口的话前端会被
//!    事件风暴打醒 N 次。250ms 的防抖窗口与「人感觉不到、又确实合并了
//!    连续写」的距离一致。
//! 2. **工作区根目录递归监视**：目录树要的是「根下任何一层的新增 / 删除 /
//!    改名」；递归交给 notify，前端不用自己维护「展开了哪些层」的监视清单。
//!    根目录之外的单个打开文件用非递归监视补上。
//! 3. **监视失败不致命**。事件窗口里目标文件可能刚被删掉（watch 返回错误）——
//!    把失败的路径作为 `watch_replace` 的**返回值**交给前端记日志，其余路径
//!    照常监视。这不是静默：下一次整表替换会自动重试，而用户此刻真正需要
//!    的信息（文件没了）会由目录树刷新与文档冲突检测给出。
//!
//! ## 与「保存时比对 mtime」的关系
//!
//! 事件是**通知**，mtime 比对是**裁决**：这里只负责把「哪些路径可能变了」
//! 尽快告诉前端，`store.checkExternalChange()` 仍然按 mtime 判定要不要
//! 弹冲突 —— 事件丢了、重了、迟了都不会造成错误的冲突提示，最多慢一拍。
//! 窗口焦点时的全量比对保留作兜底（事件系统永远可能丢）。

use std::{
    collections::{BTreeSet, HashSet},
    path::PathBuf,
    sync::Mutex,
    time::Duration,
};

use notify::RecursiveMode;
use notify_debouncer_mini::{new_debouncer, DebounceEventResult, Debouncer};
use serde::Serialize;
use tauri::{AppHandle, Emitter};

use crate::error::{AppError, AppResult};

/// 防抖窗口。250ms：编辑器保存一次的多个事件必然落进同一批；
/// 用户「保存后立刻看目录树」的等待也不会超过四分之一秒。
const DEBOUNCE_MS: u64 = 250;

/// 推给前端的一批变更路径。`error` 非空表示监听后端自身出错
/// （比如被监视的目录被整个删掉）—— 事件系统出了问题也要说出来。
#[derive(Serialize, Clone, Debug, PartialEq, Eq)]
pub struct FsChangeBatch {
    pub paths: Vec<String>,
    pub error: Option<String>,
}

/// 一个活跃的监视器：防抖器本体 + **当前真正监视着的路径**。
/// 集合只记「成功了」的 —— watch 失败的路径不进来，下一次 diff 会自动重试。
struct WatchState {
    debouncer: Debouncer<notify::RecommendedWatcher>,
    watched: BTreeSet<PathBuf>,
}

static WATCH: Mutex<Option<WatchState>> = Mutex::new(None);

/// 创建防抖器并接管事件转发。在 Tauri 的 `setup` 里调用一次；
/// 重复调用是幂等的（测试与热重载都可能再进来）。
pub fn init(app: AppHandle) -> AppResult<()> {
    let mut guard = WATCH
        .lock()
        .map_err(|_| AppError::internal("文件监听状态被锁住（有线程在持锁时崩溃？）"))?;
    if guard.is_some() {
        return Ok(());
    }

    let debouncer = new_debouncer(Duration::from_millis(DEBOUNCE_MS), move |result: DebounceEventResult| {
        let batch = match result {
            Ok(events) => {
                // HashSet 去重：一次保存的多个原始事件合并成一个路径集合，
                // 前端拿到的内容与事件顺序无关
                let mut paths: HashSet<String> = HashSet::new();
                for event in events {
                    paths.insert(event.path.to_string_lossy().to_string());
                }
                if paths.is_empty() {
                    return;
                }
                FsChangeBatch { paths: paths.into_iter().collect(), error: None }
            }
            Err(thrown) => FsChangeBatch {
                paths: Vec::new(),
                error: Some(thrown.to_string()),
            },
        };
        // 发送失败只可能意味着前端已经不在了（窗口关闭）—— 没什么可做的
        let _ = app.emit("fs://changed", batch);
    })
    .map_err(|thrown| AppError::internal(format!("文件监听器创建失败：{thrown}")))?;

    *guard = Some(WatchState { debouncer, watched: BTreeSet::new() });
    Ok(())
}

/// 把监视集合整体替换成 `paths`。返回**没能监视上**的路径（通常为空）。
///
/// 差量更新：已经在集合里的不动（notify 对重复 watch 会报错），只处理
/// 新增与移除。目录按递归监视、文件按非递归（见模块注释第 2 条）。
pub fn replace(paths: &[String]) -> AppResult<Vec<String>> {
    let desired: BTreeSet<PathBuf> = paths.iter().map(PathBuf::from).collect();

    let mut guard = WATCH
        .lock()
        .map_err(|_| AppError::internal("文件监听状态被锁住（有线程在持锁时崩溃？）"))?;
    let state = guard
        .as_mut()
        .ok_or_else(|| AppError::internal("文件监听尚未初始化（setup 没跑？）"))?;

    let (add, remove) = plan_diff(&state.watched, &desired);

    let mut failed: Vec<String> = Vec::new();
    for path in remove {
        // unwatch 失败不挡路：路径多半已经不存在，从集合里摘掉即可
        let _ = state.debouncer.watcher().unwatch(&path);
        state.watched.remove(&path);
    }
    for path in add {
        let mode = if path.is_dir() { RecursiveMode::Recursive } else { RecursiveMode::NonRecursive };
        match state.debouncer.watcher().watch(&path, mode) {
            Ok(()) => {
                state.watched.insert(path);
            }
            Err(_) => {
                // 事件窗口里的竞态（文件刚被删 / 权限变化）不该让整批失败；
                // 下一次 replace 会自动重试这条路
                failed.push(path.to_string_lossy().to_string());
            }
        }
    }
    Ok(failed)
}

/// Tauri 命令入口。见模块注释的形状图。
#[tauri::command]
pub fn watch_replace(paths: Vec<String>) -> AppResult<Vec<String>> {
    replace(&paths)
}

/// 纯函数：当前集合与期望集合的差量（要加的、要摘的）。
fn plan_diff(current: &BTreeSet<PathBuf>, desired: &BTreeSet<PathBuf>) -> (Vec<PathBuf>, Vec<PathBuf>) {
    let add = desired.difference(current).cloned().collect();
    let remove = current.difference(desired).cloned().collect();
    (add, remove)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn set(items: &[&str]) -> BTreeSet<PathBuf> {
        items.iter().map(|p| PathBuf::from(p)).collect()
    }

    #[test]
    fn 空到空_无事发生() {
        let (add, remove) = plan_diff(&set(&[]), &set(&[]));
        assert!(add.is_empty() && remove.is_empty());
    }

    #[test]
    fn 全新增与全移除() {
        let (add, remove) = plan_diff(&set(&[]), &set(&["C:\\w"]));

        assert_eq!(add, vec![PathBuf::from("C:\\w")]);
        assert!(remove.is_empty());

        let (add, remove) = plan_diff(&set(&["C:\\w"]), &set(&[]));

        assert!(add.is_empty());
        assert_eq!(remove, vec![PathBuf::from("C:\\w")]);
    }

    #[test]
    fn 差量只包含变化的路径() {
        // 已监视的保持不动 —— 对 notify 重复 watch 会报错，
        // 所以「不动」本身就是这条逻辑存在的理由
        let (add, remove) = plan_diff(&set(&["C:\\w", "C:\\a.md"]), &set(&["C:\\w", "C:\\b.md"]));

        assert_eq!(add, vec![PathBuf::from("C:\\b.md")]);
        assert_eq!(remove, vec![PathBuf::from("C:\\a.md")]);
    }

    #[test]
    fn 同一路径重复出现只算一次() {
        let desired = set(&["C:\\w", "C:\\w", "C:\\w"]);
        let (add, _) = plan_diff(&set(&[]), &desired);

        assert_eq!(add.len(), 1);
    }

    #[test]
    fn 批次载荷能序列化成前端要的形状() {
        let batch = FsChangeBatch { paths: vec!["C:\\w\\a.md".into()], error: None };
        let json = serde_json::to_value(&batch).unwrap();
        assert_eq!(json["paths"], serde_json::json!(["C:\\w\\a.md"]));
        assert_eq!(json["error"], serde_json::Value::Null);

        let failed = FsChangeBatch { paths: vec![], error: Some("gone".into()) };
        let json = serde_json::to_value(&failed).unwrap();
        assert_eq!(json["error"], serde_json::json!("gone"));
    }
}
