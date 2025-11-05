// frontend/shared/ui/sortable-table-header.tsx
import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

export type SortDirection = "asc" | "desc" | null;

export interface SortableTableHeaderProps {
  /** 見出しテキスト */
  label: string;
  /** この列の識別子 */
  sortKey: string;
  /** 現在ソート中のキー（親から） */
  activeKey?: string | null;
  /** 現在のソート方向（親から） */
  direction?: SortDirection;
  /** クリック時に親へ通知（キーと次の方向を返す） */
  onChange: (key: string, nextDirection: Exclude<SortDirection, null>) => void;

  /** 見た目調整（任意） */
  className?: string;
}

/**
 * ソート可能なテーブルヘッダー
 * - ホバーで矢印が表示
 * - クリックで asc ⇄ desc をトグル
 * - 状態は親が保持（制御コンポーネント）
 */
export default function SortableTableHeader({
  label,
  sortKey,
  activeKey = null,
  direction = null,
  onChange,
  className,
}: SortableTableHeaderProps) {
  const [hovered, setHovered] = useState(false);

  const isActive = activeKey === sortKey;
  const icon = useMemo(() => {
    if (!isActive) return <ChevronDown size={16} />;
    return direction === "asc" ? <ChevronUp size={16} /> : <ChevronDown size={16} />;
  }, [isActive, direction]);

  const handleClick = () => {
    // 未ソート → asc、asc → desc、desc → asc
    const next: Exclude<SortDirection, null> =
      isActive ? (direction === "asc" ? "desc" : "asc") : "asc";
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
        border: hovered || isActive ? "1px solid #d9dde3" : "1px solid transparent",
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

/* --- おまけ：数値ソートのための小ユーティリティ（必要なら使用） --- */

/** 文字列に含まれる通貨記号・カンマ・% を除去して数値化 */
export function toNumberLoose(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v !== "string") return Number(v);
  const s = v.replace(/[¥,\s%]/g, "");
  const n = Number(s);
  return Number.isNaN(n) ? 0 : n;
}

/** 数値列のソート用コンパレータ（asc/desc） */
export function numericComparator<T>(key: keyof T, dir: Exclude<SortDirection, null>) {
  return (a: T, b: T) => {
    const av = toNumberLoose((a as any)[key]);
    const bv = toNumberLoose((b as any)[key]);
    return dir === "asc" ? av - bv : bv - av;
  };
}
