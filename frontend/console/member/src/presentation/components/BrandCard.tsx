// frontend/console/member/src/presentation/components/BrandCard.tsx

import * as React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Badge } from "../../../../shell/src/shared/ui/badge";

type BrandCardProps = {
  assignedBrands: string[];
};

export function BrandCard({ assignedBrands }: BrandCardProps) {
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
              <Badge key={brandId}>{brandId}</Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
