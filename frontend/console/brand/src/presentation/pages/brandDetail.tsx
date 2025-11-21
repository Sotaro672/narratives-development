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
import { ManagerCard } from "./components/ManagerCard";
import { WalletCard } from "./components/WalletCard";
import { AssignedMemberCard } from "./components/AssignedMemberCard"; // ★ 追加

export default function BrandDetail() {
  // ★ statusBadgeClass は使わなくなったので分解から削除
  const { brand, handleBack } = useBrandDetail();

  return (
    <PageStyle layout="grid-2" title={`${brand.name}`} onBack={handleBack}>
      {/* 左カラム：基本情報 + AssignedMemberCard */}
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>ブランド名</CardLabel>
            <CardReadonly>{brand.name}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="border rounded-lg px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {brand.description}
            </div>

            {/* ★ WebサイトURL */}
            <CardLabel>WebサイトURL</CardLabel>
            <CardReadonly>
              {brand.websiteUrl?.trim() ? brand.websiteUrl : "（未設定）"}
            </CardReadonly>
          </CardContent>
        </Card>

        {/* ★ 追加：AssignedMemberCard（左カラムの2段目） */}
        <AssignedMemberCard
          assignedMembers={brand.assignedMembers ?? []}
        />
      </div>

      {/* 右カラム：管理情報 ＋ ウォレット情報 */}
      <div className="space-y-4">
        <ManagerCard
          managerName={brand.managerName}
          managerId={brand.managerId}
          registeredAt={brand.registeredAt}
          updatedAt={brand.updatedAt}
        />

        <WalletCard walletAddress={brand.walletAddress} />
      </div>
    </PageStyle>
  );
}
