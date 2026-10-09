//! 目录遍历与逐文件扫描。
//!
//! 这一层把「走哪些文件、哪些跳过、怎么解码」收在一起，**不碰 IPC** ——
//! 批次通过一个回调交出去，于是它可以被单测直接驱动（临时目录 + 计数回调）。
//!
//! ## 为什么用 `ignore` 而不是自己走 `walkdir`
//!
//! `.gitignore` 的语义一点都不小：否定规则 `!`、前导 `/` 的锚定、目录专属的尾随 `/`、
//! `**` 跨目录、以及「深一层的文件覆盖浅一层」的优先级。自己写一份的代价不是
//! 「写不出来」，而是**写错了不报错**：多排掉一个文件 = 用户的搜索结果里
//! **静静地少一个文件**，与硬约束 2 直接冲突。理由与 `Cargo.toml` 里那段一致。
//!
//! ## 三条会「静静地少东西」的判断，全部计数上报
//!
//! 1. **二进制**（前 8KB 含 NUL）→ `skippedBinary`；
//! 2. **超过单文件上限**（默认 4 MiB）→ `skippedLarge`；
//! 3. **读不出来**（权限、断链、解码失败）→ `skippedUnreadable`。
//!
//! 三个数字都会出现在面板的收尾统计里。用户看不到某个文件时，
//! 至少要能分辨「它被跳过了」和「它里面真的没有这个词」。

use std::fs;
use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use ignore::WalkBuilder;

use super::matcher::Matcher;
use super::{FileMatches, SearchBatch, SearchReason, SearchSummary};
use crate::fs::codec::{self, Bom};

/// 每积累这么多**有命中**的文件就推一批。
const BATCH_FILES: usize = 12;

/// 或者距上次推送超过这么久 —— 哪怕一个命中都没有，也要让进度条动起来。
const BATCH_INTERVAL: Duration = Duration::from_millis(150);

/// 二进制探测的采样长度。
const SNIFF_BYTES: usize = 8 * 1024;

/// 一次检索的可调项。默认值见 `mod.rs`。
#[derive(Debug, Clone, Copy)]
pub struct SearchOptions {
    /// 把被 `.gitignore` 之类规则忽略的文件也搜进来。
    ///
    /// **它关掉的只是「规则」，隐藏项（以 `.` 开头）仍然跳过** ——
    /// 否则 `.git/` 会被整棵搜一遍，那不是用户要的。
    pub include_ignored: bool,
    pub max_results: usize,
    pub max_file_bytes: u64,
}

/// 走完整个目录树，边扫边通过 `on_batch` 推批次。
///
/// `on_batch` 返回 `false` 表示接收端已经没了 —— 此时立刻停下，
/// 继续扫下去只是白烧 CPU。
pub fn run(
    root: &Path,
    matcher: &Matcher,
    options: SearchOptions,
    cancel: &Arc<AtomicBool>,
    on_batch: &mut dyn FnMut(SearchBatch) -> bool,
) -> SearchSummary {
    let started = Instant::now();
    let mut collector = Collector {
        matcher,
        options,
        on_batch,
        pending: Vec::new(),
        matched_files: 0,
        total_matches: 0,
        scanned_files: 0,
        scanned_bytes: 0,
        skipped_binary: 0,
        skipped_large: 0,
        skipped_unreadable: 0,
        last_emit: Instant::now(),
        reason: SearchReason::Completed,
        detached: false,
    };

    let use_ignore = !options.include_ignored;
    let mut builder = WalkBuilder::new(root);
    // 隐藏项一律跳过（`.git` 因此天然不在结果里，不需要另写一条黑名单）
    builder.hidden(true);
    builder.follow_links(false);
    // **不是 git 仓库时也认 `.gitignore`** —— 产品说的是「尊重 .gitignore」，
    // 而不是「在 git 仓库里尊重 .gitignore」
    builder.require_git(false);
    builder
        .git_ignore(use_ignore)
        .git_global(use_ignore)
        .git_exclude(use_ignore)
        .ignore(use_ignore)
        .parents(use_ignore);
    // 结果顺序必须确定：并行走完再排序也行，但那要多存一份全部结果，
    // 而文档编辑器的目录通常不需要那点吞吐。稳定顺序 = 面板里的结果不会自己跳。
    builder.sort_by_file_path(|a, b| a.cmp(b));

    for entry in builder.build() {
        if cancel.load(Ordering::Relaxed) {
            collector.reason = SearchReason::Cancelled;
            break;
        }
        let entry = match entry {
            Ok(entry) => entry,
            Err(_) => {
                collector.skipped_unreadable += 1;
                continue;
            }
        };

        let path = entry.path();
        // 目录交给 walker 自己下去；这里只处理普通文件
        if entry.file_type().map(|kind| kind.is_dir()).unwrap_or(false) {
            continue;
        }
        // `is_file()` 会跟随符号链接：链接到文件的会被搜，链接到目录的不会
        // （`follow_links(false)` 让 walker 不下钻，这里也不当成文件）
        if !path.is_file() {
            continue;
        }

        if !collector.visit(path) {
            if collector.detached {
                collector.reason = SearchReason::Cancelled;
            }
            break;
        }
    }

    collector.finish(started)
}

struct Collector<'a> {
    matcher: &'a Matcher,
    options: SearchOptions,
    on_batch: &'a mut dyn FnMut(SearchBatch) -> bool,
    pending: Vec<FileMatches>,
    matched_files: u64,
    total_matches: u64,
    scanned_files: u64,
    scanned_bytes: u64,
    skipped_binary: u64,
    skipped_large: u64,
    skipped_unreadable: u64,
    last_emit: Instant,
    reason: SearchReason,
    /// 接收端已经断开。与 `reason` 分开记，因为断开也要映射到「取消」，
    /// 而「取消」还可能来自用户点停止 —— 两个来源得能分辨。
    detached: bool,
}

impl Collector<'_> {
    /// 处理一个文件。返回 `false` 表示应当停止整个遍历。
    fn visit(&mut self, path: &Path) -> bool {
        let size = match fs::metadata(path) {
            Ok(meta) => meta.len(),
            Err(_) => {
                self.skipped_unreadable += 1;
                return true;
            }
        };
        if size > self.options.max_file_bytes {
            self.skipped_large += 1;
            return true;
        }

        let bytes = match fs::read(path) {
            Ok(bytes) => bytes,
            Err(_) => {
                self.skipped_unreadable += 1;
                return true;
            }
        };
        if looks_binary(&bytes) {
            self.skipped_binary += 1;
            return true;
        }
        let text = match decode_text(&bytes) {
            Some(text) => text,
            None => {
                self.skipped_unreadable += 1;
                return true;
            }
        };

        self.scanned_files += 1;
        self.scanned_bytes += size;

        let remaining = self.options.max_results.saturating_sub(self.total_matches as usize);
        if remaining == 0 {
            self.reason = SearchReason::Truncated;
            return false;
        }

        let hits = self.matcher.scan(&text, remaining);
        if !hits.is_empty() {
            self.total_matches += hits.len() as u64;
            self.matched_files += 1;
            self.pending.push(FileMatches {
                path: path.to_string_lossy().to_string(),
                matches: hits,
            });
        }

        if self.pending.len() >= BATCH_FILES || self.last_emit.elapsed() >= BATCH_INTERVAL {
            self.flush(false);
            if self.detached {
                return false;
            }
        }

        if self.total_matches >= self.options.max_results as u64 {
            self.reason = SearchReason::Truncated;
            return false;
        }
        true
    }

    /// 把攒着的批次推出去。
    ///
    /// **没有命中也要推** —— 否则扫一个大目录时进度条一动不动，
    /// 用户会以为它卡死了。`force` 只是让收尾那一次忽略时间间隔。
    fn flush(&mut self, _force: bool) {
        let batch = SearchBatch {
            files: std::mem::take(&mut self.pending),
            matched_files: self.matched_files,
            total_matches: self.total_matches,
            scanned_files: self.scanned_files,
            scanned_bytes: self.scanned_bytes,
            skipped_binary: self.skipped_binary,
            skipped_large: self.skipped_large,
            skipped_unreadable: self.skipped_unreadable,
        };
        if !(self.on_batch)(batch) {
            self.detached = true;
        }
        self.last_emit = Instant::now();
    }

    fn finish(mut self, started: Instant) -> SearchSummary {
        if !self.pending.is_empty() {
            self.flush(true);
        }
        SearchSummary {
            reason: self.reason,
            matched_files: self.matched_files,
            total_matches: self.total_matches,
            scanned_files: self.scanned_files,
            scanned_bytes: self.scanned_bytes,
            skipped_binary: self.skipped_binary,
            skipped_large: self.skipped_large,
            skipped_unreadable: self.skipped_unreadable,
            elapsed_ms: started.elapsed().as_millis() as u64,
        }
    }
}

/// 前 8KB 里有没有 NUL。这是 ripgrep 用的同一条启发式。
///
/// **先看 BOM**：UTF-16 的文本里**必然**每隔一个字节就是 0x00，
/// 直接按 NUL 判会把它当成二进制 —— 而我们的编解码层是支持 UTF-16 的
/// （`fs/codec.rs`），于是「打开能看、检索搜不到」。
fn looks_binary(bytes: &[u8]) -> bool {
    let (bom, rest) = codec::strip_bom(bytes);
    if matches!(bom, Bom::Utf16Le | Bom::Utf16Be) {
        return false;
    }
    let head = &rest[..rest.len().min(SNIFF_BYTES)];
    head.contains(&0)
}

/// 解码成文本。
///
/// **快路径是必须的**：`codec::decode` 会跑 `chardetng` 做统计式探测，
/// 而目录里绝大多数文件都是无 BOM 的 UTF-8 —— 对每个文件都跑一遍探测，
/// 会把「扫一万个文件」的时间翻好几倍。UTF-8 合法的直接返回。
///
/// 其余（GBK / Big5 / UTF-16 / 有 BOM 的 UTF-8）一律交给 `codec::decode`，
/// 与打开文件走**同一个**解码器 —— 否则会出现「打开看得见、检索搜不到」
/// 那种最难查的分叉。
fn decode_text(bytes: &[u8]) -> Option<String> {
    let (bom, rest) = codec::strip_bom(bytes);
    if matches!(bom, Bom::None | Bom::Utf8) {
        if let Ok(text) = std::str::from_utf8(rest) {
            return Some(text.to_owned());
        }
    }
    codec::decode(bytes, None).ok().map(|decoded| decoded.text)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs as stdfs;
    use std::path::PathBuf;

    fn options() -> SearchOptions {
        SearchOptions {
            include_ignored: false,
            max_results: 2000,
            max_file_bytes: 4 * 1024 * 1024,
        }
    }

    /// 建一个临时目录。名字带进程号与序号，避免并发测试互相踩。
    fn scratch(name: &str) -> PathBuf {
        let base = std::env::temp_dir().join(format!("levnote-search-{}-{}", std::process::id(), name));
        let _ = stdfs::remove_dir_all(&base);
        stdfs::create_dir_all(&base).unwrap();
        base
    }

    fn write(root: &Path, relative: &str, contents: impl AsRef<[u8]>) {
        let path = root.join(relative);
        if let Some(parent) = path.parent() {
            stdfs::create_dir_all(parent).unwrap();
        }
        stdfs::write(&path, contents).unwrap();
    }

    /// 走一遍，收集全部命中与收尾统计。
    fn search(root: &Path, query: &str, options: SearchOptions) -> (Vec<(String, usize)>, SearchSummary) {
        let matcher = Matcher::build(query, false, false, false).unwrap();
        let cancel = Arc::new(AtomicBool::new(false));
        let mut found: Vec<(String, usize)> = Vec::new();
        let summary = run(root, &matcher, options, &cancel, &mut |batch| {
            for file in batch.files {
                found.push((file.path, file.matches.len()));
            }
            true
        });
        (found, summary)
    }

    #[test]
    fn finds_matches_and_reports_counts() {
        let root = scratch("basic");
        write(&root, "a.md", "第一行\n这里有 needle\n第三行\n");
        write(&root, "b/c.txt", "needle needle");

        let (found, summary) = search(&root, "needle", options());
        assert_eq!(found.len(), 2);
        assert_eq!(summary.matched_files, 2);
        assert_eq!(summary.total_matches, 3);
        assert_eq!(summary.scanned_files, 2);
        assert_eq!(summary.reason, SearchReason::Completed);
        assert_eq!(summary.skipped_binary, 0);
    }

    #[test]
    fn respects_gitignore() {
        let root = scratch("gitignore");
        write(&root, ".gitignore", "ignored/\n*.log\n");
        write(&root, "keep.md", "needle");
        write(&root, "ignored/drop.md", "needle");
        write(&root, "noisy.log", "needle");

        let (found, _) = search(&root, "needle", options());
        assert_eq!(found.len(), 1);
        assert!(found[0].0.ends_with("keep.md"));
    }

    #[test]
    fn include_ignored_turns_the_rules_off() {
        let root = scratch("include-ignored");
        write(&root, ".gitignore", "*.log\n");
        write(&root, "keep.md", "needle");
        write(&root, "noisy.log", "needle");

        let (found, _) = search(
            &root,
            "needle",
            SearchOptions {
                include_ignored: true,
                ..options()
            },
        );
        assert_eq!(found.len(), 2);
    }

    #[test]
    fn hidden_entries_are_skipped_even_when_ignored_rules_are_off() {
        // `.git` 必须始终不在结果里：它是内部数据，而且大得离谱
        let root = scratch("hidden");
        write(&root, ".git/config", "needle");
        write(&root, "keep.md", "needle");

        let (found, _) = search(
            &root,
            "needle",
            SearchOptions {
                include_ignored: true,
                ..options()
            },
        );
        assert_eq!(found.len(), 1);
        assert!(found[0].0.ends_with("keep.md"));
    }

    #[test]
    fn binary_files_are_counted_not_silently_dropped() {
        let root = scratch("binary");
        write(&root, "keep.md", "needle");
        write(&root, "blob.bin", b"\x00\x01\x02needle");

        let (found, summary) = search(&root, "needle", options());
        assert_eq!(found.len(), 1);
        assert_eq!(summary.skipped_binary, 1);
    }

    #[test]
    fn oversized_files_are_counted() {
        let root = scratch("large");
        write(&root, "keep.md", "needle");
        write(&root, "big.md", "needle".repeat(100).as_bytes());
        let (_, summary) = search(
            &root,
            "needle",
            SearchOptions {
                max_file_bytes: 16,
                ..options()
            },
        );
        assert_eq!(summary.skipped_large, 1);
        assert_eq!(summary.matched_files, 1);
    }

    #[test]
    fn utf16_files_are_not_mistaken_for_binary() {
        let root = scratch("utf16");
        let mut bytes = vec![0xFF, 0xFE];
        for unit in "needle".encode_utf16() {
            bytes.extend_from_slice(&unit.to_le_bytes());
        }
        write(&root, "u16.txt", &bytes);

        let (found, summary) = search(&root, "needle", options());
        assert_eq!(found.len(), 1);
        assert_eq!(summary.skipped_binary, 0);
    }

    #[test]
    fn gbk_files_are_decoded_with_the_shared_codec() {
        let root = scratch("gbk");
        // 「中文测试」的 GBK 字节
        write(&root, "gbk.txt", &[0xD6, 0xD0, 0xCE, 0xC4, 0xB2, 0xE2, 0xCA, 0xD4]);

        let (found, _) = search(&root, "测试", options());
        assert_eq!(found.len(), 1);
    }

    #[test]
    fn truncates_at_max_results_and_says_so() {
        let root = scratch("truncate");
        write(&root, "many.md", "needle ".repeat(50).as_bytes());

        let (found, summary) = search(
            &root,
            "needle",
            SearchOptions {
                max_results: 5,
                ..options()
            },
        );
        assert_eq!(found.iter().map(|(_, n)| n).sum::<usize>(), 5);
        assert_eq!(summary.total_matches, 5);
        assert_eq!(summary.reason, SearchReason::Truncated);
    }

    #[test]
    fn cancel_stops_the_walk_early() {
        let root = scratch("cancel");
        for index in 0..200 {
            write(&root, &format!("f{index:03}.md"), "needle");
        }

        let matcher = Matcher::build("needle", false, false, false).unwrap();
        let cancel = Arc::new(AtomicBool::new(false));
        let flag = Arc::clone(&cancel);
        let summary = run(&root, &matcher, options(), &cancel, &mut |_| {
            // 收到第一批就取消
            flag.store(true, Ordering::Relaxed);
            true
        });
        assert_eq!(summary.reason, SearchReason::Cancelled);
        assert!(summary.scanned_files < 200);
    }

    #[test]
    fn a_detached_receiver_stops_the_walk() {
        let root = scratch("detach");
        for index in 0..50 {
            write(&root, &format!("f{index:03}.md"), "needle");
        }

        let matcher = Matcher::build("needle", false, false, false).unwrap();
        let cancel = Arc::new(AtomicBool::new(false));
        let summary = run(&root, &matcher, options(), &cancel, &mut |_| false);
        assert_eq!(summary.reason, SearchReason::Cancelled);
        assert!(summary.scanned_files < 50);
    }

    #[test]
    fn results_are_ordered_deterministically() {
        let root = scratch("order");
        write(&root, "z.md", "needle");
        write(&root, "a.md", "needle");
        write(&root, "m/n.md", "needle");

        let (found, _) = search(&root, "needle", options());
        let names: Vec<String> = found
            .iter()
            .map(|(path, _)| Path::new(path).file_name().unwrap().to_string_lossy().to_string())
            .collect();
        // 深度优先 + 每层按路径排序：a.md → 进 m/ → n.md → z.md
        assert_eq!(names, vec!["a.md", "n.md", "z.md"]);
    }

    #[test]
    fn missing_root_is_not_a_crash() {
        let root = scratch("missing").join("nope");
        let (found, summary) = search(&root, "needle", options());
        assert!(found.is_empty());
        assert_eq!(summary.total_matches, 0);
    }
}
