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

/**
 * 对应 Rust `fs::BinaryWriteOutcome` —— 写二进制文件的结果。
 *
 * 字段比 `WriteOutcome` 少得多，因为二进制写入**不做编码 / BOM / 行尾的协商**：
 * 那些概念只对文本成立。剩下的三个字段各自对应一个真实问题：
 *
 * - `path`：Rust 侧会做规范化（反斜杠、`..`），返回**实际落盘的那个路径**，
 *   调用方据此回写文档里的相对路径，而不是拿自己拼的字符串去猜；
 * - `size`：落盘字节数。调用方可以拿它和内存里的字节数比一次 ——
 *   不一致说明写坏了，这比「写完了但没人核对」多一道闸；
 * - `mtimeMs`：这个文件的落盘时间。**注意它不是并发检测的依据** ——
 *   二进制写入用的是「目标已存在就拒绝」（见 `write_binary_file`），
 *   与文本写入的 mtime 乐观锁不是同一套。留着它是为了让调用方能
 *   在状态栏或日志里说出「这张图是什么时候落盘的」。
 */
export interface BinaryWriteOutcome {
  path: string;
  size: number;
  mtimeMs: number;
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

/**
 * 对应 Rust `error::AppError` 的**线上形状**。
 *
 * 跨语言边界上它就是一个普通对象：Tauri 把命令返回的 `Err` 按 serde 序列化成它。
 * 所以这个类型描述的是「数据长什么样」，不是「JS 侧抛什么」—— 后者见下面的 `AppError`。
 */
export interface AppErrorShape {
  code: ErrorCode;
  message: string;
  detail?: string | null;
}

/**
 * JS 侧抛出的应用错误。**与 Rust 侧的 `AppError::new(code, message)` 对称。**
 *
 * 为什么不直接 `throw { code, message, detail }`：
 * 一个不是 `Error` 的抛出物很容易被漏掉 —— 任何 `instanceof Error` 判断、
 * 控制台里的堆栈、未捕获拒绝的上报都拿不到它。而 Rust 那边本来就有正经构造器，
 * 用裸对象字面量抛是**只有 JS 这一侧才有的不对称**。
 *
 * 继承 `Error` 的同时保持 `AppErrorShape` 的字段，于是：
 * - `asAppError()` 不用改（它按鸭子类型识别，两种形状都认）；
 * - 想 `instanceof AppError` 做精细分支的调用方也拿得到；
 * - 出问题时有堆栈，而不是只有一个 `code`。
 */
export class AppError extends Error implements AppErrorShape {
  readonly code: ErrorCode;
  readonly detail: string | null;

  constructor(code: ErrorCode, message: string, detail?: string | null) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.detail = detail ?? null;
  }
}

/**
 * 把任意抛出物收敛成 `AppErrorShape`，UI 里不必到处写 instanceof。
 *
 * 三种输入都要认，因为**它们都会真实出现**：
 * 1. 来自 Tauri 的普通对象（跨语言边界，serde 序列化的结果）；
 * 2. JS 侧抛出的 `AppError` 实例 —— 它是 `Error` 子类且带 `code`，走第 1 支；
 * 3. 意外抛出物：字符串、`null`、别的 `Error` —— 走兜底，统一成 INTERNAL。
 */
export function asAppError(thrown: unknown): AppErrorShape {
  if (typeof thrown === 'object' && thrown !== null && 'code' in thrown && 'message' in thrown) {
    const candidate = thrown as { code: unknown; message: unknown; detail?: unknown };
    return {
      code: (candidate.code as ErrorCode) ?? 'INTERNAL',
      message: String(candidate.message),
      // detail 的契约是「字符串或空」。上游多塞了个对象时一律收敛成 null，
      // 而不是 String() 成 '[object Object]' 再显示给用户。
      detail: typeof candidate.detail === 'string' ? candidate.detail : null,
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
  /**
   * 按字节写出一个二进制文件（粘贴图片时把剪贴板里的图落到磁盘上）。
   *
   * ## 为什么是「已存在就拒绝」而不是「覆盖」
   *
   * 文件名由内核生成（`image-<时间戳>-<盐><扩展名>`），撞名的概率极低；
   * 一旦真撞上，说明要么是同一秒里粘了同名文件，要么是磁盘上已经有一份
   * 我们没见过的内容。两种情况下**覆盖都是错的**：前者会丢掉刚粘的那张，
   * 后者会破坏用户已有的文件。所以这里选择明确失败，让调用方换一个盐重试，
   * 或者把冲突如实报给用户 —— 与项目「绝不静默覆盖」的硬约束一致。
   *
   * 平台不支持时抛错（浏览器预览没有二进制盘），调用方据此在界面上说明。
   */
  writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome>;
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
