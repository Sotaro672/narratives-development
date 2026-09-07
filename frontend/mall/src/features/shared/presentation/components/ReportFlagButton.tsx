// frontend/mall/src/features/shared/presentation/components/ReportFlagButton.tsx

import { Flag } from "lucide-react";

import "../../styles/report-flag-button.css";

type ReportFlagButtonProps = {
  disabled?: boolean;
  onClick: () => void | Promise<void>;
};

export default function ReportFlagButton({
  disabled = false,
  onClick,
}: ReportFlagButtonProps) {
  const label = "出品を通報";

  return (
    <button
      type="button"
      className="report-flag-button"
      disabled={disabled}
      aria-label={label}
      title={label}
      onClick={onClick}
    >
      <Flag
        size={28}
        strokeWidth={2}
        aria-hidden="true"
      />
    </button>
  );
}