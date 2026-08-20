// frontend/console/shell/src/features/list/presentation/components/ListStatusHeaderActions.tsx

import type { ListStatus } from "../../../../shared/types/list";

type ListStatusHeaderActionsProps = {
  status: ListStatus;
  onChange?: (status: ListStatus) => void;
  disabled?: boolean;
};

export default function ListStatusHeaderActions({
  status,
  onChange,
  disabled = false,
}: ListStatusHeaderActionsProps) {
  const handleChange = (next: ListStatus) => {
    if (disabled || !onChange || status === next) return;
    onChange(next);
  };

  return (
    <>
      <button
        type="button"
        className={[
          "page-header__btn",
          status === "listing" ? "page-header__btn--active" : "",
        ].filter(Boolean).join(" ")}
        onClick={() => handleChange("listing")}
        disabled={disabled}
        aria-pressed={status === "listing"}
      >
        出品
      </button>

      <button
        type="button"
        className={[
          "page-header__btn",
          status === "suspended" ? "page-header__btn--active" : "",
        ].filter(Boolean).join(" ")}
        onClick={() => handleChange("suspended")}
        disabled={disabled}
        aria-pressed={status === "suspended"}
      >
        保留
      </button>
    </>
  );
}