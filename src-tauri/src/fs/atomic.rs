//! 原子写入。
//!
//! 直接 `File::create` + `write_all` 的问题是：写到一半崩溃/断电，文件就成了半截垃圾，
//! 用户几年的笔记就没了。所以一律「同目录临时文件 → 落盘 → rename 覆盖」。
//!
//! 必须同目录：跨盘 rename 不是原子操作，会退化成「复制 + 删除」。

use std::fs::{self, File};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use crate::error::{AppError, AppResult};

fn temp_path_for(target: &Path) -> AppResult<PathBuf> {
    let dir = target
        .parent()
        .filter(|p| !p.as_os_str().is_empty())
        .map(Path::to_path_buf)
        .unwrap_or_else(|| PathBuf::from("."));
    let name = target
        .file_name()
        .ok_or_else(|| AppError::invalid_path(format!("路径没有文件名：{}", target.display())))?
        .to_string_lossy()
        .to_string();

    // 时间戳 + 进程号：并发或上次崩溃残留的临时文件不会互相踩踏
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    Ok(dir.join(format!(
        ".{name}.levnote-{}-{stamp}.tmp",
        std::process::id()
    )))
}

/// 原子写入。成功返回最终字节数。
pub fn write_atomic(target: &Path, bytes: &[u8]) -> AppResult<u64> {
    if target.is_dir() {
        return Err(AppError::new(
            crate::error::ErrorCode::IsDirectory,
            format!("是目录，不能写入：{}", target.display()),
        ));
    }

    let tmp = temp_path_for(target)?;
    if let Some(dir) = tmp.parent() {
        fs::create_dir_all(dir)?;
    }

    let result = (|| -> AppResult<()> {
        let mut file = File::create(&tmp)?;
        file.write_all(bytes)?;
        // 先让内容真正落盘，再做 rename —— 否则断电时 rename 可能先于数据可见
        file.sync_all()?;
        drop(file);

        // 保留原文件的权限位（只读文件等），避免写入后权限被悄悄放大
        if let Ok(meta) = fs::metadata(target) {
            let _ = fs::set_permissions(&tmp, meta.permissions());
        }

        // Windows 上 rename 会覆盖已存在的目标文件；Unix 上本身就是原子替换
        fs::rename(&tmp, target)?;
        Ok(())
    })();

    if result.is_err() {
        // 失败时清掉临时文件，不给用户留垃圾
        let _ = fs::remove_file(&tmp);
    }

    result?;
    Ok(bytes.len() as u64)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn scratch_dir(tag: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("levnote-atomic-{tag}-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn creates_file_and_leaves_no_temp() {
        let dir = scratch_dir("create");
        let target = dir.join("a.txt");
        let n = write_atomic(&target, b"hello").unwrap();
        assert_eq!(n, 5);
        assert_eq!(fs::read_to_string(&target).unwrap(), "hello");

        let leftovers: Vec<_> = fs::read_dir(&dir)
            .unwrap()
            .filter_map(Result::ok)
            .filter(|e| e.file_name().to_string_lossy().contains(".levnote-"))
            .collect();
        assert!(leftovers.is_empty(), "不该留下临时文件");
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn overwrites_existing_content() {
        let dir = scratch_dir("overwrite");
        let target = dir.join("b.txt");
        write_atomic(&target, b"old").unwrap();
        write_atomic(&target, b"new-and-longer").unwrap();
        assert_eq!(fs::read_to_string(&target).unwrap(), "new-and-longer");
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn refuses_directory_target() {
        let dir = scratch_dir("isdir");
        assert!(write_atomic(&dir, b"x").is_err());
        let _ = fs::remove_dir_all(&dir);
    }
}
