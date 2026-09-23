// frontend/console/shell/src/features/member/presentation/components/BrandSelect.tsx

import * as React from "react";

import type { BrandRow } from "../hooks/useMemberCreate";
import {
  Badge,
  BadgeGroup,
} from "../../../../shared/ui/badge";
import { Select } from "../../../../shared/ui/select";
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

  const brandOptions = React.useMemo(
    () =>
      brandRows.map((brand) => ({
        value: brand.id,
        label: brand.name,
      })),
    [brandRows],
  );

  return (
    <Stack gap="sm">
      <Select
        multiple
        label="ブランド（任意・複数選択可）"
        options={brandOptions}
        value={selectedBrandIds}
        placeholder="ブランドを選択"
        emptyText="現在、選択可能なブランドがありません。"
        renderValue={() => `選択中のブランド: ${selectedCount}件`}
        onChange={(id, selected) => {
          onToggleBrand(id, selected);
        }}
      />

      <BadgeGroup>
        {selectedCount === 0 ? (
          <Text as="span" size="xs" tone="muted">
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