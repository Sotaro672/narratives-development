// frontend/console/shell/src/features/list/presentation/components/ListStatusHeaderActions.tsx

import { Button } from "../../../../shared/ui/button";
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
      <Button
        variant={status === "listing" ? "default" : "outline"}
        size="sm"
        onClick={() => handleChange("listing")}
        disabled={disabled}
        aria-pressed={status === "listing"}
      >
        出品
      </Button>

      <Button
        variant={status === "suspended" ? "default" : "outline"}
        size="sm"
        onClick={() => handleChange("suspended")}
        disabled={disabled}
        aria-pressed={status === "suspended"}
      >
        保留
      </Button>
    </>
  );
}