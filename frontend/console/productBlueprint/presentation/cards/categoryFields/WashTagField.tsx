// frontend/console/productBlueprint/src/presentation/components/WashTagField.tsx

import * as React from "react";
import { ShieldCheck, X } from "lucide-react";
import { Badge } from "../../../../../shell/src/shared/ui/badge";
import { Button } from "../../../../../shell/src/shared/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shell/src/shared/ui/popover";
import { Checkbox } from "../../../../../shell/src/shared/ui/checkbox";

import {
  WASH_TAG_OPTIONS,
  type WashTagOption,
} from "../../../domain/apparel";

type WashTagFieldProps = {
  value: string[];
  mode?: "edit" | "view";
  onChange?: (nextTags: string[]) => void;
};

const WashTagField: React.FC<WashTagFieldProps> = ({
  value,
  mode = "edit",
  onChange,
}) => {
  const isEdit = mode === "edit";
  const safeValue = Array.isArray(value) ? value : [];

  const washTagGroups = React.useMemo(() => {
    const map = new Map<string, WashTagOption[]>();

    for (const option of WASH_TAG_OPTIONS) {
      const category = option.category;
      const list = map.get(category) ?? [];
      list.push(option);
      map.set(category, list);
    }

    return Array.from(map.entries());
  }, []);

  const handleToggle = React.useCallback(
    (tagValue: string) => {
      if (!onChange) return;

      if (safeValue.includes(tagValue)) {
        onChange(safeValue.filter((tag) => tag !== tagValue));
      } else {
        onChange([...safeValue, tagValue]);
      }
    },
    [onChange, safeValue],
  );

  return (
    <>
      <div className="label">品質保証（洗濯方法タグ）</div>
      <div className="chips flex flex-wrap gap-2">
        {safeValue.map((tag) => (
          <Badge
            key={tag}
            className="chip inline-flex items-center gap-1.5 px-2 py-1"
          >
            <ShieldCheck size={14} />
            {tag}
            {isEdit && onChange && (
              <button
                onClick={() => onChange(safeValue.filter((x) => x !== tag))}
                className="chip-remove"
                aria-label={`${tag} を削除`}
              >
                <X size={12} />
              </button>
            )}
          </Badge>
        ))}
      </div>

      {isEdit && onChange && (
        <div className="mt-2 flex flex-wrap gap-2">
          {washTagGroups.map(([category, options]) => (
            <Popover key={category}>
              <PopoverTrigger>
                <Button
                  variant="secondary"
                  size="sm"
                  className="btn"
                  aria-label={`${category} のタグを追加`}
                >
                  {category}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" className="p-2 space-y-1 w-64">
                {options.map((option) => {
                  const checked = safeValue.includes(option.value);
                  const checkboxId = `wash-tag-${option.value}`;

                  return (
                    <label
                      key={option.value}
                      htmlFor={checkboxId}
                      className="flex items-center gap-2 text-sm cursor-pointer py-0.5"
                    >
                      <Checkbox
                        id={checkboxId}
                        checked={checked}
                        onCheckedChange={() => handleToggle(option.value)}
                      />
                      <span>{option.label}</span>
                    </label>
                  );
                })}
              </PopoverContent>
            </Popover>
          ))}
        </div>
      )}
    </>
  );
};

export default WashTagField;