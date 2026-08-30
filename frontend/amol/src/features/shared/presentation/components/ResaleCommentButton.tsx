// frontend/amol/src/features/shared/presentation/components/ResaleCommentButton.tsx

import "../../styles/resale-comment-button.css";

type ResaleCommentButtonProps = {
  commentCount: number;
  onClick: () => void;
  disabled?: boolean;
};

export default function ResaleCommentButton({
  commentCount,
  onClick,
  disabled = false,
}: ResaleCommentButtonProps) {
  const safeCount = Math.max(0, Math.trunc(commentCount));
  const countLabel = safeCount > 99 ? "99+" : String(safeCount);

  return (
    <button
      type="button"
      className="resale-comment-button"
      onClick={onClick}
      disabled={disabled}
      aria-label={`コメントを見る、${safeCount}件`}
    >
      <svg
        className="resale-comment-button__icon"
        viewBox="0 0 24 24"
        fill="none"
        aria-hidden="true"
      >
        <path
          d="M21 11.5C21 16.1944 16.9706 20 12 20C10.8467 20 9.74407 19.7951 8.73026 19.4216L4 21L5.48949 16.8917C3.94002 15.4647 3 13.5663 3 11.5C3 6.80558 7.02944 3 12 3C16.9706 3 21 6.80558 21 11.5Z"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>

      <span className="resale-comment-button__count">
        {countLabel}
      </span>
    </button>
  );
}