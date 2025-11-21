// frontend/console/shell/src/shared/ui/sortable-table-header.tsx
import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

export type SortDirection = "asc" | "desc" | null;

export interface SortableTableHeaderProps {
  /** 見出しテキスト */
  label: string;
  /** この列の識別子 */
  sortKey: string;
  /** 現在ソート中のキー（親から） */
  activeKey?: string | null;
  /**
   * 現在のソート方向（親から）。
   * 省略された場合は「非制御モード」としてローカルで方向を保持します。
   */
  direction?: SortDirection;
  /** クリック時に親へ通知（キーと次の方向を返す） */
  onChange: (key: string, nextDirection: Exclude<SortDirection, null>) => void;

  className?: string;
}

/**
 * ソート可能なテーブルヘッダー
 * - ホバーで矢印が表示
 * - クリックで asc ⇄ desc をトグル
 * - direction 未指定時は内部状態でトグル（制御/非制御の両対応）
 */
export default function SortableTableHeader({
  label,
  sortKey,
  activeKey = null,
  direction, // ← undefined を許容（未指定なら非制御）
  onChange,
  className,
}: SortableTableHeaderProps) {
  const [hovered, setHovered] = useState(false);

  // 非制御用のローカル方向
  const [localDir, setLocalDir] = useState<SortDirection>(null);
  const isControlled = direction !== undefined;

  const isActive = activeKey === sortKey;

  // 実際に表示・判定に使う方向（制御なら props、非制御なら local）
  const effectiveDir: SortDirection = isControlled ? direction! : localDir;

  // activeKey が他列に移ったらローカル方向はリセット
  useEffect(() => {
    if (!isActive && !isControlled) setLocalDir(null);
  }, [isActive, isControlled]);

  const icon = useMemo(() => {
    if (!isActive) return <ChevronDown size={16} />;
    return effectiveDir === "asc" ? (
      <ChevronUp size={16} />
    ) : (
      <ChevronDown size={16} />
    );
  }, [isActive, effectiveDir]);

  const handleClick = () => {
    // 未ソート → asc、asc → desc、desc → asc
    const next: Exclude<SortDirection, null> =
      isActive ? (effectiveDir === "asc" ? "desc" : "asc") : "asc";

    // 非制御時は内部状態を更新
    if (!isControlled) setLocalDir(next);

    // 親にも通知（制御時はこちらを受けて方向を更新してください）
    onChange(sortKey, next);
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      className={`sth-btn ${className || ""}`}
      aria-pressed={isActive}
      aria-label={`${label} で並び替え`}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 8,
        padding: "8px 12px",
        borderRadius: 10,
        lineHeight: 1,
        fontWeight: 700,
        border:
          hovered || isActive ? "1px solid #d9dde3" : "1px solid transparent",
        background: hovered || isActive ? "#eef1f4" : "transparent",
        color: "#0f172a",
        cursor: "pointer",
        transition: "background 120ms ease, border-color 120ms ease",
      }}
    >
      <span>{label}</span>
      <span
        style={{
          opacity: hovered || isActive ? 1 : 0.3,
          display: "inline-flex",
          alignItems: "center",
        }}
        aria-hidden
      >
        {icon}
      </span>
    </button>
  );
}

/* --- おまけユーティリティはそのまま --- */
export function toNumberLoose(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v !== "string") return Number(v);
  const s = v.replace(/[¥,\s%]/g, "");
  const n = Number(s);
  return Number.isNaN(n) ? 0 : n;
}

export function numericComparator<T>(key: keyof T, dir: Exclude<SortDirection, null>) {
  return (a: T, b: T) => {
    const av = toNumberLoose((a as any)[key]);
    const bv = toNumberLoose((b as any)[key]);
    return dir === "asc" ? av - bv : bv - av;
  };
}
