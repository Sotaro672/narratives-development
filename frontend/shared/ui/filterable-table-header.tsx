// frontend/shared/ui/filterable-table-header.tsx
import React, { useEffect, useState } from "react";
import { Filter as FilterIcon } from "lucide-react";
import { Popover, PopoverTrigger, PopoverContent } from "./popover";
import { Checkbox } from "./checkbox";
import { Badge } from "./badge";

type Option = { value: string; label?: string };

interface FilterableTableHeaderProps {
  label: string;
  options: Option[];
  selected?: string[];
  onChange?: (next: string[]) => void;
  dialogTitle?: string;
  className?: string;
}

export default function FilterableTableHeader({
  label,
  options,
  selected,
  onChange,
  dialogTitle,
  className = "",
}: FilterableTableHeaderProps) {
  const [internal, setInternal] = useState<string[]>(selected ?? []);
  const isControlled = selected !== undefined;
  const current = isControlled ? selected! : internal;

  useEffect(() => {
    if (isControlled) setInternal(selected!);
  }, [isControlled, selected]);

  const count = current.length;

  const toggle = (val: string, nextChecked: boolean) => {
    const next = nextChecked
      ? [...current, val]
      : current.filter((v) => v !== val);
    if (isControlled) onChange?.(next);
    else setInternal(next);
  };

  const clearAll = () => {
    if (isControlled) onChange?.([]);
    else setInternal([]);
  };

  // ── 見た目：スクショ準拠の“薄グレー角丸ピル＋漏斗アイコン” ──
  const triggerStyle: React.CSSProperties = {
    display: "inline-flex",
    alignItems: "center",
    gap: 8,
    padding: "6px 12px",
    borderRadius: 12,
    background: "#eef1f4",
    border: "1px solid #d9dde3",
    color: "#0f172a",
    fontWeight: 700,
    lineHeight: 1,
    cursor: "pointer",
  };

  return (
    <Popover>
      <PopoverTrigger>
        <button
          type="button"
          className={className}
          style={triggerStyle}
          title={`${label}で絞り込む`}
        >
          <span>{label}</span>
          <FilterIcon size={16} aria-hidden style={{ opacity: 0.9 }} />
          {count > 0 && (
            <Badge
              variant="secondary"
              className="rounded-full"
              style={{ marginLeft: 4, padding: "0 6px", fontSize: 10 }}
            >
              {count}
            </Badge>
          )}
        </button>
      </PopoverTrigger>

      <PopoverContent align="start">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 8,
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600 }}>
            {dialogTitle ?? `${label}で絞り込み`}
          </div>
          {current.length > 0 && (
            <button
              type="button"
              onClick={clearAll}
              style={{
                fontSize: 12,
                color: "#6b7280",
                background: "transparent",
                border: "none",
                cursor: "pointer",
              }}
            >
              クリア
            </button>
          )}
        </div>

        <div style={{ display: "grid", gap: 10 }}>
          {options.length === 0 && (
            <div style={{ fontSize: 12, color: "#6b7280" }}>候補がありません</div>
          )}

          {options.map((opt) => {
            const id = `fth-${label}-${opt.value}`;
            const checked = current.includes(opt.value);
            return (
              <label
                key={opt.value}
                htmlFor={id}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  cursor: "pointer",
                }}
              >
                <Checkbox
                  id={id}
                  checked={checked}
                  onCheckedChange={(nextChecked) =>
                    toggle(opt.value, nextChecked)
                  }
                />
                <span id={id} style={{ fontSize: 14 }}>
                  {opt.label ?? opt.value}
                </span>
              </label>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}
