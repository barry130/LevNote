// Windows 下隐藏控制台窗口；release 之外保持可见，方便看 panic 信息。
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    levnote_lib::run()
}
