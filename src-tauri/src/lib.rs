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
        .invoke_handler(tauri::generate_handler![
            fs::read_text_file,
            fs::write_text_file,
            fs::read_binary_file,
            fs::write_binary_file,
            fs::stat_file,
            fs::sniff_encoding,
            fs::list_dir,
            fs::runtime_info,
        ])
        .setup(|_app| {
            // P1 只需要它启动起来；窗口状态恢复、最近打开记录等放在外壳层（P2）
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("轻墨启动失败：Tauri 运行时初始化异常");
}
