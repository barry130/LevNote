//! 检索词的编译与**逐行**匹配。
//!
//! 这一层是纯函数（不碰文件系统、不碰 IPC），所以它是整个检索里最该被单测钉住的部分。
//!
//! ## 为什么逐行
//!
//! 三个理由，缺一个都够不上「逐行」这个设计：
//! 1. **结果要能跳到**：用户点一条命中，要跳过去。跳转的落点是「第几行第几列」，
//!    而这个坐标只有在「按行扫」时才是天然的 —— 全篇扫出来的字节偏移还得再
//!    反查一次它落在哪一行。
//! 2. **跨行匹配没有意义**：用户在检索框里写 `a.*b`，期待的是「同一行里的 a 和 b」，
//!    而不是「从文件开头到结尾」。逐行让 `^` / `$` 自然等于行首行尾，不需要 `(?m)`。
//! 3. **大文件**：一行一行地处理，内存里只多一行，不需要把整篇再切一份。
//!
//! 代价要写清楚：**用户的正则无法跨行匹配**。这是刻意的，不是缺失 ——
//! 跨行匹配的命中没法用一个「第几行」表示。
//!
//! ## 与文档内查找（`src/core/search/find.ts`）的语义对齐
//!
//! 两个检索入口在同一份 UI 语言里出现（「区分大小写 / 全词 / 正则」），
//! 所以语义必须一致，否则就是本项目最忌讳的「两份实现悄悄分叉」。
//! 唯一一处刻意的差异写在 `build()` 里（`\b` 的词边界口径），并在那里说明了理由。

use regex::{Regex, RegexBuilder};
use serde::Serialize;

use crate::error::{AppError, AppResult};

/// 预览里最多显示多少个 UTF-16 码元。
///
/// 240 是「侧栏宽度 240px 时大约三行」的量级。它同时兜住了另一件事：
/// 压缩过的 JS、一行几万字的表格导出，如果原样把整行推给前端，
/// 一条命中就能撑出几十 KB 的 IPC 负载，几百条命中直接把面板卡死。
const PREVIEW_MAX: u32 = 240;

/// 预览窗口在命中**之前**保留多少个 UTF-16 码元。
///
/// 命中太靠后时窗口整体往右挪 —— 保留一点上下文，用户才知道这句话是在说什么的。
const PREVIEW_LEAD: u32 = 72;

/// 已经编译好的检索词。
#[derive(Debug)]
pub struct Matcher {
    pattern: Regex,
}

/// 一处命中。
///
/// 它直接就是 IPC 上的那个形状（不做一层 `MatchLine` 的搬运）：
/// 多一层转换意味着两处字段表，而「加了字段忘了改另一处」的表现是
/// 前端拿到 `undefined` 却一路安静 —— 与本项目一贯的取舍一致。
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LineHit {
    /// 1 基行号。
    pub line: u32,
    /// 命中在**整行**里的起始列，以 UTF-16 码元计（理由见 `mod.rs`）。
    pub column: u32,
    /// 命中长度，同样以 UTF-16 码元计。零宽匹配是 0。
    pub length: u32,
    /// 给用户看的整行（可能被截断）。
    pub text: String,
    /// `text` 从整行的第几个 UTF-16 码元开始。大于 0 表示前面被截掉了。
    pub text_offset: u32,
    /// `text` 后面还有没显示出来的内容。
    pub tail_cut: bool,
}

impl Matcher {
    /// 把用户的输入编译成模式。
    ///
    /// 四条语义与文档内查找一致：**区分大小写 / 全词 / 正则**。
    /// 非法正则返回 `Err` 而不是「当作没命中」：这里与 `find.ts` 相反，
    /// 因为那边是**边打字边查**（`(` 的中间态就是非法正则，报错会让查找栏一直闪红），
    /// 而这边是**停下来才查**（面板有防抖，用户停手才发起）。
    /// 这时候还报错，说明用户是真的写错了，该告诉他。
    pub fn build(
        query: &str,
        case_sensitive: bool,
        whole_word: bool,
        regex: bool,
    ) -> AppResult<Self> {
        if query.is_empty() {
            return Err(AppError::invalid_path("检索词为空"));
        }

        let mut source = if regex {
            query.to_string()
        } else {
            regex::escape(query)
        };
        if whole_word {
            // `(?-u:\b)` 而不是裸 `\b`：**刻意与 `find.ts` 对齐**。
            // JS 的 `\b` 按 ASCII 的 `\w`（`[A-Za-z0-9_]`）定义，
            // 而 Rust 的 `regex` 默认是 Unicode 词边界 —— 两者在中文上行为不同：
            // 裸 `\b` 会把「配置」两侧算成词边界，于是勾了「全词」反而**多**出命中，
            // 而用户以为自己只是要求「它是独立的一个词」。
            source = format!(r"(?-u:\b)(?:{source})(?-u:\b)");
        }

        let pattern = RegexBuilder::new(&source)
            .case_insensitive(!case_sensitive)
            // 关掉 `.` 匹配换行：逐行匹配本来就用不上，显式关掉免得
            // 「同一份正则在两个检索入口里行为不同」
            .dot_matches_new_line(false)
            .build()
            .map_err(|e| AppError::invalid_path(format!("正则表达式无效：{e}")))?;

        Ok(Self { pattern })
    }

    /// 一行里的全部命中，返回**字节**区间。
    ///
    /// 零宽匹配（`.*` / `a*` / `^`）的前进由 `regex` 自己的 `find_iter` 处理，
    /// 不会死循环 —— 但会产出长度 0 的命中，上层要如实画出来（标一个光标位），
    /// 而不是丢掉它。
    fn hits(&self, line: &str) -> Vec<(usize, usize)> {
        self.pattern
            .find_iter(line)
            .map(|m| (m.start(), m.end()))
            .collect()
    }

    /// 扫一整篇文本，最多返回 `max` 处命中。
    pub fn scan(&self, text: &str, max: usize) -> Vec<LineHit> {
        let mut out: Vec<LineHit> = Vec::new();
        if max == 0 {
            return out;
        }

        for (index, line) in split_lines(text).enumerate() {
            if out.len() >= max {
                break;
            }
            let raw = self.hits(line);
            if raw.is_empty() {
                continue;
            }

            // 只在「这一行真的有命中」时才建 UTF-16 映射表：绝大多数行都进不到这里，
            // 而建表是 O(行长) 的
            let map = Utf16Map::new(line);
            let line_no = index as u32 + 1;
            for (start, end) in raw {
                if out.len() >= max {
                    break;
                }
                let column = map.at(start);
                let length = map.at(end).saturating_sub(column);
                let (text, text_offset, tail_cut) = preview(line, &map, column);
                out.push(LineHit {
                    line: line_no,
                    column,
                    length,
                    text,
                    text_offset,
                    tail_cut,
                });
            }
        }

        out
    }
}

/// 按行切开。**CR 单独成行**也要认 —— 老式 Mac 行尾的文件在 `str::lines()`
/// 眼里只有一行，于是整个文件会被当成「第 1 行」推给用户，跳转也就跳不准。
fn split_lines(text: &str) -> Box<dyn Iterator<Item = &str> + '_> {
    if text.contains('\n') {
        Box::new(
            text.split('\n')
                .map(|line| line.strip_suffix('\r').unwrap_or(line)),
        )
    } else if text.contains('\r') {
        Box::new(text.split('\r'))
    } else {
        Box::new(std::iter::once(text))
    }
}

/// 字节偏移 ↔ UTF-16 码元偏移的对照表。
///
/// 只在有命中的行上建，一次遍历 O(行长)，之后每次换算 O(log 行长)。
/// 为什么不每次从头数：一行里可能有上千处命中（压缩文件里搜一个短词），
/// 那样就是 O(行长 × 命中数)。
struct Utf16Map {
    /// 每项是「某个字符的起始字节」→「它的起始 UTF-16 码元」，
    /// 末尾额外压一个哨兵 `(line.len(), 总码元数)`。
    table: Vec<(usize, u32)>,
}

impl Utf16Map {
    fn new(line: &str) -> Self {
        let mut table = Vec::with_capacity(line.chars().count() + 1);
        let mut units = 0u32;
        for (byte, ch) in line.char_indices() {
            table.push((byte, units));
            units += ch.len_utf16() as u32;
        }
        table.push((line.len(), units));
        Self { table }
    }

    /// 某个字节偏移对应的 UTF-16 码元偏移。正则的匹配边界一定落在字符边界上，
    /// 所以这里总能精确命中；命不中时回落到「前一个字符的起点」，绝不 panic。
    fn at(&self, byte: usize) -> u32 {
        match self.table.binary_search_by_key(&byte, |&(b, _)| b) {
            Ok(index) => self.table[index].1,
            Err(0) => 0,
            Err(index) => self.table[index - 1].1,
        }
    }

    fn total(&self) -> u32 {
        self.table.last().map(|&(_, units)| units).unwrap_or(0)
    }

    fn byte_at(&self, units: u32) -> usize {
        match self.table.binary_search_by_key(&units, |&(_, u)| u) {
            Ok(index) | Err(index) => self.table[index.min(self.table.len() - 1)].0,
        }
    }
}

/// 取一段「围着命中」的窗口。返回 `(文本, 窗口起点, 后面是否还有内容)`。
///
/// 窗口是 **`PREVIEW_LEAD` 在命中之前 + `PREVIEW_MAX` 那么宽**，
/// 而不是「从命中前 72 个码元截到命中结束」——
/// 后者在短行上会把命中**之后**的上下文全部丢掉
/// （`say hit here` 只会显示 `say hit`），而那半句往往正是用户判断
/// 「这是不是我要找的那一处」的依据。
fn preview(line: &str, map: &Utf16Map, column: u32) -> (String, u32, bool) {
    let total = map.total();
    let start = column.saturating_sub(PREVIEW_LEAD).min(total);
    let end = start
        .saturating_add(PREVIEW_MAX)
        .min(total)
        // 命中极长（一行几万字符被一次匹配掉）时会被 `PREVIEW_MAX` 卡住，
        // 但至少要让窗口盖住命中的**起点** —— 否则窗口里一个命中字符都没有，
        // 前端连高亮都画不出来
        .max(column.min(total))
        .max(start);

    let from = map.byte_at(start);
    let to = map.byte_at(end).max(from);
    // 两端的字节值都来自 `char_indices`（或 `line.len()`），一定落在字符边界上
    (line[from..to].to_string(), start, end < total)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn plain(query: &str) -> Matcher {
        Matcher::build(query, false, false, false).unwrap()
    }

    fn columns(matcher: &Matcher, text: &str) -> Vec<(u32, u32)> {
        matcher
            .scan(text, 100)
            .into_iter()
            .map(|h| (h.column, h.length))
            .collect()
    }

    #[test]
    fn literal_match_is_case_insensitive_by_default() {
        assert_eq!(
            columns(&plain("abc"), "ABC abc AbC"),
            vec![(0, 3), (4, 3), (8, 3)]
        );
    }

    #[test]
    fn case_sensitive_narrows_results() {
        let matcher = Matcher::build("abc", true, false, false).unwrap();
        assert_eq!(columns(&matcher, "ABC abc"), vec![(4, 3)]);
    }

    #[test]
    fn literal_query_treats_regex_metacharacters_as_text() {
        // 不勾「正则」时 `a.c` 必须只匹配字面的 `a.c`
        assert_eq!(columns(&plain("a.c"), "a.c abc"), vec![(0, 3)]);
    }

    #[test]
    fn regex_mode_uses_the_pattern() {
        let matcher = Matcher::build("a.c", false, false, true).unwrap();
        assert_eq!(columns(&matcher, "a.c abc"), vec![(0, 3), (4, 3)]);
    }

    #[test]
    fn invalid_regex_is_reported_not_swallowed() {
        let err = Matcher::build("a(", false, false, true).unwrap_err();
        assert!(err.message.contains("正则"));
    }

    #[test]
    fn empty_query_is_rejected() {
        assert!(Matcher::build("", false, false, false).is_err());
    }

    #[test]
    fn whole_word_uses_ascii_boundaries_like_the_js_side() {
        let matcher = Matcher::build("cat", false, true, false).unwrap();
        assert_eq!(columns(&matcher, "cats"), Vec::<(u32, u32)>::new());
        assert_eq!(columns(&matcher, "a cat b"), vec![(2, 3)]);
    }

    #[test]
    fn whole_word_on_cjk_finds_nothing_like_the_js_side() {
        // JS 的 `\b` 不把汉字当词字符，所以「全词 + 中文」在两个入口里都查不到东西。
        // 这条测试是**故意的**：它钉住「两个检索入口语义一致」，而不是钉住「这个行为好用」。
        let matcher = Matcher::build("配置", false, true, false).unwrap();
        assert!(columns(&matcher, "系统配置项").is_empty());
    }

    #[test]
    fn zero_width_matches_do_not_loop_forever() {
        let matcher = Matcher::build(r"\b", false, false, true).unwrap();
        let hits = columns(&matcher, "ab cd");
        assert_eq!(hits.len(), 4);
        assert!(hits.iter().all(|&(_, len)| len == 0));
    }

    #[test]
    fn columns_are_utf16_units_not_bytes() {
        // 「中」是 3 字节 / 1 个 UTF-16 码元；emoji 是 4 字节 / 2 个码元。
        // 列号必须按 UTF-16 算 —— 下游是 JS 字符串下标与 CodeMirror 的 `ch`。
        assert_eq!(columns(&plain("b"), "中a😀b"), vec![(4, 1)]);
    }

    #[test]
    fn line_numbers_are_one_based_and_crlf_safe() {
        let hits = plain("hit").scan("a\r\nhit\r\nb\nhit", 10);
        assert_eq!(hits.iter().map(|h| h.line).collect::<Vec<_>>(), vec![2, 4]);
    }

    #[test]
    fn cr_only_files_are_split_into_lines() {
        let hits = plain("hit").scan("a\rhit\rb", 10);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].line, 2);
    }

    #[test]
    fn max_caps_the_result_count() {
        assert_eq!(plain("a").scan("a a a a a", 3).len(), 3);
    }

    #[test]
    fn preview_keeps_the_head_of_a_short_line() {
        let hits = plain("hit").scan("say hit here", 10);
        assert_eq!(hits[0].text, "say hit here");
        assert_eq!(hits[0].text_offset, 0);
        assert!(!hits[0].tail_cut);
    }

    #[test]
    fn preview_windows_around_a_far_match() {
        let line = format!("{}hit{}", "x".repeat(1000), "y".repeat(1000));
        let hits = plain("hit").scan(&line, 10);
        assert_eq!(hits[0].column, 1000);
        // 窗口起点 = 1000 - 72
        assert_eq!(hits[0].text_offset, 928);
        assert!(hits[0].text.starts_with(&"x".repeat(72)));
        assert!(hits[0].text[72..].starts_with("hit"));
        // 后面还有 1000 个 y 没显示
        assert!(hits[0].tail_cut);
    }

    #[test]
    fn preview_keeps_the_rest_of_a_line_that_fits() {
        // 短行必须整行显示（而不是「截到命中结束」）—— 命中后面那半句
        // 往往正是用户判断「这是不是我要找的那一处」的依据
        let line = format!("{}hit{}", "x".repeat(1000), "y".repeat(100));
        let hits = plain("hit").scan(&line, 10);
        assert_eq!(hits[0].text_offset, 928);
        assert_eq!(
            hits[0].text,
            format!("{}hit{}", "x".repeat(72), "y".repeat(100))
        );
        assert!(!hits[0].tail_cut);
    }

    #[test]
    fn preview_truncates_an_enormous_match() {
        // 一行几万个字符、正则一次匹配整行：窗口必须被 `PREVIEW_MAX` 截住，
        // 而不是把整行推给前端
        let line = "y".repeat(50_000);
        let matcher = Matcher::build("y+", false, false, true).unwrap();
        let hits = matcher.scan(&line, 10);
        assert_eq!(hits[0].length, 50_000);
        assert!(hits[0].text.chars().map(char::len_utf16).sum::<usize>() <= PREVIEW_MAX as usize);
        assert!(hits[0].tail_cut);
    }

    #[test]
    fn preview_handles_an_empty_line() {
        // 空行 + 零宽匹配：不能 panic，也不能切出越界区间
        let matcher = Matcher::build("^", false, false, true).unwrap();
        let hits = matcher.scan("", 10);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].text, "");
        assert_eq!(hits[0].column, 0);
        assert!(!hits[0].tail_cut);
    }

    #[test]
    fn scan_reports_every_hit_on_a_line() {
        let hits = plain("ab").scan("ab-ab-ab", 10);
        assert_eq!(
            hits.iter().map(|h| h.column).collect::<Vec<_>>(),
            vec![0, 3, 6]
        );
        assert!(hits.iter().all(|h| h.text == "ab-ab-ab"));
    }

    #[test]
    fn multi_byte_characters_survive_the_preview_slice() {
        // 窗口边界落在多字节字符中间是**唯一**会让切片 panic 的情形
        let line = format!("{}命中{}", "中".repeat(100), "文".repeat(100));
        let hits = plain("命中").scan(&line, 10);
        assert_eq!(hits.len(), 1);
        assert!(hits[0].text.contains("命中"));
    }
}
