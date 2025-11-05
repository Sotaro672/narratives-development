// frontend/shared/ui/pagination.tsx

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
 * - スタイルは List.css と整合するクラス名（list-pagination / lp-btn / lp-page-info）を使用
 */
export default function Pagination({
  currentPage,
  totalPages,
  onPageChange,
  prevLabel = "前へ",
  nextLabel = "次へ",
  className,
}: PaginationProps) {
  if (!totalPages || totalPages <= 1) return null;

  const goPrev = () => onPageChange(Math.max(1, currentPage - 1));
  const goNext = () => onPageChange(Math.min(totalPages, currentPage + 1));

  return (
    <div className={`list-pagination${className ? ` ${className}` : ""}`}>
      <button
        type="button"
        className="lp-btn"
        onClick={goPrev}
        disabled={currentPage === 1}
        aria-label="前のページへ"
      >
        {prevLabel}
      </button>

      <span className="lp-page-info" aria-live="polite">
        {currentPage} / {totalPages} ページ
      </span>

      <button
        type="button"
        className="lp-btn"
        onClick={goNext}
        disabled={currentPage === totalPages}
        aria-label="次のページへ"
      >
        {nextLabel}
      </button>
    </div>
  );
}
