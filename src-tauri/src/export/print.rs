//! 打印 / 导出 PDF（P3 第 16 项，ADR 0030）。
//!
//! ## 形状
//!
//! ```text
//! 前端 ──invoke(print_html, { html, title })──▶ Rust：开一个子窗口打印这份 HTML
//! 子窗口 ──levnote-print://localhost/<id>──▶ 协议处理器：从内存表里取出 HTML 回给 WebView
//! 子窗口 load ──window.print()──▶ WebView2 / WKWebView 的打印对话框（另存为 PDF 在里面选）
//! ```
//!
//! 为什么走**打印对话框**而不是程序直接产出 PDF 字节：wry 0.57 没有把
//! WebView2 的 `ICoreWebView2_16::PrintToPdf` 暴露出来，自己 unsafe 地够
//! COM 接口意味着绑定 wry 内部结构、且 macOS / Linux 各要另写一条路。
//! 打印对话框是三个平台都有的「打印管线」，用户在里面顺便还能选真打印机 ——
//! 对一份桌面编辑器来说这是同一条能力的两个出口，不是降级。
//!
//! ## 为什么用自定义协议、不落临时文件
//!
//! 自包含 HTML 内联了 base64 图片，几 MB 很平常：`data:` URL 的导航有
//! ~2MB 的上限，临时文件则要管清理与权限。注册一个 `levnote-print` 协议、
//! 把 HTML 存在**内存表**里按 id 发出去，没有大小限制，窗口关闭时摘掉
//! 表项即可（见 `on_window_event` 的 Destroyed 分支）。
//!
//! ## 与自包含 HTML 导出的关系
//!
//! 两者共用前端的 `buildSelfContainedHtml`（同一段渲染结果，连 `@media
//! print` 的纸张样式都是同一份）；Rust 侧只负责「把这份 HTML 交给打印管线」。

use std::collections::BTreeMap;
use std::sync::{
    Mutex,
    atomic::{AtomicU64, Ordering},
};

use tauri::{AppHandle, Url, WebviewUrl, WebviewWindowBuilder, WindowEvent, http};

use crate::error::{AppError, AppResult};

/// 打印窗口里的引导条 + 自动弹打印对话框。
///
/// 引导条解决两件事：① 打印对话框万一没弹出来（老版本 WebView2 不支持
/// `window.print()`），页面上有一句兜底说法（硬约束 2：不静默）；
/// ② 告诉用户 PDF 的入口在哪 —— 打印对话框里的「另存为 PDF」。
/// 引导条本体在 `@media print` 里隐藏，不会印到纸上。
const PRINT_BOOTSTRAP: &str = r#"(function () {
  function setup() {
    var bar = document.createElement('div');
    bar.id = 'qm-print-hint';
    bar.setAttribute('style', 'position:fixed;top:0;left:0;right:0;z-index:2147483647;box-sizing:border-box;padding:10px 16px;background:#1f6f54;color:#fff;font:13px/1.6 system-ui,sans-serif;text-align:center;');
    bar.textContent = '打印预览 —— 在打印对话框里选「另存为 PDF」即可导出；对话框没自动出现就按 Ctrl+P。看完关掉这个窗口即可。';
    document.body.appendChild(bar);
    var style = document.createElement('style');
    style.textContent = '@media print { #qm-print-hint { display: none !important; } }';
    document.head.appendChild(style);
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', setup);
  else setup();
  window.addEventListener('load', function () {
    setTimeout(function () { try { window.print(); } catch (thrown) { /* 兜底说明就在页面顶上 */ } }, 150);
  });
})();"#;

/// 打印窗口的 HTML，按会话 id 存放。窗口销毁时摘除。
///
/// BTreeMap 是因为 `new()` 是 const 函数，静态表可以直接初始化。
static PRINT_PAGES: Mutex<BTreeMap<String, String>> = Mutex::new(BTreeMap::new());
static NEXT_ID: AtomicU64 = AtomicU64::new(0);

fn lock_pages() -> Result<std::sync::MutexGuard<'static, BTreeMap<String, String>>, AppError> {
    PRINT_PAGES
        .lock()
        .map_err(|_| AppError::internal("打印会话表被锁住（有线程在持锁时崩溃？）"))
}

/// 协议处理器：按 URL 里的 id 取回 HTML。注册在 `run()` 里，全局生效。
///
/// Windows 上实际请求形如 `http://levnote-print.localhost/<id>.html`、
/// macOS / Linux 是 `levnote-print://localhost/<id>.html` —— **path 两边
/// 长一个样**，所以只看 path，不猜 scheme。
pub fn serve(request: &http::Request<Vec<u8>>) -> http::Response<Vec<u8>> {
    let raw_path = request.uri().path();
    let id = raw_path.trim_start_matches('/').trim_end_matches(".html");

    // 读路径容忍毒锁：表坏了不该让打印窗口变成一片白屏
    let pages = match PRINT_PAGES.lock() {
        Ok(guard) => guard,
        Err(poisoned) => poisoned.into_inner(),
    };

    match pages.get(id) {
        Some(html) => html_response(200, html),
        None => {
            // 404 也要说清楚发生了什么 —— 不静默
            html_response(
                404,
                "<!DOCTYPE html><meta charset=\"utf-8\"><title>打印会话失效</title>\
<body style=\"font:14px/1.7 system-ui,sans-serif;padding:48px;color:#16302a;\">\
这个打印会话已经失效（窗口可能已经关过一次）。回到轻笺重新执行「打印 / 导出为 PDF」即可。\
</body>",
            )
        }
    }
}

/// 固定头部与状态的 Response 构造不会失败（没有可变输入），expect 是安全的。
fn html_response(status: u16, page: &str) -> http::Response<Vec<u8>> {
    http::Response::builder()
        .status(status)
        .header("Content-Type", "text/html; charset=utf-8")
        .body(page.as_bytes().to_vec())
        .expect("固定头部与状态的 Response 构造不会失败")
}

/// 开一个打印窗口承载 `html`。窗口销毁时自动清掉内存表里的那份 HTML。
///
/// `title` 只用在窗口标题上，页面内容以 `html` 为准。
pub fn open_print_window(app: &AppHandle, html: String, title: &str) -> AppResult<()> {
    let id = format!("p{}", NEXT_ID.fetch_add(1, Ordering::Relaxed));
    let url = Url::parse(&format!("levnote-print://localhost/{id}.html"))
        .map_err(|thrown| AppError::internal(format!("打印会话地址构造失败：{thrown}")))?;

    lock_pages()?.insert(id.clone(), html);

    let window_label = format!("print-{id}");
    let window = WebviewWindowBuilder::new(app, &window_label, WebviewUrl::CustomProtocol(url))
        .title(format!("{title} — 打印 / 另存 PDF"))
        .inner_size(880.0, 1040.0)
        .initialization_script(PRINT_BOOTSTRAP)
        .build();

    match window {
        Ok(window) => {
            let cleanup_id = id;
            window.on_window_event(move |event| {
                if matches!(event, WindowEvent::Destroyed) {
                    // 毒锁时表项会泄漏，但毒锁本身就意味着有线程崩了，不值得在这里抢救
                    if let Ok(mut pages) = PRINT_PAGES.lock() {
                        pages.remove(&cleanup_id);
                    }
                }
            });
            Ok(())
        }
        Err(thrown) => {
            // 窗口没建起来，存的 HTML 也没人要了 —— 摘掉再报错
            if let Ok(mut pages) = PRINT_PAGES.lock() {
                pages.remove(&id);
            }
            Err(AppError::internal(format!("打不开打印窗口：{thrown}")))
        }
    }
}

/// Tauri 命令入口。
///
/// **必须是 async**：Tauri 的同步命令在**主线程**上内联执行，而
/// `WebviewWindowBuilder::build()` 要等 WebView2「环境已创建」的回调 ——
/// 那个回调得靠消息循环派发，消息循环此刻正被本命令占着 → 死锁。
/// 实测表现就是本 bug：窗口框架带标题立起来了、内容一片白、
/// 整个应用的事件循环死掉（子窗口连 X 都关不掉）。
/// async 命令跑在异步运行时线程上，窗口创建作为一条事件循环消息派发，
/// 不存在嵌套等待。
#[tauri::command]
pub async fn print_html(app: AppHandle, html: String, title: String) -> AppResult<()> {
    let title = if title.trim().is_empty() { "打印预览" } else { title.as_str() };
    let owned = title.to_string();
    open_print_window(&app, html, &owned)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn request_for(path: &str) -> http::Request<Vec<u8>> {
        http::Request::builder()
            .uri(format!("levnote-print://localhost/{path}"))
            .body(Vec::new())
            .unwrap()
    }

    #[test]
    fn 存进去的会话能按路径取回() {
        let id = "t_exist".to_string();
        lock_pages().unwrap().insert(id.clone(), "<html>正文</html>".to_string());

        let response = serve(&request_for("t_exist.html"));
        assert_eq!(response.status(), 200);
        // 字节串字面量不许非 ASCII，用 as_bytes 比对
        assert_eq!(response.body(), "<html>正文</html>".as_bytes());
        let content_type = response.headers().get("Content-Type").unwrap().to_str().unwrap();
        assert!(content_type.starts_with("text/html"));

        lock_pages().unwrap().remove(&id);
    }

    #[test]
    fn 不带的会话返回说清楚的_404() {
        let response = serve(&request_for("t_missing.html"));
        assert_eq!(response.status(), 404);
        let body = String::from_utf8(response.body().clone()).unwrap();
        assert!(body.contains("失效"), "404 页面要说明发生了什么：{body}");
    }

    #[test]
    fn 没带_id_的请求不炸() {
        let response = serve(&request_for(""));
        assert_eq!(response.status(), 404);
    }

    #[test]
    fn 会话表_存了又删就查不到() {
        let id = "t_lifecycle".to_string();
        lock_pages().unwrap().insert(id.clone(), "x".to_string());
        assert!(lock_pages().unwrap().contains_key(&id));
        lock_pages().unwrap().remove(&id);
        assert!(!lock_pages().unwrap().contains_key(&id));
    }
}
