//! 编码与行尾的探测 / 解码 / 编码。
//!
//! 原则（写进 docs/format-spec.md，属于不可退让的红线）：
//! 1. **绝不静默转码**。打开时用探测到的编码解码，保存时写回同一编码。
//! 2. 若目标编码无法表示文本中的字符，**报错**而不是用 `&#NNNN;` 之类实体替换。
//! 3. 行尾、BOM 属于文件属性，原样保留，除非调用方显式要求规范化。

use chardetng::EncodingDetector;
use encoding_rs::Encoding;

use crate::error::{AppError, ErrorCode};

/// BOM 类型。`None` 表示无 BOM（此时编码交给探测）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Bom {
    None,
    Utf8,
    Utf16Le,
    Utf16Be,
}

impl Bom {
    pub fn as_str(self) -> &'static str {
        match self {
            Bom::None => "none",
            Bom::Utf8 => "utf-8",
            Bom::Utf16Le => "utf-16le",
            Bom::Utf16Be => "utf-16be",
        }
    }

    pub fn bytes(self) -> &'static [u8] {
        match self {
            Bom::None => &[],
            Bom::Utf8 => &[0xEF, 0xBB, 0xBF],
            Bom::Utf16Le => &[0xFF, 0xFE],
            Bom::Utf16Be => &[0xFE, 0xFF],
        }
    }
}

/// 探测 + 解码的结果。
#[derive(Debug, Clone)]
pub struct Decoded {
    pub text: String,
    /// 规范化后的编码名，如 `UTF-8` / `GBK` / `Big5` / `windows-1252`
    pub encoding: String,
    pub bom: Bom,
    /// 解码过程中出现过非法字节（说明探测结果不可靠，UI 可提示用户手动指定）
    pub lossy: bool,
    pub line_ending: String,
}

pub fn strip_bom(bytes: &[u8]) -> (Bom, &[u8]) {
    if bytes.starts_with(&[0xEF, 0xBB, 0xBF]) {
        (Bom::Utf8, &bytes[3..])
    } else if bytes.starts_with(&[0xFF, 0xFE]) {
        (Bom::Utf16Le, &bytes[2..])
    } else if bytes.starts_with(&[0xFE, 0xFF]) {
        (Bom::Utf16Be, &bytes[2..])
    } else {
        (Bom::None, bytes)
    }
}

fn bom_encoding(bom: Bom) -> Option<&'static Encoding> {
    match bom {
        Bom::None => None,
        Bom::Utf8 => Some(encoding_rs::UTF_8),
        Bom::Utf16Le => Some(encoding_rs::UTF_16LE),
        Bom::Utf16Be => Some(encoding_rs::UTF_16BE),
    }
}

/// 无 BOM 时用 chardetng 猜编码。`None` 表示「允许任意编码」，比限定 CJK 更稳。
fn guess_encoding(bytes: &[u8]) -> &'static Encoding {
    let mut det = EncodingDetector::new();
    det.feed(bytes, true);
    det.guess(None, true)
}

/// 通过标签找编码，供 UI 手动指定编码时使用。
pub fn lookup_encoding(label: &str) -> Result<&'static Encoding, AppError> {
    Encoding::for_label(label.as_bytes())
        .ok_or_else(|| AppError::new(ErrorCode::UnknownEncoding, format!("不认识的编码：{label}")))
}

/// 探测并解码。`forced` 为 Some 时跳过探测，强制用指定编码。
pub fn decode(bytes: &[u8], forced: Option<&str>) -> Result<Decoded, AppError> {
    let (bom, rest) = strip_bom(bytes);

    let enc = match forced {
        Some(label) => lookup_encoding(label)?,
        None => bom_encoding(bom).unwrap_or_else(|| guess_encoding(rest)),
    };

    // 用 without_bom_handling：BOM 已在上一步剥掉，避免编码库再次嗅探覆盖我们的判断。
    let (cow, had_errors) = enc.decode_without_bom_handling(rest);
    let text = cow.into_owned();

    Ok(Decoded {
        encoding: enc.name().to_string(),
        line_ending: detect_line_ending(&text).to_string(),
        lossy: had_errors,
        bom,
        text,
    })
}

/// 主要行尾。混排时返回 `mixed`，交由调用方决定是否规范化。
pub fn detect_line_ending(text: &str) -> &'static str {
    let crlf = text.matches("\r\n").count();
    let total_lf = text.matches('\n').count();
    let total_cr = text.matches('\r').count();
    let lone_lf = total_lf - crlf;
    let lone_cr = total_cr - crlf;

    if crlf > 0 && (lone_lf > 0 || lone_cr > 0) {
        return "mixed";
    }
    if crlf > 0 {
        return "crlf";
    }
    if lone_cr > 0 {
        return "cr";
    }
    "lf"
}

/// 把文本里的所有换行统一成 `ending`（`lf` / `crlf` / `cr`）。
pub fn normalize_line_endings(text: &str, ending: &str) -> String {
    let unified = text.replace("\r\n", "\n").replace('\r', "\n");
    match ending {
        "crlf" => unified.replace('\n', "\r\n"),
        "cr" => unified.replace('\n', "\r"),
        _ => unified,
    }
}

/// 编码为字节。**无法表示的字符直接报错**，不做实体替换。
pub fn encode(text: &str, label: &str, bom: Bom) -> Result<Vec<u8>, AppError> {
    let enc = lookup_encoding(label)?;
    let (bytes, _, had_errors) = enc.encode(text);
    if had_errors {
        return Err(AppError::new(
            ErrorCode::UnrepresentableEncoding,
            format!(
                "目标编码 {} 无法表示文本中的部分字符，已取消写入以避免内容损坏",
                enc.name()
            ),
        ));
    }
    // encode() 对不可表示字符会退回数字实体，这里做一次「编码→解码」回读校验作为双保险。
    let (back, _) = enc.decode_without_bom_handling(&bytes);
    if back != text {
        return Err(AppError::new(
            ErrorCode::UnrepresentableEncoding,
            format!(
                "以 {} 编码后回读内容不一致，已取消写入以避免内容损坏",
                enc.name()
            ),
        ));
    }

    let prefix = bom.bytes();
    let mut out = Vec::with_capacity(prefix.len() + bytes.len());
    out.extend_from_slice(prefix);
    out.extend_from_slice(&bytes);
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn detects_utf8_bom() {
        let mut bytes = vec![0xEF, 0xBB, 0xBF];
        bytes.extend_from_slice("你好".as_bytes());
        let d = decode(&bytes, None).unwrap();
        assert_eq!(d.encoding, "UTF-8");
        assert_eq!(d.bom, Bom::Utf8);
        assert_eq!(d.text, "你好");
        assert!(!d.lossy);
    }

    #[test]
    fn detects_gbk_without_bom() {
        // “中文测试” 的 GBK 字节
        let bytes = [0xD6, 0xD0, 0xCE, 0xC4, 0xB2, 0xE2, 0xCA, 0xD4];
        let d = decode(&bytes, None).unwrap();
        assert_eq!(d.text, "中文测试");
        assert!(d.encoding == "GBK" || d.encoding == "GB18030");
    }

    #[test]
    fn detects_utf16le_bom_and_crlf() {
        let mut bytes = vec![0xFF, 0xFE];
        for unit in "a\r\nb".encode_utf16() {
            bytes.extend_from_slice(&unit.to_le_bytes());
        }
        let d = decode(&bytes, None).unwrap();
        assert_eq!(d.bom, Bom::Utf16Le);
        assert_eq!(d.text, "a\r\nb");
        assert_eq!(d.line_ending, "crlf");
    }

    #[test]
    fn line_ending_mixed() {
        assert_eq!(detect_line_ending("a\nb\r\nc"), "mixed");
        assert_eq!(detect_line_ending("a\nb"), "lf");
        assert_eq!(detect_line_ending("a\r\nb\r\n"), "crlf");
    }

    #[test]
    fn normalize_to_crlf_and_back() {
        assert_eq!(normalize_line_endings("a\nb\n", "crlf"), "a\r\nb\r\n");
        assert_eq!(normalize_line_endings("a\r\nb\r\n", "lf"), "a\nb\n");
    }

    #[test]
    fn encode_roundtrip_keeps_bom() {
        let bytes = encode("hi", "UTF-8", Bom::Utf8).unwrap();
        assert_eq!(&bytes[..3], &[0xEF, 0xBB, 0xBF]);
        assert_eq!(decode(&bytes, None).unwrap().text, "hi");
    }

    #[test]
    fn encode_rejects_unrepresentable() {
        // 用 windows-1252 而非 GBK 来验证这条红线：
        // encoding_rs 里 GBK 与 GB18030 共用索引，几乎能表示全部 Unicode 码位，
        // 拿它当"表示不了"的反例不可靠。windows-1252 覆盖范围有限，判定明确。
        let err = encode("中文", "windows-1252", Bom::None).unwrap_err();
        assert_eq!(err.code, ErrorCode::UnrepresentableEncoding);
    }

    #[test]
    fn encode_roundtrip_for_gbk() {
        let bytes = encode("中文测试", "GBK", Bom::None).unwrap();
        assert_eq!(bytes, vec![0xD6, 0xD0, 0xCE, 0xC4, 0xB2, 0xE2, 0xCA, 0xD4]);
        assert_eq!(decode(&bytes, None).unwrap().text, "中文测试");
    }
}
