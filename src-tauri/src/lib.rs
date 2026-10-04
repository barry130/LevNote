//! 轻墨 · 桌面端入口。
//!
//! 分层约定：`main.rs` 只负责调用 `run()`，真正的组装都在这里，
//! 这样未来要做集成测试或移动端入口时不必复制一遍 builder 链。

pub mod drawio;
pub mod error;
pub mod export;
pub mod fs;
pub mod search;
pub mod watch;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        // 全文检索的取消位。放在 managed state 里，因为命令是在阻塞线程池上跑的，
        // 而「谁在跑、该不该停」这件事必须有一个进程内的、跨线程的落点。
        .manage(search::SearchState::default())
        .invoke_handler(tauri::generate_handler![
            fs::read_text_file,
            fs::write_text_file,
            fs::read_binary_file,
            fs::write_binary_file,
            fs::stat_file,
            fs::sniff_encoding,
            fs::list_dir,
            fs::runtime_info,
            search::search_in_workspace,
            search::cancel_search,
            watch::watch_replace,
        ])
        .setup(|app| {
            // 文件监听器要在任何 watch_replace 之前就位（它的回调需要 AppHandle
            // 才能把事件发出去）。窗口状态恢复、最近打开记录等放在外壳层（P2）。
            watch::init(app.handle().clone()).map_err(|e| e.to_string())?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("轻墨启动失败：Tauri 运行时初始化异常");
}
