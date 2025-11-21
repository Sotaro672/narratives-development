// frontend/console/member/src/presentation/components/PermissionCard.tsx

import * as React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

type PermissionCardProps = {
  permissions: string[];
};

export function PermissionCard({ permissions }: PermissionCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>権限</CardTitle>
      </CardHeader>
      <CardContent>
        {permissions.length === 0 ? (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            権限は未設定です。
          </p>
        ) : (
          <ul className="text-sm space-y-1">
            {permissions.map((perm) => (
              <li key={perm}>{perm}</li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
