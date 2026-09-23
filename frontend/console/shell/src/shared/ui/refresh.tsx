// frontend/console/shell/src/shared/ui/refresh.tsx

import { RotateCw } from "lucide-react";

import { Button } from "./button";

import "./refresh.css";

type RefreshButtonProps = {
  /** クリック時ハンドラ */
  onClick?: () => void;
  /** ローディング中は回転表示し、ボタンを無効化 */
  loading?: boolean;
  /** 強制的に無効化したい場合 */
  disabled?: boolean;
  /** ツールチップ/タイトル */
  title?: string;
  /** アクセシビリティ用ラベル */
  ariaLabel?: string;
  /** 任意クラス */
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
    <Button
      type="button"
      variant="outline"
      size="icon"
      className={className}
      aria-label={ariaLabel}
      title={title}
      onClick={onClick}
      disabled={disabled || loading}
      aria-busy={loading}
    >
      <RotateCw
        size={size}
        className={loading ? "refresh-button__icon--loading" : ""}
        aria-hidden="true"
      />
    </Button>
  );
}