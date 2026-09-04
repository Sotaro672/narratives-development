// frontend\src\components\ui\Copy.tsx
import "./copy.css";

type CopyProps = {
  type?: "button" | "submit" | "reset";
  onClick?: () => void | Promise<void>;
  ariaLabel?: string;
  title?: string;
  disabled?: boolean;
};

export default function Copy({
  type = "button",
  onClick,
  ariaLabel = "コピー",
  title = "コピー",
  disabled = false,
}: CopyProps) {
  return (
    <button
      type={type}
      className="copy-button"
      onClick={() => {
        void onClick?.();
      }}
      aria-label={ariaLabel}
      title={title}
      disabled={disabled}
    >
      <svg
        className="copy-button__icon"
        viewBox="0 0 24 24"
        aria-hidden="true"
        focusable="false"
      >
        <path
          d="M9 9a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7a2 2 0 0 1-2-2V9zm-4 6V6a2 2 0 0 1 2-2h7"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </button>
  );
}