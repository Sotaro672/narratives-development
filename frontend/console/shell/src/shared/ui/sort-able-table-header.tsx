// frontend\console\shell\src\shared\ui\sort-able-table-header.tsx

import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { Button } from "./button";

import "./sort-able-table-header.css";

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
  onChange: (
    key: string,
    nextDirection: Exclude<SortDirection, null>,
  ) => void;
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
  direction,
  onChange,
  className,
}: SortableTableHeaderProps) {
  const [localDir, setLocalDir] = useState<SortDirection>(null);
  const isControlled = direction !== undefined;
  const isActive = activeKey === sortKey;

  const effectiveDir: SortDirection = isControlled ? direction! : localDir;

  useEffect(() => {
    if (!isActive && !isControlled) {
      setLocalDir(null);
    }
  }, [isActive, isControlled]);

  const icon = useMemo(() => {
    if (!isActive) {
      return <ChevronDown size={16} />;
    }

    return effectiveDir === "asc" ? (
      <ChevronUp size={16} />
    ) : (
      <ChevronDown size={16} />
    );
  }, [isActive, effectiveDir]);

  const handleClick = () => {
    const next: Exclude<SortDirection, null> = isActive
      ? effectiveDir === "asc"
        ? "desc"
        : "asc"
      : "asc";

    if (!isControlled) {
      setLocalDir(next);
    }

    onChange(sortKey, next);
  };

  const buttonClassName = [
    "sth-btn",
    isActive ? "sth-btn--active" : "",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      onClick={handleClick}
      className={buttonClassName}
      aria-pressed={isActive}
      aria-label={`${label} で並び替え`}
    >
      <span>{label}</span>
      <span className="sth-btn__icon" aria-hidden>
        {icon}
      </span>
    </Button>
  );
}