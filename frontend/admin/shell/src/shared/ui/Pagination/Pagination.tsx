// frontend/admin/shell/src/shared/ui/Pagination/Pagination.tsx

import Button from "../Button/Button";

import "./Pagination.css";

export type PaginationProps = {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
  ariaLabel?: string;
  className?: string;
};

export default function Pagination({
  page,
  totalPages,
  onPageChange,
  disabled = false,
  ariaLabel = "ページ送り",
  className = "",
}: PaginationProps) {
  if (totalPages <= 1) {
    return null;
  }

  const hasPreviousPage = page > 1;
  const hasNextPage = page < totalPages;

  return (
    <nav
      className={["ui-pagination", className].filter(Boolean).join(" ")}
      aria-label={ariaLabel}
    >
      <Button
        size="sm"
        variant="secondary"
        disabled={disabled || !hasPreviousPage}
        onClick={() => onPageChange(page - 1)}
      >
        前へ
      </Button>

      <span className="ui-pagination__label" aria-live="polite">
        {page} / {totalPages}
      </span>

      <Button
        size="sm"
        variant="secondary"
        disabled={disabled || !hasNextPage}
        onClick={() => onPageChange(page + 1)}
      >
        次へ
      </Button>
    </nav>
  );
}