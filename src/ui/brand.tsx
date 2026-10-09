/** 品牌符号：一滴朱墨落在纸上。与 outputs/brand/levnote-mark.svg 同源，按需缩放。 */

export function Mark({ size = 20, ink = '#16302A' }: { size?: number; ink?: string }) {
  return (
    <svg width={size} height={size} viewBox="296 290 432 544" role="img" aria-label="轻笺">
      <path
        d="M512 316 C596 420 702 502 702 618 A190 190 0 1 1 322 618 C322 502 428 420 512 316 Z"
        fill="#E2603F"
      />
      <g fill={ink === '#16302A' ? '#F7FAF8' : ink}>
        <rect x="428" y="528" width="168" height="22" rx="11" />
        <rect x="428" y="578" width="168" height="22" rx="11" />
        <rect x="450" y="628" width="124" height="22" rx="11" />
        <rect x="470" y="678" width="84" height="22" rx="11" />
      </g>
    </svg>
  );
}
