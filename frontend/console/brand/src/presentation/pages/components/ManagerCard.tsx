// frontend/console/brand/src/presentation/pages/components/ManagerCard.tsx

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardReadonly,
  CardLabel,
} from "../../../../../shell/src/shared/ui/card";

type ManagerCardProps = {
  managerName?: string;
  managerId: string;
  registeredAt: string;
  updatedAt: string;
};

export function ManagerCard({
  managerName,
  managerId,
  registeredAt,
  updatedAt,
}: ManagerCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>管理情報</CardTitle>
      </CardHeader>
      <CardContent>
        <CardLabel>責任者</CardLabel>
        <CardReadonly>
          {managerName || managerId || "（未設定）"}
        </CardReadonly>

        <CardLabel>登録日</CardLabel>
        <CardReadonly>{registeredAt}</CardReadonly>

        <CardLabel>最終更新日</CardLabel>
        <CardReadonly>{updatedAt}</CardReadonly>
      </CardContent>
    </Card>
  );
}
