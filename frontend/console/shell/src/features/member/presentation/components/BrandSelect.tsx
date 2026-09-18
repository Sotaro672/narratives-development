// frontend/console/member/src/presentation/components/BrandSelect.tsx

import * as React from "react";

import type { BrandRow } from "../hooks/useMemberCreate";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";
import { Button } from "../../../../shared/ui/button";
import { Checkbox } from "../../../../shared/ui/checkbox";
import { Badge } from "../../../../shared/ui/badge";
import { Label } from "../../../../shared/ui/label";

type BrandSelectProps = {
  brandRows: BrandRow[];
  selectedBrandIds: Set<string>;
  onToggleBrand: (id: string, explicit?: boolean) => void;
};

export function BrandSelect({
  brandRows,
  selectedBrandIds,
  onToggleBrand,
}: BrandSelectProps) {
  const selectedCount = React.useMemo(
    () => selectedBrandIds.size,
    [selectedBrandIds],
  );

  return (
    <div>
      <Label className="mb-1 block">
        ブランド（任意・複数選択可）
      </Label>

      <Popover>
        <PopoverTrigger>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start text-left"
          >
            {selectedCount > 0
              ? `選択中のブランド: ${selectedCount}件`
              : "ブランドを選択"}
          </Button>
        </PopoverTrigger>

        <PopoverContent className="w-[320px] popover__content--compact">
          {brandRows.length === 0 ? (
            <div className="popover__empty">
              現在、選択可能なブランドがありません。
            </div>
          ) : (
            <div className="popover__list">
              {brandRows.map((brand) => {
                const checked = selectedBrandIds.has(brand.id);
                const inputId = `brand_${brand.id}`;

                return (
                  <label
                    key={brand.id}
                    htmlFor={inputId}
                    className="popover__item flex items-center gap-2"
                  >
                    <Checkbox
                      id={inputId}
                      checked={checked}
                      onCheckedChange={(value) =>
                        onToggleBrand(
                          brand.id,
                          !!value,
                        )
                      }
                    />

                    <span>
                      {brand.name}
                    </span>
                  </label>
                );
              })}
            </div>
          )}
        </PopoverContent>
      </Popover>

      {/* 選択済みブランドのバッジ表示 */}
      <div className="mt-2 flex flex-wrap gap-2">
        {selectedCount === 0 ? (
          <span className="text-xs text-[hsl(var(--muted-foreground))]">
            選択したブランドがここに表示されます。
          </span>
        ) : (
          brandRows
            .filter((brand) =>
              selectedBrandIds.has(
                brand.id,
              ),
            )
            .map((brand) => (
              <Badge
                key={`brand_badge_${brand.id}`}
              >
                {brand.name}
              </Badge>
            ))
        )}
      </div>
    </div>
  );
}