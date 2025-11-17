//frontend\console\brand\src\presentation\pages\brandDetail.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardReadonly,
  CardLabel,
} from "../../../../shell/src/shared/ui/card";

import { useBrandDetail } from "../hook/useBrandDetail";

export default function BrandDetail() {
  const {
    brand,
    assignee,
    creator,
    createdAt,
    setAssignee,
    handleBack,
    statusBadgeClass,
  } = useBrandDetail();

  return (
    <PageStyle
      layout="grid-2"
      title={`ブランド詳細：${brand.name}`}
      onBack={handleBack}
    >
      {/* 左ペイン：ブランド情報 */}
      <div className="brand-detail">
        <Card className="mb-4">
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>ブランド名</CardLabel>
            <CardReadonly>{brand.name}</CardReadonly>

            <CardLabel>ブランドコード</CardLabel>
            <CardReadonly>{brand.code}</CardReadonly>

            <CardLabel>カテゴリ</CardLabel>
            <CardReadonly>{brand.category}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="border rounded-lg px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {brand.description}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>管理情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>責任者</CardLabel>
            <CardReadonly>{brand.owner}</CardReadonly>

            <CardLabel>ステータス</CardLabel>
            <div>
              <span className={statusBadgeClass}>{brand.status}</span>
            </div>

            <CardLabel>登録日</CardLabel>
            <CardReadonly>{brand.registeredAt}</CardReadonly>

            <CardLabel>最終更新日</CardLabel>
            <CardReadonly>{brand.updatedAt}</CardReadonly>
          </CardContent>
        </Card>
      </div>

      {/* 右ペイン：AdminCard */}
      <div>
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          onEditAssignee={() => setAssignee("新担当者")}
          onClickAssignee={() => console.log("assignee clicked:", assignee)}
          onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
        />
      </div>
    </PageStyle>
  );
}
