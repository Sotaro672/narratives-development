// frontend/brand/src/pages/brandDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardReadonly,
  CardLabel,
} from "../../../../shared/ui/card";

export default function BrandDetail() {
  const navigate = useNavigate();
  const { brandId } = useParams<{ brandId: string }>();

  // ─────────────────────────────────────────────
  // モックデータ（API接続前）
  // ─────────────────────────────────────────────
  const [brand] = React.useState({
    id: brandId ?? "brand_001",
    name: "LUMINA Fashion",
    code: "LUMINA01",
    category: "ファッション",
    description:
      "上質な素材とサステナブルな生産体制を重視した女性向けファッションブランド。",
    owner: "佐藤 美咲",
    status: "アクティブ",
    registeredAt: "2024/05/10",
    updatedAt: "2025/11/01",
  });

  // ─────────────────────────────────────────────
  // AdminCard用モックデータ
  // ─────────────────────────────────────────────
  const [assignee, setAssignee] = React.useState("高橋 健太");
  const [creator] = React.useState("山田 太郎");
  const [createdAt] = React.useState("2024/05/10");

  // 戻るボタン処理
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ステータスの色分け
  const statusBadgeClass =
    brand.status === "アクティブ"
      ? "inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700 text-xs font-semibold"
      : "inline-flex items-center px-2 py-1 rounded-full bg-slate-50 text-slate-500 text-xs font-semibold";

  // ─────────────────────────────────────────────
  // JSX
  // ─────────────────────────────────────────────
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
            <div>{<span className={statusBadgeClass}>{brand.status}</span>}</div>

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
