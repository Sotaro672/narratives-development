import React from "react";
import { RotateCw } from "lucide-react";

type RefreshButtonProps = {
  /** クリック時ハンドラ */
  onClick?: () => void;
  /** ローディング中はスピン（.lp-rotate を想定）＆ボタン無効化 */
  loading?: boolean;
  /** 強制的に無効化したい場合 */
  disabled?: boolean;
  /** ツールチップ/タイトル */
  title?: string;
  /** アクセシビリティ用ラベル */
  ariaLabel?: string;
  /** 既存の .lp-icon-btn を前提に任意クラスを追加可能 */
  className?: string;
  /** アイコンサイズ（既定 18） */
  size?: number;
};

export default function RefreshButton({
  onClick,
  loading = false,
  disabled = false,
  title = "リフレッシュ",
  ariaLabel = "リフレッシュ",
  className = "",
  size = 18,
}: RefreshButtonProps) {
  return (
    <button
      type="button"
      className={`lp-icon-btn ${className}`}
      aria-label={ariaLabel}
      title={title}
      onClick={onClick}
      disabled={disabled || loading}
    >
      <RotateCw
        size={size}
        className={loading ? "lp-rotate" : ""}
        aria-hidden
      />
    </button>
  );
}
