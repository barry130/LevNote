//! 文件内核。这是「文档抽象层」在桌面端的唯一平台相关实现。
//!
//! 所有命令的返回结构都是 camelCase（`serde(rename_all = "camelCase")`），
//! 与 TypeScript 侧的 `src/core/document/types.ts` 一一对应，改一侧必须改另一侧。

pub mod atomic;
pub mod codec;

use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};

use crate::error::{AppError, AppResult, ErrorCode};
use codec::Bom;

/// 二进制探测用的采样长度：读头部 8KB 足以判断编码。
const SNIFF_BYTES: usize = 8 * 1024;

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct FileStat {
    pub path: String,
    pub size: u64,
    pub mtime_ms: u64,
    pub readonly: bool,
    pub is_dir: bool,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TextFile {
    pub path: String,
    pub text: String,
    /// 规范化编码名，如 `UTF-8` / `GBK`
    pub encoding: String,
    /// `none` | `utf-8` | `utf-16le` | `utf-16be`
    pub bom: String,
    /// `lf` | `crlf` | `cr` | `mixed`
    pub line_ending: String,
    pub size: u64,
    pub mtime_ms: u64,
    pub readonly: bool,
    /// 解码时出现过非法字节，探测结果可能不可靠
    pub lossy: bool,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct WriteOutcome {
    pub path: String,
    pub size: u64,
    pub mtime_ms: u64,
    pub encoding: String,
    pub bom: String,
    pub line_ending: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WriteRequest {
    pub path: String,
    pub text: String,
    /// 不传则按 UTF-8 写
    pub encoding: Option<String>,
    /// 不传则沿用原有 BOM 判定（新建文件无 BOM）
    pub bom: Option<String>,
    /// 不传则不做行尾规范化（原样写）
    pub line_ending: Option<String>,
    /// 传入则做乐观并发检测：磁盘 mtime 与它不一致即拒绝写入
    pub expected_mtime_ms: Option<u64>,
    /// 为 false 且文件不存在时报 NOT_FOUND；为 true 或未传则允许新建
    pub create: Option<bool>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DirEntry {
    pub name: String,
    pub path: String,
    pub is_dir: bool,
    pub size: u64,
    pub mtime_ms: u64,
}

// ---------------------------------------------------------------- 工具函数

fn mtime_ms_of(meta: &fs::Metadata) -> u64 {
    meta.modified()
        .ok()
        .and_then(|t| t.duration_since(UNIX_EPOCH).ok())
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

fn parse_bom(label: Option<&str>) -> AppResult<Option<Bom>> {
    match label {
        None => Ok(None),
        Some("none") | Some("") => Ok(Some(Bom::None)),
        Some("utf-8") => Ok(Some(Bom::Utf8)),
        Some("utf-16le") => Ok(Some(Bom::Utf16Le)),
        Some("utf-16be") => Ok(Some(Bom::Utf16Be)),
        Some(other) => Err(AppError::new(
            ErrorCode::UnknownEncoding,
            format!("不认识的 BOM 类型：{other}"),
        )),
    }
}

fn normalize_path(raw: &str) -> AppResult<PathBuf> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return Err(AppError::invalid_path("路径为空"));
    }
    Ok(PathBuf::from(trimmed))
}

fn guard_not_dir(path: &Path) -> AppResult<()> {
    if path.is_dir() {
        return Err(AppError::new(
            ErrorCode::IsDirectory,
            format!("这是一个目录，不是文件：{}", path.display()),
        ));
    }
    Ok(())
}

fn stat_of(path: &Path) -> AppResult<FileStat> {
    let meta = fs::metadata(path).map_err(|e| {
        if e.kind() == std::io::ErrorKind::NotFound {
            AppError::new(ErrorCode::NotFound, format!("文件不存在：{}", path.display()))
        } else {
            AppError::from(e)
        }
    })?;
    Ok(FileStat {
        path: path.to_string_lossy().to_string(),
        size: meta.len(),
        mtime_ms: mtime_ms_of(&meta),
        readonly: meta.permissions().readonly(),
        is_dir: meta.is_dir(),
    })
}

// ---------------------------------------------------------------- 命令

/// 读取文本文件。编码自动探测（可用 `forced_encoding` 覆盖），BOM 与行尾原样保留。
///
/// mtime 必须取自**读完之后**的元数据：它是后续保存时做乐观并发检测的基准，
/// 若取自读取之前，则「读完到保存之间被外部改写」与「读取过程中被改写」都会误判。
/// 若前后元数据不一致（读取期间文件被改动），重试以拿到一份自洽的快照。
#[tauri::command]
pub fn read_text_file(path: String, forced_encoding: Option<String>) -> AppResult<TextFile> {
    let p = normalize_path(&path)?;
    guard_not_dir(&p)?;

    let mut last_error: Option<AppError> = None;

    for _ in 0..3 {
        let before = fs::metadata(&p).map_err(|e| {
            if e.kind() == std::io::ErrorKind::NotFound {
                AppError::new(ErrorCode::NotFound, format!("文件不存在：{}", p.display()))
            } else {
                AppError::from(e)
            }
        })?;

        let bytes = fs::read(&p)?;
        let after = fs::metadata(&p)?;

        if before.len() != after.len() || mtime_ms_of(&before) != mtime_ms_of(&after) {
            // 读取期间被改动，这份快照可能撕裂，重来
            last_error = Some(AppError::new(
                ErrorCode::Conflict,
                "读取期间文件正在被其他程序改写，未取到自洽快照",
            ));
            continue;
        }

        let decoded = codec::decode(&bytes, forced_encoding.as_deref())?;
        return Ok(TextFile {
            path: p.to_string_lossy().to_string(),
            text: decoded.text,
            encoding: decoded.encoding,
            bom: decoded.bom.as_str().to_string(),
            line_ending: decoded.line_ending,
            size: bytes.len() as u64,
            mtime_ms: mtime_ms_of(&after),
            readonly: after.permissions().readonly(),
            lossy: decoded.lossy,
        });
    }

    Err(last_error.unwrap_or_else(|| {
        AppError::new(ErrorCode::Conflict, "文件持续被外部改写，暂时无法读取")
    }))
}

/// 只读文件头部做编码探测，用于「不打开文件也能提示编码」的场景。
#[tauri::command]
pub fn sniff_encoding(path: String) -> AppResult<String> {
    let p = normalize_path(&path)?;
    guard_not_dir(&p)?;
    let bytes = fs::read(&p)?;
    let head = &bytes[..bytes.len().min(SNIFF_BYTES)];
    Ok(codec::decode(head, None)?.encoding)
}

/// 写入文本文件。原子替换 + 可选乐观并发检测。
#[tauri::command]
pub fn write_text_file(req: WriteRequest) -> AppResult<WriteOutcome> {
    let p = normalize_path(&req.path)?;
    guard_not_dir(&p)?;

    let exists = p.exists();
    if !exists && req.create == Some(false) {
        return Err(AppError::new(
            ErrorCode::NotFound,
            format!("文件不存在，且本次不允许新建：{}", p.display()),
        ));
    }

    // 只读检查放在最前面：让用户看到「文件是只读的」，
    // 而不是系统那句难懂的 "os error 5 / Access is denied"。
    let existing: Option<Vec<u8>> = if exists {
        let meta = fs::metadata(&p)?;
        if meta.permissions().readonly() {
            return Err(AppError::new(
                ErrorCode::PermissionDenied,
                format!("文件是只读的，无法写入：{}", p.display()),
            ));
        }
        Some(fs::read(&p)?)
    } else {
        None
    };

    // 乐观并发检测：外部改动过就拒绝，让用户先选「重新加载 / 覆盖 / 另存」
    if let (Some(expected), Some(_)) = (req.expected_mtime_ms, existing.as_ref()) {
        let current = stat_of(&p)?.mtime_ms;
        if current != expected {
            return Err(AppError::new(
                ErrorCode::Conflict,
                "文件已被外部修改，为避免覆盖他人的改动，本次保存已取消",
            )
            .with_detail(format!("expected mtime {expected}, actual mtime {current}")));
        }
    }

    let text = match req.line_ending.as_deref() {
        Some(ending) => codec::normalize_line_endings(&req.text, ending),
        None => req.text,
    };

    // BOM 与编码的隐式继承：调用方没指定时，沿用磁盘上原文件的属性。
    // 这里复用上面已经读到的字节，避免为了探测再读一遍（也避免两次读之间被改写）。
    let paper = existing.as_deref();

    let bom = match parse_bom(req.bom.as_deref())? {
        Some(b) => b,
        None => paper.map(|b| codec::strip_bom(b).0).unwrap_or(Bom::None),
    };

    let encoding = match req.encoding.clone() {
        Some(label) => label,
        None => match paper {
            Some(bytes) => codec::decode(bytes, None)?.encoding,
            None => "UTF-8".to_string(),
        },
    };

    let bytes = codec::encode(&text, &encoding, bom)?;
    atomic::write_atomic(&p, &bytes)?;

    let meta = fs::metadata(&p)?;
    Ok(WriteOutcome {
        path: p.to_string_lossy().to_string(),
        size: meta.len(),
        mtime_ms: mtime_ms_of(&meta),
        encoding,
        bom: bom.as_str().to_string(),
        line_ending: codec::detect_line_ending(&text).to_string(),
    })
}

#[tauri::command]
pub fn stat_file(path: String) -> AppResult<FileStat> {
    let p = normalize_path(&path)?;
    stat_of(&p)
}

/// 图片内联进自包含 HTML 时允许的最大体积。
///
/// 24 MB 的原始字节内联成 base64 约 32 MB，已经是「双击打开会卡一下」的量级。
/// 再大就不该内联了 —— 宁可让导出结果里留一个指向原图的相对链接并标注出来，
/// 也好过生成一个几百 MB、打不开的 HTML 文件。
const MAX_INLINE_BYTES: u64 = 24 * 1024 * 1024;

/// 读二进制文件，原样返回字节。
///
/// 为什么不是 base64：Tauri 2 的 `ipc::Response` 支持直接传 `Vec<u8>`，
/// 前端收到的是 `ArrayBuffer`，比在 Rust 里 base64 编码再在 JS 里解码少两次全量拷贝，
/// 也省掉一个 base64 crate 依赖。转换在前端一次性完成（见 `core/export/inline-html.ts`）。
///
/// **不做任何编码处理**：这是二进制通道，图片必须逐字节原样出来，
/// 走 `read_text_file` 会被转码毁掉。
#[tauri::command]
pub fn read_binary_file(path: String) -> AppResult<tauri::ipc::Response> {
    let p = normalize_path(&path)?;
    guard_not_dir(&p)?;

    let meta = fs::metadata(&p)?;
    if meta.len() > MAX_INLINE_BYTES {
        return Err(AppError::not_supported(format!(
            "文件超过 {} MB，不适合内联进单文件 HTML",
            MAX_INLINE_BYTES / 1024 / 1024
        )));
    }

    let bytes = fs::read(&p)?;
    Ok(tauri::ipc::Response::new(bytes))
}

/// 列出目录，目录优先、同类按名称排序。隐藏文件一并返回，由 UI 决定是否过滤。
#[tauri::command]
pub fn list_dir(path: String) -> AppResult<Vec<DirEntry>> {
    let p = normalize_path(&path)?;
    if !p.is_dir() {
        return Err(AppError::invalid_path(format!("不是目录：{}", p.display())));
    }

    let mut out = Vec::new();
    for entry in fs::read_dir(&p)? {
        let entry = entry?;
        let meta = match entry.metadata() {
            Ok(m) => m,
            // 单个条目读不到元数据（权限/断链）不该让整个目录列表失败
            Err(_) => continue,
        };
        out.push(DirEntry {
            name: entry.file_name().to_string_lossy().to_string(),
            path: entry.path().to_string_lossy().to_string(),
            is_dir: meta.is_dir(),
            size: meta.len(),
            mtime_ms: mtime_ms_of(&meta),
        });
    }

    out.sort_by(|a, b| {
        b.is_dir
            .cmp(&a.is_dir)
            .then_with(|| a.name.to_lowercase().cmp(&b.name.to_lowercase()))
    });
    Ok(out)
}

/// 程序启动时的自检信息，便于排查「装的是哪个版本、跑在哪个系统」。
#[tauri::command]
pub fn runtime_info() -> serde_json::Value {
    serde_json::json!({
        "name": env!("CARGO_PKG_NAME"),
        "version": env!("CARGO_PKG_VERSION"),
        "os": std::env::consts::OS,
        "arch": std::env::consts::ARCH,
        "nowMs": SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_millis() as u64)
            .unwrap_or(0),
    })
}
