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

        {/* ★ 登録日＋更新日を横並びに配置 */}
        <div className="brand-date-row">
          <div className="brand-date-col">
            <CardLabel>登録日</CardLabel>
            <CardReadonly>{registeredAt}</CardReadonly>
          </div>
          <div className="brand-date-col">
            <CardLabel>更新日</CardLabel>
            <CardReadonly>{updatedAt}</CardReadonly>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
