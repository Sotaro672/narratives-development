// frontend/shell/src/shared/ui/checkbox.tsx
import { Check } from "lucide-react";

interface CheckboxProps {
  id?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

export function Checkbox({ id, checked, onCheckedChange }: CheckboxProps) {
  return (
    <span
      role="checkbox"
      aria-checked={checked}
      aria-labelledby={id}
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === " " || e.key === "Enter") {
          e.preventDefault();
          onCheckedChange(!checked);
        }
      }}
      onClick={() => onCheckedChange(!checked)}
      style={{
        display: "inline-flex",
        width: 18,
        height: 18,
        borderRadius: 4,
        border: "1px solid #cbd5e1",
        background: checked ? "#111827" : "#fff",
        alignItems: "center",
        justifyContent: "center",
        cursor: "pointer",
      }}
    >
      {checked && <Check size={12} color="#fff" aria-hidden />}
    </span>
  );
}
