// frontend/console/member/src/presentation/components/BrandCard.tsx

import React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import type { BrandRow } from "../../../../brand/src/application/brandService";

export function BrandCard({
  assignedBrands,
  brandRows,
}: {
  assignedBrands: string[];
  brandRows: BrandRow[];
}) {
  // brandId -> brandName のマップを作る
  const brandMap = React.useMemo(() => {
    const map: Record<string, string> = {};
    for (const b of brandRows) {
      map[b.id] = b.name;
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
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            所属ブランドは未設定です。
          </p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {assignedBrands.map((brandId) => (
              <Badge key={brandId}>
                {brandMap[brandId] ?? brandId /* fallback */}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
