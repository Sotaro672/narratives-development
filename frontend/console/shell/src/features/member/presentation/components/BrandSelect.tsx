// frontend/console/shell/src/features/member/presentation/components/BrandSelect.tsx

import * as React from "react";

import type { BrandRow } from "../hooks/useMemberCreate";
import {
  Badge,
  BadgeGroup,
} from "../../../../shared/ui/badge";
import { Button } from "../../../../shared/ui/button";
import { Checkbox } from "../../../../shared/ui/checkbox";
import { Label } from "../../../../shared/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";
import Stack from "../../../../shared/ui/stack";
import { Text } from "../../../../shared/ui/text";

import "../../../../styles/member.css";

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
    <Stack gap="sm">
      <Label>
        ブランド（任意・複数選択可）
      </Label>

      <Popover>
        <PopoverTrigger>
          <Button
            type="button"
            variant="outline"
            className="brand-select__trigger"
          >
            {selectedCount > 0
              ? `選択中のブランド: ${selectedCount}件`
              : "ブランドを選択"}
          </Button>
        </PopoverTrigger>

        <PopoverContent className="popover__content--compact popover__content--medium">
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
                    className="popover__item popover__item--control"
                  >
                    <Checkbox
                      id={inputId}
                      checked={checked}
                      onCheckedChange={(value) =>
                        onToggleBrand(brand.id, !!value)
                      }
                    />
                    <span>{brand.name}</span>
                  </label>
                );
              })}
            </div>
          )}
        </PopoverContent>
      </Popover>

      <BadgeGroup>
        {selectedCount === 0 ? (
          <Text
            as="span"
            size="xs"
            tone="muted"
          >
            選択したブランドがここに表示されます。
          </Text>
        ) : (
          brandRows
            .filter((brand) => selectedBrandIds.has(brand.id))
            .map((brand) => (
              <Badge key={`brand_badge_${brand.id}`}>
                {brand.name}
              </Badge>
            ))
        )}
      </BadgeGroup>
    </Stack>
  );
}