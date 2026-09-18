// frontend/console/shell/src/shared/ui/filterable-table-header.tsx

import React, { useEffect, useState } from "react";
import { Filter as FilterIcon } from "lucide-react";

import { Badge } from "./badge";
import { Checkbox } from "./checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "./popover";

type Option = {
  value: string;
  label?: string;
};

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
  const current = isControlled ? selected : internal;

  useEffect(() => {
    if (isControlled) {
      setInternal(selected ?? []);
    }
  }, [isControlled, selected]);

  const count = current.length;

  const toggle = (value: string, nextChecked: boolean) => {
    const next = nextChecked
      ? [...current, value]
      : current.filter((currentValue) => currentValue !== value);

    if (isControlled) {
      onChange?.(next);
    } else {
      setInternal(next);
    }
  };

  const clearAll = () => {
    if (isControlled) {
      onChange?.([]);
    } else {
      setInternal([]);
    }
  };

  return (
    <Popover>
      <PopoverTrigger>
        <button
          type="button"
          className={[
            "inline-flex items-center gap-2 rounded-xl border border-[#d9dde3] bg-[#eef1f4] px-3 py-1.5 font-bold leading-none text-[#0f172a]",
            className,
          ].filter(Boolean).join(" ")}
          title={`${label}で絞り込む`}
        >
          <span>{label}</span>
          <FilterIcon size={16} aria-hidden className="opacity-90" />

          {count > 0 && (
            <Badge
              variant="secondary"
              className="ml-1 rounded-full px-1.5 text-[10px]"
            >
              {count}
            </Badge>
          )}
        </button>
      </PopoverTrigger>

      <PopoverContent align="start" className="popover__content--compact popover__content--medium">
        <div className="popover__header">
          <div className="text-[13px] font-semibold">
            {dialogTitle ?? `${label}で絞り込み`}
          </div>

          {current.length > 0 && (
            <div className="popover__header-actions">
              <button
                type="button"
                onClick={clearAll}
                className="border-0 bg-transparent text-xs text-[hsl(var(--muted-foreground))] cursor-pointer"
              >
                クリア
              </button>
            </div>
          )}
        </div>

        {options.length === 0 ? (
          <div className="popover__empty">
            候補がありません
          </div>
        ) : (
          <div className="popover__list">
            {options.map((option) => {
              const id = `fth-${label}-${option.value}`;
              const checked = current.includes(option.value);

              return (
                <label
                  key={option.value}
                  htmlFor={id}
                  className="popover__item popover__item--control"
                >
                  <Checkbox
                    id={id}
                    checked={checked}
                    onCheckedChange={(nextChecked) =>
                      toggle(option.value, !!nextChecked)
                    }
                  />
                  <span className="text-sm">
                    {option.label ?? option.value}
                  </span>
                </label>
              );
            })}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}