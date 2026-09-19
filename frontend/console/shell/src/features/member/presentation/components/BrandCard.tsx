// frontend/console/member/src/presentation/components/BrandCard.tsx

import React from "react";

import type { BrandRow } from "../../../brand/application/brandService";
import { Badge } from "../../../../shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import "../../../../shell/src/styles/member.css";

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
          <p className="brand-card__message">
            所属ブランドは未設定です。
          </p>
        ) : (
          <div className="brand-card__badges">
            {assignedBrands.map((brandId) => (
              <Badge key={brandId}>
                {brandMap[brandId] ?? brandId}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}