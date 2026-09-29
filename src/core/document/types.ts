/**
 * 文档抽象层 —— 契约类型。
 *
 * 这里定义的每个类型都与 `src-tauri/src/fs/mod.rs` 里的 Rust 结构一一对应
 * （serde 用 camelCase 序列化）。**改一侧必须改另一侧**，没有编译期校验兜底，
 * 所以字段名的注释里标了对应的 Rust 类型名，方便互相检索。
 */

/** 对应 Rust `codec::Bom` 的字符串形式。 */
export type BomKind = 'none' | 'utf-8' | 'utf-16le' | 'utf-16be';

/** 对应 Rust `codec::detect_line_ending`；`mixed` 表示混排。 */
export type LineEndingName = 'lf' | 'crlf' | 'cr' | 'mixed';

/** 保存时可指定的目标行尾。 */
export type TargetLineEnding = 'lf' | 'crlf' | 'cr';

/** 对应 Rust `fs::FileStat` */
export interface FileStat {
  path: string;
  size: number;
  mtimeMs: number;
  readonly: boolean;
  isDir: boolean;
}

/** 对应 Rust `fs::TextFile` */
export interface TextFile {
  path: string;
  text: string;
  /** 规范化编码名，如 `UTF-8` / `GBK` */
  encoding: string;
  bom: BomKind;
  lineEnding: LineEndingName;
  size: number;
  mtimeMs: number;
  readonly: boolean;
  /** 解码时出现过非法字节，说明自动探测可能猜错了编码 */
  lossy: boolean;
}

/** 对应 Rust `fs::WriteRequest` */
export interface WriteRequest {
  path: string;
  text: string;
  encoding?: string;
  bom?: BomKind;
  /** 传入则把文本中的换行统一成这个风格 */
  lineEnding?: TargetLineEnding;
  /**
   * 传入则做乐观并发检测：磁盘 mtime 与之不符就拒绝写入。
   * 这是「绝不静默覆盖别人的改动」的实现方式。
   */
  expectedMtimeMs?: number;
  create?: boolean;
}

/** 对应 Rust `fs::WriteOutcome` */
export interface WriteOutcome {
  path: string;
  size: number;
  mtimeMs: number;
  encoding: string;
  bom: BomKind;
  lineEnding: LineEndingName;
}

/** 对应 Rust `fs::DirEntry` */
export interface DirEntry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  mtimeMs: number;
}

export interface RuntimeInfo {
  name: string;
  version: string;
  os: string;
  arch: string;
  nowMs: number;
}

/** 对应 Rust `error::ErrorCode` */
export type ErrorCode =
  | 'NOT_FOUND'
  | 'PERMISSION_DENIED'
  | 'IS_DIRECTORY'
  | 'CONFLICT'
  | 'UNREPRESENTABLE_ENCODING'
  | 'UNKNOWN_ENCODING'
  | 'INVALID_PATH'
  | 'NOT_SUPPORTED'
  | 'IO'
  | 'INTERNAL';

/** 对应 Rust `error::AppError`。Tauri 会把命令返回的 Err 原样序列化到这里。 */
export interface AppError {
  code: ErrorCode;
  message: string;
  detail?: string | null;
}

/** 把任意抛出物收敛成 AppError，UI 里不必到处写 instanceof。 */
export function asAppError(thrown: unknown): AppError {
  if (typeof thrown === 'object' && thrown !== null && 'code' in thrown && 'message' in thrown) {
    const candidate = thrown as { code: unknown; message: unknown; detail?: unknown };
    return {
      code: (candidate.code as ErrorCode) ?? 'INTERNAL',
      message: String(candidate.message),
      detail: candidate.detail === undefined || candidate.detail === null ? null : String(candidate.detail),
    };
  }
  if (thrown instanceof Error) return { code: 'INTERNAL', message: thrown.message, detail: null };
  return { code: 'INTERNAL', message: String(thrown), detail: null };
}

/**
 * 文档来源。桌面端是真实文件系统，浏览器端是内存虚拟盘
 * （仅用于纯 `vite dev` 下做界面开发，不是产品形态 —— 本项目只出桌面版）。
 */
export type ProviderKind = 'desktop' | 'browser';

/**
 * 唯一有平台差异的接口。编辑器内核只依赖它，不知道自己在桌面还是浏览器里跑。
 */
export interface DocumentProvider {
  readonly kind: ProviderKind;
  readTextFile(path: string, forcedEncoding?: string): Promise<TextFile>;
  writeTextFile(request: WriteRequest): Promise<WriteOutcome>;
  statFile(path: string): Promise<FileStat>;
  listDir(path: string): Promise<DirEntry[]>;
  sniffEncoding(path: string): Promise<string>;
  /**
   * 按字节读取二进制文件（自包含 HTML 导出时内联图片用）。
   *
   * 必须**逐字节原样**返回：走 `readTextFile` 会被编码探测与解码毁掉，
   * 一张 PNG 经过一次 UTF-8 解码再编码就不是原来那张图了。
   * 平台不支持时抛错，调用方退化为「不内联、标注出来」而不是假装成功。
   */
  readBinaryFile(path: string): Promise<ArrayBuffer>;
  runtimeInfo(): Promise<RuntimeInfo>;
  /** 弹「打开」对话框。用户取消时返回 null（不抛错）。 */
  pickOpenPath(): Promise<string | null>;
  /** 弹「另存为」对话框。用户取消时返回 null。 */
  pickSavePath(suggestedName: string): Promise<string | null>;
  /**
   * 弹「打开文件夹」对话框。用户取消时返回 null。
   *
   * 与 `pickOpenPath` 分开而不是加一个 `directory: boolean` 参数：
   * 两者的返回语义一样（取消 → null），但「选文件」与「选目录」在
   * 平台对话框里是两套完全不同的配置，混在一个方法里只会让调用方多传一个
   * 恒定的布尔值。分开后调用点读起来是自解释的。
   */
  pickDirectoryPath(): Promise<string | null>;
  /** 弹一个提示框（用于导出完成之类的通报）。 */
  notify(message: string, title?: string): Promise<void>;
}
