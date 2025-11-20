// frontend/console/brand/src/presentation/pages/brandDetail.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

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
  const { brand, handleBack, statusBadgeClass } = useBrandDetail();

  return (
    <PageStyle
      layout="single" // ★ singleレイアウト
      title={`ブランド詳細：${brand.name}`}
      onBack={handleBack}
    >
      <div className="brand-detail">
        {/* 基本情報 */}
        <Card className="mb-4">
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>ブランド名</CardLabel>
            <CardReadonly>{brand.name}</CardReadonly>

            {/* カテゴリ・ブランドコードは不使用のため削除済み */}

            <CardLabel>説明</CardLabel>
            <div className="border rounded-lg px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {brand.description}
            </div>
          </CardContent>
        </Card>

        {/* 管理情報 */}
        <Card>
          <CardHeader>
            <CardTitle>管理情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>責任者</CardLabel>
            <CardReadonly>
              {brand.managerName || brand.managerId || "（未設定）"}
            </CardReadonly>

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
    </PageStyle>
  );
}
