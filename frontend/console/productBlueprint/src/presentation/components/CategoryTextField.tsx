// frontend/console/productBlueprint/src/presentation/components/CategoryTextField.tsx

import * as React from "react";
import { Input } from "../../../../shell/src/shared/ui/input";

type CategoryTextFieldProps = {
  label: string;
  ariaLabel: string;
  value: string | number;
  mode?: "edit" | "view";
  onChange?: (value: string) => void;
};

const CategoryTextField: React.FC<CategoryTextFieldProps> = ({
  label,
  ariaLabel,
  value,
  mode = "edit",
  onChange,
}) => {
  const isEdit = mode === "edit";

  return (
    <>
      <div className="label">{label}</div>
      {isEdit ? (
        <Input
          value={value}
          onChange={(e) => onChange?.(e.target.value)}
          aria-label={ariaLabel}
        />
      ) : (
        <Input
          value={value}
          variant="readonly"
          readOnly
          aria-label={ariaLabel}
        />
      )}
    </>
  );
};

export default CategoryTextField;