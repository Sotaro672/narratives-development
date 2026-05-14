// frontend/console/productBlueprint/src/presentation/components/CategoryNumberField.tsx

import * as React from "react";
import { Input } from "../../../../shell/src/shared/ui/input";

type CategoryNumberFieldProps = {
  label: string;
  ariaLabel: string;
  value: string | number;
  suffix?: string;
  mode?: "edit" | "view";
  onChange?: (value: string) => void;
};

const CategoryNumberField: React.FC<CategoryNumberFieldProps> = ({
  label,
  ariaLabel,
  value,
  suffix,
  mode = "edit",
  onChange,
}) => {
  const isEdit = mode === "edit";

  return (
    <>
      <div className="label">{label}</div>
      <div className="flex gap-8 items-center">
        {isEdit ? (
          <>
            <Input
              type="number"
              value={value}
              onChange={(e) => onChange?.(e.target.value)}
              aria-label={ariaLabel}
            />
            {suffix && <span className="suffix">{suffix}</span>}
          </>
        ) : (
          <>
            <Input
              value={value ? `${value}` : ""}
              variant="readonly"
              readOnly
              aria-label={ariaLabel}
            />
            {suffix && <span className="suffix">{suffix}</span>}
          </>
        )}
      </div>
    </>
  );
};

export default CategoryNumberField;