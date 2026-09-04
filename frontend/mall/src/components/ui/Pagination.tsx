// frontend/amol/src/components/ui/Pagination.tsx

import "./pagination.css";

export type PaginationProps = {
  page: number;
  totalPages: number;
  canGoPrev: boolean;
  canGoNext: boolean;
  onPrev: () => void;
  onNext: () => void;
  previousLabel?: string;
  nextLabel?: string;
  ariaLabel?: string;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function Pagination({
  page,
  totalPages,
  canGoPrev,
  canGoNext,
  onPrev,
  onNext,
  previousLabel = "前へ",
  nextLabel = "次へ",
  ariaLabel = "ページ送り",
  className,
}: PaginationProps) {
  if (totalPages <= 1) {
    return null;
  }

  return (
    <nav className={joinClassNames("pagination", className)} aria-label={ariaLabel}>
      <button
        type="button"
        className="pagination__button"
        disabled={!canGoPrev}
        onClick={onPrev}
      >
        {previousLabel}
      </button>

      <span className="pagination__status" aria-current="page">
        {page} / {totalPages}
      </span>

      <button
        type="button"
        className="pagination__button"
        disabled={!canGoNext}
        onClick={onNext}
      >
        {nextLabel}
      </button>
    </nav>
  );
}