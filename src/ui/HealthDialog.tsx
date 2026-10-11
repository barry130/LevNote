/**
 * 问题清单（ADR 0046）。
 *
 * ## 它是「横幅」的反面
 *
 * 横幅回答「**此刻**有没有问题」，按掉就没了 —— 而按掉的动机恰恰是「我不想现在看」，
 * 于是这一次降级就此从用户的视野里消失第二次。这一页回答「**这次运行发生过什么**」，
 * 横幅按掉不影响它，关掉应用才清空。
 *
 * ## 三条它刻意做的事
 *
 * 1. **「已经恢复」的记录不删**。「后来好了」是答案的一部分；自动收起来等于
 *    又一次静默（而且是最难看的那种：用户连「有过这件事」都不知道）。
 * 2. **按族分组，不按时间倒序铺平**。同一个故障反复出现时，倒序会让同一条散成一片；
 *    分组之后「存储 / 会话 / 监听 / 内核」四个格子各自自洽，一眼能看出是哪一族出的问题。
 * 3. **能复制走**。这是全应用唯一能把内部状态交给别人的口子 —— 没有账号、没有网络、
 *    没有遥测（ADR 0001），要报一个 bug 只能靠这份文本。剪贴板失败要**说出来**
 *    而不是当没发生（`core.cmd.clipCopyDenied` 记着同一件事：失败要有一张脸）。
 */

import { useEffect, useMemo, useRef, useState } from 'react';

import {
  activeHealth,
  type HealthArea,
  type HealthRecord,
} from '../core/health.js';
import { t } from '../core/i18n/index.js';
import type { MessageKey } from '../core/i18n/messages.js';

/** 分组顺序按「数据先于界面」排：丢东西的两族在最上面，看不见的两族在下面。 */
const AREA_ORDER: readonly HealthArea[] = ['storage', 'session', 'watch', 'kernel'];

const AREA_LABEL_KEYS = {
  storage: 'ui.health.area.storage',
  session: 'ui.health.area.session',
  watch: 'ui.health.area.watch',
  kernel: 'ui.health.area.kernel',
} as const satisfies Readonly<Record<HealthArea, MessageKey>>;

export interface HealthDialogProps {
  log: readonly HealthRecord[];
  onClear: () => void;
  onClose: () => void;
}

type CopyState = 'copied' | 'failed' | null;

function clockOf(at: number): string {
  return new Date(at).toLocaleTimeString();
}

/** 复制出去的纯文本：给「我把这个贴给开发者」用，所以要自足（带时刻、带分组名）。 */
function reportText(log: readonly HealthRecord[]): string {
  const head = `${t('ui.app.appName')} · ${t('ui.health.title')} · ${new Date().toLocaleString()}`;
  const body = AREA_ORDER.flatMap((area) => {
    const records = log.filter((record) => record.area === area);
    if (records.length === 0) return [];
    return [
      `【${t(AREA_LABEL_KEYS[area])}】`,
      ...records.map((record) => {
        const [first, ...rest] = record.lines;
        const lines = (first === undefined ? [] : [t(first.key, first.values), ...rest.map((line) => `  ${t(line.key, line.values)}`)]);
        const meta = [
          clockOf(record.lastSeenAt),
          // 「发生过 n 次」只在 n > 1 时出现：一条永远为真的信息就是噪音，
          // 而噪音会让人把「这次是 3 次」也一起划过去。
          ...(record.count > 1 ? [t('ui.health.occurrences', { n: record.count })] : []),
          t(record.resolvedAt === null ? 'ui.health.active' : 'ui.health.resolved'),
        ];
        return `- ${lines.join('\n  ')}\n  （${meta.join(' · ')}）`;
      }),
    ];
  });
  return [head, ...body].join('\n');
}

export function HealthDialog({ log, onClear, onClose }: HealthDialogProps) {
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const [copyState, setCopyState] = useState<CopyState>(null);

  useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  /** Esc 关闭。捕获阶段 + stopPropagation：与设置页同一条理由（硬约束 9）。 */
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      onClose();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [onClose]);

  /**
   * 按族分组。`log` 本身已经是「最近发生的在前」，同一族内保持那个次序。
   *
   * **只为已有记录建组**：`HealthArea` 是闭合联合，`AREA_LABEL_KEYS` 用
   * `satisfies Record<HealthArea, …>` 钉住了每一种 —— 将来加了新的一族而忘了写标签名，
   * 编译期就会拦下，而不是运行时悄悄掉进一个「其它」组里。
   */
  const groups = useMemo(
    () =>
      AREA_ORDER.map((area) => ({
        area,
        records: log.filter((record) => record.area === area),
      })).filter((group) => group.records.length > 0),
    [log],
  );

  const active = activeHealth(log).length;

  const copyAll = () => {
    // 清单为空时没有可复制的东西，按钮也不该亮着 —— 一个「复制了空字符串」
    // 的按钮只会让人以为复制失败了。
    if (log.length === 0) return;
    void navigator.clipboard
      .writeText(reportText(log))
      .then(() => setCopyState('copied'))
      .catch(() => setCopyState('failed'));
  };

  return (
    <div
      className="palette-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="settings" role="dialog" aria-label={t('ui.health.title')} tabIndex={-1} ref={dialogRef}>
        <header className="settings-head">
          <h2>{t('ui.health.title')}</h2>
          <span className="grow" />
          <button type="button" onClick={onClose}>
            {t('ui.settings.close')}
          </button>
        </header>

        <div className="settings-body">
          <section className="settings-section">
            <p className="settings-hint">{t('ui.health.intro')}</p>
          </section>

          {log.length === 0 ? (
            <section className="settings-section">
              <p className="health-empty">{t('ui.health.empty')}</p>
            </section>
          ) : (
            groups.map((group) => (
              <section className="settings-section" key={group.area}>
                <h3>{t(AREA_LABEL_KEYS[group.area])}</h3>
                {group.records.map((record) => {
                  const [first, ...rest] = record.lines;
                  return (
                    <article
                      key={record.id}
                      className={`health-record ${record.severity === 'error' ? 'error' : 'warn'} ${record.resolvedAt === null ? '' : 'resolved'}`}
                      data-testid={`health-record-${record.id}`}
                    >
                      <h4 className="health-record-title">{first === undefined ? record.id : t(first.key, first.values)}</h4>
                      {rest.map((line, index) => (
                        <p className="settings-hint" key={`${line.key}-${index}`}>
                          {t(line.key, line.values)}
                        </p>
                      ))}
                      <p className="health-meta">
                        <span>{clockOf(record.lastSeenAt)}</span>
                        {record.count > 1 && <span>{t('ui.health.occurrences', { n: record.count })}</span>}
                        <span>{t(record.resolvedAt === null ? 'ui.health.active' : 'ui.health.resolved')}</span>
                      </p>
                    </article>
                  );
                })}
              </section>
            ))
          )}

          <section className="settings-section">
            <div className="settings-row">
              <span className="settings-hint">
                {t('ui.health.activeCount', { n: active })}
              </span>
              <span className="grow" />
              <button type="button" disabled={log.length === 0} onClick={copyAll}>
                {t('ui.health.copyAll')}
              </button>
              <button
                type="button"
                disabled={log.length === 0}
                onClick={() => {
                  setCopyState(null);
                  onClear();
                }}
              >
                {t('ui.health.clear')}
              </button>
            </div>
            {copyState !== null && (
              <p className="settings-hint">{t(copyState === 'copied' ? 'ui.health.copied' : 'ui.health.copyFailed')}</p>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}