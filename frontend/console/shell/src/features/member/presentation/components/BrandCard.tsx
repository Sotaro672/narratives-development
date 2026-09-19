// frontend/console/shell/src/features/member/presentation/components/BrandCard.tsx

import React from "react";

import type { BrandRow } from "../../../brand/application/brandService";
import {
  Badge,
  BadgeGroup,
} from "../../../../shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Text } from "../../../../shared/ui/text";

export function BrandCard({
  assignedBrands,
  brandRows,
}: {
  assignedBrands: string[];
  brandRows: BrandRow[];
}) {
  const brandMap = React.useMemo(() => {
    const map: Record<string, string> = {};

    for (const brand of brandRows) {
      map[brand.id] = brand.name;
    }

    return map;
  }, [brandRows]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>所属ブランド</CardTitle>
      </CardHeader>

      <CardContent>
        {assignedBrands.length === 0 ? (
          <Text as="p" tone="muted">
            所属ブランドは未設定です。
          </Text>
        ) : (
          <BadgeGroup>
            {assignedBrands.map((brandId) => (
              <Badge key={brandId}>
                {brandMap[brandId] ?? brandId}
              </Badge>
            ))}
          </BadgeGroup>
        )}
      </CardContent>
    </Card>
  );
}