// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard from "../components/inventoryCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";

export default function InventoryDetail() {
  const navigate = useNavigate();
  const { inventoryId } = useParams<{ inventoryId?: string }>();

  /**
   * ★ inventoryId が無い（＝ /inventory/detail だけ or /inventory に誤マッチ）
   *    → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!inventoryId) {
      console.warn("[inventory/InventoryDetail] missing inventoryId -> redirect");
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  // ★ 戻るボタンも常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ hook に分離
  const { rows, loading, error } = useInventoryDetail(inventoryId);

  return (
    <PageStyle
      layout="grid-2"
      title={`在庫詳細：${inventoryId ?? ""}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：商品情報カード + 在庫一覧カード */}
      <div>
        <ProductBlueprintCard mode="view" />

        {/* --- style elements only --- */}
        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))] mt-2">
            読み込み中...
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600 mt-2">
            読み込みに失敗しました: {error}
          </div>
        )}
        {/* --- style elements only --- */}

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム：AdminCard のみ */}
      <AdminCard title="管理情報" />
    </PageStyle>
  );
}
