// frontend/amol/src/features/list/presentation/components/ListPagination.tsx

type ListPaginationProps = {
  page: number;
  totalPages: number;
  canGoPrev: boolean;
  canGoNext: boolean;
  onPrev: () => void;
  onNext: () => void;
};

export default function ListPagination({
  page,
  totalPages,
  canGoPrev,
  canGoNext,
  onPrev,
  onNext,
}: ListPaginationProps) {
  if (totalPages <= 1) {
    return null;
  }

  return (
    <div
      className="lists-page-pagination"
      aria-label="ページ送り"
    >
      <button
        type="button"
        className="lists-page-pagination-button"
        disabled={!canGoPrev}
        onClick={onPrev}
      >
        前へ
      </button>

      <span className="lists-page-pagination-status">
        {page} / {totalPages}
      </span>

      <button
        type="button"
        className="lists-page-pagination-button"
        disabled={!canGoNext}
        onClick={onNext}
      >
        次へ
      </button>
    </div>
  );
}