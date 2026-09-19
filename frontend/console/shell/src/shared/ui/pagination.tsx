// frontend/console/shell/src/shared/ui/pagination.tsx

import { Button } from "./button";

import "./pagination.css";

export interface PaginationProps {
  /** 現在ページ (1始まり) */
  currentPage: number;
  /** 総ページ数 (1以上) */
  totalPages: number;
  /** ページ変更ハンドラ */
  onPageChange: (page: number) => void;
  /** ラベルのカスタマイズ（任意） */
  prevLabel?: string;
  nextLabel?: string;
  /** 追加クラス（任意） */
  className?: string;
}

/**
 * 共通ページネーションUI
 * - `totalPages <= 1` のときは描画しません（自動で非表示）
 * - ボタンは shared/ui/button を利用
 * - ページネーション固有スタイルは pagination.css で管理
 */
export default function Pagination({
  currentPage,
  totalPages,
  onPageChange,
  prevLabel = "前へ",
  nextLabel = "次へ",
  className,
}: PaginationProps) {
  if (!totalPages || totalPages <= 1) {
    return null;
  }

  const goPrev = () => onPageChange(Math.max(1, currentPage - 1));
  const goNext = () => onPageChange(Math.min(totalPages, currentPage + 1));

  return (
    <div className={`pagination${className ? ` ${className}` : ""}`}>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={goPrev}
        disabled={currentPage === 1}
        aria-label="前のページへ"
      >
        {prevLabel}
      </Button>

      <span className="pagination__page-info" aria-live="polite">
        {currentPage} / {totalPages} ページ
      </span>

      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={goNext}
        disabled={currentPage === totalPages}
        aria-label="次のページへ"
      >
        {nextLabel}
      </Button>
    </div>
  );
}