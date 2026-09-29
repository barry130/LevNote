use std::fmt;

use serde::Serialize;

/// 结构化错误。前端可以只 switch `code` 做分支，不必解析文案。
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AppError {
    pub code: ErrorCode,
    pub message: String,
    pub detail: Option<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ErrorCode {
    /// 路径不存在
    NotFound,
    /// 系统拒绝访问（只读文件、无权限目录）
    PermissionDenied,
    /// 目标是目录
    IsDirectory,
    /// 磁盘上的文件已被外部改动，本次写入被拒绝
    Conflict,
    /// 目标编码无法表示文本中的某些字符 —— 拒绝写入，绝不静默替换
    UnrepresentableEncoding,
    /// 不认识的编码标签
    UnknownEncoding,
    /// 路径非法（空、含非法字符、无法规范化）
    InvalidPath,
    /// 该能力尚未实现（占位模块返回）
    NotSupported,
    Io,
    Internal,
}

impl ErrorCode {
    pub fn as_str(self) -> &'static str {
        match self {
            ErrorCode::NotFound => "NOT_FOUND",
            ErrorCode::PermissionDenied => "PERMISSION_DENIED",
            ErrorCode::IsDirectory => "IS_DIRECTORY",
            ErrorCode::Conflict => "CONFLICT",
            ErrorCode::UnrepresentableEncoding => "UNREPRESENTABLE_ENCODING",
            ErrorCode::UnknownEncoding => "UNKNOWN_ENCODING",
            ErrorCode::InvalidPath => "INVALID_PATH",
            ErrorCode::NotSupported => "NOT_SUPPORTED",
            ErrorCode::Io => "IO",
            ErrorCode::Internal => "INTERNAL",
        }
    }
}

impl AppError {
    pub fn new(code: ErrorCode, message: impl Into<String>) -> Self {
        Self { code, message: message.into(), detail: None }
    }

    pub fn with_detail(mut self, detail: impl Into<String>) -> Self {
        self.detail = Some(detail.into());
        self
    }

    pub fn internal(message: impl Into<String>) -> Self {
        Self::new(ErrorCode::Internal, message)
    }

    pub fn not_supported(what: impl Into<String>) -> Self {
        Self::new(ErrorCode::NotSupported, what)
    }

    pub fn invalid_path(message: impl Into<String>) -> Self {
        Self::new(ErrorCode::InvalidPath, message)
    }
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "[{}] {}", self.code.as_str(), self.message)?;
        if let Some(detail) = &self.detail {
            write!(f, " ({detail})")?;
        }
        Ok(())
    }
}

impl std::error::Error for AppError {}

impl From<std::io::Error> for AppError {
    fn from(e: std::io::Error) -> Self {
        use std::io::ErrorKind::*;
        let code = match e.kind() {
            NotFound => ErrorCode::NotFound,
            PermissionDenied => ErrorCode::PermissionDenied,
            // 目录不匹配类错误在 Windows/macOS 上归到 IsDirectory，交由调用侧兜底判断
            _ => ErrorCode::Io,
        };
        AppError {
            code,
            message: e.to_string(),
            detail: None,
        }
    }
}

pub type AppResult<T> = Result<T, AppError>;
